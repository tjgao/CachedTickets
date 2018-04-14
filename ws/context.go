package ws

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"net/http"
	"sort"
	"time"
)

type WSContext struct {
	// registered slaves
	slaves map[*Slave]bool

	// slave slice: easy to randomize
	slaveList SlaveSlice

	// register req
	register chan *Slave

	// unregister req
	unregister chan *Slave

	// To signal the run goroutine to pick one slave
	one chan bool

	// picked slave
	picked chan *Slave

	// whether master takes requests
	masterWork bool
}

func NewWSContext(mw bool) *WSContext {
	return &WSContext{
		slaves:     make(map[*Slave]bool),
		slaveList:  make([]*Slave, 0, 20),
		register:   make(chan *Slave),
		unregister: make(chan *Slave),
		one:        make(chan bool),
		picked:     make(chan *Slave),
		masterWork: mw,
	}
}

func (w *WSContext) randomRetrieve() *Slave {
	l := len(w.slaveList)
	if l == 0 {
		return nil
	}
	m := l
	if w.masterWork {
		m++
	}
	idx := rand.Intn(m)
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
				s.status.Addr = s.conn.RemoteAddr().String()
				w.slaves[s] = true
				w.slaveList = append(w.slaveList, s)
				sort.Sort(w.slaveList)
			}
		case s := <-w.unregister:
			if _, ok := w.slaves[s]; ok {
				log.Info("Unregistered a slave server ", s.conn.RemoteAddr())
				delete(w.slaves, s)
				w.slaveList = make([]*Slave, 0, 20)
				for key := range w.slaves {
					w.slaveList = append(w.slaveList, key)
				}
				sort.Sort(w.slaveList)
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

func WSStatusHandle(ctx *WSContext, w http.ResponseWriter, r *http.Request) {
	var data []*SlaveStatus
	for _, slave := range ctx.slaveList {
		data = append(data, &(slave.status))
	}

	bts, err := json.Marshal(&data)
	if err != nil {
		log.Error("failed to marshal a json object, err: ", err)
		w.Write([]byte("{}"))
	} else {
		w.Write(bts)
	}
}
