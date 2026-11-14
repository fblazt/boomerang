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
	borderColor := ui.Transparent
	textColor := theme.TextMuted

	if active {
		borderColor = theme.RimFocus
		textColor = theme.TextPrimary
	}

	tab := ui.Row(c).
		AlignItems(ui.Center).
		Gap(6).
		PaddingY(8).
		BorderWidth(0, 0, 2, 0).
		BorderColor(borderColor).
		Focusable()

	if tab.Clicked() && onClick != nil {
		onClick()
	}

	tab.Children(func() {
		// Use consistent Bold font weight for both active and inactive states
		// to prevent text width expansion and horizontal layout shift on toggle.
		ui.Text(c, label).
			Font(theme.FontSans).
			FontSize(theme.SizeBody).
			Bold().
			TextColor(textColor)

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
						ta := ui.TextArea(c, &bodyText).
							Font(theme.FontMono).
							FontSize(theme.SizeBase).
							TextColor(theme.TextPrimary).
							Grow(1)

						if ta.Changed() {
							state.Mutate(func(s *models.AppState) {
								s.ActiveRequest.Body = bodyText
							})
						}
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

						binInput := ui.TextInput(c, &bodyText).
							Font(theme.FontMono).
							FontSize(theme.SizeBase).
							TextColor(theme.TextPrimary).
							Placeholder("/path/to/file.bin").
							Grow(1)

						if binInput.Changed() {
							state.Mutate(func(s *models.AppState) {
								s.ActiveRequest.Body = bodyText
							})
						}
					})
			}
		})
}

