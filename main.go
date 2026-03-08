package main

import (
	"log"
	"net/http"
	"os"

	"github.com/jiris80/profile-registry/db"
)

func main() {
	database := db.Connect()

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("starting profile-registry on :%s", port)
	if err := http.ListenAndServe(":"+port, newServer(database)); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
