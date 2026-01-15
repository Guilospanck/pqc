package main

import (
	"log"
	"os"
	"pqc/pkg/cryptography"
	"pqc/pkg/ui"
	"pqc/pkg/ws"
	"slices"
	"strings"

	"github.com/gorilla/websocket"
)

var QUIT_COMMANDS = []string{"/quit", "/q", "/exit", ":wq", ":q", ":wqa"}

type WSClient struct {
	conn ws.Connection
}

func (client *WSClient) connectToWSServer() {
	url := "ws://localhost:8080/ws"

	log.Print("Connecting to", url)
	conn, res, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Printf("Dial error: %s\n", err.Error())
		return
	}

	username := res.Header.Get("username")
	color := res.Header.Get("color")

	// Tell UI we're connected with some username and color
	ui.EmitToUI(ui.ToUIConnected, username, color)

	client.conn = ws.Connection{Keys: cryptography.Keys{}, Conn: conn, Metadata: ws.WSMetadata{Username: username, Color: color}}

	// Generate keys
	keys, err := cryptography.GenerateKeys()
	if err != nil {
		log.Printf("Error generating keys: %s\n", err.Error())
		return
	}
	client.conn.Keys = keys

	msg := ws.WSMessage{
		Type:     ws.ExchangeKeys,
		Value:    keys.Public,
		Nonce:    nil,
		Metadata: ws.WSMetadata{Username: username, Color: color},
	}
	jsonMsg := msg.Marshal()

	// Send public key so we can exchange keys
	if err := client.conn.WriteMessage(string(jsonMsg)); err != nil {
		log.Printf("Error trying to send public key to server: %s\n", err.Error())
		return
	}

	// goroutine to read the messages from server
	go func() {
		for {
			msg, err := client.conn.ReadMessage()
			if err != nil {
				log.Printf("Error reading from conn: %s\n", err.Error())
				// connection to server is dead...
				return
			}

			msgJson, err := ws.UnmarshalWSMessage(msg)
			if err != nil {
				log.Printf("Error unmarshalling message: %s\n", err.Error())
				continue
			}
			client.conn.HandleServerMessage(msgJson)
		}
	}()
}

func (client *WSClient) closeConnection() {
	if client.conn.Conn != nil {
		client.conn.Conn.Close()
	}
}

func (client *WSClient) sendEncrypted(message string) {
	if client.conn.Keys.SharedSecret == nil {
		log.Print("Shared secret not ready")
		return
	}

	text := strings.TrimSpace(message)
	if text == "" {
		log.Print("Empty message.")
		return
	}

	// Quit command
	if slices.Contains(QUIT_COMMANDS, text) {
		log.Print("Closing connection.")
		client.closeConnection()
		os.Exit(0)
		return
	}

	// Encrypt message
	nonce, ciphertext, err := cryptography.EncryptMessage(client.conn.Keys.SharedSecret, []byte(text))
	if err != nil {
		log.Printf("Could not encrypt message: %s\n", err.Error())
		return
	}

	msg := ws.WSMessage{
		Type:     ws.EncryptedMessage,
		Value:    ciphertext,
		Nonce:    nonce,
		Metadata: ws.WSMetadata{Username: client.conn.Metadata.Username, Color: client.conn.Metadata.Color},
	}
	jsonMsg := msg.Marshal()

	// Send encrypted message
	if err := client.conn.WriteMessage(string(jsonMsg)); err != nil {
		log.Printf("Error writing message to server: %s\n", err.Error())
	}
}
