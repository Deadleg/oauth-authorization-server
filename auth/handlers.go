package auth

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/deadleg/oauth-authorization-server/users"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

var (
	templatesFolder = "templates/auth/"
)

type authHandler struct {
	users      users.UserService
	sessions   *sessions.CookieStore
	cookieName string
}

type SignedInUser struct {
	Username string
}

type SignedInUserContextKey struct{}

func SetupHandlers(
	r *mux.Router,
	us users.UserService,
	sessions *sessions.CookieStore,
	sessionName string) {
	h := authHandler{
		users:      us,
		sessions:   sessions,
		cookieName: sessionName,
	}

	s := r.PathPrefix("/auth").Subrouter()
	s.Methods("GET").Path("/login").HandlerFunc(h.loginPageHandler)
	s.Methods("POST").Path("/login").HandlerFunc(h.loginPostHandler)
	s.Methods("GET").Path("/signout").HandlerFunc(h.signoutHandler)
}

type LoginPage struct {
	Redirect string
}

func (ad authHandler) loginPageHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := getTemplate("login.html")
	var redirect string
	maybeRedirects := r.URL.Query()["redirect"]
	if len(maybeRedirects) == 0 {
		redirect = "/"
	} else {
		redirect = maybeRedirects[0]
		validateRedirect(redirect, w)
	}
	p := LoginPage{Redirect: redirect}
	t.Execute(w, p)
}

func (ad authHandler) signoutHandler(w http.ResponseWriter, r *http.Request) {
	s, _ := ad.sessions.New(r, ad.cookieName)
	s.Options.MaxAge = -1
	s.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (ad authHandler) loginPostHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.Form["username"][0]
	password := r.Form["password"][0]
	redirect := r.Form["redirect"][0]
	validateRedirect(redirect, w)
	if ad.users.Login(username, password) {
		s, _ := ad.sessions.New(r, ad.cookieName)
		user := ad.users.GetUser(username)
		s.Values["UserId"] = user.ID
		s.Values["Username"] = user.Username
		s.Save(r, w)

		http.Redirect(w, r, redirect, http.StatusSeeOther)
	}

}

func validateRedirect(redirect string, w http.ResponseWriter) {
	if !strings.HasPrefix(redirect, "/") {
		http.Error(w, "Bad redirect parameter.", 400)
	}
}

func getTemplate(templateName string) (*template.Template, error) {
	t, err := template.ParseFiles(
		templatesFolder+templateName,
		"templates/partial/header.html",
		"templates/partial/admin/navbar.html")
	return t, err
}
