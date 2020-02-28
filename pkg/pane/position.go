package pane

type CoordinateType uint8

const (
	Fixed CoordinateType = iota
	Percentage
)

type Coordinates struct {
	Type CoordinateType
	X    uint16
	Y    uint16
}

type Position struct {
	Origin Coordinates
	Size   Coordinates
}

func NewFullscreenPosition() Position {
	return Position{
		Origin: Coordinates{
			Type: Fixed,
			X:    0,
			Y:    0,
		},
		Size: Coordinates{
			Type: Percentage,
			X:    100,
			Y:    100,
		},
	}
}

func (p Position) ToFixed(rows, cols uint16) Position {
	return Position{
		Origin: p.Origin.ToFixed(rows, cols),
		Size:   p.Size.ToFixed(rows, cols),
	}
}

func (p Coordinates) ToFixed(rows, cols uint16) Coordinates {

	if p.Type == Fixed {
		return p
	}

	return Coordinates{
		Type: Fixed,
		X:    (cols * p.X) / 100,
		Y:    (rows * p.Y) / 100,
	}
}
