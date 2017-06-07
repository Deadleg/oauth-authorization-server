package oauth

import "github.com/jmoiron/sqlx"
import "fmt"

type Alerter struct {
	db *sqlx.DB
}

type Alert struct {
	ID                 int64  `json:"id"`
	Client             string `json:"client"`
	Title              string `json:"title"`
	Message            string `json:"message"`
	Timestamp          int64  `json:"timestamp"`
	RateLimitPerMinute int    `json:"-" db:"rate_limit_per_minute"`
}

const rateLimitHit = "Ratelimit hit"

func MakeAlerter(db *sqlx.DB) *Alerter {
	return &Alerter{db: db}
}

func (a *Alert) message(template string, vars ...interface{}) {
	a.Message = fmt.Sprintf(template, vars...)
}

func (a *Alerter) createAlert(client string, title string) (int64, string, error) {
	result, err := a.db.NamedExec(
		`INSERT INTO alerts (client, alert_type) 
		VALUES (:client, (SELECT id FROM alert_types WHERE title=:title))`,
		map[string]interface{}{
			"client": client,
			"title":  title,
		})
	if err != nil {
		return 0, "", err
	}

	message := ""
	err = a.db.Get(&message, `SELECT message FROM alert_types WHERE title=?`, title)
	if err != nil {
		return 0, "", err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, "", err
	}

	return id, message, nil
}

func (a *Alerter) GetAlerts(client string) ([]Alert, error) {
	alerts := []*Alert{}

	err := a.db.Select(
		&alerts,
		`SELECT 
			a.id AS id, 
			UNIX_TIMESTAMP(a.time) AS timestamp,
			a.client AS client, 
			at.title AS title, 
			at.message AS message,
			c.rate_limit_per_minute AS rate_limit_per_minute
		FROM alerts a 
		INNER JOIN alert_types at
			ON a.alert_type=at.id
		INNER JOIN clients c 
			ON c.id=a.client
		WHERE client=? 
		LIMIT 5`,
		client)

	if err != nil {
		return nil, err
	}

	// Need to update message from template
	values := []Alert{}
	for _, a := range alerts {
		if a.Title == "Ratelimit hit" {
			a.message(a.Message, a.RateLimitPerMinute)
		}
		values = append(values, *a)
	}

	return values, nil
}
