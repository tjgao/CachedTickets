package main

import (
	"CachedTickets/handlers"
	"CachedTickets/ticketdata"
	"CachedTickets/ws"
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
)

func main() {
	port := flag.Int("p", 8086, "Port to serve on")
	logfile := flag.String("f", "", "Log file path")
	slaveSupport := flag.Bool("s", false, "Turn on slave mode")

	flag.Parse()

	if *logfile != "" {
		f, err := os.OpenFile(*logfile, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			log.Fatal("Failed to open log file ", logfile)
		}
		log.SetOutput(f)
	}

	db, err := ticketdata.NewDB("postgres://dbuser:dbuser@localhost/ticket_cache")
	if err != nil {
		db, err = ticketdata.NewDB("user=dbuser password=dbuser dbname=ticket_cache sslmode=disable")
	}

	if err != nil {
		log.Panic(err)
	}

	var ctx *ws.WSContext

	if *slaveSupport {
		ctx = ws.NewWSContext()
		go ctx.Run()
	}
	env := &handlers.AppEnv{
		Db:  db,
		Ctx: ctx,
	}

	r := mux.NewRouter()
	r.HandleFunc("/query", env.QueryHandler)
	r.HandleFunc("/queryTicketPrice", env.QueryTicketPriceHandler)
	r.HandleFunc("/", env.ShowWorkingHandler)
	r.HandleFunc("/update_cache", env.UpdateCacheHandler)
	if *slaveSupport {
		r.HandleFunc("/ws/register", func(w http.ResponseWriter, r *http.Request) {
			ws.WSConnHandle(ctx, w, r)
		})
	}
	http.Handle("/", r)

	log.Printf("Cached Proxy Server starts up, serving on port: %d", *port)
	err = http.ListenAndServe(":"+strconv.Itoa(*port), nil)

	if err != nil {
		log.Fatal("Fail to start server: ", err)
	}
}
