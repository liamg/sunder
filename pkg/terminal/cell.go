package terminal

type Cell struct {
	r    rune
	attr CellAttributes
}

type CellAttributes struct {
	fgColour  Colour
	bgColour  Colour
	bold      bool
	dim       bool
	underline bool
	blink     bool
	inverse   bool
	hidden    bool
}

func (cell *Cell) Attr() CellAttributes {
	return cell.attr
}

func (cell *Cell) Rune() rune {
	return cell.r
}

func (cell *Cell) Fg() Colour {
	if cell.Attr().inverse {
		return cell.attr.bgColour
	}
	return cell.attr.fgColour
}

func (cell *Cell) Bg() Colour {
	if cell.Attr().inverse {
		return cell.attr.fgColour
	}
	return cell.attr.bgColour
}

func (cell *Cell) erase(bgColour Colour) {
	cell.setRune(0)
	cell.attr.bgColour = bgColour
}

func (cell *Cell) setRune(r rune) {
	cell.r = r
}

func (cellAttr *CellAttributes) ReverseVideo() {
	oldFgColour := cellAttr.fgColour
	cellAttr.fgColour = cellAttr.bgColour
	cellAttr.bgColour = oldFgColour
}
