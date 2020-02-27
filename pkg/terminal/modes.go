package terminal

type Modes struct {
	ShowCursor            bool
	ApplicationCursorKeys bool
	BlinkingCursor        bool
}

type MouseMode uint
type MouseExtMode uint

const (
	MouseModeNone MouseMode = iota
	MouseModeX10
	MouseModeVT200
	MouseModeVT200Highlight
	MouseModeButtonEvent
	MouseModeAnyEvent
	MouseExtNone MouseExtMode = iota
	MouseExtUTF
	MouseExtSGR
	MouseExtURXVT
)
