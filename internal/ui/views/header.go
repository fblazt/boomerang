package views

import (
	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/primitives"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

// HeaderView renders the top application titlebar with drag offset, workspace breadcrumb,
// environment switcher, and quick action shortcuts.
func HeaderView(c *ui.Context, state *models.AppState) ui.Element {
	return ui.Row(c).
		Height(44).
		PaddingX(16).
		Background(theme.SurfaceSubPlane).
		BorderWidth(0, 0, 1, 0).
		BorderColor(theme.BorderSubtle).
		AlignItems(ui.Center).
		Children(func() {
			// Window controls spacer (macOS traffic lights spacing)
			ui.Box(c).Width(64)

			// Boomerang branding
			ui.Row(c).
				Gap(8).
				AlignItems(ui.Center).
				Children(func() {
					ui.Text(c, "Boomerang").
						Font(theme.FontSans).
						FontSize(theme.SizeHeading).
						Bold().
						TextColor(theme.TextPrimary)

					ui.Box(c).Width(1).Height(14).Background(theme.BorderMedium)

					// Breadcrumb path: Workspace / Active Collection
					ui.Text(c, "Workspace").
						Font(theme.FontSans).
						FontSize(theme.SizeBase).
						TextColor(theme.TextMuted)

					ui.Text(c, "/").
						Font(theme.FontSans).
						FontSize(theme.SizeBase).
						TextColor(theme.TextDim)

					state.View(func(s *models.AppState) {
						ui.Text(c, s.ActiveRequest.Name).
							Font(theme.FontSans).
							FontSize(theme.SizeBase).
							TextColor(theme.TextSecondary)
					})
				})

			// Flexible Spacer
			ui.Box(c).Grow(1)

			// Environment Switcher & Quick Palette Trigger
			ui.Row(c).
				Gap(10).
				AlignItems(ui.Center).
				Children(func() {
					// Command Palette shortcut cap
					ui.Row(c).
						Gap(4).
						AlignItems(ui.Center).
						Children(func() {
							primitives.ShortcutCap(c, "⌘K")
						})

					// Active Environment Indicator / Switcher Pill
					var currentEnv string
					state.View(func(s *models.AppState) {
						currentEnv = s.ActiveEnv
					})

					envBox := ui.Row(c).
						Height(24).
						PaddingX(8).
						Radius(5).
						LinearGradient(ui.LinearGradient{
							From:  theme.SurfaceCardTop,
							To:    theme.SurfaceCardBottom,
							Angle: 180,
						}).
						Border(1, theme.BorderSubtle).
						Shadow(0, 1, 0, 0, ui.RGB(0x06, 0x07, 0x0A)).
						AlignItems(ui.Center).
						Gap(6).
						Focusable()

					if envBox.Clicked() {
						// Cycle environment on click: PROD -> STG -> DEV -> LOCAL -> PROD
						nextEnv := "PROD"
						switch currentEnv {
						case "PROD":
							nextEnv = "STG"
						case "STG":
							nextEnv = "DEV"
						case "DEV":
							nextEnv = "LOCAL"
						default:
							nextEnv = "PROD"
						}
						state.Mutate(func(st *models.AppState) {
							st.ActiveEnv = nextEnv
						})
					}

					envBox.Children(func() {
						// Status indicator dot
						dotColor := ui.RGB(0x34, 0xD3, 0x99) // Emerald for PROD
						if currentEnv == "STG" {
							dotColor = ui.RGB(0xFB, 0xBF, 0x24) // Amber
						} else if currentEnv == "DEV" || currentEnv == "LOCAL" {
							dotColor = ui.RGB(0x38, 0xBD, 0xF8) // Cyan
						}
						primitives.StatusDot(c, dotColor)

						ui.Text(c, currentEnv).
							Font(theme.FontMono).
							FontSize(theme.SizeBadge).
							Bold().
							TextColor(theme.TextPrimary)

						primitives.ShortcutCap(c, "⌘1")
					})
				})
		})
}
