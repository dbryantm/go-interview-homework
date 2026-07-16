1. The bug in the seed script

- The SQL insert used backticks around column names (MySQL-style) which caused Postgres syntax errors. I fixed the query to use standard identifiers and verified `go run ./cmd/seed` populates the DB.

2. Library / structure choices

- I did not add a GraphQL server in this iteration to keep scope small. If implemented, I'd choose `github.com/99designs/gqlgen` for schema-first generation and type safety, with a `pkg/` data layer and `cmd/server` runner.

3. The new field

- Not implemented in this pass. With more time I'd expose `dueDate` end-to-end (DB → GraphQL → UI) since it's a simple, useful field.

4. Tradeoffs / what I skipped

- Skipped building the GraphQL server and resolver tests to prioritize a clean seed fix, module correctness, and repo hygiene. Next steps: implement `cmd/server`, wire resolvers to Postgres, add CORS, and update `web/app.js` to request the new field.

5. How to run everything (from a fresh clone)

```bash
# 1) Start Postgres
docker compose up -d

# 2) Populate DB
go run ./cmd/seed

# 3) (If implemented) Start API
go run ./cmd/server

# 4) Open the UI
# Serve the web directory if `file://` is blocked:
python3 -m http.server 8081 --directory web
xdg-open http://localhost:8081
```

Reviewed by: DB