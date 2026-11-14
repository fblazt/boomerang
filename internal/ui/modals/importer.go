package modals

import (
	"fmt"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/primitives"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

type importSyncedEditor struct {
	state    *editor.State
	lastText string
	lastVer  uint64
}

var (
	importEdMu  sync.Mutex
	importEdMap = make(map[*models.AppState]*importSyncedEditor)
)

func getImportEditor(c *ui.Context, state *models.AppState) *importSyncedEditor {
	importEdMu.Lock()
	defer importEdMu.Unlock()
	ed, ok := importEdMap[state]
	if !ok {
		buf := editor.NewBuffer("")
		ed = &importSyncedEditor{
			state: &editor.State{
				Buffer:          buf,
				Font:            ui.Font{Family: theme.FontMono, Size: theme.SizeBase},
				LineHeight:      1.5,
				TabSize:         2,
				SoftWrap:        true,
				NoFolding:       true,
				NoScrollPastEnd: true,
				KeepScrollbars:  true,
				Placeholder:     "Paste a cURL command, OpenAPI 3.1 JSON/YAML, Postman v2.1 JSON, or Bruno .bru snippet...",
				Label:           "Import Paste Editor",
			},
			lastVer: buf.Version(),
		}
		importEdMap[state] = ed
	}
	ed.state.Theme = editor.DefaultTheme(c.Theme())
	ed.state.Theme.Background = theme.CanvasDeep
	ed.state.Theme.Text = theme.TextPrimary
	return ed
}

// ImportModalHost wraps the single-step Universal Importer dialog in ui.DialogBase.
func ImportModalHost(c *ui.Context, state *models.AppState) ui.Element {
	var show bool
	state.View(func(s *models.AppState) {
		show = s.ShowImportModal
	})
	if !show {
		return ui.Element{}
	}

	dlg := ui.DialogBase(c, &show, func(backdrop ui.Element, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.72))
		ImportModal(c, state)
	})

	if dlg.Dismissed() || !show || c.Shortcut(0, ui.KeyEscape) {
		state.CloseImportModal()
	}
	return dlg
}

// ImportModal renders drag-and-drop file loading, cURL/spec paste auto-detection,
// collection preview breakdown, and one-click import application.
func ImportModal(c *ui.Context, state *models.AppState) ui.Element {
	var (
		rawText        string
		filePath       string
		targetName     string
		detectedFormat models.ImportFormat
		preview        *models.ImportPreview
		importErr      string
	)
	state.View(func(s *models.AppState) {
		rawText = s.ImportRawText
		filePath = s.ImportFilePath
		targetName = s.ImportTargetName
		detectedFormat = s.ImportDetectedFormat
		preview = s.ImportPreviewData
		importErr = s.ImportError
	})

	formatLabel := "Auto-Detect"
	if detectedFormat != models.ImportFormatUnknown {
		formatLabel = detectedFormat.DisplayName()
	}

	return ui.Column(c).
		Width(820).
		Height(620).
		Background(theme.SurfacePlaneTop).
		Radius(12).
		Border(1, theme.BorderStrong).
		Children(func() {
			renderImportHeader(c, state, formatLabel)

			ui.Column(c).
				Grow(1).
				Padding(16).
				Gap(12).
				Background(theme.CanvasDeep).
				Radius(0, 0, 12, 12).
				Children(func() {
					renderImportDropZone(c, state, filePath)
					renderImportPasteAndPreview(c, state, rawText, targetName, preview, importErr)
					renderImportFooter(c, state, preview)
				})
		})
}

