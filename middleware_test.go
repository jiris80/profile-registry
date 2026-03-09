package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoggingMiddleware_CapturesStatusAndFields(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	h := loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/some-id", nil)
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)

	var entry map[string]any
	if err := json.NewDecoder(&buf).Decode(&entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}

	if entry["method"] != http.MethodGet {
		t.Errorf("expected method GET, got %v", entry["method"])
	}
	if entry["path"] != "/some-id" {
		t.Errorf("expected path /some-id, got %v", entry["path"])
	}
	if entry["status"] != float64(http.StatusNotFound) {
		t.Errorf("expected status 404, got %v", entry["status"])
	}
	if _, ok := entry["duration_ms"]; !ok {
		t.Error("expected duration_ms field in log entry")
	}
	if entry["request_id"] == "" {
		t.Error("expected non-empty request_id in log entry")
	}
}

func TestLoggingMiddleware_UniqueRequestIDs(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	h := loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/some-id", nil)
		h.ServeHTTP(httptest.NewRecorder(), req)
	}

	ids := map[string]bool{}
	decoder := json.NewDecoder(&buf)
	for decoder.More() {
		var entry map[string]any
		if err := decoder.Decode(&entry); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		id, _ := entry["request_id"].(string)
		if ids[id] {
			t.Errorf("duplicate request_id: %s", id)
		}
		ids[id] = true
	}
}

func TestRecoveryMiddleware_Returns500OnPanic(t *testing.T) {
	handler := recoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	}))

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	if rw.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rw.Code)
	}
}

func TestLoggingMiddleware_DefaultsTo200(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	h := loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodPost, "/save", nil)
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)

	var entry map[string]any
	if err := json.NewDecoder(&buf).Decode(&entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}

	if entry["status"] != float64(http.StatusOK) {
		t.Errorf("expected status 200, got %v", entry["status"])
	}
}
