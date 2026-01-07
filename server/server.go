package main

import (
	"fmt"
	"log"
	"net/http"
	"pqc/pkg/cryptography"
	"pqc/pkg/ws"

	"github.com/gorilla/websocket"
)

func NewWSServer() {
	http.HandleFunc("/ws", wsHandler)

	fmt.Println("WS server started at :8080/ws")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

var upgrader = websocket.Upgrader{
	// INFO: for production you should make this more restrictive
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading WS: ", err)
		return
	}
	defer conn.Close()

	fmt.Println("New connection!")

	connection := ws.Connection{Keys: cryptography.Keys{}, Conn: conn}

	for {
		msg, err := connection.ReadMessage()
		if err != nil {
			log.Printf("Error reading from conn: %s\n", err.Error())
			return
		}

		msgJson := ws.UnmarshalClientToServerMessage(msg)
		handleMessage(&connection, msgJson)
	}
}

func handleMessage(connection *ws.Connection, msg ws.ClientToServerMessage) {
	switch msg.Type {
	case ws.ExchangeKeys:
		// Encapsulate ciphertext with the public key from client
		// And generates a sharedSecret
		sharedSecret, cipherText := cryptography.KeyExchange(msg.Value)

		// save sharedSecret
		connection.Keys.SharedSecret = sharedSecret

		// send ciphertext to client so we can exchange keys
		if err := connection.WriteMessage(string(cipherText)); err != nil {
			log.Fatal("Could not send message to client: ", err)
		}

	case ws.EncryptedMessage:
		log.Printf("Received encrypted message: %s", msg.Value)
	default:
		log.Fatal("Received a message with an unknown type")
	}
}
