package oauth

import (
	"math/rand"
	"time"

	"github.com/jmoiron/sqlx"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type NewClient struct {
	ClientId     string
	ClientSecret string
	RedirectUri  string
	Owner        int
}

type ClientService interface {
	GetClients() (*[]Client, error)
	GetClient(clientID string) (*Client, error)
	CreateClient(NewClient) error
	DeleteClient(id string) error
}

type MysqlClientService struct {
	db *sqlx.DB
}

func CreateMysqlClientService(db *sqlx.DB) ClientService {
	return &MysqlClientService{db: db}
}

func (cs MysqlClientService) GetClients() (*[]Client, error) {
	clients := &[]Client{}
	err := cs.db.Select(
		clients,
		`SELECT id, secret, extra, redirect_uri, owner, rate_limit_per_second
		 FROM clients`)
	if err != nil {
		return nil, err
	}
	return clients, nil
}

func (cs MysqlClientService) GetClient(clientID string) (*Client, error) {
	client := &Client{}
	err := cs.db.Get(
		client,
		`SELECT id, secret, extra, redirect_uri, owner, rate_limit_per_second
		 FROM clients
		 WHERE id=?`,
		clientID)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (cs MysqlClientService) CreateClient(c NewClient) error {
	_, err := cs.db.NamedExec(
		`INSERT INTO clients (id, secret, owner, extra, redirect_uri)
		VALUES (:id, :secret, :owner, :extra, :redirect_uri)`,
		map[string]interface{}{
			"id":           c.ClientId,
			"secret":       c.ClientSecret,
			"owner":        c.Owner,
			"extra":        "",
			"redirect_uri": c.RedirectUri,
		})
	return err
}

func (cs MysqlClientService) DeleteClient(id string) error {
	_, err := cs.db.NamedExec(
		`DELETE FROM clients WHERE id=:id`,
		map[string]interface{}{
			"id": id,
		})
	return err
}

var letterRunes = []rune("123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// RandStringRunes generates random string for id/secret
// TODO proper generation
func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
