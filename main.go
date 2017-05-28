package main

import (
	"log"
	"net/http"

	"github.com/RangelReale/osin"
	"github.com/deadleg/oauth-authorization-server/admin"
	"github.com/deadleg/oauth-authorization-server/auth"
	"github.com/deadleg/oauth-authorization-server/oauth"
	"github.com/deadleg/oauth-authorization-server/users"
	"github.com/deadleg/oauth-authorization-server/web"
	"github.com/go-redis/redis"
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

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	server := osin.NewServer(config, storage)
	oauth.SetupAuthorizationServer(r, server, clientService, oauth.MakeInMemoryCounter(client), client)
	auth.SetupHandlers(r, userService, sessionStore, cookieName)
	admin.SetupHandlers(r, storage, userService, clientService, sessionStore, cookieName)
	web.SetupHandlers(r, userService, clientService, sessionStore, cookieName, client)
	http.Handle("/", n)

	http.ListenAndServe(":14000", nil)
}
