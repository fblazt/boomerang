package views

import (
	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/primitives"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

// MainWorkspaceView coordinates the center stage of the application shell:
// URL Bar with method selector and Send/Cancel controls,
// Request Configuration Tabs (Params, Headers, Body, Auth, Settings),
// and Response Inspection Panel with Telemetry Waterfall.
func MainWorkspaceView(c *ui.Context, state *models.AppState) ui.Element {
	return ui.Column(c).
		Grow(1).
		Padding(16).
		Gap(14).
		Background(theme.CanvasDeep).
		Children(func() {
			// 1. URL Bar Row (URL Input Plane + Send/Cancel + Save) strictly following Brilliant Frame 304-313
			ui.Row(c).
				AlignItems(ui.Center).
				Gap(8).
				Children(func() {
					// Elevated URL Bar Container with Method inside and PROD chip
					primitives.UrlBarInputPlane(c, func() {
						var currentMethod string
						state.View(func(s *models.AppState) {
							currentMethod = s.ActiveRequest.Method
						})

						// Clickable method keycap that cycles through HTTP verbs
						methodBtn := ui.Row(c).
							Center().
							Focusable()

						if methodBtn.Clicked() {
							nextMethod := getNextMethod(currentMethod)
							state.SetMethod(nextMethod)
						}

						methodBtn.Children(func() {
							primitives.DropdownKeycapBadge(c, currentMethod)
						})

						ui.Box(c).Width(10)

						primitives.VariableChip(c, "{{base_url}}")

						ui.Box(c).Width(8)

						state.View(func(s *models.AppState) {
							ui.Text(c, s.ActiveRequest.URL).
								Font(theme.FontMono).
								FontSize(theme.SizeBody).
								TextColor(theme.TextPrimary)
						})

						ui.Box(c).Grow(1)

						// Environment indicator chip on the right inside input bar (Brilliant Frame 308)
						state.View(func(s *models.AppState) {
							ui.Row(c).
								AlignItems(ui.Center).
								Gap(5).
								Children(func() {
									ui.Box(c).Size(6, 6).Radius(3).Background(theme.RimFocus)
									ui.Text(c, s.ActiveEnv).
										Font(theme.FontMono).
										FontSize(theme.SizeBadge).
										TextColor(theme.TextMuted)
								})
						})
					}).Grow(1)

					// Action Buttons: Send/Cancel + Save with integrated shortcut caps
					var isLoading bool
					state.View(func(s *models.AppState) {
						isLoading = s.IsLoading
					})

					if isLoading {
						primitives.CancelButton(c, func() {
							state.CancelRequest()
						})
					} else {
						primitives.SendButton(c, func() {
							state.SendRequest()
						})
					}

					primitives.SaveButton(c, func() {})
				})

			// 2. Request Configuration Tabs (Params, Headers, Body, Auth, Settings)
			RequestTabsView(c, state)

			// 3. Response Panel with Telemetry Waterfall & Payload Viewer
			ResponseView(c, state)
		})
}

func getNextMethod(current string) string {
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
	for i, m := range methods {
		if m == current {
			return methods[(i+1)%len(methods)]
		}
	}
	return "GET"
}

// ShellView builds the complete three-pane application layout:
// Header at the top, Sidebar + MainWorkspace in the middle, and StatusBar at the bottom.
func ShellView(c *ui.Context, state *models.AppState) ui.Element {
	return ui.Column(c).
		Fill().
		Background(theme.CanvasBase).
		Children(func() {
			// Top Header (Titlebar / Breadcrumb / Env)
			HeaderView(c, state)

			// Center Stage: Sidebar + Main Workspace
			ui.Row(c).
				Grow(1).
				AlignItems(ui.Stretch).
				Children(func() {
					SidebarView(c, state)
					MainWorkspaceView(c, state)
				})

			// Bottom Status Bar (Sockets / Metrics)
			StatusBarView(c, state)
		})
}
