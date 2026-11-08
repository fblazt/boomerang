package primitives

import (
	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

// VariableChip renders an electric cyan pill chip for template variables (e.g. {{base_url}}).
func VariableChip(c *ui.Context, token string) ui.Element {
	return ui.Row(c).
		Height(20).
		PaddingX(6).
		Radius(4).
		Background(ui.RGBA(0x38, 0xBD, 0xF8, 0.14)).
		Border(1, ui.RGBA(0x38, 0xBD, 0xF8, 0.30)).
		Center().
		Children(func() {
			ui.Text(c, token).
				Font(theme.FontMono).
				FontSize(theme.SizeToken).
				TextColor(theme.RimFocus)
		})
}

// StatusDot renders an indicator dot (green for connected/prod, red for error, amber for warning).
func StatusDot(c *ui.Context, color ui.Color) ui.Element {
	return ui.Box(c).
		Width(6).
		Height(6).
		Radius(3).
		Background(color)
}
