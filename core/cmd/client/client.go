package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"pqc/pkg/cryptography"
	"pqc/pkg/ui"
	"pqc/pkg/ws"
	"slices"
	"strings"
	"sync/atomic"
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
	conn            ws.Connection
	reconnect       chan struct{}
	attempts        atomic.Int64
	ctx             context.Context
	cancelFunc      context.CancelFunc
	isConnected     bool
	deadLetterQueue chan string // we save non-delivered non-encrypted messages here
}

func NewClient() *WSClient {
	return &WSClient{
		conn:            ws.NewEmptyConnection(),
		reconnect:       make(chan struct{}, 1),
		isConnected:     false,
		deadLetterQueue: make(chan string, 10),
	}
}

func (client *WSClient) connectionManager() {
	for {
		<-client.reconnect
		client.isConnected = false

		// Cancel all goroutines
		log.Printf("[%s] Cancelling context\n", client.conn.Metadata.Username)
		client.cancelFunc()

		attempts := client.attempts.Load()

		if attempts == int64(1) {
			client.userDisconnected()
		}

		if attempts >= int64(MAX_ATTEMPTS) {
			log.Println("We burned through all attempts.")
			client.closeAndDisconnect()
			return
		}

		// exponential backoff
		wait := time.Duration(1<<attempts) * time.Second
		time.Sleep(wait)

		log.Printf("Attempt #%d/5 to reconnect to server\n", attempts)
		client.attempts.Add(1)

		// We try to connect to the WS server again. If it doesn't work,
		// we trigger another reconnect
		if err := client.connectToWSServer(); err != nil {
			client.triggerReconnect()
		}
	}
}

func (client *WSClient) connectToWSServer() error {
	url := "ws://localhost:8080/ws"
	log.Printf("Connecting to %s\n", url)

	// Reset connection channels because
	// we might have closed them on reconnection.
	client.conn.ResetChannels()

	requestHeader := http.Header{}
	if client.conn.Metadata.Color != "" || client.conn.Metadata.Username != "" {
		requestHeader.Set("username", client.conn.Metadata.Username)
		requestHeader.Set("color", client.conn.Metadata.Color)
	}

	conn, res, err := websocket.DefaultDialer.Dial(url, requestHeader)
	if err != nil {
		log.Printf("Dial error: %s\n", err.Error())
		return err
	}
	client.conn.Conn = conn
	client.isConnected = true
	log.Println("Dialing to WS server completed successfully!")

	client.attempts.Store(1)

	// Start a new context
	ctx, cancel := context.WithCancel(context.Background())
	client.ctx = ctx
	client.cancelFunc = cancel

	// Start write loop
	go client.conn.WriteLoop(client.ctx)

	<-client.conn.WriteLoopReady

	// Start ping routine
	go client.pingRoutine()

	// Start the read loop
	go client.readAndHandleServerMessages()

	username := res.Header.Get("username")
	color := res.Header.Get("color")
	client.conn.Metadata = ws.WSMetadata{Username: username, Color: color}
	// Tell UI we're connected with some username and color
	ui.EmitToUI(ui.ToUIConnected, username, color)

	if client.conn.Keys.Public == nil {
		if err := client.generateKeys(); err != nil {
			// If error while generating keys, we don't try to reconnect to the server,
			// hence why returning nil
			return nil
		}
	}

	if err := client.exchangeKeys(); err != nil {
		// If error while exchanging keys, we don't try to reconnect to the server,
		// hence why returning nil
		return nil
	}

	// Wait for the keys to be exchanged before proceeding.
	<-client.conn.KeysExchanged

	client.drainDLQ()

	return nil
}

// If there are any messages in the client's dead-letter queue (DLQ),
// we send them to the server.
func (client *WSClient) drainDLQ() {
	initial := len(client.deadLetterQueue)
	for range initial {
		msg := <-client.deadLetterQueue
		log.Printf("[%s] Sending message from DLQ: %s", client.conn.Metadata.Username, msg)
		client.sendEncrypted(msg)
	}
}

