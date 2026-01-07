package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"pqc/pkg/cryptography"
	"pqc/pkg/ws"
	"strings"

	"github.com/gorilla/websocket"
)

func connect() {
	url := "ws://localhost:8080/ws"

	fmt.Println("Connecting to", url)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("Dial error:", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Connected. Type your messages below:")

	connection := ws.Connection{Keys: nil, Conn: conn}

	// Generate keys
	keys, err := cryptography.GenerateKeys()
	if err != nil {
		return
	}
	connection.Keys = keys

	go func() {
		for {
			msg, err := connection.ReadMessage()
			if err != nil {
				log.Printf("Error reading from conn: %s\n", err.Error())
				return
			}

			fmt.Println("Server: ", string(msg))
		}
	}()

	for {
		fmt.Print("> ")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}

		// Quit command
		if text == "/quit" || text == "/exit" {
			fmt.Println("Closing connection.")
			return
		}

		// Send text
		if err := connection.WriteMessage(text); err != nil {
			return
		}
	}
}
