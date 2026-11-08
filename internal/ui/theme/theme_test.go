package theme_test

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

func TestTheme_ColorsAndStyles(t *testing.T) {
	bt := theme.NewBoomerangTheme()
	if !bt.Dark {
		t.Fatal("expected dark theme")
	}
	if bt.Background != theme.CanvasBase {
		t.Fatalf("expected background %v, got %v", theme.CanvasBase, bt.Background)
	}
	if bt.Font != theme.FontSans {
		t.Fatalf("expected FontSans, got %s", bt.Font)
	}

	getStyle := theme.GetVerbStyle("GET")
	if getStyle.TextColor != ui.RGB(0x34, 0xD3, 0x99) {
		t.Fatalf("unexpected GET text color: %v", getStyle.TextColor)
	}

	fallback := theme.GetVerbStyle("UNKNOWN")
	if fallback.TextColor != getStyle.TextColor {
		t.Fatalf("expected fallback to GET style")
	}
}

func TestTheme_HeadlessRender(t *testing.T) {
	rendered := false
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(theme.NewBoomerangTheme())
		ui.Column(c).
			Background(theme.CanvasBase).
			Padding(16).
			Children(func() {
				ui.Text(c, "Boomerang Theme").
					Font(theme.FontSans).
					FontSize(theme.SizeTitle).
					TextColor(theme.TextPrimary)
			})
		rendered = true
	}, 800, 600)

	tt.Frame()

	if !rendered {
		t.Fatal("expected view to render")
	}
	if !tt.HasText("Boomerang Theme") {
		t.Fatal("expected 'Boomerang Theme' text in headless tester")
	}
}
