package ws

import (
	"encoding/json"
	"fmt"
	"log"
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
