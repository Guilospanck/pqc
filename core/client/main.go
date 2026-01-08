package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"pqc/pkg/ws"
)

type UIMessage struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func main() {
	// Prevent the logging in Go to go to the TUI
	log.SetOutput(os.Stderr)

	scanner := bufio.NewScanner(os.Stdin)

	wsClient := WSClient{conn: ws.Connection{}}

	for scanner.Scan() {
		line := scanner.Bytes()

		var msg UIMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "connect":
			wsClient.connectToWSServer()

		case "send":
			wsClient.sendEncrypted(msg.Value)
		}
	}

}
