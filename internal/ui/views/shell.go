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
		Gap(14).
		Background(theme.CanvasDeep).
		Children(func() {
			// 1. Elevated Request Bar (URL, Method, Chip, Send, Save)
			primitives.ElevatedPlane(c, func() {
				ui.Row(c).
					AlignItems(ui.Center).
					Gap(10).
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

						// Action Buttons: Send + Save
						primitives.PrimaryButton(c, "Send", func() {
							state.Mutate(func(s *models.AppState) {
								s.IsLoading = true
							})
						})
						primitives.ShortcutCap(c, "⌘↵")

						ui.Box(c).Width(4)

						primitives.SecondaryButton(c, "Save", func() {})
						primitives.ShortcutCap(c, "⌘S")
					})
			})

			// 2. Request Configuration Tabs (Params, Headers, Body, Auth, Settings)
			renderRequestTabs(c)

			// 3. Params Table Preview
			renderParamsTable(c)

			// 4. Response Panel with Telemetry & JSON Body
			renderResponsePanel(c, state)
		})
}

func renderRequestTabs(c *ui.Context) {
	ui.Row(c).
		AlignItems(ui.Center).
		Gap(20).
		PaddingX(4).
		BorderWidth(0, 0, 1, 0).
		BorderColor(theme.BorderSubtle).
		Children(func() {
			// Params Tab (Active)
			ui.Row(c).
				AlignItems(ui.Center).
				Gap(6).
				PaddingY(8).
				BorderWidth(0, 0, 2, 0).
				BorderColor(theme.RimFocus).
				Children(func() {
					ui.Text(c, "Params").
						Font(theme.FontSans).
						FontSize(theme.SizeBody).
						Bold().
						TextColor(theme.TextPrimary)

					ui.Row(c).
						PaddingX(5).
						PaddingY(1).
						Radius(4).
						Background(theme.SurfaceActive).
						Border(1, theme.BorderSubtle).
						Children(func() {
							ui.Text(c, "3").
								Font(theme.FontMono).
								FontSize(theme.SizeCaption).
								Bold().
								TextColor(theme.RimFocus)
						})
				})

			renderTabItem(c, "Headers", "3")
			renderTabItem(c, "Body", "")
			renderTabItem(c, "Auth (Bearer)", "")
			renderTabItem(c, "Settings", "")
		})
}

func renderTabItem(c *ui.Context, label, badge string) {
	ui.Row(c).
		AlignItems(ui.Center).
		Gap(6).
		PaddingY(8).
		Children(func() {
			ui.Text(c, label).
				Font(theme.FontSans).
				FontSize(theme.SizeBody).
				TextColor(theme.TextMuted)

			if badge != "" {
				ui.Row(c).
					PaddingX(5).
					PaddingY(1).
					Radius(4).
					Background(theme.SurfaceSubPlane).
					Border(1, theme.BorderSubtle).
					Children(func() {
						ui.Text(c, badge).
							Font(theme.FontMono).
							FontSize(theme.SizeCaption).
							TextColor(theme.TextDim)
					})
			}
		})
}

func renderParamsTable(c *ui.Context) {
	ui.Column(c).
		Gap(6).
		Children(func() {
			// Table Header
			ui.Row(c).
				PaddingX(8).
				AlignItems(ui.Center).
				Children(func() {
					ui.Box(c).Width(24) // Checkbox space
					ui.Text(c, "KEY").Font(theme.FontSans).FontSize(theme.SizeCaption).Bold().TextColor(theme.TextDim).Width(160)
					ui.Text(c, "VALUE").Font(theme.FontSans).FontSize(theme.SizeCaption).Bold().TextColor(theme.TextDim).Width(220)
					ui.Text(c, "DESCRIPTION").Font(theme.FontSans).FontSize(theme.SizeCaption).Bold().TextColor(theme.TextDim).Grow(1)
				})

			// Rows
			renderParamRow(c, "include", "teams,permissions", "Fetch nested user relations", true)
			renderParamRow(c, "fields", "id,name,email,role", "Projection whitelist", true)
			renderParamRow(c, "debug", "1", "Trace telemetry", false)
		})
}

