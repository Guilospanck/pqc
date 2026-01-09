package ui

import (
	"encoding/json"
	"fmt"
)

type UIMessageType string

const (
	ToUIConnected     UIMessageType = "connected"
	ToUIKeysExchanged UIMessageType = "keys_exchanged"
	ToUIMessage       UIMessageType = "message"

	FromUIConnect = "connect"
	FromUISend    = "send"
)

type UIMessage struct {
	Type  UIMessageType `json:"type"`
	Value string        `json:"value"`
}

// We talk to the UI via stdout
func EmitToUI(msgType UIMessageType, value string) {
	msg := UIMessage{
		Type:  msgType,
		Value: value,
	}

	msgMarshalled, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Server errored sending message. Please restart and try again.")
		return
	}

	fmt.Println(string(msgMarshalled))
}
