package modals

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/engine/storage"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/primitives"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

type historyGroup struct {
	Label   string
	Entries []models.HistoryEntry
}

// HistoryModalHost wraps the Execution History dialog in ui.DialogBase with backdrop & Escape dismissal.
func HistoryModalHost(c *ui.Context, state *models.AppState) ui.Element {
	var show bool
	state.View(func(s *models.AppState) {
		show = s.ShowHistoryModal
	})
	if !show {
		return ui.Element{}
	}

	dlg := ui.DialogBase(c, &show, func(backdrop ui.Element, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.72))
		HistoryModal(c, state)
	})

	if dlg.Dismissed() || !show || c.Shortcut(0, ui.KeyEscape) {
		state.CloseHistoryModal()
	}
	return dlg
}

// HistoryModal renders the searchable, date-grouped Execution History timeline with HAR 1.2 export.
func HistoryModal(c *ui.Context, state *models.AppState) ui.Element {
	var (
		entries    []models.HistoryEntry
		totalCount int
		filter     models.HistoryFilter
		statusMsg  string
	)
	state.View(func(s *models.AppState) {
		entries = s.HistoryEntries
		totalCount = s.HistoryTotalCount
		filter = s.HistoryFilter
		statusMsg = s.HistoryStatusMessage
	})

	groups := groupHistoryEntries(entries, time.Now())

	return ui.Column(c).
		Width(860).
		Height(620).
		Radius(12).
		LinearGradient(ui.LinearGradient{
			From:  theme.SurfacePlaneTop,
			To:    theme.SurfacePlaneBottom,
			Angle: 180,
		}).
		Border(1, theme.BorderMedium).
		Shadow(0, 24, 60, 0, ui.RGBA(0, 0, 0, 0.80)).
		Children(func() {
			renderHistoryHeader(c, state, totalCount)
			renderHistoryFilters(c, state, filter)

			ui.Scroll(c).
				Grow(1).
				Padding(14, 18, 14, 18).
				Gap(12).
				Children(func() {
					if len(entries) == 0 {
						ui.Column(c).
							Grow(1).
							Center().
							Gap(6).
							Children(func() {
								ui.Text(c, "No matching execution history").
									Font(theme.FontSans).
									FontSize(theme.SizeHeading).
									TextColor(theme.TextSecondary)
								ui.Text(c, "Send a request or clear active filters to inspect past HTTP calls.").
									Font(theme.FontSans).
									FontSize(theme.SizeBase).
									TextColor(theme.TextDim)
							})
						return
					}

					for _, group := range groups {
						g := group
						ui.Column(c).
							Gap(6).
							Children(func() {
								ui.Text(c, g.Label).
									Font(theme.FontMono).
									FontSize(theme.SizeBadge).
									Bold().
									TextColor(theme.TextDim)

								for _, entry := range g.Entries {
									renderHistoryRow(c, state, entry)
								}
							})
					}
				})

			if statusMsg != "" {
				ui.Row(c).
					Height(30).
					PaddingX(18).
					Background(theme.SurfaceSubPlane).
					BorderWidth(1, 0, 0, 0).
					BorderColor(theme.BorderSubtle).
					Radius(0, 0, 12, 12).
					AlignItems(ui.Center).
					Children(func() {
						ui.Text(c, statusMsg).
							Font(theme.FontMono).
							FontSize(theme.SizeBadge).
							TextColor(ui.RGB(0x34, 0xD3, 0x99))
					})
			}
		})
}

