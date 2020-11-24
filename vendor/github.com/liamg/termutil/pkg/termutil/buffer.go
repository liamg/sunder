package termutil

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
)

const TabSize = 8

type Buffer struct {
	lines                 []Line
	savedX                uint16
	savedY                uint16
	savedCursorAttr       *CellAttributes
	savedCharsets         []*map[rune]rune
	savedCurrentCharset   int
	topMargin             uint // see DECSTBM docs - this is for scrollable regions
	bottomMargin          uint // see DECSTBM docs - this is for scrollable regions
	viewWidth             uint16
	viewHeight            uint16
	cursorX               uint16
	cursorY               uint16
	cursorAttr            CellAttributes
	scrollLinesFromBottom uint
	maxLines              uint64
	tabStops              []uint16
	charsets              []*map[rune]rune // array of 2 charsets, nil means ASCII (no conversion)
	currentCharset        int              // active charset index in charsets array, valid values are 0 or 1
	modes                 Modes
	mouseMode             MouseMode
	mouseExtMode          MouseExtMode
	bracketedPasteMode    bool
}

type Position struct {
	Line int
	Col  int
}

func comparePositions(pos1 *Position, pos2 *Position) int {
	if pos1.Line < pos2.Line || (pos1.Line == pos2.Line && pos1.Col < pos2.Col) {
		return 1
	}
	if pos1.Line > pos2.Line || (pos1.Line == pos2.Line && pos1.Col > pos2.Col) {
		return -1
	}

	return 0
}

// NewBuffer creates a new terminal buffer
func NewBuffer(width, height uint16, maxLines uint64) *Buffer {
	b := &Buffer{
		lines:        []Line{},
		viewHeight:   height,
		viewWidth:    width,
		maxLines:     maxLines,
		topMargin:    0,
		bottomMargin: uint(height - 1),
		charsets:     []*map[rune]rune{nil, nil},
		modes: Modes{
			LineFeedMode: true,
			AutoWrap:     true,
			ShowCursor:   true,
		},
	}
	return b
}

func (buffer *Buffer) IsCursorVisible() bool {
	return buffer.modes.ShowCursor
}

func (buffer *Buffer) HasScrollableRegion() bool {
	return buffer.topMargin > 0 || buffer.bottomMargin < uint(buffer.ViewHeight())-1
}

func (buffer *Buffer) InScrollableRegion() bool {
	return buffer.HasScrollableRegion() && uint(buffer.cursorY) >= buffer.topMargin && uint(buffer.cursorY) <= buffer.bottomMargin
}

// NOTE: bottom is exclusive
func (buffer *Buffer) getAreaScrollRange() (top uint64, bottom uint64) {
	top = buffer.convertViewLineToRawLine(uint16(buffer.topMargin))
	bottom = buffer.convertViewLineToRawLine(uint16(buffer.bottomMargin)) + 1
	if bottom > uint64(len(buffer.lines)) {
		bottom = uint64(len(buffer.lines))
	}
	return top, bottom
}

func (buffer *Buffer) areaScrollDown(lines uint16) {

	// NOTE: bottom is exclusive
	top, bottom := buffer.getAreaScrollRange()

	for i := bottom; i > top; {
		i--
		if i >= top+uint64(lines) {
			buffer.lines[i] = buffer.lines[i-uint64(lines)]
		} else {
			buffer.lines[i] = newLine()
		}
	}
}

func (buffer *Buffer) areaScrollUp(lines uint16) {

	// NOTE: bottom is exclusive
	top, bottom := buffer.getAreaScrollRange()

	for i := top; i < bottom; i++ {
		from := i + uint64(lines)
		if from < bottom {
			buffer.lines[i] = buffer.lines[from]
		} else {
			buffer.lines[i] = newLine()
		}
	}
}

