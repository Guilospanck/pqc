package main

import (
	"fmt"
	"log"
	"net/http"
	"pqc/pkg/cryptography"
	"pqc/pkg/ws"

	"github.com/gorilla/websocket"
)

type Server struct {
	connections []*ws.Connection
}

func (srv *Server) startServer() {
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

func (srv *Server) wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Error upgrading WS: ", err)
		return
	}
	defer conn.Close()

	log.Print("New connection!")

	connection := ws.Connection{Keys: cryptography.Keys{}, Conn: conn}

	srv.connections = append(srv.connections, &connection)

	for {
		msg, err := connection.ReadMessage()
		if err != nil {
			log.Printf("Error reading from conn: %s\n", err.Error())
			// Remove client from connections
			for i, v := range srv.connections {
				if v == &connection {
					srv.connections = append(srv.connections[:i], srv.connections[i+1:]...)
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

		for _, c := range srv.connections {
			if string(c.Keys.Public) == string(connection.Keys.Public) {
				continue
			}

			// Add the first 7 bytes of the PK of who (client) that sent the message
			msgWithPublicKey := fmt.Sprintf("(%s) %s", connection.Keys.Public[:7], string(decryptedMessageSent))

			c.RelayMessage(msgWithPublicKey)
		}
	}
}
