package ws

import (
	"github.com/gorilla/websocket"
	"log"
	"math/rand"
	"net/http"
	"time"
)

type WSContext struct {
	// registered slaves
	slaves map[*Slave]bool

	// slave slice: easy to randomize
	slaveList []*Slave

	// register req
	register chan *Slave

	// unregister req
	unregister chan *Slave

	// To signal the run goroutine to pick one slave
	one chan bool

	// picked slave
	picked chan *Slave
}

func NewWSContext() *WSContext {
	return &WSContext{
		slaves:     make(map[*Slave]bool),
		slaveList:  make([]*Slave, 20),
		register:   make(chan *Slave),
		unregister: make(chan *Slave),
		one:        make(chan bool),
		picked:     make(chan *Slave),
	}
}

func (w *WSContext) RandomRetrieve() *Slave {
	idx := rand.Intn(len(w.slaveList))
	return w.slaveList[idx]
}

func (w *WSContext) Run() {
	for {
		select {
		case s := <-w.register:
			if _, ok := w.slaves[s]; ok {
				log.Println("error: trying to register a registered slave")
			} else {
				w.slaves[s] = true
				w.slaveList = append(w.slaveList, s)
			}
		case s := <-w.unregister:
			if _, ok := w.slaves[s]; ok {
				delete(w.slaves, s)
				w.slaveList = make([]*Slave, 20)
				for key := range w.slaves {
					w.slaveList = append(w.slaveList, key)
				}
			}
		case <-w.one:
			w.picked <- w.RandomRetrieve()
		}
	}
}

func (w *WSContext) getOneSlave() (*Slave, error) {
	w.one <- true

	select {
	case s := <-w.picked:
		return s, nil
	case <-time.After(time.Second):
		return nil, nil
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func WSConnHandle(ctx *WSContext, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	slave := &Slave{
		ctx:         ctx,
		conn:        conn,
		in:          make(chan *writeJob),
		out:         make(chan *Message),
		toWrite:     make(chan *writeJob),
		pendingJobs: make(map[int64]*writeJob),
	}

	ctx.register <- slave

	go slave.run()
}
