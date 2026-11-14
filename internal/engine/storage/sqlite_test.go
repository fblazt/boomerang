package storage

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/fblazt/boomerang/internal/models"
)

func TestSQLiteHistoryCRUDAndFilters(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "boomerang.db")
	store, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLite failed: %v", err)
	}
	defer store.Close()

	base := time.Now().UTC().Add(-2 * time.Hour)
	samples := []models.HistoryEntry{
		{
			ID:              "hist_1",
			RequestID:       "req_users",
			Method:          "GET",
			URL:             "https://api.internal.net/v1/users",
			StatusCode:      200,
			StatusText:      "200 OK",
			DurationMs:      42,
			SizeBytes:       512,
			Protocol:        "HTTP/2.0",
			EnvironmentName: "PROD",
			ExecutedAt:      base,
			Request:         models.RequestModel{ID: "req_users", Name: "List Users", Method: "GET", URL: "https://api.internal.net/v1/users"},
			Response:        &models.ResponseModel{StatusCode: 200, StatusText: "200 OK", Body: `{"ok":true}`},
			Telemetry:       models.Telemetry{DNS: 4, TCP: 8, TLS: 12, TTFB: 15, Download: 3, Total: 42, TotalMicros: 42100, TLSVersion: "TLS 1.3"},
		},
		{
			ID:              "hist_2",
			RequestID:       "req_auth",
			Method:          "POST",
			URL:             "https://api.internal.net/v1/auth/login",
			StatusCode:      401,
			StatusText:      "401 Unauthorized",
			DurationMs:      19,
			SizeBytes:       98,
			Protocol:        "HTTP/2.0",
			EnvironmentName: "STG",
			ExecutedAt:      base.Add(30 * time.Minute),
			Request:         models.RequestModel{ID: "req_auth", Name: "Auth Login", Method: "POST", URL: "https://api.internal.net/v1/auth/login"},
			Response:        &models.ResponseModel{StatusCode: 401, StatusText: "401 Unauthorized"},
			Telemetry:       models.Telemetry{Total: 19, ConnReused: true},
		},
		{
			ID:              "hist_3",
			Method:          "DELETE",
			URL:             "https://unreachable.local/session",
			Error:           "connection refused",
			DurationMs:      110,
			EnvironmentName: "DEV",
			ExecutedAt:      base.Add(90 * time.Minute),
			Request:         models.RequestModel{Method: "DELETE", URL: "https://unreachable.local/session"},
			Telemetry:       models.Telemetry{Total: 110},
		},
	}

	for _, entry := range samples {
		if _, err := store.RecordHistory(entry); err != nil {
			t.Fatalf("RecordHistory(%s) failed: %v", entry.ID, err)
		}
	}

	// Reopen from disk to prove persistence across restarts
	if err := store.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	reopened, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("Reopen SQLite failed: %v", err)
	}
	defer reopened.Close()

	all, err := reopened.ListHistory(models.HistoryFilter{})
	if err != nil {
		t.Fatalf("ListHistory failed: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 history rows, got %d", len(all))
	}
	// Ordered newest first
	if all[0].ID != "hist_3" || all[2].ID != "hist_1" {
		t.Fatalf("expected reverse chronological order, got %s -> %s", all[0].ID, all[2].ID)
	}
	if all[2].Telemetry.TLSVersion != "TLS 1.3" || all[2].Telemetry.TotalMicros != 42100 {
		t.Fatalf("expected telemetry fields preserved, got %+v", all[2].Telemetry)
	}

	// Filter by method
	gets, err := reopened.ListHistory(models.HistoryFilter{Method: "GET"})
	if err != nil || len(gets) != 1 || gets[0].ID != "hist_1" {
		t.Fatalf("method filter mismatch: len=%d err=%v", len(gets), err)
	}

	// Filter by status class & errors
	f4xx, err := reopened.ListHistory(models.HistoryFilter{StatusClass: "4xx"})
	if err != nil || len(f4xx) != 1 || f4xx[0].ID != "hist_2" {
		t.Fatalf("4xx filter mismatch: len=%d err=%v", len(f4xx), err)
	}
	ferr, err := reopened.ListHistory(models.HistoryFilter{StatusClass: "err"})
	if err != nil || len(ferr) != 1 || ferr[0].ID != "hist_3" {
		t.Fatalf("err filter mismatch: len=%d err=%v", len(ferr), err)
	}

	// Filter by timestamp range & search string
	sinceCount, err := reopened.CountHistory(models.HistoryFilter{Since: base.Add(20 * time.Minute), Search: "login"})
	if err != nil || sinceCount != 1 {
		t.Fatalf("timestamp + search count mismatch: count=%d err=%v", sinceCount, err)
	}

	got, err := reopened.GetHistory("hist_1")
	if err != nil || got.Response == nil || got.Response.Body != `{"ok":true}` {
		t.Fatalf("GetHistory mismatch: %+v err=%v", got, err)
	}

	if err := reopened.DeleteHistory("hist_2"); err != nil {
		t.Fatalf("DeleteHistory failed: %v", err)
	}
	cnt, _ := reopened.CountHistory(models.HistoryFilter{})
	if cnt != 2 {
		t.Fatalf("expected 2 entries after delete, got %d", cnt)
	}

	if err := reopened.ClearHistory(); err != nil {
		t.Fatalf("ClearHistory failed: %v", err)
	}
	cnt, _ = reopened.CountHistory(models.HistoryFilter{})
	if cnt != 0 {
		t.Fatalf("expected 0 entries after ClearHistory, got %d", cnt)
	}
}

func TestSQLiteScratchpadDraftsAndConcurrency(t *testing.T) {
	store, err := OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("OpenSQLite memory failed: %v", err)
	}
	defer store.Close()

	saved, err := store.SaveDraft(models.ScratchpadDraft{
		Name:  "Quick Health Probe",
		Order: 0,
		Request: models.RequestModel{
			Method: "GET",
			URL:    "https://api.internal.net/v1/health",
		},
	})
	if err != nil {
		t.Fatalf("SaveDraft failed: %v", err)
	}

	// Update existing draft
	saved.Request.URL = "https://api.internal.net/v2/health"
	updated, err := store.SaveDraft(saved)
	if err != nil {
		t.Fatalf("Update SaveDraft failed: %v", err)
	}
	if updated.ID != saved.ID {
		t.Fatalf("expected upsert to keep ID %q, got %q", saved.ID, updated.ID)
	}

	drafts, err := store.ListDrafts()
	if err != nil || len(drafts) != 1 || drafts[0].Request.URL != "https://api.internal.net/v2/health" {
		t.Fatalf("ListDrafts mismatch: %+v err=%v", drafts, err)
	}

	// Concurrent history writes under race detector
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := store.RecordHistory(models.HistoryEntry{
				Method:     "GET",
				URL:        "https://example.com/ping",
				StatusCode: 200,
				Request:    models.RequestModel{Method: "GET", URL: "https://example.com/ping"},
			})
			if err != nil {
				t.Errorf("concurrent RecordHistory %d failed: %v", idx, err)
			}
		}(i)
	}
	wg.Wait()

	total, _ := store.CountHistory(models.HistoryFilter{})
	if total != 10 {
		t.Fatalf("expected 10 history entries after concurrent writes, got %d", total)
	}

	if err := store.DeleteDraft(saved.ID); err != nil {
		t.Fatalf("DeleteDraft failed: %v", err)
	}
	left, _ := store.ListDrafts()
	if len(left) != 0 {
		t.Fatalf("expected 0 drafts after DeleteDraft, got %d", len(left))
	}
}
