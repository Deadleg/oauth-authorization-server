package web

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/deadleg/oauth-authorization-server/auth"
	"github.com/deadleg/oauth-authorization-server/oauth"
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

func (h webHandler) deleteClientHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	// TODO verify owner
	h.clients.DeleteClient(vars["ID"])
}

func (h webHandler) createClientHandler(w http.ResponseWriter, r *http.Request) {
	ownerID, _ := h.sessions.Get(r, h.cookieName)
	newClient := oauth.NewClient{
		ClientId:     oauth.RandStringRunes(10),
		ClientSecret: oauth.RandStringRunes(15),
		Owner:        ownerID.Values["UserId"].(int),
		RedirectUri:  "http://example.com",
	}
	h.clients.CreateClient(newClient)

	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(newClient)
}

func (h webHandler) clientsHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := h.sessions.Get(r, h.cookieName)
	if err != nil {
		http.Error(w, "Not logged in", 403)
		return
	}
	ownerID := cookie.Values["UserId"].(int)
	clients, err := h.clients.GetUsersClients(ownerID)
	if err != nil {
		log.Error(err.Error())
		http.Error(w, "Internal error", 500)
		return
	}

	c := []Client{}
	user, _ := h.users.GetUserById(ownerID)
	for _, client := range *clients {
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

func (h webHandler) clientHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := h.sessions.Get(r, h.cookieName)
	if err != nil {
		http.Error(w, "Not logged in", 403)
		return
	}
	ownerID := cookie.Values["UserId"].(int)
	vars := mux.Vars(r)
	client, err := h.clients.GetClient(vars["ID"])
	if err != nil {
		log.Error(err.Error())
		http.Error(w, "Internal error", 500)
		return
	}

	user, _ := h.users.GetUserById(ownerID)
	c := Client{Client: *client, User: *user}

	p := &ClientPage{
		Client:       c,
		Title:        "Client",
		SignedInUser: r.Context().Value(auth.SignedInUserContextKey{}).(auth.SignedInUser),
	}

	t, err := getTemplate("client.html")
	t.Execute(w, map[string]interface{}{
		"page":           p,
		csrf.TemplateTag: csrf.Token(r),
	})
}
