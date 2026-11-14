package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteAndReadJSONAtomic(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "collections")
	path := filepath.Join(dir, "sample.json")

	type doc struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	want := doc{ID: "col_1", Name: "Billing Service"}
	if err := WriteJSONAtomic(path, want); err != nil {
		t.Fatalf("WriteJSONAtomic failed: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if !strings.Contains(string(raw), "\n  \"name\": \"Billing Service\"") {
		t.Fatalf("expected readable indented JSON, got:\n%s", string(raw))
	}
	if !json.Valid(raw) {
		t.Fatalf("expected valid JSON")
	}

	var got doc
	if err := ReadJSON(path, &got); err != nil {
		t.Fatalf("ReadJSON failed: %v", err)
	}
	if got != want {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", got, want)
	}
}

func TestSafeSlugAndNewID(t *testing.T) {
	cases := map[string]string{
		"Core API / v2":     "core-api-v2",
		"   ":               "untitled",
		"Users__Service!!":  "users-service",
		"Auth 123 Endpoint": "auth-123-endpoint",
	}
	for input, want := range cases {
		if got := SafeSlug(input); got != want {
			t.Errorf("SafeSlug(%q) = %q, want %q", input, got, want)
		}
	}

	id1 := NewID("col")
	id2 := NewID("col")
	if !strings.HasPrefix(id1, "col_") || id1 == id2 {
		t.Fatalf("expected unique prefixed IDs, got %q and %q", id1, id2)
	}
}
