package terminal

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh/terminal"
)

func parseCSI(readChan chan MeasuredRune) (final rune, param string, intermediate []rune, raw []rune) {
	var b MeasuredRune

	param = ""
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

	return final, param, intermediate, raw
}

func (t *Terminal) handleCSI(readChan chan MeasuredRune) (renderRequired bool) {
	final, param, intermediate, raw := parseCSI(readChan)

	for _, b := range intermediate {
		t.processRunes(MeasuredRune{
			Rune:  b,
			Width: 1, // TODO: measure these? should only be control characters...
		})
	}

	params := strings.Split(param, ";")
	if param == "" {
		params = []string{}
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
		//return t.sgrSequenceHandler(params)
	case 'n':
		return t.csiDeviceStatusReportHandler(params)
	case 'r':
		return t.csiSetMarginsHandler(params)
	case 't':
		// TODO return t.csiWindowManipulation(params)
		/*

				{id: 'm', handler: sgrSequenceHandler, description: "Character Attributes (SGR)"},

		*/

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
	case '':
		return t.(params)
	case '':
		return t.(params)
	case '':
		return t.(params)
	case '':
		return t.(params)
	case '':
		return t.(params)
	case '':
		return t.(params)
	default:
		// TODO review this:
		// if this is an unknown CSI sequence, write it to stdout as we can't handle it
		_ = t.writeToRealStdOut(append([]rune{0x1b, '['}, raw...)...)
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
			t.ActiveBuffer().CursorLine()+1,
			t.ActiveBuffer().CursorColumn()+1,
		)))
	}

	return false
}

// CSI A
func (t *Terminal) csiCursorUpHandler(params []string) (renderRequired bool) {
	distance := 1
	if len(params) > 0 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}
	t.ActiveBuffer().MovePosition(0, -int16(distance))
	return true
}

// CSI B
func (t *Terminal) csiCursorDownHandler(params []string) (renderRequired bool) {
	distance := 1
	if len(params) > 0 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}

	t.ActiveBuffer().MovePosition(0, int16(distance))
	return true
}

// CSI C
func (t *Terminal) csiCursorForwardHandler(params []string) (renderRequired bool) {
	distance := 1
	if len(params) > 0 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}

	t.ActiveBuffer().MovePosition(int16(distance), 0)
	return true
}

// CSI D
func (t *Terminal) csiCursorBackwardHandler(params []string) (renderRequired bool) {
	distance := 1
	if len(params) > 0 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}

	t.ActiveBuffer().MovePosition(-int16(distance), 0)
	return true
}

// CSI E
func (t *Terminal) csiCursorNextLineHandler(params []string) (renderRequired bool) {

	distance := 1
	if len(params) > 0 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}

	t.ActiveBuffer().MovePosition(0, int16(distance))
	t.ActiveBuffer().SetPosition(0, t.ActiveBuffer().CursorLine())
	return true
}

// CSI F
func (t *Terminal) csiCursorPrecedingLineHandler(params []string) (renderRequired bool) {

	distance := 1
	if len(params) > 0 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}
	t.ActiveBuffer().MovePosition(0, -int16(distance))
	t.ActiveBuffer().SetPosition(0, t.ActiveBuffer().CursorLine())
	return true
}

func (t *Terminal) csiCursorCharacterAbsoluteHandler(params []string) (renderRequired bool) {
	distance := 1
	if len(params) > 0 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || params[0] == "" {
			distance = 1
		}
	}

	t.ActiveBuffer().SetPosition(uint16(distance-1), t.ActiveBuffer().CursorLine())
	return nil
}

