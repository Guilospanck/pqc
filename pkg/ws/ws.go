package ws

import (
	"log"
	"pqc/pkg/cryptography"

	"github.com/gorilla/websocket"
)

type Connection struct {
	Keys *cryptography.Keys
	Conn *websocket.Conn
}

func (ws *Connection) WriteMessage(text string) error {
	err := ws.Conn.WriteMessage(websocket.TextMessage, []byte(text))
	if err != nil {
		log.Println("Write error:", err)
		return err
	}

	return nil
}

func (ws *Connection) ReadMessage() ([]byte, error) {
	_, msg, err := ws.Conn.ReadMessage()
	if err != nil {
		log.Println("Read error:", err)
		return []byte(""), err
	}

	return msg, nil
}
