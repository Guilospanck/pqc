package main

import (
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
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Error upgrading WS: ", err)
		return
	}
	defer conn.Close()

	username := GetRandomName()
	color := GetRandomColor()
	log.Printf("New connection: %s - %s\n", username, color)

	// Send the generated username and color to the WSClient
	w.Header().Add("username", string(username))
	w.Header().Add("color", string(color))

	connection := ws.Connection{Keys: cryptography.Keys{}, Conn: conn, Metadata: ws.WSMetadata{Username: username, Color: color}}

	srv.connections = append(srv.connections, &connection)

	for {
		msg, err := connection.ReadMessage()
		if err != nil {
			log.Printf("Error reading from conn: %s\n", err.Error())
			// Remove client from connections
			for i, v := range srv.connections {
				if v == &connection {
					srv.connections = append(srv.connections[:i], srv.connections[i+1:]...)
					srv.fanOutClientMessage(connection, []byte("Disconnected."))
					break
				}
			}

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

		srv.fanOutClientMessage(connection, decryptedMessageSent)

	}
}

func (srv *WSServer) fanOutClientMessage(client ws.Connection, decryptedMessage []byte) {
	for _, c := range srv.connections {
		if string(c.Keys.Public) == string(client.Keys.Public) {
			continue
		}

		msgWithPublicKey := fmt.Sprintf("%s: %s", client.Metadata.Username, string(decryptedMessage))

		log.Printf("Relaying message: \"%s\" from \"%s\" to client \"%s\"\n", msgWithPublicKey, string(client.Metadata.Username), c.Metadata.Username)
		c.RelayMessage(msgWithPublicKey, client.Metadata.Username, client.Metadata.Color)
	}
}
