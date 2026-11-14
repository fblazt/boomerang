package modals

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/engine/storage"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

func setupHistoryModalState(t *testing.T) (*models.AppState, *storage.SQLiteStore) {
	t.Helper()
	store, err := storage.OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("OpenSQLite memory failed: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	now := time.Now().UTC()
	entries := []models.HistoryEntry{
		{
			ID:         "h_recent",
			Method:     "POST",
			URL:        "https://api.example.com/v2/auth/token",
			StatusCode: 201,
			StatusText: "201 Created",
			DurationMs: 38,
			SizeBytes:  44,
			Protocol:   "HTTP/2.0",
			ExecutedAt: now.Add(-2 * time.Minute),
			Request: models.RequestModel{
				Name:   "Issue Token",
				Method: "POST",
				URL:    "https://api.example.com/v2/auth/token",
			},
			Response: &models.ResponseModel{
				StatusCode: 201,
				StatusText: "201 Created",
				Body:       `{"token":"abc"}`,
			},
		},
		{
			ID:         "h_yesterday",
			Method:     "GET",
			URL:        "https://api.example.com/v2/orders",
			StatusCode: 502,
			StatusText: "502 Bad Gateway",
			DurationMs: 210,
			SizeBytes:  18,
			Protocol:   "HTTP/1.1",
			ExecutedAt: now.AddDate(0, 0, -1),
			Request: models.RequestModel{
				Name:   "Fetch Orders",
				Method: "GET",
				URL:    "https://api.example.com/v2/orders",
			},
			Response: &models.ResponseModel{
				StatusCode: 502,
				StatusText: "502 Bad Gateway",
				Body:       `upstream down`,
			},
		},
	}
	for _, e := range entries {
		if _, err := store.RecordHistory(e); err != nil {
			t.Fatalf("RecordHistory failed: %v", err)
		}
	}

	state := models.NewAppState(nil)
	state.SetHistoryStore(store)
	state.OpenHistoryModal()
	return state, store
}

func TestHistoryModal_RenderFilterReplayAndHARExport(t *testing.T) {
	state, _ := setupHistoryModalState(t)

	exportPath := filepath.Join(t.TempDir(), "export.har")
	state.SetSaveFilePicker(func(title, defaultName string, extensions []string, onPicked func(path string)) {
		onPicked(exportPath)
	})

	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(theme.NewBoomerangTheme())
		HistoryModalHost(c, state)
	}, 960, 700)
	tt.Frame()

	for _, want := range []string{
		"Execution History",
		"2 Executed Calls",
		"TODAY • LAST 15 MINS",
		"YESTERDAY",
		"201 Created",
		"502 Bad Gateway",
		"Export HAR",
	} {
		if !tt.HasText(want) {
			t.Fatalf("expected %q in history modal", want)
		}
	}

	// Filter by 5xx status chip
	if err := tt.Click("5xx"); err != nil {
		t.Fatalf("click 5xx filter: %v", err)
	}
	tt.Frame()
	if tt.HasText("201 Created") {
		t.Fatal("expected 201 Created filtered out by 5xx chip")
	}
	if !tt.HasText("502 Bad Gateway") {
		t.Fatal("expected 502 Bad Gateway visible under 5xx chip")
	}

	// Toggle 5xx off and verify Export HAR writes valid HAR 1.2 JSON via SaveFilePicker
	if err := tt.Click("5xx"); err != nil {
		t.Fatalf("uncheck 5xx filter: %v", err)
	}
	tt.Frame()
	if err := tt.Click("Export HAR"); err != nil {
		t.Fatalf("click Export HAR: %v", err)
	}
	tt.Frame()

	raw, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("read exported HAR file: %v", err)
	}
	var har storage.HARArchive
	if err := json.Unmarshal(raw, &har); err != nil {
		t.Fatalf("invalid HAR JSON: %v", err)
	}
	if har.Log.Version != "1.2" || len(har.Log.Entries) != 2 {
		t.Fatalf("unexpected exported HAR archive: version=%q entries=%d", har.Log.Version, len(har.Log.Entries))
	}

	// Click Issue Token call row -> loads request & response and closes modal
	if err := tt.Click("Issue Token"); err != nil {
		t.Fatalf("click Issue Token history row: %v", err)
	}
	tt.Frame()
	if state.ShowHistoryModal {
		t.Fatal("expected ShowHistoryModal closed after selecting history row")
	}
	if state.ActiveRequest.URL != "https://api.example.com/v2/auth/token" || state.ActiveResponse == nil || state.ActiveResponse.StatusCode != 201 {
		t.Fatalf("expected historical request & response replayed into workspace, got %+v / %+v", state.ActiveRequest, state.ActiveResponse)
	}
}

func TestHistoryModal_SearchAndClear(t *testing.T) {
	state, _ := setupHistoryModalState(t)

	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(theme.NewBoomerangTheme())
		HistoryModalHost(c, state)
	}, 960, 700)
	tt.Frame()

	state.SetHistorySearch("orders")
	tt.Frame()
	if tt.HasText("Issue Token") {
		t.Fatal("expected Issue Token hidden under search 'orders'")
	}
	if !tt.HasText("Fetch Orders") {
		t.Fatal("expected Fetch Orders matching search 'orders'")
	}

	if err := tt.Click("Clear"); err != nil {
		t.Fatalf("click Clear: %v", err)
	}
	tt.Frame()
	if !tt.HasText("No matching execution history") || !tt.HasText("0 Executed Calls") {
		t.Fatal("expected empty history message and 0 Executed Calls after Clear")
	}
}
