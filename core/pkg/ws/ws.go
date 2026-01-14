package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"pqc/pkg/cryptography"
	"pqc/pkg/ui"

	"github.com/gorilla/websocket"
)

// Type of communications between WS client and WS server
type WSMessageType string

type WSMetadata struct {
	Username []byte `json:"username"`
	Color    []byte `json:"color"`
}

const (
	ExchangeKeys     WSMessageType = "exchange_keys"
	EncryptedMessage WSMessageType = "encrypted_message"
	NewConnection    WSMessageType = "new_connection"
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

type Connection struct {
	Keys     cryptography.Keys
	Conn     *websocket.Conn
	Metadata WSMetadata
}

func (ws *Connection) WriteMessage(text string) error {
	err := ws.Conn.WriteMessage(websocket.TextMessage, []byte(text))
	if err != nil {
		log.Println("Write error:", err)
		return err
	}

	return nil
}

func (ws *Connection) ReadMessage() ([]byte, error) {
	_, msg, err := ws.Conn.ReadMessage()
	if err != nil {
		return []byte(""), err
	}

	return msg, nil
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
		if err := connection.WriteMessage(string(jsonMsg)); err != nil {
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
func (connection *Connection) RelayMessage(message string, fromUsername, fromColor []byte) {
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
	if err := connection.WriteMessage(string(jsonMsg)); err != nil {
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
		ui.EmitToUI(ui.ToUIKeysExchanged, "", nil)

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
	case NewConnection:
		metadata := msg.Metadata

		ui.EmitToUI(ui.ToUIUserEnteredChat, string(metadata.Username), metadata.Color)
	default:
		log.Printf("Received a message with an unknown type: %s\n", msg.Type)
	}
}
