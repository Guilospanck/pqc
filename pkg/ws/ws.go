package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"pqc/pkg/cryptography"

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
		log.Fatal("Error marshalling msg: ", err)
	}

	return jsonMsg
}

// This function panics if unmarshalling goes wrong
func UnmarshalWSMessage(data []byte) WSMessage {
	var msg WSMessage

	if err := json.Unmarshal(data, &msg); err != nil {
		log.Fatal("Error unmarshalling message: ", err)
	}

	return msg
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
			log.Fatal("Could not send message to client: ", err)
		}

	case EncryptedMessage:
		nonce := msg.Nonce
		ciphertext := msg.Value

		log.Printf("Received encrypted message: >>> %s <<<, with nonce: >>> %s <<<\n", ciphertext, nonce)
		decrypted, err := cryptography.DecryptMessage(connection.Keys.SharedSecret, nonce, ciphertext)
		if err != nil {
			log.Fatal("Could not decrypt message from client: ", err)
		}

		log.Printf("Decrypted message: \"%s\"\n", decrypted)
	default:
		log.Fatal("Received a message with an unknown type")
	}
}

// To be handled by the client
func (msg *WSMessage) HandleServerMessage(connection *Connection) {
	switch msg.Type {
	case ExchangeKeys:
		ciphertext := msg.Value
		sharedSecret, err := connection.Keys.Private.Decapsulate(ciphertext)
		if err != nil {
			log.Fatal("Could not get shared secret from ciphertext: ", err)
		}

		// Now the client also have the shared secret
		connection.Keys.SharedSecret = cryptography.DeriveKey(sharedSecret)
		fmt.Println("Key exchange done!")

	case EncryptedMessage:
		nonce := msg.Nonce
		ciphertext := msg.Value

		log.Printf("Received encrypted message: >>> %s <<<, with nonce: >>> %s <<<\n", ciphertext, nonce)
		decrypted, err := cryptography.DecryptMessage(connection.Keys.SharedSecret, nonce, ciphertext)
		if err != nil {
			log.Fatal("Could not decrypt message from server: ", err)
		}

		log.Printf("Decrypted message: \"%s\"\n", decrypted)
	default:
		log.Fatal("Received a message with an unknown type")
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
		log.Println("Read error:", err)
		return []byte(""), err
	}

	return msg, nil
}
