package multiplexer

import (
	"io"
	"time"
)

// Read reads data from the multiplexer into stdout for the parent terminal to process/render
func (m *Multiplexer) Read(data []byte) (n int, err error) {

	for i := 0; i < cap(data); i++ {
		select {
		case b := <-m.output:
			data[i] = b
		default:
			if i == 0 {
				select {
				case <-m.closeChan:
					return 0, io.EOF
				default:
					// TODO sort this out
					time.Sleep(time.Millisecond * 10)
				}
			}
			return i, nil
		}
	}
	return cap(data), nil
}
