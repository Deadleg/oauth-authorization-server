package oauth

import "github.com/jmoiron/sqlx"

type Alerter struct {
	db *sqlx.DB
}

type Alert struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

const rateLimitHit = "Ratelimit hit"

func MakeAlerter(db *sqlx.DB) *Alerter {
	return &Alerter{db: db}
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
