# Rooms Feature Design

## Overview
Add topic-based rooms that users can create on-demand with custom names, fully persistent until server restart, with both room list browsing and direct join capabilities.

## Architecture

### Core Data Structures

```go
type roomId string

type Room struct {
    Name        string
    ID          roomId // unique identifier
    CreatedBy   string // username of creator
    CreatedAt   time.Time
    Connections map[clientId]*ws.Connection
    mu          sync.RWMutex
}

type WSServer struct {
    rooms         map[roomId]*Room // room ID -> Room
    connections   map[clientId]*ws.Connection
    usedUsernames []string
    mu            sync.RWMutex
    ctx           context.Context
}
```

### Connection Updates
Each `Connection` will add `CurrentRoomID roomId` to track which room the user is in, defaulting to "lobby".

### New Message Types
- `MessageTypeCreateRoom` - create a named room
- `MessageTypeJoinRoom` - join an existing room by name  
- `MessageTypeLeaveRoom` - leave current room
- `MessageTypeRoomList` - get list of all rooms
- `MessageTypeRoomMessage` - message sent to current room

## User Experience Flow

1. New users join default "lobby" room automatically
2. Users can create rooms with `/create <room-name>` (returns roomId to user)
3. Users can join rooms with `/join <room-name>` (server looks up by name, then uses ID internally)
4. Users can list rooms with `/list` (shows room names + IDs)
5. Users can leave rooms with `/leave` (returns to lobby)

## Core Functions

```go
// Creates a new room with unique ID
func (srv *WSServer) createRoom(name, creatorUsername string) roomId

// User-facing: joins room by name (looks up ID internally)
func (srv *WSServer) joinRoomByName(roomName string, client *ws.Connection) error

// Internal: joins room by unique ID
func (srv *WSServer) joinRoom(roomId roomId, client *ws.Connection) error

// Leaves current room, returns to lobby
func (srv *WSServer) leaveRoom(client *ws.Connection)

// Returns all room metadata for listing
func (srv *WSServer) getRoomList() []RoomInfo

// Replaces fanOutUserMessage - only sends to room members
func (srv *WSServer) relayMessageToRoom(roomId roomId, message string, sender *ws.Connection)
```

## Message Handling

### Command Processing
- Parse user input for `/create`, `/join`, `/leave`, `/list` commands
- Route to appropriate room management functions
- Handle invalid room names, duplicate joins, etc.

### Message Routing
- `HandleClientMessage` extended to handle room commands
- `fanOutUserMessage` replaced with `relayMessageToRoom`
- Room state changes (join/leave) broadcast to affected room members
- Default "lobby" room behavior maintained

## Error Handling

- Duplicate room names allowed (different IDs)
- Invalid room IDs/names return user-friendly errors
- Room creation validation (name length, characters)
- Graceful handling of room operations during connection issues

## Persistence
- Rooms persist until server restart
- Room membership persists during user reconnections
- Default lobby always exists