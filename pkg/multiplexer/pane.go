package multiplexer

import (
	"github.com/google/uuid"
	"github.com/liamg/sunder/pkg/terminal"
)

type Pane struct {
	id         uuid.UUID
	terminal   *terminal.Terminal
	childPanes []*Pane
	pos        PanePosition
	renderChan chan struct{}
	updateChan chan<- *Pane
	exists     bool
	closeChan  chan struct{}
}

func NewPane(pos PanePosition, updateChan chan<- *Pane, term *terminal.Terminal) *Pane {
	return &Pane{
		id:         uuid.New(),
		pos:        pos,
		terminal:   term,
		updateChan: updateChan,
		closeChan:  make(chan struct{}),
	}
}

func (p *Pane) Start() error {

	updateChan := make(chan struct{})

	go func() {
		for {
			select {
			case <-updateChan:
				p.requestRender()
			case <-p.closeChan:
				return

			}
		}
	}()

	if err := p.terminal.Run(updateChan, p.pos.Size.Y, p.pos.Size.X); err != nil {
		return err
	}
	p.close()
	return nil
}

func (p *Pane) Exists() bool {
	return p.exists
}

func (p *Pane) close() {
	p.exists = false
}

func (p *Pane) requestRender() {
	select {
	case p.updateChan <- p:
	default:
		// TODO handle this case when buffer is full and channel blocks?
	}
}

func (p *Pane) MoveTo(x uint16, y uint16) {
	p.pos.Origin.X = x
	p.pos.Origin.Y = y
}

func (p *Pane) Resize(parentRows uint16, parentCols uint16) error {

	if p.pos.Size.Type == Fixed {
		if p.pos.Size.Y > parentRows {
			p.pos.Size.Y = parentRows
		}
		if p.pos.Size.X > parentCols {
			p.pos.Size.X = parentCols
		}
	}

	fixed := p.pos.Size.ToFixed(parentRows, parentCols)

	if err := p.terminal.SetSize(fixed.Y, fixed.X); err != nil {
		return err
	}

	return nil
}

func (p *Pane) Add(n *Pane) {
	p.childPanes = append(p.childPanes, n)
}

func (p *Pane) ProcessStdIn(data []byte) error {
	_, err := p.terminal.Pty().Write(data)
	return err
}

// TODO splitH/V methods
