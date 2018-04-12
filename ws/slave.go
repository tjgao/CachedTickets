package ws

import (
	"errors"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
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
	exit        chan struct{}
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
		exit:        make(chan struct{}),
	}
}

func (s *Slave) run() {
	go s.bridge()
	go s.write()
	go s.read()
}

func (s *Slave) bridge() {
	pendingJobs := make(map[int64]*writeJob)
	log.Debug("bridge coroutine for ", s.conn.RemoteAddr(), " is running")
OUTSIDE:
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
		case <-s.exit:
			break OUTSIDE
		}
	}
	log.Debug("bridge coroutine for ", s.conn.RemoteAddr(), " exited")
}

func (s *Slave) write() {
	log.Debug("write coroutine for ", s.conn.RemoteAddr(), " is running")
OUTSIDE:
	for {
		select {
		case job := <-s.toWrite:
			job.data.TransID = job.transID
			b, err := Encode(job.data)
			if err != nil {
				panic("failed to encode write data")
			} else {
				err := s.conn.WriteMessage(websocket.BinaryMessage, b)
				if err != nil {
					log.Printf("failed to write message: %v\n")
				}
			}
		case <-s.exit:
			break OUTSIDE
		}
	}
	log.Debug("write coroutine for ", s.conn.RemoteAddr(), " exited")
}

func (s *Slave) read() {
	log.Debug("read coroutine for ", s.conn.RemoteAddr(), " is running")
	defer func() {
		s.ctx.unregister <- s
		s.conn.Close()
		close(s.exit)
	}()

	for {
		t, data, err := s.conn.ReadMessage()
		if t == websocket.BinaryMessage {
			s.onMessage(data)
		}

		if err != nil {
			log.Error("ws read error: %v", err)
			break
		}
	}
	log.Debug("read coroutine for ", s.conn.RemoteAddr(), " exited")
}

func (s *Slave) getNextTransID() int64 {
	s.nextTransID++
	return s.nextTransID
}

func (s *Slave) onMessage(data []byte) {
	var m Message
	err := Decode(data, &m)
	if err != nil {
		log.Error("failed to decode message: ", err)
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
	case <-time.After(10 * time.Second):
		return nil, errors.New("timeout while waiting for response")
	}
}

func (s *Slave) DoTask(url string) (*TaskResult, error) {
	t := Task{
		TargetURL: url,
	}

	b, err := EncodeTask(&t)
	if err != nil {
		log.Panic("failed to encode task")
	}

	m := Message{
		ID:   TaskRequestType,
		Body: b,
	}

	resp, e := s.writeData(&m)
	if e != nil {
		log.Error("failed to write data: ", err)
		return nil, e
	}

	if resp.ID != TaskResultType {
		return nil, errors.New("Task result does not contain correct ID")
	}

	var tr TaskResult
	e = DecodeTaskResult(resp.Body, &tr)
	if e != nil {
		log.Panic("failed to decode task result")
	}
	return &tr, nil
}
