package views

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/primitives"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

// SidebarView renders the left navigation sidebar with collections tree, nested folders,
// request actions, realtime streams, and bottom quick-launch items.
func SidebarView(c *ui.Context, state *models.AppState) ui.Element {
	var (
		cols          []models.Collection
		drafts        []models.ScratchpadDraft
		activeColID   string
		activeItemID  string
		activeDraftID string
		selectedDir   string
		filter        string
		renamingID    string
		renameDraft   string
		collapsed     map[string]bool
	)

	state.View(func(s *models.AppState) {
		cols = s.Collections
		drafts = s.Drafts
		activeColID = s.ActiveCollectionID
		activeItemID = s.ActiveItemID
		activeDraftID = s.ActiveDraftID
		selectedDir = s.SelectedFolderID
		filter = s.CollectionFilter
		renamingID = s.RenamingNodeID
		renameDraft = s.RenameDraft
		collapsed = s.CollapsedNodes
	})

	if activeColID == "" && len(cols) > 0 {
		activeColID = cols[0].ID
	}

	return ui.Column(c).
		Width(264).
		Background(theme.SurfaceSubPlane).
		BorderWidth(0, 1, 0, 0).
		BorderColor(theme.BorderSubtle).
		Padding(12).
		Gap(10).
		Children(func() {
			// Section 0: Scratchpad Uncommitted Drafts
			renderScratchpadSection(c, state, drafts, activeDraftID)

			// Section 1: Collections Header + Quick Creation Actions
			ui.Row(c).
				AlignItems(ui.Center).
				Gap(4).
				Children(func() {
					ui.Text(c, "COLLECTIONS").
						Font(theme.FontSans).
						FontSize(theme.SizeBadge).
						TextColor(theme.TextDim).
						Bold()

					ui.Box(c).Grow(1)

					renderTinyHeaderAction(c, "+Imp", "Import Collection (⌘I)", func() {
						state.OpenImportModal()
					})
					renderTinyHeaderAction(c, "+Col", "New Collection", func() {
						_, _ = state.CreateCollection(fmt.Sprintf("Collection %d", len(cols)+1))
					})
					renderTinyHeaderAction(c, "+Dir", "New Folder", func() {
						_, _ = state.CreateFolder(activeColID, selectedDir, "New Folder")
					})
					primitives.IconButton(c, "+", "New Request", func() {
						_, _ = state.CreateRequestInCollection(activeColID, selectedDir, "New Request")
					})
				})

			// Collections Search Filter Bar
			filterVal := filter
			ui.Row(c).
				Height(28).
				PaddingX(8).
				Radius(5).
				Background(theme.CanvasDeep).
				Border(1, theme.BorderSubtle).
				AlignItems(ui.Center).
				Gap(6).
				Children(func() {
					ui.Text(c, "⌕").
						Font(theme.FontMono).
						FontSize(theme.SizeBase).
						TextColor(theme.TextDim)

					filterInput := ui.EditableText(c, &filterVal).
						Placeholder("Filter collections...").
						PlaceholderColor(theme.TextDim).
						Font(theme.FontSans).
						FontSize(theme.SizeBase).
						TextColor(theme.TextPrimary).
						Tooltip("Filter collections").
						Grow(1)

					if filterInput.Changed() {
						state.SetCollectionFilter(filterVal)
					}
				})

			// Scrollable Nested Collections Tree
			ui.Scroll(c).
				Grow(1).
				Gap(6).
				Children(func() {
					if len(cols) == 0 {
						ui.Text(c, "No collections yet").
							Font(theme.FontSans).
							FontSize(theme.SizeBase).
							TextColor(theme.TextDim)
						return
					}
					for _, col := range cols {
						renderCollectionBranch(c, state, col, activeColID, activeItemID, selectedDir, filter, renamingID, renameDraft, collapsed)
					}
				})

			// Section 2: Realtime Streams
			ui.Row(c).
				AlignItems(ui.Center).
				Children(func() {
					ui.Text(c, "REALTIME STREAMS").
						Font(theme.FontSans).
						FontSize(theme.SizeBadge).
						TextColor(theme.TextDim).
						Bold()
				})

			ui.Column(c).
				Gap(4).
				Children(func() {
					renderStreamItem(c, state, "Order Live Feed", "WSS", "wss://stream.internal.net/orders")
					renderStreamItem(c, state, "Telemetry SSE", "SSE", "https://stream.internal.net/sse")
					renderGraphQLLaunchItem(c, state)
				})

			// Section 3: History & Runner Quick Launch + Workspace Summary
			ui.Column(c).
				Gap(6).
				BorderWidth(1, 0, 0, 0).
				BorderColor(theme.BorderSubtle).
				Padding(8, 0, 0, 0).
				Children(func() {
					histRow := ui.Row(c).
						Height(28).
						PaddingX(8).
						Radius(4).
						AlignItems(ui.Center).
						Gap(8).
						Focusable()
					if histRow.Clicked() {
						state.OpenHistoryModal()
					}
					histRow.Children(func() {
						ui.Text(c, "⏱").Font(theme.FontSans).FontSize(theme.SizeBase)
						ui.Text(c, "Execution History").
							Font(theme.FontSans).
							FontSize(theme.SizeBase).
							TextColor(theme.TextSecondary)
						ui.Box(c).Grow(1)
						primitives.ShortcutCap(c, "⌘H")
					})

					runRow := ui.Row(c).
						Height(28).
						PaddingX(8).
						Radius(4).
						AlignItems(ui.Center).
						Gap(8).
						Focusable()
					if runRow.Clicked() {
						state.OpenRunnerModal()
					}
					runRow.Children(func() {
						ui.Text(c, "▶").Font(theme.FontSans).FontSize(theme.SizeBase)
						ui.Text(c, "Suite Runner").
							Font(theme.FontSans).
							FontSize(theme.SizeBase).
							TextColor(theme.TextSecondary)
						ui.Box(c).Grow(1)
						primitives.ShortcutCap(c, "⌘R")
					})

					colCount, _ := state.CollectionCounts()
					ui.Row(c).
						PaddingX(6).
						AlignItems(ui.Center).
						Children(func() {
							ui.Text(c, formatDraftSummary(len(drafts), colCount)).
								Font(theme.FontMono).
								FontSize(theme.SizeShortcut).
								TextColor(theme.TextDim)

							ui.Box(c).Grow(1)

							renderTinyHeaderAction(c, "+ Add", "New Scratchpad Draft", func() {
								state.NewScratchpadDraft()
							})
						})
				})
		})
}

