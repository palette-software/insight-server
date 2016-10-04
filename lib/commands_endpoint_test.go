package insight_server

import (
	"encoding/json"
	"fmt"
	tassert "github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAddCommandHandler_NoParams(t *testing.T) {
	handler := NewAddCommandHandler()
	req, _ := http.NewRequest("PUT", "/api/v1/config", nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	tassert.Equal(t, rr.Code, http.StatusBadRequest, fmt.Sprintf("command put endpoint should return BadRequest if called without command argument: %s", rr.Body.String()))
}

func TestAddCommandHandler_WithCommand(t *testing.T) {
	handler := NewAddCommandHandler()
	req, _ := http.NewRequest("PUT", "/api/v1/config", strings.NewReader("command=cmd"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	tassert.Equal(t, rr.Code, http.StatusOK, fmt.Sprintf("Add command handler failed: %s", rr.Body.String()))
	var ac AgentCommand
	err := json.Unmarshal(rr.Body.Bytes(), &ac)
	tassert.Nil(t, err, fmt.Sprintf("Error while decoding json: %s", err))
	tassert.Equal(t, ac.Cmd, "cmd", "Returned command parameter should be the same as was sent up.")
}

func TestAddCommandHandler_WithGetCommand(t *testing.T) {
	putHandler := NewAddCommandHandler()
	putReq, _ := http.NewRequest("PUT", "/api/v1/config", strings.NewReader("command=newcmd"))
	putReq.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	putrr := httptest.NewRecorder()
	putHandler.ServeHTTP(putrr, putReq)

	getReq, _ := http.NewRequest("GET", "/api/v1/config", nil)
	getReq.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	getHandler := NewGetCommandHandler()
	getrr := httptest.NewRecorder()
	getHandler.ServeHTTP(getrr, getReq)
	tassert.Equal(t, getrr.Code, http.StatusOK, fmt.Sprintf("Get command handler failed when called without parameters: %s", getrr.Body.String()))
	var ac AgentCommand
	err := json.Unmarshal(getrr.Body.Bytes(), &ac)
	tassert.Nil(t, err, fmt.Sprintf("Error while decoding json: %s ", err))
	tassert.Equal(t, ac.Cmd, "newcmd", "Returned command parameter should be the same as was sent up.")
}
