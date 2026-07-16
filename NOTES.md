1. The bug in the seed script

The seeder failed to connect with a sensible default DSN. The default connection string was invalid/masked; set the fallback to match docker-compose: postgres://admin:todo@localhost:5432/homework?sslmode=disable. I found this by attempting to run the seeder (go run ./cmd/seed) and inspecting cmd/seed/main.go.

2. Library / structure choices

- GraphQL library: github.com/graphql-go/graphql + github.com/graphql-go/handler. Chosen because it is minimal, requires no code generation, and is fast to wire up for a small exercise (fewer steps to a working API). This minimizes setup time.
- Structure: small cmd/ directories: cmd/seed and cmd/server. Data-access helpers are in cmd/server/main.go to keep the change minimal for the exercise; in a larger project they'd be refactored into packages.

3. The new field

- Exposed field: dueDate on Task (string, formatted as YYYY-MM-DD). Rationale: exposes a real DB field end-to-end and is trivial to display in the provided UI.

4. Tradeoffs / what was skipped

- Skipped: more thorough package layout (separate data layer and resolvers), tests, structured JSON logger, resolver timing, and Dockerizing the API. These are straightforward extensions but would add time.

5. How to run (from a fresh clone)

Option A — Run API locally (fastest during development)

1. Start Postgres (docker-compose provides the DB and schema):

   docker compose up -d

2. Seed the database (requires Go 1.25+ installed locally):

   go run ./cmd/seed

3. Run the API server locally (serves GraphQL on :8080):

   go run ./cmd/server

4. Serve the UI (if your browser blocks file:// → http:// requests):

   python3 -m http.server 8081 --directory web

   Open http://localhost:8081 in your browser. The web UI expects the API at http://localhost:8080/graphql (the local server).

Option B — Run everything via Docker (recommended for "one command" runs)

1. From the repo root, bring up the full stack (Postgres, API, and static web UI). The compose file serves the UI on host port 8080 and the API on host port 8081:

   docker compose up -d --build

   - UI will be available at: http://localhost:8080
   - GraphQL API will be available at: http://localhost:8081/graphql

2. Seed the database (from the host) after postgres is up:

   go run ./cmd/seed

3. Open http://localhost:8080 in a browser — the web UI will fetch GraphQL data from http://localhost:8081/graphql (the compose API).

Notes

- This compose setup lets you run everything without any python/http-server step.
- To stop everything started by compose:

  docker compose down


Reviewed By: DM
