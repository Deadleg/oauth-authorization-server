package web

import (
	"context"
	"html/template"
	"net/http"
	"net/url"

	"gopkg.in/igm/sockjs-go.v2/sockjs"

	"encoding/json"

	"github.com/deadleg/oauth-authorization-server/auth"
	"github.com/deadleg/oauth-authorization-server/oauth"
	"github.com/deadleg/oauth-authorization-server/users"
	"github.com/go-redis/redis"
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/negroni"
)

type webHandler struct {
	users      users.UserService
	clients    oauth.ClientService
	sessions   *sessions.CookieStore
	cookieName string
	redis      *redis.Client
}

const (
	templatesFolder = "templates/web/"
)

var upgrader = websocket.Upgrader{}

func SetupHandlers(
	r *mux.Router,
	us users.UserService,
	cs oauth.ClientService,
	sessionStore *sessions.CookieStore,
	cookieName string,
	redis *redis.Client) {
	h := &webHandler{
		users:      us,
		clients:    cs,
		sessions:   sessionStore,
		cookieName: cookieName,
		redis:      redis,
	}

	n := negroni.New(negroni.HandlerFunc(h.authMiddleware))
	s := mux.NewRouter()
	CSRF := csrf.Protect([]byte("32-byte-long-auth-key"), csrf.Secure(false), csrf.Path("/"))
	n.UseHandler(CSRF(s))

	s.Path("/").HandlerFunc(h.indexHandler)
	s.Methods("GET").Path("/account/clients").HandlerFunc(h.clientsHandler)
	s.Methods("GET").Path("/account/clients/{ID}").HandlerFunc(h.clientHandler)
	s.Methods("GET").Path("/account/clients/create").HandlerFunc(h.createClientHandler)
	s.Methods("POST").Path("/account/clients/delete/{ID}").HandlerFunc(h.deleteClientHandler)

	socketHandler := sockjs.NewHandler("/ws/account/clients", sockjs.DefaultOptions, h.activityWebsocket)
	s.PathPrefix("/ws/account/clients").Handler(socketHandler)

	r.PathPrefix("/").Handler(n)
}

type Client struct {
	Client oauth.Client
	User   users.User
}

type ClientsPage struct {
	Clients      []Client
	Title        string
	SignedInUser auth.SignedInUser
}

type ClientPage struct {
	Client       Client
	Title        string
	SignedInUser auth.SignedInUser
}

type IndexPage struct {
	AppName      string
	Title        string
	SignedInUser auth.SignedInUser
}

func (h webHandler) activityWebsocket(session sockjs.Session) {
	wsMsg, err := session.Recv()
	if err != nil {
		log.Info(err)
		return
	}

	ids := []string{}
	log.Info(wsMsg)
	err = json.Unmarshal([]byte(wsMsg), &ids)
	if err != nil {
		log.Info(err)
		return
	}

	channels := []string{}
	for _, id := range ids {
		channels = append(channels, "oauth:"+id+":events")
	}

	pubsub := h.redis.Subscribe(channels...)
	defer pubsub.Close()
	for {
		msg, err := pubsub.ReceiveMessage()
		if err != nil {
			log.Info(err)
			break
		}
		log.Info(msg.Payload)
		session.Send(msg.Payload)
	}
}

func (h webHandler) indexHandler(w http.ResponseWriter, r *http.Request) {
	maybeUser := r.Context().Value(auth.SignedInUserContextKey{})
	var user auth.SignedInUser
	if maybeUser != nil {
		user = maybeUser.(auth.SignedInUser)
	}

	p := &IndexPage{
		AppName:      "Authorizaton server",
		Title:        "Authorizaton server",
		SignedInUser: user,
	}

	t, _ := getTemplate("index.html")
	t.Execute(w, map[string]interface{}{
		"page":           p,
		csrf.TemplateTag: csrf.Token(r),
	})
}

func getTemplate(templateName string) (*template.Template, error) {
	t, err := template.ParseFiles(
		templatesFolder+templateName,
		"templates/partial/header.html",
		"templates/partial/navbar.html")
	return t, err
}

func (h webHandler) authMiddleware(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	session, _ := h.sessions.Get(r, h.cookieName)
	if session.IsNew && r.URL.Path != "/" {
		redirect, _ := url.Parse("/auth/login?")
		query := url.Values{}
		query.Add("redirect", r.URL.Path)
		redirect.RawQuery = query.Encode()
		http.Redirect(rw, r, redirect.String(), http.StatusSeeOther)
	}

	if !session.IsNew {
		u := auth.SignedInUser{
			Username: session.Values["Username"].(string),
		}

		ctx := context.WithValue(r.Context(), auth.SignedInUserContextKey{}, u)

		next(rw, r.WithContext(ctx))
	} else {
		next(rw, r)
	}
}
