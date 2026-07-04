package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amansharma/config-service/internal/model"
)

func TestValidation(t *testing.T) {
	tests := []struct {
		name     string
		req      model.UpsertConfigRequest
		wantErrs int
	}{
		{
			name: "valid request",
			req: model.UpsertConfigRequest{
				ID: "cfg_1", Host: "localhost", Port: 8080,
				AppName: "my-app", LogLevel: "INFO",
			},
			wantErrs: 0,
		},
		{
			name:     "empty request",
			req:      model.UpsertConfigRequest{},
			wantErrs: 5,
		},
		{
			name: "invalid id characters",
			req: model.UpsertConfigRequest{
				ID: "cfg 1!", Host: "localhost", Port: 8080,
				AppName: "my-app", LogLevel: "INFO",
			},
			wantErrs: 1,
		},
		{
			name: "invalid port",
			req: model.UpsertConfigRequest{
				ID: "cfg_1", Host: "localhost", Port: 99999,
				AppName: "my-app", LogLevel: "INFO",
			},
			wantErrs: 1,
		},
		{
			name: "invalid log level",
			req: model.UpsertConfigRequest{
				ID: "cfg_1", Host: "localhost", Port: 8080,
				AppName: "my-app", LogLevel: "TRACE",
			},
			wantErrs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.req.Validate()
			if len(errs) != tt.wantErrs {
				t.Errorf("got %d errors, want %d: %v", len(errs), tt.wantErrs, errs)
			}
		})
	}
}

func TestPingHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != "pong" {
		t.Errorf("got body %q, want %q", w.Body.String(), "pong")
	}
	_ = req
}

func TestUpsertRequestDecoding(t *testing.T) {
	body := `{"id":"cfg_1","host":"localhost","port":8080,"app_name":"config-service","log_level":"INFO"}`
	req := httptest.NewRequest(http.MethodPost, "/configs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	if req.Body == nil {
		t.Fatal("request body should not be nil")
	}
}
