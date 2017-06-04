package oauth

import (
	"net/http"

	"strconv"

	"encoding/json"

	"time"

	"fmt"

	"github.com/RangelReale/osin"
	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/negroni"
)

type OAuthHandler struct {
	server      *osin.Server
	rateLimiter *RateLimiterPool
	counter     Counter
	redis       *redis.Client
	alerter     *Alerter
}

// SetupAuthorizationServer adds http handlers for the authorization server
func SetupAuthorizationServer(
	r *mux.Router,
	osinServer *osin.Server,
	clientService ClientService,
	counter Counter,
	redis *redis.Client,
	alerter *Alerter) {
	h := OAuthHandler{
		server:      osinServer,
		rateLimiter: MakeRateLimiter(clientService),
		counter:     counter,
		redis:       redis,
		alerter:     alerter,
	}

	n := negroni.New(negroni.HandlerFunc(h.counterMiddleware), negroni.HandlerFunc(h.rateLimitMiddleware))
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
		id, message, err := h.alerter.createAlert(clientID, rateLimitHit)
		if err != nil {
			log.Info(err)
		} else {
			alert := Alert{
				ID:        id,
				Title:     rateLimitHit,
				Message:   fmt.Sprintf(message, result.Limit),
				Timestamp: time.Now().Unix(),
			}
			bytes, err := json.Marshal(alert)
			if err != nil {
				log.Error(err)
			} else {
				h.redis.Publish("oauth:"+clientID+":info", string(bytes))
			}
		}
		http.Error(rw, string(jsonResponse), 429)
	} else {
		next(rw, r)
	}
}

func (h OAuthHandler) counterMiddleware(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	r.ParseForm()
	clientID := r.Form["client_id"][0]
	h.counter.Add(&Event{
		ClientID: clientID,
		Time:     time.Now()})

	next(rw, r)
}
