package termutil

import (
	"strings"
)

type Line struct {
	wrapped bool // whether line was wrapped onto from the previous one
	nobreak bool // true if no line break at the beginning of the line
	cells   []Cell
}

func newLine() Line {
	return Line{
		wrapped: false,
		nobreak: false,
		cells:   []Cell{},
	}
}

func (line *Line) reverseVideo() {
	for i, _ := range line.cells {
		line.cells[i].attr.reverseVideo()
	}
}

// cleanse removes null bytes from the end of the row
func (line *Line) cleanse() {
	cut := 0
	for i := len(line.cells) - 1; i >= 0; i-- {
		if line.cells[i].r.Rune != 0 {
			break
		}
		cut++
	}
	if cut == 0 {
		return
	}
	line.cells = line.cells[:len(line.cells)-cut]
}

func (line *Line) setWrapped(wrapped bool) {
	line.wrapped = wrapped
}

func (line *Line) setNoBreak(nobreak bool) {
	line.nobreak = nobreak
}

func (line *Line) String() string {
	runes := []rune{}
	for _, cell := range line.cells {
		runes = append(runes, cell.r.Rune)
	}
	return strings.TrimRight(string(runes), "\x00 ")
}

// @todo test these (ported from legacy) ------------------
func (line *Line) cutCellsAfter(n int) []Cell {
	cut := line.cells[n:]
	line.cells = line.cells[:n]
	return cut
}

func (line *Line) cutCellsFromBeginning(n int) []Cell {
	if n > len(line.cells) {
		n = len(line.cells)
	}
	cut := line.cells[:n]
	line.cells = line.cells[n:]
	return cut
}

func (line *Line) cutCellsFromEnd(n int) []Cell {
	cut := line.cells[len(line.cells)-n:]
	line.cells = line.cells[:len(line.cells)-n]
	return cut
}

func (line *Line) append(cells ...Cell) {
	line.cells = append(line.cells, cells...)
}

// -------------------------------------------------------
