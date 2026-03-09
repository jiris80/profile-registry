# profile-registry

A simple microservice for storing and retrieving profile records, built in Go with PostgreSQL.

## Quick Start

Requires Docker.

```bash
docker compose up --build
```

Service will be available at `http://localhost:8080`.

## Building the binary locally

Requires Go 1.21+ and a running PostgreSQL instance.

```bash
go build -o profile-registry .
```

Then run it (override env vars as needed):

```bash
DB_HOST=localhost DB_USER=postgres DB_PASSWORD=postgres DB_NAME=profile_registry ./profile-registry
```

## Test with curl

```bash
# Store a profile
curl -X POST http://localhost:8080/save \
  -H "Content-Type: application/json" \
  -d '{
    "external_id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Jane Doe",
    "email": "jane@example.com",
    "date_of_birth": "1990-05-15T00:00:00Z"
  }'

# Retrieve it
curl http://localhost:8080/550e8400-e29b-41d4-a716-446655440000
```

## Endpoints

### Store a profile
```
POST /save
```
Request body:
```json
{
  "external_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Jane Doe",
  "email": "jane@example.com",
  "date_of_birth": "1990-05-15T00:00:00Z"
}
```

| Status | Meaning |
|--------|---------|
| 201 | Record created |
| 400 | Invalid or missing fields |
| 409 | Duplicate `external_id` |
| 500 | Internal error |

### Retrieve a profile
```
GET /{external_id}
```

| Status | Meaning |
|--------|---------|
| 200 | Record returned |
| 404 | Not found |
| 500 | Internal error |

## Running Tests

Requires Docker (integration tests use testcontainers to spin up a real PostgreSQL instance and run the compiled binary).

```bash
go test -v -timeout 120s ./...
```

To run only integration tests:
```bash
go test -v -run TestIntegration -timeout 120s
```

## Configuration

| Environment variable | Default            | Description               |
|----------------------|--------------------|---------------------------|
| `DB_HOST`            | `localhost`        | Postgres host             |
| `DB_PORT`            | `5432`             | Postgres port             |
| `DB_USER`            | `postgres`        | Postgres user             |
| `DB_PASSWORD`        | `postgres`        | Postgres password         |
| `DB_NAME`            | `profile_registry` | Database name             |
| `SERVER_PORT`        | `8080`             | HTTP server port          |
| `VALIDATION_ENABLED` | `true`             | Set to `false` to disable input validation |
