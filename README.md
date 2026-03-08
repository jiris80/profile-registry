# profile-registry

A simple microservice for storing and retrieving profile records, built in Go with PostgreSQL.

## Endpoints

### Store a profile
```
POST /save
```
```json
{
  "external_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Jane Doe",
  "email": "jane@example.com",
  "date_of_birth": "1990-05-15T00:00:00Z"
}
```

### Retrieve a profile
```
GET /{external_id}
```

## Running

Requires Docker.

```bash
docker compose up --build
```

Service will be available at `http://localhost:8080`.

## Example

```bash
# Save a profile
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

## Configuration

| Environment variable | Default           | Description        |
|----------------------|-------------------|--------------------|
| `DB_HOST`            | `localhost`       | Postgres host      |
| `DB_PORT`            | `5432`            | Postgres port      |
| `DB_USER`            | `postgres`        | Postgres user      |
| `DB_PASSWORD`        | `postgres`        | Postgres password  |
| `DB_NAME`            | `profile_registry`| Database name      |
| `SERVER_PORT`        | `8080`            | HTTP server port   |

## Testing

Requires Docker (integration tests use testcontainers).

```bash
go test -v -run TestIntegration -timeout 120s
```
