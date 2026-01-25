package ui

import (
	"encoding/json"
	"fmt"

	"github.com/Guilospanck/pqc/core/pkg/types"
)

type UIMessage struct {
	Type  types.MessageType `json:"type"`
	Value string            `json:"value"`
	Color string            `json:"color"`
}

// We talk to the UI via stdout
func EmitToUI(msgType types.MessageType, value, color string) {
	var messageColor string = color
	if len(color) == 0 {
		messageColor = "#7ee787"
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
