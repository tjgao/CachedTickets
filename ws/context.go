package ws

import (
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
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
		slaveList:  make([]*Slave, 0, 20),
		register:   make(chan *Slave),
		unregister: make(chan *Slave),
		one:        make(chan bool),
		picked:     make(chan *Slave),
	}
}

func (w *WSContext) randomRetrieve() *Slave {
	l := len(w.slaveList)
	idx := rand.Intn(l + 1)
	log.Debug("we have ", l, " slaves, random idx is ", idx)
	if idx == l {
		return nil
	}
	return w.slaveList[idx]
}

func (w *WSContext) Run() {
	rand.Seed(time.Now().UTC().UnixNano())
	for {
		select {
		case s := <-w.register:
			if _, ok := w.slaves[s]; ok {
				log.Error("error: trying to register a registered slave")
			} else {
				log.Info("Registered a slave server ", s.conn.RemoteAddr())
				w.slaves[s] = true
				w.slaveList = append(w.slaveList, s)
			}
		case s := <-w.unregister:
			if _, ok := w.slaves[s]; ok {
				log.Info("Unregistered a slave server ", s.conn.RemoteAddr())
				delete(w.slaves, s)
				w.slaveList = make([]*Slave, 0, 20)
				for key := range w.slaves {
					w.slaveList = append(w.slaveList, key)
				}
			}
		case <-w.one:
			w.picked <- w.randomRetrieve()
		}
	}
}

func (w *WSContext) GetOneSlave() *Slave {
	w.one <- true

	select {
	case s := <-w.picked:
		return s
	case <-time.After(time.Second * 3):
		return nil
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func WSConnHandle(ctx *WSContext, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("failed to upgrade protocol ", err)
		return
	}

	slave := newSlave(ctx, conn)
	ctx.register <- slave

	slave.run()
}
