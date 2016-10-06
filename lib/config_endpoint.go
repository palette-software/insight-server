package insight_server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"net/url"
)

const UploadFileParam = "uploadfile"
const AgentConfigFileName = "Config.yml"

// Make sure that 'hostname' URL parameter is specified in the request.
func checkHostnameParam(w http.ResponseWriter, req *http.Request) (string, error) {
	hostname, err := url.QueryUnescape(req.FormValue("hostname"))
	if err != nil {
		WriteResponse(w, http.StatusBadRequest, fmt.Sprint("Failed to unescape hostname URL parameter! Error :", err))
		return nil, err
	}
	if hostname == "" {
		err := fmt.Errorf("Required hostname parameter is not specified in request URL: %s!", req.URL.RawQuery)
		WriteResponse(w, http.StatusBadRequest, err)
		return nil, err
	}

	return hostname
}

// Handler for GET /config endpoint
func ServeConfig(configDirectory string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		hostname, err := checkHostnameParam(w, req)
		if err != nil {
			// Bad request response has already been written
			return
		}

		sourcePath := path.Join(configDirectory, hostname, AgentConfigFileName)
		// FIXME Try FileServer instead!
		http.ServeFile(w, req, sourcePath)
	}
}

// Handler for PUT /config endpoint
func UploadConfig(configDirectory string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		hostname, err := checkHostnameParam(w, req)
		if err != nil {
			// Bad request response has already been written
			return
		}

		req.ParseMultipartForm(multipartMaxSize)
		uploadFile, _, err := req.FormFile("uploadfile")
		if err != nil {
			fmt.Println(err)
			return
		}
		defer uploadFile.Close()

		// Make sure that the destination folder exists
		destinationPath := path.Join(configDirectory, hostname, AgentConfigFileName)
		err = os.MkdirAll(path.Dir(destinationPath), 0777)
		if err != nil {
			WriteResponse(w, http.StatusInternalServerError,
				fmt.Sprintf("Failed to store uploaded config file! Error: %v", err))
			return
		}

		localFile, err := os.OpenFile(destinationPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			WriteResponse(w, http.StatusInternalServerError,
				fmt.Sprintf("Failed to create destination file! Error: %v", err))
			return
		}
		defer localFile.Close()

		written, err := io.Copy(localFile, uploadFile)
		WriteResponse(w, http.StatusOK, fmt.Sprintf("Successfully stored Config.yml for %s. Written bytes: %v", hostname, written))
	}
}
