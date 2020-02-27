package terminal

func (t *Terminal) handleANSI(readChan chan MeasuredRune) (renderRequired bool) {
	// if the byte is an escape character, read the next byte to determine which one
	r := <-readChan

	switch r.Rune {
	case '[':
		return t.handleCSI(readChan)
	default: // TODO if the escape sequence is unknown, pass it to real stdout - review as this is kind of risky...
		_ = t.writeToRealStdOut(0x1b, r.Rune)
		return false
	}
}
