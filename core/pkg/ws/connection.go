package ws

import (
	"context"
	"errors"
	"log"

	"github.com/Guilospanck/pqc/core/pkg/cryptography"
	"github.com/Guilospanck/pqc/core/pkg/types"
	"github.com/Guilospanck/pqc/core/pkg/utils"

	"github.com/gorilla/websocket"
)

type WriteMessageRequest struct {
	msgType int // websocket.TextMessage, websocket.PingMessage
	text    []byte
	err     chan error
}

type ClientId string

type Connection struct {
	ID       ClientId
	Keys     cryptography.Keys
	Conn     *websocket.Conn
	Metadata WSMetadata

	WriteMessageReq chan WriteMessageRequest

	WriteLoopReady  chan struct{}
	WriteLoopClosed chan struct{}

	KeysExchanged chan struct{}
}

func NewEmptyConnection() Connection {
	return Connection{
		ID:       ClientId(utils.UUID()),
		Keys:     cryptography.Keys{},
		Conn:     nil,
		Metadata: WSMetadata{},

		WriteMessageReq: make(chan WriteMessageRequest, 10),

		WriteLoopReady:  make(chan struct{}),
		WriteLoopClosed: make(chan struct{}),

		KeysExchanged: make(chan struct{}),
	}
}

func (ws *Connection) ResetChannels() {
	ws.WriteMessageReq = make(chan WriteMessageRequest, 10)
	ws.WriteLoopReady = make(chan struct{})
	ws.WriteLoopClosed = make(chan struct{})
	ws.KeysExchanged = make(chan struct{})
}

func (ws *Connection) WriteLoop(ctx context.Context) {
	log.Println("Starting WRITE loop...")

	close(ws.WriteLoopReady)
	defer close(ws.WriteLoopClosed)

	for {
		select {
		case msg := <-ws.WriteMessageReq:

			text := msg.text
			msgType := msg.msgType

			err := ws.Conn.WriteMessage(msgType, text)

			select {
			case msg.err <- err:
			case <-ctx.Done():
				log.Println("Context cancelled while selecting the write message result. Returning from WRITE loop.")
				return
			}

			if err != nil {
				log.Println("Error while writing message. Returning from WRITE loop")
				return
			}

		case <-ctx.Done():
			log.Println("Context cancelled. Returning from WRITE loop.")
			return
		}
	}
}

// Understanding the channels/select here:
// 1. Can I hand the letter to the courier?
// 2. Will the courier ever reply?
func (ws *Connection) WriteMessage(text string, msgType int) error {
	errCh := make(chan error, 1)

	req := WriteMessageRequest{
		msgType: msgType,
		text:    []byte(text),
		err:     errCh,
	}

	select {
	case ws.WriteMessageReq <- req:
	case <-ws.WriteLoopClosed:
		return errors.New("connection closed")
	}

	select {
	case err := <-errCh:
		return err
	case <-ws.WriteLoopClosed:
		return errors.New("connection closed")
	}
}

func (ws *Connection) ReadMessage() ([]byte, error) {
	_, msg, err := ws.Conn.ReadMessage()
	return msg, err
}

func (connection *Connection) RelayMessage(message, fromUsername, fromColor string) {
	nonce, ciphertext, err := cryptography.EncryptMessage(connection.Keys.SharedSecret, []byte(message))
	if err != nil {
		log.Printf("Could not encrypt message: %s\n", err.Error())
		return
	}

	msg := WSMessage{
		Type:     types.MessageTypeEncryptedMessage,
		Value:    ciphertext,
		Nonce:    nonce,
		Metadata: WSMetadata{Username: fromUsername, Color: fromColor},
	}
	jsonMsg := msg.Marshal()

	// send encrypted message
	if err := connection.WriteMessage(string(jsonMsg), websocket.TextMessage); err != nil {
		log.Printf("Could not send message to client: %s\n", err.Error())
	}

}
