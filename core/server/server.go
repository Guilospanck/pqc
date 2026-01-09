package server

import (
	"log"
	"net/http"
	"pqc/pkg/cryptography"
	"pqc/pkg/ws"

	"github.com/gorilla/websocket"
)

func startServer() {
	http.HandleFunc("/ws", wsHandler)

	log.Print("WS server started at localhost:8080/ws")
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
		log.Print("Error upgrading WS: ", err)
		return
	}
	defer conn.Close()

	log.Print("New connection!")

	connection := ws.Connection{Keys: cryptography.Keys{}, Conn: conn}

	for {
		msg, err := connection.ReadMessage()
		if err != nil {
			log.Printf("Error reading from conn: %s\n", err.Error())
			return
		}

		msgJson, err := ws.UnmarshalWSMessage(msg)
		if err != nil {
			log.Printf("Error unmarshalling message: %s\n", err.Error())
			continue
		}
		msgJson.HandleClientMessage(&connection)
	}
}
