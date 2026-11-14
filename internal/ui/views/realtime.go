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

// RealtimeView renders the WebSocket & SSE Realtime Stream Inspector with live connection bar,
// colored direction frame timeline, and interactive WebSocket frame composer.
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

	if c.Shortcut(ui.Cmd, ui.KeyEnter) && session.Status == models.StreamConnected && session.Protocol == models.StreamProtocolWSS {
		_ = state.SendStreamFrame()
	}

	return ui.Column(c).
		Grow(1).
		Padding(16).
		Gap(12).
		Background(theme.CanvasDeep).
		Children(func() {
			renderRealtimeConnectionBar(c, state, session)

			renderStreamStatusStrip(c, state, session, len(frames))

			renderStreamTimelinePanel(c, state, session, frames, filter)

			if session.Protocol == models.StreamProtocolWSS {
				renderFrameComposer(c, state, session)
			}
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

			if session.Protocol == models.StreamProtocolWSS && session.Status == models.StreamConnected {
				renderStreamPillBtn(c, "Send Ping", false, func() {
					_ = state.SendStreamPing()
				})
			}

			if session.Error != "" {
				errPreview := strings.TrimSpace(session.Error)
				ui.Text(c, errPreview).
					Font(theme.FontMono).
					FontSize(theme.SizeBadge).
					TextColor(theme.RimDanger)
			}
		})
}

func renderStreamTimelinePanel(c *ui.Context, state *models.AppState, session models.StreamSession, frames []models.StreamFrame, activeFilter string) {
	primitives.ElevatedPlane(c, func() {
		ui.Column(c).
			Grow(1).
			Children(func() {
				// Timeline Filter Header
				ui.Row(c).
					Height(38).
					PaddingX(14).
					Background(theme.SurfaceSubPlane).
					BorderWidth(0, 0, 1, 0).
					BorderColor(theme.BorderSubtle).
					Radius(8, 8, 0, 0).
					AlignItems(ui.Center).
					Gap(8).
					Children(func() {
						ui.Text(c, fmt.Sprintf("Stream Log (%s)", session.Protocol)).
							Font(theme.FontSans).
							FontSize(theme.SizeBody).
							Bold().
							TextColor(theme.TextPrimary)

						ui.Box(c).Grow(1)

						for _, dir := range []string{"SENT", "RECV", "PING"} {
							d := dir
							renderStreamPillBtn(c, d, activeFilter == d, func() {
								state.SetStreamDirectionFilter(d)
							})
						}

						renderStreamPillBtn(c, "Clear Log", false, func() {
							state.ClearStreamFrames()
						})
					})

				// Scrollable Colored Frame List
				ui.Scroll(c).
					Grow(1).
					Padding(12).
					Gap(6).
					Children(func() {
						visible := filterStreamFrames(frames, activeFilter)
						if len(visible) == 0 {
							ui.Column(c).
								Grow(1).
								Center().
								Gap(4).
								Children(func() {
									ui.Text(c, "No stream frames yet").
										Font(theme.FontSans).
										FontSize(theme.SizeBody).
										TextColor(theme.TextSecondary)
									ui.Text(c, fmt.Sprintf("Connect to %s to capture live frames.", session.URL)).
										Font(theme.FontSans).
										FontSize(theme.SizeBase).
										TextColor(theme.TextDim)
								})
							return
						}

						for _, f := range visible {
							renderStreamFrameRow(c, f)
						}
					})
			})
	}).Grow(1)
}

func renderStreamFrameRow(c *ui.Context, f models.StreamFrame) {
	dirLabel, dirColor := streamDirectionBadge(f)
	opLabel := streamOpcodeLabel(f)

	ui.Row(c).
		Padding(8, 10, 8, 10).
		Radius(6).
		Background(theme.SurfaceCardTop).
		Border(1, theme.BorderSubtle).
		AlignItems(ui.Center).
		Gap(10).
		Children(func() {
			// Direction Pill
			ui.Row(c).
				Height(20).
				PaddingX(7).
				Radius(4).
				Background(theme.CanvasDeep).
				Border(1, dirColor).
				Center().
				Children(func() {
					ui.Text(c, dirLabel).
						Font(theme.FontMono).
						FontSize(theme.SizeShortcut).
						Bold().
						TextColor(dirColor)
				})

			// Opcode Chip
			ui.Row(c).
				Height(20).
				PaddingX(6).
				Radius(4).
				Background(theme.SurfaceSubPlane).
				Border(1, theme.BorderSubtle).
				Center().
				Children(func() {
					ui.Text(c, opLabel).
						Font(theme.FontMono).
						FontSize(theme.SizeShortcut).
						TextColor(theme.TextSecondary)
				})

			// Payload preview
			ui.Text(c, f.Payload).
				Font(theme.FontMono).
				FontSize(theme.SizeBase).
				TextColor(theme.TextPrimary).
				Grow(1)

			if f.LatencyMs > 0 {
				ui.Text(c, fmt.Sprintf("%d ms", f.LatencyMs)).
					Font(theme.FontMono).
					FontSize(theme.SizeShortcut).
					TextColor(ui.RGB(0xFB, 0xBF, 0x24))
			}

			ui.Text(c, fmt.Sprintf("%d B", f.SizeBytes)).
				Font(theme.FontMono).
				FontSize(theme.SizeShortcut).
				TextColor(theme.TextMuted)

			ui.Text(c, f.Timestamp.Local().Format("15:04:05")).
				Font(theme.FontMono).
				FontSize(theme.SizeShortcut).
				TextColor(theme.TextDim)
		})
}

