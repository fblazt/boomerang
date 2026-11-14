package modals

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/primitives"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

// PaletteModalHost wraps the Global Command Palette (⌘K) in ui.DialogBase.
func PaletteModalHost(c *ui.Context, state *models.AppState) ui.Element {
	var open bool
	state.View(func(s *models.AppState) {
		open = s.CommandPalOpen
	})
	if !open {
		return ui.Element{}
	}

	dlg := ui.DialogBase(c, &open, func(backdrop ui.Element, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.72))
		PaletteModal(c, state)
	})

	if dlg.Dismissed() || !open || c.Shortcut(0, ui.KeyEscape) {
		state.CloseCommandPalette()
	}
	return dlg
}

// PaletteModal renders the keyboard-first fuzzy-searchable global command palette.
func PaletteModal(c *ui.Context, state *models.AppState) ui.Element {
	var (
		query       string
		selectedIdx int
	)
	state.View(func(s *models.AppState) {
		query = s.PaletteQuery
		selectedIdx = s.PaletteSelectedIndex
	})

	items := state.FilteredPaletteItems()
	if len(items) == 0 {
		selectedIdx = 0
	} else if selectedIdx >= len(items) {
		selectedIdx = len(items) - 1
	} else if selectedIdx < 0 {
		selectedIdx = 0
	}

	executeSelected := func() {
		if len(items) > 0 {
			state.ExecutePaletteItem(items[selectedIdx])
		}
	}

	if c.Shortcut(0, ui.KeyDown) {
		state.MovePaletteSelection(1, len(items))
	}
	if c.Shortcut(0, ui.KeyUp) {
		state.MovePaletteSelection(-1, len(items))
	}
	if c.Shortcut(0, ui.KeyEnter) {
		executeSelected()
	}

	queryDraft := query

	return ui.Column(c).
		Width(660).
		Height(480).
		Background(theme.SurfacePlaneTop).
		Radius(12).
		Border(1, theme.BorderStrong).
		Children(func() {
			// Fuzzy Search Input Row
			ui.Row(c).
				Height(50).
				PaddingX(16).
				AlignItems(ui.Center).
				Gap(10).
				Background(theme.SurfacePlaneTop).
				Radius(12, 12, 0, 0).
				BorderWidth(0, 0, 1, 0).
				BorderColor(theme.BorderSubtle).
				Children(func() {
					ui.Text(c, "⌕").
						Font(theme.FontMono).
						FontSize(theme.SizeHeading).
						TextColor(theme.RimFocus)

					in := ui.TextInputBase(c, &queryDraft).
						Placeholder("Search endpoints, environments, and commands...").
						PlaceholderColor(theme.TextDim).
						Font(theme.FontSans).
						FontSize(theme.SizeHeading).
						TextColor(theme.TextPrimary).
						Grow(1)
					if in.Changed() {
						state.SetPaletteQuery(queryDraft)
					}
					if in.Submitted() {
						executeSelected()
					}

					primitives.ShortcutCap(c, "⌘K")
				})

			// Fuzzy Search Results
			ui.Scroll(c).
				Grow(1).
				Padding(10).
				Gap(4).
				Background(theme.CanvasDeep).
				Children(func() {
					if len(items) == 0 {
						ui.Column(c).
							Grow(1).
							Padding(32).
							Center().
							Children(func() {
								ui.Text(c, "No matching endpoints, environments, or commands.").
									Font(theme.FontSans).
									FontSize(theme.SizeBase).
									TextColor(theme.TextMuted)
							})
						return
					}

					for i, item := range items {
						rowItem := item
						isSel := i == selectedIdx
						bg := theme.SurfaceSubPlane
						borderColor := theme.BorderSubtle
						if isSel {
							bg = theme.SurfaceActive
							borderColor = theme.RimFocus
						}

						row := ui.Row(c).
							Height(38).
							PaddingX(10).
							Radius(6).
							Background(bg).
							Border(1, borderColor).
							AlignItems(ui.Center).
							Gap(10).
							Focusable()
						if row.Clicked() {
							state.ExecutePaletteItem(rowItem)
						}

						row.Children(func() {
							badgeColor := paletteBadgeColor(rowItem.Category)
							ui.Row(c).
								Width(54).
								Height(22).
								Radius(4).
								Background(theme.CanvasDeep).
								Border(1, badgeColor).
								Center().
								Children(func() {
									ui.Text(c, rowItem.Badge).
										Font(theme.FontMono).
										FontSize(theme.SizeShortcut).
										Bold().
										TextColor(badgeColor)
								})

							ui.Column(c).
								Grow(1).
								Gap(1).
								Children(func() {
									ui.Text(c, rowItem.Title).
										Font(theme.FontSans).
										FontSize(theme.SizeBase).
										Bold().
										TextColor(theme.TextPrimary)
									if rowItem.Subtitle != "" {
										ui.Text(c, rowItem.Subtitle).
											Font(theme.FontMono).
											FontSize(theme.SizeShortcut).
											TextColor(theme.TextDim)
									}
								})

							ui.Text(c, rowItem.Category).
								Font(theme.FontMono).
								FontSize(theme.SizeShortcut).
								TextColor(theme.TextMuted)

							if rowItem.Shortcut != "" {
								primitives.ShortcutCap(c, rowItem.Shortcut)
							}
						})
					}
				})

			// Keyboard Legend Footer
			ui.Row(c).
				Height(34).
				PaddingX(16).
				AlignItems(ui.Center).
				Gap(12).
				Background(theme.SurfaceSubPlane).
				Radius(0, 0, 12, 12).
				BorderWidth(1, 0, 0, 0).
				BorderColor(theme.BorderSubtle).
				Children(func() {
					ui.Text(c, "↑↓ Navigate").Font(theme.FontMono).FontSize(theme.SizeShortcut).TextColor(theme.TextDim)
					ui.Text(c, "↵ Open / Run").Font(theme.FontMono).FontSize(theme.SizeShortcut).TextColor(theme.TextDim)
					ui.Text(c, "Esc Dismiss").Font(theme.FontMono).FontSize(theme.SizeShortcut).TextColor(theme.TextDim)
					ui.Box(c).Grow(1)
					ui.Text(c, fmt.Sprintf("%d matches", len(items))).
						Font(theme.FontMono).
						FontSize(theme.SizeShortcut).
						TextColor(theme.TextSecondary)
				})
		})
}

func paletteBadgeColor(category string) ui.Color {
	switch category {
	case "Endpoint":
		return theme.RimFocus
	case "Environment":
		return ui.RGB(0x34, 0xD3, 0x99)
	case "Scratchpad":
		return ui.RGB(0xFB, 0xBF, 0x24)
	default:
		return ui.RGB(0xA8, 0x55, 0xF7)
	}
}