func formatDraftSummary(draftCount, colCount int) string {
	draftLabel := "Drafts"
	if draftCount == 1 {
		draftLabel = "Draft"
	}
	colLabel := "Collections"
	if colCount == 1 {
		colLabel = "Collection"
	}
	return fmt.Sprintf("%d %s • %d %s", draftCount, draftLabel, colCount, colLabel)
}

func renderScratchpadSection(c *ui.Context, state *models.AppState, drafts []models.ScratchpadDraft, activeDraftID string) {
	ui.Column(c).
		Gap(4).
		Children(func() {
			ui.Row(c).
				AlignItems(ui.Center).
				Children(func() {
					ui.Text(c, "SCRATCHPAD").
						Font(theme.FontSans).
						FontSize(theme.SizeBadge).
						TextColor(theme.TextDim).
						Bold()

					ui.Box(c).Grow(1)

					renderTinyHeaderAction(c, "+ Draft", "New Scratchpad Draft", func() {
						state.NewScratchpadDraft()
					})
				})

			for _, d := range drafts {
				draft := d
				isSelected := draft.ID == activeDraftID
				bg := theme.SurfaceSubPlane
				if isSelected {
					bg = theme.SurfaceActive
				}

				row := ui.Row(c).
					Height(26).
					PaddingX(8).
					Radius(5).
					Background(bg).
					AlignItems(ui.Center).
					Gap(7).
					Focusable()

				if isSelected {
					row.Border(1, theme.BorderMedium)
				}
				if row.Clicked() {
					state.SelectDraft(draft.ID)
				}

				row.Children(func() {
					primitives.KeycapBadge(c, draft.Request.Method)
					ui.Text(c, draft.Name).
						Font(theme.FontSans).
						FontSize(theme.SizeBase).
						TextColor(theme.TextPrimary)
					ui.Box(c).Grow(1)
					renderInlineTreeBtn(c, "×", "Discard "+draft.Name, func() {
						_ = state.DeleteDraft(draft.ID)
					})
				})
			}
		})
}

