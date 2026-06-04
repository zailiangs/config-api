package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/zailiangs/config-api/internal/sshconfig"
)

type PortManager interface {
	SetPort(port int) (sshconfig.Result, error)
}

type response struct {
	Success    bool   `json:"success"`
	Port       int    `json:"port,omitempty"`
	Changed    bool   `json:"changed"`
	RolledBack bool   `json:"rolled_back"`
	Error      string `json:"error,omitempty"`
}

func NewHandler(manager PortManager) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, response{Success: true})
	})
	mux.HandleFunc("GET /config/ssh", func(w http.ResponseWriter, r *http.Request) {
		rawPort := r.URL.Query().Get("port")
		port, err := parsePort(rawPort)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, response{Success: false, Error: err.Error()})
			return
		}

		result, err := manager.SetPort(port)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, response{
				Success: false, Port: port, Changed: result.Changed,
				RolledBack: result.RolledBack, Error: err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, response{
			Success: true, Port: port, Changed: result.Changed,
			RolledBack: result.RolledBack,
		})
	})
	return mux
}

func parsePort(value string) (int, error) {
	if value == "" {
		return 0, fmt.Errorf("port query parameter is required")
	}
	if strings.Trim(value, "0123456789") != "" {
		return 0, fmt.Errorf("port must be a decimal integer between 1 and 65535")
	}
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("port must be a decimal integer between 1 and 65535")
	}
	return port, nil
}

func writeJSON(w http.ResponseWriter, status int, payload response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
