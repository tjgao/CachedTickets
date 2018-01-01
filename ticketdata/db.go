package ticketdata

import (
	"database/sql"
	_ "github.com/lib/pq"
)

type TicketInfo interface {
	GetLeftTickets(string, string, string) (string, error)
	GetTicketPrice(string, string, string, string) (string, error)
}

type DB struct {
	*sql.DB
}

func NewDB(dataSourceName string) (*DB, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return &DB{db}, nil
}
