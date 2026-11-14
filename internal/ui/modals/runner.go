package modals

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/primitives"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

// RunnerModalHost wraps the Collection Suite Runner dialog in ui.DialogBase with backdrop & Escape dismissal.
func RunnerModalHost(c *ui.Context, state *models.AppState) ui.Element {
	var show bool
	state.View(func(s *models.AppState) {
		show = s.ShowRunnerModal
	})
	if !show {
		return ui.Element{}
	}

	dlg := ui.DialogBase(c, &show, func(backdrop ui.Element, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.72))
		RunnerModal(c, state)
	})

	if dlg.Dismissed() || !show || c.Shortcut(0, ui.KeyEscape) {
		state.CloseRunnerModal()
	}
	return dlg
}

// RunnerModal renders the Collection Suite Runner configuration, concurrency slider (1..32),
// live progress bar, and execution results summary.
func RunnerModal(c *ui.Context, state *models.AppState) ui.Element {
	var (
		cols          []models.Collection
		selectedColID string
		concurrency   int
		stopOnErr     bool
		running       bool
		progress      models.SuiteRunProgress
		report        *models.SuiteRunReport
		statusMsg     string
	)
	state.View(func(s *models.AppState) {
		cols = s.Collections
		selectedColID = s.RunnerCollectionID
		concurrency = s.RunnerConcurrency
		stopOnErr = s.RunnerStopOnError
		running = s.RunnerRunning
		progress = s.RunnerProgress
		report = s.RunnerReport
		statusMsg = s.RunnerStatusMessage
	})
	if concurrency < 1 {
		concurrency = 4
	}

	return ui.Column(c).
		Width(840).
		Height(620).
		Background(theme.SurfacePlaneTop).
		Radius(12).
		Border(1, theme.BorderStrong).
		Children(func() {
			renderRunnerHeader(c, state, statusMsg)
			renderRunnerControls(c, state, cols, selectedColID, concurrency, stopOnErr, running)
			renderRunnerProgressBar(c, running, progress)
			renderRunnerResultsPlaceholder(c, report, running)
		})
}

func renderRunnerHeader(c *ui.Context, state *models.AppState, statusMsg string) {
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
			ui.Text(c, "▶").Font(theme.FontSans).FontSize(theme.SizeHeading).TextColor(theme.RimFocus)
			ui.Text(c, "Collection Suite Runner").
				Font(theme.FontSans).
				FontSize(theme.SizeHeading).
				Bold().
				TextColor(theme.TextPrimary)
			primitives.ShortcutCap(c, "⌘R")
			ui.Box(c).Grow(1)
			if statusMsg != "" {
				ui.Text(c, statusMsg).
					Font(theme.FontSans).
					FontSize(theme.SizeBadge).
					TextColor(theme.TextSecondary)
			}
			closeBtn := ui.Row(c).
				Width(26).
				Height(26).
				Radius(5).
				Background(theme.SurfaceCardTop).
				Border(1, theme.BorderSubtle).
				Center().
				Focusable()
			if closeBtn.Clicked() {
				state.CloseRunnerModal()
			}
			closeBtn.Children(func() {
				ui.Text(c, "✕").Font(theme.FontSans).FontSize(theme.SizeBase).TextColor(theme.TextMuted)
			})
		})
}

func renderRunnerControls(
	c *ui.Context,
	state *models.AppState,
	cols []models.Collection,
	selectedColID string,
	concurrency int,
	stopOnErr bool,
	running bool,
) {
	ui.Column(c).
		Padding(14, 16, 12, 16).
		Gap(12).
		Background(theme.SurfaceSubPlane).
		BorderWidth(0, 0, 1, 0).
		BorderColor(theme.BorderSubtle).
		Children(func() {
			// Target Collection Selector Strip
			ui.Row(c).
				AlignItems(ui.Center).
				Gap(8).
				Wrap().
				Children(func() {
					ui.Text(c, "COLLECTION").
						Font(theme.FontMono).
						FontSize(theme.SizeShortcut).
						Bold().
						TextColor(theme.TextDim)

					if len(cols) == 0 {
						ui.Text(c, "No workspace collections available").
							Font(theme.FontSans).
							FontSize(theme.SizeBase).
							TextColor(theme.TextMuted)
					} else {
						for _, col := range cols {
							colItem := col
							isSel := colItem.ID == selectedColID
							renderRunnerChip(c, colItem.Name, isSel, func() {
								if !running {
									state.SetRunnerCollection(colItem.ID)
								}
							})
						}
					}
				})

			// Concurrency Slider (1..32 threads) + Run / Stop Controls
			ui.Row(c).
				AlignItems(ui.Center).
				Gap(12).
				Children(func() {
					ui.Text(c, "WORKERS").
						Font(theme.FontMono).
						FontSize(theme.SizeShortcut).
						Bold().
						TextColor(theme.TextDim)

					concVal := float64(concurrency)
					slider := ui.StepSlider(c, &concVal, 1, 32, 1).
						Width(190)
					if slider.Changed() && !running {
						state.SetRunnerConcurrency(int(math.Round(concVal)))
					}

					renderRunnerChip(c, "-", false, func() {
						if !running {
							state.SetRunnerConcurrency(concurrency - 1)
						}
					})
					renderRunnerChip(c, "+", false, func() {
						if !running {
							state.SetRunnerConcurrency(concurrency + 1)
						}
					})

					for _, preset := range []int{1, 4, 8, 16, 32} {
						p := preset
						renderRunnerChip(c, fmt.Sprintf("%dx", p), concurrency == p, func() {
							if !running {
								state.SetRunnerConcurrency(p)
							}
						})
					}

					ui.Row(c).
						PaddingX(8).
						PaddingY(4).
						Radius(4).
						Background(theme.CanvasDeep).
						Border(1, theme.BorderMedium).
						Children(func() {
							ui.Text(c, fmt.Sprintf("%d threads", concurrency)).
								Font(theme.FontMono).
								FontSize(theme.SizeBadge).
								Bold().
								TextColor(theme.RimFocus)
						})

					renderRunnerChip(c, "Stop on Fail", stopOnErr, func() {
						if !running {
							state.ToggleRunnerStopOnError()
						}
					})

					ui.Box(c).Grow(1)

					if running {
						cancelBtn := ui.Row(c).
							Height(32).
							PaddingX(14).
							Radius(6).
							Background(ui.RGBA(0xEF, 0x44, 0x44, 0.16)).
							Border(1, theme.RimDanger).
							Center().
							Focusable()
						if cancelBtn.Clicked() {
							state.CancelSuiteRun()
						}
						cancelBtn.Children(func() {
							ui.Text(c, "Cancel Run").
								Font(theme.FontSans).
								FontSize(theme.SizeBase).
								Bold().
								TextColor(theme.RimDanger)
						})
					} else {
						primitives.PrimaryButton(c, "Run Suite", func() {
							state.StartSuiteRun()
						})
					}
				})
		})
}