func renderCollectionBranch(
	c *ui.Context,
	state *models.AppState,
	col models.Collection,
	activeColID, activeItemID, selectedDir, filter, renamingID, renameDraft string,
	collapsed map[string]bool,
) {
	if filter != "" && !collectionMatchesFilter(col, filter) {
		return
	}

	expanded := filter != "" || !collapsed[col.ID]
	caret := "▾"
	if !expanded {
		caret = "▸"
	}

	ui.Column(c).
		Gap(2).
		Children(func() {
			if renamingID == col.ID {
				renderInlineRenameRow(c, state, renameDraft, 0)
			} else {
				hdr := ui.Row(c).
					Height(26).
					PaddingX(6).
					Radius(5).
					AlignItems(ui.Center).
					Gap(6).
					Focusable()

				if col.ID == activeColID && selectedDir == "" {
					hdr.Background(theme.SurfaceCardTop)
				}
				if hdr.Clicked() {
					state.ToggleNodeExpanded(col.ID)
					state.SetSelectedFolder(col.ID, "")
				}

				hdr.Children(func() {
					ui.Text(c, caret).
						Font(theme.FontMono).
						FontSize(theme.SizeShortcut).
						TextColor(theme.TextDim)
					ui.Text(c, col.Name).
						Font(theme.FontSans).
						FontSize(theme.SizeBase).
						Bold().
						TextColor(theme.TextSecondary).
						Tooltip("Collection " + col.Name)
					ui.Box(c).Grow(1)
					renderInlineTreeBtn(c, "✎", "Rename "+col.Name, func() {
						state.BeginRenameNode("collection", col.ID, col.ID, col.Name)
					})
					renderInlineTreeBtn(c, "+", "Add request to "+col.Name, func() {
						_, _ = state.CreateRequestInCollection(col.ID, "", "New Request")
					})
				})
			}

			if expanded {
				for _, item := range col.Items {
					if filter == "" || itemMatchesFilter(item, filter) {
						renderCollectionRequestRow(c, state, col.ID, item, item.ID == activeItemID, renamingID, renameDraft, 1)
					}
				}
				for _, folder := range col.Folders {
					renderFolderBranch(c, state, col.ID, folder, activeItemID, selectedDir, filter, renamingID, renameDraft, collapsed, 1)
				}
			}
		})
}

func renderFolderBranch(
	c *ui.Context,
	state *models.AppState,
	colID string,
	folder models.Folder,
	activeItemID, selectedDir, filter, renamingID, renameDraft string,
	collapsed map[string]bool,
	depth int,
) {
	if filter != "" && !folderMatchesFilter(folder, filter) {
		return
	}

	expanded := filter != "" || !collapsed[folder.ID]
	caret := "▾"
	if !expanded {
		caret = "▸"
	}
	padLeft := depth * 12

	ui.Column(c).
		Gap(2).
		Children(func() {
			if renamingID == folder.ID {
				renderInlineRenameRow(c, state, renameDraft, padLeft)
			} else {
				row := ui.Row(c).
					Height(26).
					Padding(0, 6, 0, float32(padLeft+6)).
					Radius(5).
					AlignItems(ui.Center).
					Gap(6).
					Focusable()

				if folder.ID == selectedDir {
					row.Background(theme.SurfaceCardTop)
				}
				if row.Clicked() {
					state.ToggleNodeExpanded(folder.ID)
					state.SetSelectedFolder(colID, folder.ID)
				}

				row.Children(func() {
					ui.Text(c, caret).
						Font(theme.FontMono).
						FontSize(theme.SizeShortcut).
						TextColor(theme.TextDim)
					ui.Text(c, folder.Name).
						Font(theme.FontSans).
						FontSize(theme.SizeBase).
						TextColor(theme.TextSecondary)
					ui.Box(c).Grow(1)
					renderInlineTreeBtn(c, "+", "Add request in "+folder.Name, func() {
						_, _ = state.CreateRequestInCollection(colID, folder.ID, "New Request")
					})
					renderInlineTreeBtn(c, "✎", "Rename "+folder.Name, func() {
						state.BeginRenameNode("folder", colID, folder.ID, folder.Name)
					})
					renderInlineTreeBtn(c, "×", "Delete "+folder.Name, func() {
						_ = state.DeleteCollectionFolder(colID, folder.ID)
					})
				})
			}

			if expanded {
				for _, item := range folder.Items {
					if filter == "" || itemMatchesFilter(item, filter) {
						renderCollectionRequestRow(c, state, colID, item, item.ID == activeItemID, renamingID, renameDraft, depth+1)
					}
				}
				for _, sub := range folder.Folders {
					renderFolderBranch(c, state, colID, sub, activeItemID, selectedDir, filter, renamingID, renameDraft, collapsed, depth+1)
				}
			}
		})
}

