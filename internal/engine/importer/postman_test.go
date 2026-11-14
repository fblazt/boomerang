package importer

import (
	"strings"
	"testing"

	"github.com/fblazt/boomerang/internal/models"
)

func TestParsePostmanCollection(t *testing.T) {
	raw := `{
		"info": {
			"name": "Payments v2",
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
		},
		"item": [
			{
				"name": "Auth",
				"item": [
					{
						"name": "Create Token",
						"request": {
							"method": "POST",
							"header": [{"key": "X-Client", "value": "desktop"}],
							"url": {"raw": "{{base_url}}/oauth/token?grant=client", "query": [{"key": "grant", "value": "client"}]},
							"body": {"mode": "raw", "raw": "{\"scope\":\"read\"}", "options": {"raw": {"language": "json"}}},
							"auth": {"type": "bearer", "bearer": [{"key": "token", "value": "{{jwt}}"}]}
						},
						"event": [
							{
								"listen": "test",
								"script": {"exec": ["pm.test('OK', () => { pm.response.to.have.status(200); });"]}
							}
						]
					}
				]
			},
			{
				"name": "Healthcheck",
				"request": {
					"method": "GET",
					"url": "{{base_url}}/health"
				}
			}
		]
	}`

	svc := NewService()
	prev, err := svc.Parse(raw, "payments.json")
	if err != nil {
		t.Fatalf("Parse Postman failed: %v", err)
	}
	if prev.Format != models.ImportFormatPostman || prev.Collection.Name != "Payments v2" {
		t.Fatalf("unexpected collection metadata: %+v", prev)
	}
	if prev.FolderCount != 1 || prev.RequestCount != 2 {
		t.Fatalf("expected 1 folder and 2 requests, got folders=%d requests=%d", prev.FolderCount, prev.RequestCount)
	}

	req := prev.Collection.Folders[0].Items[0].Request
	if req.Method != "POST" || req.AuthType != "bearer" || req.Token != "{{jwt}}" {
		t.Fatalf("unexpected nested folder request: %+v", req)
	}
	if !strings.Contains(req.TestScript, "pm.response.to.have.status(200)") {
		t.Fatalf("expected Postman test script extracted, got %q", req.TestScript)
	}
}
