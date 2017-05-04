package oauth

import (
	"net/http"

	"github.com/RangelReale/osin"
	"github.com/gorilla/mux"
)

var server *osin.Server

// SetupAuthorizationServer adds http handlers
func SetupAuthorizationServer(r *mux.Router, osinServer *osin.Server) {
	server = osinServer
	r.HandleFunc("/authorize", authorizeHandler)
	r.HandleFunc("/token", tokenHandler)
}

func tokenHandler(w http.ResponseWriter, r *http.Request) {
	resp := server.NewResponse()

	defer resp.Close()

	if ar := server.HandleAccessRequest(resp, r); ar != nil {
		ar.Authorized = true
		server.FinishAccessRequest(resp, r, ar)
	}
	osin.OutputJSON(resp, w, r)
}

func authorizeHandler(w http.ResponseWriter, r *http.Request) {
	resp := server.NewResponse()
	defer resp.Close()

	if ar := server.HandleAuthorizeRequest(resp, r); ar != nil {

		// HANDLE LOGIN PAGE HERE

		ar.Authorized = true
		server.FinishAuthorizeRequest(resp, r, ar)
	}
	osin.OutputJSON(resp, w, r)
}
