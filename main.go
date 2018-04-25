package main

import (
	"CachedTickets/handlers"
	"CachedTickets/ticketdata"
	"CachedTickets/ws"
	"flag"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
)

func main() {
	logLevelTable := map[string]log.Level{
		"panic": log.PanicLevel,
		"error": log.ErrorLevel,
		"warn":  log.WarnLevel,
		"info":  log.InfoLevel,
		"debug": log.DebugLevel,
	}

	port := flag.Int("p", 8086, "Port to serve on")
	logfile := flag.String("f", "", "Log file path")
	slaveSupport := flag.Bool("s", false, "Turn on slave mode")
	masterWork := flag.Bool("m", false, "whether master server works on requests")
	logLevel := flag.String("l", "info", "specify log level, available levels are: panic, error, warn, info and debug")

	flag.Parse()

	if level, ok := logLevelTable[*logLevel]; ok {
		log.SetLevel(level)
	} else {
		log.Warn("unrecognized log level specified, use warn level instead")
	}

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
		ctx = ws.NewWSContext(*masterWork)
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
	r.HandleFunc("/config/current_api", env.Current12306APIHandler)
	r.HandleFunc("/config/update_api", env.Update12306APIHandler)

	if *slaveSupport {
		r.HandleFunc("/ws/register", func(w http.ResponseWriter, r *http.Request) {
			ws.WSConnHandle(ctx, w, r)
		})
		r.HandleFunc("/ws/status", func(w http.ResponseWriter, r *http.Request) {
			ws.WSStatusHandle(ctx, w, r)
		})
		http.Handle("/ws/info/", http.StripPrefix("/ws/info/", http.FileServer(http.Dir("./ws_info/"))))
	}
	http.Handle("/config", http.StripPrefix("/config", http.FileServer(http.Dir("./config"))))
	http.Handle("/", r)

	log.Info("Cached Proxy Server starts up, serving on port: ", *port)
	err = http.ListenAndServe(":"+strconv.Itoa(*port), nil)

	if err != nil {
		log.Fatal("Fail to start server: ", err)
	}
}
