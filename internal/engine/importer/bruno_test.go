package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fblazt/boomerang/internal/models"
)

func TestParseBruno_SingleSnippetAndDirectory(t *testing.T) {
	bru := `meta {
  name: Create User
  type: http
  seq: 1
}

post {
  url: https://api.boomerang.dev/v1/users
  body: json
  auth: bearer
}

params:query {
  invite: true
  ~debug: 1
}

headers {
  X-Trace: bruno-import
}

auth:bearer {
  token: {{access_token}}
}

body:json {
  {
    "email": "ada@boomerang.dev"
  }
}

tests {
  pm.test("Created", () => {
    pm.response.to.have.status(201);
  });
}
`

	svc := NewService()
	prev, err := svc.Parse(bru, "create-user.bru")
	if err != nil {
		t.Fatalf("Parse Bruno snippet failed: %v", err)
	}
	if prev.Format != models.ImportFormatBruno || prev.RequestCount != 1 {
		t.Fatalf("unexpected Bruno preview: %+v", prev)
	}
	req := prev.Collection.Items[0].Request
	if req.Name != "Create User" || req.Method != "POST" || req.AuthType != "bearer" || req.Token != "{{access_token}}" {
		t.Fatalf("unexpected Bruno request: %+v", req)
	}
	if len(req.Params) != 2 || !req.Params[0].Enabled || req.Params[1].Enabled {
		t.Fatalf("expected disabled ~debug param preserved, got %+v", req.Params)
	}
	if req.BodyType != "json" || !strings.Contains(req.Body, "ada@boomerang.dev") {
		t.Fatalf("unexpected Bruno JSON body: %s (%s)", req.BodyType, req.Body)
	}
	if !strings.Contains(req.TestScript, "pm.response.to.have.status(201)") {
		t.Fatalf("expected Bruno tests block preserved, got %q", req.TestScript)
	}

	// Verify ParseFile on directory of .bru files
	dir := filepath.Join(t.TempDir(), "AuthService")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "create.bru"), []byte(bru), 0o600); err != nil {
		t.Fatal(err)
	}
	dirPrev, err := svc.ParseFile(dir)
	if err != nil {
		t.Fatalf("ParseFile Bruno directory failed: %v", err)
	}
	if dirPrev.Collection.Name != "AuthService" || dirPrev.RequestCount != 1 {
		t.Fatalf("unexpected directory import preview: %+v", dirPrev)
	}
}
