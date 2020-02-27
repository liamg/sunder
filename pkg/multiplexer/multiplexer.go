package multiplexer

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/creack/pty"
	mpterm "github.com/liamg/sunder/pkg/terminal"
	"golang.org/x/crypto/ssh/terminal"
)

type Multiplexer struct {
	// all top level panes
	panes []*Pane
	// pane which the user has selected
	activePane *Pane
	// write to this to get stdout on the parent terminal
	output chan byte
	// panes write to this channel to request to be rendered by the multiplexer
	updateChan <-chan *Pane
	closeChan  chan struct{}
	closeOnce  sync.Once
}

func New() *Multiplexer {
	update := make(chan *Pane, 0xff)
	mp := &Multiplexer{
		panes:      []*Pane{NewPane(NewFullscreenPosition(), update, mpterm.New())},
		output:     make(chan byte, 0xffff),
		updateChan: update,
		closeChan:  make(chan struct{}),
	}
	return mp
}

func (m *Multiplexer) writeToStdOut(data []byte) {
	for _, b := range data {
		m.output <- b
	}
}

func (m *Multiplexer) Start() error {

	// default the active pane
	m.activePane = m.panes[0]

	// Handle pty size.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for {
			select {
			case <-ch:

				size, err := pty.GetsizeFull(os.Stdin)
				if err != nil {
					continue
				}

				_ = m.resize(size.Rows, size.Cols)
			case <-m.closeChan:
				return
			}
		}
	}()
	ch <- syscall.SIGWINCH // Initial resize.

	// Set stdin in raw mode.
	oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer func() { _ = terminal.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort restore.

	// Copy stdin to the multiplexer and the multiplexer output to stdout.
	go func() { _, _ = io.Copy(m, os.Stdin) }()
	go func() {

		// create render loop here ?
		for {
			select {
			case pane := <-m.updateChan:
				m.render(pane)
			case <-m.closeChan:
				return
			}

		}

	}()
	_, _ = io.Copy(os.Stdout, m)
	return nil

}

func (m *Multiplexer) Close() {

	// TODO: close all panes

	m.closeOnce.Do(func() {
		close(m.closeChan)
	})
}

func (m *Multiplexer) render(pane *Pane) {
	if !pane.Exists() {
		// TODO remove pane and resize all children
		return
	}

	// TODO render all pane specific output

}

func (m *Multiplexer) resize(rows uint16, cols uint16) error {
	// resize root pane
	for _, pane := range m.panes {
		if err := pane.Resize(rows, cols); err != nil {
			return err
		}
	}
	return nil
}

// Process StdIn and send it on to the active pane's process
func (m *Multiplexer) Write(data []byte) (n int, err error) {
	return len(data), m.activePane.ProcessStdIn(data)
}

// Read reads data from the multiplexer into stdout for the parent terminal to process/render
func (m *Multiplexer) Read(data []byte) (n int, err error) {
	for i := 0; i < cap(data); i++ {
		select {
		case b := <-m.output:
			data[i] = b
		default:
			return i, nil
		}
	}
	return 0, fmt.Errorf("buffer has no capacity")
}
