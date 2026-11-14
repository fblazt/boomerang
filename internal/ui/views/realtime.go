package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/primitives"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

// RealtimeView renders the WebSocket & SSE Realtime Stream Inspector with its live connection bar.
func RealtimeView(c *ui.Context, state *models.AppState) ui.Element {
	var (
		session models.StreamSession
		frames  []models.StreamFrame
		filter  string
	)
	state.View(func(s *models.AppState) {
		session = s.Stream
		frames = s.StreamFrames
		filter = s.StreamFilterDirection
	})

	return ui.Column(c).
		Grow(1).
		Padding(16).
		Gap(14).
		Background(theme.CanvasDeep).
		Children(func() {
			renderRealtimeConnectionBar(c, state, session)

			renderStreamStatusStrip(c, state, session, len(frames))

			// Placeholder timeline surface enhanced in Commit #26 with frame composer & direction log
			primitives.ElevatedPlane(c, func() {
				ui.Column(c).
					Grow(1).
					Padding(16).
					Gap(8).
					Children(func() {
						ui.Text(c, fmt.Sprintf("Live Stream Log (%s)", session.Protocol)).
							Font(theme.FontSans).
							FontSize(theme.SizeHeading).
							Bold().
							TextColor(theme.TextPrimary)
						if len(frames) == 0 {
							ui.Text(c, fmt.Sprintf("Connect to %s to stream realtime frames (filter=%q).", session.URL, filter)).
								Font(theme.FontSans).
								FontSize(theme.SizeBase).
								TextColor(theme.TextMuted)
						} else {
							ui.Text(c, fmt.Sprintf("%d stream frames captured.", len(frames))).
								Font(theme.FontMono).
								FontSize(theme.SizeBase).
								TextColor(theme.TextSecondary)
						}
					})
			}).Grow(1)
		})
}

func renderRealtimeConnectionBar(c *ui.Context, state *models.AppState, session models.StreamSession) {
	urlDraft := session.URL
	primitives.ElevatedPlane(c, func() {
		ui.Row(c).
			Height(48).
			PaddingX(10).
			Gap(10).
			AlignItems(ui.Center).
			Children(func() {
				// Protocol switcher pill (WSS / SSE)
				protoBtn := ui.Row(c).
					AlignItems(ui.Center).
					Gap(4).
					Focusable()
				if protoBtn.Clicked() {
					if session.Protocol == models.StreamProtocolSSE {
						state.SetStreamProtocol(models.StreamProtocolWSS)
					} else {
						state.SetStreamProtocol(models.StreamProtocolSSE)
					}
				}
				protoBtn.Children(func() {
					primitives.KeycapBadge(c, string(session.Protocol))
				})

				// URL Input Bar
				primitives.InputPlane(c, session.Status == models.StreamConnected, session.Status == models.StreamError, func() {
					urlIn := ui.EditableText(c, &urlDraft).
						Font(theme.FontMono).
						FontSize(theme.SizeBody).
						TextColor(theme.TextPrimary).
						Grow(1)
					if urlIn.Changed() {
						state.SetStreamURL(urlDraft)
					}
					if urlIn.Submitted() && session.Status != models.StreamConnected {
						state.ConnectStream()
					}
				}).Grow(1)

				// Connect / Disconnect Trigger
				switch session.Status {
				case models.StreamConnected, models.StreamConnecting:
					discBtn := ui.Row(c).
						Height(34).
						PaddingX(12).
						Gap(6).
						Radius(6).
						LinearGradient(ui.LinearGradient{
							From:  ui.RGB(0x99, 0x1B, 0x1B),
							To:    ui.RGB(0x7F, 0x1D, 0x1D),
							Angle: 180,
						}).
						Border(1, ui.RGBA(248, 113, 113, 0.40)).
						AlignItems(ui.Center).
						Focusable()
					if discBtn.Clicked() || c.Shortcut(0, ui.KeyEscape) {
						state.DisconnectStream()
					}
					discBtn.Children(func() {
						ui.Text(c, "Disconnect").
							Font(theme.FontSans).
							FontSize(theme.SizeBody).
							Bold().
							TextColor(theme.TextPrimary)
						primitives.ShortcutCap(c, "Esc")
					})
				default:
					primitives.PrimaryButton(c, "Connect", func() {
						state.ConnectStream()
					})
				}

				backBtn := ui.Row(c).
					Height(34).
					PaddingX(10).
					Radius(6).
					Background(theme.SurfaceCardTop).
					Border(1, theme.BorderSubtle).
					Center().
					Focusable()
				if backBtn.Clicked() {
					state.SetWorkspaceMode("rest")
				}
				backBtn.Children(func() {
					ui.Text(c, "REST").
						Font(theme.FontMono).
						FontSize(theme.SizeBadge).
						TextColor(theme.TextSecondary)
				})
			})
	})
}

func renderStreamStatusStrip(c *ui.Context, state *models.AppState, session models.StreamSession, frameCount int) {
	statusColor := theme.TextDim
	statusText := "DISCONNECTED"
	switch session.Status {
	case models.StreamConnecting:
		statusColor = ui.RGB(0xFB, 0xBF, 0x24)
		statusText = "CONNECTING"
	case models.StreamConnected:
		statusColor = ui.RGB(0x34, 0xD3, 0x99)
		statusText = "CONNECTED"
	case models.StreamError:
		statusColor = theme.RimDanger
		statusText = "ERROR"
	}

	pingText := "Ping: --"
	if session.PingRTTMs > 0 {
		pingText = fmt.Sprintf("Ping: %d ms", session.PingRTTMs)
	}
	durationText := "Up: 0s"
	if !session.ConnectedAt.IsZero() {
		sec := int(time.Since(session.ConnectedAt).Seconds())
		durationText = fmt.Sprintf("Up: %ds", sec)
	}

	ui.Row(c).
		Height(34).
		PaddingX(12).
		Radius(6).
		Background(theme.SurfaceSubPlane).
		Border(1, theme.BorderSubtle).
		AlignItems(ui.Center).
		Gap(14).
		Children(func() {
			ui.Row(c).
				Gap(6).
				AlignItems(ui.Center).
				Children(func() {
					primitives.StatusDot(c, statusColor)
					ui.Text(c, statusText).
						Font(theme.FontMono).
						FontSize(theme.SizeBadge).
						Bold().
						TextColor(statusColor)
				})

			ui.Text(c, pingText).
				Font(theme.FontMono).
				FontSize(theme.SizeBadge).
				TextColor(ui.RGB(0xFB, 0xBF, 0x24))

			ui.Text(c, durationText).
				Font(theme.FontMono).
				FontSize(theme.SizeBadge).
				TextColor(theme.TextSecondary)

			ui.Text(c, fmt.Sprintf("Sent: %d  •  Recv: %d  •  Frames: %d", session.SentCount, session.RecvCount, frameCount)).
				Font(theme.FontMono).
				FontSize(theme.SizeBadge).
				TextColor(theme.TextMuted)

			ui.Box(c).Grow(1)

			if session.Error != "" {
				errPreview := strings.TrimSpace(session.Error)
				ui.Text(c, errPreview).
					Font(theme.FontMono).
					FontSize(theme.SizeBadge).
					TextColor(theme.RimDanger)
			}
		})
}
