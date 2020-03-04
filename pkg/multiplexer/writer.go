package multiplexer

const SunderShortcutKey = 0x1 // ctrl-a

// Process StdIn and send it on to the active pane's process
func (m *Multiplexer) Write(data []byte) (n int, err error) {

	if len(data) == 0 {
		return 0, nil
	}

	active := m.rootPane.FindActive()

	if m.inEscapeSequence {
		m.inEscapeSequence = false
		m.handleShortcut(data[0])
		return len(data), active.HandleStdIn(data[1:])
	} else if data[0] == SunderShortcutKey {
		if len(data) > 1 {
			m.handleShortcut(data[1])
			return len(data), active.HandleStdIn(data[2:])
		} else {
			m.inEscapeSequence = true
		}
	}

	return len(data), active.HandleStdIn(data)
}
