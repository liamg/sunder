package termutil

import (
	"fmt"
	"strconv"
	"strings"
)

func parseCSI(readChan chan MeasuredRune) (final rune, params []string, intermediate []rune, raw []rune) {
	var b MeasuredRune

	param := ""
	intermediate = []rune{}
CSI:
	for {
		b = <-readChan
		raw = append(raw, b.Rune)
		switch true {
		case b.Rune >= 0x30 && b.Rune <= 0x3F:
			param = param + string(b.Rune)
		case b.Rune > 0 && b.Rune <= 0x2F:
			intermediate = append(intermediate, b.Rune)
		case b.Rune >= 0x40 && b.Rune <= 0x7e:
			final = b.Rune
			break CSI
		}
	}

	unprocessed := strings.Split(param, ";")
	for _, par := range unprocessed {
		if par != "" {
			par = strings.TrimLeft(par, "0")
			if par == "" {
				par = "0"
			}
			params = append(params, par)
		}
	}

	return final, params, intermediate, raw
}

func (t *Terminal) handleCSI(readChan chan MeasuredRune) (renderRequired bool) {
	final, params, intermediate, raw := parseCSI(readChan)

	t.log("CSI P(%q) I(%q) %c", strings.Join(params, ";"), string(intermediate), final)

	for _, b := range intermediate {
		t.processRunes(MeasuredRune{
			Rune:  b,
			Width: 1, // TODO: measure these? should only be control characters...
		})
	}

	switch final {
	case 'c':
		return t.csiSendDeviceAttributesHandler(params)
	case 'd':
		return t.csiLinePositionAbsoluteHandler(params)
	case 'f':
		return t.csiCursorPositionHandler(params)
	case 'g':
		return t.csiTabClearHandler(params)
	case 'h':
		return t.csiSetModeHandler(params)
	case 'l':
		return t.csiResetModeHandler(params)
	case 'm':
		return t.sgrSequenceHandler(params)
	case 'n':
		return t.csiDeviceStatusReportHandler(params)
	case 'r':
		return t.csiSetMarginsHandler(params)
	//case 't':
	// TODO return t.csiWindowManipulation(params)
	case 'A':
		return t.csiCursorUpHandler(params)
	case 'B':
		return t.csiCursorDownHandler(params)
	case 'C':
		return t.csiCursorForwardHandler(params)
	case 'D':
		return t.csiCursorBackwardHandler(params)
	case 'E':
		return t.csiCursorNextLineHandler(params)
	case 'F':
		return t.csiCursorPrecedingLineHandler(params)
	case 'G':
		return t.csiCursorCharacterAbsoluteHandler(params)
	case 'H':
		return t.csiCursorPositionHandler(params)
	case 'J':
		return t.csiEraseInDisplayHandler(params)
	case 'K':
		return t.csiEraseInLineHandler(params)
	case 'L':
		return t.csiInsertLinesHandler(params)
	case 'M':
		return t.csiDeleteLinesHandler(params)
	case 'P':
		return t.csiDeleteHandler(params)
	case 'S':
		return t.csiScrollUpHandler(params)
	case 'T':
		return t.csiScrollDownHandler(params)
	case 'X':
		return t.csiEraseCharactersHandler(params)
	case '@':
		return t.csiInsertBlankCharactersHandler(params)
	default:
		// TODO review this:
		// if this is an unknown CSI sequence, write it to stdout as we can't handle it?
		//_ = t.writeToRealStdOut(append([]rune{0x1b, '['}, raw...)...)
		_ = raw
		t.log("UNKNOWN CSI P(%s) I(%s) %c", strings.Join(params, ";"), string(intermediate), final)
		return false
	}

}

// CSI c
// Send Device Attributes (Primary/Secondary/Tertiary DA)
func (t *Terminal) csiSendDeviceAttributesHandler(params []string) (renderRequired bool) {

	// we are VT100
	// for DA1 we'll respond ?1;2
	// for DA2 we'll respond >0;0;0
	response := "?1;2"
	if len(params) > 0 && len(params[0]) > 0 && params[0][0] == '>' {
		response = ">0;0;0"
	}

	// write response to source pty
	t.respondToPty([]byte("\x1b[" + response + "c"))
	return false
}

// CSI n
// Device Status Report (DSR)
func (t *Terminal) csiDeviceStatusReportHandler(params []string) (renderRequired bool) {

	if len(params) == 0 {
		return false
	}

	switch params[0] {
	case "5":
		t.respondToPty([]byte("\x1b[0n")) // everything is cool
	case "6": // report cursor position
		t.respondToPty([]byte(fmt.Sprintf(
			"\x1b[%d;%dR",
			t.GetActiveBuffer().CursorLine()+1,
			t.GetActiveBuffer().CursorColumn()+1,
		)))
	}

	return false
}

