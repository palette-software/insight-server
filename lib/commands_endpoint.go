package insight_server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	log "github.com/palette-software/insight-tester/common/logging"
)

// An agent command with a timestamp
type AgentCommand struct {
	Ts  string `json:"ts"`
	Cmd string `json:"command"`
}

var baseDir string
var lastCommand AgentCommand

////////////////////////////////////////////////

// the name of the commands file we serialize to
const CommandsFileName = "commands.json"

func saveFileName() string {
	return filepath.Join(baseDir, CommandsFileName)
}

// Saves the list of current commands with timestamps
func saveLastCommands() error {
	tmpFile, err := ioutil.TempFile(baseDir, "commands-list")
	if err != nil {
		return fmt.Errorf("Error opening temp file: %v", err)
	}
	defer tmpFile.Close()

	// try to save as json
	if err := json.NewEncoder(tmpFile).Encode(lastCommand); err != nil {
		return fmt.Errorf("Error while serializing commands list to JSON: %v", err)
	}

	// close the temp file so we flush
	tmpFile.Close()

	// move to its final destination
	if err := os.Rename(tmpFile.Name(), saveFileName()); err != nil {
		return fmt.Errorf("Error while moving commands file '%s' to '%s': %v", tmpFile.Name(), saveFileName(), err)
	}

	return nil
}

func InitCommandEndpoints() {
	cmdFile, err := os.Open(saveFileName())
	if err != nil {
		return
	}
	defer cmdFile.Close()

	// decode the commands list
	if err := json.NewDecoder(cmdFile).Decode(&lastCommand); err != nil {
		// clean the commands list if load failed
		lastCommand = AgentCommand{}
	}
}

func AddCommand(command string) (*AgentCommand, error) {
	cmd := AgentCommand{
		Ts:  time.Now().UTC().Format(time.RFC3339),
		Cmd: command,
	}
	lastCommand = cmd
	if err := saveLastCommands(); err != nil {
		return nil, err
	}
	return &lastCommand, nil
}

func AddCommandHandler(w http.ResponseWriter, r *http.Request) {
	command := r.PostFormValue("command")
	if command == "" {
		WriteResponse(w, http.StatusBadRequest, "No 'command' parameter given", r)
		return
	}

	cmd, err := AddCommand(command)
	if err != nil {
		log.Error("Error while saving commands list.", err)
		WriteResponse(w, http.StatusInternalServerError, "", r)
	}

	if err := json.NewEncoder(w).Encode(cmd); err != nil {
		log.Error("Error encoding commands for json.", err)
		WriteResponse(w, http.StatusInternalServerError, "", r)
		return
	}

	// the json should have been rendered at this point
}

func NewGetCommandHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// add the command to the backend
		cmd := &lastCommand

		// if we dont have the command
		if cmd.Cmd == "" || cmd.Ts == "" {
			WriteResponse(w, http.StatusNoContent, "", r)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(cmd); err != nil {
			// log the error
			log.Error("Error encoding command json for http.", err)
			// but hide this fact from the outside world
			WriteResponse(w, http.StatusInternalServerError, "", r)
			return
		}
		// the json should have been rendered at this point
	}
}
