package web

import (
	"context"
	"html/template"
	"net/http"
	"net/url"
	"sort"

	"gopkg.in/igm/sockjs-go.v2/sockjs"

	"encoding/json"

	"strconv"

	"math"

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
	counter    oauth.Counter
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
	redis *redis.Client,
	counter oauth.Counter) {
	h := &webHandler{
		users:      us,
		clients:    cs,
		sessions:   sessionStore,
		cookieName: cookieName,
		redis:      redis,
		counter:    counter,
	}

	n := negroni.New(negroni.HandlerFunc(h.authMiddleware))
	s := mux.NewRouter()
	CSRF := csrf.Protect([]byte("32-byte-long-auth-key"), csrf.Secure(false), csrf.Path("/"))
	n.UseHandler(CSRF(s))

	s.Path("/").HandlerFunc(h.indexHandler)
	s.Methods("GET").Path("/account/clients").HandlerFunc(h.clientsHandler)
	s.Methods("GET").Path("/account/clients/{ID}").HandlerFunc(h.clientHandler)
	s.Methods("GET").Path("/account/clients/{ID}/eventCounts").HandlerFunc(h.clientEventsCount)
	s.Methods("GET").Path("/account/clients/{ID}/events").HandlerFunc(h.clientEvents)
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

type EventType struct {
	Type string `json:"type"`
}

type Notification struct {
	EventType
	oauth.Alert
}

func (h webHandler) clientEvents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	data := h.counter.GetEvents(vars["ID"])
	json.NewEncoder(w).Encode(data)
}

func (h webHandler) clientEventsCount(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	jsonData := []map[string]interface{}{}
	data := h.redis.HGetAll("event:" + vars["ID"] + ":1:count")
	keys := []int{}

	for k, v := range data.Val() {
		t, _ := strconv.Atoi(k)
		i, _ := strconv.Atoi(v)
		jsonData = append(jsonData, map[string]interface{}{
			"timestamp": t,
			"value":     i,
		})
		keys = append(keys, t)
	}

	sort.Sort(sort.Reverse(sort.IntSlice(keys)))
	keys = keys[0:int(math.Min(10, float64(len(keys))))]
	for i := 0; i < 10; i++ {
		for j := 1; i+j < 10; j++ {
			if keys[i+1] != keys[i]-(j*60) {
				keys = append(keys, keys[i]-(j*60))
				sort.Sort(sort.Reverse(sort.IntSlice(keys)))
				keys = keys[0:int(math.Min(10, float64(len(keys))))]
				break
			} else {
				break
			}
		}
	}

	realJSONData := []map[string]interface{}{}
	for _, timestamp := range keys {
		var d map[string]interface{}
		for _, v := range jsonData {
			if v["timestamp"] == timestamp {
				d = v
			}
		}
		if d == nil {
			realJSONData = append(realJSONData, map[string]interface{}{
				"timestamp": timestamp,
				"value":     0,
			})
		} else {
			realJSONData = append(realJSONData, d)
		}
	}

	json.NewEncoder(w).Encode(realJSONData)
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
		channels = append(channels, "oauth:"+id+":info")
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
		msgBytes := []byte(msg.Payload)

		note := Notification{
			EventType: EventType{
				Type: "info",
			},
		}

		err = json.Unmarshal(msgBytes, &note)
		if err != nil {
			log.Info(err)
		}

		resp, err := json.Marshal(note)
		if err != nil {
			log.Info(err)
		}

		session.Send(string(resp))
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
