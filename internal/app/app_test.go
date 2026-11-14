package app_test

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/app"
	"github.com/fblazt/boomerang/internal/engine/storage"
	"github.com/fblazt/boomerang/internal/models"
)

type stubHistoryExecutor struct{}

func (stubHistoryExecutor) Execute(_ context.Context, req models.RequestModel) (*models.ResponseModel, error) {
	return &models.ResponseModel{
		StatusCode: http.StatusCreated,
		StatusText: "201 Created",
		Duration:   12 * time.Millisecond,
		SizeBytes:  16,
		Proto:      "HTTP/2.0",
		Body:       `{"created":true}`,
		Telemetry:  models.Telemetry{Total: 12, TotalMicros: 12000},
	}, nil
}

func TestApp_WindowOptionsAndInitialization(t *testing.T) {
	opts := app.DefaultOptions()
	opts.DataDir = t.TempDir()
	application := app.NewApp(opts)
	defer application.Close()

	winOpts := application.BuildWindowOptions()
	if winOpts.Title != "Boomerang" {
		t.Fatalf("expected title 'Boomerang', got %s", winOpts.Title)
	}
	if winOpts.Width != 1440 || winOpts.Height != 900 {
		t.Fatalf("expected 1440x900, got %dx%d", winOpts.Width, winOpts.Height)
	}
	if winOpts.TitleBarStyle != mygo.TitleBarHiddenInset {
		t.Fatalf("expected TitleBarHiddenInset, got %v", winOpts.TitleBarStyle)
	}
	if winOpts.BackgroundColor != "#090A0F" {
		t.Fatalf("expected #090A0F, got %s", winOpts.BackgroundColor)
	}
}

func TestApp_ViewHeadlessRender(t *testing.T) {
	opts := app.DefaultOptions()
	opts.DataDir = t.TempDir()
	application := app.NewApp(opts)
	defer application.Close()

	tt := ui.NewTester(application.View, 1440, 900)
	tt.Frame()

	// Verify Header and Title
	if !tt.HasText("Boomerang") {
		t.Fatal("expected 'Boomerang' in header")
	}

	// Verify Environment chip
	if !tt.HasText("PROD") {
		t.Fatal("expected 'PROD' env chip")
	}

	// Verify Sidebar placeholder
	if !tt.HasText("COLLECTIONS") {
		t.Fatal("expected 'COLLECTIONS' sidebar")
	}

	// Verify Workspace endpoint text
	if !tt.HasText("https://api.internal.net/v1/health") {
		t.Fatal("expected health endpoint text in workspace")
	}

	// Mutate state and assert reactive update
	application.State().SetURL("https://api.github.com")
	application.State().SetActiveEnv("STG")

	tt.Frame()

	if !tt.HasText("https://api.github.com") {
		t.Fatal("expected updated URL in workspace")
	}

	if !tt.HasText("STG") {
		t.Fatal("expected 'STG' env chip after update")
	}
}

func TestApp_PreviewComponents(t *testing.T) {
	opts := app.DefaultOptions()
	opts.DataDir = t.TempDir()
	opts.Preview = "components"
	application := app.NewApp(opts)
	defer application.Close()

	tt := ui.NewTester(application.View, 1440, 900)
	tt.Frame()

	if !tt.HasText("Boomerang Component Catalog") {
		t.Fatal("expected Component Catalog view in preview mode")
	}
	if !tt.HasText("1. 3D Keycap Badges") {
		t.Fatal("expected 3D Keycap Badges section in preview mode")
	}
}

func TestApp_RecordsExecutionHistoryInSQLite(t *testing.T) {
	dataDir := t.TempDir()
	opts := app.DefaultOptions()
	opts.DataDir = dataDir
	application := app.NewApp(opts)

	st := application.State()
	st.SetExecutor(stubHistoryExecutor{})
	st.SetMethod("POST")
	st.SetURL("https://api.internal.net/v1/items")
	st.SendRequest()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var loading bool
		st.View(func(s *app.AppState) {
			loading = s.IsLoading
		})
		if !loading {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := application.Close(); err != nil {
		t.Fatalf("close app failed: %v", err)
	}

	reopened, err := storage.OpenSQLite(filepath.Join(dataDir, "boomerang.db"))
	if err != nil {
		t.Fatalf("open sqlite db failed: %v", err)
	}
	defer reopened.Close()

	rows, err := reopened.ListHistory(models.HistoryFilter{})
	if err != nil {
		t.Fatalf("ListHistory failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 persisted history row, got %d", len(rows))
	}
	if rows[0].Method != "POST" || rows[0].StatusCode != http.StatusCreated {
		t.Fatalf("unexpected history row: %+v", rows[0])
	}
}
