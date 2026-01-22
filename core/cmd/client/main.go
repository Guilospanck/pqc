package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"pqc/pkg/logger"
	"pqc/pkg/ui"
)

func main() {
	logger.CreateMultiWriterLogger("ws-client-pqc")

	wsClient := NewClient()

	// Start connection manager that will handle things like `reconnect`
	go wsClient.connectionManager()

	scanner := bufio.NewScanner(os.Stdin)
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
