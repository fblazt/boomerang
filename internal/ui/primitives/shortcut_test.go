package primitives

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestShortcutCap(t *testing.T) {
	shortcuts := []string{"⌘↵", "⌘S", "Esc", "⌘K", "⌘1"}

	for _, sc := range shortcuts {
		t.Run(sc, func(t *testing.T) {
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Row(c).Children(func() {
					ShortcutCap(c, sc)
				})
			}, 200, 100)
			tt.Frame()

			if !tt.HasText(sc) {
				t.Fatalf("expected shortcut cap to display %q", sc)
			}
		})
	}
}

func TestVariableChip(t *testing.T) {
	tokens := []string{"{{base_url}}", "{{jwt_token}}", "{{api_key}}"}

	for _, token := range tokens {
		t.Run(token, func(t *testing.T) {
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Row(c).Children(func() {
					VariableChip(c, token)
				})
			}, 300, 100)
			tt.Frame()

			if !tt.HasText(token) {
				t.Fatalf("expected variable chip to display %q", token)
			}
		})
	}
}
