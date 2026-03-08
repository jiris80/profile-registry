package main

import (
	"net/http"

	"github.com/jiris80/profile-registry/handler"
	"gorm.io/gorm"
)

func newServer(db *gorm.DB) http.Handler {
	h := handler.New(db)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /save", h.Save)
	mux.HandleFunc("GET /{id}", h.Get)

	return mux
}
