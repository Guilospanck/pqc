package main

import (
	"io"
	"log"
	"os"
)

func main() {
	f, err := os.OpenFile(
		"/tmp/ws-server-pqc.log",
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0644,
	)
	if err != nil {
		log.Fatal(err)
	}

	mw := io.MultiWriter(os.Stderr, f)
	log.SetOutput(mw)

	server := WSServer{connections: nil}
	server.startServer()
}
