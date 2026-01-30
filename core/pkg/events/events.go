package events

import (
	"log"

	"encoding/json"
	"github.com/Guilospanck/pqc/core/pkg/types"
	"github.com/Guilospanck/pqc/core/pkg/ws"
	"github.com/gorilla/websocket"
)

func sendMessageToConnection(connection *ws.Connection, value []byte, metadata types.WSMetadata, msgType types.MessageType) {
	wsMessage := ws.WSMessage{
		Value:    value,
		Nonce:    nil,
		Metadata: metadata,
		Type:     msgType,
	}

	jsonMsg := wsMessage.Marshal()

	if err := connection.WriteMessage(string(jsonMsg), websocket.TextMessage); err != nil {
		log.Printf("Could not send message `%s` to client: %s\n", msgType, err.Error())
		return
	}
}

func triggerJoinedRoom(connection *ws.Connection, value []byte, metadata types.WSMetadata) {
	sendMessageToConnection(connection, value, metadata, types.MessageTypeJoinedRoom)
}

func triggerLeftRoom(connection *ws.Connection, value []byte, metadata types.WSMetadata) {
	sendMessageToConnection(connection, value, metadata, types.MessageTypeLeftRoom)
}

func triggerCreatedRoom(connection *ws.Connection, value []byte, metadata types.WSMetadata) {
	sendMessageToConnection(connection, value, metadata, types.MessageTypeCreatedRoom)
}

func triggerDeletedRoom(connection *ws.Connection, value []byte, metadata types.WSMetadata) {
	sendMessageToConnection(connection, value, metadata, types.MessageTypeDeletedRoom)
}

// TODO: maybe check to implement a "RoomType"
func TriggerRoomEvent(eventType types.MessageType, room *ws.Room, connection *ws.Connection) {
	roomInfo := room.GetRoomInfo()
	marshalledRoomInfo, err := json.Marshal(roomInfo)
	if err != nil {
		log.Printf("Error trying to marshall room in the `%s` event: %s\n", eventType, err.Error())
	}

	switch eventType {
	case types.MessageTypeJoinedRoom:
		triggerJoinedRoom(connection, marshalledRoomInfo, *connection.Metadata)

	case types.MessageTypeLeftRoom:
		triggerLeftRoom(connection, marshalledRoomInfo, *connection.Metadata)

	case types.MessageTypeCreatedRoom:
		triggerCreatedRoom(connection, marshalledRoomInfo, *connection.Metadata)

	case types.MessageTypeDeletedRoom:
		triggerDeletedRoom(connection, marshalledRoomInfo, *connection.Metadata)
	}
}
