package ws

import (
	"errors"
	"github.com/gorilla/websocket"
	"log"
	"time"
)

type writeJob struct {
	data    *Message
	resp    chan *Message
	transID int64
}

type Slave struct {
	ctx         *WSContext
	conn        *websocket.Conn
	in          chan *writeJob
	out         chan *Message
	toWrite     chan *writeJob
	pendingJobs map[int64]*writeJob
	nextTransID int64
}

func newSlave(w *WSContext, c *websocket.Conn) *Slave {
	return &Slave{
		ctx:         w,
		conn:        c,
		in:          make(chan *writeJob),
		out:         make(chan *Message),
		toWrite:     make(chan *writeJob),
		pendingJobs: make(map[int64]*writeJob),
		nextTransID: 0,
	}
}

func (s *Slave) run() {
	go s.bridge()
	go s.write()
	go s.read()
}

func (s *Slave) bridge() {
	pendingJobs := make(map[int64]*writeJob)
	for {
		select {
		case job := <-s.in:
			job.transID = s.getNextTransID()
			if _, ok := pendingJobs[job.transID]; ok {
				panic("We already have this ID in pending jobs, but this cannot happen!")
			} else {
				pendingJobs[job.transID] = job
				s.toWrite <- job
			}
		case dataResp := <-s.out:
			if job, ok := pendingJobs[dataResp.TransID]; ok {
				job.resp <- dataResp
				delete(pendingJobs, dataResp.TransID)
			}
		}
	}
}

func (s *Slave) write() {
	for {
		select {
		case job := <-s.toWrite:
			b, err := Encode(job.data)
			if err != nil {
				panic("failed to encode write data")
			} else {
				err := s.conn.WriteMessage(websocket.BinaryMessage, b)
				if err != nil {
					log.Printf("failed to write message: %v\n")
				}
			}
		}
	}
}

func (s *Slave) read() {
	defer func() {
		s.ctx.unregister <- s
		s.conn.Close()
	}()

	for {
		t, data, err := s.conn.ReadMessage()
		if t != websocket.BinaryMessage {
			log.Println("Text message received, ignore")
			continue
		}

		if err != nil {
			log.Printf("ws read error: %v", err)
			break
		}

		s.onMessage(data)
	}
}

func (s *Slave) getNextTransID() int64 {
	s.nextTransID++
	return s.nextTransID
}

func (s *Slave) onMessage(data []byte) {
	var m Message
	err := Decode(data, &m)
	if err != nil {
		log.Printf("failed to decode message: %v", err)
	} else {
		s.out <- &m
	}
}

func (s *Slave) writeData(m *Message) (*Message, error) {
	job := writeJob{
		data: m,
		resp: make(chan *Message),
	}

	go func() {
		s.in <- &job
	}()

	select {
	case msg := <-job.resp:
		return msg, nil
	case <-time.After(6 * time.Second):
		return nil, errors.New("timeout while waiting for response")
	}
}

func (s *Slave) DoTask(url string) (*TaskResult, error) {
	t := Task{
		TargetURL: url,
	}

	b, err := EncodeTask(&t)
	if err != nil {
		panic("failed to encode task")
	}

	m := Message{
		ID:   TaskRequestType,
		Body: b,
	}

	resp, e := s.writeData(&m)
	if e != nil {
		return nil, e
	}

	if resp.ID != TaskResultType {
		return nil, errors.New("Task result does not contain correct ID")
	}

	var tr TaskResult
	e = DecodeTaskResult(resp.Body, &tr)
	if e != nil {
		return nil, e
	}
	return &tr, nil
}
