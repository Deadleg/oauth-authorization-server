package users

import (
	"github.com/jmoiron/sqlx"
)

type User struct {
	ID       int `db:"id"`
	Username string
	Password string
	Roles    Roles
}

type Role struct {
	User int
	Role string
}

type Roles []Role

type UserService interface {
	GetUser(username string) *User
	GetUserById(id int) (*User, error)
	HasRole(username string, role string) bool
	Login(username string, password string) bool
}

type MysqlUserService struct {
	db *sqlx.DB
}

func CreateMysqlDatastore(db *sqlx.DB) UserService {
	return &MysqlUserService{db: db}
}

func (us MysqlUserService) Login(username string, password string) bool {
	user := us.GetUser(username)
	// TODO proper check
	return user.Password == password
}

func (us MysqlUserService) GetUser(username string) *User {
	user := &User{}
	us.db.Get(user, "SELECT id, username, password FROM users WHERE username=?", username)
	roles := &Roles{}
	us.db.Select(
		roles,
		`SELECT role FROM user_roles 
		WHERE user=(
			SELECT id FROM users WHERE username=?)`,
		username)
	user.Roles = *roles
	return user
}

func (us MysqlUserService) GetUserById(id int) (*User, error) {
	user := &User{}
	err := us.db.Get(user, "SELECT id, username, password FROM users WHERE id=?", id)
	if err != nil {
		return nil, err
	}
	roles := &Roles{}
	err = us.db.Select(
		roles,
		`SELECT role FROM user_roles 
		WHERE user=?`,
		id)
	if err != nil {
		return nil, err
	}
	user.Roles = *roles
	return user, nil
}

func (us MysqlUserService) HasRole(username string, role string) bool {
	var hasRole bool
	us.db.Get(
		hasRole,
		`SELECT EXISTS(
			SELECT 1 FROM user_roles 
			WHERE user=(
				SELECT id FROM users WHERE username=?)
			AND
				role=?`,
		username,
		role)
	return hasRole
}
