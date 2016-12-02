package insight_server

import (
	"encoding/json"
	"net/http"
	"time"

	log "github.com/palette-software/insight-tester/common/logging"
)

// Map of hostname => lastContact time
var agents map[string]string = make(map[string]string)

func removeExpiredAgents() {
	for hostname, lastContactString := range agents {
		lastContact, err := time.Parse(time.RFC3339, lastContactString)
		if err == nil {
			if time.Now().Sub(lastContact) > 24*time.Hour {
				delete(agents, hostname)
			}
		}
	}
}

func checkForConfigs() {
	for hostname := range agents {
		if !DoesConfigExist(hostname) {
			askForConfig()
			return
		}
	}
}

func askForConfig() {
	AddCommand("PUT-CONFIG")
}

func AgentHeartbeat(hostname string) {
	agents[hostname] = time.Now().UTC().Format(time.RFC3339)
	checkForConfigs()
}

func AgentListHandler(w http.ResponseWriter, req *http.Request) {
	removeExpiredAgents()
	if err := json.NewEncoder(w).Encode(agents); err != nil {
		log.Error("Error encoding command json for http", err)
		WriteResponse(w, http.StatusInternalServerError, "", req)
		return
	}
}
