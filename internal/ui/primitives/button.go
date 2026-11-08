package primitives

import (
	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

// PrimaryButton renders an elevated action button with linear gradient and physical shadow.
func PrimaryButton(c *ui.Context, label string, onClick func()) ui.Element {
	btn := ui.Row(c).
		Height(36).
		PaddingX(14).
		Radius(6).
		LinearGradient(ui.LinearGradient{
			From:  ui.RGB(0x3B, 0x82, 0xF6), // #3B82F6 -> #2563EB (Brilliant Frame 310)
			To:    ui.RGB(0x25, 0x63, 0xEB),
			Angle: 180,
		}).
		Border(1, ui.RGBA(255, 255, 255, 0.22)).
		Shadow(0, 4, 10, 0, ui.RGBA(0, 0, 0, 0.50)).
		Center().
		Focusable()

	if btn.Clicked() && onClick != nil {
		onClick()
	}

	btn.Children(func() {
		ui.Text(c, label).
			Font(theme.FontSans).
			FontSize(theme.SizeBody).
			Bold().
			TextColor(theme.TextPrimary)
	})

	return btn
}

// SplitButton renders a split action button with primary trigger and a dropdown toggle.
func SplitButton(c *ui.Context, label string, onMainClick func(), onMenuClick func()) ui.Element {
	container := ui.Row(c).
		Height(32).
		Radius(6).
		Border(1, ui.RGBA(255, 255, 255, 0.20)).
		Shadow(0, 2, 0, 0, ui.RGB(0x0F, 0x2A, 0x75))

	container.Children(func() {
		// Main action part
		mainPart := ui.Row(c).
			Height(32).
			PaddingX(12).
			LinearGradient(ui.LinearGradient{
				From:  ui.RGB(0x25, 0x63, 0xEB),
				To:    ui.RGB(0x1D, 0x4E, 0xD8),
				Angle: 180,
			}).
			Center().
			Focusable()

		if mainPart.Clicked() && onMainClick != nil {
			onMainClick()
		}

		mainPart.Children(func() {
			ui.Text(c, label).
				Font(theme.FontSans).
				FontSize(theme.SizeBody).
				Bold().
				TextColor(theme.TextPrimary)
		})

		// Divider
		ui.Box(c).
			Width(1).
			Height(32).
			Background(ui.RGBA(255, 255, 255, 0.25))

		// Menu toggle part
		menuPart := ui.Row(c).
			Height(32).
			PaddingX(8).
			LinearGradient(ui.LinearGradient{
				From:  ui.RGB(0x25, 0x63, 0xEB),
				To:    ui.RGB(0x1D, 0x4E, 0xD8),
				Angle: 180,
			}).
			Center().
			Focusable()

		if menuPart.Clicked() && onMenuClick != nil {
			onMenuClick()
		}

		menuPart.Children(func() {
			ui.Text(c, "▾").
				Font(theme.FontSans).
				FontSize(theme.SizeBody).
				Bold().
				TextColor(theme.TextPrimary)
		})
	})

	return container
}

// SecondaryButton renders an elevated neutral button with linear gradient and subtle border.
func SecondaryButton(c *ui.Context, label string, onClick func()) ui.Element {
	btn := ui.Row(c).
		Height(36).
		PaddingX(14).
		Radius(6).
		LinearGradient(ui.LinearGradient{
			From:  theme.SurfaceHover,
			To:    theme.SurfaceActive,
			Angle: 180,
		}).
		Border(1, theme.BorderMedium).
		Shadow(0, 4, 10, 0, ui.RGBA(0, 0, 0, 0.40)).
		Center().
		Focusable()

	if btn.Clicked() && onClick != nil {
		onClick()
	}

	btn.Children(func() {
		ui.Text(c, label).
			Font(theme.FontSans).
			FontSize(theme.SizeBody).
			Bold().
			TextColor(theme.TextPrimary)
	})

	return btn
}

// SendButton composite button with embedded ⌘↵ shortcut cap (Brilliant Frame 310/311).
func SendButton(c *ui.Context, onClick func()) ui.Element {
	btn := ui.Row(c).
		Height(36).
		PaddingX(14).
		Gap(6).
		Radius(6).
		LinearGradient(ui.LinearGradient{
			From:  ui.RGB(0x3B, 0x82, 0xF6),
			To:    ui.RGB(0x25, 0x63, 0xEB),
			Angle: 180,
		}).
		Border(1, ui.RGBA(255, 255, 255, 0.22)).
		Shadow(0, 4, 10, 0, ui.RGBA(0, 0, 0, 0.50)).
		AlignItems(ui.Center).
		Focusable()

	if btn.Clicked() && onClick != nil {
		onClick()
	}

	btn.Children(func() {
		ui.Text(c, "Send").
			Font(theme.FontSans).
			FontSize(theme.SizeBody).
			Bold().
			TextColor(theme.TextPrimary)

		// ⌘↵ cap embedded inside button
		ui.Row(c).
			Height(18).
			PaddingX(5).
			Radius(4).
			Background(ui.RGBA(0x0F, 0x17, 0x2A, 0.35)).
			Border(1, ui.RGBA(255, 255, 255, 0.20)).
			Center().
			Children(func() {
				ui.Text(c, "⌘↵").
					Font(theme.FontMono).
					FontSize(theme.SizeShortcut).
					Bold().
					TextColor(ui.RGB(0xEF, 0xF6, 0xFF))
			})
	})

	return btn
}

// CancelButton renders an in-flight cancel button with embedded Esc shortcut cap.
func CancelButton(c *ui.Context, onClick func()) ui.Element {
	btn := ui.Row(c).
		Height(36).
		PaddingX(14).
		Gap(6).
		Radius(6).
		LinearGradient(ui.LinearGradient{
			From:  ui.RGB(0x99, 0x1B, 0x1B),
			To:    ui.RGB(0x7F, 0x1D, 0x1D),
			Angle: 180,
		}).
		Border(1, ui.RGBA(248, 113, 113, 0.40)).
		Shadow(0, 4, 10, 0, ui.RGBA(0, 0, 0, 0.50)).
		AlignItems(ui.Center).
		Focusable()

	if btn.Clicked() && onClick != nil {
		onClick()
	}

	btn.Children(func() {
		ui.Text(c, "Cancel").
			Font(theme.FontSans).
			FontSize(theme.SizeBody).
			Bold().
			TextColor(theme.TextPrimary)

		// Esc cap embedded inside button
		ui.Row(c).
			Height(18).
			PaddingX(5).
			Radius(4).
			Background(ui.RGBA(0x0F, 0x17, 0x2A, 0.35)).
			Border(1, ui.RGBA(255, 255, 255, 0.20)).
			Center().
			Children(func() {
				ui.Text(c, "Esc").
					Font(theme.FontMono).
					FontSize(theme.SizeShortcut).
					Bold().
					TextColor(ui.RGB(0xFE, 0xCA, 0xCA))
			})
	})

	return btn
}

// SaveButton composite button with embedded ⌘S shortcut cap (Brilliant Frame 312/313).
func SaveButton(c *ui.Context, onClick func()) ui.Element {
	btn := ui.Row(c).
		Height(36).
		PaddingX(14).
		Gap(6).
		Radius(6).
		LinearGradient(ui.LinearGradient{
			From:  theme.SurfaceHover,
			To:    theme.SurfaceActive,
			Angle: 180,
		}).
		Border(1, theme.BorderMedium).
		Shadow(0, 4, 10, 0, ui.RGBA(0, 0, 0, 0.40)).
		AlignItems(ui.Center).
		Focusable()

	if btn.Clicked() && onClick != nil {
		onClick()
	}

	btn.Children(func() {
		ui.Text(c, "Save").
			Font(theme.FontSans).
			FontSize(theme.SizeBody).
			Bold().
			TextColor(theme.TextPrimary)

		// ⌘S cap embedded inside button
		ui.Row(c).
			Height(18).
			PaddingX(5).
			Radius(4).
			LinearGradient(ui.LinearGradient{
				From:  ui.RGB(0x25, 0x29, 0x38),
				To:    ui.RGB(0x19, 0x1C, 0x27),
				Angle: 180,
			}).
			Border(1, ui.RGBA(255, 255, 255, 0.15)).
			Center().
			Children(func() {
				ui.Text(c, "⌘S").
					Font(theme.FontMono).
					FontSize(theme.SizeShortcut).
					Bold().
					TextColor(ui.RGB(0xD1, 0xD5, 0xDB))
			})
	})

	return btn
}

// IconButton renders a compact tactile icon / glyph button.
func IconButton(c *ui.Context, icon string, tooltip string, onClick func()) ui.Element {
	btn := ui.Row(c).
		Width(28).
		Height(28).
		Radius(5).
		LinearGradient(ui.LinearGradient{
			From:  theme.SurfaceCardTop,
			To:    theme.SurfaceCardBottom,
			Angle: 180,
		}).
		Border(1, theme.BorderSubtle).
		Shadow(0, 1, 0, 0, ui.RGB(0x08, 0x09, 0x0D)).
		Center().
		Focusable()

	if tooltip != "" {
		btn.Tooltip(tooltip)
	}

	if btn.Clicked() && onClick != nil {
		onClick()
	}

	btn.Children(func() {
		ui.Text(c, icon).
			Font(theme.FontSans).
			FontSize(theme.SizeBase).
			TextColor(theme.TextSecondary)
	})

	return btn
}
