package types

import "time"

// INFO: all types here will be transpiled into Typescript (file `generated-types.ts` in tui/)
// using `tygo` if you run the `just start-server` (or `just generate-types`) recipes.

type MessageType = string

const (
	// Go to TUI
	MessageTypeConnected     MessageType = "connected"
	MessageTypeDisconnected  MessageType = "disconnected"
	MessageTypeReconnecting  MessageType = "reconnecting"
	MessageTypeKeysExchanged MessageType = "keys_exchanged"
	MessageTypeMessage       MessageType = "message"

	// Go <-> Go (ws) and Go to TUI
	MessageTypeUserEnteredChat MessageType = "user_entered_chat"
	MessageTypeUserLeftChat    MessageType = "user_left_chat"
	MessageTypeCurrentUsers    MessageType = "current_users"
	MessageTypeSuccess         MessageType = "success"
	MessageTypeError           MessageType = "error"
	MessageTypeJoinedRoom      MessageType = "joined_room"
	MessageTypeLeftRoom        MessageType = "left_room"
	MessageTypeCreatedRoom     MessageType = "created_room"
	MessageTypeDeletedRoom     MessageType = "deleted_room"
	MessageTypeAvailableRooms  MessageType = "available_rooms"

	// Go <-> Go (ws)
	MessageTypeExchangeKeys     MessageType = "exchange_keys"
	MessageTypeEncryptedMessage MessageType = "encrypted_message"
	MessageTypeJoinRoom         MessageType = "join_room"
	MessageTypeLeaveRoom        MessageType = "leave_room"
	MessageTypeCreateRoom       MessageType = "create_room"
	MessageTypeDeleteRoom       MessageType = "delete_room"

	// TUI to Go
	MessageTypeConnect MessageType = "connect"
	MessageTypeSend    MessageType = "send"
)

type ClientId string
type RoomId string

type WSMetadata struct {
	UserId        ClientId `json:"userId"`
	Username      string   `json:"username"`
	Color         string   `json:"color"`
	CurrentRoomId RoomId   `json:"currentRoomId"`
}

type UIMessage struct {
	Type     MessageType `json:"type"`
	Value    string      `json:"value"`
	Metadata WSMetadata  `json:"metadata"`
}

type RoomInfo struct {
	ID        RoomId
	Name      string
	CreatedBy ClientId
	CreatedAt time.Time
}
