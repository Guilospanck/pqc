package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"pqc/pkg/logger"
	"pqc/pkg/ui"
	"pqc/pkg/ws"
)

func main() {
	logger.CreateMultiWriterLogger("ws-client-pqc")

	scanner := bufio.NewScanner(os.Stdin)

	wsClient := WSClient{conn: ws.Connection{WriteMessageReq: make(chan ws.WriteMessageRequest, 10)}, reconnect: make(chan struct{}, 1)}

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
