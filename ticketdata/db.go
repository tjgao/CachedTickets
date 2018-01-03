package ticketdata

import (
	"database/sql"
	_ "github.com/lib/pq"
)

type TicketInfo interface {
	GetLeftTickets(t *TicketEntity) (*TicketEntity, error)
	SaveLeftTickets(t *TicketEntity) error
	GetTicketPrice(t *TicketPriceEntity) (*TicketPriceEntity, error)
	SaveTicketPrice(t *TicketPriceEntity) error
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
