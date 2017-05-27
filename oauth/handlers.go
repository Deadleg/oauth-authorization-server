package oauth

import (
	"net/http"

	"strconv"

	"encoding/json"

	"github.com/RangelReale/osin"
	"github.com/gorilla/mux"
	"github.com/urfave/negroni"
)

type OAuthHandler struct {
	server      *osin.Server
	rateLimiter *RateLimiterPool
}

// SetupAuthorizationServer adds http handlers for the authorization server
func SetupAuthorizationServer(r *mux.Router, osinServer *osin.Server, clientService ClientService) {
	h := OAuthHandler{
		server:      osinServer,
		rateLimiter: MakeRateLimiter(clientService),
	}

	n := negroni.New(negroni.HandlerFunc(h.rateLimitMiddleware))
	o := mux.NewRouter()
	n.UseHandler(o)
	o.HandleFunc("/token", h.tokenHandler)

	r.HandleFunc("/authorize", h.authorizeHandler)
	r.PathPrefix("/token").Handler(n)
}

func (h OAuthHandler) tokenHandler(w http.ResponseWriter, r *http.Request) {
	resp := h.server.NewResponse()

	defer resp.Close()

	if ar := h.server.HandleAccessRequest(resp, r); ar != nil {
		ar.Authorized = true
		h.server.FinishAccessRequest(resp, r, ar)
	}
	osin.OutputJSON(resp, w, r)
}

func (h OAuthHandler) authorizeHandler(w http.ResponseWriter, r *http.Request) {
	resp := h.server.NewResponse()
	defer resp.Close()

	if ar := h.server.HandleAuthorizeRequest(resp, r); ar != nil {

		// HANDLE LOGIN PAGE HERE

		ar.Authorized = true
		h.server.FinishAuthorizeRequest(resp, r, ar)
	}
	osin.OutputJSON(resp, w, r)
}

type ApplicationProblem struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type RateLimit struct {
	ApplicationProblem
	ClientID string `json:"client_id"`
}

const clientRateLimitKey = "client"

func (h OAuthHandler) rateLimitMiddleware(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	r.ParseForm()
	clientID := r.Form["client_id"][0]
	result := h.rateLimiter.GetRateLimiter(clientID).Take(1)

	rw.Header().Set("X-Rate-Limit-Limit", strconv.Itoa(result.Limit))
	rw.Header().Set("X-Rate-Limit-Remaining", strconv.Itoa(result.Remaining))

	if result.Limited {
		rw.Header().Set("content-type", "application/problem+json")
		jsonResponse, _ := json.Marshal(RateLimit{
			ClientID: clientID,
			ApplicationProblem: ApplicationProblem{
				Title:       "Rate limit exceeded",
				Description: "",
			},
		})
		http.Error(rw, string(jsonResponse), 429)
	} else {
		next(rw, r)
	}
}