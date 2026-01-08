package main

import (
	"log"
	"os"
)

func main() {
	// Prevent the logging in Go to go to the TUI
	log.SetOutput(os.Stderr)
	startServer()
}
