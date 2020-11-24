package pane

import (
	"fmt"
	"sync"

	"github.com/liamg/sunder/pkg/logger"

	terminal "github.com/liamg/termutil/pkg/termutil"

	"github.com/liamg/sunder/pkg/ansi"
)

type ContainerPane struct {
	mode       SplitMode
	children   []Pane
	updateChan chan<- Pane
	closeChan  chan struct{}
	closeOnce  sync.Once
	childWait  sync.WaitGroup
	rows       uint16
	cols       uint16
}

func NewContainerPane(updateChan chan<- Pane, mode SplitMode, children ...Pane) *ContainerPane {
	return &ContainerPane{
		mode:       mode,
		children:   children,
		updateChan: updateChan,
		closeChan:  make(chan struct{}),
	}
}

func (p *ContainerPane) SetActive(target Pane) {

	if len(p.children) == 0 {
		return
	}

	if p == target {
		p.children[0].SetActive(p.children[0])
		return
	}

	for _, child := range p.children {
		child.SetActive(target)
	}
}

func (p *ContainerPane) Start(rows, cols uint16) error {

	updateChan := make(chan struct{}, 1)

	p.cols = cols
	p.rows = rows

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

	for i, child := range p.children {
		_, _, w, h := p.calculateOffsetPositionForChildN(cols, rows, i)
		p.childWait.Add(1)
		go func(c Pane, w, h uint16) {
			_ = c.Resize(h, w)
			_ = c.Start(h, w)
			p.clean()
			p.childWait.Done()
		}(child, w, h)
	}

	p.requestRender()

	p.childWait.Wait()

	p.requestRender()
	p.Close()

	return nil
}

func (p *ContainerPane) clean() {

	var setNewActive bool

	// remove inactive children
	var filtered []Pane
	for _, p := range p.children {
		if !p.Exists() {
			if p.FindActive() != nil {
				setNewActive = true
			}
			continue
		}
		filtered = append(filtered, p)
	}

	if setNewActive {
		if len(filtered) > 0 {
			p.SetActive(filtered[len(filtered)-1])
		}
	}

	if len(filtered) != len(p.children) {
		p.children = filtered
		_ = p.Resize(p.rows, p.cols)
	}
}

func (p *ContainerPane) Exists() bool {
	for _, child := range p.children {
		if child.Exists() {
			return true
		}
	}
	return false
}

func (p *ContainerPane) Close() {
	p.closeOnce.Do(func() {
		for _, child := range p.children {
			child.Close()
		}
		close(p.closeChan)
	})
}

func (p *ContainerPane) requestRender() {
	select {
	case p.updateChan <- p:
	default:
		// TODO handle this case when buffer is full and channel blocks?
	}
}

func (p *ContainerPane) Resize(rows uint16, cols uint16) error {

	for i, child := range p.children {
		_, _, w, h := p.calculateOffsetPositionForChildN(cols, rows, i)
		logger.Log("Resizing child to %dx%d", w, h)
		if err := child.Resize(h, w); err != nil {
			return err
		}
	}

	p.cols = cols
	p.rows = rows

	p.requestRender()
	return nil
}

func (p *ContainerPane) HandleStdIn(data []byte) error {
	// should not be possible, as containers cannot be returned from FindActive()
	return fmt.Errorf("not supported")
}

func (p *ContainerPane) Render(target Pane, offsetX, offsetY, rows, cols uint16, writer *ansi.Writer) {

	sendChildAsTarget := target == p

	// TODO draw dividers

	for i, child := range p.children {
		// recalculate offsets/sizes before rendering
		childOffsetX, childOffsetY, w, h := p.calculateOffsetPositionForChildN(cols, rows, i)
		if sendChildAsTarget {
			target = child

			// only draw border if rendering of whole container requested

			if i < len(p.children)-1 {

				writer.Write([]byte("\x1b[31m"))
				writer.SetCursorVisible(false)

				switch p.mode {
				case Horizontal:
					writer.MoveCursorTo(offsetY+childOffsetY+h, offsetX+childOffsetX)
					for x := uint16(0); x < w; x++ {
						_, _ = writer.Write([]byte("━"))
					}
				case Vertical:
					for y := uint16(0); y < h; y++ {
						writer.MoveCursorTo(offsetY+childOffsetY+y, offsetX+childOffsetX+w)
						_, _ = writer.Write([]byte("┃"))
					}
				}

				writer.SetCursorVisible(true)
			}

		}

		child.Render(target, offsetX+childOffsetX, offsetY+childOffsetY, h, w, writer)
	}

}

func (p *ContainerPane) FindActive() Pane {
	for _, child := range p.children {
		if active := child.FindActive(); active != nil {
			return active
		}
	}
	return nil
}

func (p *ContainerPane) calculateOffsetPositionForChildN(cols, rows uint16, childN int) (x, y, w, h uint16) {

	if len(p.children) == 1 {
		return 0, 0, cols, rows
	}

	w = cols
	h = rows

	count := uint16(len(p.children))

	switch p.mode { // height is affected
	case Horizontal:
		availableHeight := rows - (count - 1) // available height is height minus dividers
		eachH := availableHeight / count

		// last pane always gets leftovers e.g. 10/3 = 3, 3, 4
		if uint16(childN) == count-1 {
			h = eachH + (availableHeight - (eachH * count))
		} else {
			h = eachH
		}

		y = (eachH + 1) * uint16(childN)
	case Vertical:
		availableWidth := cols - (count - 1) // available height is height minus dividers
		eachW := availableWidth / count

		// last pane always gets leftovers e.g. 10/3 = 3, 3, 4
		if uint16(childN) == count-1 {
			w = eachW + (availableWidth - (eachW * count))
		} else {
			w = eachW
		}

		x = (eachW + 1) * uint16(childN)
	}

	//logger.Log("Offset = %d, %d, Size = %dx%d", x, y, w, h)

	return
}

func (p *ContainerPane) Split(target Pane, mode SplitMode) bool {
	for i, child := range p.children {
		if child == target {

			logger.Log("Found child to split!")

			term := terminal.New()
			termPane := NewTerminalPane(p.updateChan, term)
			container := NewContainerPane(p.updateChan, mode, child, termPane)

			_, _, w, h := p.calculateOffsetPositionForChildN(p.cols, p.rows, i)

			logger.Log("New dimensions for entire container should be %dx%d", w, h)

			p.children[i] = container
			p.childWait.Add(1)

			go func() {
				// approx dimensions, we can adjust this on the subsequent resize
				_ = container.Start(h, w)
				p.childWait.Done()
			}()

			// make new pane the active
			container.SetActive(termPane)

			return true
		} else if splitter, ok := child.(Splitter); ok {
			_ = splitter
			if splitter.Split(target, mode) {
				return true
			}
		}
	}
	return false
}
