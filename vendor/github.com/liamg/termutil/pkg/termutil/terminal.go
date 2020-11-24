package termutil

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	MainBuffer     uint8 = 0
	AltBuffer      uint8 = 1
	InternalBuffer uint8 = 2
)

// Terminal communicates with the underlying terminal which is running shox
type Terminal struct {
	pty          *os.File
	updateChan   chan struct{}
	processChan  chan MeasuredRune
	closeChan    chan struct{}
	buffers      []*Buffer
	activeBuffer *Buffer
	title        string
	logFile      *os.File
}

// NewTerminal creates a new terminal instance
func New(options ...Option) *Terminal {
	term := &Terminal{
		processChan: make(chan MeasuredRune, 0xffff),
		closeChan:   make(chan struct{}),
	}
	term.buffers = []*Buffer{
		NewBuffer(1, 1, 0xffff),
		NewBuffer(1, 1, 0xffff),
		NewBuffer(1, 1, 0xffff),
	}
	term.activeBuffer = term.buffers[0]
	for _, opt := range options {
		opt(term)
	}
	return term
}

func (t *Terminal) log(line string, params ...interface{}) {
	if t.logFile != nil {
		_, _ = fmt.Fprintf(t.logFile, line+"\n", params...)
	}
}

// Pty exposes the underlying terminal pty, if it exists
func (t *Terminal) Pty() *os.File {
	return t.pty
}

func (t *Terminal) GetTitle() string {
	return t.title
}

// write takes data from StdOut of the child shell and processes it
func (t *Terminal) Write(data []byte) (n int, err error) {
	reader := bufio.NewReader(bytes.NewBuffer(data))
	for {
		r, size, err := reader.ReadRune()
		if err == io.EOF {
			break
		}
		t.processChan <- MeasuredRune{Rune: r, Width: size}
	}
	return len(data), nil
}

func (t *Terminal) SetSize(rows, cols uint16) error {
	if t.pty == nil {
		return fmt.Errorf("terminal is not running")
	}

	t.log("RESIZE %d, %d\n", cols, rows)

	t.activeBuffer.resizeView(cols, rows)
	if err := pty.Setsize(t.pty, &pty.Winsize{
		Rows: rows,
		Cols: cols,
	}); err != nil {
		return err
	}

	return nil
}

// Run starts the terminal/shell proxying process
func (t *Terminal) Run(updateChan chan struct{}, rows uint16, cols uint16) error {

	t.updateChan = updateChan

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	// Create arbitrary command.
	c := exec.Command(shell)

	// Start the command with a pty.
	var err error
	t.pty, err = pty.Start(c)
	if err != nil {
		return err
	}
	// Make sure to close the pty at the end.
	defer func() { _ = t.pty.Close() }() // Best effort.

	if err := t.SetSize(rows, cols); err != nil {
		return err
	}

	// Set stdin in raw mode.
	oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer func() { _ = terminal.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.

	go t.process()

	_, _ = io.Copy(t, t.pty)
	close(t.closeChan)
	return nil
}

// TODO close() method and kill goroutines

func (t *Terminal) requestRender() {
	select {
	case t.updateChan <- struct{}{}:
	default:
	}
}

func (t *Terminal) writeToRealStdOut(data ...rune) error {
	_, err := os.Stdout.Write([]byte(string(data)))
	return err
}

func (t *Terminal) respondToPty(data []byte) {
	_, _ = t.Pty().Write(data)
}

func (t *Terminal) process() {
	for {
		select {
		case <-t.closeChan:
			return
		case mr := <-t.processChan:
			if mr.Rune == 0x1b { // ANSI escape char, which means this is a sequence
				if t.handleANSI(t.processChan) {
					t.requestRender()
				}
			} else if t.processRunes(mr) { // otherwise it's just an individual rune we need to process
				t.requestRender()
			}
		}
	}
}

func (t *Terminal) processRunes(runes ...MeasuredRune) (renderRequired bool) {

	for _, r := range runes {

		t.log("%c 0x%X", r.Rune, r.Rune)

		switch r.Rune {
		case 0x05: //enq
			continue
		case 0x07: //bell
			//TODO handle this properly
			continue
		case 0x8: //backspace
			t.GetActiveBuffer().backspace()
			renderRequired = true
		case 0x9: //tab
			t.GetActiveBuffer().tab()
			renderRequired = true
		case 0xa, 0xc: //newLine/form feed
			t.GetActiveBuffer().newLine()
			renderRequired = true
		case 0xb: //vertical tab
			t.GetActiveBuffer().verticalTab()
			renderRequired = true
		case 0xd: //carriageReturn
			t.GetActiveBuffer().carriageReturn()
			renderRequired = true
		case 0xe: //shiftOut
			t.GetActiveBuffer().currentCharset = 1
		case 0xf: //shiftIn
			t.GetActiveBuffer().currentCharset = 0
		default:
			if r.Rune < 0x20 {
				// TODO handle any other control chars here
				continue
			}
			
			t.GetActiveBuffer().write(t.translateRune(r))
			renderRequired = true
		}
	}

	return renderRequired
}

func (t *Terminal) translateRune(b MeasuredRune) MeasuredRune {
	table := t.GetActiveBuffer().charsets[t.GetActiveBuffer().currentCharset]
	if table == nil {
		return b
	}
	chr, ok := (*table)[b.Rune]
	if ok {
		return MeasuredRune{Rune: chr, Width: 1}
	}
	return b
}

func (t *Terminal) setTitle(title string) {
	t.title = title
}

func (t *Terminal) switchBuffer(index uint8) {
	var carrySize bool
	var w, h uint16
	if t.activeBuffer != nil {
		w, h = t.activeBuffer.viewWidth, t.activeBuffer.viewHeight
		carrySize = true
	}
	t.activeBuffer = t.buffers[index]
	if carrySize {
		t.activeBuffer.resizeView(w, h)
	}
}

func (t *Terminal) GetActiveBuffer() *Buffer {
	return t.activeBuffer
}

func (t *Terminal) useMainBuffer() {
	t.switchBuffer(MainBuffer)
}

func (t *Terminal) useAltBuffer() {
	t.switchBuffer(AltBuffer)
}
