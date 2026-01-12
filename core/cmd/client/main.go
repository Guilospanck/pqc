package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"pqc/pkg/ui"
	"pqc/pkg/ws"
)

func main() {
	// Prevent the logging in Go to go to the TUI
	log.SetOutput(os.Stderr)

	scanner := bufio.NewScanner(os.Stdin)

	wsClient := WSClient{conn: ws.Connection{}, tagColor: GetRandomColor()}

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
