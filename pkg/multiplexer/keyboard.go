package multiplexer

import "github.com/liamg/sunder/pkg/pane"

func (m *Multiplexer) handleShortcut(input byte) {
	switch input {
	case 'v':
		// TODO how to handle errors here? message box? output to stdout in active pane?
		_ = m.SplitActivePane(pane.Vertical)
	case 'h':
		// TODO how to handle errors here? message box? output to stdout in active pane?
		_ = m.SplitActivePane(pane.Horizontal)
	}
}
