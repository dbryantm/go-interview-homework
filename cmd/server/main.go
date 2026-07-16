package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/graphql-go/graphql"
	ghandler "github.com/graphql-go/handler"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// defaultDSN is the fallback Postgres DSN used when DATABASE_URL is not set.
const defaultDSN = "postgres://admin:todo@localhost:5432/homework?sslmode=disable"

// User represents an application user.
//
// Public fields are included in the GraphQL API responses.
type User struct {
	// ID is the unique numeric identifier for the user.
	ID    int64  `json:"id"`
	// Email is the user's contact email address.
	Email string `json:"email"`
	// Name is the display name for the user.
	Name  string `json:"name"`
}

// Task represents a todo item owned by a User.
//
// DueDate may be nil when a task has no deadline. Tags is the list of
// associated string labels.
type Task struct {
	// ID is the unique numeric identifier for the task.
	ID          int64      `json:"id"`
	// UserID is the owner user's numeric ID.
	UserID      int64      `json:"userId"`
	// Title is the short title for the task.
	Title       string     `json:"title"`
	// Description is an optional longer description for the task.
	Description *string    `json:"description"`
	// Status is one of: pending, in_progress, done.
	Status      string     `json:"status"`
	// DueDate is optional and, when present, indicates the task deadline.
	DueDate     *time.Time `json:"dueDate"`
	// Tags are the labels associated with the task.
	Tags        []string   `json:"tags"`
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = defaultDSN
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	schema := buildSchema(db)

	h := ghandler.New(&ghandler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	})

	hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})

	http.Handle("/graphql", hf)

	addr := ":8080"
	fmt.Printf("listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func buildSchema(db *sql.DB) graphql.Schema {
	// Enums
	statusEnum := graphql.NewEnum(graphql.EnumConfig{
		Name: "TaskStatus",
		Values: graphql.EnumValueConfigMap{
			"PENDING":     &graphql.EnumValueConfig{Value: "pending"},
			"IN_PROGRESS": &graphql.EnumValueConfig{Value: "in_progress"},
			"DONE":        &graphql.EnumValueConfig{Value: "done"},
		},
	})

	// Forward declarations
	var taskType *graphql.Object
	var userType *graphql.Object

	// Task type
	taskType = graphql.NewObject(graphql.ObjectConfig{
		Name: "Task",
		Fields: graphql.Fields{
			"id":          &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"user":        &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"title":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"description": &graphql.Field{Type: graphql.String},
			"status":      &graphql.Field{Type: graphql.NewNonNull(statusEnum)},
			"dueDate": &graphql.Field{
				Type: graphql.String,
				// Return the date in YYYY-MM-DD format for the UI; nil if absent
				Resolve: func(p graphql.ResolveParams) (any, error) {
					task := p.Source.(Task)
					if task.DueDate == nil {
						return nil, nil
					}
					return task.DueDate.Format("2006-01-02"), nil
				},
			},
			"tags": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
		},
	})

	// User type
	userType = graphql.NewObject(graphql.ObjectConfig{
		Name: "User",
		Fields: graphql.Fields{
			"id":    &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"email": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"name":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"tasks": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(taskType))),
				Args: graphql.FieldConfigArgument{
					"status": &graphql.ArgumentConfig{Type: statusEnum},
				},
				Resolve: func(p graphql.ResolveParams) (any, error) {
					userObj := p.Source.(User)
					var status *string
					if s, ok := p.Args["status"].(string); ok {
						status = &s
					}
					return fetchTasksForUser(p.Context, db, userObj.ID, status)
				},
			},
		},
	})

	// Now fix the 'user' field in taskType to return a User object
	// Because taskType was created earlier, set the field now
	taskType.AddFieldConfig("user", &graphql.Field{
		Type: graphql.NewNonNull(userType),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			task := p.Source.(Task)
			u, err := fetchUserByID(p.Context, db, task.UserID)
			return u, err
		},
	})

	// Root Query
	query := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"user": &graphql.Field{
				Type: userType,
				Args: graphql.FieldConfigArgument{"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)}},
				Resolve: func(p graphql.ResolveParams) (any, error) {
					idStr := p.Args["id"].(string)
					id, err := strconv.ParseInt(idStr, 10, 64)
					if err != nil {
						return nil, nil
					}
					return fetchUserByID(p.Context, db, id)
				},
			},
			"users": &graphql.Field{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(userType))),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return fetchAllUsers(p.Context, db)
				},
			},
			"task": &graphql.Field{
				Type: taskType,
				Args: graphql.FieldConfigArgument{"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)}},
				Resolve: func(p graphql.ResolveParams) (any, error) {
					idStr := p.Args["id"].(string)
					id, err := strconv.ParseInt(idStr, 10, 64)
					if err != nil {
						return nil, nil
					}
					return fetchTaskByID(p.Context, db, id)
				},
			},
			"tasks": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(taskType))),
				Args: graphql.FieldConfigArgument{
					"status": &graphql.ArgumentConfig{Type: statusEnum},
					"userId": &graphql.ArgumentConfig{Type: graphql.ID},
				},
				Resolve: func(p graphql.ResolveParams) (any, error) {
					var status *string
					if s, ok := p.Args["status"].(string); ok {
						status = &s
					}
					var userId *int64
					if uid, ok := p.Args["userId"].(string); ok {
						id, err := strconv.ParseInt(uid, 10, 64)
						if err == nil {
							userId = &id
						}
					}
					return fetchTasks(p.Context, db, status, userId)
				},
			},
		},
	})

	// Root Mutation
	mutation := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"createTask": &graphql.Field{
				Type: taskType,
				Args: graphql.FieldConfigArgument{
					"userId":      &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					"title":       &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"description": &graphql.ArgumentConfig{Type: graphql.String},
					"dueDate":     &graphql.ArgumentConfig{Type: graphql.String},
					"tags":        &graphql.ArgumentConfig{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
				},
				Resolve: func(p graphql.ResolveParams) (any, error) {
					uidStr := p.Args["userId"].(string)
					uid, err := strconv.ParseInt(uidStr, 10, 64)
					if err != nil {
						return nil, fmt.Errorf("invalid userId")
					}
					title := p.Args["title"].(string)
					descPtr := (*string)(nil)
					if d, ok := p.Args["description"].(string); ok {
						descPtr = &d
					}
					var duePtr *time.Time
					if ds, ok := p.Args["dueDate"].(string); ok && ds != "" {
						t, err := time.Parse("2006-01-02", ds)
						if err == nil {
							duePtr = &t
						}
					}
					tags := []string{}
					if ts, ok := p.Args["tags"].([]interface{}); ok {
						for _, it := range ts {
							if s, ok := it.(string); ok {
								tags = append(tags, s)
							}
						}
					}
					return createTask(p.Context, db, uid, title, descPtr, "pending", duePtr, tags)
				},
			},
			"updateTaskStatus": &graphql.Field{
				Type: taskType,
				Args: graphql.FieldConfigArgument{
					"id":     &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					"status": &graphql.ArgumentConfig{Type: graphql.NewNonNull(statusEnum)},
				},
				Resolve: func(p graphql.ResolveParams) (any, error) {
					idStr := p.Args["id"].(string)
					id, err := strconv.ParseInt(idStr, 10, 64)
					if err != nil {
						return nil, fmt.Errorf("invalid id")
					}
					status := p.Args["status"].(string)
					return updateTaskStatus(p.Context, db, id, status)
				},
			},
		},
	})

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query:    query,
		Mutation: mutation,
	})
	if err != nil {
		log.Fatalf("create schema: %v", err)
	}
	return schema
}

