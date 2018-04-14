package handlers

import (
	"CachedTickets/ticketdata"
	"CachedTickets/ws"
	"crypto/tls"
	"encoding/json"
	"errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"
)

type AppEnv struct {
	Db  ticketdata.TicketInfo
	Ctx *ws.WSContext
}

type urlChangeMsg struct {
	Status bool   `json:"status"`
	URL    string `json:"c_url"`
	Name   string `json:"c_name"`
}

type ticketsData struct {
	Flag   string            `json:"flag"`
	Map    map[string]string `json:"map"`
	Result []string          `json:"result"`
}

type leftTicketsJSON struct {
	HTTPStatus int         `json:"httpstatus"`
	Messages   string      `json:"messages"`
	Status     bool        `json:"status"`
	Data       ticketsData `json:"data"`
	UpdateTime int64       `json:"updatetime"`
}

type ticketPriceJSON struct {
	ValidateMessagesShowID string                 `json:"ValidateMessagesShowId"`
	Status                 bool                   `json:"status"`
	HTTPStatus             int                    `json:"httpstatus"`
	Data                   map[string]interface{} `json:"data"`
	Messages               []string               `json:"messages"`
	ValidateMessages       interface{}            `json:"validateMessages"`
	UpdateTime             int64                  `json:"updatetime"`
}

const (
	queryEntry string = "leftTicket/query"
	priceEntry string = "leftTicket/queryTicketPrice"
)

func getQueryParam(r *http.Request, name string) string {
	value, existed := r.Form[name]
	if !existed {
		return ""
	}

	return value[0]
}

func grab12306L(ch chan []byte, url string) []byte {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	var result []byte
	client := &http.Client{Transport: tr}
	resp, err := client.Get(url)
	if err != nil {
		log.Error("Failed to access url: ", err)
	} else {
		b, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			log.Error("Failed to read results from http response: ", err)
		}

		result = b
	}
	ch <- result
	return result
}

func (env *AppEnv) grab12306(ch chan []byte, url string) []byte {
	if env.Ctx != nil {
		// find a slave
		// if returned slave is nil, that means we are using master
		var ret []byte
		slave := env.Ctx.GetOneSlave()
		if slave != nil {
			result, err := slave.DoTask(url)
			if err != nil {
				ch <- nil
			} else if result != nil {
				ret = result.Result
				ch <- ret
			}
			return ret
		}
		log.Debug("master takes the task")
	}
	return grab12306L(ch, url)
}

func (env *AppEnv) UpdateCacheHandler(w http.ResponseWriter, r *http.Request) {
	log.Info("updateCacheHandler")
}

func (env *AppEnv) ShowWorkingHandler(w http.ResponseWriter, r *http.Request) {
	log.Info(w, "Cached Proxy Server is running!")
}

func (env *AppEnv) saveTicketPriceToDB(t *ticketdata.TicketPriceEntity, js *ticketPriceJSON) error {
	js.UpdateTime = time.Now().Unix()
	bs, err := json.Marshal(&js)
	if err != nil {
		log.Error("Severe problem! Unable to marshal json with update time field")
		return err
	}
	t.Content = string(bs)
	return env.Db.SaveTicketPrice(t)
}

func (env *AppEnv) getTicketPriceFromDB(w http.ResponseWriter, t *ticketdata.TicketPriceEntity) error {
	_, err := env.Db.GetTicketPrice(t)
	if err != nil {
		w.Write([]byte(`{"isempty":1}`))
	} else {
		w.Write([]byte(t.Content))
	}
	return err
}

func emptyTicketPriceJSON(js *ticketPriceJSON) bool {
	// Sometimes 12306 return valid json data contains no information
	emptyInfo := true
	for k := range js.Data {
		if k == "train_no" || k == "OT" {
			continue
		} else {
			emptyInfo = false
		}
	}
	return emptyInfo
}

func verifyTicketPrice(content *string) (*ticketPriceJSON, error) {
	var js ticketPriceJSON
	if *content == "" {
		return &js, errors.New("got empty string from server")
	}
	err := json.Unmarshal([]byte(*content), &js)
	if err != nil {
		return &js, err
	}
	if !js.Status {
		return &js, errors.New("returned ticket price json does not contain data,")
	}
	return &js, err
}