func renderHistoryHeader(c *ui.Context, state *models.AppState, totalCount int) {
	ui.Row(c).
		Height(56).
		PaddingX(20).
		Background(theme.SurfaceSubPlane).
		BorderWidth(0, 0, 1, 0).
		BorderColor(theme.BorderSubtle).
		Radius(12, 12, 0, 0).
		AlignItems(ui.Center).
		Gap(10).
		Children(func() {
			ui.Column(c).
				Gap(2).
				Children(func() {
					ui.Text(c, "Execution History").
						Font(theme.FontSans).
						FontSize(theme.SizeHeading).
						Bold().
						TextColor(theme.TextPrimary)
					ui.Text(c, fmt.Sprintf("%d Executed Calls", totalCount)).
						Font(theme.FontMono).
						FontSize(theme.SizeBadge).
						TextColor(theme.TextMuted)
				})

			ui.Box(c).Grow(1)

			renderModalActionButton(c, "Export HAR", true, func() {
				state.ExportHistoryHAR(storage.BuildHAR)
			})
			renderModalActionButton(c, "Clear", false, func() {
				_ = state.ClearHistoryLog()
			})

			closeBtn := ui.Row(c).
				Height(30).
				PaddingX(10).
				Radius(6).
				Background(theme.SurfaceCardTop).
				Border(1, theme.BorderSubtle).
				AlignItems(ui.Center).
				Gap(6).
				Focusable()
			if closeBtn.Clicked() {
				state.CloseHistoryModal()
			}
			closeBtn.Children(func() {
				ui.Text(c, "Close").
					Font(theme.FontSans).
					FontSize(theme.SizeBase).
					TextColor(theme.TextSecondary)
				primitives.ShortcutCap(c, "Esc")
			})
		})
}

func renderHistoryFilters(c *ui.Context, state *models.AppState, filter models.HistoryFilter) {
	searchVal := filter.Search
	ui.Row(c).
		Padding(10, 18, 10, 18).
		Gap(8).
		Background(theme.CanvasDeep).
		BorderWidth(0, 0, 1, 0).
		BorderColor(theme.BorderSubtle).
		AlignItems(ui.Center).
		Children(func() {
			ui.Row(c).
				Height(30).
				PaddingX(10).
				Radius(6).
				Background(theme.SurfaceSubPlane).
				Border(1, theme.BorderSubtle).
				AlignItems(ui.Center).
				Gap(6).
				Grow(1).
				Children(func() {
					ui.Text(c, "⌕").
						Font(theme.FontMono).
						FontSize(theme.SizeBase).
						TextColor(theme.TextDim)

					in := ui.EditableText(c, &searchVal).
						Placeholder("Search URL, method, or status...").
						PlaceholderColor(theme.TextDim).
						Font(theme.FontSans).
						FontSize(theme.SizeBase).
						TextColor(theme.TextPrimary).
						Grow(1)

					if in.Changed() {
						state.SetHistorySearch(searchVal)
					}
				})

			for _, method := range []string{"GET", "POST", "PUT", "DELETE"} {
				m := method
				renderHistoryFilterChip(c, m, filter.Method == m, func() {
					state.SetHistoryMethodFilter(m)
				})
			}

			ui.Box(c).Width(1).Height(16).Background(theme.BorderSubtle)

			for _, sc := range []string{"2xx", "4xx", "5xx", "err"} {
				cls := sc
				label := cls
				if cls == "err" {
					label = "ERR"
				}
				renderHistoryFilterChip(c, label, filter.StatusClass == cls, func() {
					state.SetHistoryStatusFilter(cls)
				})
			}
		})
}

func renderHistoryRow(c *ui.Context, state *models.AppState, entry models.HistoryEntry) {
	row := ui.Row(c).
		Height(36).
		PaddingX(10).
		Radius(6).
		Background(theme.SurfaceCardTop).
		Border(1, theme.BorderSubtle).
		AlignItems(ui.Center).
		Gap(10).
		Focusable()

	if row.Clicked() {
		state.SelectAndReplayHistory(entry.ID)
	}

	row.Children(func() {
		primitives.KeycapBadge(c, entry.Method)

		if entry.Request.Name != "" && entry.Request.Name != "Untitled Request" {
			ui.Text(c, entry.Request.Name).
				Font(theme.FontSans).
				FontSize(theme.SizeBase).
				Bold().
				TextColor(theme.TextPrimary)
		}
		ui.Text(c, entry.URL).
			Font(theme.FontMono).
			FontSize(theme.SizeBase).
			TextColor(theme.TextMuted).
			Grow(1)

		statusText, statusColor := historyStatusDisplay(entry)
		ui.Row(c).
			Height(20).
			PaddingX(7).
			Radius(4).
			Background(theme.CanvasDeep).
			Border(1, theme.BorderSubtle).
			Center().
			Children(func() {
				ui.Text(c, statusText).
					Font(theme.FontMono).
					FontSize(theme.SizeBadge).
					Bold().
					TextColor(statusColor)
			})

		ui.Text(c, fmt.Sprintf("%d ms", entry.DurationMs)).
			Font(theme.FontMono).
			FontSize(theme.SizeBadge).
			TextColor(theme.TextMuted)

		ui.Text(c, entry.ExecutedAt.Local().Format("15:04:05")).
			Font(theme.FontMono).
			FontSize(theme.SizeBadge).
			TextColor(theme.TextDim)
	})
}

