package views

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/engine/collection"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

func setupSidebarTestState(t *testing.T) (*models.AppState, *collection.Store, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := collection.NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	if err := store.SeedDefaultsIfEmpty(); err != nil {
		t.Fatalf("SeedDefaultsIfEmpty failed: %v", err)
	}
	state := models.NewAppState(nil)
	state.SetCollectionStore(store)
	return state, store, dir
}

func TestSidebar_NestedTreeAndFolderCollapse(t *testing.T) {
	state, _, _ := setupSidebarTestState(t)

	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(theme.NewBoomerangTheme())
		SidebarView(c, state)
	}, 320, 800)
	tt.Frame()

	for _, want := range []string{"Core API", "Users Service", "Auth Verify", "Catalog", "Product Feed", "0 Drafts • 1 Collections"} {
		if !tt.HasText(want) {
			t.Fatalf("expected %q in sidebar tree", want)
		}
	}

	// Collapse nested Catalog folder -> hides child item Product Feed
	if err := tt.Click("Catalog"); err != nil {
		t.Fatalf("click Catalog folder: %v", err)
	}
	tt.Frame()
	if tt.HasText("Product Feed") {
		t.Fatal("expected Product Feed hidden when Catalog folder collapsed")
	}

	// Expand Catalog again -> shows Product Feed
	if err := tt.Click("Catalog"); err != nil {
		t.Fatalf("re-click Catalog folder: %v", err)
	}
	tt.Frame()
	if !tt.HasText("Product Feed") {
		t.Fatal("expected Product Feed visible after expanding Catalog")
	}
}

func TestSidebar_RequestActionsDuplicateRenameReorderDeleteAndSave(t *testing.T) {
	state, _, dir := setupSidebarTestState(t)

	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(theme.NewBoomerangTheme())
		ShellView(c, state)
	}, 1280, 840)
	tt.Frame()

	// Select Auth Verify
	if err := tt.Click("Auth Verify"); err != nil {
		t.Fatalf("click Auth Verify: %v", err)
	}
	tt.Frame()
	if state.ActiveItemID != "req_auth_verify" || state.ActiveRequest.Name != "Auth Verify" {
		t.Fatalf("expected Auth Verify selected, got id=%q name=%q", state.ActiveItemID, state.ActiveRequest.Name)
	}

	// Duplicate Auth Verify -> Auth Verify Copy
	if err := tt.Click("Dup"); err != nil {
		t.Fatalf("click Dup: %v", err)
	}
	tt.Frame()
	if !tt.HasText("Auth Verify Copy") {
		t.Fatal("expected duplicated item 'Auth Verify Copy'")
	}

	// Select copy and move it Up (-1) so it precedes Auth Verify
	if err := tt.Click("Auth Verify Copy"); err != nil {
		t.Fatalf("click Auth Verify Copy: %v", err)
	}
	tt.Frame()
	if err := tt.Click("↑"); err != nil {
		t.Fatalf("click Move Up: %v", err)
	}
	tt.Frame()

	// Rename selected copy inline
	if err := tt.Click("Rename"); err != nil {
		t.Fatalf("click Rename: %v", err)
	}
	tt.Frame()
	state.SetRenameDraft("Refresh Token")
	tt.Frame()
	if err := tt.Click("✓"); err != nil {
		t.Fatalf("click Apply Rename: %v", err)
	}
	tt.Frame()
	if !tt.HasText("Refresh Token") {
		t.Fatal("expected renamed item 'Refresh Token'")
	}

	// Edit URL on selected renamed item and click Save in UrlBar
	if err := tt.Click("Refresh Token"); err != nil {
		t.Fatalf("select Refresh Token: %v", err)
	}
	tt.Frame()
	state.SetURL("{{base_url}}/v2/auth/refresh")
	if err := tt.Click("Save"); err != nil {
		t.Fatalf("click Save: %v", err)
	}
	tt.Frame()

	// Reload store from disk and assert persisted order + rename + URL save!
	reloaded, err := collection.NewStore(dir)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	cols := reloaded.Collections()
	items := cols[0].Items
	if len(items) != 4 {
		t.Fatalf("expected 4 root items after duplicate, got %d", len(items))
	}
	if items[1].Name != "Refresh Token" || items[2].Name != "Auth Verify" {
		t.Fatalf("expected Refresh Token reordered before Auth Verify, got %q, %q", items[1].Name, items[2].Name)
	}
	if items[1].Request.URL != "{{base_url}}/v2/auth/refresh" {
		t.Fatalf("expected saved URL persisted on disk, got %q", items[1].Request.URL)
	}

	// Delete Refresh Token
	if err := tt.Click("Remove"); err != nil {
		t.Fatalf("click Remove: %v", err)
	}
	tt.Frame()
	if _, reqCount := state.CollectionCounts(); reqCount != 4 {
		t.Fatalf("expected 4 remaining requests after delete, got %d", reqCount)
	}
	reloadedAfterDel, err := collection.NewStore(dir)
	if err != nil {
		t.Fatalf("reload store after delete: %v", err)
	}
	if len(reloadedAfterDel.Collections()[0].Items) != 3 {
		t.Fatalf("expected 3 root items on disk after delete, got %d", len(reloadedAfterDel.Collections()[0].Items))
	}
}

func TestSidebar_CreateCollectionFolderRequestAndFilter(t *testing.T) {
	state, _, _ := setupSidebarTestState(t)

	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(theme.NewBoomerangTheme())
		SidebarView(c, state)
	}, 320, 840)
	tt.Frame()

	if err := tt.Click("+Col"); err != nil {
		t.Fatalf("click +Col: %v", err)
	}
	tt.Frame()
	if !tt.HasText("Collection 2") {
		t.Fatal("expected new 'Collection 2'")
	}

	if err := tt.Click("+Dir"); err != nil {
		t.Fatalf("click +Dir: %v", err)
	}
	tt.Frame()
	if !tt.HasText("New Folder") {
		t.Fatal("expected new folder 'New Folder'")
	}

	state.SetCollectionFilter("auth")
	tt.Frame()
	if !tt.HasText("Auth Verify") {
		t.Fatal("expected Auth Verify matched by filter 'auth'")
	}
	if tt.HasText("Delete Session") {
		t.Fatal("expected Delete Session hidden by filter 'auth'")
	}
}
