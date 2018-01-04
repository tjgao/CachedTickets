package ticketdata

import (
	"log"
	"time"
)

type TicketEntity struct {
	Id         int64     `db:"id"`
	From       string    `db:"from_station"`
	To         string    `db:"to_station"`
	Date       string    `db:"travel_date"`
	Content    string    `db:"content"`
	UpdateTime time.Time `db:"update_time"`
}

type TicketPriceEntity struct {
	Id            int64     `db:"id"`
	TrainNo       string    `db:"train_no"`
	FromStationNo string    `db:"from_station_no"`
	ToStationNo   string    `db:"to_station_no"`
	SeatTypes     string    `db:"seat_types"`
	Content       string    `db:"content"`
	UpdateTime    time.Time `db:"update_time"`
	//	TrainDate     string `db:"train_date"`
}

func (db *DB) GetLeftTickets(t *TicketEntity) (*TicketEntity, error) {
	stmt, err := db.Prepare("select * from tickets where from_station = $1 and to_station = $2 and travel_date = $3")
	defer stmt.Close()

	if err != nil {
		log.Printf("%v", err)
		return t, err
	}
	err = stmt.QueryRow(t.From, t.To, t.Date).Scan(t)

	return t, err
}

// insert or update, depending on the availability of this particular entry.
func (db *DB) SaveLeftTickets(t *TicketEntity) error {
	stmt, err := db.Prepare("insert into tickets (from_station, to_station, travel_date, content, update_time) values ($1, $2, $3, $4, now())")
	defer stmt.Close()
	if err != nil {
		log.Printf("%v", err)
		return err
	}

	_, err = stmt.Exec(t.From, t.To, t.Date, t.Content)
	if err != nil {
		// we try to update it
		updateStmt, err := db.Prepare("update tickets set content = $1, update_time = now() where from_station = $2 and to_station = $3 and travel_date = $4")
		defer updateStmt.Close()

		if err != nil {
			log.Printf("%v", err)
			return err
		}

		_, err = updateStmt.Exec(t.Content, t.From, t.To, t.Date)
		return err
	}
	return nil
}

func (db *DB) GetTicketPrice(t *TicketPriceEntity) (*TicketPriceEntity, error) {
	stmt, err := db.Prepare("select * from ticket_price where train_no = $1 and from_station_no = $2 and to_station_no = $3 and seat_types = $4")
	defer stmt.Close()

	if err != nil {
		log.Printf("%v", err)
		return t, err
	}

	err = stmt.QueryRow(t.TrainNo, t.FromStationNo, t.ToStationNo, t.SeatTypes).Scan(t)

	return t, err
}

// insert or update
func (db *DB) SaveTicketPrice(t *TicketPriceEntity) error {
	stmt, err := db.Prepare("insert into ticket_price (train_no, from_station_no, to_station_no, seat_types, content, update_time) values ($1, $2, $3, $4, $5, now())")
	defer stmt.Close()
	if err != nil {
		log.Printf("%v", err)
		return err
	}

	_, err = stmt.Exec(t.TrainNo, t.FromStationNo, t.ToStationNo, t.SeatTypes, t.Content)
	if err != nil {
		// try update
		updateStmt, err := db.Prepare("update ticket_price set content = $1, update_time = now() where train_no = $2 and from_station_no = $3 and to_station_no = $4 and seat_types = $5")
		defer updateStmt.Close()

		if err != nil {
			log.Printf("%v", err)
			return err
		}

		_, err = updateStmt.Exec(t.Content, t.TrainNo, t.FromStationNo, t.ToStationNo, t.SeatTypes)
		return err
	}
	return nil
}
