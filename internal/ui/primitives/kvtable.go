package primitives

import (
	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

// StandardHeaderSuggestions provides autocompletion for common HTTP request headers.
var StandardHeaderSuggestions = []string{
	"Accept",
	"Accept-Charset",
	"Accept-Encoding",
	"Accept-Language",
	"Authorization",
	"Cache-Control",
	"Connection",
	"Content-Disposition",
	"Content-Encoding",
	"Content-Length",
	"Content-Type",
	"Cookie",
	"Host",
	"If-Match",
	"If-Modified-Since",
	"If-None-Match",
	"Origin",
	"Pragma",
	"Range",
	"Referer",
	"User-Agent",
	"X-API-Key",
	"X-Correlation-ID",
	"X-Forwarded-For",
	"X-Request-ID",
}

// KVTableProps holds configuration and data for the key-value pair table.
type KVTableProps struct {
	Items       *[]models.KeyValuePair
	OnChange    func()
	KeyPrompt   string
	ValuePrompt string
	DescPrompt  string
	Suggestions []string
	AddLabel    string
}

// KVTable renders an interactive key-value pair table with enabled checkboxes,
// editable keys, values, descriptions, autocompletions, and row deletion.
func KVTable(c *ui.Context, props KVTableProps) ui.Element {
	if props.KeyPrompt == "" {
		props.KeyPrompt = "Key"
	}
	if props.ValuePrompt == "" {
		props.ValuePrompt = "Value"
	}
	if props.DescPrompt == "" {
		props.DescPrompt = "Description"
	}
	if props.AddLabel == "" {
		props.AddLabel = "+ Add Item"
	}

	return ui.Column(c).
		Gap(6).
		Children(func() {
			// Table Header strictly matching Brilliant Frame 327
			ui.Row(c).
				Height(18).
				PaddingX(10).
				Gap(10).
				AlignItems(ui.Center).
				Children(func() {
					ui.Box(c).Width(14) // Checkbox spacer (14x14)
					ui.Text(c, "KEY").
						Font(theme.FontSans).
						FontSize(10).
						Bold().
						TextColor(theme.TextDim).
						Width(180)
					ui.Box(c).Width(1) // Divider spacer (1px)
					ui.Text(c, "VALUE").
						Font(theme.FontSans).
						FontSize(10).
						Bold().
						TextColor(theme.TextDim).
						Width(230)
					ui.Box(c).Width(1) // Divider spacer (1px)
					ui.Text(c, "DESCRIPTION").
						Font(theme.FontSans).
						FontSize(10).
						Bold().
						TextColor(theme.TextDim).
						Grow(1)
					ui.Box(c).Width(20) // Delete button spacer (20px)
				})

			// Rows strictly matching Brilliant Frame 331 & 339
			if props.Items != nil {
				items := *props.Items
				for i := 0; i < len(items); i++ {
					idx := i
					item := &items[idx]

					borderColor := ui.RGBA(255, 255, 255, 0.10)
					shadowColor := ui.RGBA(0, 0, 0, 0.35)
					dividerColor := ui.RGBA(255, 255, 255, 0.08)
					keyColor := ui.RGB(0xF3, 0xF4, 0xF6)
					valColor := theme.RimFocus   // #38BDF8
					descColor := theme.TextMuted // #8B8F9E

					if !item.Enabled {
						borderColor = ui.RGBA(255, 255, 255, 0.05)
						shadowColor = ui.Transparent
						dividerColor = ui.RGBA(255, 255, 255, 0.06)
						keyColor = theme.TextMuted  // #9CA3AF
						valColor = theme.TextMuted  // #9CA3AF
						descColor = theme.TextDim   // #6B7280
					}

					row := ui.Row(c).
						Height(28).
						PaddingX(10).
						Gap(10).
						Radius(6).
						Border(1, borderColor).
						Shadow(0, 2, 6, 0, shadowColor).
						AlignItems(ui.Center)

					if item.Enabled {
						row.LinearGradient(ui.LinearGradient{
							From:  ui.RGB(0x19, 0x1C, 0x26),
							To:    ui.RGB(0x15, 0x18, 0x21),
							Angle: 180,
						})
					} else {
						row.Background(ui.RGB(0x12, 0x14, 0x1C))
					}

					row.Children(func() {
						// Active Checkbox Toggle (14x14, rd 3)
						chk := ui.Row(c).
							Width(14).
							Height(14).
							Radius(3).
							Center().
							Focusable()

						if item.Enabled {
							chk.LinearGradient(ui.LinearGradient{
								From:  ui.RGB(0x3B, 0x82, 0xF6),
								To:    ui.RGB(0x25, 0x63, 0xEB),
								Angle: 180,
							}).Shadow(0, 1, 0, 0, ui.RGB(0x1E, 0x3A, 0x8A))
						} else {
							chk.Border(1, ui.RGB(0x4B, 0x55, 0x63)).
								Background(ui.Transparent)
						}

						if chk.Clicked() {
							item.Enabled = !item.Enabled
							if props.OnChange != nil {
								props.OnChange()
							}
						}

						chk.Children(func() {
							if item.Enabled {
								ui.Text(c, "✓").
									Font(theme.FontSans).
									FontSize(9).
									Bold().
									TextColor(ui.RGB(0xFF, 0xFF, 0xFF))
							}
						})

						// Key Input (Width 180) - borderless inline input
						keyEl := ui.TextInputBase(c, &item.Key).
							Font(theme.FontMono).
							FontSize(theme.SizeToken).
							TextColor(keyColor).
							Placeholder(props.KeyPrompt).
							Width(180)

						// Divider 1 (Width 1, Height 14)
						ui.Box(c).
							Width(1).
							Height(14).
							Background(dividerColor)

						// Value Input (Width 230) - borderless inline input
						valEl := ui.TextInputBase(c, &item.Value).
							Font(theme.FontMono).
							FontSize(theme.SizeToken).
							TextColor(valColor).
							Placeholder(props.ValuePrompt).
							Width(230)

						// Divider 2 (Width 1, Height 14)
						ui.Box(c).
							Width(1).
							Height(14).
							Background(dividerColor)

						// Description Input (Grow 1) - borderless inline input
						descEl := ui.TextInputBase(c, &item.Description).
							Font(theme.FontSans).
							FontSize(theme.SizeToken).
							TextColor(descColor).
							Placeholder(props.DescPrompt).
							Grow(1)

						// Trigger change on typing
						if (keyEl.Changed() || valEl.Changed() || descEl.Changed()) && props.OnChange != nil {
							props.OnChange()
						}

						// Delete row button (Width 20, Height 20)
						delBtn := ui.Row(c).
							Width(20).
							Height(20).
							Radius(4).
							Center().
							Focusable()

						if delBtn.Clicked() {
							newItems := make([]models.KeyValuePair, 0, len(items)-1)
							newItems = append(newItems, items[:idx]...)
							newItems = append(newItems, items[idx+1:]...)
							*props.Items = newItems
							if props.OnChange != nil {
								props.OnChange()
							}
							return
						}

						delBtn.Children(func() {
							ui.Text(c, "✕").
								Font(theme.FontSans).
								FontSize(11).
								TextColor(theme.TextDim)
						})
					})
				}
			}

			// Add Row Button Row
			ui.Row(c).
				PaddingX(2).
				Children(func() {
					addBtn := ui.Row(c).
						Height(24).
						PaddingX(8).
						Radius(4).
						Background(theme.SurfaceSubPlane).
						Border(1, theme.BorderSubtle).
						Center().
						Focusable()

					if addBtn.Clicked() {
						*props.Items = append(*props.Items, models.KeyValuePair{
							Enabled: true,
						})
						if props.OnChange != nil {
							props.OnChange()
						}
					}

					addBtn.Children(func() {
						ui.Text(c, props.AddLabel).
							Font(theme.FontSans).
							FontSize(theme.SizeCaption).
							TextColor(theme.RimFocus)
					})
				})
		})
}