// CSI A
// Cursor Up Ps Times (default = 1) (CUU)
func (t *Terminal) csiCursorUpHandler(params []string) (renderRequired bool) {
	distance := 1
	if len(params) > 0 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}
	t.GetActiveBuffer().movePosition(0, -int16(distance))
	return true
}

// CSI B
// Cursor Down Ps Times (default = 1) (CUD)
func (t *Terminal) csiCursorDownHandler(params []string) (renderRequired bool) {
	distance := 1
	if len(params) > 0 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}

	t.GetActiveBuffer().movePosition(0, int16(distance))
	return true
}

// CSI C
// Cursor Forward Ps Times (default = 1) (CUF)
func (t *Terminal) csiCursorForwardHandler(params []string) (renderRequired bool) {
	distance := 1
	if len(params) > 0 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}

	t.GetActiveBuffer().movePosition(int16(distance), 0)
	return true
}

// CSI D
// Cursor Backward Ps Times (default = 1) (CUB)
func (t *Terminal) csiCursorBackwardHandler(params []string) (renderRequired bool) {
	distance := 1
	if len(params) > 0 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}

	t.GetActiveBuffer().movePosition(-int16(distance), 0)
	return true
}

// CSI E
// Cursor Next Line Ps Times (default = 1) (CNL)
func (t *Terminal) csiCursorNextLineHandler(params []string) (renderRequired bool) {

	distance := 1
	if len(params) > 0 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}

	t.GetActiveBuffer().movePosition(0, int16(distance))
	t.GetActiveBuffer().setPosition(0, t.GetActiveBuffer().CursorLine())
	return true
}

// CSI F
// Cursor Preceding Line Ps Times (default = 1) (CPL)
func (t *Terminal) csiCursorPrecedingLineHandler(params []string) (renderRequired bool) {

	distance := 1
	if len(params) > 0 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}
	t.GetActiveBuffer().movePosition(0, -int16(distance))
	t.GetActiveBuffer().setPosition(0, t.GetActiveBuffer().CursorLine())
	return true
}

// CSI G
// Cursor Horizontal Absolute  [column] (default = [row,1]) (CHA)
func (t *Terminal) csiCursorCharacterAbsoluteHandler(params []string) (renderRequired bool) {
	distance := 1
	if len(params) > 0 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || params[0] == "" {
			distance = 1
		}
	}

	t.GetActiveBuffer().setPosition(uint16(distance-1), t.GetActiveBuffer().CursorLine())
	return true
}

func parseCursorPosition(params []string) (x, y int) {
	x, y = 1, 1
	if len(params) >= 1 {
		var err error
		if params[0] != "" {
			y, err = strconv.Atoi(string(params[0]))
			if err != nil || y < 1 {
				y = 1
			}
		}
	}
	if len(params) >= 2 {
		if params[1] != "" {
			var err error
			x, err = strconv.Atoi(string(params[1]))
			if err != nil || x < 1 {
				x = 1
			}
		}
	}
	return x, y
}

// CSI f
// Horizontal and Vertical Position [row;column] (default = [1,1]) (HVP)
// AND
// CSI H
// Cursor Position [row;column] (default = [1,1]) (CUP)
func (t *Terminal) csiCursorPositionHandler(params []string) (renderRequired bool) {
	x, y := parseCursorPosition(params)
	t.GetActiveBuffer().setPosition(uint16(x-1), uint16(y-1))
	return true
}

// CSI S
// Scroll up Ps lines (default = 1) (SU), VT420, ECMA-48
func (t *Terminal) csiScrollUpHandler(params []string) (renderRequired bool) {
	distance := 1
	if len(params) > 1 {
		return false
	}
	if len(params) == 1 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}
	t.GetActiveBuffer().areaScrollUp(uint16(distance))
	return true
}

// CSI @
// Insert Ps (Blank) Character(s) (default = 1) (ICH)
func (t *Terminal) csiInsertBlankCharactersHandler(params []string) (renderRequired bool) {
	count := 1
	if len(params) > 1 {
		return false
	}
	if len(params) == 1 {
		var err error
		count, err = strconv.Atoi(params[0])
		if err != nil || count < 1 {
			count = 1
		}
	}

	t.GetActiveBuffer().insertBlankCharacters(count)
	return true
}