func renderCollectionRequestRow(
	c *ui.Context,
	state *models.AppState,
	colID string,
	item models.CollectionItem,
	isSelected bool,
	renamingID, renameDraft string,
	depth int,
) {
	padLeft := depth * 12

	if renamingID == item.ID {
		renderInlineRenameRow(c, state, renameDraft, padLeft)
		return
	}

	bg := theme.SurfaceSubPlane
	if isSelected {
		bg = theme.SurfaceActive
	}

	ui.Column(c).
		Gap(3).
		Children(func() {
			row := ui.Row(c).
				Height(28).
				Padding(0, 6, 0, float32(padLeft+6)).
				Radius(5).
				Background(bg).
				AlignItems(ui.Center).
				Gap(7).
				Focusable()

			if isSelected {
				row.Border(1, theme.BorderMedium)
			}
			if row.Clicked() {
				state.SelectCollectionItem(colID, item.ID)
			}

			row.Children(func() {
				primitives.KeycapBadge(c, item.Request.Method)
				ui.Text(c, item.Name).
					Font(theme.FontSans).
					FontSize(theme.SizeBase).
					TextColor(theme.TextPrimary)
			})

			if isSelected {
				ui.Row(c).
					Padding(0, 4, 0, float32(padLeft+10)).
					Gap(4).
					AlignItems(ui.Center).
					Children(func() {
						renderRowActionPill(c, "Rename", "Rename Request", func() {
							state.BeginRenameNode("item", colID, item.ID, item.Name)
						})
						renderRowActionPill(c, "Dup", "Duplicate Request", func() {
							_, _ = state.DuplicateCollectionItem(colID, item.ID)
						})
						renderRowActionPill(c, "↑", "Move Request Up", func() {
							_ = state.ReorderCollectionItem(colID, item.ID, -1)
						})
						renderRowActionPill(c, "↓", "Move Request Down", func() {
							_ = state.ReorderCollectionItem(colID, item.ID, 1)
						})
						renderRowActionPill(c, "Remove", "Delete Request", func() {
							_ = state.DeleteCollectionItem(colID, item.ID)
						})
					})
			}
		})
}

func renderInlineRenameRow(c *ui.Context, state *models.AppState, currentDraft string, padLeft int) {
	draft := currentDraft
	ui.Row(c).
		Height(28).
		Padding(0, 4, 0, float32(padLeft+4)).
		Radius(5).
		Background(theme.CanvasDeep).
		Border(1, theme.RimFocus).
		AlignItems(ui.Center).
		Gap(4).
		Children(func() {
			in := ui.EditableText(c, &draft).
				Font(theme.FontSans).
				FontSize(theme.SizeBase).
				TextColor(theme.TextPrimary).
				Tooltip("Rename input").
				Grow(1)
			if in.Changed() {
				state.SetRenameDraft(draft)
			}
			if in.Submitted() {
				_ = state.CommitRenameNode()
			}
			renderRowActionPill(c, "✓", "Apply Rename", func() {
				_ = state.CommitRenameNode()
			})
			renderRowActionPill(c, "✕", "Cancel Rename", func() {
				state.CancelRenameNode()
			})
		})
}

func renderTinyHeaderAction(c *ui.Context, label, tooltip string, onClick func()) {
	btn := ui.Row(c).
		Height(24).
		PaddingX(6).
		Radius(4).
		Background(theme.SurfaceCardTop).
		Border(1, theme.BorderSubtle).
		Center().
		Tooltip(tooltip).
		Focusable()
	if btn.Clicked() && onClick != nil {
		onClick()
	}
	btn.Children(func() {
		ui.Text(c, label).
			Font(theme.FontMono).
			FontSize(theme.SizeShortcut).
			TextColor(theme.TextSecondary)
	})
}