// Data access helpers

func fetchUserByID(ctx context.Context, db *sql.DB, id int64) (User, error) {
	var u User
	row := db.QueryRowContext(ctx, "SELECT id, email, name FROM users WHERE id = $1", id)
	if err := row.Scan(&u.ID, &u.Email, &u.Name); err != nil {
		if err == sql.ErrNoRows {
			return User{}, nil
		}
		return User{}, err
	}
	return u, nil
}

func fetchAllUsers(ctx context.Context, db *sql.DB) ([]User, error) {
	rows, err := db.QueryContext(ctx, "SELECT id, email, name FROM users ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, nil
}

func fetchTaskByID(ctx context.Context, db *sql.DB, id int64) (Task, error) {
	var t Task
	var desc sql.NullString
	var due sql.NullTime
	row := db.QueryRowContext(ctx, "SELECT id, user_id, title, description, status, due_date FROM tasks WHERE id = $1", id)
	if err := row.Scan(&t.ID, &t.UserID, &t.Title, &desc, &t.Status, &due); err != nil {
		if err == sql.ErrNoRows {
			return Task{}, nil
		}
		return Task{}, err
	}
	if desc.Valid {
		d := desc.String
		t.Description = &d
	}
	if due.Valid {
		t.DueDate = &due.Time
	}
	// fetch tags
	t.Tags = []string{}
	tagsRows, err := db.QueryContext(ctx, "SELECT tag FROM task_tags WHERE task_id = $1 ORDER BY tag", t.ID)
	if err == nil {
		defer tagsRows.Close()
		for tagsRows.Next() {
			var tag string
			if err := tagsRows.Scan(&tag); err == nil {
				t.Tags = append(t.Tags, tag)
			}
		}
	}
	return t, nil
}

func fetchTasksForUser(ctx context.Context, db *sql.DB, userID int64, status *string) ([]Task, error) {
	q := "SELECT id, user_id, title, description, status, due_date FROM tasks WHERE user_id = $1"
	args := []any{userID}
	if status != nil {
		q += " AND status = $2"
		args = append(args, *status)
	}
	q += " ORDER BY id"
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Task{}
	for rows.Next() {
		var t Task
		var desc sql.NullString
		var due sql.NullTime
		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &desc, &t.Status, &due); err != nil {
			return nil, err
		}
		if desc.Valid {
			d := desc.String
			t.Description = &d
		}
		if due.Valid {
			t.DueDate = &due.Time
		}
		// tags
		t.Tags = []string{}
		tagsRows, err := db.QueryContext(ctx, "SELECT tag FROM task_tags WHERE task_id = $1 ORDER BY tag", t.ID)
		if err == nil {
			defer tagsRows.Close()
			for tagsRows.Next() {
				var tag string
				if err := tagsRows.Scan(&tag); err == nil {
					t.Tags = append(t.Tags, tag)
				}
			}
		}
		out = append(out, t)
	}
	return out, nil
}

