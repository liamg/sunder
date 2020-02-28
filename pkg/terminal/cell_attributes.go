package terminal

import "strings"

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

func (cellAttr *CellAttributes) reverseVideo() {
	oldFgColour := cellAttr.fgColour
	cellAttr.fgColour = cellAttr.bgColour
	cellAttr.bgColour = oldFgColour
}

// GetDiffANSI takes a previous cell attribute set and diffs to this one, producing the
// most efficient ANSI output to achieve the diff
func (cellAttr CellAttributes) GetDiffANSI(prev CellAttributes) string {
	var reset bool
	var segments []string

	if cellAttr.bgColour == "" {
		cellAttr.bgColour = "0"
	}

	// set fg
	if prev.fgColour != cellAttr.fgColour {
		if cellAttr.fgColour == "" || cellAttr.fgColour == "0" {
			reset = true
		} else {
			segments = append(segments, string(cellAttr.fgColour))
		}
	}

	// set bg
	if prev.bgColour != cellAttr.bgColour {
		if cellAttr.bgColour == "" || cellAttr.bgColour == "0" {
			reset = true
		} else {
			segments = append(segments, string(cellAttr.bgColour))
		}
	}

	// TODO add sequences for bold, dim, blink etc. diffs

	if reset {
		segments = append([]string{"0"}, segments...)
	}

	if len(segments) == 0 {
		return ""
	}

	return "\x1b[" + strings.Join(segments, ";") + "m"
}
