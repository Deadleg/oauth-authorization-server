package oauth

import (
	"fmt"

	"database/sql"
	"time"

	"github.com/RangelReale/osin"
	_ "github.com/go-sql-driver/mysql" // sql driver
	"github.com/jmoiron/sqlx"
)

// Client entity for "clients" table
type Client struct {
	ID                 string `db:"id"`
	Secret             string
	Extra              string
	RedirectURI        string `db:"redirect_uri"`
	Owner              int
	RateLimitPerSecond int `db:"rate_limit_per_second"`
}

// AccessToken entity for "access_token" table
type AccessToken struct {
	Client       string
	Authorize    string
	Previous     string
	AccessToken  string `db:"access_token"`
	RefreshToken string `db:"refresh_token"`
	ExpiresIn    int32  `db:"expires_in"`
	Scope        string
	RedirectURI  string `db:"redirect_uri"`
	Extra        sql.NullString
	CreatedAt    time.Time `db:"created_at"`
}

// RefreshToken entity for "refresh_token" table
type RefreshToken struct {
	Token  string
	access string
}

// MysqlStore implements osin.Storage
type MysqlStore struct {
	db *sqlx.DB
}

// SetupStorage sets up database connections and any caches
func SetupStorage(db *sqlx.DB) (*MysqlStore, error) {
	return &MysqlStore{db: db}, nil
}

func (s *MysqlStore) Clone() osin.Storage {
	return s
}

func (s *MysqlStore) Close() {
	//	s.db.Close()
}

func (s *MysqlStore) GetClient(id string) (osin.Client, error) {
	client := &Client{}
	err := s.db.Get(client, "SELECT * from clients WHERE id=?", id)
	if err != nil {
		fmt.Print(err.Error())
		return nil, err
	}

	return &osin.DefaultClient{
		Id:          client.ID,
		Secret:      client.Secret,
		RedirectUri: client.RedirectURI}, nil
}

func (s *MysqlStore) SaveAuthorize(*osin.AuthorizeData) error {
	return nil
}

func (s *MysqlStore) LoadAuthorize(code string) (*osin.AuthorizeData, error) {
	return nil, nil
}

// RemoveAuthorize revokes or deletes the authorization code.
func (s *MysqlStore) RemoveAuthorize(code string) error {
	return nil
}

func (s *MysqlStore) SaveAccess(access *osin.AccessData) error {
	var previousToken string
	if access.AccessData != nil {
		previousToken = access.AccessData.AccessToken
	}

	var authorizationToken string
	if access.AuthorizeData != nil {
		authorizationToken = access.AuthorizeData.Code
	}

	_, err := s.db.NamedExec(`INSERT INTO access_tokens (
			client,
			authorize,
			previous,
			access_token,
			refresh_token,
			expires_in,
			scope,
			redirect_uri
		) VALUES (
			:client,
			:authorize,
			:previous,
			:access_token,
			:refresh_token,
			:expires_in,
			:scope,
			:redirect_uri
		)`,
		map[string]interface{}{
			"client":        access.Client.GetId(),
			"authorize":     authorizationToken,
			"previous":      previousToken,
			"access_token":  access.AccessToken,
			"refresh_token": access.RefreshToken,
			"expires_in":    access.ExpiresIn,
			"scope":         access.Scope,
			"redirect_uri":  access.RedirectUri,
		})
	return err
}

func (s *MysqlStore) LoadAccess(token string) (*osin.AccessData, error) {
	accessToken := &AccessToken{}
	err := s.db.Get(
		accessToken,
		`SELECT 
			client, 
			authorize,
			prevous,
			access_token,
			refresh_token,
			expires_token,
			scope,
			redirect_uri,
			extra,
			created_at
		WHERE access_token=?`,
		token)

	client, err := s.GetClient(accessToken.Client)
	if err != nil {
		return nil, err
	}

	var authorizeData *osin.AuthorizeData
	if accessToken.Authorize != "" {
		authorizeData, err = s.LoadAuthorize(accessToken.Authorize)
		if err != nil {
			return nil, err
		}
	}

	var accessData *osin.AccessData
	if accessToken.Previous != "" {
		accessData, err = s.LoadAccess(accessToken.Previous)
		if err != nil {
			return nil, err
		}
	}

	data := &osin.AccessData{
		Client:        client,
		AuthorizeData: authorizeData,
		AccessData:    accessData,
		AccessToken:   accessToken.AccessToken,
		RefreshToken:  accessToken.RefreshToken,
		ExpiresIn:     accessToken.ExpiresIn,
		Scope:         accessToken.Scope,
		RedirectUri:   accessToken.RedirectURI,
		CreatedAt:     accessToken.CreatedAt,
		UserData:      accessToken.Extra,
	}

	return data, err
}

func (s *MysqlStore) RemoveAccess(token string) error {
	_, err := s.db.NamedExec(
		`DELETE FROM access_tokens WHERE access_token=:access_token`,
		map[string]interface{}{
			"access_token": token,
		})
	return err
}

func (s *MysqlStore) LoadRefresh(token string) (*osin.AccessData, error) {
	refreshToken := &RefreshToken{}
	err := s.db.Get(
		refreshToken,
		"SELECT token, access FROM refresh_tokens WHERE token=?",
		token)

	if err != nil {
		return nil, err
	}

	return s.LoadAccess(refreshToken.access)
}

func (s *MysqlStore) RemoveRefresh(token string) error {
	_, err := s.db.Exec("DELETE FROM refresh_tokens WHERE token=?", token)
	return err
}
