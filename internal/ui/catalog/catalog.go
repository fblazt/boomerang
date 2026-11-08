package catalog

import (
	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/primitives"
	"github.com/fblazt/boomerang/internal/ui/theme"
	"github.com/fblazt/boomerang/internal/ui/views"
)

// ComponentCatalogView renders an interactive preview showcase of all design system components.
func ComponentCatalogView(c *ui.Context) ui.Element {
	return ui.Column(c).
		Grow(1).
		Padding(24).
		Gap(20).
		Background(theme.CanvasBase).
		Children(func() {
			// Title Header
			ui.Row(c).
				AlignItems(ui.Center).
				Children(func() {
					ui.Text(c, "Boomerang Component Catalog").
						Font(theme.FontSans).
						FontSize(theme.SizeTitle).
						Bold().
						TextColor(theme.TextPrimary)

					ui.Box(c).Grow(1)

					primitives.ShortcutCap(c, "Esc to Exit")
				})

			// Catalog Content Container
			ui.Column(c).
				Grow(1).
				Gap(20).
				Children(func() {
					// 1. 3D Keycap Badges (HTTP Verbs & Status Codes)
					renderKeycapsSection(c)

					// 2. Elevated Planes & Input Planes
					renderPlanesAndInputsSection(c)

					// 3. Tactile Buttons (Primary, Split, Icon)
					renderButtonsSection(c)

					// 4. Keyboard Shortcut Caps & Variable Token Chips
					renderChipsAndShortcutsSection(c)

					// 5. Key-Value Table & Telemetry Waterfall
					renderKVAndTelemetrySection(c)
				})
		})
}

func renderKeycapsSection(c *ui.Context) {
	primitives.ElevatedPlane(c, func() {
		ui.Column(c).
			Gap(12).
			Children(func() {
				ui.Text(c, "1. 3D Keycap Badges (HTTP Verbs, Protocols & Status Codes)").
					Font(theme.FontSans).
					FontSize(theme.SizeHeading).
					Bold().
					TextColor(theme.TextPrimary)

				// Verbs
				ui.Row(c).
					Gap(8).
					AlignItems(ui.Center).
					Children(func() {
						primitives.KeycapBadge(c, "GET")
						primitives.KeycapBadge(c, "POST")
						primitives.KeycapBadge(c, "PUT")
						primitives.KeycapBadge(c, "PATCH")
						primitives.KeycapBadge(c, "DEL")
						primitives.KeycapBadge(c, "GQL")
						primitives.KeycapBadge(c, "WSS")
						primitives.KeycapBadge(c, "SSE")
					})

				// Status codes
				ui.Row(c).
					Gap(8).
					AlignItems(ui.Center).
					Children(func() {
						primitives.StatusCodeBadge(c, 200, "OK")
						primitives.StatusCodeBadge(c, 201, "CREATED")
						primitives.StatusCodeBadge(c, 429, "LIMIT")
						primitives.StatusCodeBadge(c, 502, "ERR")
					})
			})
	})
}

func renderPlanesAndInputsSection(c *ui.Context) {
	primitives.ElevatedPlane(c, func() {
		ui.Column(c).
			Gap(12).
			Children(func() {
				ui.Text(c, "2. Elevated Surface Planes & Input Containers").
					Font(theme.FontSans).
					FontSize(theme.SizeHeading).
					Bold().
					TextColor(theme.TextPrimary)

				ui.Row(c).
					Gap(12).
					AlignItems(ui.Center).
					Children(func() {
						// Normal input
						primitives.InputPlane(c, false, false, func() {
							ui.Text(c, "Standard Input Plane").
								Font(theme.FontMono).
								FontSize(theme.SizeBody).
								TextColor(theme.TextSecondary)
						})

						// Cyan focused input
						primitives.InputPlane(c, true, false, func() {
							ui.Text(c, "Focused Input (Cyan Rim)").
								Font(theme.FontMono).
								FontSize(theme.SizeBody).
								TextColor(theme.TextPrimary)
						})

						// Error red input
						primitives.InputPlane(c, false, true, func() {
							ui.Text(c, "Error Input (Rose Rim)").
								Font(theme.FontMono).
								FontSize(theme.SizeBody).
								TextColor(theme.RimDanger)
						})
					})
			})
	})
}

func renderButtonsSection(c *ui.Context) {
	primitives.ElevatedPlane(c, func() {
		ui.Column(c).
			Gap(12).
			Children(func() {
				ui.Text(c, "3. Tactile Action Buttons").
					Font(theme.FontSans).
					FontSize(theme.SizeHeading).
					Bold().
					TextColor(theme.TextPrimary)

				ui.Row(c).
					Gap(12).
					AlignItems(ui.Center).
					Children(func() {
						primitives.PrimaryButton(c, "Primary Action", nil)
						primitives.SplitButton(c, "Save & Dispatch", nil, nil)
						primitives.IconButton(c, "⚡", "Quick Action", nil)
						primitives.IconButton(c, "⚙", "Preferences", nil)
					})
			})
	})
}

func renderChipsAndShortcutsSection(c *ui.Context) {
	primitives.ElevatedPlane(c, func() {
		ui.Column(c).
			Gap(12).
			Children(func() {
				ui.Text(c, "4. Keyboard Shortcut Caps & Variable Token Chips").
					Font(theme.FontSans).
					FontSize(theme.SizeHeading).
					Bold().
					TextColor(theme.TextPrimary)

				ui.Row(c).
					Gap(10).
					AlignItems(ui.Center).
					Children(func() {
						primitives.VariableChip(c, "{{base_url}}")
						primitives.VariableChip(c, "{{auth_token}}")
						primitives.VariableChip(c, "{{user_id}}")

						ui.Box(c).Width(1).Height(16).Background(theme.BorderMedium)

						primitives.ShortcutCap(c, "⌘↵")
						primitives.ShortcutCap(c, "⌘S")
						primitives.ShortcutCap(c, "Esc")
						primitives.ShortcutCap(c, "⌘K")
						primitives.ShortcutCap(c, "⌘1")
					})
			})
	})
}

func renderKVAndTelemetrySection(c *ui.Context) {
	primitives.ElevatedPlane(c, func() {
		ui.Column(c).
			Gap(12).
			Children(func() {
				ui.Text(c, "5. Key-Value Table & Telemetry Waterfall").
					Font(theme.FontSans).
					FontSize(theme.SizeHeading).
					Bold().
					TextColor(theme.TextPrimary)

				ui.Row(c).
					Gap(10).
					AlignItems(ui.Center).
					Children(func() {
						primitives.SendButton(c, nil)
						primitives.CancelButton(c, nil)
					})

				views.TelemetryWaterfallBar(c, models.Telemetry{
					DNS:      12,
					TCP:      19,
					TLS:      28,
					TTFB:     42,
					Download: 19,
					Total:    120,
				})
			})
	})
}