// CSI L
// Insert Ps Line(s) (default = 1) (IL)
func (t *Terminal) csiInsertLinesHandler(params []string) (renderRequired bool) {
	count := 1
	if len(params) > 1 {
		return false
	}
	if len(params) == 1 {
		var err error
		count, err = strconv.Atoi(params[0])
		if err != nil || count < 1 {
			count = 1
		}
	}

	t.GetActiveBuffer().insertLines(count)
	return true
}

// CSI M
// Delete Ps Line(s) (default = 1) (DL)
func (t *Terminal) csiDeleteLinesHandler(params []string) (renderRequired bool) {
	count := 1
	if len(params) > 1 {
		return false
	}
	if len(params) == 1 {
		var err error
		count, err = strconv.Atoi(params[0])
		if err != nil || count < 1 {
			count = 1
		}
	}

	t.GetActiveBuffer().deleteLines(count)
	return true
}

// CSI T
// Scroll down Ps lines (default = 1) (SD), VT420
func (t *Terminal) csiScrollDownHandler(params []string) (renderRequired bool) {
	distance := 1
	if len(params) > 1 {
		return false
	}
	if len(params) == 1 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}
	t.GetActiveBuffer().areaScrollDown(uint16(distance))
	return true
}

// CSI r
// Set Scrolling Region [top;bottom] (default = full size of window) (DECSTBM), VT100
func (t *Terminal) csiSetMarginsHandler(params []string) (renderRequired bool) {
	top := 1
	bottom := int(t.GetActiveBuffer().ViewHeight())

	if len(params) > 2 {
		return false
	}

	if len(params) > 0 {
		var err error
		top, err = strconv.Atoi(params[0])
		if err != nil || top < 1 {
			top = 1
		}

		if len(params) > 1 {
			var err error
			bottom, err = strconv.Atoi(params[1])
			if err != nil || bottom > int(t.GetActiveBuffer().ViewHeight()) || bottom < 1 {
				bottom = int(t.GetActiveBuffer().ViewHeight())
			}
		}
	}
	top--
	bottom--

	t.activeBuffer.setVerticalMargins(uint(top), uint(bottom))
	t.GetActiveBuffer().setPosition(0, 0)
	return true
}

// CSI X
// Erase Ps Character(s) (default = 1) (ECH)
func (t *Terminal) csiEraseCharactersHandler(params []string) (renderRequired bool) {
	count := 1
	if len(params) > 0 {
		var err error
		count, err = strconv.Atoi(params[0])
		if err != nil || count < 1 {
			count = 1
		}
	}

	t.GetActiveBuffer().eraseCharacters(count)
	return true
}

// CSI l
// Reset Mode (RM)
func (t *Terminal) csiResetModeHandler(params []string) (renderRequired bool) {
	return t.csiSetModes(params, false)
}

// CSI h
// Set Mode (SM)
func (t *Terminal) csiSetModeHandler(params []string) (renderRequired bool) {
	return t.csiSetModes(params, true)
}

func (t *Terminal) csiSetModes(modes []string, enabled bool) bool {
	if len(modes) == 0 {
		return false
	}
	if len(modes) == 1 {
		return t.csiSetMode(modes[0], enabled)
	}
	// should we propagate DEC prefix?
	const decPrefix = '?'
	isDec := len(modes[0]) > 0 && modes[0][0] == decPrefix

	var render bool

	// iterate through params, propagating DEC prefix to subsequent elements
	for i, v := range modes {
		updatedMode := v
		if i > 0 && isDec {
			updatedMode = string(decPrefix) + v
		}
		render = t.csiSetMode(updatedMode, enabled) || render
	}

	return render
}

