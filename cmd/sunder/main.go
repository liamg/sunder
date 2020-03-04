package main

import (
	"fmt"

	"github.com/liamg/sunder/pkg/multiplexer"
)

func main() {

	mp := multiplexer.New()
	if err := mp.Start(); err != nil {
		panic(err)
	}

	// reset terminal on exit
	fmt.Printf("\x1bc")
}
