package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
)

// Simple integration test that exercises the GraphQL API (DB → API).
// It posts a users query and verifies we get at least one user and tasks back.

func TestGraphQLUsers(t *testing.T) {
	endpoint := os.Getenv("GRAPHQL_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8081/graphql"
	}

	query := `{"query":"{ users { id name email tasks { id title status dueDate tags } } }"}`
	var lastErr error
	// wait up to 10s for the endpoint to be ready (useful in CI or when starting services)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Post(endpoint, "application/json", bytes.NewBufferString(query))
		if err != nil {
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}

		var body struct {
			Data   map[string]any   `json:"data"`
			Errors []map[string]any `json:"errors"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			lastErr = err
			break
		}
		if len(body.Errors) > 0 {
			lastErr = err
			break
		}
		usersI, ok := body.Data["users"]
		if !ok {
			t.Fatalf("response missing users field")
		}
		users, ok := usersI.([]any)
		if !ok {
			t.Fatalf("users has unexpected type: %T", usersI)
		}
		if len(users) == 0 {
			t.Fatalf("expected >=1 user, got 0")
		}
		// inspect the first user to ensure tasks array exists
		first := users[0].(map[string]any)
		if _, ok := first["id"]; !ok {
			t.Fatalf("first user missing id")
		}
		if tasksI, ok := first["tasks"]; ok {
			if tasks, ok := tasksI.([]any); ok {
				// ensure tasks are objects with title
				if len(tasks) > 0 {
					task := tasks[0].(map[string]any)
					if _, ok := task["title"]; !ok {
						t.Fatalf("task missing title")
					}
				}
			}
		}
		// success
		return
	}
	if lastErr != nil {
		t.Fatalf("failed to query endpoint %s: %v", endpoint, lastErr)
	}
	t.Fatalf("timeout waiting for endpoint %s", endpoint)
}
