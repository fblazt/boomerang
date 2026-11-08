package primitives

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

// KeycapStyle holds visual styling tokens for a 3D keycap badge.
type KeycapStyle struct {
	TextColor   ui.Color
	BgFrom      ui.Color
	BgTo        ui.Color
	BorderColor ui.Color
	ShadowColor ui.Color
}

// VerbKeycaps maps HTTP verbs and realtime protocols to their keycap styling tokens.
var VerbKeycaps = map[string]KeycapStyle{
	"GET": {
		TextColor:   ui.RGB(0x34, 0xD3, 0x99),
		BgFrom:      ui.RGB(0x11, 0x2A, 0x22),
		BgTo:        ui.RGB(0x0A, 0x1B, 0x16),
		BorderColor: ui.RGBA(0x34, 0xD3, 0x99, 0.35),
		ShadowColor: ui.RGB(0x05, 0x0E, 0x0B),
	},
	"POST": {
		TextColor:   ui.RGB(0xFB, 0xBF, 0x24),
		BgFrom:      ui.RGB(0x2E, 0x21, 0x11),
		BgTo:        ui.RGB(0x1C, 0x14, 0x0A),
		BorderColor: ui.RGBA(0xFB, 0xBF, 0x24, 0.35),
		ShadowColor: ui.RGB(0x0F, 0x0A, 0x05),
	},
	"PUT": {
		TextColor:   ui.RGB(0x60, 0xA5, 0xFA),
		BgFrom:      ui.RGB(0x15, 0x25, 0x42),
		BgTo:        ui.RGB(0x0D, 0x17, 0x2A),
		BorderColor: ui.RGBA(0x60, 0xA5, 0xFA, 0.30),
		ShadowColor: ui.RGB(0x07, 0x0C, 0x16),
	},
	"PATCH": {
		TextColor:   ui.RGB(0xC0, 0x84, 0xFC),
		BgFrom:      ui.RGB(0x24, 0x18, 0x38),
		BgTo:        ui.RGB(0x16, 0x0F, 0x24),
		BorderColor: ui.RGBA(0xC0, 0x84, 0xFC, 0.30),
		ShadowColor: ui.RGB(0x0B, 0x07, 0x14),
	},
	"DEL": {
		TextColor:   ui.RGB(0xF8, 0x71, 0x71),
		BgFrom:      ui.RGB(0x33, 0x16, 0x1B),
		BgTo:        ui.RGB(0x1F, 0x0D, 0x10),
		BorderColor: ui.RGBA(0xF8, 0x71, 0x71, 0.35),
		ShadowColor: ui.RGB(0x11, 0x06, 0x08),
	},
	"DELETE": {
		TextColor:   ui.RGB(0xF8, 0x71, 0x71),
		BgFrom:      ui.RGB(0x33, 0x16, 0x1B),
		BgTo:        ui.RGB(0x1F, 0x0D, 0x10),
		BorderColor: ui.RGBA(0xF8, 0x71, 0x71, 0.35),
		ShadowColor: ui.RGB(0x11, 0x06, 0x08),
	},
	"GQL": {
		TextColor:   ui.RGB(0xF4, 0x72, 0xB6),
		BgFrom:      ui.RGB(0x2D, 0x14, 0x24),
		BgTo:        ui.RGB(0x1B, 0x0B, 0x15),
		BorderColor: ui.RGBA(0xF4, 0x72, 0xB6, 0.35),
		ShadowColor: ui.RGB(0x11, 0x05, 0x0D),
	},
	"WSS": {
		TextColor:   ui.RGB(0x38, 0xBD, 0xF8),
		BgFrom:      ui.RGB(0x10, 0x24, 0x34),
		BgTo:        ui.RGB(0x0B, 0x16, 0x20),
		BorderColor: ui.RGBA(0x38, 0xBD, 0xF8, 0.35),
		ShadowColor: ui.RGB(0x04, 0x0C, 0x12),
	},
	"SSE": {
		TextColor:   ui.RGB(0x2D, 0xD4, 0xBF), // Teal
		BgFrom:      ui.RGB(0x0F, 0x2D, 0x2A),
		BgTo:        ui.RGB(0x0A, 0x1E, 0x1C),
		BorderColor: ui.RGBA(0x2D, 0xD4, 0xBF, 0.35),
		ShadowColor: ui.RGB(0x05, 0x10, 0x0E),
	},
}

