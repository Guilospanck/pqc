package ws

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Guilospanck/pqc/core/pkg/types"
)

type WSMetadata struct {
	Username string `json:"username"`
	Color    string `json:"color"`
}

type WSMessage struct {
	Type     types.MessageType `json:"type"`
	Value    []byte            `json:"value"`
	Nonce    []byte            `json:"nonce"`
	Metadata WSMetadata        `json:"metadata"`
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
