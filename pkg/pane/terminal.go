package pane

import (
	"sync"

	"github.com/liamg/sunder/pkg/logger"

	"github.com/liamg/sunder/pkg/ansi"

	"github.com/liamg/sunder/pkg/terminal"
)

type TerminalPane struct {
	terminal   *terminal.Terminal
	updateChan chan<- Pane
	exists     bool
	active     bool
	closeChan  chan struct{}
	closeOnce  sync.Once
	startLock  sync.Mutex
	started    bool
}

func NewTerminalPane(updateChan chan<- Pane, term *terminal.Terminal) *TerminalPane {
	return &TerminalPane{
		terminal:   term,
		updateChan: updateChan,
		closeChan:  make(chan struct{}),
		exists:     true,
	}
}

func (p *TerminalPane) SetActive(target Pane) {
	p.active = p == target
}

func (p *TerminalPane) Start(rows, cols uint16) error {

	p.startLock.Lock()
	defer p.startLock.Unlock()

	if p.started {
		return nil
	}
	p.started = true

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

	if err := p.terminal.Run(updateChan, rows, cols); err != nil {
		return err
	}
	p.requestRender()
	p.Close()
	return nil
}

func (p *TerminalPane) Exists() bool {
	if p == nil {
		return false
	}
	return p.exists
}

func (p *TerminalPane) Close() {
	p.closeOnce.Do(func() {
		close(p.closeChan)
		p.exists = false
	})
}

func (p *TerminalPane) requestRender() {
	select {
	case p.updateChan <- p:
	default:
		// TODO handle this case when buffer is full and channel blocks?
	}
}

func (p *TerminalPane) Resize(rows uint16, cols uint16) error {

	logger.Log("Resizing terminal pane to %dx%d", cols, rows)
	if p.terminal.Pty() != nil {
		if err := p.terminal.SetSize(rows, cols); err != nil {
			return err
		}
	}

	return nil
}

func (p *TerminalPane) HandleStdIn(data []byte) error {
	_, err := p.terminal.Pty().Write(data)
	return err
}

func (p *TerminalPane) Render(target Pane, offsetX, offsetY, rows, cols uint16, w *ansi.Writer) {

	if p != target {
		return
	}

	if p.terminal == nil {
		return
	}

	buffer := p.terminal.GetActiveBuffer()
	if buffer == nil {
		return
	}

	// grab cursor to restore afterwards - could use ansi code and let parent terminal handle this?
	cursorX, cursorY := buffer.CursorColumn(), buffer.CursorLine()

	w.SetCursorVisible(false)

	// replace mode!
	_, _ = w.Write([]byte("\x1b[?4l"))

	cursorX += offsetX
	cursorY += offsetY

	var lastCellAttr terminal.CellAttributes

	for y := uint16(0); y < rows; y++ {
		for x := uint16(0); x < cols; x++ {
			cell := buffer.GetCell(x, y)

			w.MoveCursorTo(offsetY+y, offsetX+x)

			if cell != nil {
				measuredRune := cell.Rune()
				// TODO if rune + attr are the same as last render, skip this step
				// TODO render diffs more efficiently, this is very basic/slow atm

				// TODO can we remove this? safely replaces control characters/null bytes with spaces
				if measuredRune.Rune < 0x20 {
					measuredRune.Rune = 0x20
				}

				sgr := cell.Attr().GetDiffANSI(lastCellAttr)

				//if measuredRune.Width > 0 {
				_, _ = w.Write([]byte(sgr + string(measuredRune.Rune)))
				lastCellAttr = cell.Attr()
				//}
			} else {
				lastCellAttr = terminal.CellAttributes{}
				w.ResetFormatting()
				_, _ = w.Write([]byte{0x20})
			}
		}
	}

	// only reposition the cursor for the active pane
	if p.active {
		// move cursor back
		//logger.Log("Moving cursor to %d, %d", cursorX, cursorY)
		w.MoveCursorTo(cursorY, cursorX)
		w.SetCursorVisible(buffer.IsCursorVisible())
	}

}

func (p *TerminalPane) FindActive() Pane {
	if !p.active {
		return nil
	}
	return p
}
