package importer

import (
	"testing"

	"github.com/fblazt/boomerang/internal/models"
)

func TestDetectFormat(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  models.ImportFormat
	}{
		{"cURL", "curl -X POST https://api.test/v1/users -H 'Content-Type: application/json'", models.ImportFormatCurl},
		{"Postman v2.1", `{"info":{"name":"Demo","schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},"item":[]}`, models.ImportFormatPostman},
		{"OpenAPI YAML", "openapi: 3.1.0\ninfo:\n  title: Users\npaths:\n  /users: {}", models.ImportFormatOpenAPI},
		{"OpenAPI JSON", `{"openapi":"3.1.0","paths":{"/health":{}}}`, models.ImportFormatOpenAPI},
		{"Bruno DSL", "meta {\n  name: Get Profile\n  type: http\n}\nget {\n  url: https://api.test/me\n}", models.ImportFormatBruno},
		{"Unknown", "just plain words", models.ImportFormatUnknown},
	}

	svc := NewService()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := svc.DetectFormat(tc.input); got != tc.want {
				t.Fatalf("DetectFormat = %q, want %q", got, tc.want)
			}
		})
	}
}