func (t *Terminal) csiSetMode(modeStr string, enabled bool) bool {

	/*
	   Mouse support

	   		#define SET_X10_MOUSE               9
	        #define SET_VT200_MOUSE             1000
	        #define SET_VT200_HIGHLIGHT_MOUSE   1001
	        #define SET_BTN_EVENT_MOUSE         1002
	        #define SET_ANY_EVENT_MOUSE         1003

	        #define SET_FOCUS_EVENT_MOUSE       1004

	        #define SET_EXT_MODE_MOUSE          1005
	        #define SET_SGR_EXT_MODE_MOUSE      1006
	        #define SET_URXVT_EXT_MODE_MOUSE    1015

	        #define SET_ALTERNATE_SCROLL        1007
	*/

	switch modeStr {
	case "4":
		// TODO handle replace mode
		t.activeBuffer.modes.ReplaceMode = !enabled
	case "20":
		t.activeBuffer.modes.LineFeedMode = false
	case "?1":
		t.activeBuffer.modes.ApplicationCursorKeys = enabled
	case "?3":
		if enabled {
			// DECCOLM - COLumn mode, 132 characters per line
			t.activeBuffer.resizeView(132, t.activeBuffer.viewHeight)
		} else {
			// DECCOLM - 80 characters per line (erases screen)
			t.activeBuffer.resizeView(80, t.activeBuffer.viewHeight)
		}
		t.activeBuffer.clear()
		/*
			case "?4":
				// DECSCLM
				// @todo smooth scrolling / jump scrolling
		*/
	case "?5": // DECSCNM
		t.activeBuffer.modes.ScreenMode = enabled
	case "?6":
		// DECOM
		t.activeBuffer.modes.OriginMode = enabled
	case "?7":
		// auto-wrap mode
		//DECAWM
		t.activeBuffer.modes.AutoWrap = enabled
	case "?9":
		if enabled {
			//terminal.logger.Infof("Turning on X10 mouse mode")
			t.activeBuffer.mouseMode = (MouseModeX10)
		} else {
			//terminal.logger.Infof("Turning off X10 mouse mode")
			t.activeBuffer.mouseMode = (MouseModeNone)
		}
	case "?12", "?13":
		t.activeBuffer.modes.BlinkingCursor = enabled
	case "?25":
		t.activeBuffer.modes.ShowCursor = enabled
	case "?47", "?1047":
		if enabled {
			t.useAltBuffer()
		} else {
			t.useMainBuffer()
		}
	case "?1000", "?10061000": // ?10061000 seen from htop
		// enable mouse tracking
		// 1000 refers to ext mode for extended mouse click area - otherwise only x <= 255-31
		if enabled {
			//terminal.logger.Infof("Turning on VT200 mouse mode")
			t.activeBuffer.mouseMode = (MouseModeVT200)
		} else {
			//terminal.logger.Infof("Turning off VT200 mouse mode")
			t.activeBuffer.mouseMode = (MouseModeNone)
		}
	case "?1002":
		if enabled {
			//terminal.logger.Infof("Turning on Button Event mouse mode")
			t.activeBuffer.mouseMode = (MouseModeButtonEvent)
		} else {
			//terminal.logger.Infof("Turning off Button Event mouse mode")
			t.activeBuffer.mouseMode = (MouseModeNone)
		}
	case "?1003":
		//return errors.New("Any Event mouse mode is not supported")
		/*
			if enabled {
				terminal.logger.Infof("Turning on Any Event mouse mode")
				terminal.SetMouseMode(MouseModeAnyEvent)
			} else {
				terminal.logger.Infof("Turning off Any Event mouse mode")
				terminal.SetMouseMode(MouseModeNone)
			}
		*/
	case "?1005":
		//return errors.New("UTF-8 ext mouse mode is not supported")
		/*
			if enabled {
				terminal.logger.Infof("Turning on UTF-8 ext mouse mode")
				terminal.SetMouseExtMode(MouseExtUTF)
			} else {
				terminal.logger.Infof("Turning off UTF-8 ext mouse mode")
				terminal.SetMouseExtMode(MouseExtNone)
			}
		*/
	case "?1006":
		if enabled {
			//.logger.Infof("Turning on SGR ext mouse mode")
			t.activeBuffer.mouseExtMode = MouseExtSGR
		} else {
			//terminal.logger.Infof("Turning off SGR ext mouse mode")
			t.activeBuffer.mouseExtMode = (MouseExtNone)
		}
	case "?1015":
		if enabled {
			//terminal.logger.Infof("Turning on URXVT ext mouse mode")
			t.activeBuffer.mouseExtMode = (MouseExtURXVT)
		} else {
			//terminal.logger.Infof("Turning off URXVT ext mouse mode")
			t.activeBuffer.mouseExtMode = (MouseExtNone)
		}
	case "?1048":
		if enabled {
			t.GetActiveBuffer().saveCursor()
		} else {
			t.GetActiveBuffer().restoreCursor()
		}
	case "?1049":
		if enabled {
			t.useAltBuffer()
		} else {
			t.useMainBuffer()
		}
	case "?2004":
		t.activeBuffer.bracketedPasteMode = enabled
	default:
		//return fmt.Errorf("Unsupported CSI %s%s code", modeStr, recoverCodeFromEnabled(enabled))
	}

	return false
}

