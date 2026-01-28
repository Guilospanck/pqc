package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"sync"

	"github.com/Guilospanck/pqc/core/pkg/cryptography"
	"github.com/Guilospanck/pqc/core/pkg/types"
	"github.com/Guilospanck/pqc/core/pkg/utils"
	"github.com/Guilospanck/pqc/core/pkg/ws"

	"github.com/gorilla/websocket"
)

type WSServer struct {
	rooms         map[types.RoomId]*ws.Room
	connections   map[types.ClientId]*ws.Connection
	usedUsernames []string
	mu            sync.RWMutex
	ctx           context.Context
}

func NewServer(ctx context.Context) *WSServer {
	// create lobby room
	rooms := make(map[types.RoomId]*ws.Room)

	lobbyRoom := ws.NewLobbyRoom()
	rooms[lobbyRoom.ID] = &lobbyRoom

	return &WSServer{
		rooms:         rooms,
		connections:   make(map[types.ClientId]*ws.Connection),
		ctx:           ctx,
		usedUsernames: make([]string, 0),
	}
}

func (srv *WSServer) startServer() {
	http.HandleFunc("/ws", srv.wsHandler)

	log.Print("WS server started at localhost:8080/ws")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (srv *WSServer) wsHandler(w http.ResponseWriter, r *http.Request) {
	connection := ws.NewEmptyConnection()

	srv.handleConnectionMetadata(r.Header, &connection)

	conn, err := srv.upgradeWSConnection(w, r, *connection.Metadata)
	if err != nil {
		log.Print("Error upgrading WS: ", err)
		return
	}
	defer conn.Close()

	connection.Conn = conn
	srv.addConnection(&connection)

	log.Printf("New connection: %s - %s\n", connection.Metadata.Username, connection.Metadata.Color)

	// Start write loop
	go connection.WriteLoop(srv.ctx)

	<-connection.WriteLoopReady

	// Update this newly connected user with info regarding all connected users
	srv.informUserOfAllCurrentUsers(&connection)

	// Send to other clients the event of a newly connected client
	srv.informRoomOfNewUser(&connection)

	// Start read loop
	srv.readAndHandleClientMessages(&connection)
}

func (srv *WSServer) addConnection(connection *ws.Connection) {
	// Add to server connections
	srv.connections[connection.ID] = connection

	// Add/Update to the correct room
	currentRoomId := connection.Metadata.CurrentRoomId
	_, roomExists := srv.rooms[currentRoomId]
	if !roomExists {
		log.Printf("User %s tried to access roomId %s that does not exist. Adding him to lobby.\n", connection.Metadata.Username, currentRoomId)
		currentRoomId = utils.LOBBY_ROOM
	}

	srv.joinRoomById(currentRoomId, connection)
}

func (srv *WSServer) removeConnection(id types.ClientId) {
	// Remove client from rooms
	for _, r := range srv.rooms {
		r.RemoveConnection(id)
	}

	// Remove from server connections
	delete(srv.connections, id)
}

func (srv *WSServer) getRandomUsername() string {
	generatedUsername := ""

	for {
		generatedUsername = GetRandomName()
		if !slices.Contains(srv.usedUsernames, generatedUsername) {
			break
		}
	}

	srv.usedUsernames = append(srv.usedUsernames, generatedUsername)
	return generatedUsername
}

func (srv *WSServer) upgradeWSConnection(w http.ResponseWriter, r *http.Request, connectionMetadata types.WSMetadata) (*websocket.Conn, error) {
	upgrader := websocket.Upgrader{
		// INFO: for production you should make this more restrictive
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// Send the generated username and color to the WSClient
	// INFO: it needs to be *before* the upgrade
	responseHeader := http.Header{}
	responseHeader.Set("username", connectionMetadata.Username)
	responseHeader.Set("color", connectionMetadata.Color)
	responseHeader.Set("roomId", string(connectionMetadata.CurrentRoomId))

	return upgrader.Upgrade(w, r, responseHeader)
}

func (srv *WSServer) handleConnectionMetadata(headers http.Header, connection *ws.Connection) {
	username := headers.Get("username")
	color := headers.Get("color")
	currentRoomId := headers.Get("roomId")

	if username == "" {
		username = srv.getRandomUsername()
	}
	if color == "" {
		color = GetRandomColor()
	}
	if currentRoomId == "" {
		currentRoomId = utils.LOBBY_ROOM
	}

	metadata := types.WSMetadata{
		Username:      username,
		Color:         color,
		CurrentRoomId: types.RoomId(currentRoomId),
	}

	connection.Metadata = &metadata
}

func (srv *WSServer) joinRoomById(roomId types.RoomId, connection *ws.Connection) {
	if room, roomExists := srv.rooms[roomId]; roomExists {
		room.AddConnection(connection)

		// point user to new room
		connection.Metadata.CurrentRoomId = room.ID
	}
}

// TODO: change the message if a user tries to leave a room he is not in.
func (srv *WSServer) leaveRoomById(roomId types.RoomId, connection *ws.Connection) {
	if room, roomExists := srv.rooms[roomId]; roomExists {
		room.RemoveConnection(connection.ID)

		isConnectionCurrentlyInRoom := connection.Metadata.CurrentRoomId == room.ID
		if isConnectionCurrentlyInRoom {
			srv.joinRoomById(utils.LOBBY_ROOM, connection)
		}
	}
}

func (srv *WSServer) joinRoomByName(name string, connection *ws.Connection) error {
	for _, room := range srv.rooms {
		if room.Name == name {
			srv.joinRoomById(room.ID, connection)
			return nil
		}
	}

	return fmt.Errorf("could not find a room named \"%s\"", name)
}

func (srv *WSServer) leaveRoomByName(name string, connection *ws.Connection) error {
	for _, room := range srv.rooms {
		if room.Name == name {
			srv.leaveRoomById(room.ID, connection)
			return nil
		}
	}

	return fmt.Errorf("could not find a room named \"%s\"", name)
}

func (srv *WSServer) createRoom(name string, creator types.ClientId) {
	room := ws.NewRoom(creator, name)
	srv.rooms[room.ID] = &room
}

func (srv *WSServer) deleteRoomByName(name string, connection *ws.Connection) error {
	var room *ws.Room = nil

	for _, r := range srv.rooms {
		if r.Name == name {
			room = r
			break
		}
	}

	if room == nil {
		return fmt.Errorf("could not delete the room named \"%s\" because it does not exist", name)
	}

	if room.CreatedBy != connection.ID {
		return fmt.Errorf("could not delete the room named \"%s\" because you do not have permissions to do that", name)
	}

	isConnectionCurrentlyInRoom := connection.Metadata.CurrentRoomId == room.ID
	if len(room.Connections) > 1 || len(room.Connections) == 1 && !isConnectionCurrentlyInRoom {
		return fmt.Errorf("cannot delete the room as it has other participants there.")
	}

	if isConnectionCurrentlyInRoom {
		srv.joinRoomById(utils.LOBBY_ROOM, connection)
	}

	delete(srv.rooms, room.ID)

	return nil
}

func (srv *WSServer) readAndHandleClientMessages(connection *ws.Connection) {
	for {
		msg, err := connection.ReadMessage()
		if err != nil {
			log.Printf("Error reading from conn: %s\n", err.Error())
			srv.userDisconnected(connection)
			return
		}

		msgJson, err := ws.UnmarshalWSMessage(msg)
		if err != nil {
			log.Printf("Error unmarshalling message: %s\n", err.Error())
			continue
		}

		srv.handleClientMessage(msgJson, connection)
	}
}

func (srv *WSServer) handleClientMessage(msg ws.WSMessage, connection *ws.Connection) {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	wsMessage := ws.WSMessage{
		Value:    nil,
		Nonce:    nil,
		Metadata: types.WSMetadata{Username: connection.Metadata.Username, Color: connection.Metadata.Color, CurrentRoomId: connection.Metadata.CurrentRoomId},
	}

	sendMessageToClient := func() {
		jsonMsg := wsMessage.Marshal()

		if err := connection.WriteMessage(string(jsonMsg), websocket.TextMessage); err != nil {
			log.Printf("Could not send message to client: %s\n", err.Error())
			return
		}
	}

	switch msg.Type {
	case types.MessageTypeExchangeKeys:
		// Encapsulate ciphertext with the public key from client
		// and generates a sharedSecret
		sharedSecret, cipherText := cryptography.KeyExchange(msg.Value)

		// save the HKDF'ed sharedSecret
		connection.Keys.SharedSecret = cryptography.DeriveKey(sharedSecret)
		connection.Keys.Public = msg.Value

		wsMessage.Value = cipherText
		wsMessage.Type = types.MessageTypeExchangeKeys

		// send ciphertext to client so we can exchange keys
		sendMessageToClient()

	case types.MessageTypeEncryptedMessage:
		nonce := msg.Nonce
		ciphertext := msg.Value

		decrypted, err := cryptography.DecryptMessage(connection.Keys.SharedSecret, nonce, ciphertext)
		if err != nil {
			log.Printf("Could not decrypt message from client (%s): %s\n", connection.Metadata.Username, err.Error())
			return
		}

		if decrypted == nil {
			return
		}

		srv.sendEncryptedMessageToAllConnectionsInTheSameRoom(connection, decrypted)

	case types.MessageTypeJoinRoom:
		oldRoom := connection.Metadata.CurrentRoomId
		roomName := string(msg.Value)
		if err := srv.joinRoomByName(roomName, connection); err != nil {
			wsMessage.Type = types.MessageTypeError
			wsMessage.Value = []byte(err.Error())
		} else {
			wsMessage.Type = types.MessageTypeSuccess
			wsMessage.Value = fmt.Appendf(nil, "Joined room %s", roomName)
			log.Printf("%s joined room %s", connection.Metadata.Username, roomName)

			// Remove connection from old room
			log.Printf("Removing %s from old room %s\n", connection.Metadata.Username, oldRoom)
			srv.leaveRoomById(oldRoom, connection)
		}

		wsMessage.Metadata.CurrentRoomId = connection.Metadata.CurrentRoomId
		sendMessageToClient()

	case types.MessageTypeDeleteRoom:
		roomName := string(msg.Value)
		if err := srv.deleteRoomByName(roomName, connection); err != nil {
			wsMessage.Type = types.MessageTypeError
			wsMessage.Value = []byte(err.Error())
		} else {
			wsMessage.Type = types.MessageTypeSuccess
			wsMessage.Value = fmt.Appendf(nil, "Deleted room %s", roomName)
		}

		wsMessage.Metadata.CurrentRoomId = connection.Metadata.CurrentRoomId
		sendMessageToClient()

	case types.MessageTypeCreateRoom:
		roomName := string(msg.Value)
		srv.createRoom(roomName, connection.ID)
		wsMessage.Type = types.MessageTypeSuccess
		wsMessage.Value = fmt.Appendf(nil, "Created room %s", roomName)

		wsMessage.Metadata.CurrentRoomId = connection.Metadata.CurrentRoomId
		sendMessageToClient()

	case types.MessageTypeLeaveRoom:
		roomName := string(msg.Value)
		if err := srv.leaveRoomByName(roomName, connection); err != nil {
			wsMessage.Type = types.MessageTypeError
			wsMessage.Value = []byte(err.Error())
		} else {
			wsMessage.Type = types.MessageTypeSuccess
			wsMessage.Value = fmt.Appendf(nil, "Left room %s", roomName)
		}

		wsMessage.Metadata.CurrentRoomId = connection.Metadata.CurrentRoomId
		sendMessageToClient()

	default:
		log.Printf("Received a message with an unknown type: %s\n", msg.Type)
	}

	log.Printf(">>>> CURRENT ROOM for user %s: %s\n", connection.Metadata.Username, connection.Metadata.CurrentRoomId)
}

// Remove client from connections and broadcast user left event to its current room
func (srv *WSServer) userDisconnected(connection *ws.Connection) {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	srv.removeConnection(connection.ID)

	room, exists := srv.rooms[connection.Metadata.CurrentRoomId]
	if !exists {
		return
	}

	for _, c := range room.Connections {
		// Broadcast user left event to other clients
		leftMsg := ws.WSMessage{
			Type:     types.MessageTypeUserLeftChat,
			Value:    nil,
			Nonce:    nil,
			Metadata: types.WSMetadata{Username: connection.Metadata.Username, Color: connection.Metadata.Color},
		}
		leftJsonMsg := leftMsg.Marshal()

		if err := c.WriteMessage(string(leftJsonMsg), websocket.TextMessage); err != nil {
			log.Printf("Error trying to inform client %s that user %s left: %s\n", c.ID, connection.ID, err.Error())
		}
	}
}

func (srv *WSServer) informUserOfAllCurrentUsers(newUser *ws.Connection) {
	srv.mu.RLock()
	defer srv.mu.RUnlock()

	room := srv.rooms[newUser.Metadata.CurrentRoomId]
	connections := room.Connections

	users := make([]types.WSMetadata, 0, len(connections))

	for _, c := range connections {
		users = append(users, types.WSMetadata{Username: c.Metadata.Username, Color: c.Metadata.Color, CurrentRoomId: c.Metadata.CurrentRoomId})
	}

	marshalledUsers, err := json.Marshal(users)
	if err != nil {
		log.Println("Could not marshal users to inform newly connected user")
		return
	}

	msg := ws.WSMessage{
		Type:     types.MessageTypeCurrentUsers,
		Value:    marshalledUsers,
		Nonce:    nil,
		Metadata: types.WSMetadata{Username: newUser.Metadata.Username, Color: newUser.Metadata.Color, CurrentRoomId: newUser.Metadata.CurrentRoomId},
	}
	jsonMsg := msg.Marshal()

	if err = newUser.WriteMessage(string(jsonMsg), websocket.TextMessage); err != nil {
		log.Println("Problem sending message to the client regarding the currently connected users: ", err)
	}
}

// Inform people of a room that a new user has connected.
func (srv *WSServer) informRoomOfNewUser(connection *ws.Connection) {
	srv.mu.RLock()
	defer srv.mu.RUnlock()

	msg := ws.WSMessage{
		Type:     types.MessageTypeUserEnteredChat,
		Value:    nil,
		Nonce:    nil,
		Metadata: *connection.Metadata,
	}
	jsonMsg := msg.Marshal()

	room, exists := srv.rooms[connection.Metadata.CurrentRoomId]
	if !exists {
		log.Printf("Room with id %s does not exist on server.\n", connection.Metadata.CurrentRoomId)
		return
	}

	for _, c := range room.Connections {
		if c.ID == connection.ID {
			continue
		}

		if err := c.WriteMessage(string(jsonMsg), websocket.TextMessage); err != nil {
			log.Printf("Error trying to inform the client %s that a new connection was made: %s\n", c.Metadata.Username, err.Error())
		}
	}
}

func (srv *WSServer) sendEncryptedMessageToAllConnectionsInTheSameRoom(client *ws.Connection, decryptedMessage []byte) {
	room, exists := srv.rooms[client.Metadata.CurrentRoomId]
	if !exists {
		log.Printf("Room with id %s does not exist on server.\n", client.Metadata.CurrentRoomId)
		return
	}

	for _, c := range room.Connections {
		if c.ID == client.ID {
			continue
		}

		msgWithPublicKey := fmt.Sprintf("%s: %s", client.Metadata.Username, string(decryptedMessage))

		c.RelayMessage(msgWithPublicKey, client.Metadata.Username, client.Metadata.Color)
	}
}
