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
		log.Fatal("Error generating keys: ", err)
		return
	}
	connection.Keys = keys

	msg := ws.ClientToServerMessage{
		Type:  ws.ExchangeKeys,
		Value: keys.Public,
	}
	jsonMsg := msg.Marshal()

	// Send public key so we can exchange keys
	if err := connection.WriteMessage(string(jsonMsg)); err != nil {
		log.Fatal("Error trying to send public key to server: ", err)
		return
	}

	// goroutine to read the messages from server
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

		msg := ws.ClientToServerMessage{
			// TODO: encrypt message
			Type:  ws.EncryptedMessage,
			Value: []byte(text),
		}
		jsonMsg := msg.Marshal()

		// Send text
		if err := connection.WriteMessage(string(jsonMsg)); err != nil {
			return
		}
	}
}
