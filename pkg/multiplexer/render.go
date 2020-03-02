package multiplexer

import (
	"github.com/liamg/sunder/pkg/pane"
	sunderterm "github.com/liamg/sunder/pkg/terminal"
)

func (m *Multiplexer) render(pane *pane.Pane) {

	m.renderLock.Lock()
	defer m.renderLock.Unlock()

	if !pane.Exists() {
		// TODO remove pane and resize all children
		m.removePane(pane)
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

	m.setCursorVisible(false)

	cursorX += offsetPosition.Origin.X
	cursorY += offsetPosition.Origin.Y

	var lastCellAttr sunderterm.CellAttributes

	localX, localY := cursorX, cursorY
	var targetX, targetY uint16

	for y := uint16(0); y < m.rows; y++ {
		for x := uint16(0); x < m.cols; x++ {
			cell := buffer.GetCell(x, y)

			targetX, targetY = offsetPosition.Origin.X+x, offsetPosition.Origin.Y+y
			if localX != targetX || localY != targetY {
				localX, localY = targetX, targetY
				m.moveCursor(localX, localY)
			}

			if cell != nil {
				measuredRune := cell.Rune()
				// TODO if rune + attr are the same as last render, skip this step

				// TODO can we remove this? safely replaces control characters/null bytes with spaces
				if measuredRune.Rune < 0x20 {
					measuredRune.Rune = 0x20
				}

				sgr := cell.Attr().GetDiffANSI(lastCellAttr)

				//if measuredRune.Width > 0 {
				m.writeToStdOut([]byte(sgr + string(measuredRune.Rune)))
				lastCellAttr = cell.Attr()
				//}
			} else {
				// TODO reset SGR?
				lastCellAttr = sunderterm.CellAttributes{}
				m.writeToStdOut([]byte("\x1b[0m"))
				m.writeToStdOut([]byte{0x20})
			}
			localX++
		}
	}

	// only show the cursor for the active pane
	if pane == m.activePane {
		// move cursor back
		m.moveCursor(cursorX, cursorY)
		m.setCursorVisible(buffer.IsCursorVisible())
	}

}
