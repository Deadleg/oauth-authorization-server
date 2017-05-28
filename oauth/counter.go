package oauth

import (
	"sync"
	"time"

	"encoding/json"

	"github.com/go-redis/redis"
)

type WarningEvent struct {
	Message   string
	EventType string `json:"type"`
	Time      time.Time
}

type EventChannels struct {
	Warnings chan WarningEvent
	Requests chan []Event
}

func MakeEventChannels() *EventChannels {
	return &EventChannels{
		Warnings: make(chan WarningEvent),
		Requests: make(chan []Event),
	}
}

type Event struct {
	ClientID string    `json:"clientID"`
	Time     time.Time `json:"time"`
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
	redis      *redis.Client
}

func MakeInMemoryCounter(redis *redis.Client) Counter {
	return &InMemoryCounter{
		counters: map[string]*ClientCounter{},
		redis:    redis,
	}
}

func (c *InMemoryCounter) Add(e *Event) error {
	bytes, err := json.Marshal(*e)
	if err != nil {
		return err
	}
	c.redis.Publish("oauth:"+e.ClientID+":events", string(bytes))
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
