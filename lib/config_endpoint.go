package insight_server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
)

func ServeConfig(configDirectory string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		hostname := req.FormValue("hostname")
		if hostname == "" {
			WriteResponse(w, http.StatusBadRequest, "Required hostname parameter is not specified!")
			return
		}

		sourcePath := path.Join(configDirectory, hostname, "Config.yml")
		http.ServeFile(w, req, sourcePath)
	}
}

func UploadConfig(configDirectory string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		hostname := req.FormValue("hostname")
		if hostname == "" {
			WriteResponse(w, http.StatusBadRequest, "Required hostname parameter is not specified!")
			return
		}

		req.ParseMultipartForm(32 << 20)
		uploadFile, _, err := req.FormFile("uploadfile")
		if err != nil {
			fmt.Println(err)
			return
		}
		defer uploadFile.Close()

		// Make sure that the destination folder exists
		destinationPath := path.Join(configDirectory, hostname, "Config.yml")
		err = os.MkdirAll(path.Dir(destinationPath), 777)
		if err != nil {
			WriteResponse(w, http.StatusInternalServerError,
				fmt.Sprintf("Failed to store uploaded config file! Error: %v", err))
			return
		}

		localFile, err := os.OpenFile(destinationPath, os.O_WRONLY|os.O_CREATE, 0666)
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
