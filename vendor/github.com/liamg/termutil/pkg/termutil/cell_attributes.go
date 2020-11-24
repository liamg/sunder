package termutil

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

	var segments []string

	// set fg
	if prev.fgColour != cellAttr.fgColour {
		if cellAttr.fgColour == "" {
			segments = append(segments, "39")
		} else {
			segments = append(segments, string(cellAttr.fgColour))
		}
	}

	// set bg
	if prev.bgColour != cellAttr.bgColour {
		if cellAttr.bgColour == "" {
			segments = append(segments, "49")
		} else {
			segments = append(segments, string(cellAttr.bgColour))
		}
	}

	// TODO add sequences for bold, dim, blink etc. diffs

	if len(segments) == 0 {
		return ""
	}

	return "\x1b[" + strings.Join(segments, ";") + "m"
}
