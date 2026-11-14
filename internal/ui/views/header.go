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
	// Global window keyboard shortcuts for environment switching and vault
	c.OnWindowShortcut(ui.Cmd, ui.Key1, func() { state.SwitchEnvByIndex(0) })
	c.OnWindowShortcut(ui.Cmd, ui.Key2, func() { state.SwitchEnvByIndex(1) })
	c.OnWindowShortcut(ui.Cmd, ui.Key3, func() { state.SwitchEnvByIndex(2) })
	c.OnWindowShortcut(ui.Cmd, ui.Key4, func() { state.SwitchEnvByIndex(3) })
	c.OnWindowShortcut(ui.Cmd, ui.KeyE, func() { state.OpenVaultModal() })
	c.OnWindowShortcut(ui.Cmd, ui.KeyH, func() { state.ToggleHistoryModal() })
	c.OnWindowShortcut(ui.Cmd, ui.KeyR, func() { state.ToggleRunnerModal() })
	c.OnWindowShortcut(0, ui.KeyEscape, func() {
		state.CloseEnvDropdown()
		state.CloseVaultModal()
		state.CloseHistoryModal()
		state.CloseRunnerModal()
	})

	return ui.Row(c).
		Height(44).
		PaddingX(16).
		Background(theme.SurfaceSubPlane).
		BorderWidth(0, 0, 1, 0).
		BorderColor(theme.BorderSubtle).
		AlignItems(ui.Center).
		ZIndex(100).
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

					// Active Environment Indicator / Switcher Pill with Dropdown
					var currentEnv string
					var isDropdownOpen bool
					state.View(func(s *models.AppState) {
						currentEnv = s.ActiveEnv
						isDropdownOpen = s.EnvDropdownOpen
					})

					ui.Column(c).
						Children(func() {
							ui.Row(c).
								AlignItems(ui.Center).
								Gap(2).
								Children(func() {
									// Env pill
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
										state.SetActiveEnv(nextEnv)
									}

									envBox.Children(func() {
										dotColor := EnvironmentColor(currentEnv)
										primitives.StatusDot(c, dotColor)

										ui.Text(c, currentEnv).
											Font(theme.FontMono).
											FontSize(theme.SizeBadge).
											Bold().
											TextColor(theme.TextPrimary)

										capText := "⌘1"
										switch currentEnv {
										case "STG":
											capText = "⌘2"
										case "DEV":
											capText = "⌘3"
										case "LOCAL":
											capText = "⌘4"
										}
										primitives.ShortcutCap(c, capText)
									})

									// Chevron dropdown toggle button
									chevronBtn := ui.Row(c).
										Height(24).
										Width(18).
										Center().
										Radius(4).
										Background(theme.SurfaceCardTop).
										Border(1, theme.BorderSubtle).
										Focusable()

									if chevronBtn.Clicked() {
										state.ToggleEnvDropdown()
									}

									chevronBtn.Children(func() {
										ui.Text(c, "▾").
											Font(theme.FontSans).
											FontSize(10).
											TextColor(theme.TextMuted)
									})
								})

							// Dropdown menu if open
							if isDropdownOpen {
								EnvironmentDropdownMenu(c, state)
							}
						})
				})
		})
}
