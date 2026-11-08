package primitives

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestElevatedPlane(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		ElevatedPlane(c, func() {
			ui.Text(c, "Plane Content")
		})
	}, 400, 300)
	tt.Frame()

	if !tt.HasText("Plane Content") {
		t.Fatal("expected plane content to be rendered")
	}
}

func TestInputPlane(t *testing.T) {
	tests := []struct {
		name     string
		focused  bool
		hasError bool
		text     string
	}{
		{"Normal", false, false, "Normal Input"},
		{"Focused", true, false, "Focused Input"},
		{"Error", false, true, "Error Input"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val := tc.text
			tt := ui.NewTester(func(c *ui.Context) {
				InputPlane(c, tc.focused, tc.hasError, func() {
					ui.Text(c, val)
				})
			}, 400, 100)
			tt.Frame()

			if !tt.HasText(tc.text) {
				t.Fatalf("expected input plane to display %q, got texts: %v", tc.text, tt.Texts())
			}
		})
	}
}

func TestButtons(t *testing.T) {
	var primaryClicked bool
	var splitMainClicked bool
	var splitMenuClicked bool
	var iconClicked bool

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Children(func() {
			PrimaryButton(c, "Send Request", func() {
				primaryClicked = true
			})
			SplitButton(c, "Save", func() {
				splitMainClicked = true
			}, func() {
				splitMenuClicked = true
			})
			IconButton(c, "⚙", "Settings", func() {
				iconClicked = true
			})
		})
	}, 600, 100)
	tt.Frame()

	if !tt.HasText("Send Request") {
		t.Fatal("expected Send Request button")
	}
	if !tt.HasText("Save") {
		t.Fatal("expected Save button")
	}
	if !tt.HasText("⚙") {
		t.Fatal("expected icon button")
	}

	if err := tt.Click("Send Request"); err != nil {
		t.Fatalf("click Send Request failed: %v", err)
	}
	tt.Frame()
	if !primaryClicked {
		t.Fatal("expected primaryClicked to be true")
	}

	if err := tt.Click("Save"); err != nil {
		t.Fatalf("click Save failed: %v", err)
	}
	tt.Frame()
	if !splitMainClicked {
		t.Fatal("expected splitMainClicked to be true")
	}

	if err := tt.Click("▾"); err != nil {
		t.Fatalf("click ▾ failed: %v", err)
	}
	tt.Frame()
	if !splitMenuClicked {
		t.Fatal("expected splitMenuClicked to be true")
	}

	if err := tt.Click("⚙"); err != nil {
		t.Fatalf("click ⚙ failed: %v", err)
	}
	tt.Frame()
	if !iconClicked {
		t.Fatal("expected iconClicked to be true")
	}
}
