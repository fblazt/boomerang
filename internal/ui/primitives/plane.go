package primitives

import (
	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

// ElevatedPlane builds a floating surface container with linear gradient,
// subtle specular white rim border, and soft deep drop shadow.
func ElevatedPlane(c *ui.Context, children func()) ui.Element {
	return ui.Column(c).
		Radius(12).
		Padding(16).
		LinearGradient(ui.LinearGradient{
			From:  theme.SurfacePlaneTop,
			To:    theme.SurfacePlaneBottom,
			Angle: 180,
		}).
		Border(1, theme.BorderMedium).
		Shadow(0, 4, 12, 0, ui.RGBA(0, 0, 0, 0.45)).
		Children(children)
}

// SubPlane builds a secondary elevated plane container (e.g. for sidebar, panels).
func SubPlane(c *ui.Context, children func()) ui.Element {
	return ui.Column(c).
		Radius(8).
		Padding(12).
		Background(theme.SurfaceSubPlane).
		Border(1, theme.BorderSubtle).
		Children(children)
}

// InputPlane builds an elevated input container with cyan specular focus rim or red danger rim.
func InputPlane(c *ui.Context, focused bool, hasError bool, children func()) ui.Element {
	borderColor := theme.BorderMedium
	shadowColor := ui.RGBA(0, 0, 0, 0.45)
	shadowBlur := float32(12)

	if focused {
		borderColor = theme.RimFocus
		shadowColor = theme.RimFocusGlow
		shadowBlur = 14
	} else if hasError {
		borderColor = theme.RimDanger
		shadowColor = theme.RimDangerGlow
	}

	return ui.Row(c).
		Height(34).
		Radius(6).
		PaddingX(10).
		LinearGradient(ui.LinearGradient{
			From:  theme.SurfacePlaneTop,
			To:    theme.SurfacePlaneBottom,
			Angle: 180,
		}).
		Border(1, borderColor).
		Shadow(0, 4, shadowBlur, 0, shadowColor).
		AlignItems(ui.Center).
		Children(children)
}
