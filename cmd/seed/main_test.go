package main

import (
	"testing"
)

func TestNullableString(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want interface{}
	}{
		{"empty string returns nil", "", nil},
		{"non-empty returns same string", "hello", "hello"},
		{"whitespace returns whitespace string", " ", " "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := nullableString(tc.in)
			if got == nil && tc.want == nil {
				return
			}
			if got == nil || tc.want == nil {
				t.Fatalf("nullableString(%q) = %v, want %v", tc.in, got, tc.want)
			}
			if gs, ok := got.(string); !ok || gs != tc.want.(string) {
				t.Fatalf("nullableString(%q) = %v (type %T), want %v (type %T)", tc.in, got, got, tc.want, tc.want)
			}
		})
	}
}
