package ws

import (
	"time"

	"github.com/Guilospanck/pqc/core/pkg/utils"
)

type RoomId string

type Room struct {
	ID          RoomId
	Name        string
	CreatedBy   ClientId
	Connections map[ClientId]*Connection
	createdAt   time.Time
}

func NewLobbyRoom() Room {
	return Room{
		ID:          RoomId(utils.LOBBY_ROOM),
		Name:        utils.LOBBY_ROOM,
		CreatedBy:   utils.SYSTEM,
		createdAt:   time.Now(),
		Connections: make(map[ClientId]*Connection),
	}
}

func NewRoom(creator ClientId, name string) Room {
	return Room{
		ID:          RoomId(utils.UUID()),
		Name:        name,
		CreatedBy:   creator,
		createdAt:   time.Now(),
		Connections: make(map[ClientId]*Connection),
	}
}

func (room *Room) AddConnection(connection *Connection) {
	room.Connections[connection.ID] = connection
}

func (room *Room) RemoveConnection(connection *Connection) {
	delete(room.Connections, connection.ID)
}
