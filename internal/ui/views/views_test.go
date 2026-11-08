package views

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
)

func TestShellView(t *testing.T) {
	state := models.NewAppState(nil)
	state.ActiveRequest.Name = "Test Request"
	state.ActiveRequest.Method = "GET"
	state.ActiveRequest.URL = "https://api.internal.net/v1/test"

	tt := ui.NewTester(func(c *ui.Context) {
		ShellView(c, state)
	}, 1440, 900)
	tt.Frame()

	// Assert Header
	if !tt.HasText("Boomerang") {
		t.Fatal("expected Boomerang header text")
	}
	if !tt.HasText("PROD") {
		t.Fatal("expected PROD environment badge")
	}
	if !tt.HasText("⌘K") {
		t.Fatal("expected ⌘K shortcut cap")
	}

	// Assert Sidebar
	if !tt.HasText("COLLECTIONS") {
		t.Fatal("expected COLLECTIONS sidebar section")
	}
	if !tt.HasText("REALTIME STREAMS") {
		t.Fatal("expected REALTIME STREAMS sidebar section")
	}
	if !tt.HasText("Users Service") {
		t.Fatal("expected Users Service collection item")
	}

	// Assert Main Workspace
	if !tt.HasText("Send") {
		t.Fatal("expected Send button")
	}
	if !tt.HasText("{{base_url}}") {
		t.Fatal("expected {{base_url}} variable chip")
	}

	// Assert Status Bar
	if !tt.HasText("Engine Online") {
		t.Fatal("expected Engine Online status indicator")
	}
	if !tt.HasText("Sockets:") {
		t.Fatal("expected Sockets: label")
	}
	if !tt.HasText("Ping:") {
		t.Fatal("expected Ping: label")
	}
}

func TestHeaderView_EnvToggle(t *testing.T) {
	state := models.NewAppState(nil)
	state.ActiveEnv = "PROD"

	tt := ui.NewTester(func(c *ui.Context) {
		HeaderView(c, state)
	}, 1000, 44)
	tt.Frame()

	if !tt.HasText("PROD") {
		t.Fatal("expected PROD")
	}

	if err := tt.Click("PROD"); err != nil {
		t.Fatalf("failed to click PROD: %v", err)
	}
	tt.Frame()

	if state.ActiveEnv != "STG" {
		t.Fatalf("expected ActiveEnv to change to STG, got %s", state.ActiveEnv)
	}
	if !tt.HasText("STG") {
		t.Fatal("expected UI to show STG")
	}
}

func TestSidebarView_ItemSelection(t *testing.T) {
	state := models.NewAppState(nil)

	tt := ui.NewTester(func(c *ui.Context) {
		SidebarView(c, state)
	}, 260, 600)
	tt.Frame()

	if err := tt.Click("Auth Verify"); err != nil {
		t.Fatalf("failed to click Auth Verify: %v", err)
	}
	tt.Frame()

	if state.ActiveRequest.Name != "Auth Verify" {
		t.Fatalf("expected ActiveRequest.Name to be 'Auth Verify', got %q", state.ActiveRequest.Name)
	}
	if state.ActiveRequest.Method != "POST" {
		t.Fatalf("expected ActiveRequest.Method to be 'POST', got %q", state.ActiveRequest.Method)
	}
}