func renderImportHeader(c *ui.Context, state *models.AppState, formatLabel string) {
	ui.Row(c).
		Height(48).
		PaddingX(16).
		AlignItems(ui.Center).
		Gap(10).
		Background(theme.SurfacePlaneTop).
		Radius(12, 12, 0, 0).
		BorderWidth(0, 0, 1, 0).
		BorderColor(theme.BorderSubtle).
		Children(func() {
			ui.Text(c, "⇣").Font(theme.FontMono).FontSize(theme.SizeHeading).TextColor(theme.RimFocus)
			ui.Text(c, "Universal API Importer").
				Font(theme.FontSans).
				FontSize(theme.SizeHeading).
				Bold().
				TextColor(theme.TextPrimary)
			primitives.ShortcutCap(c, "⌘I")

			ui.Box(c).Grow(1)

			ui.Row(c).
				PaddingX(8).
				PaddingY(3).
				Radius(4).
				Background(theme.SurfaceActive).
				Border(1, theme.RimFocus).
				Children(func() {
					ui.Text(c, formatLabel).
						Font(theme.FontMono).
						FontSize(theme.SizeBadge).
						Bold().
						TextColor(theme.RimFocus)
				})

			closeBtn := ui.Row(c).
				Width(26).
				Height(26).
				Radius(5).
				Background(theme.SurfaceCardTop).
				Border(1, theme.BorderSubtle).
				Center().
				Focusable()
			if closeBtn.Clicked() {
				state.CloseImportModal()
			}
			closeBtn.Children(func() {
				ui.Text(c, "✕").Font(theme.FontSans).FontSize(theme.SizeBase).TextColor(theme.TextMuted)
			})
		})
}

func renderImportDropZone(c *ui.Context, state *models.AppState, filePath string) {
	dropBorder := theme.BorderMedium
	dropBg := theme.SurfaceSubPlane
	dropZone := ui.Row(c).
		Height(68).
		PaddingX(16).
		Radius(8).
		AlignItems(ui.Center).
		Gap(12)

	if dropZone.FileDragOver() {
		dropBorder = theme.RimFocus
		dropBg = theme.SurfaceActive
	}
	if files := dropZone.DroppedFiles(); len(files) > 0 {
		state.LoadImportPath(files[0])
	}

	dropZone.
		Background(dropBg).
		Border(1, dropBorder).
		Children(func() {
			ui.Column(c).
				Grow(1).
				Gap(3).
				Children(func() {
					title := "Drop OpenAPI 3.1 (.json/.yaml), Postman v2.1 (.json), or Bruno (.bru) here"
					if filePath != "" {
						title = "Loaded: " + filePath
					}
					ui.Text(c, title).
						Font(theme.FontSans).
						FontSize(theme.SizeBase).
						Bold().
						TextColor(theme.TextPrimary)
					ui.Text(c, "Supports cURL paste, Postman v2.1 collections, OpenAPI 3.0/3.1 specs, and Bruno files.").
						Font(theme.FontSans).
						FontSize(theme.SizeBadge).
						TextColor(theme.TextMuted)
				})

			renderRunnerChip(c, "Choose File", false, func() {
				state.PickFile("Import API File", []string{"json", "yaml", "yml", "bru", "txt"}, func(path string) {
					if strings.TrimSpace(path) != "" {
						state.LoadImportPath(path)
					}
				})
			})
		})
}

