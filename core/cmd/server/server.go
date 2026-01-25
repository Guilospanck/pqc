package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"github.com/Guilospanck/pqc/core/pkg/ws"
	"slices"
	"sync"

	"github.com/gorilla/websocket"
)

type clientId string

type WSServer struct {
	// TODO: create concept of rooms
	connections   map[clientId]*ws.Connection
	usedUsernames []string
	mu            sync.RWMutex
	ctx           context.Context
}

func NewServer(ctx context.Context) *WSServer {
	return &WSServer{
		connections:   make(map[clientId]*ws.Connection),
		ctx:           ctx,
		usedUsernames: make([]string, 0),
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

	log.Printf("New connection: %s - %s\n", username, color)

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

		if msgJson.Type != ws.EncryptedMessage || decryptedMessageSent == nil {
			continue
		}

		srv.fanOutUserMessage(connection, decryptedMessageSent)
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
			Type:     ws.UserLeft,
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

		break
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
		Type:     ws.CurrentUsers,
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
		Type:     ws.UserEntered,
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

func (srv *WSServer) fanOutUserMessage(client *ws.Connection, decryptedMessage []byte) {
	connections := srv.currentConnections()

	for _, c := range connections {
		if string(c.Keys.Public) == string(client.Keys.Public) {
			continue
		}

		msgWithPublicKey := fmt.Sprintf("%s: %s", client.Metadata.Username, string(decryptedMessage))

		log.Printf("Relaying message: \"%s\" from \"%s\" to client \"%s\"\n", msgWithPublicKey, client.Metadata.Username, c.Metadata.Username)
		c.RelayMessage(msgWithPublicKey, client.Metadata.Username, client.Metadata.Color)
	}
}