func renderRunnerProgressBar(c *ui.Context, running bool, prog models.SuiteRunProgress) {
	if !running && prog.Total == 0 {
		return
	}
	frac := 0.0
	if prog.Total > 0 {
		frac = float64(prog.Completed) / float64(prog.Total)
	}
	ui.Column(c).
		Padding(10, 16, 10, 16).
		Gap(6).
		Background(theme.CanvasDeep).
		BorderWidth(0, 0, 1, 0).
		BorderColor(theme.BorderSubtle).
		Children(func() {
			ui.Row(c).
				AlignItems(ui.Center).
				Gap(10).
				Children(func() {
					ui.Text(c, fmt.Sprintf("PROGRESS %d / %d", prog.Completed, prog.Total)).
						Font(theme.FontMono).
						FontSize(theme.SizeShortcut).
						Bold().
						TextColor(theme.TextSecondary)
					ui.Text(c, fmt.Sprintf("Active Workers: %d", prog.ActiveWorkers)).
						Font(theme.FontMono).
						FontSize(theme.SizeShortcut).
						TextColor(theme.RimFocus)
					ui.Box(c).Grow(1)
					if prog.LastStep.Name != "" {
						ui.Text(c, fmt.Sprintf("Last: %s %s (%d ms)", prog.LastStep.Method, prog.LastStep.Name, prog.LastStep.DurationMs)).
							Font(theme.FontMono).
							FontSize(theme.SizeShortcut).
							TextColor(theme.TextDim)
					}
				})
			ui.Progress(c, frac).
				Height(6)
		})
}

func renderRunnerResultsPlaceholder(c *ui.Context, report *models.SuiteRunReport, running bool) {
	ui.Column(c).
		Grow(1).
		Padding(16).
		Gap(10).
		Background(theme.CanvasDeep).
		Radius(0, 0, 12, 12).
		Children(func() {
			if report == nil {
				msg := "Choose a collection, set worker concurrency (1 to 32 threads), and click Run Suite."
				if running {
					msg = "Executing collection suite across worker pool..."
				}
				ui.Column(c).Grow(1).Center().Children(func() {
					ui.Text(c, msg).
						Font(theme.FontSans).
						FontSize(theme.SizeBase).
						TextColor(theme.TextMuted)
				})
				return
			}

			ui.Text(c, fmt.Sprintf("Executed %d/%d requests (%d passed, %d failed)",
				len(report.Steps), report.TotalRequests, report.PassedRequests, report.FailedRequests)).
				Font(theme.FontMono).
				FontSize(theme.SizeBase).
				TextColor(theme.TextPrimary)
		})
}

func renderRunnerChip(c *ui.Context, label string, active bool, onClick func()) {
	bg := theme.SurfaceCardTop
	border := theme.BorderSubtle
	txt := theme.TextSecondary
	if active {
		bg = theme.SurfaceActive
		border = theme.RimFocus
		txt = theme.RimFocus
	}
	btn := ui.Row(c).
		Height(24).
		PaddingX(8).
		Radius(4).
		Background(bg).
		Border(1, border).
		Center().
		Focusable()
	if btn.Clicked() && onClick != nil {
		onClick()
	}
	btn.Children(func() {
		ui.Text(c, label).
			Font(theme.FontMono).
			FontSize(theme.SizeBadge).
			TextColor(txt)
	})
}