func (client *WSClient) generateKeys() error {
	keys, err := cryptography.GenerateKeys()
	if err != nil {
		log.Printf("[%s] Error generating keys: %s\n", client.conn.Metadata.Username, err.Error())
		return err
	}
	client.conn.Keys = keys

	return nil
}

func (client *WSClient) exchangeKeys() error {
	msg := ws.WSMessage{
		Type:     ws.ExchangeKeys,
		Value:    client.conn.Keys.Public,
		Nonce:    nil,
		Metadata: client.conn.Metadata,
	}
	jsonMsg := msg.Marshal()

	// Send public key so we can exchange keys
	if err := client.conn.WriteMessage(string(jsonMsg), websocket.TextMessage); err != nil {
		log.Printf("[%s] Error trying to send public key to server: %s\n", client.conn.Metadata.Username, err.Error())
		return err
	}

	return nil
}

func (client *WSClient) readAndHandleServerMessages() {
	log.Println("Starting READ loop...")

	for {
		client.conn.Conn.SetReadDeadline(time.Now().Add(PONG_WAIT))

		msg, err := client.conn.ReadMessage()
		if err != nil {
			log.Printf("[%s] Error reading from conn: %s\n", client.conn.Metadata.Username, err.Error())
			client.triggerReconnect()
			return
		}

		msgJson, err := ws.UnmarshalWSMessage(msg)
		if err != nil {
			log.Printf("[%s] Error unmarshalling message: %s\n", client.conn.Metadata.Username, err.Error())
			continue
		}
		client.conn.HandleServerMessage(msgJson)

		select {
		case <-client.ctx.Done():
			log.Printf("[%s] Context cancelled. Returning from READ loop.\n", client.conn.Metadata.Username)
			return
		default:
		}
	}
}

func (client *WSClient) triggerReconnect() {
	// If reconnect was already triggered, it won't trigger again
	select {
	case client.reconnect <- struct{}{}:
		log.Printf("[%s] Triggering reconnect...", client.conn.Metadata.Username)
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
		log.Printf("[%s] Quit command received.\n", client.conn.Metadata.Username)
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

	if !client.isConnected {
		client.deadLetterQueue <- message
		return
	}

	// Send encrypted message
	if err := client.conn.WriteMessage(string(jsonMsg), websocket.TextMessage); err != nil {
		log.Printf("Error writing message to server: %s\n", err.Error())
		client.deadLetterQueue <- message
		client.triggerReconnect()
	}
}

func (client *WSClient) closeAndDisconnect() {
	log.Printf("[%s] Closing connection.", client.conn.Metadata.Username)

	client.cancelFunc()
	if client.conn.Conn != nil {
		client.conn.Conn.Close()
	}

	os.Exit(0)
}

func (client *WSClient) pingRoutine() {
	log.Println("Starting PING routine...")
	// set pong handler (server will respond to our ping with a pong)
	// gorilla ws server automatically responds to pings
	client.conn.Conn.SetReadDeadline(time.Now().Add(PONG_WAIT))
	client.conn.Conn.SetPongHandler(func(string) error {
		client.conn.Conn.SetReadDeadline(time.Now().Add(PONG_WAIT))
		return nil
	})

	// set ping routine
	ticker := time.NewTicker(PING_PERIOD)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Printf("[%s] pinging server...\n", client.conn.Metadata.Username)

			client.conn.Conn.SetWriteDeadline(time.Now().Add(WRITE_WAIT))
			if err := client.conn.WriteMessage("", websocket.PingMessage); err != nil {
				log.Printf("[%s] ping error: %s\n", client.conn.Metadata.Username, err)
				client.triggerReconnect()
				return
			}

		case <-client.ctx.Done():
			log.Printf("[%s] PING routine stopped (context cancelled)\n",
				client.conn.Metadata.Username)
			return
		}
	}
}

// Inform TUI that user is disconnected
func (client *WSClient) userDisconnected() {
	connection := client.conn

	metadata := connection.Metadata
	ui.EmitToUI(ui.ToUIDisconnected, string(metadata.Username), metadata.Color)
}
