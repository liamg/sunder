package pane

import (
	"fmt"
	"sync"
	"time"

	"github.com/liamg/sunder/pkg/ansi"
)

type Anchor uint8

const (
	Top Anchor = iota
	Bottom
)

type StatusPane struct {
	child      Pane
	updateChan chan<- Pane
	closeChan  chan struct{}
	closeOnce  sync.Once
	anchor     Anchor
}

func NewStatusPane(updateChan chan<- Pane, child Pane, anchor Anchor) *StatusPane {
	return &StatusPane{
		child:      child,
		updateChan: updateChan,
		closeChan:  make(chan struct{}),
		anchor:     anchor,
	}
}

func (p *StatusPane) SetActive(target Pane) {
	if p == target {
		p.child.SetActive(p.child)
	} else {
		p.child.SetActive(target)
	}
}

func (p *StatusPane) Start(rows, cols uint16) error {

	updateChan := make(chan struct{}, 1)

	go func() {
		for {
			select {
			case <-updateChan:
				p.requestRender()
			case <-p.closeChan:
				return

			}
		}
	}()

	p.requestRender()

	err := p.child.Start(rows-1, cols)

	p.requestRender()
	p.Close()

	return err
}

func (p *StatusPane) Exists() bool {
	return p.child.Exists()
}

func (p *StatusPane) Close() {
	p.closeOnce.Do(func() {
		p.child.Close()
		close(p.closeChan)
	})
}

func (p *StatusPane) requestRender() {
	select {
	case p.updateChan <- p:
	default:
		// TODO handle this case when buffer is full and channel blocks?
	}
}

func (p *StatusPane) Resize(rows uint16, cols uint16) error {

	_ = p.child.Resize(rows-1, cols)
	p.requestRender()
	return nil
}

func (p *StatusPane) HandleStdIn(data []byte) error {
	// should not be possible, as containers cannot be returned from FindActive()
	return fmt.Errorf("not supported")
}

func (p *StatusPane) Render(target Pane, offsetX, offsetY, rows, cols uint16, writer *ansi.Writer) {

	if p == target {
		// draw status bar

		switch p.anchor {
		case Top:
			writer.MoveCursorTo(offsetY, offsetX)
		case Bottom:
			writer.MoveCursorTo(offsetY+rows-1, offsetX)
		}

		writer.ClearLine()

		// set colours
		_, _ = writer.Write([]byte("\r\x1b[41m\x1b[97m"))

		output := fmt.Sprintf(" Sunder %s", time.Now().String())

		for len(output) < int(cols) {
			output += " "
		}

		_, _ = writer.Write([]byte(output))

		return
	}

	// bump child pane down if status bar goes at the top
	if p.anchor == Top {
		offsetY += 1
	}

	p.child.Render(target, offsetX, offsetY, rows-1, cols, writer)

}

func (p *StatusPane) FindActive() Pane {
	return p.child.FindActive()
}

func (p *StatusPane) Split(target Pane, mode SplitMode) bool {
	splitter, ok := p.child.(Splitter)
	if !ok {
		return false
	}
	if target == p {
		target = p.child
	}
	return splitter.Split(target, mode)
}
