package ui

import (
	"encoding/json"
	"fmt"
)

// UI to Go `Type`: "connect" | "send"
// Go to UI `Type`: "connected" | "keys_exchanged" | "message"

type UIMessage struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// We talk to the UI via stdout
func EmitToUI(msgType, value string) {
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
