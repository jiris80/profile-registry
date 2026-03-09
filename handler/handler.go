package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jiris80/profile-registry/model"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Handler {
	return &Handler{db: db}
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

	if req.ExternalID == "" || req.Name == "" || req.Email == "" || req.DateOfBirth == "" {
		http.Error(w, "all fields are required", http.StatusBadRequest)
		return
	}

	dob, err := time.Parse(time.RFC3339, req.DateOfBirth)
	if err != nil {
		http.Error(w, "invalid date_of_birth: expected RFC3339 format", http.StatusBadRequest)
		return
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(record)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

// isUniqueConstraintError checks for Postgres unique constraint violation (error code 23505).
func isUniqueConstraintError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
