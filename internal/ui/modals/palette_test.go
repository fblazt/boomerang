package modals

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

func setupPaletteTestState() *models.AppState {
	state := models.NewAppState(nil)
	state.Mutate(func(s *models.AppState) {
		s.Collections = []models.Collection{
			{
				ID:   "col_auth",
				Name: "Auth Service",
				Folders: []models.Folder{
					{
						ID:   "fld_sessions",
						Name: "Sessions",
						Items: []models.CollectionItem{
							{
								ID:   "item_login",
								Name: "User Login",
								Request: models.RequestModel{
									ID:     "req_login",
									Name:   "User Login",
									Method: "POST",
									URL:    "https://api.test/v1/login",
								},
							},
						},
					},
				},
				Items: []models.CollectionItem{
					{
						ID:   "item_profile",
						Name: "Fetch Profile",
						Request: models.RequestModel{
							ID:     "req_profile",
							Name:   "Fetch Profile",
							Method: "GET",
							URL:    "https://api.test/v1/me",
						},
					},
				},
			},
		}
		s.Drafts = []models.ScratchpadDraft{
			{
				ID:   "draft_1",
				Name: "Billing Probe",
				Request: models.RequestModel{
					Method: "GET",
					URL:    "https://api.test/billing",
				},
			},
		}
	})
	return state
}

func TestPaletteModal_FuzzySearchAndKeyboardNavigation(t *testing.T) {
	state := setupPaletteTestState()
	state.OpenCommandPalette()

	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(theme.NewBoomerangTheme())
		PaletteModalHost(c, state)
	}, 960, 680)
	tt.Frame()

	// Verify endpoints, environments, scratchpad, and commands populate immediately
	for _, want := range []string{"User Login", "Fetch Profile", "Billing Probe", "STG", "GraphQL Explorer"} {
		if !tt.HasText(want) {
			t.Fatalf("expected %q in initial command palette list", want)
		}
	}

	// Fuzzy match nested endpoint ("ulgn" -> "User Login")
	state.SetPaletteQuery("lgn")
	tt.Frame()
	if !tt.HasText("User Login") || tt.HasText("Fetch Profile") {
		t.Fatal("expected fuzzy query 'lgn' to match User Login and exclude Fetch Profile")
	}

	// Execute top fuzzy match via Enter key -> loads POST User Login & closes palette
	tt.Key(0, ui.KeyEnter)
	tt.Frame()

	var open bool
	var activeReqName, activeURL string
	state.View(func(s *models.AppState) {
		open = s.CommandPalOpen
		activeReqName = s.ActiveRequest.Name
		activeURL = s.ActiveRequest.URL
	})
	if open {
		t.Fatal("expected Enter on palette item to dismiss modal")
	}
	if activeReqName != "User Login" || activeURL != "https://api.test/v1/login" {
		t.Fatalf("expected ActiveRequest switched to User Login, got name=%q url=%q", activeReqName, activeURL)
	}

	// Re-open, navigate via arrow keys to switch environment to STG
	state.OpenCommandPalette()
	state.SetPaletteQuery("ENV")
	tt.Frame()

	items := state.FilteredPaletteItems()
	stgIdx := -1
	for i, it := range items {
		if it.Category == "Environment" && it.Title == "STG" {
			stgIdx = i
			break
		}
	}
	if stgIdx < 0 {
		t.Fatalf("expected STG environment in ENV search results: %+v", items)
	}
	for i := 0; i < stgIdx; i++ {
		tt.Key(0, ui.KeyDown)
		tt.Frame()
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()

	var activeEnv string
	state.View(func(s *models.AppState) {
		activeEnv = s.ActiveEnv
	})
	if activeEnv != "STG" {
		t.Fatalf("expected arrow+Enter selection to switch environment to STG, got %q", activeEnv)
	}

	// Verify command execution switches workspace to GraphQL Explorer
	state.OpenCommandPalette()
	state.SetPaletteQuery("GraphQL")
	tt.Frame()
	if err := tt.Click("GraphQL Explorer"); err != nil {
		t.Fatalf("click GraphQL Explorer command: %v", err)
	}
	tt.Frame()

	var mode string
	state.View(func(s *models.AppState) {
		mode = s.WorkspaceMode
	})
	if mode != "graphql" {
		t.Fatalf("expected GraphQL Explorer palette action to set WorkspaceMode='graphql', got %q", mode)
	}
}
