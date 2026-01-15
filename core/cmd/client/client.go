package main

import (
	"log"
	"os"
	"pqc/pkg/cryptography"
	"pqc/pkg/ui"
	"pqc/pkg/ws"
	"slices"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const PONG_WAIT = 10 * time.Second
const WRITE_WAIT = 5 * time.Second

// INFO: ping period needs to be less than pong wait, otherwise it will
// timeout the pong before we can ping
const PING_PERIOD = 5 * time.Second

var QUIT_COMMANDS = []string{"/quit", "/q", "/exit", ":wq", ":q", ":wqa"}

// How many reconnect attemps we are able to do
const MAX_ATTEMPTS int = 5

type WSClient struct {
	conn      ws.Connection
	reconnect chan struct{}
}

func (client *WSClient) connectionManager() {
	client.reconnect = make(chan struct{}, 1)

	go func() {
		attempts := 0
		for {
			<-client.reconnect

			if attempts >= MAX_ATTEMPTS {
				log.Println("We burned through all attempts.")
				client.closeAndDisconnect()
				return
			}

			// exponential backoff
			wait := time.Duration(1<<attempts) * time.Second
			time.Sleep(wait)

			log.Printf("Attempt #%d/5 to reconnect to server\n", attempts)
			attempts++

			// We try to connect to the WS server again. If it doesn't work,
			// we trigger another reconnect
			if err := client.connectToWSServer(); err != nil {
				client.triggerReconnect()
			}
		}
	}()
}

func (client *WSClient) connectToWSServer() error {
	url := "ws://localhost:8080/ws"

	log.Print("Connecting to ", url)
	conn, res, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Printf("Dial error: %s\n", err.Error())
		return err
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
		// If error while generating keys, we don't try to reconnect to the server,
		// hence why returning nil
		return nil
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
		return nil
	}

	// Start ping routine
	go client.pingRoutine()

	// Read the messages from server
	go client.readAndHandleServerMessages()

	return nil
}

func (client *WSClient) readAndHandleServerMessages() {
	for {
		msg, err := client.conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading from conn: %s\n", err.Error())
			client.triggerReconnect()
			return
		}

		msgJson, err := ws.UnmarshalWSMessage(msg)
		if err != nil {
			log.Printf("Error unmarshalling message: %s\n", err.Error())
			continue
		}
		client.conn.HandleServerMessage(msgJson)
	}

}

// TODO: see if we can reconnect with same credentials...
func (client *WSClient) triggerReconnect() {
	// If reconnect was already triggered, it won't trigger again
	select {
	case client.reconnect <- struct{}{}:
		log.Println("Triggering reconnect...")
	default:
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
		client.closeAndDisconnect()
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
		// TODO: save the message somewhere and then retry it after connection
		client.triggerReconnect()
	}
}

func (client *WSClient) closeAndDisconnect() {
	log.Print("Closing connection.")
	if client.conn.Conn != nil {
		client.conn.Conn.Close()
	}

	os.Exit(0)
}

func (client *WSClient) pingRoutine() {
	// set pong handler (server will respond to our ping with a pong)
	// gorilla ws automatically responds to pings
	client.conn.Conn.SetReadDeadline(time.Now().Add(PONG_WAIT))
	client.conn.Conn.SetPongHandler(func(string) error {
		client.conn.Conn.SetReadDeadline(time.Now().Add(PONG_WAIT))
		return nil
	})

	// set ping routine
	ticker := time.NewTicker(PING_PERIOD)
	defer ticker.Stop()

	for {
		<-ticker.C

		log.Printf("Client %s is pinging server...\n", client.conn.Metadata.Username)

		client.conn.Conn.SetWriteDeadline(time.Now().Add(WRITE_WAIT))
		if err := client.conn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			log.Println("ping error:", err)
			client.triggerReconnect()
			return
		}
	}
}
