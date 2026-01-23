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
			continue
		}

		switch msg.Type {
		case "connect":
			wsClient.connectToWSServer()
			log.Println("Connected to server!")

		// TODO: do not allow this if we're not connected to the server, otherwise
		// we will have problems with sending to a closed channel or something like that.
		// ACTUALLY: we can receive, but not send to.
		// TEST: when you disconnect the server, send some messages in the client TUI.
		// Then, initiate the server back. The TUI will connect. You won't be able to send messages,
		// but you will be able to receive from other clients.
		case "send":
			wsClient.sendEncrypted(msg.Value)
		}
	}
}
