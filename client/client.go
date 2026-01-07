package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"pqc/pkg/cryptography"
	"pqc/pkg/ws"
	"strings"
	"time"

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

	connection := ws.Connection{Keys: cryptography.Keys{}, Conn: conn}

	// Generate keys
	keys, err := cryptography.GenerateKeys()
	if err != nil {
		log.Fatal("Error generating keys: ", err)
		return
	}
	connection.Keys = keys

	msg := ws.WSMessage{
		Type:  ws.ExchangeKeys,
		Value: keys.Public,
		Nonce: nil,
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

			msgJson := ws.UnmarshalWSMessage(msg)
			msgJson.HandleServerMessage(&connection)
		}
	}()

	fmt.Println("Exchanging keys...")

	for {
		if connection.Keys.SharedSecret == nil {
			time.Sleep(1 * time.Second)
			continue
		}

		fmt.Println("Connected. Type your messages below:")
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

		// Encrypt message
		nonce, ciphertext, err := cryptography.EncryptMessage(connection.Keys.SharedSecret, []byte(text))
		if err != nil {
			log.Fatal("Could not encrypt message")
		}

		msg := ws.WSMessage{
			Type:  ws.EncryptedMessage,
			Value: ciphertext,
			Nonce: nonce,
		}
		jsonMsg := msg.Marshal()

		// Send encrypted message
		if err := connection.WriteMessage(string(jsonMsg)); err != nil {
			return
		}
	}
}