func fetchTasks(ctx context.Context, db *sql.DB, status *string, userId *int64) ([]Task, error) {
	q := "SELECT id, user_id, title, description, status, due_date FROM tasks"
	args := []any{}
	clauses := []string{}
	if userId != nil {
		clauses = append(clauses, fmt.Sprintf("user_id = $%d", len(args)+1))
		args = append(args, *userId)
	}
	if status != nil {
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(args)+1))
		args = append(args, *status)
	}
	if len(clauses) > 0 {
		q += " WHERE " + clauses[0]
		for i := 1; i < len(clauses); i++ {
			q += " AND " + clauses[i]
		}
	}
	q += " ORDER BY id"
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Task{}
	for rows.Next() {
		var t Task
		var desc sql.NullString
		var due sql.NullTime
		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &desc, &t.Status, &due); err != nil {
			return nil, err
		}
		if desc.Valid {
			d := desc.String
			t.Description = &d
		}
		if due.Valid {
			t.DueDate = &due.Time
		}
		// tags
		t.Tags = []string{}
		tagsRows, err := db.QueryContext(ctx, "SELECT tag FROM task_tags WHERE task_id = $1 ORDER BY tag", t.ID)
		if err == nil {
			defer tagsRows.Close()
			for tagsRows.Next() {
				var tag string
				if err := tagsRows.Scan(&tag); err == nil {
					t.Tags = append(t.Tags, tag)
				}
			}
		}
		out = append(out, t)
	}
	return out, nil
}

func createTask(ctx context.Context, db *sql.DB, userID int64, title string, description *string, status string, due *time.Time, tags []string) (Task, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var id int64
	// Note: argument order must match the SQL placeholders ($1..$5): user_id, title, description, status, due_date
	if err := tx.QueryRowContext(ctx, "INSERT INTO tasks (user_id, title, description, status, due_date) VALUES ($1, $2, $3, $4, $5) RETURNING id", userID, title, description, status, due).Scan(&id); err != nil {
		return Task{}, err
	}
	for _, tag := range tags {
		if _, err := tx.ExecContext(ctx, "INSERT INTO task_tags (task_id, tag) VALUES ($1, $2)", id, tag); err != nil {
			return Task{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Task{}, err
	}
	return fetchTaskByID(ctx, db, id)
}

func updateTaskStatus(ctx context.Context, db *sql.DB, id int64, status string) (Task, error) {
	if _, err := db.ExecContext(ctx, "UPDATE tasks SET status = $1 WHERE id = $2", status, id); err != nil {
		return Task{}, err
	}
	return fetchTaskByID(ctx, db, id)
}

// helper to encode debug JSON when needed
func mustJSON(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
