package main

import (
	"fmt"
	"log"
	"pqc/pkg/cryptography"
	"pqc/pkg/ws"
	"strings"

	"github.com/gorilla/websocket"
)

type WSClient struct {
	conn ws.Connection
}

func (client *WSClient) connectToWSServer() {
	url := "ws://localhost:8080/ws"

	fmt.Println("Connecting to", url)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("Dial error:", err)
	}
	defer conn.Close()

	client.conn = ws.Connection{Keys: cryptography.Keys{}, Conn: conn}

	// Generate keys
	keys, err := cryptography.GenerateKeys()
	if err != nil {
		log.Fatal("Error generating keys: ", err)
		return
	}
	client.conn.Keys = keys

	msg := ws.WSMessage{
		Type:  ws.ExchangeKeys,
		Value: keys.Public,
		Nonce: nil,
	}
	jsonMsg := msg.Marshal()

	// Send public key so we can exchange keys
	if err := client.conn.WriteMessage(string(jsonMsg)); err != nil {
		log.Fatal("Error trying to send public key to server: ", err)
		return
	}

	// goroutine to read the messages from server
	go func() {
		for {
			msg, err := client.conn.ReadMessage()
			if err != nil {
				log.Printf("Error reading from conn: %s\n", err.Error())
				return
			}

			msgJson := ws.UnmarshalWSMessage(msg)
			msgJson.HandleServerMessage(&client.conn)
		}
	}()

	fmt.Println("Exchanging keys...")

}

func (client *WSClient) sendEncrypted(message string) {
	if client.conn.Keys.SharedSecret == nil {
		fmt.Println("Shared secret not ready")
		return
	}

	text := strings.TrimSpace(message)
	if text == "" {
		fmt.Println("Empty message.")
		return
	}

	// Quit command
	if text == "/quit" || text == "/exit" {
		fmt.Println("Closing connection.")
		return
	}

	// Encrypt message
	nonce, ciphertext, err := cryptography.EncryptMessage(client.conn.Keys.SharedSecret, []byte(text))
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
	if err := client.conn.WriteMessage(string(jsonMsg)); err != nil {
		log.Fatal("Error writing message to server")
	}
}
