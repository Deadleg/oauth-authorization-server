package main

import (
	"log"
	"net/http"

	"flavouredproductions.com/oauth-authorization-server/admin"
	"flavouredproductions.com/oauth-authorization-server/auth"
	"flavouredproductions.com/oauth-authorization-server/oauth"
	"flavouredproductions.com/oauth-authorization-server/users"
	"github.com/RangelReale/osin"
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	"github.com/urfave/negroni"
)

func main() {
	cookieName := "auth-server"
	connectionString := "tick:tick@/tick"

	db, err := sqlx.Connect("mysql", connectionString)
	if err != nil {
		log.Fatalln(err)
	}

	storage, _ := oauth.SetupStorage(db)
	config := &osin.ServerConfig{
		AuthorizationExpiration:   250,
		AccessExpiration:          3600,
		TokenType:                 "Bearer",
		AllowedAuthorizeTypes:     osin.AllowedAuthorizeType{osin.CODE},
		AllowedAccessTypes:        osin.AllowedAccessType{osin.AUTHORIZATION_CODE, osin.CLIENT_CREDENTIALS},
		ErrorStatusCode:           200,
		AllowClientSecretInParams: true,
		AllowGetAccessRequest:     true,
		RetainTokenAfterRefresh:   false,
	}

	csrf.Secure(false)

	sessionStore := sessions.NewCookieStore([]byte("something-very-secret"))

	userService := users.CreateMysqlDatastore(db)
	clientService := oauth.CreateMysqlClientService(db)

	n := negroni.New(negroni.NewRecovery(), negroni.NewLogger())
	r := mux.NewRouter()
	n.UseHandler(r)

	server := osin.NewServer(config, storage)
	oauth.SetupAuthorizationServer(r, server, clientService)
	auth.SetupHandlers(r, userService, sessionStore, cookieName)
	admin.SetupHandlers(r, storage, userService, clientService, sessionStore, cookieName)
	http.Handle("/", n)

	http.ListenAndServe(":14000", nil)
}
