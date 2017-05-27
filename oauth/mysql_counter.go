package oauth

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type MysqlCounter struct {
	db sqlx.DB
}

func MakeMysqlCounter(db sqlx.DB) Counter {
	return &MysqlCounter{db: db}
}

func (c *MysqlCounter) Add(e *Event) error {
	return fmt.Errorf("Error")
}

func (c *MysqlCounter) GetEvents(clientID string) []Event {
	return []Event{}
}

func (c *MysqlCounter) GetEventsFrom(clientID string, last time.Time) []Event {
	return []Event{}
}
