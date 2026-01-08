package server

import (
	"log"
	"os"
)

func Run() {
	// Prevent the logging in Go to go to the TUI
	log.SetOutput(os.Stderr)
	startServer()
}
