package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

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

	host, err := pgContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}
	mappedPort, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get mapped port: %v", err)
	}

	// Build the binary
	bin := filepath.Join(t.TempDir(), "profile-registry-test")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("failed to build binary: %v", err)
	}

	// Start the binary with env vars pointing at the container
	port := "18080"
	srv := exec.Command(bin)
	srv.Env = []string{
		"SERVER_PORT=" + port,
		"DB_HOST=" + host,
		"DB_PORT=" + mappedPort.Port(),
		"DB_USER=postgres",
		"DB_PASSWORD=postgres",
		"DB_NAME=profile_registry",
	}
	srv.Stdout = os.Stdout
	srv.Stderr = os.Stderr
	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start server binary: %v", err)
	}
	t.Cleanup(func() { srv.Process.Kill() })

	baseURL := "http://localhost:" + port
	waitForReady(t, baseURL)

	externalID := "550e8400-e29b-41d4-a716-446655440000"

	t.Run("POST /save stores a record", func(t *testing.T) {
		body := fmt.Sprintf(`{
			"external_id": "%s",
			"name": "Jane Doe",
			"email": "jane@example.com",
			"date_of_birth": "1990-05-15T00:00:00Z"
		}`, externalID)

		resp, err := http.Post(baseURL+"/save", "application/json", bytes.NewBufferString(body))
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
		resp, err := http.Get(baseURL + "/" + externalID)
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
		resp, err := http.Get(baseURL + "/does-not-exist")
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

		resp, err := http.Post(baseURL+"/save", "application/json", bytes.NewBufferString(body))
		if err != nil {
			t.Fatalf("POST /save failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusConflict {
			t.Errorf("expected 409, got %d", resp.StatusCode)
		}
	})
}

func waitForReady(t *testing.T, baseURL string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/nonexistent")
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("server did not become ready within 10s")
}
