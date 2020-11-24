package termutil

type Cell struct {
	r    MeasuredRune
	attr CellAttributes
}

func (cell *Cell) Attr() CellAttributes {
	return cell.attr
}

func (cell *Cell) Rune() MeasuredRune {
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
	cell.setRune(MeasuredRune{Rune: 0})
	cell.attr.bgColour = bgColour
}

func (cell *Cell) setRune(r MeasuredRune) {
	cell.r = r
}
