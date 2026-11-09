package theme

import "github.com/egoist/mygo/ui"

// Canvas & Surface Colors
var (
	// CanvasBase is the base window background color (#090A0F)
	CanvasBase = ui.RGB(0x09, 0x0A, 0x0F)
	// CanvasDeep is the inner workspace background color (#0C0D12)
	CanvasDeep = ui.RGB(0x0C, 0x0D, 0x12)

	// Elevated Surface Planes
	SurfacePlaneTop    = ui.RGB(0x1E, 0x22, 0x2E)
	SurfacePlaneBottom = ui.RGB(0x17, 0x1A, 0x24)
	SurfaceSubPlane    = ui.RGB(0x14, 0x17, 0x20)
	SurfaceCardTop     = ui.RGB(0x19, 0x1C, 0x26)
	SurfaceCardBottom  = ui.RGB(0x15, 0x18, 0x21)
	SurfaceActive      = ui.RGB(0x22, 0x27, 0x36)
	SurfaceHover       = ui.RGB(0x27, 0x2C, 0x3D)

	// Specular and Subtle Borders
	BorderSubtle = ui.RGBA(255, 255, 255, 0.08)
	BorderMedium = ui.RGBA(255, 255, 255, 0.14)
	BorderStrong = ui.RGBA(255, 255, 255, 0.22)

	// Focus Rims (Electric Cyan)
	RimFocus     = ui.RGB(0x38, 0xBD, 0xF8)
	RimFocusGlow = ui.RGBA(0x38, 0xBD, 0xF8, 0.22)

	// Danger Rims (Rose / Red)
	RimDanger     = ui.RGB(0xF8, 0x71, 0x71)
	RimDangerGlow = ui.RGBA(0xEF, 0x44, 0x44, 0.20)

	// Text Hierarchy
	TextPrimary   = ui.RGB(0xFF, 0xFF, 0xFF)
	TextSecondary = ui.RGB(0xD1, 0xD5, 0xDB)
	TextMuted     = ui.RGB(0x9C, 0xA3, 0xAF)
	TextDim       = ui.RGB(0x6B, 0x72, 0x80)
)

// Telemetry Waterfall Colors
var (
	TelemetryDNS  = ui.RGB(0xA8, 0x55, 0xF7) // Purple (DNS)
	TelemetryTCP  = ui.RGB(0xF5, 0x9E, 0x0B) // Amber (TCP)
	TelemetryTLS  = ui.RGB(0x38, 0xBD, 0xF8) // Cyan (TLS)
	TelemetryTTFB = ui.RGB(0x10, 0xB9, 0x81) // Emerald (TTFB)
	TelemetryDL   = ui.RGB(0x3B, 0x82, 0xF6) // Blue (Download)
)

// EnvironmentColor returns the standard theme color for an environment name.
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
		return RimFocus
	}
}

// HTTP Verb & Protocol Keycap Styles
type VerbStyle struct {
	TextColor   ui.Color
	BgFrom      ui.Color
	BgTo        ui.Color
	BorderColor ui.Color
	ShadowColor ui.Color
}

var VerbStyles = map[string]VerbStyle{
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
}

// GetVerbStyle returns the VerbStyle for a given HTTP verb/protocol with safe fallback.
func GetVerbStyle(verb string) VerbStyle {
	if s, ok := VerbStyles[verb]; ok {
		return s
	}
	return VerbStyles["GET"]
}
