package ui

import (
	"encoding/json"
	"fmt"

	"github.com/Guilospanck/pqc/core/pkg/types"
)

// We talk to the UI via stdout
func EmitToUI(msgType types.MessageType, value string, metadata types.WSMetadata) {
	if len(metadata.Color) == 0 {
		metadata.Color = "#7ee787"
	}

	msg := types.UIMessage{
		Type:     msgType,
		Value:    value,
		Metadata: metadata,
	}

	msgMarshalled, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Server errored sending message. Please restart and try again.")
		return
	}

	fmt.Println(string(msgMarshalled))
}