func renderAuthTab(c *ui.Context, state *models.AppState) {
	var authType string
	var apiKeyIn string
	var showSecret bool
	var token string
	var basicUser string
	var basicPass string
	var apiKeyKey string
	var apiKeyValue string

	state.View(func(s *models.AppState) {
		authType = s.ActiveRequest.AuthType
		if authType == "" {
			authType = "none"
		}
		apiKeyIn = s.ActiveRequest.APIKeyIn
		if apiKeyIn == "" {
			apiKeyIn = "header"
		}
		showSecret = s.ShowAuthSecret
		token = s.ActiveRequest.Token
		basicUser = s.ActiveRequest.BasicUser
		basicPass = s.ActiveRequest.BasicPassword
		apiKeyKey = s.ActiveRequest.APIKeyKey
		apiKeyValue = s.ActiveRequest.APIKeyValue
	})

	renderSecretToggle := func() {
		showBtn := ui.Row(c).
			Height(20).
			PaddingX(6).
			Radius(4).
			Background(theme.SurfaceSubPlane).
			Border(1, theme.BorderSubtle).
			Center().
			Focusable()

		if showBtn.Clicked() {
			state.ToggleAuthSecret()
		}

		showLabel := "SHOW"
		if showSecret {
			showLabel = "HIDE"
		}
		showBtn.Children(func() {
			ui.Text(c, showLabel).
				Font(theme.FontMono).
				FontSize(9).
				Bold().
				TextColor(theme.RimFocus)
		})
	}

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
						ui.Row(c).
							AlignItems(ui.Center).
							Children(func() {
								ui.Text(c, "Bearer Token").
									Font(theme.FontSans).
									FontSize(theme.SizeCaption).
									Bold().
									TextColor(theme.TextDim)

								ui.Box(c).Grow(1)
								renderSecretToggle()
							})

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

								tokenInput := ui.TextInput(c, &token).
									Font(theme.FontMono).
									FontSize(theme.SizeBase).
									TextColor(theme.RimFocus).
									Placeholder("eyJh... or {{jwt_token}}").
									Grow(1)

								if !showSecret {
									tokenInput.Password()
								}

								if tokenInput.Changed() {
									state.Mutate(func(s *models.AppState) {
										s.ActiveRequest.Token = token
									})
								}
							})
					})

			case "basic":
				ui.Column(c).
					Gap(6).
					Children(func() {
						ui.Row(c).
							AlignItems(ui.Center).
							Children(func() {
								ui.Text(c, "Basic Authentication").
									Font(theme.FontSans).
									FontSize(theme.SizeCaption).
									Bold().
									TextColor(theme.TextDim)

								ui.Box(c).Grow(1)
								renderSecretToggle()
							})

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
												userInput := ui.TextInput(c, &basicUser).
													Font(theme.FontMono).
													FontSize(theme.SizeBase).
													TextColor(theme.TextPrimary).
													Placeholder("username").
													Grow(1)

												if userInput.Changed() {
													state.Mutate(func(s *models.AppState) {
														s.ActiveRequest.BasicUser = basicUser
													})
												}
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
												passInput := ui.TextInput(c, &basicPass).
													Font(theme.FontMono).
													FontSize(theme.SizeBase).
													TextColor(theme.TextPrimary).
													Placeholder("password").
													Grow(1)

												if !showSecret {
													passInput.Password()
												}

												if passInput.Changed() {
													state.Mutate(func(s *models.AppState) {
														s.ActiveRequest.BasicPassword = basicPass
													})
												}
											})
									})
							})
					})

			case "apikey":
				ui.Column(c).
					Gap(8).
					Children(func() {
						ui.Row(c).
							AlignItems(ui.Center).
							Children(func() {
								ui.Text(c, "API Key Credentials").
									Font(theme.FontSans).
									FontSize(theme.SizeCaption).
									Bold().
									TextColor(theme.TextDim)

								ui.Box(c).Grow(1)
								renderSecretToggle()
							})

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
												keyInput := ui.TextInput(c, &apiKeyKey).
													Font(theme.FontMono).
													FontSize(theme.SizeBase).
													TextColor(theme.TextPrimary).
													Placeholder("X-API-Key").
													Grow(1)

												if keyInput.Changed() {
													state.Mutate(func(s *models.AppState) {
														s.ActiveRequest.APIKeyKey = apiKeyKey
													})
												}
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
												valInput := ui.TextInput(c, &apiKeyValue).
													Font(theme.FontMono).
													FontSize(theme.SizeBase).
													TextColor(theme.RimFocus).
													Placeholder("secret_key_123").
													Grow(1)

												if !showSecret {
													valInput.Password()
												}

												if valInput.Changed() {
													state.Mutate(func(s *models.AppState) {
														s.ActiveRequest.APIKeyValue = apiKeyValue
													})
												}
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
	var envPolicy models.TransportPolicy
	var req models.RequestModel

	state.View(func(s *models.AppState) {
		req = s.ActiveRequest
		for _, env := range s.Environments {
			if env.Name == s.ActiveEnv {
				envPolicy = env.TransportPolicy
				break
			}
		}
	})

	effective := req.EffectiveTransportPolicy(&envPolicy)
	strictSSL := effective.StrictSSL
	followRedirects := req.EffectiveFollowRedirects()
	enableHTTP2 := effective.Protocol != "HTTP/1.1"
	hasOverrides := req.StrictSSL != nil || req.FollowRedirects != nil || req.EnableHTTP2 != nil

	ui.Column(c).
		Gap(8).
		Children(func() {
			renderSettingRow(c, "Strict SSL Verification", "Validate SSL certificates against trust store / pinned Root CA", strictSSL, req.StrictSSL != nil, func() {
				next := !strictSSL
				state.Mutate(func(s *models.AppState) {
					s.ActiveRequest.StrictSSL = &next
				})
			})
			renderSettingRow(c, "Follow Redirects", "Automatically follow 3xx HTTP redirects", followRedirects, req.FollowRedirects != nil, func() {
				next := !followRedirects
				state.Mutate(func(s *models.AppState) {
					s.ActiveRequest.FollowRedirects = &next
				})
			})
			renderSettingRow(c, "Enable HTTP/2", "Use HTTP/2 multiplexing where available (overrides to HTTP/1.1 when OFF)", enableHTTP2, req.EnableHTTP2 != nil, func() {
				next := !enableHTTP2
				state.Mutate(func(s *models.AppState) {
					s.ActiveRequest.EnableHTTP2 = &next
				})
			})

			if hasOverrides {
				ui.Row(c).
					Children(func() {
						resetBtn := ui.Row(c).
							Height(26).
							PaddingX(10).
							Radius(4).
							Background(theme.SurfaceSubPlane).
							Border(1, theme.BorderSubtle).
							AlignItems(ui.Center).
							Focusable()

						if resetBtn.Clicked() {
							state.Mutate(func(s *models.AppState) {
								s.ActiveRequest.StrictSSL = nil
								s.ActiveRequest.FollowRedirects = nil
								s.ActiveRequest.EnableHTTP2 = nil
							})
						}

						resetBtn.Children(func() {
							ui.Text(c, "Reset to Environment Defaults").
								Font(theme.FontSans).
								FontSize(theme.SizeCaption).
								TextColor(theme.RimFocus)
						})
					})
			}
		})
}

func renderSettingRow(c *ui.Context, title, desc string, enabled, overridden bool, onToggle func()) {
	row := ui.Row(c).
		Height(40).
		PaddingX(10).
		Radius(6).
		Background(theme.SurfaceCardTop).
		Border(1, theme.BorderSubtle).
		AlignItems(ui.Center).
		Focusable()

	if row.Clicked() && onToggle != nil {
		onToggle()
	}

	row.Children(func() {
		ui.Column(c).
			Grow(1).
			Children(func() {
				ui.Text(c, title).Font(theme.FontSans).FontSize(theme.SizeBase).TextColor(theme.TextPrimary)
				ui.Text(c, desc).Font(theme.FontSans).FontSize(theme.SizeCaption).TextColor(theme.TextDim)
			})

		ui.Row(c).
			Gap(6).
			AlignItems(ui.Center).
			Children(func() {
				if overridden {
					ui.Text(c, "OVERRIDE").
						Font(theme.FontMono).
						FontSize(8).
						Bold().
						TextColor(theme.TextDim)
				}

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
	})
}