func renderFrameComposer(c *ui.Context, state *models.AppState, session models.StreamSession) {
	body := session.ComposerBody
	primitives.ElevatedPlane(c, func() {
		ui.Column(c).
			Height(148).
			Children(func() {
				ui.Row(c).
					Height(34).
					PaddingX(12).
					Background(theme.SurfaceSubPlane).
					BorderWidth(0, 0, 1, 0).
					BorderColor(theme.BorderSubtle).
					Radius(8, 8, 0, 0).
					AlignItems(ui.Center).
					Gap(8).
					Children(func() {
						ui.Text(c, "FRAME COMPOSER").
							Font(theme.FontMono).
							FontSize(theme.SizeBadge).
							Bold().
							TextColor(theme.TextDim)

						renderStreamPillBtn(c, "TEXT", session.ComposerType != string(models.OpcodeBinary), func() {
							state.SetStreamComposerType(models.OpcodeText)
						})
						renderStreamPillBtn(c, "BINARY", session.ComposerType == string(models.OpcodeBinary), func() {
							state.SetStreamComposerType(models.OpcodeBinary)
						})

						ui.Box(c).Grow(1)

						pushBtn := ui.Row(c).
							Height(26).
							PaddingX(10).
							Radius(5).
							Gap(6).
							LinearGradient(ui.LinearGradient{
								From:  ui.RGB(0x3B, 0x82, 0xF6),
								To:    ui.RGB(0x25, 0x63, 0xEB),
								Angle: 180,
							}).
							Border(1, ui.RGBA(255, 255, 255, 0.22)).
							AlignItems(ui.Center).
							Focusable()

						if pushBtn.Clicked() {
							_ = state.SendStreamFrame()
						}

						pushBtn.Children(func() {
							ui.Text(c, "↑ Push Frame").
								Font(theme.FontSans).
								FontSize(theme.SizeBase).
								Bold().
								TextColor(theme.TextPrimary)
							primitives.ShortcutCap(c, "⌘↵")
						})
					})

				ui.Row(c).
					Grow(1).
					Padding(10).
					Background(theme.CanvasDeep).
					Radius(0, 0, 8, 8).
					Children(func() {
						in := ui.EditableText(c, &body).
							Font(theme.FontMono).
							FontSize(theme.SizeBase).
							TextColor(theme.TextPrimary).
							Grow(1)
						if in.Changed() {
							state.SetStreamComposerBody(body)
						}
					})
			})
	})
}

func renderStreamPillBtn(c *ui.Context, label string, active bool, onClick func()) {
	bg := theme.SurfaceCardTop
	border := theme.BorderSubtle
	fg := theme.TextSecondary
	if active {
		bg = theme.SurfaceActive
		border = theme.RimFocus
		fg = theme.TextPrimary
	}
	btn := ui.Row(c).
		Height(22).
		PaddingX(8).
		Radius(4).
		Background(bg).
		Border(1, border).
		Center().
		Focusable()
	if btn.Clicked() && onClick != nil {
		onClick()
	}
	btn.Children(func() {
		ui.Text(c, label).
			Font(theme.FontMono).
			FontSize(theme.SizeShortcut).
			TextColor(fg)
	})
}

func filterStreamFrames(frames []models.StreamFrame, dirFilter string) []models.StreamFrame {
	if dirFilter == "" {
		return frames
	}
	var out []models.StreamFrame
	for _, f := range frames {
		if string(f.Direction) == dirFilter {
			out = append(out, f)
		}
	}
	return out
}

func streamDirectionBadge(f models.StreamFrame) (string, ui.Color) {
	switch f.Direction {
	case models.DirectionSent:
		return "↑ SENT", ui.RGB(0x38, 0xBD, 0xF8)
	case models.DirectionRecv:
		return "↓ RECV", ui.RGB(0x34, 0xD3, 0x99)
	case models.DirectionPing:
		if f.Opcode == models.OpcodePong {
			return "↕ PONG", ui.RGB(0xFB, 0xBF, 0x24)
		}
		return "↕ PING", ui.RGB(0xFB, 0xBF, 0x24)
	default:
		return "• SYS", theme.TextMuted
	}
}

func streamOpcodeLabel(f models.StreamFrame) string {
	switch f.Opcode {
	case models.OpcodeText:
		return "Text 1"
	case models.OpcodeBinary:
		return "Binary 2"
	case models.OpcodePing:
		return "Ping 9"
	case models.OpcodePong:
		return "Pong 10"
	case models.OpcodeEvent:
		if f.EventName != "" {
			return "SSE: " + f.EventName
		}
		return "SSE Event"
	default:
		return string(f.Opcode)
	}
}
