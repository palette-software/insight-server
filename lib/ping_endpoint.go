package insight_server

import (
	"net/http"
)

func PingHandler(w http.ResponseWriter, req *http.Request) {
	WriteResponse(w, http.StatusOK, "PONG", req)
}
