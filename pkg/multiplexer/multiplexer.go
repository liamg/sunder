package multiplexer

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/liamg/sunder/pkg/ansi"

	"github.com/liamg/sunder/pkg/pane"

	"github.com/creack/pty"
	sunderterm "github.com/liamg/sunder/pkg/terminal"
	"golang.org/x/crypto/ssh/terminal"
)

type Multiplexer struct {
	// root pane
	rootPane   pane.Pane
	activePane pane.Pane
	// write to this to get stdout on the parent terminal
	output       chan byte
	stdoutWriter *ansi.Writer
	// panes write to this channel to request to be rendered by the multiplexer
	updateChan <-chan pane.Pane
	closeChan  chan struct{}
	closeOnce  sync.Once
	rows       uint16
	cols       uint16
	renderLock sync.Mutex
	paneLock   sync.Mutex
}

func New() *Multiplexer {
	update := make(chan pane.Pane, 0xff)
	out := make(chan byte, 0xffff)
	stdoutWriter := NewChanWriter(out)

	terminalPane := pane.NewTerminalPane(update, sunderterm.New(sunderterm.WithLogFile("/tmp/sunder.log")))
	container := pane.NewContainerPane(update, pane.Horizontal, terminalPane)
	status := pane.NewStatusPane(update, container, pane.Bottom)

	mp := &Multiplexer{
		rootPane:     status,
		activePane:   terminalPane,
		output:       out,
		updateChan:   update,
		closeChan:    make(chan struct{}),
		stdoutWriter: ansi.NewWriter(stdoutWriter),
	}
	return mp
}

func (m *Multiplexer) SplitActivePane(mode pane.SplitMode) error {
	active := m.rootPane.FindActive()
	if active == nil {
		return fmt.Errorf("no active pane found")
	}
	splitter, ok := m.rootPane.(pane.Splitter)
	if !ok {
		return fmt.Errorf("root pane does not support splitting")
	}
	if !splitter.Split(active, mode) {
		return fmt.Errorf("failed to split active pane")
	}
	return nil
}

func (m *Multiplexer) Start() error {

	m.renderLock.Lock()

	// TODO for debugging, remove later
	_ = os.Setenv("SUNDER", "1")

	// RIS
	m.stdoutWriter.Reset()

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

	// kick off root pane
	m.rootPane.SetActive(m.activePane)
	go func() { _ = m.rootPane.Start(size.Rows, size.Cols) }()

	// tidy up root pane on exit
	defer m.rootPane.Close()

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

	m.renderLock.Unlock()
	_, err = io.Copy(os.Stdout, m)

	// reset terminal on exit
	ansi.NewWriter(os.Stdout).Reset()

	return err

}

func (m *Multiplexer) Close() {

	m.paneLock.Lock()
	defer m.paneLock.Unlock()

	m.rootPane.Close()

	m.closeOnce.Do(func() {
		close(m.closeChan)
	})
}

func (m *Multiplexer) resize(rows uint16, cols uint16) error {
	m.paneLock.Lock()
	defer m.paneLock.Unlock()
	// resize root pane

	if err := m.rootPane.Resize(rows, cols); err != nil {
		return err
	}

	m.cols = cols
	m.rows = rows
	return nil
}

func (m *Multiplexer) render(target pane.Pane) {

	m.renderLock.Lock()
	defer m.renderLock.Unlock()

	if !m.rootPane.Exists() {
		m.Close()
		return
	}

	m.rootPane.Render(target, 0, 0, m.rows, m.cols, m.stdoutWriter)

	// render active again to fix cursor position etc.
	active := m.rootPane.FindActive()
	if active != target {
		m.rootPane.Render(active, 0, 0, m.rows, m.cols, m.stdoutWriter)
	}
}
