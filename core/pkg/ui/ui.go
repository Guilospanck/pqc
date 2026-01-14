package ui

import (
	"encoding/json"
	"fmt"
)

type UIMessageType string

const (
	ToUIConnected       UIMessageType = "connected"
	ToUIKeysExchanged   UIMessageType = "keys_exchanged"
	ToUIMessage         UIMessageType = "message"
	ToUIUserEnteredChat UIMessageType = "user_entered_chat"

	FromUIConnect = "connect"
	FromUISend    = "send"
)

type UIMessage struct {
	Type  UIMessageType `json:"type"`
	Value string        `json:"value"`
	Color string        `json:"color"`
}

// We talk to the UI via stdout
func EmitToUI(msgType UIMessageType, value string, color []byte) {
	var messageColor []byte = color
	if color == nil {
		messageColor = []byte("#7ee787")
	}

	msg := UIMessage{
		Type:  msgType,
		Value: value,
		Color: string(messageColor),
	}

	msgMarshalled, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Server errored sending message. Please restart and try again.")
		return
	}

	fmt.Println(string(msgMarshalled))
}
