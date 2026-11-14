package modals

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/engine/collection"
	"github.com/fblazt/boomerang/internal/engine/importer"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

func setupImporterTestState(t *testing.T) (*models.AppState, *collection.Store) {
	t.Helper()
	store, err := collection.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	state := models.NewAppState(nil)
	state.SetCollectionStore(store)
	state.SetImporterService(importer.NewService())
	return state, store
}

func TestImportModal_PasteCurlAndOneClickImport(t *testing.T) {
	state, store := setupImporterTestState(t)
	state.OpenImportModal()

	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(theme.NewBoomerangTheme())
		ImportModalHost(c, state)
	}, 1024, 720)
	tt.Frame()

	state.SetImportRawText(`curl -X POST https://api.boomerang.dev/v1/webhooks -H 'Authorization: Bearer tok_live' -d '{"event":"push"}'`)
	tt.Frame()

	for _, want := range []string{"Universal API Importer", "cURL", "POST", "POST /v1/webhooks", "Import Collection"} {
		if !tt.HasText(want) {
			t.Fatalf("expected %q in import modal preview", want)
		}
	}

	if err := tt.Click("Import Collection"); err != nil {
		t.Fatalf("click Import Collection: %v", err)
	}
	tt.Frame()

	cols := store.Collections()
	if len(cols) != 1 || len(cols[0].Items) != 1 {
		t.Fatalf("expected 1 persisted collection with 1 item, got %+v", cols)
	}

	var activeURL, activeAuth string
	var stillOpen bool
	state.View(func(s *models.AppState) {
		activeURL = s.ActiveRequest.URL
		activeAuth = s.ActiveRequest.AuthType
		stillOpen = s.ShowImportModal
	})
	if stillOpen {
		t.Fatal("expected import modal closed after confirmation")
	}
	if activeURL != "https://api.boomerang.dev/v1/webhooks" || activeAuth != "bearer" {
		t.Fatalf("expected imported request loaded into workspace, got url=%q auth=%q", activeURL, activeAuth)
	}
}

func TestImportModal_ChooseFileOpenAPI(t *testing.T) {
	state, store := setupImporterTestState(t)

	specPath := filepath.Join(t.TempDir(), "billing.yaml")
	spec := `openapi: 3.1.0
info:
  title: Billing Gateway
servers:
  - url: https://billing.internal.net
paths:
  /invoices:
    get:
      summary: List Invoices
      tags:
        - Invoices
`
	if err := os.WriteFile(specPath, []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}

	state.SetFilePicker(func(_ string, _ []string, onPicked func(string)) {
		onPicked(specPath)
	})
	state.OpenImportModal()

	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(theme.NewBoomerangTheme())
		ImportModalHost(c, state)
	}, 1024, 720)
	tt.Frame()

	if err := tt.Click("Choose File"); err != nil {
		t.Fatalf("click Choose File: %v", err)
	}
	tt.Frame()

	for _, want := range []string{"OpenAPI 3.1", "Billing Gateway", "List Invoices"} {
		if !tt.HasText(want) {
			t.Fatalf("expected %q after choosing OpenAPI file", want)
		}
	}

	if err := tt.Click("Import Collection"); err != nil {
		t.Fatalf("click Import Collection: %v", err)
	}
	tt.Frame()

	cols := store.Collections()
	if len(cols) != 1 || cols[0].Name != "Billing Gateway" || len(cols[0].Folders) != 1 {
		t.Fatalf("expected Billing Gateway collection imported with folder, got %+v", cols)
	}
}
