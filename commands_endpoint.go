package insight_server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"time"
)

// An agent command with a timestamp
type AgentCommand struct {
	Ts  string `json:"ts"`
	Cmd string `json:"command"`
}

type CommandsEndpoint interface {
	AddCommand(tenant, command string) *AgentCommand
	// Returns the last command for a tenants agents or nil
	// if we cannot find a command for it
	GetCommand(tenant string) *AgentCommand
}

// IMPLEMENTATION
// ==============

////////////////////////////////////////////////

type fileCommandsEndpoint struct {
	baseDir string

	// The last commands issued by tenant name
	lastCommands map[string]AgentCommand
}

func NewFileCommandsEndpoint(baseDir string) CommandsEndpoint {
	log.Printf("[commands] Using '%s' as commands file basedir", baseDir)
	return &fileCommandsEndpoint{
		baseDir:      baseDir,
		lastCommands: map[string]AgentCommand{},
	}
}

func (f *fileCommandsEndpoint) AddCommand(tenant, command string) *AgentCommand {
	// create the command
	cmd := AgentCommand{
		Ts:  time.Now().UTC().Format(time.RFC3339),
		Cmd: command,
	}
	// store it
	f.lastCommands[tenant] = cmd

	if err := f.saveLastCommands(); err != nil {
		log.Printf("[commands] ERROR while saving commands list: %v", err)
	}

	// return the address of the command, but dont touch lastCommands
	return &cmd
}

func (f *fileCommandsEndpoint) GetCommand(tenant string) *AgentCommand {
	cmd, hasCmd := f.lastCommands[tenant]
	// if no commands available, return an empty one
	if !hasCmd {
		return nil
	}
	return &cmd
}

////////////////////////////////////////////////

// the name of the commands file we serialize to
const CommandsFileName = "commands.json"

func (f *fileCommandsEndpoint) saveFileName() string {
	return path.Join(f.baseDir, CommandsFileName)
}

// Saves the list of current commands with timestamps
func (f *fileCommandsEndpoint) saveLastCommands() error {
	tmpFile, err := ioutil.TempFile(f.baseDir, "commands-list")
	if err != nil {
		return fmt.Errorf("Error opening temp file: %v", err)
	}
	defer tmpFile.Close()

	// try to save as json
	if err := json.NewEncoder(tmpFile).Encode(f.lastCommands); err != nil {
		return fmt.Errorf("Error while serialzing commands list to JSON: %v", err)
	}

	// close the temp file so we flush
	tmpFile.Close()

	// move to its final destination
	if err := os.Rename(tmpFile.Name(), f.saveFileName()); err != nil {
		return fmt.Errorf("Error while moving commands file '%s' to '%s': %v", tmpFile.Name(), f.saveFileName(), err)
	}

	log.Printf("[commands] Moved temporary commands file '%s' to '%s'", tmpFile.Name(), f.saveFileName())

	return nil
}

// Loads the list of commands back from the serialized form
func (f *fileCommandsEndpoint) loadLastCommands() error {
	// open the commands file
	cmdFile, err := os.Open(f.saveFileName())
	if err != nil {
		return fmt.Errorf("Error opening command file '%s': %v", f.saveFileName(), err)
	}
	defer cmdFile.Close()

	// decode the commands list
	if err := json.NewDecoder(cmdFile).Decode(f.lastCommands); err != nil {
		// clean the commands list if load failed
		f.lastCommands = map[string]AgentCommand{}
		return fmt.Errorf("Error deserializing commands json: %v", err)
	}

	// log some status
	log.Printf("[commands] Loaded commmands list from commands file '%s': %v", f.saveFileName(), f.lastCommands)

	return nil
}

////////////////////////////////////////////////

func NewAddCommandHandler(cep CommandsEndpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenant, err := getUrlParam(r.URL, "tenant")
		if err != nil {
			writeResponse(w, http.StatusBadRequest, "No 'tenant' parameter given")
			return
		}

		command, err := getUrlParam(r.URL, "command")
		if err != nil {
			writeResponse(w, http.StatusBadRequest, "No 'command' parameter given")
			return
		}

		// add the command to the backend
		cmd := cep.AddCommand(tenant, command)
		if err := json.NewEncoder(w).Encode(cmd); err != nil {
			// log the error
			log.Printf("[commands] Error encoding command json for http: %v", err)
			// but hide this fact from the outside world
			writeResponse(w, http.StatusInternalServerError, "")
			return
		}

		// the json should have been rendered at this point
	}
}

func NewGetCommandHandler(cep CommandsEndpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		tenant, err := getUrlParam(r.URL, "tenant")
		if err != nil {
			writeResponse(w, http.StatusBadRequest, "No 'tenant' parameter given")
			return
		}
		// add the command to the backend
		cmd := cep.GetCommand(tenant)

		// if we dont have the command
		if cmd == nil {
			writeResponse(w, http.StatusNoContent, "")
			return
		}

		if err := json.NewEncoder(w).Encode(cmd); err != nil {
			// log the error
			log.Printf("[commands] Error encoding command json for http: %v", err)
			// but hide this fact from the outside world
			writeResponse(w, http.StatusInternalServerError, "")
			return
		}

		// the json should have been rendered at this point
	}
}
