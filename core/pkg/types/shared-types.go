package types

type MessageType = string

const (
	// Go to TUI
	MessageTypeConnected     MessageType = "connected"
	MessageTypeDisconnected  MessageType = "disconnected"
	MessageTypeKeysExchanged MessageType = "keys_exchanged"
	MessageTypeMessage       MessageType = "message"

	// Go <-> Go (ws) and Go to TUI
	MessageTypeUserEnteredChat MessageType = "user_entered_chat"
	MessageTypeUserLeftChat    MessageType = "user_left_chat"
	MessageTypeCurrentUsers    MessageType = "current_users"

	// Go <-> Go (ws)
	MessageTypeExchangeKeys     MessageType = "exchange_keys"
	MessageTypeEncryptedMessage MessageType = "encrypted_message"

	// TUI to Go
	MessageTypeConnect MessageType = "connect"
	MessageTypeSend    MessageType = "send"
)
