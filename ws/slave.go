package ws

import (
	"github.com/gorilla/websocket"
	"log"
)

type Slave struct {
	ctx  *WSContext
	conn *websocket.Conn
	in   chan []byte
	out  chan []byte
}

func (s *Slave) run() {
	go s.write()
	go s.read()
}

func (s *Slave) read() {
	defer func() {
		s.ctx.unregister <- s
		s.conn.Close()
	}()

	for {
		t, message, err := s.conn.ReadMessage()
		if t != websocket.BinaryMessage {
			log.Println("Text message received, ignore")
			continue
		}

		if err != nil {
			log.Printf("ws read error: %v", err)
			break
		}

		s.onMessage(message)
	}
}

func (s *Slave) onMessage(msg []byte) {

}

func (s *Slave) writeData(buf []byte) {

}

func (s *Slave) write() {
	for {
		select {
		case b := <-s.in:
			err := s.conn.WriteMessage(websocket.BinaryMessage, b)
			if err != nil {
				log.Printf("failed to write message: %v\n")
			}
		}
	}
}