func renderParamRow(c *ui.Context, key, val, desc string, active bool) {
	bg := theme.SurfaceCardTop
	textColor := theme.TextPrimary
	if !active {
		bg = theme.SurfaceSubPlane
		textColor = theme.TextDim
	}

	ui.Row(c).
		Height(30).
		PaddingX(8).
		Radius(6).
		Background(bg).
		Border(1, theme.BorderSubtle).
		AlignItems(ui.Center).
		Children(func() {
			// Active toggle indicator
			ui.Row(c).
				Width(16).
				Height(16).
				Radius(3).
				Background(theme.SurfaceSubPlane).
				Border(1, theme.BorderSubtle).
				Center().
				Children(func() {
					if active {
						ui.Text(c, "✓").Font(theme.FontMono).FontSize(10).TextColor(theme.RimFocus)
					}
				})

			ui.Box(c).Width(8)

			// Key
			ui.Text(c, key).
				Font(theme.FontMono).
				FontSize(theme.SizeBase).
				TextColor(textColor).
				Width(160)

			// Value
			ui.Text(c, val).
				Font(theme.FontMono).
				FontSize(theme.SizeBase).
				TextColor(theme.RimFocus).
				Width(220)

			// Description
			ui.Text(c, desc).
				Font(theme.FontSans).
				FontSize(theme.SizeBase).
				TextColor(theme.TextMuted).
				Grow(1)
		})
}

func renderResponsePanel(c *ui.Context, state *models.AppState) {
	ui.Column(c).
		Grow(1).
		Radius(8).
		Background(theme.SurfaceSubPlane).
		Border(1, theme.BorderSubtle).
		Padding(14).
		Gap(10).
		Children(func() {
			// Status & Telemetry Header
			ui.Row(c).
				AlignItems(ui.Center).
				Children(func() {
					primitives.StatusCodeBadge(c, 200, "200 OK")

					ui.Box(c).Width(12)

					ui.Text(c, "118 ms").
						Font(theme.FontMono).
						FontSize(theme.SizeBase).
						TextColor(theme.TextPrimary)

					ui.Text(c, " • ").TextColor(theme.TextDim)

					ui.Text(c, "1.34 KB").
						Font(theme.FontMono).
						FontSize(theme.SizeBase).
						TextColor(theme.TextSecondary)

					ui.Text(c, " • ").TextColor(theme.TextDim)

					ui.Row(c).
						PaddingX(6).
						PaddingY(2).
						Radius(4).
						Background(theme.SurfaceCardTop).
						Border(1, theme.BorderSubtle).
						Children(func() {
							ui.Text(c, "HTTP/2").
								Font(theme.FontMono).
								FontSize(theme.SizeCaption).
								TextColor(theme.RimFocus)
						})

					ui.Box(c).Grow(1)

					// JSON & Copy actions
					ui.Row(c).
						PaddingX(8).
						PaddingY(4).
						Radius(4).
						Background(theme.SurfaceActive).
						Border(1, theme.BorderSubtle).
						Children(func() {
							ui.Text(c, "JSON").Font(theme.FontSans).FontSize(theme.SizeCaption).Bold().TextColor(theme.TextPrimary)
						})

					ui.Box(c).Width(8)

					ui.Row(c).
						PaddingX(8).
						PaddingY(4).
						Radius(4).
						Background(theme.SurfaceCardTop).
						Border(1, theme.BorderSubtle).
						Children(func() {
							ui.Text(c, "Copy").Font(theme.FontSans).FontSize(theme.SizeCaption).TextColor(theme.TextMuted)
						})
				})

			// Response Body (Syntax-highlighted preview with line numbers)
			renderResponseBodyPreview(c)
		})
}

func renderResponseBodyPreview(c *ui.Context) {
	lines := []struct {
		num  string
		text string
		col  ui.Color
	}{
		{"1", "{", theme.TextDim},
		{"2", "  \"status\": \"success\",", theme.TextPrimary},
		{"3", "  \"user\": {", theme.TextPrimary},
		{"4", "    \"id\": \"usr_01HK789AB\",", theme.RimFocus},
		{"5", "    \"name\": \"Alex Mercer\",", theme.TextPrimary},
		{"6", "    \"email\": \"alex@company.internal\",", theme.TextPrimary},
		{"7", "    \"role\": \"lead_developer\",", theme.TextMuted},
		{"8", "    \"teams\": [\"platform-core\", \"api-infra\"]", theme.TextSecondary},
		{"9", "  }", theme.TextPrimary},
		{"10", "}", theme.TextDim},
	}

	ui.Row(c).
		Grow(1).
		Background(theme.CanvasDeep).
		Radius(6).
		Padding(10).
		Border(1, theme.BorderSubtle).
		Children(func() {
			// Line Numbers Column
			ui.Column(c).
				Width(28).
				Children(func() {
					for _, l := range lines {
						ui.Text(c, l.num).
							Font(theme.FontMono).
							FontSize(theme.SizeCaption).
							TextColor(theme.TextDim)
					}
				})

			ui.Box(c).Width(12)

			// JSON Body Column
			ui.Column(c).
				Grow(1).
				Children(func() {
					for _, l := range lines {
						ui.Text(c, l.text).
							Font(theme.FontMono).
							FontSize(theme.SizeCaption).
							TextColor(l.col)
					}
				})
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
		})
}
