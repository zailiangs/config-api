package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/zailiangs/config-api/internal/sshconfig"
)

type fakeManager struct {
	result sshconfig.Result
	err    error
	port   int
}

func (f *fakeManager) SetPort(port int) (sshconfig.Result, error) {
	f.port = port
	return f.result, f.err
}

func TestConfigSSHValidatesPort(t *testing.T) {
	tests := []string{"", "abc", "-1", "0", "65536", "22.0", " 22"}
	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/config/ssh?port="+url.QueryEscape(value), nil)
			NewHandler(&fakeManager{}).ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("got status %d", rec.Code)
			}
		})
	}
}

func TestConfigSSHSuccess(t *testing.T) {
	manager := &fakeManager{result: sshconfig.Result{Port: 18822, Changed: true}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/config/ssh?port=18822", nil)
	NewHandler(manager).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || manager.port != 18822 {
		t.Fatalf("status=%d port=%d", rec.Code, manager.port)
	}
	var got response
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if !got.Success || !got.Changed || got.Port != 18822 {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestConfigSSHReportsRollback(t *testing.T) {
	manager := &fakeManager{
		result: sshconfig.Result{Port: 18822, Changed: true, RolledBack: true},
		err:    errors.New("restart failed"),
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/config/ssh?port=18822", nil)
	NewHandler(manager).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("got status %d", rec.Code)
	}
	var got response
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if !got.RolledBack || got.Error == "" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	NewHandler(&fakeManager{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d", rec.Code)
	}
}
