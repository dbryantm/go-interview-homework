Summary

- Fixed seeder fallback DSN to match docker-compose.
- Minimal GraphQL API exposing Task.dueDate (YYYY-MM-DD).
- Added structured JSON logger and resolver-level timing (wrapResolve).
- Docker: Dockerfile + docker-compose to run Postgres, API, and UI (UI :8080, API :8081).
- Tests: focused table-driven unit tests and an integration test.

Run (from repo root)

Docker (recommended):

  docker compose up -d --build
  - UI: http://localhost:8080
  - API: http://localhost:8081/graphql
  After Postgres is ready, seed the DB:

  go run ./cmd/seed

Local development:

  docker compose up -d
  go run ./cmd/seed
  go run ./cmd/server
  # optionally serve the UI if needed
  python3 -m http.server 8081 --directory web

Notes

- Logger writes structured JSON to stderr; SetLoggerOutput redirects logs for tests.
- wrapResolve logs resolver name, scalar args only, duration_ms, and status (info/error).

Reviewed by: DM
