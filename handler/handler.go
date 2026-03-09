package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jiris80/profile-registry/model"
	"gorm.io/gorm"
)

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)


type Handler struct {
	db            *gorm.DB
	validateInput bool
}

type Option func(*Handler)

func WithValidation(enabled bool) Option {
	return func(h *Handler) { h.validateInput = enabled }
}

func New(db *gorm.DB, opts ...Option) *Handler {
	h := &Handler{db: db, validateInput: true}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

type saveRequest struct {
	ExternalID  string `json:"external_id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	DateOfBirth string `json:"date_of_birth"`
}

func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	var req saveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var dob time.Time
	if h.validateInput {
		if req.ExternalID == "" || req.Name == "" || req.Email == "" || req.DateOfBirth == "" {
			http.Error(w, "all fields are required", http.StatusBadRequest)
			return
		}

		if !uuidRegex.MatchString(req.ExternalID) {
			http.Error(w, "external_id must be a valid UUID", http.StatusBadRequest)
			return
		}

		if _, err := mail.ParseAddress(req.Email); err != nil {
			http.Error(w, "email is not valid", http.StatusBadRequest)
			return
		}

		parsed, err := time.Parse(time.RFC3339, req.DateOfBirth)
		if err != nil {
			http.Error(w, "invalid date_of_birth: expected RFC3339 format", http.StatusBadRequest)
			return
		}
		dob = parsed
	} else {
		dob, _ = time.Parse(time.RFC3339, req.DateOfBirth)
	}

	record := model.Record{
		ExternalID:  req.ExternalID,
		Name:        req.Name,
		Email:       req.Email,
		DateOfBirth: dob,
	}

	if result := h.db.WithContext(r.Context()).Create(&record); result.Error != nil {
		if isUniqueConstraintError(result.Error) {
			http.Error(w, "record with this external_id already exists", http.StatusConflict)
			return
		}
		slog.Error("failed to save record", "error", result.Error, "external_id", req.ExternalID)
		http.Error(w, "failed to save record", http.StatusInternalServerError)
		return
	}

	buf, err := json.Marshal(record)
	if err != nil {
		slog.Error("failed to encode response", "error", err, "external_id", req.ExternalID)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(buf)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	externalID := r.PathValue("id")

	var record model.Record
	result := h.db.WithContext(r.Context()).Where("external_id = ?", externalID).First(&record)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			http.Error(w, "record not found", http.StatusNotFound)
		} else {
			slog.Error("failed to retrieve record", "error", result.Error, "external_id", externalID)
			http.Error(w, "failed to retrieve record", http.StatusInternalServerError)
		}
		return
	}

	buf, err := json.Marshal(record)
	if err != nil {
		slog.Error("failed to encode response", "error", err, "external_id", externalID)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

// isUniqueConstraintError checks for Postgres unique constraint violation (error code 23505).
func isUniqueConstraintError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
