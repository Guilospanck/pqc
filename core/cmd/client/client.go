package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Guilospanck/pqc/core/pkg/cryptography"
	"github.com/Guilospanck/pqc/core/pkg/types"
	"github.com/Guilospanck/pqc/core/pkg/ui"
	"github.com/Guilospanck/pqc/core/pkg/ws"

	"github.com/gorilla/websocket"
)

type WSClient struct {
	conn            *ws.Connection
	reconnect       chan struct{}
	attempts        atomic.Int64
	ctx             context.Context
	cancelFunc      context.CancelFunc
	isConnected     bool
	deadLetterQueue chan string // we save non-delivered non-encrypted messages here
	currentRoomID   types.RoomId
}

func NewClient() *WSClient {
	connection := ws.NewEmptyConnection()

	return &WSClient{
		conn:            &connection,
		reconnect:       make(chan struct{}, 1),
		isConnected:     false,
		deadLetterQueue: make(chan string, 10),
		currentRoomID:   types.RoomId(""), // tbd on connection
	}
}

// Responsible for handling reconnections.
//
// Maximum attemps is defined by the MAX_ATTEMPS constant variable.
func (client *WSClient) connectionManager() {
	for {
		<-client.reconnect
		client.isConnected = false

		// Cancel all goroutines that depend on the context
		log.Printf("[%s] Cancelling context\n", client.conn.Metadata.Username)
		client.cancelFunc()

		attempts := client.attempts.Load()

		if attempts == int64(1) {
			ui.EmitToUI(types.MessageTypeDisconnected, string(client.conn.Metadata.Username), *client.conn.Metadata)
		} else {
			ui.EmitToUI(types.MessageTypeReconnecting, string(client.conn.Metadata.Username), *client.conn.Metadata)
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
	if client.conn.Metadata.Color != "" {
		requestHeader.Set("color", client.conn.Metadata.Color)
	}
	if client.conn.Metadata.Username != "" {
		requestHeader.Set("username", client.conn.Metadata.Username)
	}
	if client.currentRoomID != types.RoomId("") {
		requestHeader.Set("roomId", string(client.currentRoomID))
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
	client.conn.Metadata = &types.WSMetadata{Username: username, Color: color, UserId: client.conn.ID, CurrentRoomId: client.currentRoomID}
	// Tell UI we're connected with some username and color
	ui.EmitToUI(types.MessageTypeConnected, username, *client.conn.Metadata)

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
		client.handleTUIMessage(msg)
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
		Type:     types.MessageTypeExchangeKeys,
		Value:    client.conn.Keys.Public,
		Nonce:    nil,
		Metadata: *client.conn.Metadata,
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

		client.handleServerMessage(msgJson, client.conn)

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

func (client *WSClient) handleTUIMessage(message string) {
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

	// Check if it's another command other than QUIT
	if strings.HasPrefix(text, "/") {
		client.handleCommand(text)
		return
	}

	// Then it's just a normal message to be sent encrypted
	client.sendMessageToServer(text, types.MessageTypeEncryptedMessage)
}

func (client *WSClient) closeAndDisconnect() {
	log.Printf("[%s] Closing connection.", client.conn.Metadata.Username)

	client.cancelFunc()
	if client.conn.Conn != nil {
		client.conn.Conn.Close()
	}

	os.Exit(0)
}

func (client *WSClient) handleCommand(input string) {
	fields := strings.Fields(input)
	command := fields[0]
	args := fields[1:]

	validateRoomsArgs := func() bool {
		if len(args) != 1 {
			client.sendSystemMessage(fmt.Sprintf("Error.\nUsage: %s <room-name>\n", command))
			return false
		}

		return true
	}

	switch command {
	case "/join":
		if validateRoomsArgs() {
			client.sendMessageToServer(args[0], types.MessageTypeJoinRoom)
		}
	case "/leave":
		if validateRoomsArgs() {
			client.sendMessageToServer(args[0], types.MessageTypeLeaveRoom)
		}
	case "/create":
		if validateRoomsArgs() {
			client.sendMessageToServer(args[0], types.MessageTypeCreateRoom)
		}
	case "/delete":
		if validateRoomsArgs() {
			client.sendMessageToServer(args[0], types.MessageTypeDeleteRoom)
		}
	default:
		client.sendSystemMessage(fmt.Sprintf("Command \"%s\" not recognised.\nAvailable commands: /join, /leave, /create-room, /delete-room", command))
	}
}

func (client *WSClient) sendSystemMessage(message string) {
	metadata := client.conn.Metadata
	metadata.Color = "#F00"

	ui.EmitToUI(types.MessageTypeMessage, message, *metadata)
}

func (client *WSClient) sendMessageToServer(tuiMessage string, msgType types.MessageType) {
	wsMessage := ws.WSMessage{
		Type: msgType,
		Metadata: types.WSMetadata{
			Username:      client.conn.Metadata.Username,
			Color:         client.conn.Metadata.Color,
			CurrentRoomId: client.conn.Metadata.CurrentRoomId,
			UserId:        client.conn.ID,
		},
	}

	switch msgType {
	case types.MessageTypeEncryptedMessage:
		nonce, ciphertext, err := cryptography.EncryptMessage(client.conn.Keys.SharedSecret, []byte(tuiMessage))
		if err != nil {
			log.Printf("Could not encrypt message: %s\n", err.Error())
			return
		}

		wsMessage.Value = ciphertext
		wsMessage.Nonce = nonce

	case types.MessageTypeJoinRoom, types.MessageTypeCreateRoom, types.MessageTypeLeaveRoom, types.MessageTypeDeleteRoom:
		wsMessage.Value = []byte(tuiMessage)

	default:
		log.Printf("This message type (%s) is not recognisable in the client from TUI.\n", msgType)
	}

	jsonMsg := wsMessage.Marshal()

	if !client.isConnected {
		client.deadLetterQueue <- tuiMessage
		return
	}

	if err := client.conn.WriteMessage(string(jsonMsg), websocket.TextMessage); err != nil {
		log.Printf("Error writing message to server: %s\n", err.Error())
		client.deadLetterQueue <- tuiMessage
		client.triggerReconnect()
	}
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

func (client *WSClient) handleServerMessage(msg ws.WSMessage, connection *ws.Connection) {
	switch msg.Type {
	case types.MessageTypeExchangeKeys:
		ciphertext := msg.Value
		sharedSecret, err := connection.Keys.Private.Decapsulate(ciphertext)
		if err != nil {
			log.Printf("Could not get shared secret from ciphertext: %s\n", err.Error())
			return
		}

		// Now the client also have the shared secret
		connection.Keys.SharedSecret = cryptography.DeriveKey(sharedSecret)
		ui.EmitToUI(types.MessageTypeKeysExchanged, connection.Metadata.Username, *connection.Metadata)

		close(connection.KeysExchanged)

	case types.MessageTypeEncryptedMessage:
		nonce := msg.Nonce
		ciphertext := msg.Value

		log.Printf("Received encrypted message: >>> %s <<<, with nonce: >>> %s <<<\n", ciphertext, nonce)
		decrypted, err := cryptography.DecryptMessage(connection.Keys.SharedSecret, nonce, ciphertext)
		if err != nil {
			log.Printf("Could not decrypt message from server: %s\n", err.Error())
			return
		}

		ui.EmitToUI(types.MessageTypeMessage, string(decrypted), msg.Metadata)
	case types.MessageTypeUserEnteredChat:
		metadata := msg.Metadata
		ui.EmitToUI(types.MessageTypeUserEnteredChat, string(metadata.Username), metadata)
	case types.MessageTypeUserLeftChat:
		metadata := msg.Metadata
		ui.EmitToUI(types.MessageTypeUserLeftChat, string(metadata.Username), metadata)
	case types.MessageTypeCurrentUsers:
		metadata := msg.Metadata
		value := msg.Value
		ui.EmitToUI(types.MessageTypeCurrentUsers, string(value), metadata)
	case types.MessageTypeSuccess:
		value := msg.Value
		metadata := msg.Metadata
		metadata.Color = "#0F0"
		ui.EmitToUI(types.MessageTypeSuccess, string(value), metadata)
	case types.MessageTypeError:
		value := msg.Value
		metadata := msg.Metadata
		metadata.Color = "#F00"
		ui.EmitToUI(types.MessageTypeError, string(value), metadata)
	case types.MessageTypeJoinedRoom:
		value := msg.Value
		metadata := msg.Metadata
		ui.EmitToUI(types.MessageTypeJoinedRoom, string(value), metadata)
	case types.MessageTypeLeftRoom:
		value := msg.Value
		metadata := msg.Metadata
		ui.EmitToUI(types.MessageTypeLeftRoom, string(value), metadata)
	case types.MessageTypeCreatedRoom:
		value := msg.Value
		metadata := msg.Metadata
		ui.EmitToUI(types.MessageTypeCreatedRoom, string(value), metadata)
	case types.MessageTypeDeletedRoom:
		value := msg.Value
		metadata := msg.Metadata
		ui.EmitToUI(types.MessageTypeDeletedRoom, string(value), metadata)
	case types.MessageTypeAvailableRooms:
		value := msg.Value
		metadata := msg.Metadata
		ui.EmitToUI(types.MessageTypeAvailableRooms, string(value), metadata)
	default:
		log.Printf("Received a message with an unknown type: %s\n", msg.Type)
	}

}
