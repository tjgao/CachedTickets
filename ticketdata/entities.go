package ticketdata

import (
	"log"
)

type TicketEntity struct {
	From    string `db:"from_station"`
	To      string `db:"to_station"`
	Date    string `db:"travel_date"`
	Content string `db:"content"`
}

type TicketPriceEntity struct {
	TrainNo       string `db:"train_no"`
	FromStationNo string `db:"from_station_no"`
	ToStationNo   string `db:"to_station_no"`
	SeatTypes     string `db:"seat_types"`
	Content       string `db:"content"`
	//	TrainDate     string `db:"train_date"`
}

func (db *DB) GetLeftTickets(from string, to string, date string) (string, error) {
	row, err := db.Query("select content from tickets where from_station = ? and to_station = ? and travel_date = ", from, to, date)
	var content string
	if err != nil {
		return content, err
	}
	defer row.Close()
	for row.Next() {
		if err := row.Scan(&content); err != nil {
			return content, err
		}
		break
	}
	return content, nil
}

func (db *DB) GetTicketPrice(train_no string, from_station_no string, to_station_no string, seat_types string) string {
	row, err := db.Query("select content from ticket_price where train_no = ? and from_station_no = ? and to_station_no = ? and seat_types = ?",
		train_no, from_station_no, to_station_no, seat_types)
	var content string
	if err != nil {
		return content, err
	}
	defer row.Close()
	for row.Next() {
		if err := row.Scan(&content); err != nil {
			return content, err
		}
		break
	}
	return content, nil
}
