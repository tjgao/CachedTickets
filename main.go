package main

import (
	"CachedTickets/ticketdata"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type Env struct {
	db ticketdata.TicketInfo
}

type URLChangeMsg struct {
	Status bool   `json:"status"`
	Url    string `json:"c_url"`
	Name   string `json:"c_name"`
}

func getQueryParam(r *http.Request, name string) string {
	value, existed := r.Form[name]
	if !existed {
		return ""
	}

	return value[0]
}

func grab12306(ch *chan string, url string) string {
	defer close(*ch)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	var result string
	client := &http.Client{Transport: tr}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("Failed to access url: %v\n", err)
	} else {
		b, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			log.Printf("Failed to read results from http response: %v\n", err)
		}

		result = string(b)
	}
	*ch <- result
	return result
}

func (env *Env) updateCacheHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("updateCacheHandler")
}

func (env *Env) showWorkingHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Cached Proxy Server is running!")
}

func (env *Env) queryTicketPriceHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	trainNo := getQueryParam(r, "train_no")
	from := getQueryParam(r, "from_station_no")
	to := getQueryParam(r, "to_station_no")
	seatType := getQueryParam(r, "seat_types")
	date := getQueryParam(r, "train_date")

	w.Header().Set("Content-Type", "application/json")
	if len(trainNo)*len(from)*len(to)*len(seatType)*len(date) == 0 {
		log.Printf("No enough params")
		w.Write([]byte("{}"))
	} else {
		url := "https://kyfw.12306.cn/otn/leftTicket/queryTicketPrice?train_no=" + trainNo +
			"&from_station_no=" + from + "&to_station_no=" + to + "&seat_types=" + seatType +
			"&train_date=" + date

		log.Printf(url)
		ch := make(chan string)

		go grab12306(&ch, url)

		select {
		case res := <-ch:
			w.Write([]byte(res))
		case <-time.After(time.Second * 10):
			w.Write([]byte("{\"result\":\"timeout\"}"))
		}
	}
}

func (env *Env) queryHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	date := getQueryParam(r, "leftTicketDTO.train_date")
	from := getQueryParam(r, "leftTicketDTO.from_station")
	to := getQueryParam(r, "leftTicketDTO.to_station")
	codes := getQueryParam(r, "purpose_codes")

	w.Header().Set("Content-Type", "application/json")
	if len(date)*len(from)*len(to)*len(codes) == 0 {
		w.Write([]byte("{}"))
	} else {
		url := "https://kyfw.12306.cn/otn/leftTicket/query?leftTicketDTO.train_date=" +
			date + "&leftTicketDTO.from_station=" + from + "&leftTicketDTO.to_station=" +
			to + "&purpose_codes=" + codes

		ch := make(chan string)

		go grab12306(&ch, url)

		select {
		case res := <-ch:
			msg := new(URLChangeMsg)
			err := json.Unmarshal([]byte(res), &msg)
			log.Printf(res)
			if err != nil {
				log.Printf("Cannot decode data: %v\n", err)
				w.Write([]byte(res))
				return
			}
			if !msg.Status && len(msg.Url) > 0 {
				newurl := "https://kyfw.12306.cn/otn/" + msg.Url + "?leftTicketDTO.train_date=" +
					date + "&leftTicketDTO.from_station=" + from + "&leftTicketDTO.to_station=" +
					to + "&purpose_codes=" + codes

				newch := make(chan string)
				go grab12306(&newch, newurl)
				select {
				case newres := <-newch:
					w.Write([]byte(newres))
				case <-time.After(time.Second * 10):
					w.Write([]byte("{\"result\":\"timeout\"}"))
				}

			} else {
				w.Write([]byte(res))
			}
		case <-time.After(time.Second * 10):
			w.Write([]byte("{\"result\":\"timeout\"}"))
		}
	}
}

func main() {
	port := flag.Int("p", 8086, "Port to serve on")
	flag.Parse()

	db, err := ticketdata.NewDB("postgres://dbuser:dbuser@localhost/ticket_cache")

	if err != nil {
		log.Panic(err)
	}

	env := &Env{db}

	r := mux.NewRouter()
	r.HandleFunc("/query", env.queryHandler)
	r.HandleFunc("/queryTicketPrice", env.queryTicketPriceHandler)
	r.HandleFunc("/", env.showWorkingHandler)
	r.HandleFunc("/update_cache", env.updateCacheHandler)
	http.Handle("/", r)

	log.Printf("Cached Proxy Server starts up, serving on port: %d", *port)
	err = http.ListenAndServe(":"+strconv.Itoa(*port), nil)

	if err != nil {
		log.Fatal("Fail to start server: ", err)
	}
}
