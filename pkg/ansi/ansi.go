package ansi

import (
	"fmt"
	"io"
)

type Writer struct {
	writer io.Writer
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{writer: w}
}

func (w *Writer) Write(data []byte) (n int, err error) {
	return w.writer.Write(data)
}

// ClearLine clears the current terminal line at the cursor position
func (w *Writer) ClearLine() {
	_, _ = fmt.Fprintf(w.writer, "\x1b[K")
}

// Clear clears all content from the terminal
func (w *Writer) Clear() {
	_, _ = fmt.Fprintf(w.writer, "\x1b[2J")
}

// Reset performs a full reset on the terminal
func (w *Writer) Reset() {
	_, _ = fmt.Fprintf(w.writer, "\x1bc")
}

// SaveCursorPosition pushes the cursor position to the stack
func (w *Writer) SaveCursorPosition() {
	_, _ = fmt.Fprintf(w.writer, "\x1b[s")
}

// RestoreCursorPosition pops the cursor position from the stack
func (w *Writer) RestoreCursorPosition() {
	_, _ = fmt.Fprintf(w.writer, "\x1b[u")
}

// MoveCursorTo a 0-indexed position
func (w *Writer) MoveCursorTo(row, col uint16) {
	_, _ = fmt.Fprintf(w.writer, "\x1b[%d;%dH", row+1, col+1)
}

func (w *Writer) ResetFormatting() {
	_, _ = w.Write([]byte("\x1b[0m"))
}

func (w *Writer) SetCursorVisible(visible bool) {
	ctrl := "\x1b[?25"
	if visible {
		ctrl += "h"
	} else {
		ctrl += "l"
	}
	_, _ = w.Write([]byte(ctrl)) // 1-indexed
}
