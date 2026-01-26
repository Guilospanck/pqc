package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Guilospanck/pqc/core/pkg/cryptography"
	"github.com/Guilospanck/pqc/core/pkg/types"
	"github.com/Guilospanck/pqc/core/pkg/ws"

	"github.com/gorilla/websocket"
)

type clientId string

type Room struct {
	Name        string
	ID          types.RoomId
	CreatedBy   string
	CreatedAt   time.Time
	Connections map[clientId]*ws.Connection
	mu          sync.RWMutex
}

type WSServer struct {
	rooms         map[types.RoomId]*Room
	connections   map[clientId]*ws.Connection
	usedUsernames []string
	mu            sync.RWMutex
	ctx           context.Context
}

func NewServer(ctx context.Context) *WSServer {
	server := &WSServer{
		rooms:         make(map[types.RoomId]*Room),
		connections:   make(map[clientId]*ws.Connection),
		ctx:           ctx,
		usedUsernames: make([]string, 0),
	}

	// Create default lobby room
	lobby := &Room{
		Name:        "lobby",
		ID:          "lobby",
		CreatedBy:   "system",
		CreatedAt:   time.Now(),
		Connections: make(map[clientId]*ws.Connection),
	}
	server.rooms["lobby"] = lobby

	return server
}

func (srv *WSServer) generateRoomId() types.RoomId {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return types.RoomId(hex.EncodeToString(bytes))
}

type RoomInfo struct {
	Name        string
	ID          types.RoomId
	CreatedBy   string
	MemberCount int
}

func (srv *WSServer) createRoom(name, creatorUsername string) types.RoomId {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	roomID := srv.generateRoomId()
	room := &Room{
		Name:        name,
		ID:          roomID,
		CreatedBy:   creatorUsername,
		CreatedAt:   time.Now(),
		Connections: make(map[clientId]*ws.Connection),
	}
	srv.rooms[roomID] = room
	return roomID
}

func (srv *WSServer) findRoomByName(name string) (types.RoomId, error) {
	srv.mu.RLock()
	defer srv.mu.RUnlock()

	for id, room := range srv.rooms {
		if room.Name == name {
			return id, nil
		}
	}
	return "", fmt.Errorf("room '%s' not found", name)
}

func (srv *WSServer) joinRoomByName(roomName string, client *ws.Connection) error {
	roomID, err := srv.findRoomByName(roomName)
	if err != nil {
		return err
	}
	return srv.joinRoom(roomID, client)
}

func (srv *WSServer) joinRoom(roomID types.RoomId, client *ws.Connection) error {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	room, exists := srv.rooms[roomID]
	if !exists {
		return fmt.Errorf("room with ID '%s' not found", roomID)
	}

	// Remove from current room if already in one
	if client.CurrentRoomID != "" {
		if currentRoom, exists := srv.rooms[client.CurrentRoomID]; exists {
			delete(currentRoom.Connections, clientId(client.Metadata.Username))
		}
	}

	// Add to new room
	room.Connections[clientId(client.Metadata.Username)] = client
	client.CurrentRoomID = roomID
	return nil
}

func (srv *WSServer) leaveRoom(client *ws.Connection) error {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	if client.CurrentRoomID == "" {
		return fmt.Errorf("not currently in any room")
	}

	// Remove from current room
	if currentRoom, exists := srv.rooms[client.CurrentRoomID]; exists {
		delete(currentRoom.Connections, clientId(client.Metadata.Username))
	}

	// Join lobby
	client.CurrentRoomID = "lobby"
	srv.rooms["lobby"].Connections[clientId(client.Metadata.Username)] = client
	return nil
}

func (srv *WSServer) getRoomList() []RoomInfo {
	srv.mu.RLock()
	defer srv.mu.RUnlock()

	roomList := make([]RoomInfo, 0, len(srv.rooms))
	for _, room := range srv.rooms {
		roomList = append(roomList, RoomInfo{
			Name:        room.Name,
			ID:          room.ID,
			CreatedBy:   room.CreatedBy,
			MemberCount: len(room.Connections),
		})
	}
	return roomList
}

