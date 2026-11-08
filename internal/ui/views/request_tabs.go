package views

import (
	"encoding/json"
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/primitives"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

// RequestTabsView renders the horizontal navigation tab bar and active configuration stage:
// Params, Headers, Body, Auth, and Settings.
func RequestTabsView(c *ui.Context, state *models.AppState) ui.Element {
	var activeTab string
	var paramsCount int
	var headersCount int
	var authType string
	var bodyType string

	state.View(func(s *models.AppState) {
		activeTab = s.ActiveTab
		if activeTab == "" {
			activeTab = "params"
		}
		paramsCount = len(s.ActiveRequest.Params)
		headersCount = len(s.ActiveRequest.Headers)
		authType = s.ActiveRequest.AuthType
		bodyType = s.ActiveRequest.BodyType
	})

	return ui.Column(c).
		Gap(12).
		Children(func() {
			// Horizontal Navigation Bar
			ui.Row(c).
				AlignItems(ui.Center).
				Gap(20).
				PaddingX(4).
				BorderWidth(0, 0, 1, 0).
				BorderColor(theme.BorderSubtle).
				Children(func() {
					renderTabButton(c, "Params", fmt.Sprintf("%d", paramsCount), activeTab == "params", func() {
						state.SetActiveTab("params")
					})

					renderTabButton(c, "Headers", fmt.Sprintf("%d", headersCount), activeTab == "headers", func() {
						state.SetActiveTab("headers")
					})

					bodyBadge := ""
					if bodyType != "" && bodyType != "none" {
						bodyBadge = bodyType
					}
					renderTabButton(c, "Body", bodyBadge, activeTab == "body", func() {
						state.SetActiveTab("body")
					})

					authBadge := ""
					if authType != "" && authType != "none" {
						authBadge = authType
					}
					renderTabButton(c, "Auth", authBadge, activeTab == "auth", func() {
						state.SetActiveTab("auth")
					})

					renderTabButton(c, "Settings", "", activeTab == "settings", func() {
						state.SetActiveTab("settings")
					})
				})

			// Tab Content Panel
			switch activeTab {
			case "headers":
				renderHeadersTab(c, state)
			case "body":
				renderBodyTab(c, state)
			case "auth":
				renderAuthTab(c, state)
			case "settings":
				renderSettingsTab(c, state)
			case "params":
				fallthrough
			default:
				renderParamsTab(c, state)
			}
		})
}

func renderTabButton(c *ui.Context, label, badge string, active bool, onClick func()) {
	borderBottom := float32(0)
	borderColor := theme.BorderSubtle
	textColor := theme.TextMuted
	bold := false

	if active {
		borderBottom = 2
		borderColor = theme.RimFocus
		textColor = theme.TextPrimary
		bold = true
	}

	tab := ui.Row(c).
		AlignItems(ui.Center).
		Gap(6).
		PaddingY(8).
		BorderWidth(0, 0, borderBottom, 0).
		BorderColor(borderColor).
		Focusable()

	if tab.Clicked() && onClick != nil {
		onClick()
	}

	tab.Children(func() {
		t := ui.Text(c, label).
			Font(theme.FontSans).
			FontSize(theme.SizeBody).
			TextColor(textColor)
		if bold {
			t.Bold()
		}

		if badge != "" {
			bg := theme.SurfaceSubPlane
			badgeColor := theme.TextDim
			if active {
				bg = theme.SurfaceActive
				badgeColor = theme.RimFocus
			}

			ui.Row(c).
				PaddingX(5).
				PaddingY(1).
				Radius(4).
				Background(bg).
				Border(1, theme.BorderSubtle).
				Children(func() {
					ui.Text(c, badge).
						Font(theme.FontMono).
						FontSize(theme.SizeCaption).
						Bold().
						TextColor(badgeColor)
				})
		}
	})
}

func renderParamsTab(c *ui.Context, state *models.AppState) {
	var params []models.KeyValuePair
	state.View(func(s *models.AppState) {
		params = make([]models.KeyValuePair, len(s.ActiveRequest.Params))
		copy(params, s.ActiveRequest.Params)
	})

	primitives.KVTable(c, primitives.KVTableProps{
		Items: &params,
		OnChange: func() {
			state.SetParams(params)
		},
		KeyPrompt:   "Parameter",
		ValuePrompt: "Value",
		DescPrompt:  "Description",
		AddLabel:    "+ Add Parameter",
	})
}

func renderHeadersTab(c *ui.Context, state *models.AppState) {
	var headers []models.KeyValuePair
	state.View(func(s *models.AppState) {
		headers = make([]models.KeyValuePair, len(s.ActiveRequest.Headers))
		copy(headers, s.ActiveRequest.Headers)
	})

	primitives.KVTable(c, primitives.KVTableProps{
		Items: &headers,
		OnChange: func() {
			state.SetHeaders(headers)
		},
		KeyPrompt:   "Header",
		ValuePrompt: "Value",
		DescPrompt:  "Description",
		Suggestions: primitives.StandardHeaderSuggestions,
		AddLabel:    "+ Add Header",
	})
}

func renderBodyTab(c *ui.Context, state *models.AppState) {
	var bodyType string
	var bodyText string
	var formData []models.KeyValuePair
	var urlEncoded []models.KeyValuePair

	state.View(func(s *models.AppState) {
		bodyType = s.ActiveRequest.BodyType
		if bodyType == "" {
			bodyType = "json"
		}
		bodyText = s.ActiveRequest.Body
		formData = make([]models.KeyValuePair, len(s.ActiveRequest.FormData))
		copy(formData, s.ActiveRequest.FormData)
		urlEncoded = make([]models.KeyValuePair, len(s.ActiveRequest.URLEncoded))
		copy(urlEncoded, s.ActiveRequest.URLEncoded)
	})

	ui.Column(c).
		Gap(10).
		Children(func() {
			// Body Format Selector Pills
			ui.Row(c).
				AlignItems(ui.Center).
				Gap(6).
				Children(func() {
					formats := []string{"none", "json", "text", "form-data", "urlencoded", "binary"}
					labels := []string{"None", "JSON", "Text", "Form-Data", "x-www-form-urlencoded", "Binary"}

					for i, fmtKey := range formats {
						lbl := labels[i]
						key := fmtKey
						active := (bodyType == key)

						bg := theme.SurfaceSubPlane
						txtCol := theme.TextDim
						borderCol := theme.BorderSubtle
						if active {
							bg = theme.SurfaceActive
							txtCol = theme.RimFocus
							borderCol = theme.RimFocus
						}

						pill := ui.Row(c).
							Height(24).
							PaddingX(8).
							Radius(4).
							Background(bg).
							Border(1, borderCol).
							Center().
							Focusable()

						if pill.Clicked() {
							state.Mutate(func(s *models.AppState) {
								s.ActiveRequest.BodyType = key
							})
						}

						pill.Children(func() {
							ui.Text(c, lbl).
								Font(theme.FontSans).
								FontSize(theme.SizeCaption).
								TextColor(txtCol)
						})
					}

					ui.Box(c).Grow(1)

					// Beautify JSON button if JSON mode
					if bodyType == "json" {
						beautifyBtn := ui.Row(c).
							Height(24).
							PaddingX(8).
							Radius(4).
							Background(theme.SurfaceCardTop).
							Border(1, theme.BorderSubtle).
							Center().
							Focusable()

						if beautifyBtn.Clicked() {
							var obj any
							if err := json.Unmarshal([]byte(bodyText), &obj); err == nil {
								if formatted, err := json.MarshalIndent(obj, "", "  "); err == nil {
									state.Mutate(func(s *models.AppState) {
										s.ActiveRequest.Body = string(formatted)
									})
								}
							}
						}

						beautifyBtn.Children(func() {
							ui.Text(c, "Beautify").
								Font(theme.FontSans).
								FontSize(theme.SizeCaption).
								TextColor(theme.TextSecondary)
						})
					}
				})

			// Editor by body type
			switch bodyType {
			case "none":
				ui.Row(c).
					Height(100).
					Background(theme.SurfaceSubPlane).
					Radius(6).
					Border(1, theme.BorderSubtle).
					Center().
					Children(func() {
						ui.Text(c, "This request has no body payload").
							Font(theme.FontSans).
							FontSize(theme.SizeBase).
							TextColor(theme.TextDim)
					})

			case "json", "text":
				ui.Column(c).
					Height(160).
					Background(theme.CanvasDeep).
					Radius(6).
					Padding(8).
					Border(1, theme.BorderSubtle).
					Children(func() {
						ui.TextArea(c, &state.ActiveRequest.Body).
							Font(theme.FontMono).
							FontSize(theme.SizeBase).
							TextColor(theme.TextPrimary).
							Grow(1)
					})

			case "form-data":
				primitives.KVTable(c, primitives.KVTableProps{
					Items: &formData,
					OnChange: func() {
						state.Mutate(func(s *models.AppState) {
							s.ActiveRequest.FormData = formData
						})
					},
					KeyPrompt:   "Field Name",
					ValuePrompt: "Field Value",
					DescPrompt:  "Description",
					AddLabel:    "+ Add Form Field",
				})

			case "urlencoded":
				primitives.KVTable(c, primitives.KVTableProps{
					Items: &urlEncoded,
					OnChange: func() {
						state.Mutate(func(s *models.AppState) {
							s.ActiveRequest.URLEncoded = urlEncoded
						})
					},
					KeyPrompt:   "Parameter",
					ValuePrompt: "Value",
					DescPrompt:  "Description",
					AddLabel:    "+ Add Encoded Param",
				})

			case "binary":
				ui.Row(c).
					Height(36).
					Background(theme.SurfaceCardTop).
					Radius(6).
					PaddingX(10).
					Border(1, theme.BorderSubtle).
					AlignItems(ui.Center).
					Children(func() {
						ui.Text(c, "File Path:").
							Font(theme.FontSans).
							FontSize(theme.SizeBase).
							TextColor(theme.TextDim).
							Width(80)

						ui.TextInput(c, &state.ActiveRequest.Body).
							Font(theme.FontMono).
							FontSize(theme.SizeBase).
							TextColor(theme.TextPrimary).
							Placeholder("/path/to/file.bin").
							Grow(1)
					})
			}
		})
}

func renderAuthTab(c *ui.Context, state *models.AppState) {
	var authType string
	var apiKeyIn string

	state.View(func(s *models.AppState) {
		authType = s.ActiveRequest.AuthType
		if authType == "" {
			authType = "none"
		}
		apiKeyIn = s.ActiveRequest.APIKeyIn
		if apiKeyIn == "" {
			apiKeyIn = "header"
		}
	})

	ui.Column(c).
		Gap(12).
		Children(func() {
			// Auth Type Selector
			ui.Row(c).
				AlignItems(ui.Center).
				Gap(6).
				Children(func() {
					types := []string{"none", "bearer", "basic", "apikey"}
					labels := []string{"No Auth", "Bearer Token", "Basic Auth", "API Key"}

					for i, aType := range types {
						key := aType
						lbl := labels[i]
						active := (authType == key)

						bg := theme.SurfaceSubPlane
						txtCol := theme.TextDim
						borderCol := theme.BorderSubtle
						if active {
							bg = theme.SurfaceActive
							txtCol = theme.RimFocus
							borderCol = theme.RimFocus
						}

						pill := ui.Row(c).
							Height(24).
							PaddingX(8).
							Radius(4).
							Background(bg).
							Border(1, borderCol).
							Center().
							Focusable()

						if pill.Clicked() {
							state.Mutate(func(s *models.AppState) {
								s.ActiveRequest.AuthType = key
							})
						}

						pill.Children(func() {
							ui.Text(c, lbl).
								Font(theme.FontSans).
								FontSize(theme.SizeCaption).
								TextColor(txtCol)
						})
					}
				})

			// Controls by Auth Type
			switch authType {
			case "none":
				ui.Row(c).
					Height(80).
					Background(theme.SurfaceSubPlane).
					Radius(6).
					Border(1, theme.BorderSubtle).
					Center().
					Children(func() {
						ui.Text(c, "No authentication credentials will be sent with this request").
							Font(theme.FontSans).
							FontSize(theme.SizeBase).
							TextColor(theme.TextDim)
					})

			case "bearer":
				ui.Column(c).
					Gap(6).
					Children(func() {
						ui.Text(c, "Bearer Token").
							Font(theme.FontSans).
							FontSize(theme.SizeCaption).
							Bold().
							TextColor(theme.TextDim)

						ui.Row(c).
							Height(34).
							Radius(6).
							Background(theme.SurfaceCardTop).
							Border(1, theme.BorderSubtle).
							PaddingX(10).
							AlignItems(ui.Center).
							Children(func() {
								primitives.VariableChip(c, "Bearer")
								ui.Box(c).Width(8)

								ui.TextInput(c, &state.ActiveRequest.Token).
									Font(theme.FontMono).
									FontSize(theme.SizeBase).
									TextColor(theme.RimFocus).
									Placeholder("eyJh... or {{jwt_token}}").
									Grow(1)
							})
					})

			case "basic":
				ui.Row(c).
					Gap(12).
					Children(func() {
						ui.Column(c).
							Grow(1).
							Gap(4).
							Children(func() {
								ui.Text(c, "Username").Font(theme.FontSans).FontSize(theme.SizeCaption).Bold().TextColor(theme.TextDim)
								ui.Row(c).
									Height(34).
									Radius(6).
									Background(theme.SurfaceCardTop).
									Border(1, theme.BorderSubtle).
									PaddingX(10).
									AlignItems(ui.Center).
									Children(func() {
										ui.TextInput(c, &state.ActiveRequest.BasicUser).
											Font(theme.FontMono).
											FontSize(theme.SizeBase).
											TextColor(theme.TextPrimary).
											Placeholder("username").
											Grow(1)
									})
							})

						ui.Column(c).
							Grow(1).
							Gap(4).
							Children(func() {
								ui.Text(c, "Password").Font(theme.FontSans).FontSize(theme.SizeCaption).Bold().TextColor(theme.TextDim)
								ui.Row(c).
									Height(34).
									Radius(6).
									Background(theme.SurfaceCardTop).
									Border(1, theme.BorderSubtle).
									PaddingX(10).
									AlignItems(ui.Center).
									Children(func() {
										ui.TextInput(c, &state.ActiveRequest.BasicPassword).
											Password().
											Font(theme.FontMono).
											FontSize(theme.SizeBase).
											TextColor(theme.TextPrimary).
											Placeholder("password").
											Grow(1)
									})
							})
					})

			case "apikey":
				ui.Column(c).
					Gap(8).
					Children(func() {
						ui.Row(c).
							Gap(12).
							Children(func() {
								ui.Column(c).
									Width(200).
									Gap(4).
									Children(func() {
										ui.Text(c, "Key").Font(theme.FontSans).FontSize(theme.SizeCaption).Bold().TextColor(theme.TextDim)
										ui.Row(c).
											Height(34).
											Radius(6).
											Background(theme.SurfaceCardTop).
											Border(1, theme.BorderSubtle).
											PaddingX(10).
											AlignItems(ui.Center).
											Children(func() {
												ui.TextInput(c, &state.ActiveRequest.APIKeyKey).
													Font(theme.FontMono).
													FontSize(theme.SizeBase).
													TextColor(theme.TextPrimary).
													Placeholder("X-API-Key").
													Grow(1)
											})
									})

								ui.Column(c).
									Grow(1).
									Gap(4).
									Children(func() {
										ui.Text(c, "Value").Font(theme.FontSans).FontSize(theme.SizeCaption).Bold().TextColor(theme.TextDim)
										ui.Row(c).
											Height(34).
											Radius(6).
											Background(theme.SurfaceCardTop).
											Border(1, theme.BorderSubtle).
											PaddingX(10).
											AlignItems(ui.Center).
											Children(func() {
												ui.TextInput(c, &state.ActiveRequest.APIKeyValue).
													Font(theme.FontMono).
													FontSize(theme.SizeBase).
													TextColor(theme.RimFocus).
													Placeholder("secret_key_123").
													Grow(1)
											})
									})
							})

						ui.Row(c).
							AlignItems(ui.Center).
							Gap(10).
							Children(func() {
								ui.Text(c, "Add to:").Font(theme.FontSans).FontSize(theme.SizeCaption).TextColor(theme.TextDim)

								renderRadioOption(c, "Header", apiKeyIn == "header", func() {
									state.Mutate(func(s *models.AppState) {
										s.ActiveRequest.APIKeyIn = "header"
									})
								})

								renderRadioOption(c, "Query Params", apiKeyIn == "query", func() {
									state.Mutate(func(s *models.AppState) {
										s.ActiveRequest.APIKeyIn = "query"
									})
								})
							})
					})
			}
		})
}

func renderRadioOption(c *ui.Context, label string, selected bool, onClick func()) {
	bg := theme.SurfaceSubPlane
	txtCol := theme.TextDim
	borderCol := theme.BorderSubtle
	if selected {
		bg = theme.SurfaceActive
		txtCol = theme.RimFocus
		borderCol = theme.RimFocus
	}

	pill := ui.Row(c).
		Height(22).
		PaddingX(8).
		Radius(4).
		Background(bg).
		Border(1, borderCol).
		Center().
		Focusable()

	if pill.Clicked() && onClick != nil {
		onClick()
	}

	pill.Children(func() {
		ui.Text(c, label).
			Font(theme.FontSans).
			FontSize(theme.SizeCaption).
			TextColor(txtCol)
	})
}

func renderSettingsTab(c *ui.Context, state *models.AppState) {
	ui.Column(c).
		Gap(8).
		Children(func() {
			renderSettingRow(c, "Strict SSL Verification", "Validate SSL certificates against system trust store", true)
			renderSettingRow(c, "Follow Redirects", "Automatically follow 3xx HTTP redirects", true)
			renderSettingRow(c, "Enable HTTP/2", "Use HTTP/2 multiplexing where available", true)
		})
}

func renderSettingRow(c *ui.Context, title, desc string, enabled bool) {
	ui.Row(c).
		Height(40).
		PaddingX(10).
		Radius(6).
		Background(theme.SurfaceCardTop).
		Border(1, theme.BorderSubtle).
		AlignItems(ui.Center).
		Children(func() {
			ui.Column(c).
				Grow(1).
				Children(func() {
					ui.Text(c, title).Font(theme.FontSans).FontSize(theme.SizeBase).TextColor(theme.TextPrimary)
					ui.Text(c, desc).Font(theme.FontSans).FontSize(theme.SizeCaption).TextColor(theme.TextDim)
				})

			ui.Row(c).
				PaddingX(8).
				PaddingY(3).
				Radius(4).
				Background(theme.SurfaceSubPlane).
				Border(1, theme.BorderSubtle).
				Children(func() {
					val := "OFF"
					col := theme.TextDim
					if enabled {
						val = "ON"
						col = theme.RimFocus
					}
					ui.Text(c, val).Font(theme.FontMono).FontSize(10).Bold().TextColor(col)
				})
		})
}
