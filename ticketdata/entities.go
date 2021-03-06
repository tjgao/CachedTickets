package ticketdata

import (
	log "github.com/sirupsen/logrus"
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

func (db *DB) GetAlternativeTickets(t *TicketEntity) (*TicketEntity, error) {
	stmt, err := db.Prepare("select * from tickets where id=(select max(id) as mid from tickets where from_station = $1 and to_station = $2)")
	defer stmt.Close()

	if err != nil {
		log.Error("failed to prepare sql statement: ", err, stmt)
		return t, err
	}

	err = stmt.QueryRow(t.From, t.To).
		Scan(&t.Id, &t.From, &t.To, &t.Date, &t.Content, &t.UpdateTime)

	return t, err
}

func (db *DB) GetLeftTickets(t *TicketEntity) (*TicketEntity, error) {
	stmt, err := db.Prepare("select * from tickets where from_station = $1 and to_station = $2 and travel_date = $3")
	defer stmt.Close()

	if err != nil {
		log.Error("failed to prepare sql statement: ", err, stmt)
		return t, err
	}
	err = stmt.QueryRow(t.From, t.To, t.Date).
		Scan(&t.Id, &t.From, &t.To, &t.Date, &t.Content, &t.UpdateTime)

	if err != nil {
		// It is possible we do not have the correct data. We just try to find anything we have or good enough
		return db.GetAlternativeTickets(t)
	}
	return t, err
}

// SaveLeftTickets Save tickets info into database. insert or update, depending on the availability of this particular entry.
func (db *DB) SaveLeftTickets(t *TicketEntity) error {
	stmt, err := db.Prepare("insert into tickets (from_station, to_station, travel_date, content, update_time) values ($1, $2, $3, $4, now())")
	defer stmt.Close()
	if err != nil {
		log.Error("failed to prepare sql statement: ", err, stmt)
		return err
	}

	_, err = stmt.Exec(t.From, t.To, t.Date, t.Content)
	if err != nil {
		log.Error("failed to execute sql statement: ", err, stmt)
		// we try to update it
		updateStmt, err := db.Prepare("update tickets set content = $1, update_time = now() where from_station = $2 and to_station = $3 and travel_date = $4")
		defer updateStmt.Close()

		if err != nil {
			log.Error("failed to prepare sql statement: ", err, updateStmt)
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
		log.Error("failed to prepare sql statement: ", err, stmt)
		return t, err
	}

	err = stmt.QueryRow(t.TrainNo, t.FromStationNo, t.ToStationNo, t.SeatTypes).
		Scan(&t.Id, &t.TrainNo, &t.FromStationNo, &t.ToStationNo, &t.SeatTypes, &t.Content, &t.UpdateTime)

	return t, err
}

// insert or update
func (db *DB) SaveTicketPrice(t *TicketPriceEntity) error {
	stmt, err := db.Prepare("insert into ticket_price (train_no, from_station_no, to_station_no, seat_types, content, update_time) values ($1, $2, $3, $4, $5, now())")
	defer stmt.Close()
	if err != nil {
		log.Error("failed to prepare sql statement: ", err, stmt)
		return err
	}

	_, err = stmt.Exec(t.TrainNo, t.FromStationNo, t.ToStationNo, t.SeatTypes, t.Content)
	if err != nil {
		// try update
		updateStmt, err := db.Prepare("update ticket_price set content = $1, update_time = now() where train_no = $2 and from_station_no = $3 and to_station_no = $4 and seat_types = $5")
		defer updateStmt.Close()

		if err != nil {
			log.Error("failed to prepare sql statement: ", err, updateStmt)
			return err
		}

		_, err = updateStmt.Exec(t.Content, t.TrainNo, t.FromStationNo, t.ToStationNo, t.SeatTypes)
		return err
	}
	return nil
}
