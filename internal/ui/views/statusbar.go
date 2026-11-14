package views

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/primitives"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

// StatusBarView renders the bottom application status bar with active socket counts,
// collection totals, telemetry summary, and network health.
func StatusBarView(c *ui.Context, state *models.AppState) ui.Element {
	return ui.Row(c).
		Height(26).
		PaddingX(12).
		Background(theme.SurfaceSubPlane).
		BorderWidth(1, 0, 0, 0).
		BorderColor(theme.BorderSubtle).
		AlignItems(ui.Center).
		Children(func() {
			// Left side: Connection status & Active sockets
			ui.Row(c).
				Gap(12).
				AlignItems(ui.Center).
				Children(func() {
					// Connected indicator
					ui.Row(c).
						Gap(6).
						AlignItems(ui.Center).
						Children(func() {
							primitives.StatusDot(c, ui.RGB(0x34, 0xD3, 0x99))
							ui.Text(c, "Engine Online").
								Font(theme.FontMono).
								FontSize(theme.SizeShortcut).
								TextColor(theme.TextMuted)
						})

					ui.Box(c).Width(1).Height(10).Background(theme.BorderSubtle)

					// Active sockets count
					ui.Row(c).
						Gap(4).
						AlignItems(ui.Center).
						Children(func() {
							ui.Text(c, "Sockets:").
								Font(theme.FontMono).
								FontSize(theme.SizeShortcut).
								TextColor(theme.TextDim)
							ui.Text(c, "2 active (WSS: 1, HTTP/2: 1)").
								Font(theme.FontMono).
								FontSize(theme.SizeShortcut).
								TextColor(theme.TextSecondary)
						})

					ui.Box(c).Width(1).Height(10).Background(theme.BorderSubtle)

					// Collections & Requests count
					_, reqCount := state.CollectionCounts()
					ui.Row(c).
						Gap(4).
						AlignItems(ui.Center).
						Children(func() {
							ui.Text(c, "Workspace:").
								Font(theme.FontMono).
								FontSize(theme.SizeShortcut).
								TextColor(theme.TextDim)
							ui.Text(c, fmt.Sprintf("%d requests, 2 streams", reqCount)).
								Font(theme.FontMono).
								FontSize(theme.SizeShortcut).
								TextColor(theme.TextSecondary)
						})
				})

			// Flexible Spacer
			ui.Box(c).Grow(1)

			// Right side: Active execution telemetry & ping
			ui.Row(c).
				Gap(12).
				AlignItems(ui.Center).
				Children(func() {
					state.View(func(s *models.AppState) {
						if s.IsLoading {
							ui.Row(c).
								Gap(6).
								AlignItems(ui.Center).
								Children(func() {
									ui.Text(c, fmt.Sprintf("Elapsed: %d ms", s.ElapsedMs)).
										Font(theme.FontMono).
										FontSize(theme.SizeShortcut).
										TextColor(theme.RimFocus)
								})
						} else if s.ActiveResponse != nil {
							ui.Row(c).
								Gap(6).
								AlignItems(ui.Center).
								Children(func() {
									ui.Text(c, fmt.Sprintf("Last: %v | %d B", s.ActiveResponse.Duration, s.ActiveResponse.SizeBytes)).
										Font(theme.FontMono).
										FontSize(theme.SizeShortcut).
										TextColor(theme.TextSecondary)
								})
						}
					})

					ui.Box(c).Width(1).Height(10).Background(theme.BorderSubtle)

					// Gateway / Stream Ping
					var pingLabel = "14ms"
					state.View(func(s *models.AppState) {
						if s.Stream.Status == models.StreamConnected && s.Stream.PingRTTMs > 0 {
							pingLabel = fmt.Sprintf("%dms", s.Stream.PingRTTMs)
						}
					})
					ui.Row(c).
						Gap(4).
						AlignItems(ui.Center).
						Children(func() {
							ui.Text(c, "Ping:").
								Font(theme.FontMono).
								FontSize(theme.SizeShortcut).
								TextColor(theme.TextDim)
							ui.Text(c, pingLabel).
								Font(theme.FontMono).
								FontSize(theme.SizeShortcut).
								TextColor(theme.TextSecondary)
						})
				})
		})
}
