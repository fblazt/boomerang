package primitives

import (
	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

// ShortcutCap renders a 3D tactile keyboard shortcut keycap (e.g. ⌘↵, ⌘S, Esc, ⌘K).
func ShortcutCap(c *ui.Context, shortcut string) ui.Element {
	return ui.Row(c).
		Height(18).
		PaddingX(5).
		Radius(4).
		LinearGradient(ui.LinearGradient{
			From:  ui.RGB(0x25, 0x29, 0x38),
			To:    ui.RGB(0x19, 0x1C, 0x27),
			Angle: 180,
		}).
		Border(1, theme.BorderStrong).
		Shadow(0, 2, 0, 0, ui.RGB(0x07, 0x08, 0x0C)).
		Center().
		Children(func() {
			ui.Text(c, shortcut).
				Font(theme.FontMono).
				FontSize(theme.SizeShortcut).
				Bold().
				TextColor(theme.TextSecondary)
		})
}
