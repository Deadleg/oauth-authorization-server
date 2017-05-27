package oauth

import (
	"sync"
	"time"
)

type Event struct {
	ClientID string
	Time     time.Time
}

type Counter interface {
	Add(e *Event) error
	GetEvents(clientID string) []Event
	GetEventsFrom(clientID string, last time.Time) []Event
}

type ClientCounter struct {
	sync.Mutex
	events *[]Event
}

type InMemoryCounter struct {
	sync.Mutex // For counter creation
	counters   map[string]*ClientCounter
}

func MakeInMemoryCounter() Counter {
	return &InMemoryCounter{counters: map[string]*ClientCounter{}}
}

func (c *InMemoryCounter) Add(e *Event) error {
	counter := c.counters[e.ClientID]
	if counter == nil {
		c.Lock()
		if c.counters[e.ClientID] == nil {
			counter = &ClientCounter{events: &[]Event{}}
			c.counters[e.ClientID] = counter
		}
		c.Unlock()
	}
	counter.Lock()
	defer counter.Unlock()
	newEvents := append(*counter.events, *e)
	counter.events = &newEvents
	return nil
}

func (c *InMemoryCounter) GetEvents(clientID string) []Event {
	return *c.counters[clientID].events
}

func (c *InMemoryCounter) GetEventsFrom(clientID string, last time.Time) []Event {
	return *c.counters[clientID].events
}
