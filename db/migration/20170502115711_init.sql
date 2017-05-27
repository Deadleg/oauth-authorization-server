-- +goose Up
CREATE TABLE IF NOT EXISTS clients (
	id           		  varchar(255) NOT NULL PRIMARY KEY,
	secret 		 		  varchar(255) NOT NULL,
	extra 		 		  varchar(255),
	redirect_uri 		  varchar(255),
	rate_limit_per_second int NOT NULL default 60,
	owner		 		  int(255),
    FOREIGN KEY (owner)
        REFERENCES users(id)
        ON DELETE CASCADE 
);

CREATE TABLE IF NOT EXISTS authorization_tokens (
	client       varchar(255) NOT NULL,
	code         varchar(255) NOT NULL PRIMARY KEY,
	expires_in   int(10) NOT NULL,
	scope        varchar(255) NOT NULL,
	redirect_uri varchar(255) NOT NULL,
	state        varchar(255) NOT NULL,
	extra 		 varchar(255),
	created_at   timestamp NOT NULL,
    FOREIGN KEY (client)
        REFERENCES clients(id)
        ON DELETE CASCADE 
);

CREATE TABLE IF NOT EXISTS access_tokens (
	client        varchar(255) NOT NULL,
	authorize     varchar(255) NOT NULL,
	previous      varchar(255) NOT NULL,
	access_token  varchar(255) NOT NULL PRIMARY KEY,
	refresh_token varchar(255) NOT NULL,
	expires_in    int(10) NOT NULL,
	scope         varchar(255) NOT NULL,
	redirect_uri  varchar(255) NOT NULL,
	extra 		  varchar(255),
	created_at    timestamp NOT NULL,
    FOREIGN KEY (client)
        REFERENCES clients(id)
        ON DELETE CASCADE 
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
	token         varchar(255) NOT NULL PRIMARY KEY,
	access        varchar(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
	id int NOT NULL PRIMARY KEY AUTO_INCREMENT,
	username varchar(255) NOT NULL UNIQUE,
	password varchar(255) NOT NULL,
	email varchar(255)
);

CREATE TABLE IF NOT EXISTS user_roles (
	user int NOT NULL,
	role varchar(255) NOT NULL,
	UNIQUE KEY role_unique (user, role),
	FOREIGN KEY (user)
		REFERENCES users(id)
		ON DELETE CASCADE
);