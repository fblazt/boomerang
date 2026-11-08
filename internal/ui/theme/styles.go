package theme

import "github.com/egoist/mygo/ui"

// NewBoomerangTheme returns the customized dark theme matching Boomerang design system.
func NewBoomerangTheme() *ui.Theme {
	t := ui.DarkTheme()
	t.Dark = true
	t.Background = CanvasBase
	t.Surface = SurfacePlaneTop
	t.SurfaceHover = SurfaceHover
	t.SurfacePressed = SurfaceActive
	t.Border = BorderSubtle
	t.Text = TextPrimary
	t.TextMuted = TextMuted
	t.Accent = RimFocus
	t.AccentHover = ui.RGB(0x7D, 0xD3, 0xFC)
	t.AccentPressed = ui.RGB(0x02, 0x84, 0xC7)
	t.AccentText = CanvasBase
	t.InactiveAccent = SurfaceActive
	t.Danger = RimDanger
	t.Warning = ui.RGB(0xFB, 0xBF, 0x24)
	t.Success = ui.RGB(0x34, 0xD3, 0x99)
	t.Selection = ui.RGBA(0x38, 0xBD, 0xF8, 0.30)
	t.Focus = RimFocus
	t.FocusRingWidth = 2
	t.FocusRingOffset = 1
	t.Radius = 6
	t.Spacing = 4
	t.FontSize = SizeBase
	t.Font = FontSans
	return t
}
