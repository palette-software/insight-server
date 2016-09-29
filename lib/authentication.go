package insight_server

import (
	"net/http"
)

// PUBLIC API
// ==========

// a handler function taking a tenant
type HandlerFuncWithTenant func(w http.ResponseWriter, req *http.Request, user string)

func MakeUserAuthHandler(internalHandler HandlerFuncWithTenant) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		internalHandler(w, r, "palette")
		return
	})
}
