package multiplexer

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/liamg/sunder/pkg/pane"

	"github.com/creack/pty"
	sunderterm "github.com/liamg/sunder/pkg/terminal"
	"golang.org/x/crypto/ssh/terminal"
)

type Multiplexer struct {
	// all top level panes
	panes []*pane.Pane
	// pane which the user has selected
	activePane *pane.Pane
	// write to this to get stdout on the parent terminal
	output chan byte
	// panes write to this channel to request to be rendered by the multiplexer
	updateChan <-chan *pane.Pane
	closeChan  chan struct{}
	closeOnce  sync.Once
	rows       uint16
	cols       uint16
}

func New() *Multiplexer {
	update := make(chan *pane.Pane, 0xff)
	mp := &Multiplexer{
		panes:      []*pane.Pane{pane.NewPane(pane.NewFullscreenPosition(), update, sunderterm.New(sunderterm.WithLogFile("/tmp/sunder.log")))},
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

	// TODO for debugging, remove later
	_ = os.Setenv("SUNDER", "1")

	// RIS
	m.writeToStdOut([]byte{0x1b, 'c'})

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

	size, err := pty.GetsizeFull(os.Stdin)
	if err != nil {
		return err
	}

	// kick off all paned terminals
	for _, p := range m.panes {
		func(p *pane.Pane) {
			go func() { _ = p.Start(size.Rows, size.Cols) }()
		}(p)
	}

	// tidy up any hanging terminals on exit
	defer func() {
		for _, p := range m.panes {
			p.Close()
		}
	}()

	// Copy stdin to the multiplexer and the multiplexer output to stdout.
	go func() { _, _ = io.Copy(m, os.Stdin) }()
	go func() {
		for {
			select {
			case p := <-m.updateChan:
				m.render(p)
			case <-m.closeChan:
				return
			}
		}
	}()
	_, err = io.Copy(os.Stdout, m)
	return err

}

func (m *Multiplexer) Close() {

	// TODO: close all panes

	m.closeOnce.Do(func() {
		close(m.closeChan)
	})
}

func (m *Multiplexer) moveCursor(x, y uint16) {
	m.writeToStdOut([]byte(fmt.Sprintf("\x1b[%d;%dH", y+1, x+1))) // 1-indexed
}

func (m *Multiplexer) setCursorVisible(visible bool) {
	ctrl := "\x1b[?25"
	if visible {
		ctrl += "h"
	} else {
		ctrl += "l"
	}
	m.writeToStdOut([]byte(ctrl)) // 1-indexed
}

func (m *Multiplexer) resize(rows uint16, cols uint16) error {
	// resize root pane
	for _, p := range m.panes {
		if err := p.Resize(rows, cols); err != nil {
			return err
		}
	}
	m.cols = cols
	m.rows = rows
	return nil
}
