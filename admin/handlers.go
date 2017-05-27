package admin

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"

	"github.com/RangelReale/osin"
	"github.com/deadleg/oauth-authorization-server/auth"
	"github.com/deadleg/oauth-authorization-server/oauth"
	"github.com/deadleg/oauth-authorization-server/users"
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/urfave/negroni"
)

type adminHandler struct {
	users        users.UserService
	clients      oauth.ClientService
	oauthStorage osin.Storage
	sessions     *sessions.CookieStore
	cookieName   string
}

var (
	templatesFolder = "templates/admin/"
)

func SetupHandlers(
	r *mux.Router,
	storage osin.Storage,
	us users.UserService,
	cs oauth.ClientService,
	sessionStore *sessions.CookieStore,
	cookieName string) {
	adminHandler := &adminHandler{
		users:        us,
		oauthStorage: storage,
		clients:      cs,
		sessions:     sessionStore,
		cookieName:   cookieName,
	}

	n := negroni.New(negroni.HandlerFunc(adminHandler.authMiddleware))
	s := mux.NewRouter()
	CSRF := csrf.Protect([]byte("32-byte-long-auth-key"), csrf.Secure(false), csrf.Path("/"))
	n.UseHandler(CSRF(s))

	s.Path("/clients").HandlerFunc(adminHandler.clientsHandler)
	s.Methods("GET").Path("/clients/create").HandlerFunc(adminHandler.createClientHandler)
	s.Methods("POST").Path("/clients/delete/{ID}").HandlerFunc(adminHandler.deleteClientHandler)

	r.PathPrefix("/admin").Handler(http.StripPrefix("/admin", n))
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

func (ad adminHandler) clientsHandler(w http.ResponseWriter, r *http.Request) {
	clients, _ := ad.clients.GetClients()

	c := []Client{}
	for _, client := range *clients {
		user, _ := ad.users.GetUserById(client.Owner)
		c = append(c, Client{Client: client, User: *user})
	}

	p := &ClientsPage{
		Clients:      c,
		Title:        "Clients",
		SignedInUser: r.Context().Value(auth.SignedInUserContextKey{}).(auth.SignedInUser),
	}

	t, _ := getTemplate("clients.html")
	t.Execute(w, map[string]interface{}{
		"page":           p,
		csrf.TemplateTag: csrf.Token(r),
	})
}

func (ad adminHandler) deleteClientHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ad.clients.DeleteClient(vars["ID"])
}

func (ad adminHandler) createClientHandler(w http.ResponseWriter, r *http.Request) {
	newClient := oauth.NewClient{
		ClientId:     oauth.RandStringRunes(10),
		ClientSecret: oauth.RandStringRunes(15),
		Owner:        1,
		RedirectUri:  "http://example.com",
	}
	ad.clients.CreateClient(newClient)

	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(newClient)
}

func getTemplate(templateName string) (*template.Template, error) {
	t, err := template.ParseFiles(
		templatesFolder+templateName,
		"templates/partial/header.html",
		"templates/partial/admin/navbar.html")
	return t, err
}

func (ad adminHandler) authMiddleware(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	session, _ := ad.sessions.Get(r, ad.cookieName)
	if session.IsNew {
		redirect, _ := url.Parse("/auth/login?")
		query := url.Values{}
		// prepend /admin since it gets stripped in the router
		query.Add("redirect", "/admin"+r.URL.Path)
		redirect.RawQuery = query.Encode()
		http.Redirect(rw, r, redirect.String(), http.StatusSeeOther)
	}

	u := auth.SignedInUser{
		Username: session.Values["Username"].(string),
	}

	ctx := context.WithValue(r.Context(), auth.SignedInUserContextKey{}, u)

	next(rw, r.WithContext(ctx))
}