func (buffer *Buffer) saveCursor() {
	copiedAttr := buffer.cursorAttr
	buffer.savedCursorAttr = &copiedAttr
	buffer.savedX = buffer.cursorX
	buffer.savedY = buffer.cursorY
	buffer.savedCharsets = make([]*map[rune]rune, len(buffer.charsets))
	copy(buffer.savedCharsets, buffer.charsets)
	buffer.savedCurrentCharset = buffer.currentCharset
}

func (buffer *Buffer) restoreCursor() {
	if buffer.savedCursorAttr != nil {
		copiedAttr := *buffer.savedCursorAttr
		buffer.cursorAttr = copiedAttr // @todo ignore colors?
	}
	buffer.cursorX = buffer.savedX
	buffer.cursorY = buffer.savedY
	if buffer.savedCharsets != nil {
		buffer.charsets = make([]*map[rune]rune, len(buffer.savedCharsets))
		copy(buffer.charsets, buffer.savedCharsets)
		buffer.currentCharset = buffer.savedCurrentCharset
	}
}

func (buffer *Buffer) getCursorAttr() *CellAttributes {
	return &buffer.cursorAttr
}

func (buffer *Buffer) GetCell(viewCol uint16, viewRow uint16) *Cell {
	rawLine := buffer.convertViewLineToRawLine(viewRow)
	return buffer.getRawCell(viewCol, rawLine)
}

func (buffer *Buffer) getRawCell(viewCol uint16, rawLine uint64) *Cell {

	if viewCol < 0 || rawLine < 0 || int(rawLine) >= len(buffer.lines) {
		return nil
	}
	line := &buffer.lines[rawLine]
	if int(viewCol) >= len(line.cells) {
		return nil
	}
	return &line.cells[viewCol]
}

// Column returns cursor column
func (buffer *Buffer) CursorColumn() uint16 {
	// @todo originMode and left margin
	return buffer.cursorX
}

// CursorLineAbsolute returns absolute cursor line coordinate (ignoring Origin Mode)
func (buffer *Buffer) CursorLineAbsolute() uint16 {
	return buffer.cursorY
}

// CursorLine returns cursor line (in Origin Mode it is relative to the top margin)
func (buffer *Buffer) CursorLine() uint16 {
	if buffer.modes.OriginMode {
		result := buffer.cursorY - uint16(buffer.topMargin)
		if result < 0 {
			result = 0
		}
		return result
	}
	return buffer.cursorY
}

func (buffer *Buffer) TopMargin() uint {
	return buffer.topMargin
}

func (buffer *Buffer) BottomMargin() uint {
	return buffer.bottomMargin
}

// translates the cursor line to the raw buffer line
func (buffer *Buffer) RawLine() uint64 {
	return buffer.convertViewLineToRawLine(buffer.cursorY)
}

func (buffer *Buffer) convertViewLineToRawLine(viewLine uint16) uint64 {
	rawHeight := buffer.Height()
	if int(buffer.viewHeight) > rawHeight {
		return uint64(viewLine)
	}
	return uint64(int(viewLine) + (rawHeight - int(buffer.viewHeight)))
}

func (buffer *Buffer) convertRawLineToViewLine(rawLine uint64) uint16 {
	rawHeight := buffer.Height()
	if int(buffer.viewHeight) > rawHeight {
		return uint16(rawLine)
	}
	return uint16(int(rawLine) - (rawHeight - int(buffer.viewHeight)))
}

func (buffer *Buffer) GetVPosition() int {
	result := int(uint(buffer.Height()) - uint(buffer.ViewHeight()) - buffer.scrollLinesFromBottom)
	if result < 0 {
		result = 0
	}

	return result
}

// Width returns the width of the buffer in columns
func (buffer *Buffer) Width() uint16 {
	return buffer.viewWidth
}

func (buffer *Buffer) ViewWidth() uint16 {
	return buffer.viewWidth
}

func (buffer *Buffer) Height() int {
	return len(buffer.lines)
}

func (buffer *Buffer) ViewHeight() uint16 {
	return buffer.viewHeight
}

func (buffer *Buffer) deleteLine() {
	index := int(buffer.RawLine())
	buffer.lines = buffer.lines[:index+copy(buffer.lines[index:], buffer.lines[index+1:])]
}

