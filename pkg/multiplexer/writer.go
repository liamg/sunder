package multiplexer

// Process StdIn and send it on to the active pane's process
func (m *Multiplexer) Write(data []byte) (n int, err error) {
	return len(data), m.rootPane.FindActive().HandleStdIn(data)
}
