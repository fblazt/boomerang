package views

import (
	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/modals"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

// MainWorkspaceView coordinates the center stage of the application shell based on WorkspaceMode.
func MainWorkspaceView(c *ui.Context, state *models.AppState) ui.Element {
	var mode string
	state.View(func(s *models.AppState) {
		mode = s.WorkspaceMode
	})
	if mode == "stream" {
		return RealtimeView(c, state)
	}

	return ui.Column(c).
		Grow(1).
		Padding(16).
		Gap(14).
		Background(theme.CanvasDeep).
		Children(func() {
			// 1. URL Bar Row strictly following Brilliant Frame 304-313
			UrlBarView(c, state)

			// 2. Request Configuration Tabs (Params, Headers, Body, Auth, Settings)
			RequestTabsView(c, state)

			// 3. Response Panel with Telemetry Waterfall & Payload Viewer
			ResponseView(c, state)
		})
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

			// Environment & Secret Vault Manager Modal Overlay
			modals.VaultModalHost(c, state)

			// Execution History & HAR Timeline Modal Overlay
			modals.HistoryModalHost(c, state)
		})
}