func (buffer *Buffer) insertLine() {

	if !buffer.InScrollableRegion() {
		pos := buffer.RawLine()
		maxLines := buffer.GetMaxLines()
		newLineCount := uint64(len(buffer.lines) + 1)
		if newLineCount > maxLines {
			newLineCount = maxLines
		}

		out := make([]Line, newLineCount)
		copy(
			out[:pos-(uint64(len(buffer.lines))+1-newLineCount)],
			buffer.lines[uint64(len(buffer.lines))+1-newLineCount:pos])
		out[pos] = newLine()
		copy(out[pos+1:], buffer.lines[pos:])
		buffer.lines = out
	} else {
		topIndex := buffer.convertViewLineToRawLine(uint16(buffer.topMargin))
		bottomIndex := buffer.convertViewLineToRawLine(uint16(buffer.bottomMargin))
		before := buffer.lines[:topIndex]
		after := buffer.lines[bottomIndex+1:]
		out := make([]Line, len(buffer.lines))
		copy(out[0:], before)

		pos := buffer.RawLine()
		for i := topIndex; i < bottomIndex; i++ {
			if i < pos {
				out[i] = buffer.lines[i]
			} else {
				out[i+1] = buffer.lines[i]
			}
		}

		copy(out[bottomIndex+1:], after)

		out[pos] = newLine()
		buffer.lines = out
	}
}

func (buffer *Buffer) insertBlankCharacters(count int) {

	index := int(buffer.RawLine())
	for i := 0; i < count; i++ {
		cells := buffer.lines[index].cells
		buffer.lines[index].cells = append(cells[:buffer.cursorX], append([]Cell{buffer.defaultCell(true)}, cells[buffer.cursorX:]...)...)
	}
}

func (buffer *Buffer) insertLines(count int) {

	if buffer.HasScrollableRegion() && !buffer.InScrollableRegion() {
		// should have no effect outside of scrollable region
		return
	}

	buffer.cursorX = 0

	for i := 0; i < count; i++ {
		buffer.insertLine()
	}

}

func (buffer *Buffer) deleteLines(count int) {

	if buffer.HasScrollableRegion() && !buffer.InScrollableRegion() {
		// should have no effect outside of scrollable region
		return
	}

	buffer.cursorX = 0

	for i := 0; i < count; i++ {
		buffer.deleteLine()
	}

}

func (buffer *Buffer) index() {

	// This sequence causes the active position to move downward one line without changing the column position.
	// If the active position is at the bottom margin, a scroll up is performed."

	if buffer.InScrollableRegion() {

		if uint(buffer.cursorY) < buffer.bottomMargin {
			buffer.cursorY++
		} else {
			buffer.areaScrollUp(1)
		}

		return
	}

	if buffer.cursorY >= buffer.ViewHeight()-1 {
		buffer.lines = append(buffer.lines, newLine())
		maxLines := buffer.GetMaxLines()
		if uint64(len(buffer.lines)) > maxLines {
			copy(buffer.lines, buffer.lines[uint64(len(buffer.lines))-maxLines:])
			buffer.lines = buffer.lines[:maxLines]
		}
	} else {
		buffer.cursorY++
	}
}

func (buffer *Buffer) reverseIndex() {

	if uint(buffer.cursorY) == buffer.topMargin {
		buffer.areaScrollDown(1)
	} else if buffer.cursorY > 0 {
		buffer.cursorY--
	}
}

