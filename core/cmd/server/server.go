package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"pqc/pkg/cryptography"
	"pqc/pkg/ws"

	"github.com/gorilla/websocket"
)

type WSServer struct {
	connections []*ws.Connection
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
	username := GetRandomName()
	color := GetRandomColor()

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

	log.Printf("New connection: %s - %s\n", username, color)

	connection := ws.Connection{Keys: cryptography.Keys{}, Conn: conn, Metadata: ws.WSMetadata{Username: username, Color: color}}

	// Update this newly connected user with info regarding all connected users
	srv.informNewUserOfAllCurrentUsers(&connection)

	// Send to other clients the event of a newly connected client
	srv.fanOutUserEnteredChat(username, color)

	srv.connections = append(srv.connections, &connection)

	for {
		msg, err := connection.ReadMessage()
		if err != nil {
			log.Printf("Error reading from conn: %s\n", err.Error())
			srv.userDisconnected(&connection)
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
	for i, v := range srv.connections {
		if v != connection {
			continue
		}

		srv.connections = append(srv.connections[:i], srv.connections[i+1:]...)

		// Broadcast user left event to other clients
		leftMsg := ws.WSMessage{
			Type:     ws.UserLeft,
			Value:    nil,
			Nonce:    nil,
			Metadata: ws.WSMetadata{Username: connection.Metadata.Username, Color: connection.Metadata.Color},
		}
		leftJsonMsg := leftMsg.Marshal()
		for _, c := range srv.connections {
			if err := c.WriteMessage(string(leftJsonMsg)); err != nil {
				log.Printf("Error trying to inform clients that user left: %s\n", err.Error())
			}
		}

		break
	}
}

func (srv *WSServer) informNewUserOfAllCurrentUsers(newUser *ws.Connection) {
	users := []ws.WSMetadata{}
	for _, c := range srv.connections {
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

	if err = newUser.WriteMessage(string(jsonMsg)); err != nil {
		log.Println("Problem sending message to the client regarding the currently connected users")
	}
}

func (srv *WSServer) fanOutUserEnteredChat(username, color string) {
	msg := ws.WSMessage{
		Type:     ws.UserEntered,
		Value:    nil,
		Nonce:    nil,
		Metadata: ws.WSMetadata{Username: username, Color: color},
	}
	jsonMsg := msg.Marshal()
	for _, c := range srv.connections {
		if err := c.WriteMessage(string(jsonMsg)); err != nil {
			log.Printf("Error trying to inform the client %s that a new connection was made: %s\n", c.Metadata.Username, err.Error())
		}
	}
}

func (srv *WSServer) fanOutUserMessage(client ws.Connection, decryptedMessage []byte) {
	for _, c := range srv.connections {
		if string(c.Keys.Public) == string(client.Keys.Public) {
			continue
		}

		msgWithPublicKey := fmt.Sprintf("%s: %s", client.Metadata.Username, string(decryptedMessage))

		log.Printf("Relaying message: \"%s\" from \"%s\" to client \"%s\"\n", msgWithPublicKey, client.Metadata.Username, c.Metadata.Username)
		c.RelayMessage(msgWithPublicKey, client.Metadata.Username, client.Metadata.Color)
	}
}
