package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

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

	if result := h.db.Create(&record); result.Error != nil {
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
	result := h.db.Where("external_id = ?", externalID).First(&record)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			http.Error(w, "record not found", http.StatusNotFound)
		} else {
			http.Error(w, "failed to retrieve record", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}
