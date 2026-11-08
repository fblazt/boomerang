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
			// Table Header
			ui.Row(c).
				PaddingX(8).
				AlignItems(ui.Center).
				Children(func() {
					ui.Box(c).Width(24) // Checkbox width
					ui.Text(c, "KEY").Font(theme.FontSans).FontSize(theme.SizeCaption).Bold().TextColor(theme.TextDim).Width(160)
					ui.Text(c, "VALUE").Font(theme.FontSans).FontSize(theme.SizeCaption).Bold().TextColor(theme.TextDim).Width(220)
					ui.Text(c, "DESCRIPTION").Font(theme.FontSans).FontSize(theme.SizeCaption).Bold().TextColor(theme.TextDim).Grow(1)
					ui.Box(c).Width(28) // Action width
				})

			// Rows
			if props.Items != nil {
				items := *props.Items
				for i := 0; i < len(items); i++ {
					idx := i
					item := &items[idx]

					bg := theme.SurfaceCardTop
					textColor := theme.TextPrimary
					if !item.Enabled {
						bg = theme.SurfaceSubPlane
						textColor = theme.TextDim
					}

					ui.Row(c).
						Height(32).
						PaddingX(8).
						Radius(6).
						Background(bg).
						Border(1, theme.BorderSubtle).
						AlignItems(ui.Center).
						Children(func() {
							// Active Checkbox Toggle
							chk := ui.Row(c).
								Width(18).
								Height(18).
								Radius(3).
								Background(theme.SurfaceSubPlane).
								Border(1, theme.BorderSubtle).
								Center().
								Focusable()

							if chk.Clicked() {
								item.Enabled = !item.Enabled
								if props.OnChange != nil {
									props.OnChange()
								}
							}

							chk.Children(func() {
								if item.Enabled {
									ui.Text(c, "✓").
										Font(theme.FontMono).
										FontSize(11).
										Bold().
										TextColor(theme.RimFocus)
								}
							})

							ui.Box(c).Width(8)

							// Key Input / Autocomplete
							if len(props.Suggestions) > 0 {
								ui.Autocomplete(c, &item.Key, props.Suggestions).
									Font(theme.FontMono).
									FontSize(theme.SizeBase).
									TextColor(textColor).
									Placeholder(props.KeyPrompt).
									Width(160)
							} else {
								ui.TextInput(c, &item.Key).
									Font(theme.FontMono).
									FontSize(theme.SizeBase).
									TextColor(textColor).
									Placeholder(props.KeyPrompt).
									Width(160)
							}

							ui.Box(c).Width(8)

							// Value Input
							ui.TextInput(c, &item.Value).
								Font(theme.FontMono).
								FontSize(theme.SizeBase).
								TextColor(theme.RimFocus).
								Placeholder(props.ValuePrompt).
								Width(220)

							ui.Box(c).Width(8)

							// Description Input
							ui.TextInput(c, &item.Description).
								Font(theme.FontSans).
								FontSize(theme.SizeBase).
								TextColor(theme.TextMuted).
								Placeholder(props.DescPrompt).
								Grow(1)

							ui.Box(c).Width(6)

							// Delete row button
							delBtn := ui.Row(c).
								Width(20).
								Height(20).
								Radius(4).
								Center().
								Focusable()

							if delBtn.Clicked() {
								*props.Items = append(items[:idx], items[idx+1:]...)
								if props.OnChange != nil {
									props.OnChange()
								}
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
