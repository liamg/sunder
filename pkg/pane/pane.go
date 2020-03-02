package pane

import (
	"github.com/liamg/sunder/pkg/ansi"
)

type SplitMode uint8

const (
	Horizontal SplitMode = iota
	Vertical
)

type Pane interface {
	Start(rows, cols uint16) error
	Resize(rows uint16, cols uint16) error
	Render(target Pane, offsetX, offsetY, rows, cols uint16, w *ansi.Writer)
	SetActive(target Pane)
	FindActive() Pane
	HandleStdIn(data []byte) error
	Exists() bool
	Close()
}

type Splitter interface {
	Split(target Pane, mode SplitMode) bool
}