func parseCursorPosition(params []string) (x, y int) {
	x, y = 1, 1
	if len(params) == 2 {
		var err error
		if params[0] != "" {
			y, err = strconv.Atoi(string(params[0]))
			if err != nil || y < 1 {
				y = 1
			}
		}
		if params[1] != "" {
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
func (t *Terminal) csiCursorPositionHandler(params []string) (renderRequired bool) {
	x, y := parseCursorPosition(params)

	t.ActiveBuffer().SetPosition(uint16(x-1), uint16(y-1))
	return true
}

func (t *Terminal) csiScrollUpHandler(params []string) (renderRequired bool) {
	distance := 1
	if len(params) > 1 {
		return fmt.Errorf("Not supported")
	}
	if len(params) == 1 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}
	terminal.logger.Debugf("Scrolling up %d", distance)
	terminal.AreaScrollUp(uint16(distance))
	return nil
}

func (t *Terminal) csiInsertBlankCharactersHandler(params []string) (renderRequired bool) {
	count := 1
	if len(params) > 1 {
		return fmt.Errorf("Not supported")
	}
	if len(params) == 1 {
		var err error
		count, err = strconv.Atoi(params[0])
		if err != nil || count < 1 {
			count = 1
		}
	}

	t.ActiveBuffer().InsertBlankCharacters(count)

	return nil
}

func (t *Terminal) csiInsertLinesHandler(params []string) (renderRequired bool) {
	count := 1
	if len(params) > 1 {
		return fmt.Errorf("Not supported")
	}
	if len(params) == 1 {
		var err error
		count, err = strconv.Atoi(params[0])
		if err != nil || count < 1 {
			count = 1
		}
	}

	t.ActiveBuffer().InsertLines(count)

	return nil
}

func (t *Terminal) csiDeleteLinesHandler(params []string) (renderRequired bool) {
	count := 1
	if len(params) > 1 {
		return fmt.Errorf("Not supported")
	}
	if len(params) == 1 {
		var err error
		count, err = strconv.Atoi(params[0])
		if err != nil || count < 1 {
			count = 1
		}
	}

	t.ActiveBuffer().DeleteLines(count)

	return nil
}

func (t *Terminal) csiScrollDownHandler(params []string) (renderRequired bool) {
	distance := 1
	if len(params) > 1 {
		return fmt.Errorf("Not supported")
	}
	if len(params) == 1 {
		var err error
		distance, err = strconv.Atoi(params[0])
		if err != nil || distance < 1 {
			distance = 1
		}
	}
	terminal.logger.Debugf("Scrolling down %d", distance)
	terminal.AreaScrollDown(uint16(distance))
	return nil
}

// CSI r
// Set Scrolling Region [top;bottom] (default = full size of window) (DECSTBM), VT100
func (t *Terminal) csiSetMarginsHandler(params []string) (renderRequired bool) {
	top := 1
	bottom := int(t.ActiveBuffer().ViewHeight())

	if len(params) > 2 {
		return fmt.Errorf("Not set margins")
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
			if err != nil || bottom > int(t.ActiveBuffer().ViewHeight()) || bottom < 1 {
				bottom = int(t.ActiveBuffer().ViewHeight())
			}
		}
	}
	top--
	bottom--

	terminal.terminalState.SetVerticalMargins(uint(top), uint(bottom))
	t.ActiveBuffer().SetPosition(0, 0)

	return nil
}

func (t *Terminal) csiEraseCharactersHandler(params []string) (renderRequired bool) {
	count := 1
	if len(params) > 0 {
		var err error
		count, err = strconv.Atoi(params[0])
		if err != nil || count < 1 {
			count = 1
		}
	}

	t.ActiveBuffer().EraseCharacters(count)

	return nil
}

// CSI l
// Reset Mode (RM)
func (t *Terminal) csiResetModeHandler(params []string) (renderRequired bool) {
	return csiSetModes(params, false, terminal)
}

// CSI h
// Set Mode (SM)
func (t *Terminal) csiSetModeHandler(params []string) (renderRequired bool) {
	return csiSetModes(params, true, terminal)
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


func (t *Terminal)csiSetMode(modeStr string, enabled bool) bool {

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
		if enabled { // @todo support replace mode
			t.activeBuffer.SetInsertMode()
		} else {
			t.activeBuffer.SetReplaceMode()
		}
	case "20":
		if enabled {
			t.activeBuffer.SetNewLineMode()
		} else {
			t.activeBuffer.SetLineFeedMode()
		}
	case "?1":
		t.activeBuffer.modes.ApplicationCursorKeys = enabled
	case "?3":
		if enabled {
			// DECCOLM - COLumn mode, 132 characters per line
			t.activeBuffer.ResizeView(132, t.activeBuffer.viewHeight)
		} else {
			// DECCOLM - 80 characters per line (erases screen)
			t.activeBuffer.ResizeView(80, t.activeBuffer.viewHeight)
		}
		t.activeBuffer.Clear()
		/*
			case "?4":
				// DECSCLM
				// @todo smooth scrolling / jump scrolling
		*/
	case "?5": // DECSCNM
		t.activeBuffer.SetScreenMode(enabled)
	case "?6":
		// DECOM
		t.activeBuffer.SetOriginMode(enabled)
	case "?7":
		// auto-wrap mode
		//DECAWM
		t.activeBuffer.SetAutoWrap(enabled)
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
			t.UseAltBuffer()
		} else {
			t.UseMainBuffer()
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
			t.ActiveBuffer().SaveCursor()
		} else {
			t.ActiveBuffer().RestoreCursor()
		}
	case "?1049":
		if enabled {
			t.UseAltBuffer()
		} else {
			t.UseMainBuffer()
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

	t.ActiveBuffer().SetPosition(t.ActiveBuffer().CursorColumn(), uint16(row-1))

	return true
}

func (t *Terminal) csiDeleteHandler(params []string) (renderRequired bool) {
	n := 1
	if len(params) >= 1 {
		var err error
		n, err = strconv.Atoi(params[0])
		if err != nil || n < 1 {
			n = 1
		}
	}

	t.ActiveBuffer().DeleteChars(n)
	return nil
}

// CSI g
// Tab Clear (TBC)
func (t *Terminal) csiTabClearHandler(params []string) (renderRequired bool) {
	n := "0"
	if len(params) > 0 {
		n = params[0]
	}
	switch n {

	case "0", "":
		terminal.terminalState.TabClearAtCursor()
	case "3":
		terminal.terminalState.TabZonk()
	default:
		return fmt.Errorf("Ignored TBC: CSI %s g", n)
	}

	return nil
}

// CSI Ps J
func (t *Terminal) csiEraseInDisplayHandler(params []string) (renderRequired bool) {
	n := "0"
	if len(params) > 0 {
		n = params[0]
	}

	switch n {

	case "0", "":
		t.ActiveBuffer().EraseDisplayFromCursor()
	case "1":
		t.ActiveBuffer().EraseDisplayToCursor()
	case "2", "3":
		t.ActiveBuffer().EraseDisplay()
	default:
		return fmt.Errorf("Unsupported ED: CSI %s J", n)
	}

	return nil
}

// CSI Ps K
func (t *Terminal) csiEraseInLineHandler(params []string) (renderRequired bool) {

	n := "0"
	if len(params) > 0 {
		n = params[0]
	}

	switch n {
	case "0", "": //erase adter cursor
		t.ActiveBuffer().EraseLineFromCursor()
	case "1": // erase to cursor inclusive
		t.ActiveBuffer().EraseLineToCursor()
	case "2": // erase entire
		t.ActiveBuffer().EraseLine()
	default:
		return fmt.Errorf("Unsupported EL: CSI %s K", n)
	}
	return nil
}