// StatusKeycaps maps HTTP status code ranges to 3D badge styles.
var StatusKeycaps = map[int]KeycapStyle{
	200: { // 200 OK / 2xx
		TextColor:   ui.RGB(0x34, 0xD3, 0x99),
		BgFrom:      ui.RGB(0x11, 0x2A, 0x22),
		BgTo:        ui.RGB(0x0A, 0x1B, 0x16),
		BorderColor: ui.RGBA(0x34, 0xD3, 0x99, 0.40),
		ShadowColor: ui.RGB(0x05, 0x0E, 0x0B),
	},
	201: { // 201 CREATED
		TextColor:   ui.RGB(0x34, 0xD3, 0x99),
		BgFrom:      ui.RGB(0x11, 0x2A, 0x22),
		BgTo:        ui.RGB(0x0A, 0x1B, 0x16),
		BorderColor: ui.RGBA(0x34, 0xD3, 0x99, 0.40),
		ShadowColor: ui.RGB(0x05, 0x0E, 0x0B),
	},
	429: { // 429 RATE LIMIT / 4xx warning
		TextColor:   ui.RGB(0xFB, 0xBF, 0x24),
		BgFrom:      ui.RGB(0x2E, 0x21, 0x11),
		BgTo:        ui.RGB(0x1C, 0x14, 0x0A),
		BorderColor: ui.RGBA(0xFB, 0xBF, 0x24, 0.40),
		ShadowColor: ui.RGB(0x0F, 0x0A, 0x05),
	},
	502: { // 502 BAD GATEWAY / 5xx error
		TextColor:   ui.RGB(0xF8, 0x71, 0x71),
		BgFrom:      ui.RGB(0x33, 0x16, 0x1B),
		BgTo:        ui.RGB(0x1F, 0x0D, 0x10),
		BorderColor: ui.RGBA(0xF8, 0x71, 0x71, 0.45),
		ShadowColor: ui.RGB(0x11, 0x06, 0x08),
	},
}

// GetVerbKeycapStyle returns KeycapStyle for a given verb or protocol.
func GetVerbKeycapStyle(verb string) KeycapStyle {
	if s, ok := VerbKeycaps[verb]; ok {
		return s
	}
	return VerbKeycaps["GET"]
}

// GetStatusKeycapStyle returns KeycapStyle for a given HTTP status code.
func GetStatusKeycapStyle(code int) KeycapStyle {
	if s, ok := StatusKeycaps[code]; ok {
		return s
	}
	switch {
	case code >= 200 && code < 300:
		return StatusKeycaps[200]
	case code >= 300 && code < 400:
		return VerbKeycaps["PUT"] // Blue 3xx redirects
	case code >= 400 && code < 500:
		return StatusKeycaps[429]
	default:
		return StatusKeycaps[502]
	}
}

// KeycapBadge renders a 3D tactile badge for HTTP verbs and protocols.
func KeycapBadge(c *ui.Context, verb string) ui.Element {
	s := GetVerbKeycapStyle(verb)
	return ui.Row(c).
		Height(20).
		PaddingX(7).
		Radius(5).
		LinearGradient(ui.LinearGradient{From: s.BgFrom, To: s.BgTo, Angle: 180}).
		Border(1, s.BorderColor).
		Shadow(0, 2, 0, 0, s.ShadowColor).
		Center().
		Children(func() {
			ui.Text(c, verb).
				Font(theme.FontMono).
				FontSize(theme.SizeBadge).
				Bold().
				TextColor(s.TextColor)
		})
}

// StatusCodeBadge renders a 3D tactile badge for HTTP status codes (e.g. "200 OK", "502 ERR").
func StatusCodeBadge(c *ui.Context, code int, text string) ui.Element {
	s := GetStatusKeycapStyle(code)
	label := fmt.Sprintf("%d %s", code, text)
	if text == "" {
		label = fmt.Sprintf("%d", code)
	}

	return ui.Row(c).
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