func renderHistoryFilterChip(c *ui.Context, label string, active bool, onClick func()) {
	bg := theme.SurfaceCardTop
	border := theme.BorderSubtle
	fg := theme.TextSecondary
	if active {
		bg = theme.SurfaceActive
		border = theme.RimFocus
		fg = theme.TextPrimary
	}

	chip := ui.Row(c).
		Height(26).
		PaddingX(8).
		Radius(5).
		Background(bg).
		Border(1, border).
		Center().
		Focusable()
	if chip.Clicked() && onClick != nil {
		onClick()
	}
	chip.Children(func() {
		ui.Text(c, label).
			Font(theme.FontMono).
			FontSize(theme.SizeBadge).
			TextColor(fg)
	})
}

func renderModalActionButton(c *ui.Context, label string, primary bool, onClick func()) {
	btn := ui.Row(c).
		Height(30).
		PaddingX(12).
		Radius(6).
		Center().
		Focusable()
	if primary {
		btn.LinearGradient(ui.LinearGradient{
			From:  ui.RGB(0x3B, 0x82, 0xF6),
			To:    ui.RGB(0x25, 0x63, 0xEB),
			Angle: 180,
		}).Border(1, ui.RGBA(255, 255, 255, 0.22))
	} else {
		btn.Background(theme.SurfaceCardTop).
			Border(1, theme.BorderSubtle)
	}
	if btn.Clicked() && onClick != nil {
		onClick()
	}
	btn.Children(func() {
		ui.Text(c, label).
			Font(theme.FontSans).
			FontSize(theme.SizeBase).
			Bold().
			TextColor(theme.TextPrimary)
	})
}

func historyStatusDisplay(entry models.HistoryEntry) (string, ui.Color) {
	if entry.Error != "" && entry.StatusCode == 0 {
		return "ERR", theme.RimDanger
	}
	text := entry.StatusText
	if text == "" {
		text = fmt.Sprintf("%d", entry.StatusCode)
	}
	switch {
	case entry.StatusCode >= 200 && entry.StatusCode < 300:
		return text, ui.RGB(0x34, 0xD3, 0x99)
	case entry.StatusCode >= 400 && entry.StatusCode < 500:
		return text, ui.RGB(0xFB, 0xBF, 0x24)
	case entry.StatusCode >= 500:
		return text, theme.RimDanger
	default:
		return text, theme.TextSecondary
	}
}

func groupHistoryEntries(entries []models.HistoryEntry, now time.Time) []historyGroup {
	localNow := now.Local()
	todayY, todayM, todayD := localNow.Date()
	yesterday := localNow.AddDate(0, 0, -1)
	yY, yM, yD := yesterday.Date()

	bucketIdx := make(map[string]int)
	var groups []historyGroup

	for _, entry := range entries {
		localTime := entry.ExecutedAt.Local()
		eY, eM, eD := localTime.Date()

		var label string
		switch {
		case eY == todayY && eM == todayM && eD == todayD && now.Sub(entry.ExecutedAt) <= 15*time.Minute:
			label = "TODAY • LAST 15 MINS"
		case eY == todayY && eM == todayM && eD == todayD:
			label = "TODAY"
		case eY == yY && eM == yM && eD == yD:
			label = "YESTERDAY"
		default:
			label = strings.ToUpper(localTime.Format("Jan 02, 2006"))
		}

		idx, exists := bucketIdx[label]
		if !exists {
			idx = len(groups)
			bucketIdx[label] = idx
			groups = append(groups, historyGroup{Label: label})
		}
		groups[idx].Entries = append(groups[idx].Entries, entry)
	}
	return groups
}
