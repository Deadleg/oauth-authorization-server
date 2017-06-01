package oauth

import (
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis"
	log "github.com/sirupsen/logrus"
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

// InMemoryCounter stores all events in memory with additional
// counting in redis.
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
	key := "event:" + e.ClientID + ":1:count"
	hkey := strconv.FormatInt((time.Now().Unix()/60)*60, 10)

	c.redis.HIncrBy(key, hkey, 1)
	c.redis.ZAdd("counters:"+e.ClientID+":1", redis.Z{Member: hkey, Score: 0})

	eventJSONBytes, err := json.Marshal(*e)
	if err != nil {
		return nil
	}

	lastKey := "last:100:" + e.ClientID
	pipe := c.redis.Pipeline()
	pipe.LPush(lastKey, string(eventJSONBytes))
	pipe.LTrim(lastKey, 0, 100)
	_, err = pipe.Exec()
	return err
}

func (c *InMemoryCounter) GetEvents(clientID string) []Event {
	list := c.redis.LRange("last:100:"+clientID, 0, -1)
	data := []Event{}
	for _, v := range list.Val() {
		e := Event{}
		err := json.Unmarshal([]byte(v), &e)
		if err != nil {
			log.Info(err)
		}
		data = append(data, e)
	}
	return data
	//return *c.counters[clientID].events
}

func (c *InMemoryCounter) GetEventsFrom(clientID string, last time.Time) []Event {
	return *c.counters[clientID].events
}