// write will write a rune to the terminal at the position of the cursor, and increment the cursor position
func (buffer *Buffer) write(runes ...MeasuredRune) {

	// scroll to bottom on input
	buffer.scrollLinesFromBottom = 0

	for _, r := range runes {

		line := buffer.getCurrentLine()

		if buffer.modes.ReplaceMode {

			if buffer.CursorColumn() >= buffer.Width() {
				// @todo replace rune at position 0 on next line down
				return
			}

			for int(buffer.CursorColumn()) >= len(line.cells) {
				line.append(buffer.defaultCell(int(buffer.CursorColumn()) == len(line.cells)))
			}
			line.cells[buffer.cursorX].attr = buffer.cursorAttr
			line.cells[buffer.cursorX].setRune(r)
			buffer.incrementCursorPosition()
			continue
		}

		if buffer.CursorColumn() >= buffer.Width() { // if we're after the line, move to next

			if buffer.modes.AutoWrap {

				buffer.newLineEx(true)

				newLine := buffer.getCurrentLine()
				newLine.setNoBreak(true)
				if len(newLine.cells) == 0 {
					newLine.append(buffer.defaultCell(true))
				}
				cell := &newLine.cells[0]
				cell.setRune(r)
				cell.attr = buffer.cursorAttr

			} else {
				// no more room on line and wrapping is disabled
				return
			}

			// @todo if next line is wrapped then prepend to it and shuffle characters along line, wrapping to next if necessary
		} else {

			for int(buffer.CursorColumn()) >= len(line.cells) {
				line.append(buffer.defaultCell(int(buffer.CursorColumn()) == len(line.cells)))
			}

			cell := &line.cells[buffer.CursorColumn()]
			cell.setRune(r)
			cell.attr = buffer.cursorAttr
		}

		buffer.incrementCursorPosition()
	}
}

func (buffer *Buffer) incrementCursorPosition() {
	// we can increment one column past the end of the line.
	// this is effectively the beginning of the next line, except when we \r etc.
	if buffer.CursorColumn() < buffer.Width() {
		buffer.cursorX++
	}
}

func (buffer *Buffer) inDoWrap() bool {
	// xterm uses 'do_wrap' flag for this special terminal state
	// we use the cursor position right after the boundary
	// let's see how it works out
	return buffer.cursorX == buffer.viewWidth // @todo rightMargin
}

func (buffer *Buffer) backspace() {

	if buffer.cursorX == 0 {
		line := buffer.getCurrentLine()
		if line.wrapped {
			buffer.movePosition(int16(buffer.Width()-1), -1)
		} else {
			//@todo ring bell or whatever - actually i think the pty will trigger this
		}
	} else if buffer.inDoWrap() {
		// the "do_wrap" implementation
		buffer.movePosition(-2, 0)
	} else {
		buffer.movePosition(-1, 0)
	}
}

func (buffer *Buffer) carriageReturn() {

	for {
		line := buffer.getCurrentLine()
		if line == nil {
			break
		}
		if line.wrapped && buffer.cursorY > 0 {
			buffer.cursorY--
		} else {
			break
		}
	}

	buffer.cursorX = 0
}

func (buffer *Buffer) tab() {

	tabStop := buffer.getNextTabStopAfter(buffer.cursorX)
	for buffer.cursorX < tabStop && buffer.cursorX < buffer.viewWidth-1 { // @todo rightMargin
		buffer.write(MeasuredRune{Rune: ' ', Width: 1})
	}
}

// return next tab stop x pos
func (buffer *Buffer) getNextTabStopAfter(col uint16) uint16 {

	defaultStop := col + (TabSize - (col % TabSize))
	if defaultStop == col {
		defaultStop += TabSize
	}

	var low uint16
	for _, stop := range buffer.tabStops {
		if stop > col {
			if stop < low || low == 0 {
				low = stop
			}
		}
	}

	if low == 0 {
		return defaultStop
	}

	return low
}

func (buffer *Buffer) newLine() {
	buffer.newLineEx(false)
}

func (buffer *Buffer) verticalTab() {
	buffer.index()

	for {
		line := buffer.getCurrentLine()
		if !line.wrapped {
			break
		}
		buffer.index()
	}
}

func (buffer *Buffer) newLineEx(forceCursorToMargin bool) {

	if buffer.IsNewLineMode() || forceCursorToMargin {
		buffer.cursorX = 0
	}
	buffer.index()

	for {
		line := buffer.getCurrentLine()
		if !line.wrapped {
			break
		}
		buffer.index()
	}
}

