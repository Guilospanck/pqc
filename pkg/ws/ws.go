package ws

import (
	"encoding/json"
	"log"
	"pqc/pkg/cryptography"

	"github.com/gorilla/websocket"
)

type ClientToServerMessageType string

const (
	ExchangeKeys     ClientToServerMessageType = "exchange_keys"
	EncryptedMessage ClientToServerMessageType = "encrypted_message"
)

type ClientToServerMessage struct {
	Type  ClientToServerMessageType `json:"type"`
	Value []byte                    `json:"value"`
}

// This function panics if marshalling goes wrong
func (msg *ClientToServerMessage) Marshal() []byte {
	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		log.Fatal("Error marshalling msg: ", err)
	}

	return jsonMsg
}

// This function panics if unmarshalling goes wrong
func UnmarshalClientToServerMessage(data []byte) ClientToServerMessage {
	var msg ClientToServerMessage

	if err := json.Unmarshal(data, &msg); err != nil {
		log.Fatal("Error unmarshalling message from client: ", err)
	}

	return msg
}

type Connection struct {
	Keys *cryptography.Keys
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
