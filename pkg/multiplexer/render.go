package multiplexer

import (
	"fmt"

	"github.com/liamg/sunder/pkg/pane"
	sunderterm "github.com/liamg/sunder/pkg/terminal"
)

func (m *Multiplexer) render(pane *pane.Pane) {

	if !pane.Exists() {
		// TODO remove pane and resize all children
		fmt.Println("Shell exited.")
		return
	}

	// TODO render all pane specific output
	term := pane.GetTerminal()
	if term == nil {
		return
	}

	buffer := term.GetActiveBuffer()
	if buffer == nil {
		return
	}

	offsetPosition := pane.GetPosition().ToFixed(m.rows, m.cols)

	// grab cursor to restore afterwards - could use ansi code and let parent terminal handle this?
	cursorX, cursorY := buffer.CursorColumn(), buffer.CursorLine()

	cursorX += offsetPosition.Origin.X
	cursorY += offsetPosition.Origin.Y

	var lastCellAttr sunderterm.CellAttributes

	for y := uint16(0); y < m.rows; y++ {
		for x := uint16(0); x < m.cols; x++ {
			cell := buffer.GetCell(x, y)

			if cell != nil {
				measuredRune := cell.Rune()
				// TODO if run + attr are the same as last render, skip this step

				if measuredRune.Rune == 0 {
					measuredRune.Rune = 0x20
				}

				m.moveCursor(offsetPosition.Origin.X+x, offsetPosition.Origin.Y+y)
				sgr := cell.Attr().GetDiffANSI(lastCellAttr)
				m.writeToStdOut([]byte(sgr + string(measuredRune.Rune)))
				lastCellAttr = cell.Attr()
			} else {
				m.moveCursor(offsetPosition.Origin.X+x, offsetPosition.Origin.Y+y)
				// TODO reset SGR?
				m.writeToStdOut([]byte{0x20})
			}

		}
	}

	// move cursor back
	m.moveCursor(cursorX, cursorY)

}
