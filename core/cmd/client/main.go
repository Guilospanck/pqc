package main

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"os"
	"pqc/pkg/ui"
	"pqc/pkg/ws"
)

func main() {
	f, err := os.OpenFile(
		"/tmp/ws-client-pqc.log",
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0644,
	)
	if err != nil {
		log.Fatal(err)
	}

	mw := io.MultiWriter(os.Stderr, f)
	log.SetOutput(mw)

	scanner := bufio.NewScanner(os.Stdin)

	wsClient := WSClient{conn: ws.Connection{}}

	// Start connection manager that will handle things like `reconnect`
	wsClient.connectionManager()

	for scanner.Scan() {
		line := scanner.Bytes()

		var msg ui.UIMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "connect":
			log.Print("Trying to connect")
			wsClient.connectToWSServer()

		case "send":
			log.Print("Sending: ", msg.Value)
			wsClient.sendEncrypted(msg.Value)
		}
	}

}
