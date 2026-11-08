package app_test

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/app"
)

func TestApp_WindowOptionsAndInitialization(t *testing.T) {
	opts := app.DefaultOptions()
	application := app.NewApp(opts)

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
	application := app.NewApp(opts)

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
	opts.Preview = "components"
	application := app.NewApp(opts)

	tt := ui.NewTester(application.View, 1440, 900)
	tt.Frame()

	if !tt.HasText("Boomerang Component Catalog") {
		t.Fatal("expected Component Catalog view in preview mode")
	}
	if !tt.HasText("1. 3D Keycap Badges") {
		t.Fatal("expected 3D Keycap Badges section in preview mode")
	}
}
