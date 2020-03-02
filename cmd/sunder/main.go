package main

import (
	"time"

	"github.com/liamg/sunder/pkg/multiplexer"
	"github.com/liamg/sunder/pkg/pane"
)

func main() {

	mp := multiplexer.New()

	go func() {
		time.Sleep(time.Second * 3)
		if err := mp.SplitActivePane(pane.Vertical); err != nil {
			panic(err)
		}
	}()

	if err := mp.Start(); err != nil {
		panic(err)
	}
}
