package storage

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/fblazt/boomerang/internal/models"
)

func TestBuildHAR_Standard12Schema(t *testing.T) {
	ts := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	entries := []models.HistoryEntry{
		{
			ID:         "hist_1",
			Method:     "POST",
			URL:        "https://api.example.com/v2/orders?dryRun=true",
			StatusCode: 201,
			StatusText: "201 Created",
			DurationMs: 85,
			SizeBytes:  28,
			Protocol:   "HTTP/2.0",
			ExecutedAt: ts,
			Request: models.RequestModel{
				Method:   "POST",
				URL:      "https://api.example.com/v2/orders?dryRun=true",
				BodyType: "json",
				Body:     `{"sku":"abc"}`,
				Headers:  []models.KeyValuePair{{Key: "Content-Type", Value: "application/json", Enabled: true}},
			},
			Response: &models.ResponseModel{
				StatusCode: 201,
				StatusText: "201 Created",
				SizeBytes:  28,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       `{"orderId":"ord_42"}`,
			},
			Telemetry: models.Telemetry{DNS: 5, TCP: 12, TLS: 18, TTFB: 45, Download: 5, Total: 85},
		},
	}

	raw, err := BuildHAR(entries)
	if err != nil {
		t.Fatalf("BuildHAR failed: %v", err)
	}

	var decoded HARArchive
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal HAR JSON failed: %v", err)
	}

	if decoded.Log.Version != "1.2" {
		t.Fatalf("expected HAR version 1.2, got %q", decoded.Log.Version)
	}
	if decoded.Log.Creator.Name != "Boomerang" {
		t.Fatalf("expected creator Boomerang, got %q", decoded.Log.Creator.Name)
	}
	if len(decoded.Log.Entries) != 1 {
		t.Fatalf("expected 1 HAR entry, got %d", len(decoded.Log.Entries))
	}

	e := decoded.Log.Entries[0]
	if e.Request.Method != "POST" || e.Response.Status != 201 {
		t.Fatalf("unexpected entry request/response: %+v / %+v", e.Request, e.Response)
	}
	if e.Request.PostData == nil || e.Request.PostData.Text != `{"sku":"abc"}` {
		t.Fatalf("expected postData body preserved, got %+v", e.Request.PostData)
	}
	if len(e.Request.QueryString) != 1 || e.Request.QueryString[0].Name != "dryRun" {
		t.Fatalf("expected dryRun query param in HAR, got %+v", e.Request.QueryString)
	}
	if e.Timings.DNS != 5 || e.Timings.SSL != 18 || e.Timings.Wait != 45 {
		t.Fatalf("expected timings preserved, got %+v", e.Timings)
	}
}
