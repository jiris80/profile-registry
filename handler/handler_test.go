package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSave_ValidationOn(t *testing.T) {
	h := New(nil, WithValidation(true))

	cases := []struct {
		name   string
		body   string
		status int
	}{
		{
			name:   "missing fields",
			body:   `{"external_id":"","name":"","email":"","date_of_birth":""}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "invalid UUID",
			body:   `{"external_id":"not-a-uuid","name":"Jane","email":"jane@example.com","date_of_birth":"1990-05-15T00:00:00Z"}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "invalid email",
			body:   `{"external_id":"550e8400-e29b-41d4-a716-446655440000","name":"Jane","email":"not-an-email","date_of_birth":"1990-05-15T00:00:00Z"}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "invalid date format",
			body:   `{"external_id":"550e8400-e29b-41d4-a716-446655440000","name":"Jane","email":"jane@example.com","date_of_birth":"15-05-1990"}`,
			status: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/save", strings.NewReader(tc.body))
			rw := httptest.NewRecorder()
			h.Save(rw, req)
			if rw.Code != tc.status {
				t.Errorf("expected %d, got %d", tc.status, rw.Code)
			}
		})
	}
}

func TestSave_ValidationOff(t *testing.T) {
	h := New(nil, WithValidation(false))

	cases := []struct {
		name string
		body string
	}{
		{
			name: "missing fields pass through",
			body: `{"external_id":"","name":"","email":"","date_of_birth":""}`,
		},
		{
			name: "invalid UUID passes through",
			body: `{"external_id":"not-a-uuid","name":"Jane","email":"not-an-email","date_of_birth":"bad-date"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/save", strings.NewReader(tc.body))
			rw := httptest.NewRecorder()

			// nil db will panic after validation — that confirms validation was skipped
			func() {
				defer func() { recover() }()
				h.Save(rw, req)
			}()

			if rw.Code == http.StatusBadRequest {
				t.Errorf("expected validation to be skipped, got 400")
			}
		})
	}
}
