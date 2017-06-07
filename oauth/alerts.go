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

func (a *Alerter) isAlerting(client string, title string) (bool, error) {
	exists := false
	err := a.db.Get(&exists,
		`SELECT EXISTS(*) 
		FROM alerts WHERE title=?
		GROUP BY time > date_sub(now(), interval 5 min)`,
		title)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (a *Alerter) createAlert(client string, title string) (int64, string, bool, error) {
	result, err := a.db.NamedExec(
		`INSERT IGNORE INTO alerts (client, alert_type) 
		SELECT * FROM (SELECT :client, (SELECT id FROM alert_types WHERE title=:title)) AS tmp
		WHERE NOT EXISTS (
			SELECT time 
			FROM alerts 
			WHERE time > date_sub(now(), interval 1 minute)
		) LIMIT 1`,
		map[string]interface{}{
			"client": client,
			"title":  title,
		})
	if err != nil {
		return 0, "", false, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, "", false, err
	}
	if rows == 0 {
		return 0, "", false, nil
	}

	message := ""
	err = a.db.Get(&message, `SELECT message FROM alert_types WHERE title=?`, title)
	if err != nil {
		return 0, "", false, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, "", false, err
	}

	return id, message, true, nil
}

func (a *Alerter) GetAlerts(client string) ([]Alert, error) {
	alerts := []*Alert{}

	err := a.db.Select(
		&alerts,
		`SELECT
			UNIX_TIMESTAMP(a.time) AS timestamp,
			a.id AS id, 
			a.client AS client, 
			at.title AS title, 
			at.message AS message,
			c.rate_limit_per_minute AS rate_limit_per_minute
		FROM alerts a 
		INNER JOIN alert_types at
			ON a.alert_type=at.id
		INNER JOIN clients c 
			ON c.id=a.client
		WHERE a.id in (
			SELECT max(id) FROM alerts WHERE client=? GROUP BY UNIX_TIMESTAMP(time) DIV 60
        )
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
