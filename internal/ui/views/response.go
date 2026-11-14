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
	var stage int
	var hasAuth bool

	state.View(func(s *models.AppState) {
		resp = s.ActiveResponse
		isLoading = s.IsLoading
		elapsedMs = s.ElapsedMs
		method = s.ActiveRequest.Method
		stage = s.InFlightStage
		hasAuth = s.ActiveRequest.AuthType != "" && s.ActiveRequest.AuthType != "none"
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
				renderInFlightView(c, method, elapsedMs, stage, hasAuth)
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
			if len(resp.Assertions) > 0 {
				renderAssertionStrip(c, resp.Assertions)
			}
			renderResponseBody(c, state, resp.Body)
		})
}

func renderInFlightView(c *ui.Context, method string, elapsedMs int64, stage int, hasAuth bool) {
	if method == "" {
		method = "GET"
	}
	if stage <= 0 {
		stage = 3
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

			// Stepper indicator driven by live execution stage
			ui.Row(c).
				AlignItems(ui.Center).
				Gap(8).
				Children(func() {
					authLabel := "2. Auth"
					if hasAuth {
						authLabel = "2. OAuth2"
					}
					renderStepBadge(c, "1. Pre-Script", stage >= 1)
					ui.Text(c, "→").Font(theme.FontMono).FontSize(11).TextColor(theme.TextDim)
					renderStepBadge(c, authLabel, stage >= 2 && (hasAuth || stage >= 3))
					ui.Text(c, "→").Font(theme.FontMono).FontSize(11).TextColor(theme.TextDim)
					renderStepBadge(c, "3. Dispatch", stage >= 3)
					ui.Text(c, "→").Font(theme.FontMono).FontSize(11).TextColor(theme.TextDim)
					renderStepBadge(c, "4. Tests", stage >= 4)
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
	title, hint := classifyRequestError(resp.Error)

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
					if resp.StatusCode > 0 {
						primitives.StatusCodeBadge(c, resp.StatusCode, resp.StatusText)
					} else {
						renderTransportErrorBadge(c, "TRANSPORT ERR")
					}
					ui.Text(c, title).
						Font(theme.FontSans).
						FontSize(theme.SizeBody).
						Bold().
						TextColor(theme.RimDanger)
				})

			ui.Text(c, resp.Error).
				Font(theme.FontMono).
				FontSize(theme.SizeBase).
				TextColor(theme.TextPrimary)

			if resp.Telemetry.Total > 0 {
				TelemetryWaterfallBar(c, resp.Telemetry)
			}

			ui.Text(c, hint).
				Font(theme.FontSans).
				FontSize(theme.SizeCaption).
				TextColor(theme.TextDim)
		})
}

func renderTransportErrorBadge(c *ui.Context, label string) {
	s := primitives.GetStatusKeycapStyle(500)
	ui.Row(c).
		Height(20).
		PaddingX(7).
		Radius(5).
		LinearGradient(ui.LinearGradient{From: s.BgFrom, To: s.BgTo, Angle: 180}).
		Border(1, s.BorderColor).
		Shadow(0, 2, 0, 0, s.ShadowColor).
		Center().
		Children(func() {
			ui.Text(c, label).
				Font(theme.FontMono).
				FontSize(theme.SizeBadge).
				Bold().
				TextColor(s.TextColor)
		})
}

func classifyRequestError(errMsg string) (string, string) {
	lower := strings.ToLower(errMsg)
	switch {
	case strings.Contains(lower, "x509") || strings.Contains(lower, "tls") || strings.Contains(lower, "certificate"):
		return "Request Failed", "TLS/mTLS verification failed. Check Pinned Root CA, client identity, or Strict SSL in Vault Transport Policy."
	case strings.Contains(lower, "deadline exceeded") || strings.Contains(lower, "timeout"):
		return "Request Failed", "Upstream timed out before completing the response. Check TTFB bottleneck or upstream service health."
	case strings.Contains(lower, "context canceled"):
		return "Request Failed", "Request dispatch was canceled before completion."
	default:
		return "Request Failed", "Ensure upstream service is reachable and local transport policy allows network egress."
	}
}

func renderResponseHeader(c *ui.Context, resp *models.ResponseModel) {
	ui.Row(c).
		AlignItems(ui.Center).
		Children(func() {
			primitives.StatusCodeBadge(c, resp.StatusCode, resp.StatusText)

			ui.Box(c).Width(12)

			ui.Text(c, formatDurationBadge(resp)).
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

			renderHeaderPill(c, proto, theme.RimFocus)

			if resp.Telemetry.TLSVersion != "" {
				ui.Box(c).Width(6)
				renderHeaderPill(c, resp.Telemetry.TLSVersion, ui.RGB(0x34, 0xD3, 0x99))
			}

			if resp.Telemetry.RemoteAddr != "" {
				ui.Box(c).Width(6)
				renderHeaderPill(c, resp.Telemetry.RemoteAddr, theme.TextMuted)
			}

			if resp.Telemetry.ConnReused {
				ui.Box(c).Width(6)
				renderHeaderPill(c, "REUSED", theme.TextMuted)
			}

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

func renderHeaderPill(c *ui.Context, text string, color ui.Color) {
	ui.Row(c).
		PaddingX(6).
		PaddingY(2).
		Radius(4).
		Background(theme.SurfaceCardTop).
		Border(1, theme.BorderSubtle).
		Children(func() {
			ui.Text(c, text).
				Font(theme.FontMono).
				FontSize(theme.SizeCaption).
				TextColor(color)
		})
}

func formatDurationBadge(resp *models.ResponseModel) string {
	ms := resp.Duration.Milliseconds()
	if ms == 0 && resp.Telemetry.TotalMicros > 0 && resp.Telemetry.TotalMicros < 1000 {
		return fmt.Sprintf("%.2f ms", float64(resp.Telemetry.TotalMicros)/1000.0)
	}
	return fmt.Sprintf("%d ms", ms)
}

func renderAssertionStrip(c *ui.Context, assertions []models.TestAssertionResult) {
	passed := 0
	for _, a := range assertions {
		if a.Passed {
			passed++
		}
	}
	ui.Row(c).
		Padding(6, 8, 6, 8).
		Radius(6).
		Background(theme.SurfaceCardTop).
		Border(1, theme.BorderSubtle).
		AlignItems(ui.Center).
		Gap(8).
		Wrap().
		Children(func() {
			summaryColor := ui.RGB(0x34, 0xD3, 0x99)
			if passed < len(assertions) {
				summaryColor = theme.RimDanger
			}
			ui.Text(c, fmt.Sprintf("TESTS %d/%d", passed, len(assertions))).
				Font(theme.FontMono).
				FontSize(theme.SizeShortcut).
				Bold().
				TextColor(summaryColor)

			for _, a := range assertions {
				badgeColor := ui.RGB(0x34, 0xD3, 0x99)
				prefix := "PASS "
				if !a.Passed {
					badgeColor = theme.RimDanger
					prefix = "FAIL "
				}
				renderHeaderPill(c, prefix+a.Name, badgeColor)
			}
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

func renderResponseBody(c *ui.Context, state *models.AppState, body string) {
	renderResponseBodyEditor(c, state, body)
}

func formatBytes(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%d B", b)
	} else if b < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(b)/1024.0)
	}
	return fmt.Sprintf("%.2f MB", float64(b)/(1024.0*1024.0))
}
