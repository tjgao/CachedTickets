package ws

import (
	"github.com/gorilla/websocket"
	"log"
)

type writeJob struct {
	data chan []byte
	done chan struct{}
}

type Slave struct {
	ctx         *WSContext
	conn        *websocket.Conn
	in          chan *writeJob
	out         chan []byte
	pendingJobs map[int]*writeJob
}

func newSlave(w *WSContext, c *websocket.Conn) *Slave {
	return &Slave{
		ctx:         w,
		conn:        c,
		in:          make(chan *writeJob),
		out:         make(chan []byte),
		pendingJobs: make(map[int]*writeJob),
	}
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
		case job := <-s.in:
			err := s.conn.WriteMessage(websocket.BinaryMessage, job.data)
			if err != nil {
				log.Printf("failed to write message: %v\n")
			}
		}
	}
}
