package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jiris80/profile-registry/db"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestIntegration(t *testing.T) {
	ctx := context.Background()

	// Start a real PostgreSQL container
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		postgres.WithDatabase("profile_registry"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	defer pgContainer.Terminate(ctx)

	// Point the app at the container
	host, err := pgContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}
	mappedPort, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get mapped port: %v", err)
	}

	os.Setenv("DB_HOST", host)
	os.Setenv("DB_PORT", mappedPort.Port())
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASSWORD", "postgres")
	os.Setenv("DB_NAME", "profile_registry")

	// Connect and migrate — same path as production startup
	database := db.Connect()
	srv := httptest.NewServer(newServer(database))
	defer srv.Close()

	externalID := "550e8400-e29b-41d4-a716-446655440000"

	t.Run("POST /save stores a record", func(t *testing.T) {
		body := fmt.Sprintf(`{
			"external_id": "%s",
			"name": "Jane Doe",
			"email": "jane@example.com",
			"date_of_birth": "1990-05-15T00:00:00Z"
		}`, externalID)

		resp, err := http.Post(srv.URL+"/save", "application/json", bytes.NewBufferString(body))
		if err != nil {
			t.Fatalf("POST /save failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("expected 201, got %d", resp.StatusCode)
		}

		var result map[string]any
		json.NewDecoder(resp.Body).Decode(&result)

		if result["external_id"] != externalID {
			t.Errorf("unexpected external_id: %v", result["external_id"])
		}
		if result["name"] != "Jane Doe" {
			t.Errorf("unexpected name: %v", result["name"])
		}
	})

	t.Run("GET /{id} retrieves the record", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/" + externalID)
		if err != nil {
			t.Fatalf("GET /{id} failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}

		var result map[string]any
		json.NewDecoder(resp.Body).Decode(&result)

		if result["external_id"] != externalID {
			t.Errorf("unexpected external_id: %v", result["external_id"])
		}
		if result["email"] != "jane@example.com" {
			t.Errorf("unexpected email: %v", result["email"])
		}
	})

	t.Run("GET /{id} returns 404 for unknown id", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/does-not-exist")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404, got %d", resp.StatusCode)
		}
	})

	t.Run("POST /save returns 409 on duplicate external_id", func(t *testing.T) {
		body := fmt.Sprintf(`{
			"external_id": "%s",
			"name": "Jane Doe",
			"email": "jane@example.com",
			"date_of_birth": "1990-05-15T00:00:00Z"
		}`, externalID)

		resp, err := http.Post(srv.URL+"/save", "application/json", bytes.NewBufferString(body))
		if err != nil {
			t.Fatalf("POST /save failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusConflict {
			t.Errorf("expected 409, got %d", resp.StatusCode)
		}
	})
}
