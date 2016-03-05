package insight_server

import (
	"net/http"
)

func PingHandler(w http.ResponseWriter, req *http.Request) {
	writeResponse(w, http.StatusOK, "PONG")
}
