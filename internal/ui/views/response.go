package views

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/primitives"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

// ResponseView renders the response inspection panel:
// status code, latency, payload size, HTTP protocol, telemetry waterfall, and formatted body.
func ResponseView(c *ui.Context, state *models.AppState) ui.Element {
	var resp *models.ResponseModel
	var isLoading bool
	var elapsedMs int64
	var method string

	state.View(func(s *models.AppState) {
		resp = s.ActiveResponse
		isLoading = s.IsLoading
		elapsedMs = s.ElapsedMs
		method = s.ActiveRequest.Method
	})

	return ui.Column(c).
		Grow(1).
		Radius(8).
		Background(theme.SurfaceSubPlane).
		Border(1, theme.BorderSubtle).
		Padding(14).
		Gap(10).
		Children(func() {
			if isLoading {
				renderInFlightView(c, method, elapsedMs)
				return
			}

			if resp == nil {
				renderEmptyState(c)
				return
			}

			if resp.Error != "" {
				renderErrorDiagnostic(c, resp)
				return
			}

			renderResponseHeader(c, resp)
			TelemetryWaterfallBar(c, resp.Telemetry)
			renderResponseBody(c, resp.Body)
		})
}

func renderInFlightView(c *ui.Context, method string, elapsedMs int64) {
	if method == "" {
		method = "GET"
	}
	ui.Column(c).
		Grow(1).
		Center().
		Gap(14).
		Children(func() {
			ui.Row(c).
				AlignItems(ui.Center).
				Gap(10).
				Children(func() {
					primitives.KeycapBadge(c, method)
					ui.Text(c, fmt.Sprintf("Request in-flight • %d ms elapsed", elapsedMs)).
						Font(theme.FontMono).
						FontSize(theme.SizeBody).
						TextColor(theme.RimFocus)
				})

			// Stepper indicator (Brilliant Screen 02)
			ui.Row(c).
				AlignItems(ui.Center).
				Gap(8).
				Children(func() {
					renderStepBadge(c, "1. Pre-Script", true)
					ui.Text(c, "→").Font(theme.FontMono).FontSize(11).TextColor(theme.TextDim)
					renderStepBadge(c, "2. OAuth2", true)
					ui.Text(c, "→").Font(theme.FontMono).FontSize(11).TextColor(theme.TextDim)
					renderStepBadge(c, "3. Dispatch", true)
					ui.Text(c, "→").Font(theme.FontMono).FontSize(11).TextColor(theme.TextDim)
					renderStepBadge(c, "4. Tests", false)
				})
		})
}

func renderStepBadge(c *ui.Context, label string, active bool) {
	bg := theme.SurfaceSubPlane
	txt := theme.TextDim
	border := theme.BorderSubtle
	if active {
		bg = theme.SurfaceActive
		txt = theme.RimFocus
		border = theme.RimFocus
	}

	ui.Row(c).
		PaddingX(8).
		PaddingY(3).
		Radius(4).
		Background(bg).
		Border(1, border).
		Children(func() {
			ui.Text(c, label).Font(theme.FontMono).FontSize(10).Bold().TextColor(txt)
		})
}

func renderEmptyState(c *ui.Context) {
	ui.Column(c).
		Grow(1).
		Center().
		Gap(8).
		Children(func() {
			ui.Row(c).
				AlignItems(ui.Center).
				Gap(6).
				Children(func() {
					ui.Text(c, "Hit").Font(theme.FontSans).FontSize(theme.SizeBody).TextColor(theme.TextDim)
					primitives.ShortcutCap(c, "⌘↵")
					ui.Text(c, "to dispatch request and inspect telemetry").Font(theme.FontSans).FontSize(theme.SizeBody).TextColor(theme.TextDim)
				})
		})
}

func renderErrorDiagnostic(c *ui.Context, resp *models.ResponseModel) {
	ui.Column(c).
		Grow(1).
		Padding(16).
		Radius(6).
		Background(theme.CanvasDeep).
		Border(1, theme.RimDanger).
		Shadow(0, 4, 12, 0, theme.RimDangerGlow).
		Gap(10).
		Children(func() {
			ui.Row(c).
				AlignItems(ui.Center).
				Gap(8).
				Children(func() {
					primitives.StatusCodeBadge(c, 502, "ERROR")
					ui.Text(c, "Request Failed").
						Font(theme.FontSans).
						FontSize(theme.SizeBody).
						Bold().
						TextColor(theme.RimDanger)
				})

			ui.Text(c, resp.Error).
				Font(theme.FontMono).
				FontSize(theme.SizeBase).
				TextColor(theme.TextPrimary)

			ui.Text(c, "Ensure upstream service is reachable and local transport policy allows network egress.").
				Font(theme.FontSans).
				FontSize(theme.SizeCaption).
				TextColor(theme.TextDim)
		})
}

