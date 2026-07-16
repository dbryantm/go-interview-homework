package main

import (
	"strings"
	"testing"
)

func TestMustJSON(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"simple struct", struct{ A string }{A: "x"}, `"A": "x"`},
		{"map", map[string]int{"n": 2}, `"n": 2`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mustJSON(tc.in)
			if !strings.Contains(got, tc.want) {
				t.Fatalf("mustJSON(%v) = %s, want substring %q", tc.in, got, tc.want)
			}
		})
	}
}