func (buffer *Buffer) movePosition(x int16, y int16) {

	var toX uint16
	var toY uint16

	if int16(buffer.CursorColumn())+x < 0 {
		toX = 0
	} else {
		toX = uint16(int16(buffer.CursorColumn()) + x)
	}

	// should either use CursorLine() and setPosition() or use absolutes, mind Origin Mode (DECOM)
	if int16(buffer.CursorLine())+y < 0 {
		toY = 0
	} else {
		toY = uint16(int16(buffer.CursorLine()) + y)
	}

	buffer.setPosition(toX, toY)
}

func (buffer *Buffer) setPosition(col uint16, line uint16) {

	useCol := col
	useLine := line
	maxLine := buffer.ViewHeight() - 1

	if buffer.modes.OriginMode {
		useLine += uint16(buffer.topMargin)
		maxLine = uint16(buffer.bottomMargin)
		// @todo left and right margins
	}
	if useLine > maxLine {
		useLine = maxLine
	}

	if useCol >= buffer.ViewWidth() {
		useCol = buffer.ViewWidth() - 1
	}

	buffer.cursorX = useCol
	buffer.cursorY = useLine
}

func (buffer *Buffer) GetVisibleLines() []Line {
	lines := []Line{}

	for i := buffer.Height() - int(buffer.ViewHeight()); i < buffer.Height(); i++ {
		y := i - int(buffer.scrollLinesFromBottom)
		if y >= 0 && y < len(buffer.lines) {
			lines = append(lines, buffer.lines[y])
		}
	}
	return lines
}

// tested to here

func (buffer *Buffer) clear() {
	for i := 0; i < int(buffer.ViewHeight()); i++ {
		buffer.lines = append(buffer.lines, newLine())
	}
	buffer.setPosition(0, 0)
}

func (buffer *Buffer) reallyClear() {
	buffer.lines = []Line{}
	buffer.SetScrollOffset(0)
	buffer.setPosition(0, 0)
}

// creates if necessary
func (buffer *Buffer) getCurrentLine() *Line {
	return buffer.getViewLine(buffer.cursorY)
}

func (buffer *Buffer) getViewLine(index uint16) *Line {

	if index >= buffer.ViewHeight() { // @todo is this okay? error?
		return &buffer.lines[len(buffer.lines)-1]
	}

	if len(buffer.lines) < int(buffer.ViewHeight()) {
		for int(index) >= len(buffer.lines) {
			buffer.lines = append(buffer.lines, newLine())
		}
		return &buffer.lines[int(index)]
	}

	if int(buffer.convertViewLineToRawLine(index)) < len(buffer.lines) {
		return &buffer.lines[buffer.convertViewLineToRawLine(index)]
	}

	panic(fmt.Sprintf("Failed to retrieve line for %d", index))
}

func (buffer *Buffer) eraseLine() {
	line := buffer.getCurrentLine()
	line.cells = []Cell{}
}

func (buffer *Buffer) eraseLineToCursor() {
	line := buffer.getCurrentLine()
	for i := 0; i <= int(buffer.cursorX); i++ {
		if i < len(line.cells) {
			line.cells[i].erase(buffer.cursorAttr.bgColour)
		}
	}
}

func (buffer *Buffer) eraseLineFromCursor() {
	line := buffer.getCurrentLine()

	if len(line.cells) > 0 {
		cx := buffer.cursorX
		if int(cx) < len(line.cells) {
			line.cells = line.cells[:buffer.cursorX]
		}
	}
	max := int(buffer.ViewWidth()) - len(line.cells)

	for i := 0; i < max; i++ {
		line.append(buffer.defaultCell(true))
	}

}

func (buffer *Buffer) eraseDisplay() {
	for i := uint16(0); i < (buffer.ViewHeight()); i++ {
		rawLine := buffer.convertViewLineToRawLine(i)
		if int(rawLine) < len(buffer.lines) {
			buffer.lines[int(rawLine)].cells = []Cell{}
		}
	}
}

