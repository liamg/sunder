package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main(){

	// disable input buffering
	fmt.Println("Disabling input buffering...")
	exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()
	// do not display entered characters on the screen
	fmt.Println("Hiding input...")
	exec.Command("stty", "-F", "/dev/tty", "-echo").Run()

	fmt.Println("Ready for input...")

	buffer := make([]byte, 1024)
	for {
		size, err := os.Stdin.Read(buffer)
		for _, byt := range buffer[:size] {
			fmt.Printf("0x%X ", byt)
		}
		if err != nil {
			break
		}
	}
}