func renderInlineTreeBtn(c *ui.Context, glyph, tooltip string, onClick func()) {
	btn := ui.Row(c).
		Width(18).
		Height(18).
		Radius(4).
		Center().
		Tooltip(tooltip).
		Focusable()
	if btn.Clicked() && onClick != nil {
		onClick()
	}
	btn.Children(func() {
		ui.Text(c, glyph).
			Font(theme.FontMono).
			FontSize(theme.SizeShortcut).
			TextColor(theme.TextDim)
	})
}

func renderRowActionPill(c *ui.Context, label, tooltip string, onClick func()) {
	btn := ui.Row(c).
		Height(20).
		PaddingX(5).
		Radius(4).
		Background(theme.SurfaceCardTop).
		Border(1, theme.BorderSubtle).
		Center().
		Tooltip(tooltip).
		Focusable()
	if btn.Clicked() && onClick != nil {
		onClick()
	}
	btn.Children(func() {
		ui.Text(c, label).
			Font(theme.FontMono).
			FontSize(theme.SizeShortcut).
			TextColor(theme.TextSecondary)
	})
}

func renderStreamItem(c *ui.Context, state *models.AppState, name, proto, url string) {
	var isSelected bool
	state.View(func(s *models.AppState) {
		isSelected = s.WorkspaceMode == "stream" && s.Stream.URL == url
	})
	bg := theme.SurfaceSubPlane
	if isSelected {
		bg = theme.SurfaceActive
	}

	row := ui.Row(c).
		Height(28).
		PaddingX(8).
		Radius(5).
		Background(bg).
		AlignItems(ui.Center).
		Gap(8).
		Focusable()

	if isSelected {
		row.Border(1, theme.BorderMedium)
	}
	if row.Clicked() {
		state.OpenStreamEndpoint(models.StreamProtocol(proto), name, url)
	}

	row.Children(func() {
		primitives.KeycapBadge(c, proto)
		ui.Text(c, name).
			Font(theme.FontSans).
			FontSize(theme.SizeBase).
			TextColor(theme.TextPrimary)
	})
}

func renderGraphQLLaunchItem(c *ui.Context, state *models.AppState) {
	var isSelected bool
	state.View(func(s *models.AppState) {
		isSelected = s.WorkspaceMode == "graphql"
	})
	bg := theme.SurfaceSubPlane
	if isSelected {
		bg = theme.SurfaceActive
	}
	row := ui.Row(c).
		Height(28).
		PaddingX(8).
		Radius(5).
		Background(bg).
		AlignItems(ui.Center).
		Gap(8).
		Focusable()
	if isSelected {
		row.Border(1, theme.BorderMedium)
	}
	if row.Clicked() {
		state.OpenGraphQLExplorer("")
	}
	row.Children(func() {
		primitives.KeycapBadge(c, "GQL")
		ui.Text(c, "GraphQL Explorer").
			Font(theme.FontSans).
			FontSize(theme.SizeBase).
			TextColor(theme.TextPrimary)
	})
}

func itemMatchesFilter(item models.CollectionItem, filter string) bool {
	q := strings.ToLower(strings.TrimSpace(filter))
	if q == "" {
		return true
	}
	return strings.Contains(strings.ToLower(item.Name), q) ||
		strings.Contains(strings.ToLower(item.Request.Method), q) ||
		strings.Contains(strings.ToLower(item.Request.URL), q)
}

func folderMatchesFilter(folder models.Folder, filter string) bool {
	q := strings.ToLower(strings.TrimSpace(filter))
	if q == "" || strings.Contains(strings.ToLower(folder.Name), q) {
		return true
	}
	for _, item := range folder.Items {
		if itemMatchesFilter(item, q) {
			return true
		}
	}
	for _, sub := range folder.Folders {
		if folderMatchesFilter(sub, q) {
			return true
		}
	}
	return false
}

func collectionMatchesFilter(col models.Collection, filter string) bool {
	q := strings.ToLower(strings.TrimSpace(filter))
	if q == "" || strings.Contains(strings.ToLower(col.Name), q) {
		return true
	}
	for _, item := range col.Items {
		if itemMatchesFilter(item, q) {
			return true
		}
	}
	for _, folder := range col.Folders {
		if folderMatchesFilter(folder, q) {
			return true
		}
	}
	return false
}
