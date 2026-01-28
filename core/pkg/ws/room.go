package ws

import (
	"time"

	"github.com/Guilospanck/pqc/core/pkg/types"
	"github.com/Guilospanck/pqc/core/pkg/utils"
)

type Room struct {
	ID          types.RoomId
	Name        string
	CreatedBy   types.ClientId
	Connections map[types.ClientId]*Connection
	createdAt   time.Time
}

func NewLobbyRoom() Room {
	return Room{
		ID:          types.RoomId(utils.LOBBY_ROOM),
		Name:        utils.LOBBY_ROOM,
		CreatedBy:   utils.SYSTEM,
		createdAt:   time.Now(),
		Connections: make(map[types.ClientId]*Connection),
	}
}

func NewRoom(creator types.ClientId, name string) Room {
	return Room{
		ID:          types.RoomId(utils.UUID()),
		Name:        name,
		CreatedBy:   creator,
		createdAt:   time.Now(),
		Connections: make(map[types.ClientId]*Connection),
	}
}

func (room *Room) AddConnection(connection *Connection) {
	room.Connections[connection.ID] = connection
}

func (room *Room) RemoveConnection(id types.ClientId) {
	delete(room.Connections, id)
}