func (buffer *Buffer) deleteChars(n int) {

	line := buffer.getCurrentLine()
	if int(buffer.cursorX) >= len(line.cells) {
		return
	}
	before := line.cells[:buffer.cursorX]
	if int(buffer.cursorX)+n >= len(line.cells) {
		n = len(line.cells) - int(buffer.cursorX)
	}
	after := line.cells[int(buffer.cursorX)+n:]
	line.cells = append(before, after...)
}

func (buffer *Buffer) eraseCharacters(n int) {

	line := buffer.getCurrentLine()

	max := int(buffer.cursorX) + n
	if max > len(line.cells) {
		max = len(line.cells)
	}

	for i := int(buffer.cursorX); i < max; i++ {
		line.cells[i].erase(buffer.cursorAttr.bgColour)
	}
}

func (buffer *Buffer) eraseDisplayFromCursor() {
	line := buffer.getCurrentLine()

	max := int(buffer.cursorX)
	if max > len(line.cells) {
		max = len(line.cells)
	}

	line.cells = line.cells[:max]

	for rawLine := buffer.convertViewLineToRawLine(buffer.cursorY) + 1; int(rawLine) < len(buffer.lines); rawLine++ {
		buffer.lines[int(rawLine)].cells = []Cell{}
	}
}

func (buffer *Buffer) eraseDisplayToCursor() {
	line := buffer.getCurrentLine()

	for i := 0; i <= int(buffer.cursorX); i++ {
		if i >= len(line.cells) {
			break
		}
		line.cells[i].erase(buffer.cursorAttr.bgColour)
	}
	for i := uint16(0); i < buffer.cursorY; i++ {
		rawLine := buffer.convertViewLineToRawLine(i)
		if int(rawLine) < len(buffer.lines) {
			buffer.lines[int(rawLine)].cells = []Cell{}
		}
	}
}

func (buffer *Buffer) resizeView(width uint16, height uint16) {

	if buffer.viewHeight == 0 {
		buffer.viewWidth = width
		buffer.viewHeight = height
		return
	}

	// @todo scroll to bottom on resize
	line := buffer.getCurrentLine()
	cXFromEndOfLine := len(line.cells) - int(buffer.cursorX+1)

	cursorYMovement := 0

	if width < buffer.viewWidth { // wrap lines if we're shrinking
		for i := 0; i < len(buffer.lines); i++ {
			line := &buffer.lines[i]
			//line.cleanse()
			if len(line.cells) > int(width) { // only try wrapping a line if it's too long
				sillyCells := line.cells[width:] // grab the cells we need to wrap
				line.cells = line.cells[:width]

				// we need to move cut cells to the next line
				// if the next line is wrapped anyway, we can push them onto the beginning of that line
				// otherwise, we need add a new wrapped line
				if i+1 < len(buffer.lines) {
					nextLine := &buffer.lines[i+1]
					if nextLine.wrapped {

						nextLine.cells = append(sillyCells, nextLine.cells...)
						continue
					}
				}

				if i+1 <= int(buffer.cursorY) {
					cursorYMovement++
				}

				newLine := newLine()
				newLine.setWrapped(true)
				newLine.cells = sillyCells
				after := append([]Line{newLine}, buffer.lines[i+1:]...)
				buffer.lines = append(buffer.lines[:i+1], after...)

			}
		}
	} else if width > buffer.viewWidth { // unwrap lines if we're growing
		for i := 0; i < len(buffer.lines)-1; i++ {
			line := &buffer.lines[i]
			//line.cleanse()
			for offset := 1; i+offset < len(buffer.lines); offset++ {
				nextLine := &buffer.lines[i+offset]
				//nextLine.cleanse()
				if !nextLine.wrapped { // if the next line wasn't wrapped, we don't need to move characters back to this line
					break
				}
				spaceOnLine := int(width) - len(line.cells)
				if spaceOnLine <= 0 { // no more space to unwrap
					break
				}
				moveCount := spaceOnLine
				if moveCount > len(nextLine.cells) {
					moveCount = len(nextLine.cells)
				}
				line.append(nextLine.cells[:moveCount]...)
				if moveCount == len(nextLine.cells) {

					if i+offset <= int(buffer.cursorY) {
						cursorYMovement--
					}

					// if we unwrapped all cells off the next line, delete it
					buffer.lines = append(buffer.lines[:i+offset], buffer.lines[i+offset+1:]...)

					offset--

				} else {
					// otherwise just remove the characters we moved up a line
					nextLine.cells = nextLine.cells[moveCount:]
				}
			}

		}
	}

	buffer.viewWidth = width
	buffer.viewHeight = height

	cY := uint16(len(buffer.lines) - 1)
	if cY >= buffer.viewHeight {
		cY = buffer.viewHeight - 1
	}
	buffer.cursorY = cY

	// position cursorX
	line = buffer.getCurrentLine()
	buffer.cursorX = uint16((len(line.cells) - cXFromEndOfLine) - 1)

	buffer.resetVerticalMargins(uint(buffer.viewHeight))
}

