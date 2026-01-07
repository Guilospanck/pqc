package main

import (
	"fmt"
	"log"
	"net/http"
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

	connection := ws.Connection{Keys: nil, Conn: conn}

	for {
		// // Send text
		// if err := connection.WriteMessage(text); err != nil {
		// 	break
		// }

		// Read response
		msg, err := connection.ReadMessage()
		if err != nil {
			log.Printf("Error reading from conn: %s\n", err.Error())
			return
		}

		fmt.Println("Client: ", string(msg))
	}
}
