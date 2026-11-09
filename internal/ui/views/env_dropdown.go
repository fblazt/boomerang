package views

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/primitives"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

// EnvironmentColor returns the standard theme status color for an environment name.
func EnvironmentColor(envName string) ui.Color {
	switch envName {
	case "PROD":
		return ui.RGB(0x34, 0xD3, 0x99) // Emerald
	case "STG":
		return ui.RGB(0xFB, 0xBF, 0x24) // Amber
	case "DEV":
		return ui.RGB(0x38, 0xBD, 0xF8) // Cyan
	case "LOCAL":
		return ui.RGB(0xA8, 0x55, 0xF7) // Purple
	default:
		return theme.RimFocus
	}
}

// EnvironmentDropdownMenu renders the elevated floating dropdown menu for switching environments.
func EnvironmentDropdownMenu(c *ui.Context, state *models.AppState) ui.Element {
	var currentEnv string
	var envs []models.Environment

	state.View(func(s *models.AppState) {
		currentEnv = s.ActiveEnv
		envs = s.Environments
	})

	return ui.Column(c).
		Width(280).
		Padding(10).
		Radius(8).
		LinearGradient(ui.LinearGradient{
			From:  theme.SurfacePlaneTop,
			To:    theme.SurfacePlaneBottom,
			Angle: 180,
		}).
		Border(1, theme.BorderMedium).
		Shadow(0, 8, 24, 0, ui.RGBA(0, 0, 0, 0.65)).
		Gap(6).
		Children(func() {
			// Header Label
			ui.Row(c).
				PaddingX(6).
				PaddingY(4).
				Children(func() {
					ui.Text(c, "WORKSPACE ENVIRONMENTS").
						Font(theme.FontMono).
						FontSize(9).
						Bold().
						TextColor(theme.TextDim)
				})

			// Environment Items List
			for i, env := range envs {
				envCopy := env
				idx := i
				isActive := envCopy.Name == currentEnv
				dotColor := EnvironmentColor(envCopy.Name)
				shortcut := ""
				if idx < 4 {
					shortcut = fmt.Sprintf("⌘%d", idx+1)
				}

				itemRow := ui.Row(c).
					Height(32).
					PaddingX(8).
					Radius(6).
					AlignItems(ui.Center).
					Gap(8).
					Focusable()

				if isActive {
					itemRow.Background(theme.SurfaceActive).
						Border(1, theme.BorderSubtle)
				}

				if itemRow.Clicked() {
					state.SetActiveEnv(envCopy.Name)
					state.CloseEnvDropdown()
				}

				itemRow.Children(func() {
					// Status dot
					primitives.StatusDot(c, dotColor)

					// Environment Name
					ui.Text(c, envCopy.Name).
						Font(theme.FontMono).
						FontSize(theme.SizeBody).
						Bold().
						TextColor(theme.TextPrimary)

					ui.Box(c).Grow(1)

					// Active indicator or variable count
					if isActive {
						ui.Text(c, "✓").
							Font(theme.FontSans).
							FontSize(12).
							Bold().
							TextColor(dotColor)
						ui.Box(c).Width(4)
					}

					// Shortcut cap (⌘1, ⌘2, ⌘3, ⌘4)
					if shortcut != "" {
						primitives.ShortcutCap(c, shortcut)
					}
				})
			}

			// Divider Line
			ui.Box(c).
				Height(1).
				Background(theme.BorderSubtle).
				MarginY(4)

			// Manage Vault / Environments Action
			manageBtn := ui.Row(c).
				Height(32).
				PaddingX(8).
				Radius(6).
				AlignItems(ui.Center).
				Gap(8).
				Focusable()

			if manageBtn.Clicked() {
				state.CloseEnvDropdown()
				state.OpenVaultModal()
			}

			manageBtn.Children(func() {
				ui.Text(c, "⚙").
					Font(theme.FontSans).
					FontSize(12).
					TextColor(theme.TextMuted)

				ui.Text(c, "Manage Environments & Secret Vault...").
					Font(theme.FontSans).
					FontSize(theme.SizeBadge).
					TextColor(theme.TextSecondary)

				ui.Box(c).Grow(1)

				primitives.ShortcutCap(c, "⌘E")
			})
		})
}
