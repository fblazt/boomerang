package importer

import (
	"strings"
	"testing"

	"github.com/fblazt/boomerang/internal/models"
)

func TestParseOpenAPI31_YAMLAndJSON(t *testing.T) {
	yamlSpec := `openapi: 3.1.0
info:
  title: Catalog API
servers:
  - url: https://catalog.internal.net/v1
paths:
  /products:
    get:
      summary: List Products
      tags:
        - Products
      parameters:
        - name: limit
          in: query
          required: true
          schema:
            type: integer
            default: 20
    post:
      summary: Create Product
      tags:
        - Products
      security:
        - BearerAuth: []
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                title:
                  type: string
                  example: Mechanical Keyboard
`

	svc := NewService()
	prev, err := svc.Parse(yamlSpec, "catalog.yaml")
	if err != nil {
		t.Fatalf("Parse OpenAPI YAML failed: %v", err)
	}
	if prev.Format != models.ImportFormatOpenAPI || prev.Collection.Name != "Catalog API" {
		t.Fatalf("unexpected OpenAPI preview: %+v", prev)
	}
	if prev.FolderCount != 1 || prev.RequestCount != 2 {
		t.Fatalf("expected 1 tag folder & 2 operations, got folders=%d reqs=%d", prev.FolderCount, prev.RequestCount)
	}

	getReq := prev.Collection.Folders[0].Items[0].Request
	if getReq.Method != "GET" || !strings.Contains(getReq.URL, "limit=20") {
		t.Fatalf("expected query default serialized on GET URL, got %+v", getReq)
	}
	postReq := prev.Collection.Folders[0].Items[1].Request
	if postReq.Method != "POST" || postReq.AuthType != "bearer" || !strings.Contains(postReq.Body, "Mechanical Keyboard") {
		t.Fatalf("unexpected POST operation synthesis: %+v", postReq)
	}

	jsonSpec := `{"openapi":"3.1.0","info":{"title":"JSON Spec"},"paths":{"/ping":{"get":{"summary":"Ping"}}}}`
	jsonPrev, err := svc.Parse(jsonSpec, "spec.json")
	if err != nil {
		t.Fatalf("Parse OpenAPI JSON failed: %v", err)
	}
	if jsonPrev.RequestCount != 1 || jsonPrev.Collection.Items[0].Request.Method != "GET" {
		t.Fatalf("unexpected OpenAPI JSON preview: %+v", jsonPrev)
	}
}
