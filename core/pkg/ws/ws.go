package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"pqc/pkg/cryptography"
	"pqc/pkg/ui"

	"github.com/gorilla/websocket"
)

// Type of communications between WS client and WS server
type WSMessageType string

type WSMetadata struct {
	Username string `json:"username"`
	Color    string `json:"color"`
}

const (
	ExchangeKeys     WSMessageType = "exchange_keys"
	EncryptedMessage WSMessageType = "encrypted_message"
	UserEntered      WSMessageType = "user_entered_chat"
	UserLeft         WSMessageType = "user_left_chat"
	CurrentUsers     WSMessageType = "current_users"
)

type WSMessage struct {
	Type     WSMessageType `json:"type"`
	Value    []byte        `json:"value"`
	Nonce    []byte        `json:"nonce"`
	Metadata WSMetadata    `json:"metadata"`
}

// This function panics if marshalling goes wrong
func (msg *WSMessage) Marshal() []byte {
	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshalling msg: %s\n", err.Error())
		return []byte{}
	}

	return jsonMsg
}

// This function returns error if unmarshalling goes wrong
func UnmarshalWSMessage(data []byte) (WSMessage, error) {
	var msg WSMessage

	if err := json.Unmarshal(data, &msg); err != nil {
		return WSMessage{}, fmt.Errorf("error unmarshalling message: %w", err)
	}

	return msg, nil
}

type WriteMessageRequest struct {
	msgType int // websocket.TextMessage, websocket.PingMessage
	text    []byte
	err     chan error
}

type Connection struct {
	Keys            cryptography.Keys
	Conn            *websocket.Conn
	Metadata        WSMetadata
	WriteMessageReq chan WriteMessageRequest

	WriteLoopReady chan struct{}
	done           chan struct{}
}

func (ws *Connection) WriteLoop(ctx context.Context) {
	close(ws.WriteLoopReady)
	defer close(ws.done)

	for {
		select {
		case msg := <-ws.WriteMessageReq:

			text := msg.text
			msgType := msg.msgType

			err := ws.Conn.WriteMessage(msgType, text)

			select {
			case msg.err <- err:
			case <-ctx.Done():
				log.Println("Context cancelled. Returning from write loop.")
				return
			}

			if err != nil {
				log.Println("Error while writing message. Returning from write loop")
				return
			}

		case <-ctx.Done():
			log.Println("Context cancelled. Returning from write loop.")
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
	case <-ws.done:
		return errors.New("connection closed")
	}

	select {
	case err := <-errCh:
		return err
	case <-ws.done:
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
	case ExchangeKeys:
		// Encapsulate ciphertext with the public key from client
		// and generates a sharedSecret
		sharedSecret, cipherText := cryptography.KeyExchange(msg.Value)

		// save the HKDF'ed sharedSecret
		connection.Keys.SharedSecret = cryptography.DeriveKey(sharedSecret)
		connection.Keys.Public = msg.Value

		msg := WSMessage{
			Type:     ExchangeKeys,
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

	case EncryptedMessage:
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
		Type:     EncryptedMessage,
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
	case ExchangeKeys:
		ciphertext := msg.Value
		sharedSecret, err := connection.Keys.Private.Decapsulate(ciphertext)
		if err != nil {
			log.Printf("Could not get shared secret from ciphertext: %s\n", err.Error())
			return
		}

		// Now the client also have the shared secret
		connection.Keys.SharedSecret = cryptography.DeriveKey(sharedSecret)
		ui.EmitToUI(ui.ToUIKeysExchanged, connection.Metadata.Username, connection.Metadata.Color)

	case EncryptedMessage:
		nonce := msg.Nonce
		ciphertext := msg.Value

		log.Printf("Received encrypted message: >>> %s <<<, with nonce: >>> %s <<<\n", ciphertext, nonce)
		decrypted, err := cryptography.DecryptMessage(connection.Keys.SharedSecret, nonce, ciphertext)
		if err != nil {
			log.Printf("Could not decrypt message from server: %s\n", err.Error())
			return
		}

		ui.EmitToUI(ui.ToUIMessage, string(decrypted), msg.Metadata.Color)
	case UserEntered:
		metadata := msg.Metadata
		ui.EmitToUI(ui.ToUIUserEnteredChat, string(metadata.Username), metadata.Color)
	case UserLeft:
		metadata := msg.Metadata
		ui.EmitToUI(ui.ToUIUserLeftChat, string(metadata.Username), metadata.Color)
	case CurrentUsers:
		metadata := msg.Metadata
		value := msg.Value
		ui.EmitToUI(ui.ToUICurrentUsers, string(value), metadata.Color)
	default:
		log.Printf("Received a message with an unknown type: %s\n", msg.Type)
	}
}
