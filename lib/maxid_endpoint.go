package insight_server

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	log "github.com/palette-software/insight-tester/common/logging"
)

// HTTP HANDLERS
// =============

func MakeMaxIdHandler(backend MaxIdBackend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tableName, err := getUrlParam(r.URL, "table")
		if err != nil {
			WriteResponse(w, http.StatusBadRequest, "No 'table' parameter provided", r)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		maxId, err := backend.GetMaxId(tableName)
		if err != nil {
			if os.IsNotExist(err) {
				WriteResponse(w, http.StatusNoContent, "", r)
				return
			}

			WriteResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error while reading: %v", err), r)
			return
		}

		// signal that everything went ok
		WriteResponse(w, http.StatusOK, maxId, r)
	}
}

// HELPERS
// =======

var tableNameRegex *regexp.Regexp = regexp.MustCompile("^([^-]+)")

func getTableNameFromFilename(fileName string) (string, error) {
	tn := tableNameRegex.FindString(fileName)
	if tn == "" {
		return "", fmt.Errorf("Cannot find table name from file name: '%v'", fileName)
	}
	return tn, nil
}

// INTERFACE
// =========

// Implements storing and recalling a maxId
type MaxIdBackend interface {
	SaveMaxId(tableName, maxid string) error
	GetMaxId(tableName string) (string, error)
}

const (
	maxid_backend_default_filemode = 0666
)

// Creates a new file backend for the maxid
func MakeFileMaxIdBackend(basePath string) MaxIdBackend {
	return &fileMaxIdBackend{
		basePath: basePath,
	}
}

// IMPLEMENTATION
// =============

type fileMaxIdBackend struct {
	// The path where we'll save the maxid files
	basePath string
}

// gets the file name of a tables maxid file
func (m *fileMaxIdBackend) getFileName(tableName string) string {
	return filepath.Join(m.basePath, PALETTE_BASE_FOLDER, SanitizeName(tableName))
}

func (m *fileMaxIdBackend) SaveMaxId(tableName, maxid string) error {
	fileName := m.getFileName(tableName)
	log.Debugf("Writing maxid: table=%s file=%s maxid=%s", tableName, fileName, maxid)

	// create the output file path
	if err := os.MkdirAll(filepath.Dir(fileName), OUTPUT_DEFAULT_DIRMODE); err != nil {
		return err
	}

	return ioutil.WriteFile(fileName, []byte(maxid), maxid_backend_default_filemode)
}

func (m *fileMaxIdBackend) GetMaxId(tableName string) (string, error) {
	fileName := m.getFileName(tableName)

	log.Debugf("Getting maxid for table: table=%s file=%s", tableName, fileName)

	contents, err := ioutil.ReadFile(fileName)
	if err != nil {
		return "", err
	}

	log.Infof("Got maxid for table: table=%s maxid=%s", tableName, contents)

	return string(contents), nil
}