func (env *AppEnv) QueryTicketPriceHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	trainNo := getQueryParam(r, "train_no")
	from := getQueryParam(r, "from_station_no")
	to := getQueryParam(r, "to_station_no")
	seatType := getQueryParam(r, "seat_types")
	date := getQueryParam(r, "train_date")

	t := ticketdata.TicketPriceEntity{Id: 0, TrainNo: trainNo, FromStationNo: from, ToStationNo: to, SeatTypes: seatType, Content: "", UpdateTime: time.Now()}
	w.Header().Set("Content-Type", "application/json")
	if len(trainNo)*len(from)*len(to)*len(seatType)*len(date) == 0 {
		log.Warn("No enough params")
		w.Write([]byte("{}"))
	} else {
		url := "https://kyfw.12306.cn/otn/" + priceEntry + "?train_no=" + trainNo +
			"&from_station_no=" + from + "&to_station_no=" + to + "&seat_types=" + seatType +
			"&train_date=" + date

		ch := make(chan []byte)

		log.Debug("request ticket price -> " + url)

		go env.grab12306(ch, url)

		select {
		case b := <-ch:
			res := string(b)
			js, err := verifyTicketPrice(&res)
			if err != nil {
				env.getTicketPriceFromDB(w, &t)
			} else {
				w.Write([]byte(res))
				if !emptyTicketPriceJSON(js) {
					env.saveTicketPriceToDB(&t, js)
				}
			}
		case <-time.After(time.Second * 10):
			w.Write([]byte("{\"result\":\"timeout\"}"))
		}
	}
}

func verifyTickets(content *string) (*leftTicketsJSON, error) {
	var js leftTicketsJSON
	if *content == "" {
		return &js, errors.New("got empty string from server")
	}
	err := json.Unmarshal([]byte(*content), &js)
	if err != nil {
		return &js, err
	}
	if !js.Status || js.HTTPStatus != 200 {
		err = errors.New("returned ticket json does not contain data,")
	}
	return &js, err
}

func (env *AppEnv) saveTicketsToDB(t *ticketdata.TicketEntity, js *leftTicketsJSON) error {
	js.UpdateTime = time.Now().Unix()
	bs, err := json.Marshal(&js)
	if err != nil {
		log.Error("Severe problem! Unable to marshal json with update time field")
		return err
	}
	t.Content = string(bs)
	return env.Db.SaveLeftTickets(t)
}

func (env *AppEnv) getTicketsFromDB(w http.ResponseWriter, t *ticketdata.TicketEntity) error {
	_, err := env.Db.GetLeftTickets(t)
	if err != nil {
		log.Error("failed to get tickets from db: ", err)
		w.Write([]byte(`{"isempty":1}`))
	} else {
		w.Write([]byte(t.Content))
	}
	return err
}

func (env *AppEnv) QueryHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	date := getQueryParam(r, "leftTicketDTO.train_date")
	from := getQueryParam(r, "leftTicketDTO.from_station")
	to := getQueryParam(r, "leftTicketDTO.to_station")
	codes := getQueryParam(r, "purpose_codes")

	t := ticketdata.TicketEntity{Id: 0, From: from, To: to, Date: date, Content: "", UpdateTime: time.Now()}
	w.Header().Set("Content-Type", "application/json")
	if len(date)*len(from)*len(to)*len(codes) == 0 {
		log.Warn("no enough params")
		w.Write([]byte("Error, no enough params"))
	} else {
		url := "https://kyfw.12306.cn/otn/" + queryEntry + "?leftTicketDTO.train_date=" +
			date + "&leftTicketDTO.from_station=" + from + "&leftTicketDTO.to_station=" +
			to + "&purpose_codes=" + codes

		ch := make(chan []byte)

		log.Debug("request train info -> " + url)
		go env.grab12306(ch, url)

		select {
		case b := <-ch:
			res := string(b)
			msg := new(urlChangeMsg)
			err := json.Unmarshal([]byte(res), &msg)
			if err != nil {
				env.getTicketsFromDB(w, &t)
				return
			}
			if !msg.Status && len(msg.URL) > 0 {
				newurl := "https://kyfw.12306.cn/otn/" + msg.URL + "?leftTicketDTO.train_date=" +
					date + "&leftTicketDTO.from_station=" + from + "&leftTicketDTO.to_station=" +
					to + "&purpose_codes=" + codes

				newch := make(chan []byte)
				go env.grab12306(newch, newurl)
				select {
				case nb := <-newch:
					newres := string(nb)
					js, e := verifyTickets(&newres)
					if e != nil {
						env.getTicketsFromDB(w, &t)
					} else {
						w.Write([]byte(newres))
						env.saveTicketsToDB(&t, js)
					}
				case <-time.After(time.Second * 10):
					log.Warn("Timeout !")
					env.getTicketsFromDB(w, &t)
				}

			} else {
				js, e := verifyTickets(&res)
				if e != nil {
					env.getTicketsFromDB(w, &t)
				} else {
					w.Write([]byte(res))
					e = env.saveTicketsToDB(&t, js)
					if e != nil {
						log.Warn("failed to write data into db: ", e)
					}
				}
			}
		case <-time.After(time.Second * 10):
			log.Warn("Timeout !")
			env.getTicketsFromDB(w, &t)
		}
	}
}
