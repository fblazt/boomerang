package catalog

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestComponentCatalogView(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		ComponentCatalogView(c)
	}, 1440, 900)
	tt.Frame()

	// Assert title
	if !tt.HasText("Boomerang Component Catalog") {
		t.Fatal("expected catalog title")
	}

	// Assert keycaps section
	if !tt.HasText("1. 3D Keycap Badges") {
		t.Fatal("expected keycaps section header")
	}
	if !tt.HasText("GET") || !tt.HasText("POST") || !tt.HasText("GQL") || !tt.HasText("SSE") {
		t.Fatal("expected HTTP verb keycaps")
	}
	if !tt.HasText("200 OK") || !tt.HasText("502 ERR") {
		t.Fatal("expected status code badges")
	}

	// Assert planes section
	if !tt.HasText("2. Elevated Surface Planes & Input Containers") {
		t.Fatal("expected planes section header")
	}
	if !tt.HasText("Standard Input Plane") || !tt.HasText("Focused Input (Cyan Rim)") {
		t.Fatal("expected input planes")
	}

	// Assert buttons section
	if !tt.HasText("3. Tactile Action Buttons") {
		t.Fatal("expected buttons section header")
	}
	if !tt.HasText("Primary Action") || !tt.HasText("Save & Dispatch") {
		t.Fatal("expected buttons")
	}

	// Assert chips and shortcuts section
	if !tt.HasText("4. Keyboard Shortcut Caps & Variable Token Chips") {
		t.Fatal("expected chips and shortcuts section header")
	}
	if !tt.HasText("{{base_url}}") || !tt.HasText("⌘↵") || !tt.HasText("⌘K") {
		t.Fatal("expected variable chip and shortcut caps")
	}
}
