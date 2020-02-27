package multiplexer

type CoordinateType uint8

const (
	Fixed CoordinateType = iota
	Percentage
)

type PaneCoordinates struct {
	Type CoordinateType
	X    uint16
	Y    uint16
}

type PanePosition struct {
	Origin PaneCoordinates
	Size   PaneCoordinates
}

func NewFullscreenPosition() PanePosition {
	return PanePosition{
		Origin: PaneCoordinates{
			Type: Fixed,
			X:    0,
			Y:    0,
		},
		Size: PaneCoordinates{
			Type: Percentage,
			X:    100,
			Y:    100,
		},
	}
}

func (p PanePosition) ToFixed(rows, cols uint16) PanePosition {
	return PanePosition{
		Origin: p.Origin.ToFixed(rows, cols),
		Size:   p.Size.ToFixed(rows, cols),
	}
}

func (p PaneCoordinates) ToFixed(rows, cols uint16) PaneCoordinates {

	if p.Type == Fixed {
		return p
	}

	return PaneCoordinates{
		Type: Fixed,
		X:    (cols * p.X) / 100,
		Y:    (rows * p.Y) / 100,
	}
}
