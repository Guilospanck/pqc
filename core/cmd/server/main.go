package main

import (
	"log"
	"math/rand"
	"os"
	"time"
)

func main() {
	// Prevent the logging in Go to go to the TUI
	log.SetOutput(os.Stderr)
	// seed it for getting random names
	rand.Seed(time.Now().UnixNano())

	server := Server{connections: nil}
	server.startServer()
}