func renderResponseHeader(c *ui.Context, resp *models.ResponseModel) {
	ui.Row(c).
		AlignItems(ui.Center).
		Children(func() {
			primitives.StatusCodeBadge(c, resp.StatusCode, resp.StatusText)

			ui.Box(c).Width(12)

			ui.Text(c, fmt.Sprintf("%d ms", resp.Duration.Milliseconds())).
				Font(theme.FontMono).
				FontSize(theme.SizeBase).
				TextColor(theme.TextPrimary)

			ui.Text(c, " • ").TextColor(theme.TextDim)

			ui.Text(c, formatBytes(resp.SizeBytes)).
				Font(theme.FontMono).
				FontSize(theme.SizeBase).
				TextColor(theme.TextSecondary)

			proto := resp.Proto
			if proto == "" {
				proto = "HTTP/2.0"
			}

			ui.Text(c, " • ").TextColor(theme.TextDim)

			ui.Row(c).
				PaddingX(6).
				PaddingY(2).
				Radius(4).
				Background(theme.SurfaceCardTop).
				Border(1, theme.BorderSubtle).
				Children(func() {
					ui.Text(c, proto).
						Font(theme.FontMono).
						FontSize(theme.SizeCaption).
						TextColor(theme.RimFocus)
				})

			ui.Box(c).Grow(1)

			// Copy Body Button
			copyBtn := ui.Row(c).
				PaddingX(8).
				PaddingY(4).
				Radius(4).
				Background(theme.SurfaceCardTop).
				Border(1, theme.BorderSubtle).
				Center().
				Focusable()

			if copyBtn.Clicked() {
				c.Services().WriteClipboard(resp.Body)
			}

			copyBtn.Children(func() {
				ui.Text(c, "Copy").
					Font(theme.FontSans).
					FontSize(theme.SizeCaption).
					TextColor(theme.TextMuted)
			})
		})
}

// TelemetryWaterfallBar renders a segmented progress bar of DNS, TCP, TLS, TTFB, and DL timings.
func TelemetryWaterfallBar(c *ui.Context, t models.Telemetry) ui.Element {
	total := t.DNS + t.TCP + t.TLS + t.TTFB + t.Download
	if total <= 0 {
		total = 1
	}

	dnsPct := float32(t.DNS) / float32(total) * 100
	tcpPct := float32(t.TCP) / float32(total) * 100
	tlsPct := float32(t.TLS) / float32(total) * 100
	ttfbPct := float32(t.TTFB) / float32(total) * 100
	dlPct := float32(t.Download) / float32(total) * 100

	return ui.Column(c).
		Gap(6).
		Children(func() {
			// Segmented Bar
			ui.Row(c).
				Height(8).
				Radius(4).
				Background(theme.CanvasDeep).
				Border(1, theme.BorderSubtle).
				Children(func() {
					if dnsPct > 0 {
						ui.Box(c).WidthPercent(dnsPct).Height(6).Background(theme.TelemetryDNS).Radius(2)
					}
					if tcpPct > 0 {
						ui.Box(c).WidthPercent(tcpPct).Height(6).Background(theme.TelemetryTCP).Radius(2)
					}
					if tlsPct > 0 {
						ui.Box(c).WidthPercent(tlsPct).Height(6).Background(theme.TelemetryTLS).Radius(2)
					}
					if ttfbPct > 0 {
						ui.Box(c).WidthPercent(ttfbPct).Height(6).Background(theme.TelemetryTTFB).Radius(2)
					}
					if dlPct > 0 {
						ui.Box(c).WidthPercent(dlPct).Height(6).Background(theme.TelemetryDL).Radius(2)
					}
				})

			// Legend
			ui.Row(c).
				Gap(16).
				AlignItems(ui.Center).
				Children(func() {
					legendItem(c, "DNS", t.DNS, theme.TelemetryDNS)
					legendItem(c, "TCP", t.TCP, theme.TelemetryTCP)
					legendItem(c, "TLS", t.TLS, theme.TelemetryTLS)
					legendItem(c, "TTFB", t.TTFB, theme.TelemetryTTFB)
					legendItem(c, "DL", t.Download, theme.TelemetryDL)
				})
		})
}

func legendItem(c *ui.Context, name string, ms int64, color ui.Color) {
	ui.Row(c).
		Gap(5).
		AlignItems(ui.Center).
		Children(func() {
			ui.Box(c).Size(6, 6).Radius(2).Background(color)
			ui.Text(c, fmt.Sprintf("%s %dms", name, ms)).
				Font(theme.FontMono).
				FontSize(10).
				TextColor(theme.TextMuted)
		})
}

func renderResponseBody(c *ui.Context, body string) {
	if body == "" {
		body = "{}"
	}

	lines := strings.Split(body, "\n")
	if len(lines) > 200 {
		lines = lines[:200] // prevent huge layout strain in simple viewer
	}

	ui.Scroll(c).
		Grow(1).
		Children(func() {
			ui.Row(c).
				Grow(1).
				Background(theme.CanvasDeep).
				Radius(6).
				Padding(10).
				Border(1, theme.BorderSubtle).
				Children(func() {
					// Line numbers column
					ui.Column(c).
						Width(32).
						Children(func() {
							for i := 1; i <= len(lines); i++ {
								ui.Text(c, fmt.Sprintf("%d", i)).
									Font(theme.FontMono).
									FontSize(theme.SizeCaption).
									TextColor(theme.TextDim)
							}
						})

					ui.Box(c).Width(10)

					// Content lines column
					ui.Column(c).
						Grow(1).
						Children(func() {
							for _, line := range lines {
								col := theme.TextPrimary
								trim := strings.TrimSpace(line)
								if strings.HasPrefix(trim, "\"") && strings.Contains(trim, "\":") {
									col = theme.RimFocus
								} else if strings.HasPrefix(trim, "{") || strings.HasPrefix(trim, "}") || strings.HasPrefix(trim, "[") || strings.HasPrefix(trim, "]") {
									col = theme.TextDim
								}

								ui.Text(c, line).
									Font(theme.FontMono).
									FontSize(theme.SizeCaption).
									TextColor(col)
							}
						})
				})
		})
}

func formatBytes(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%d B", b)
	} else if b < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(b)/1024.0)
	}
	return fmt.Sprintf("%.2f MB", float64(b)/(1024.0*1024.0))
}