func (buffer *Buffer) GetMaxLines() uint64 {
	result := buffer.maxLines
	if result < uint64(buffer.viewHeight) {
		result = uint64(buffer.viewHeight)
	}

	return result
}

func (buffer *Buffer) SaveViewLines(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	for i := uint16(0); i <= buffer.ViewHeight(); i++ {
		if _, err := f.WriteString(buffer.getViewLine(i).String()); err != nil {
			return err
		}
	}

	return nil
}

func (buffer *Buffer) CompareViewLines(path string) bool {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	bufferContent := []byte{}
	for i := uint16(0); i <= buffer.ViewHeight(); i++ {
		lineBytes := []byte(buffer.getViewLine(i).String())
		bufferContent = append(bufferContent, lineBytes...)
	}
	return bytes.Equal(f, bufferContent)
}

func (buffer *Buffer) reverseVideo() {
	for _, line := range buffer.lines {
		line.reverseVideo()
	}
}

func (buffer *Buffer) setVerticalMargins(top uint, bottom uint) {
	buffer.topMargin = top
	buffer.bottomMargin = bottom
}

// resetVerticalMargins resets margins to extreme positions
func (buffer *Buffer) resetVerticalMargins(height uint) {
	buffer.setVerticalMargins(0, height-1)
}

func (buffer *Buffer) defaultCell(applyEffects bool) Cell {
	attr := buffer.cursorAttr
	if !applyEffects {
		attr.blink = false
		attr.bold = false
		attr.dim = false
		attr.inverse = false
		attr.underline = false
		attr.dim = false
	}
	return Cell{attr: attr}
}

func (buffer *Buffer) IsNewLineMode() bool {
	return buffer.modes.LineFeedMode == false
}

func (buffer *Buffer) tabReset() {
	buffer.tabStops = nil
}

func (buffer *Buffer) tabSet(index uint16) {
	buffer.tabStops = append(buffer.tabStops, index)
}

func (buffer *Buffer) tabClear(index uint16) {
	var filtered []uint16
	for _, stop := range buffer.tabStops {
		if stop != buffer.cursorX {
			filtered = append(filtered, stop)
		}
	}
	buffer.tabStops = filtered
}

func (buffer *Buffer) IsTabSetAtCursor() bool {
	if buffer.cursorX%TabSize > 0 {
		return false
	}
	for _, stop := range buffer.tabStops {
		if stop == buffer.cursorX {
			return true
		}
	}
	return false
}

func (buffer *Buffer) tabClearAtCursor() {
	buffer.tabClear(buffer.cursorX)
}

func (buffer *Buffer) tabSetAtCursor() {
	buffer.tabSet(buffer.cursorX)
}

func (buffer *Buffer) GetScrollOffset() uint {
	return buffer.scrollLinesFromBottom
}

func (buffer *Buffer) SetScrollOffset(offset uint) {
	buffer.scrollLinesFromBottom = offset
}
