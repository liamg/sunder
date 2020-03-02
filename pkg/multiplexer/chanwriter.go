package multiplexer

type ChanWriter struct {
	output chan byte
}

func NewChanWriter(output chan byte) *ChanWriter {
	return &ChanWriter{output: output}
}

func (w *ChanWriter) Write(data []byte) (n int, err error) {
	for _, b := range data {
		w.output <- b
	}
	return len(data), nil
}
