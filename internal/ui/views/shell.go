package views

import (
	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/primitives"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

// MainWorkspaceView coordinates the center stage of the application shell.
func MainWorkspaceView(c *ui.Context, state *models.AppState) ui.Element {
	return ui.Column(c).
		Grow(1).
		Padding(16).
		Gap(16).
		Background(theme.CanvasDeep).
		Children(func() {
			// Elevated Request Card Header
			primitives.ElevatedPlane(c, func() {
				ui.Row(c).
					AlignItems(ui.Center).
					Gap(12).
					Children(func() {
						state.View(func(s *models.AppState) {
							primitives.KeycapBadge(c, s.ActiveRequest.Method)
						})

						// Elevated URL Bar Container
						primitives.InputPlane(c, true, false, func() {
							primitives.VariableChip(c, "{{base_url}}")
							ui.Box(c).Width(6)
							state.View(func(s *models.AppState) {
								ui.Text(c, s.ActiveRequest.URL).
									Font(theme.FontMono).
									FontSize(theme.SizeBody).
									TextColor(theme.TextPrimary)
							})
						})

						ui.Box(c).Grow(1)

						// Send Button with ⌘↵ shortcut cap
						primitives.PrimaryButton(c, "Send", func() {
							state.Mutate(func(s *models.AppState) {
								s.IsLoading = true
							})
						})
						primitives.ShortcutCap(c, "⌘↵")
					})
			})

			// Content area placeholder for requests & response
			ui.Column(c).
				Grow(1).
				Radius(8).
				Background(theme.SurfaceSubPlane).
				Border(1, theme.BorderSubtle).
				Padding(16).
				Children(func() {
					state.View(func(s *models.AppState) {
						ui.Text(c, "Active Request: "+s.ActiveRequest.Name).
							Font(theme.FontSans).
							FontSize(theme.SizeHeading).
							Bold().
							TextColor(theme.TextPrimary)

						ui.Text(c, "Endpoint: "+s.ActiveRequest.URL).
							Font(theme.FontMono).
							FontSize(theme.SizeBase).
							TextColor(theme.TextMuted)
					})
				})
		})
}

// ShellView builds the complete three-pane application layout:
// Header at the top, Sidebar + MainWorkspace in the middle, and StatusBar at the bottom.
func ShellView(c *ui.Context, state *models.AppState) ui.Element {
	return ui.Column(c).
		Grow(1).
		Background(theme.CanvasBase).
		Children(func() {
			// Top Header (Titlebar / Breadcrumb / Env)
			HeaderView(c, state)

			// Center Stage: Sidebar + Main Workspace
			ui.Row(c).
				Grow(1).
				Children(func() {
					SidebarView(c, state)
					MainWorkspaceView(c, state)
				})

			// Bottom Status Bar (Sockets / Metrics)
			StatusBarView(c, state)
		})
}
