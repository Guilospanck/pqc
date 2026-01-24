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

	defer log.Println("> Client is gone!")

	wsClient := NewClient()

	// Start connection manager that will handle things like `reconnect`
	go wsClient.connectionManager()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Bytes()

		var msg ui.UIMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			log.Println("Error unmarshalling message from stdin: ", err)
			continue
		}

		switch msg.Type {
		case "connect":
			wsClient.connectToWSServer()
			log.Println("Connected to server!")

		case "send":
			wsClient.sendEncrypted(msg.Value)
		}
	}
}