func (srv *WSServer) relayMessageToRoom(roomID types.RoomId, message string, sender *ws.Connection) {
	srv.mu.RLock()
	room, exists := srv.rooms[roomID]
	srv.mu.RUnlock()

	if !exists {
		log.Printf("Attempted to relay message to non-existent room '%s'", roomID)
		return
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	for _, recipient := range room.Connections {
		// Don't send message back to sender
		if recipient.Metadata.Username == sender.Metadata.Username {
			continue
		}

		msgWithSender := fmt.Sprintf("%s: %s", sender.Metadata.Username, message)
		log.Printf("Relaying message: \"%s\" from \"%s\" to client \"%s\" in room \"%s\"", msgWithSender, sender.Metadata.Username, recipient.Metadata.Username, room.Name)
		recipient.RelayMessage(msgWithSender, sender.Metadata.Username, sender.Metadata.Color)
	}
}

func (srv *WSServer) addConnection(connection *ws.Connection) {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	srv.connections[clientId(connection.Metadata.Username)] = connection
}

func (srv *WSServer) removeConnection(id clientId) {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	delete(srv.connections, id)
}

func (srv *WSServer) currentConnections() []ws.Connection {
	srv.mu.RLock()
	defer srv.mu.RUnlock()

	connections := make([]ws.Connection, 0, len(srv.connections))

	for _, c := range srv.connections {
		connections = append(connections, *c)
	}

	return connections
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

func (srv *WSServer) startServer() {
	http.HandleFunc("/ws", srv.wsHandler)

	log.Print("WS server started at localhost:8080/ws")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

var upgrader = websocket.Upgrader{
	// INFO: for production you should make this more restrictive
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (srv *WSServer) wsHandler(w http.ResponseWriter, r *http.Request) {
	connection := ws.NewEmptyConnection()

	// If a client is reconnecting,
	// then it will send what was its last known name and color.
	// Also their keys, for that matter.
	headers := r.Header
	username := headers.Get("username")
	color := headers.Get("color")
	if username == "" || color == "" {
		username = srv.getRandomUsername()
		color = GetRandomColor()
	}

	connection.Metadata.Username = username
	connection.Metadata.Color = color

	// Send the generated username and color to the WSClient
	// INFO: it needs to be *before* the upgrade
	responseHeader := http.Header{}
	responseHeader.Set("username", username)
	responseHeader.Set("color", color)

	conn, err := upgrader.Upgrade(w, r, responseHeader)

	if err != nil {
		log.Print("Error upgrading WS: ", err)
		return
	}
	defer conn.Close()

	connection.Conn = conn
	srv.addConnection(&connection)

	// Add user to lobby room
	srv.joinRoom("lobby", &connection)

	log.Printf("New connection: %s - %s (joined lobby)\n", username, color)

	// Start write loop
	go connection.WriteLoop(srv.ctx)

	<-connection.WriteLoopReady

	// Update this newly connected user with info regarding all connected users
	srv.informUserOfAllCurrentUsers(&connection)

	// Send to other clients the event of a newly connected client
	srv.fanOutUserEnteredChat(username, color)

	// Start read loop
	srv.readAndHandleClientMessages(&connection)
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

		decryptedMessageSent := connection.HandleClientMessage(msgJson)

		if msgJson.Type != types.MessageTypeEncryptedMessage || decryptedMessageSent == nil {
			continue
		}

		// Check if this is a room command
		message := string(decryptedMessageSent)
		if strings.HasPrefix(message, "/") {
			srv.handleRoomCommand(message, connection)
			continue
		}

		srv.relayMessageToRoom(connection.CurrentRoomID, message, connection)
	}
}

// Remove client from connections and broadcast user left event
func (srv *WSServer) userDisconnected(connection *ws.Connection) {
	connections := srv.currentConnections()

	for _, v := range connections {
		if v.Metadata.Username != connection.Metadata.Username {
			continue
		}

		srv.removeConnection(clientId(v.Metadata.Username))

		// Broadcast user left event to other clients
		leftMsg := ws.WSMessage{
			Type:     types.MessageTypeUserLeftChat,
			Value:    nil,
			Nonce:    nil,
			Metadata: ws.WSMetadata{Username: connection.Metadata.Username, Color: connection.Metadata.Color},
		}
		leftJsonMsg := leftMsg.Marshal()
		for _, c := range srv.currentConnections() {
			if err := c.WriteMessage(string(leftJsonMsg), websocket.TextMessage); err != nil {
				log.Printf("Error trying to inform clients that user left: %s\n", err.Error())
			}
		}

		// Remove from current room
		if connection.CurrentRoomID != "" {
			if currentRoom, exists := srv.rooms[connection.CurrentRoomID]; exists {
				delete(currentRoom.Connections, clientId(connection.Metadata.Username))
			}
		}

		break
	}
}

func (srv *WSServer) sendSystemMessage(message, color string, connection *ws.Connection) {
	nonce, ciphertext, err := cryptography.EncryptMessage(connection.Keys.SharedSecret, []byte(message))
	if err != nil {
		log.Printf("Could not encrypt system message: %s\n", err.Error())
		return
	}

	msg := ws.WSMessage{
		Type:     types.MessageTypeEncryptedMessage,
		Value:    ciphertext,
		Nonce:    nonce,
		Metadata: ws.WSMetadata{Username: "system", Color: color},
	}
	jsonMsg := msg.Marshal()

	if err := connection.WriteMessage(string(jsonMsg), websocket.TextMessage); err != nil {
		log.Printf("Could not send system message to client: %s\n", err.Error())
	}
}

func (srv *WSServer) handleRoomCommand(message string, connection *ws.Connection) {
	parts := strings.Fields(message)
	if len(parts) == 0 {
		return
	}

	command := parts[0]

	switch command {
	case "/create":
		if len(parts) < 2 {
			srv.sendSystemMessage("Usage: /create <room-name>", "red", connection)
			return
		}
		roomName := strings.Join(parts[1:], " ")
		roomID := srv.createRoom(roomName, connection.Metadata.Username)
		msg := fmt.Sprintf("Created room '%s' with ID '%s'", roomName, roomID)
		srv.sendSystemMessage(msg, "green", connection)

	case "/join":
		if len(parts) < 2 {
			srv.sendSystemMessage("Usage: /join <room-name>", "red", connection)
			return
		}
		roomName := strings.Join(parts[1:], " ")
		if err := srv.joinRoomByName(roomName, connection); err != nil {
			srv.sendSystemMessage(fmt.Sprintf("Error: %s", err.Error()), "red", connection)
		} else {
			srv.sendSystemMessage(fmt.Sprintf("Joined room '%s'", roomName), "green", connection)
		}

	case "/leave":
		if err := srv.leaveRoom(connection); err != nil {
			srv.sendSystemMessage(fmt.Sprintf("Error: %s", err.Error()), "red", connection)
		} else {
			srv.sendSystemMessage("Left room and returned to lobby", "green", connection)
		}

	case "/list":
		roomList := srv.getRoomList()
		if len(roomList) == 0 {
			srv.sendSystemMessage("No rooms available", "yellow", connection)
		} else {
			response := "Available rooms:\n"
			for _, room := range roomList {
				response += fmt.Sprintf("- %s (ID: %s, members: %d, created by: %s)\n", room.Name, room.ID, room.MemberCount, room.CreatedBy)
			}
			srv.sendSystemMessage(response, "yellow", connection)
		}

	default:
		srv.sendSystemMessage(fmt.Sprintf("Unknown command: %s. Available commands: /create, /join, /leave, /list", command), "red", connection)
	}
}

func (srv *WSServer) informUserOfAllCurrentUsers(newUser *ws.Connection) {
	connections := srv.currentConnections()
	users := make([]ws.WSMetadata, 0, len(connections))

	for _, c := range connections {
		users = append(users, ws.WSMetadata{Username: c.Metadata.Username, Color: c.Metadata.Color})
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
		Metadata: ws.WSMetadata{Username: newUser.Metadata.Username, Color: newUser.Metadata.Color},
	}
	jsonMsg := msg.Marshal()

	if err = newUser.WriteMessage(string(jsonMsg), websocket.TextMessage); err != nil {
		log.Println("Problem sending message to the client regarding the currently connected users")
	}
}

func (srv *WSServer) fanOutUserEnteredChat(username, color string) {
	connections := srv.currentConnections()

	msg := ws.WSMessage{
		Type:     types.MessageTypeUserEnteredChat,
		Value:    nil,
		Nonce:    nil,
		Metadata: ws.WSMetadata{Username: username, Color: color},
	}
	jsonMsg := msg.Marshal()
	for _, c := range connections {
		if err := c.WriteMessage(string(jsonMsg), websocket.TextMessage); err != nil {
			log.Printf("Error trying to inform the client %s that a new connection was made: %s\n", c.Metadata.Username, err.Error())
		}
	}
}
