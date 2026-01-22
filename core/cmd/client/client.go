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
	conn       ws.Connection
	reconnect  chan struct{}
	attempts   atomic.Int64
	ctx        context.Context
	cancelFunc context.CancelFunc
}

func NewClient(ctx context.Context, cancel context.CancelFunc) *WSClient {
	return &WSClient{
		conn: ws.Connection{
			WriteMessageReq: make(chan ws.WriteMessageRequest, 10),
			WriteLoopReady:  make(chan struct{}, 1),
		},
		reconnect:  make(chan struct{}, 1),
		ctx:        ctx,
		cancelFunc: cancel,
	}
}

func (client *WSClient) connectionManager() {
	for {
		<-client.reconnect

		// Cancel all goroutines
		client.cancelFunc()

		client.userDisconnected()

		attempts := client.attempts.Load()

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

	client.attempts.Store(1)

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

	return nil
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
	if err := client.conn.WriteMessage(string(jsonMsg), websocket.TextMessage); err != nil {
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
			log.Printf("[%s] ping routine stopped (context cancelled)\n",
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
