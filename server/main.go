package main

import (
	"fmt"
	"net/http"

	"github.com/palette-software/insight-server"
	"os"
)

const (
	// The address where the server will bind itself
	BindAddress = ":9000"
)

func pingHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "PONG")
}

func getBindAddress() string {
	port := os.Getenv("PORT")
	if port == "" {
		return BindAddress
	}
	return fmt.Sprintf(":%v", port)
}

func main() {
	insight_server.Boot()

	http.HandleFunc("/", pingHandler)
	http.HandleFunc("/upload-with-meta", insight_server.CheckUserAuth( insight_server.UploadHanlder))

	bindAddress := getBindAddress()
	fmt.Printf("Webservice starting on %v\n", bindAddress)
	http.ListenAndServe(bindAddress, nil)
}