func renderImportPasteAndPreview(
	c *ui.Context,
	state *models.AppState,
	rawText string,
	targetName string,
	preview *models.ImportPreview,
	importErr string,
) {
	ui.Row(c).
		Grow(1).
		Gap(12).
		AlignItems(ui.Stretch).
		Children(func() {
			// Left column: Paste / Edit cURL or raw spec
			ui.Column(c).
				Grow(1).
				Padding(12).
				Gap(8).
				Radius(8).
				Background(theme.SurfaceSubPlane).
				Border(1, theme.BorderSubtle).
				Children(func() {
					ui.Row(c).
						AlignItems(ui.Center).
						Gap(8).
						Children(func() {
							ui.Text(c, "PASTE CURL / SPEC").
								Font(theme.FontMono).
								FontSize(theme.SizeShortcut).
								Bold().
								TextColor(theme.TextSecondary)
							ui.Box(c).Grow(1)
							renderRunnerChip(c, "Clear", false, func() {
								state.SetImportRawText("")
							})
						})

					ed := getImportEditor(c, state)
					curVer := ed.state.Buffer.Version()
					if curVer != ed.lastVer {
						curText := ed.state.Buffer.String()
						ed.lastVer = curVer
						if curText != ed.lastText {
							ed.lastText = curText
							state.SetImportRawText(curText)
						}
					} else if rawText != ed.lastText {
						ed.state.Buffer.Set(rawText)
						ed.lastText = rawText
						ed.lastVer = ed.state.Buffer.Version()
					}

					ui.Column(c).
						Grow(1).
						Background(theme.CanvasDeep).
						Radius(6).
						Padding(4).
						Border(1, theme.BorderSubtle).
						Children(func() {
							editor.View(c, ed.state).Fill()
						})
				})

			// Right column: Live parsed collection preview
			ui.Column(c).
				Width(300).
				Padding(12).
				Gap(10).
				Radius(8).
				Background(theme.SurfaceSubPlane).
				Border(1, theme.BorderSubtle).
				Children(func() {
					ui.Text(c, "IMPORT PREVIEW").
						Font(theme.FontMono).
						FontSize(theme.SizeShortcut).
						Bold().
						TextColor(theme.TextSecondary)

					if importErr != "" {
						ui.Row(c).
							Padding(8, 10, 8, 10).
							Radius(6).
							Background(ui.RGBA(0xEF, 0x44, 0x44, 0.12)).
							Border(1, theme.RimDanger).
							Children(func() {
								ui.Text(c, importErr).
									Font(theme.FontMono).
									FontSize(theme.SizeBadge).
									TextColor(theme.RimDanger)
							})
						return
					}

					if preview == nil {
						ui.Column(c).
							Grow(1).
							Center().
							Children(func() {
								ui.Text(c, "Paste cURL or drop a file to inspect requests & folders before importing.").
									Font(theme.FontSans).
									FontSize(theme.SizeBase).
									TextColor(theme.TextMuted)
							})
						return
					}

					ui.Text(c, "COLLECTION NAME").
						Font(theme.FontMono).
						FontSize(theme.SizeShortcut).
						TextColor(theme.TextDim)

					nameDraft := targetName
					ui.Row(c).
						Height(30).
						PaddingX(8).
						Radius(5).
						Background(theme.CanvasDeep).
						Border(1, theme.BorderMedium).
						AlignItems(ui.Center).
						Children(func() {
							in := ui.EditableText(c, &nameDraft).
								Font(theme.FontSans).
								FontSize(theme.SizeBase).
								TextColor(theme.TextPrimary).
								Grow(1)
							if in.Changed() {
								state.SetImportTargetName(nameDraft)
							}
						})

					ui.Row(c).
						Gap(8).
						Children(func() {
							renderMetricCard(c, "FORMAT", preview.FormatLabel, theme.RimFocus)
							renderMetricCard(c, "REQUESTS", fmt.Sprintf("%d", preview.RequestCount), ui.RGB(0x34, 0xD3, 0x99))
							renderMetricCard(c, "FOLDERS", fmt.Sprintf("%d", preview.FolderCount), theme.TextSecondary)
						})

					ui.Scroll(c).
						Grow(1).
						Gap(4).
						Children(func() {
							for _, item := range preview.Collection.Items {
								renderPreviewEndpointRow(c, item)
							}
							for _, folder := range preview.Collection.Folders {
								ui.Text(c, "▸ "+folder.Name).
									Font(theme.FontSans).
									FontSize(theme.SizeBadge).
									Bold().
									TextColor(theme.TextSecondary)
								for _, item := range folder.Items {
									renderPreviewEndpointRow(c, item)
								}
							}
						})
				})
		})
}

func renderPreviewEndpointRow(c *ui.Context, item models.CollectionItem) {
	ui.Row(c).
		Height(24).
		PaddingX(6).
		Radius(4).
		Background(theme.CanvasDeep).
		AlignItems(ui.Center).
		Gap(6).
		Children(func() {
			ui.Text(c, item.Request.Method).
				Font(theme.FontMono).
				FontSize(theme.SizeShortcut).
				Bold().
				TextColor(theme.RimFocus)
			ui.Text(c, item.Name).
				Font(theme.FontSans).
				FontSize(theme.SizeBadge).
				TextColor(theme.TextPrimary)
		})
}

func renderImportFooter(c *ui.Context, state *models.AppState, preview *models.ImportPreview) {
	ui.Row(c).
		AlignItems(ui.Center).
		Gap(10).
		Children(func() {
			ui.Box(c).Grow(1)
			primitives.CancelButton(c, func() {
				state.CloseImportModal()
			})
			if preview != nil {
				primitives.PrimaryButton(c, "Import Collection", func() {
					_ = state.ConfirmImport()
				})
			}
		})
}
