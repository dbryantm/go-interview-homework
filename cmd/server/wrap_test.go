package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/graphql-go/graphql"
)

func TestWrapResolveLogsAndReturns(t *testing.T) {
	tests := []struct {
		name       string
		fn         func(p graphql.ResolveParams) (any, error)
		expectsErr bool
	}{
		{"success", func(p graphql.ResolveParams) (any, error) { return "ok", nil }, false},
		{"failure", func(p graphql.ResolveParams) (any, error) { return nil, ErrTest{} }, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			SetLoggerOutput(&buf)
			defer SetLoggerOutput(nil)

			w := wrapResolve("test.res", tc.fn)
			res, err := w(graphql.ResolveParams{Args: map[string]interface{}{"id": "123"}})
			if tc.expectsErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res != "ok" {
					t.Fatalf("unexpected result: %v", res)
				}
			}

			out := buf.String()
			if !strings.Contains(out, `"resolver":"test.res"`) {
				t.Fatalf("log output missing resolver: %s", out)
			}
			if !strings.Contains(out, `"duration_ms"`) {
				t.Fatalf("log output missing duration_ms: %s", out)
			}
			if tc.expectsErr {
				if !strings.Contains(out, `"level":"error"`) {
					t.Fatalf("expected error level in log: %s", out)
				}
			} else {
				if !strings.Contains(out, `"level":"info"`) {
					t.Fatalf("expected info level in log: %s", out)
				}
			}
		})
	}
}

// ErrTest implements error for test purposes.
type ErrTest struct{}

func (ErrTest) Error() string { return "test error" }
