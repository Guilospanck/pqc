package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"pqc/pkg/cryptography"
	"pqc/pkg/ui"

	"github.com/gorilla/websocket"
)

type WSMessageType string

const (
	ExchangeKeys     WSMessageType = "exchange_keys"
	EncryptedMessage WSMessageType = "encrypted_message"
)

type WSMessage struct {
	Type  WSMessageType `json:"type"`
	Value []byte        `json:"value"`
	Nonce []byte        `json:"nonce"`
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

// To be handled by the server
func (msg *WSMessage) HandleClientMessage(connection *Connection) {
	switch msg.Type {
	case ExchangeKeys:
		// Encapsulate ciphertext with the public key from client
		// and generates a sharedSecret
		sharedSecret, cipherText := cryptography.KeyExchange(msg.Value)

		// save the HKDF'ed sharedSecret
		connection.Keys.SharedSecret = cryptography.DeriveKey(sharedSecret)

		msg := WSMessage{
			Type:  ExchangeKeys,
			Value: cipherText,
			Nonce: nil,
		}
		jsonMsg := msg.Marshal()

		// send ciphertext to client so we can exchange keys
		if err := connection.WriteMessage(string(jsonMsg)); err != nil {
			log.Printf("Could not send message to client: %s\n", err.Error())
			return
		}

	case EncryptedMessage:
		nonce := msg.Nonce
		ciphertext := msg.Value

		log.Printf("Received encrypted message: >>> %s <<<, with nonce: >>> %s <<<\n", ciphertext, nonce)
		decrypted, err := cryptography.DecryptMessage(connection.Keys.SharedSecret, nonce, ciphertext)
		if err != nil {
			log.Printf("Could not decrypt message from client: %s\n", err.Error())
			return
		}

		log.Printf("Decrypted message (from client): \"%s\"\n", decrypted)
	default:
		log.Printf("Received a message with an unknown type: %s\n", msg.Type)
	}
}

// To be handled by the client
func (msg *WSMessage) HandleServerMessage(connection *Connection) {
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
		ui.EmitToUI(ui.ToUIKeysExchanged, "")

	case EncryptedMessage:
		nonce := msg.Nonce
		ciphertext := msg.Value

		log.Printf("Received encrypted message: >>> %s <<<, with nonce: >>> %s <<<\n", ciphertext, nonce)
		decrypted, err := cryptography.DecryptMessage(connection.Keys.SharedSecret, nonce, ciphertext)
		if err != nil {
			log.Printf("Could not decrypt message from server: %s\n", err.Error())
			return
		}

		log.Printf("Decrypted message (from server): \"%s\"\n", decrypted)
		ui.EmitToUI(ui.ToUIMessage, string(decrypted))
	default:
		log.Printf("Received a message with an unknown type: %s\n", msg.Type)
	}
}

type Connection struct {
	Keys cryptography.Keys
	Conn *websocket.Conn
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
		// Don't log normal connection closures
		if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
			log.Println("Read error:", err)
		}
		return []byte(""), err
	}

	return msg, nil
}
