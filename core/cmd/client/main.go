package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"

	"github.com/Guilospanck/pqc/core/pkg/logger"
	"github.com/Guilospanck/pqc/core/pkg/types"
)

func main() {
	defer log.Println("> Client is gone!")
	logger.CreateMultiWriterLogger("ws-client-pqc")

	wsClient := NewClient()

	go wsClient.connectionManager()

	readFromStdin(wsClient)
}

// Blocking function that keeps reading from the stdin
func readFromStdin(wsClient *WSClient) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Bytes()

		var msg types.UIMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			log.Println("Error unmarshalling message from stdin: ", err)
			continue
		}

		switch msg.Type {
		case "connect":
			wsClient.connectToWSServer()
			log.Println("Connected to server!")

		case "send":
			wsClient.handleTUIMessage(msg.Value)
		}
	}
}
