package importer

import (
	"testing"

	"github.com/fblazt/boomerang/internal/models"
)

func TestParseCurlCollection(t *testing.T) {
	raw := `curl -X POST 'https://api.example.com/v1/orders?dry=true' \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer tok_secret_123' \
  --data-raw '{"sku":"boom-99","qty":2}'`

	svc := NewService()
	prev, err := svc.Parse(raw, "paste")
	if err != nil {
		t.Fatalf("Parse cURL failed: %v", err)
	}
	if prev.Format != models.ImportFormatCurl || prev.RequestCount != 1 {
		t.Fatalf("unexpected preview: %+v", prev)
	}
	req := prev.Collection.Items[0].Request
	if req.Method != "POST" || req.URL != "https://api.example.com/v1/orders?dry=true" {
		t.Fatalf("unexpected method/url: %+v", req)
	}
	if req.BodyType != "json" || req.Body != `{"sku":"boom-99","qty":2}` {
		t.Fatalf("unexpected body: type=%s body=%s", req.BodyType, req.Body)
	}
	if req.AuthType != "bearer" || req.Token != "tok_secret_123" {
		t.Fatalf("unexpected bearer auth extraction: %+v", req)
	}
	if len(req.Params) != 1 || req.Params[0].Key != "dry" || req.Params[0].Value != "true" {
		t.Fatalf("expected dry=true param synced, got %+v", req.Params)
	}
}
