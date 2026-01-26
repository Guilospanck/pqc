package ws

import (
	"context"
	"errors"
	"log"

	"github.com/Guilospanck/pqc/core/pkg/cryptography"
	"github.com/Guilospanck/pqc/core/pkg/types"
	"github.com/Guilospanck/pqc/core/pkg/ui"

	"github.com/gorilla/websocket"
)

type WriteMessageRequest struct {
	msgType int // websocket.TextMessage, websocket.PingMessage
	text    []byte
	err     chan error
}

type Connection struct {
	Keys     cryptography.Keys
	Conn     *websocket.Conn
	Metadata WSMetadata

	WriteMessageReq chan WriteMessageRequest

	WriteLoopReady  chan struct{}
	WriteLoopClosed chan struct{}

	KeysExchanged chan struct{}

	CurrentRoomID types.RoomId
}

func NewEmptyConnection() Connection {
	return Connection{
		Keys:     cryptography.Keys{},
		Conn:     nil,
		Metadata: WSMetadata{},

		WriteMessageReq: make(chan WriteMessageRequest, 10),
		WriteLoopReady:  make(chan struct{}),
		WriteLoopClosed: make(chan struct{}),

		KeysExchanged: make(chan struct{}),
		CurrentRoomID: "", // will be set when joining lobby
	}
}

func (ws *Connection) ResetChannels() {
	ws.WriteMessageReq = make(chan WriteMessageRequest, 10)
	ws.WriteLoopReady = make(chan struct{})
	ws.WriteLoopClosed = make(chan struct{})
	ws.KeysExchanged = make(chan struct{})
}

func (ws *Connection) WriteLoop(ctx context.Context) {
	log.Println("Starting WRITE loop...")

	close(ws.WriteLoopReady)
	defer close(ws.WriteLoopClosed)

	for {
		select {
		case msg := <-ws.WriteMessageReq:

			text := msg.text
			msgType := msg.msgType

			err := ws.Conn.WriteMessage(msgType, text)

			select {
			case msg.err <- err:
			case <-ctx.Done():
				log.Println("Context cancelled while selecting the write message result. Returning from WRITE loop.")
				return
			}

			if err != nil {
				log.Println("Error while writing message. Returning from WRITE loop")
				return
			}

		case <-ctx.Done():
			log.Println("Context cancelled. Returning from WRITE loop.")
			return
		}
	}
}

// Understanding the channels/select here:
// 1. Can I hand the letter to the courier?
// 2. Will the courier ever reply?
func (ws *Connection) WriteMessage(text string, msgType int) error {
	errCh := make(chan error, 1)

	req := WriteMessageRequest{
		msgType: msgType,
		text:    []byte(text),
		err:     errCh,
	}

	select {
	case ws.WriteMessageReq <- req:
	case <-ws.WriteLoopClosed:
		return errors.New("connection closed")
	}

	select {
	case err := <-errCh:
		return err
	case <-ws.WriteLoopClosed:
		return errors.New("connection closed")
	}
}

func (ws *Connection) ReadMessage() ([]byte, error) {
	_, msg, err := ws.Conn.ReadMessage()
	return msg, err
}

// To be handled by the server
func (connection *Connection) HandleClientMessage(msg WSMessage) []byte {
	switch msg.Type {
	case types.MessageTypeExchangeKeys:
		// Encapsulate ciphertext with the public key from client
		// and generates a sharedSecret
		sharedSecret, cipherText := cryptography.KeyExchange(msg.Value)

		// save the HKDF'ed sharedSecret
		connection.Keys.SharedSecret = cryptography.DeriveKey(sharedSecret)
		connection.Keys.Public = msg.Value

		msg := WSMessage{
			Type:     types.MessageTypeExchangeKeys,
			Value:    cipherText,
			Nonce:    nil,
			Metadata: WSMetadata{Username: connection.Metadata.Username, Color: connection.Metadata.Color},
		}
		jsonMsg := msg.Marshal()

		// send ciphertext to client so we can exchange keys
		if err := connection.WriteMessage(string(jsonMsg), websocket.TextMessage); err != nil {
			log.Printf("Could not send message to client: %s\n", err.Error())
			return nil
		}

	case types.MessageTypeEncryptedMessage:
		nonce := msg.Nonce
		ciphertext := msg.Value

		log.Printf("Received encrypted message: >>> %s <<<, with nonce: >>> %s <<<\n", ciphertext, nonce)
		decrypted, err := cryptography.DecryptMessage(connection.Keys.SharedSecret, nonce, ciphertext)
		if err != nil {
			log.Printf("Could not decrypt message from client (%s): %s\n", connection.Metadata.Username, err.Error())
			return nil
		}
		log.Printf("Decrypted message (%s): \"%s\"\n", connection.Metadata.Username, decrypted)

		return decrypted

	default:
		log.Printf("Received a message with an unknown type: %s\n", msg.Type)
	}

	return nil
}

// This is used by the server to fan out a message from one client to others
func (connection *Connection) RelayMessage(message, fromUsername, fromColor string) {
	nonce, ciphertext, err := cryptography.EncryptMessage(connection.Keys.SharedSecret, []byte(message))
	if err != nil {
		log.Printf("Could not encrypt message: %s\n", err.Error())
		return
	}

	msg := WSMessage{
		Type:     types.MessageTypeEncryptedMessage,
		Value:    ciphertext,
		Nonce:    nonce,
		Metadata: WSMetadata{Username: fromUsername, Color: fromColor},
	}
	jsonMsg := msg.Marshal()

	// send encrypted message
	if err := connection.WriteMessage(string(jsonMsg), websocket.TextMessage); err != nil {
		log.Printf("Could not send message to client: %s\n", err.Error())
	}

}

// To be handled by the client
func (connection *Connection) HandleServerMessage(msg WSMessage) {
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
		ui.EmitToUI(types.MessageTypeKeysExchanged, connection.Metadata.Username, connection.Metadata.Color)

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

		ui.EmitToUI(types.MessageTypeMessage, string(decrypted), msg.Metadata.Color)
	case types.MessageTypeUserEnteredChat:
		metadata := msg.Metadata
		ui.EmitToUI(types.MessageTypeUserEnteredChat, string(metadata.Username), metadata.Color)
	case types.MessageTypeUserLeftChat:
		metadata := msg.Metadata
		ui.EmitToUI(types.MessageTypeUserLeftChat, string(metadata.Username), metadata.Color)
	case types.MessageTypeCurrentUsers:
		metadata := msg.Metadata
		value := msg.Value
		ui.EmitToUI(types.MessageTypeCurrentUsers, string(value), metadata.Color)
	default:
		log.Printf("Received a message with an unknown type: %s\n", msg.Type)
	}
}
