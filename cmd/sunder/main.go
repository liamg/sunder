package main

import (
	"github.com/liamg/sunder/pkg/multiplexer"
)

func main() {
	mp := multiplexer.New()
	if err := mp.Start(); err != nil {
		panic(err)
	}
}