// CSI d
// Line Position Absolute  [row] (default = [1,column]) (VPA)
func (t *Terminal) csiLinePositionAbsoluteHandler(params []string) (renderRequired bool) {
	row := 1
	if len(params) > 0 {
		var err error
		row, err = strconv.Atoi(params[0])
		if err != nil || row < 1 {
			row = 1
		}
	}

	t.GetActiveBuffer().setPosition(t.GetActiveBuffer().CursorColumn(), uint16(row-1))

	return true
}

// CSI P
// Delete Ps Character(s) (default = 1) (DCH)
func (t *Terminal) csiDeleteHandler(params []string) (renderRequired bool) {
	n := 1
	if len(params) >= 1 {
		var err error
		n, err = strconv.Atoi(params[0])
		if err != nil || n < 1 {
			n = 1
		}
	}

	t.GetActiveBuffer().deleteChars(n)
	return true
}

// CSI g
// tab clear (TBC)
func (t *Terminal) csiTabClearHandler(params []string) (renderRequired bool) {
	n := "0"
	if len(params) > 0 {
		n = params[0]
	}
	switch n {
	case "0", "":
		t.activeBuffer.tabClearAtCursor()
	case "3":
		t.activeBuffer.tabReset()
	default:
		return false
	}

	return true
}

// CSI J
// Erase in Display (ED), VT100
func (t *Terminal) csiEraseInDisplayHandler(params []string) (renderRequired bool) {
	n := "0"
	if len(params) > 0 {
		n = params[0]
	}

	switch n {
	case "0", "":
		t.GetActiveBuffer().eraseDisplayFromCursor()
	case "1":
		t.GetActiveBuffer().eraseDisplayToCursor()
	case "2", "3":
		t.GetActiveBuffer().eraseDisplay()
	default:
		return false
	}

	return true
}

// CSI K
// Erase in Line (EL), VT100
func (t *Terminal) csiEraseInLineHandler(params []string) (renderRequired bool) {

	n := "0"
	if len(params) > 0 {
		n = params[0]
	}

	switch n {
	case "0", "": //erase adter cursor
		t.GetActiveBuffer().eraseLineFromCursor()
	case "1": // erase to cursor inclusive
		t.GetActiveBuffer().eraseLineToCursor()
	case "2": // erase entire
		t.GetActiveBuffer().eraseLine()
	default:
		return false
	}
	return true
}

// CSI m
// Character Attributes (SGR)
func (t *Terminal) sgrSequenceHandler(params []string) bool {

	if len(params) == 0 {
		params = []string{"0"}
	}

	for i := range params {

		p := strings.Replace(strings.Replace(params[i], "[", "", -1), "]", "", -1)

		switch p {
		case "00", "0", "":
			attr := t.GetActiveBuffer().getCursorAttr()
			*attr = CellAttributes{}
		case "1", "01":
			t.GetActiveBuffer().getCursorAttr().bold = true
		case "2", "02":
			t.GetActiveBuffer().getCursorAttr().dim = true
		case "4", "04":
			t.GetActiveBuffer().getCursorAttr().underline = true
		case "5", "05":
			t.GetActiveBuffer().getCursorAttr().blink = true
		case "7", "07":
			t.GetActiveBuffer().getCursorAttr().inverse = true
		case "8", "08":
			t.GetActiveBuffer().getCursorAttr().hidden = true
		case "21":
			t.GetActiveBuffer().getCursorAttr().bold = false
		case "22":
			t.GetActiveBuffer().getCursorAttr().dim = false
		case "23":
			// not italic
		case "24":
			t.GetActiveBuffer().getCursorAttr().underline = false
		case "25":
			t.GetActiveBuffer().getCursorAttr().blink = false
		case "27":
			t.GetActiveBuffer().getCursorAttr().inverse = false
		case "28":
			t.GetActiveBuffer().getCursorAttr().hidden = false
		case "29":
			// not strikethrough
		case "38": // set foreground
			t.GetActiveBuffer().getCursorAttr().fgColour = Colour(p + ";" + strings.Join(params[i:], ";"))
		case "48": // set background
			t.GetActiveBuffer().getCursorAttr().bgColour = Colour(p + ";" + strings.Join(params[i:], ";"))
		default:
			i, err := strconv.Atoi(p)
			if err != nil {
				return false
			}
			switch true {
			case i >= 30 && i <= 37, i >= 90 && i <= 97, i == 39:
				t.GetActiveBuffer().getCursorAttr().fgColour = Colour(p)
			case i >= 40 && i <= 47, i >= 100 && i <= 107, i == 49:
				t.GetActiveBuffer().getCursorAttr().bgColour = Colour(p)
			}

		}
	}

	return false
}
