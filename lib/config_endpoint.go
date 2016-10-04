package insight_server

import (
	"net/http"
)

func ServeConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		WriteResponse(w, http.StatusOK, "cica")
	}
}

func UploadConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		WriteResponse(w, http.StatusOK, "cica")
	}
}
