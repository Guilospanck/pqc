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
	"github.com/Guilospanck/pqc/core/pkg/ws"

	"github.com/gorilla/websocket"
)

type WSServer struct {
	rooms         map[ws.RoomId]*ws.Room
	connections   map[ws.ClientId]*ws.Connection
	usedUsernames []string
	mu            sync.RWMutex
	ctx           context.Context
}

func NewServer(ctx context.Context) *WSServer {
	// create lobby room
	rooms := make(map[ws.RoomId]*ws.Room)
	lobbyRoom := ws.NewRoom(ws.ClientId(SYSTEM), LOBBY_ROOM)
	rooms[lobbyRoom.ID] = &lobbyRoom

	return &WSServer{
		rooms:         rooms,
		connections:   make(map[ws.ClientId]*ws.Connection),
		ctx:           ctx,
		usedUsernames: make([]string, 0),
	}
}

func (srv *WSServer) addConnection(connection *ws.Connection) {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	srv.connections[ws.ClientId(connection.Metadata.Username)] = connection
}

func (srv *WSServer) removeConnection(id ws.ClientId) {
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

		decryptedMessageSent := srv.handleClientMessage(msgJson, connection)

		if msgJson.Type != types.MessageTypeEncryptedMessage || decryptedMessageSent == nil {
			continue
		}

		srv.fanOutUserMessage(connection, decryptedMessageSent)
	}
}

func (srv *WSServer) handleClientMessage(msg ws.WSMessage, connection *ws.Connection) []byte {
	switch msg.Type {
	case types.MessageTypeExchangeKeys:
		// Encapsulate ciphertext with the public key from client
		// and generates a sharedSecret
		sharedSecret, cipherText := cryptography.KeyExchange(msg.Value)

		// save the HKDF'ed sharedSecret
		connection.Keys.SharedSecret = cryptography.DeriveKey(sharedSecret)
		connection.Keys.Public = msg.Value

		msg := ws.WSMessage{
			Type:     types.MessageTypeExchangeKeys,
			Value:    cipherText,
			Nonce:    nil,
			Metadata: ws.WSMetadata{Username: connection.Metadata.Username, Color: connection.Metadata.Color},
		}
		jsonMsg := msg.Marshal()

		// send ciphertext to client so we can exchange keys
		if err := connection.WriteMessage(string(jsonMsg), websocket.TextMessage); err != nil {
			log.Printf("Could not send message to client: %s\n", err.Error())
			return nil
		}

	case types.MessageTypeEncryptedMessage:
		nonce := msg.Nonce
		ciphertext := msg.Value

		log.Printf("Received encrypted message: >>> %s <<<, with nonce: >>> %s <<<\n", ciphertext, nonce)
		decrypted, err := cryptography.DecryptMessage(connection.Keys.SharedSecret, nonce, ciphertext)
		if err != nil {
			log.Printf("Could not decrypt message from client (%s): %s\n", connection.Metadata.Username, err.Error())
			return nil
		}
		log.Printf("Decrypted message (%s): \"%s\"\n", connection.Metadata.Username, decrypted)

		return decrypted

	default:
		log.Printf("Received a message with an unknown type: %s\n", msg.Type)
	}

	return nil

}

// Remove client from connections and broadcast user left event
func (srv *WSServer) userDisconnected(connection *ws.Connection) {
	connections := srv.currentConnections()

	for _, v := range connections {
		if v.Metadata.Username != connection.Metadata.Username {
			continue
		}

		srv.removeConnection(ws.ClientId(v.Metadata.Username))

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
