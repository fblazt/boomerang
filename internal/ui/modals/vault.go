package modals

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/engine/env"
	enginehttp "github.com/fblazt/boomerang/internal/engine/http"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/primitives"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

// VaultModalHost wraps the modal dialog in ui.DialogBase for clean, borderless popup dimming and escape dismissal.
func VaultModalHost(c *ui.Context, state *models.AppState) ui.Element {
	var showModal bool
	state.View(func(s *models.AppState) {
		showModal = s.ShowVaultModal
	})

	if !showModal {
		return ui.Element{}
	}

	dlg := ui.DialogBase(c, &showModal, func(backdrop ui.Element, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.70))
		VaultManagerModal(c, state)
	})

	if dlg.Dismissed() || !showModal || c.Shortcut(0, ui.KeyEscape) {
		state.CloseVaultModal()
	}

	return dlg
}

// VaultManagerModal renders the full Environment & Secret Vault Manager dialog strictly matching Brilliant Board 05.
func VaultManagerModal(c *ui.Context, state *models.AppState) ui.Element {
	var (
		currentActiveEnv string
		selectedVaultEnv string
		activeTab        string
		envs             []models.Environment
	)

	state.View(func(s *models.AppState) {
		currentActiveEnv = s.ActiveEnv
		selectedVaultEnv = s.ActiveVaultEnv
		activeTab = s.VaultModalTab
		envs = s.Environments
	})

	if selectedVaultEnv == "" && len(envs) > 0 {
		selectedVaultEnv = envs[0].Name
	}

	var currentEnvTarget *models.Environment
	for i := range envs {
		if envs[i].Name == selectedVaultEnv {
			currentEnvTarget = &envs[i]
			break
		}
	}
	if currentEnvTarget == nil && len(envs) > 0 {
		currentEnvTarget = &envs[0]
		selectedVaultEnv = envs[0].Name
	}

	return ui.Column(c).
		Width(920).
		Height(580).
		Radius(12).
		Clip().
		LinearGradient(ui.LinearGradient{
			From:  theme.SurfacePlaneTop,
			To:    theme.SurfacePlaneBottom,
			Angle: 180,
		}).
		Border(1, theme.BorderMedium).
		Shadow(0, 24, 60, 0, ui.RGBA(0, 0, 0, 0.80)).
		Shadow(0, 0, 0, 1, ui.RGBA(255, 255, 255, 0.08)).
		Children(func() {
			// 1. Modal Titlebar Header
			renderModalHeader(c, state)

			// 2. Main Two-Pane Split Layout
			ui.Row(c).
				Grow(1).
				AlignItems(ui.Stretch).
				Children(func() {
					// Left Pane: Environments List Sidebar
					renderEnvironmentsSidebar(c, state, envs, selectedVaultEnv, currentActiveEnv)

					// Right Pane: Active Environment Details & Variables Grid
					if currentEnvTarget != nil {
						renderEnvironmentDetailPane(c, state, *currentEnvTarget, currentActiveEnv, activeTab)
					}
				})
		})
}

func renderModalHeader(c *ui.Context, state *models.AppState) {
	ui.Row(c).
		Height(54).
		PaddingX(20).
		Background(theme.SurfaceSubPlane).
		BorderWidth(0, 0, 1, 0).
		BorderColor(theme.BorderSubtle).
		Radius(12, 12, 0, 0).
		AlignItems(ui.Center).
		Children(func() {
			ui.Column(c).
				Gap(2).
				Children(func() {
					ui.Text(c, "Workspace Environment & Secret Vault Manager").
						Font(theme.FontSans).
						FontSize(theme.SizeHeading).
						Bold().
						TextColor(theme.TextPrimary)

					ui.Text(c, "Encrypted secrets storage, variable scopes, and transport policies").
						Font(theme.FontSans).
						FontSize(10).
						TextColor(theme.TextMuted)
				})

			ui.Box(c).Grow(1)

			// Close Button (Esc cap)
			closeBtn := ui.Row(c).
				Height(28).
				PaddingX(8).
				Radius(6).
				Background(theme.SurfaceCardTop).
				Border(1, theme.BorderSubtle).
				AlignItems(ui.Center).
				Gap(6).
				Focusable()

			if closeBtn.Clicked() || c.Shortcut(0, ui.KeyEscape) {
				state.CloseVaultModal()
			}

			closeBtn.Children(func() {
				ui.Text(c, "Close").
					Font(theme.FontSans).
					FontSize(theme.SizeBadge).
					TextColor(theme.TextSecondary)
				primitives.ShortcutCap(c, "Esc")
			})
		})
}

func renderEnvironmentsSidebar(
	c *ui.Context,
	state *models.AppState,
	envs []models.Environment,
	selectedVaultEnv string,
	currentActiveEnv string,
) {
	ui.Column(c).
		Width(220).
		Background(theme.SurfaceSubPlane).
		BorderWidth(0, 1, 0, 0).
		BorderColor(theme.BorderSubtle).
		Radius(0, 0, 0, 12).
		Padding(12).
		Gap(6).
		Children(func() {
			ui.Row(c).
				PaddingX(6).
				PaddingY(4).
				Children(func() {
					ui.Text(c, "ENVIRONMENTS").
						Font(theme.FontMono).
						FontSize(9).
						Bold().
						TextColor(theme.TextDim)
				})

			// Environment Items
			ui.Column(c).
				Grow(1).
				Gap(4).
				Children(func() {
					for _, envItem := range envs {
						e := envItem
						isSelected := e.Name == selectedVaultEnv
						isCurrentWorkspace := e.Name == currentActiveEnv
						dotColor := theme.EnvironmentColor(e.Name)

						row := ui.Row(c).
							Height(32).
							PaddingX(8).
							Radius(6).
							AlignItems(ui.Center).
							Gap(8).
							Focusable()

						if isSelected {
							row.Background(theme.SurfaceActive).
								Border(1, theme.BorderMedium)
						}

						if row.Clicked() {
							state.SetActiveVaultEnv(e.Name)
						}

						row.Children(func() {
							primitives.StatusDot(c, dotColor)

							ui.Text(c, e.Name).
								Font(theme.FontMono).
								FontSize(theme.SizeBody).
								Bold().
								TextColor(theme.TextPrimary)

							ui.Box(c).Grow(1)

							if isCurrentWorkspace {
								ui.Text(c, "ACTIVE").
									Font(theme.FontMono).
									FontSize(8).
									Bold().
									TextColor(dotColor)
							} else {
								ui.Text(c, fmt.Sprintf("%d", len(e.Variables))).
									Font(theme.FontMono).
									FontSize(10).
									TextColor(theme.TextDim)
							}
						})
					}
				})

			// New Environment Action Button
			newEnvBtn := ui.Row(c).
				Height(30).
				PaddingX(8).
				Radius(6).
				Background(theme.SurfaceCardTop).
				Border(1, theme.BorderSubtle).
				AlignItems(ui.Center).
				Center().
				Gap(6).
				Focusable()

			if newEnvBtn.Clicked() {
				nextNum := len(envs) + 1
				newName := fmt.Sprintf("ENV-%d", nextNum)
				state.CreateEnvironment(newName, "#38BDF8")
			}

			newEnvBtn.Children(func() {
				ui.Text(c, "+ New Environment").
					Font(theme.FontSans).
					FontSize(theme.SizeBadge).
					TextColor(theme.TextSecondary)
			})
		})
}

func renderEnvironmentDetailPane(
	c *ui.Context,
	state *models.AppState,
	target models.Environment,
	currentActiveEnv string,
	activeTab string,
) {
	ui.Column(c).
		Grow(1).
		Padding(16).
		Gap(12).
		Children(func() {
			// Environment Header & Action Bar
			ui.Row(c).
				AlignItems(ui.Center).
				Gap(8).
				Children(func() {
					primitives.StatusDot(c, theme.EnvironmentColor(target.Name))

					ui.Text(c, target.Name).
						Font(theme.FontSans).
						FontSize(theme.SizeHeading).
						Bold().
						TextColor(theme.TextPrimary)

					ui.Box(c).Grow(1)

					// Set as Active Button
					if target.Name != currentActiveEnv {
						setActiveBtn := ui.Row(c).
							Height(26).
							PaddingX(8).
							Radius(5).
							Background(theme.SurfaceCardTop).
							Border(1, theme.BorderSubtle).
							AlignItems(ui.Center).
							Focusable()

						if setActiveBtn.Clicked() {
							state.SetActiveEnv(target.Name)
						}

						setActiveBtn.Children(func() {
							ui.Text(c, "Set as Active").
								Font(theme.FontSans).
								FontSize(theme.SizeBadge).
								TextColor(theme.RimFocus)
						})
					}

					// Export .env Button
					exportBtn := ui.Row(c).
						Height(26).
						PaddingX(8).
						Radius(5).
						Background(theme.SurfaceCardTop).
						Border(1, theme.BorderSubtle).
						AlignItems(ui.Center).
						Focusable()

					if exportBtn.Clicked() {
						if dotEnv, err := state.ExportDotEnv(target.Name, false); err == nil {
							c.WriteClipboard(dotEnv)
						}
					}

					exportBtn.Children(func() {
						ui.Text(c, "Export .env").
							Font(theme.FontSans).
							FontSize(theme.SizeBadge).
							TextColor(theme.TextSecondary)
					})

					// Import .env Button
					importBtn := ui.Row(c).
						Height(26).
						PaddingX(8).
						Radius(5).
						Background(theme.SurfaceCardTop).
						Border(1, theme.BorderSubtle).
						AlignItems(ui.Center).
						Focusable()

					if importBtn.Clicked() {
						clip := c.ReadClipboard()
						if strings.Contains(clip, "=") {
							_ = state.ImportDotEnv(target.Name, clip)
						}
					}

					importBtn.Children(func() {
						ui.Text(c, "Import .env").
							Font(theme.FontSans).
							FontSize(theme.SizeBadge).
							TextColor(theme.TextSecondary)
					})

					// Duplicate Button
					dupBtn := ui.Row(c).
						Height(26).
						PaddingX(8).
						Radius(5).
						Background(theme.SurfaceCardTop).
						Border(1, theme.BorderSubtle).
						AlignItems(ui.Center).
						Focusable()

					if dupBtn.Clicked() {
						state.DuplicateEnvironment(target.Name, target.Name+"-COPY")
					}

					dupBtn.Children(func() {
						ui.Text(c, "Duplicate").
							Font(theme.FontSans).
							FontSize(theme.SizeBadge).
							TextColor(theme.TextSecondary)
					})

					// Delete Button (disabled for PROD)
					if target.Name != "PROD" {
						delBtn := ui.Row(c).
							Height(26).
							PaddingX(8).
							Radius(5).
							Background(theme.SurfaceCardTop).
							Border(1, theme.RimDangerGlow).
							AlignItems(ui.Center).
							Focusable()

						if delBtn.Clicked() {
							state.DeleteEnvironment(target.Name)
						}

						delBtn.Children(func() {
							ui.Text(c, "Delete").
								Font(theme.FontSans).
								FontSize(theme.SizeBadge).
								TextColor(theme.RimDanger)
						})
					}
				})

			// Sub-tabs: Variables vs Transport Policy
			ui.Row(c).
				Height(30).
				BorderWidth(0, 0, 1, 0).
				BorderColor(theme.BorderSubtle).
				Gap(16).
				Children(func() {
					isVarActive := activeTab == "variables" || activeTab == ""
					varTextColor := theme.TextSecondary
					if isVarActive {
						varTextColor = theme.TextPrimary
					}

					tabVar := ui.Row(c).
						Height(30).
						AlignItems(ui.Center).
						Focusable()
					if isVarActive {
						tabVar.BorderWidth(0, 0, 2, 0).BorderColor(theme.RimFocus)
					}
					if tabVar.Clicked() {
						state.SetVaultModalTab("variables")
					}
					tabVar.Children(func() {
						ui.Text(c, fmt.Sprintf("Variables (%d)", len(target.Variables))).
							Font(theme.FontSans).
							FontSize(theme.SizeBody).
							Bold().
							TextColor(varTextColor)
					})

					isTransportActive := activeTab == "transport"
					transportTextColor := theme.TextSecondary
					if isTransportActive {
						transportTextColor = theme.TextPrimary
					}

					tabTransport := ui.Row(c).
						Height(30).
						AlignItems(ui.Center).
						Focusable()
					if isTransportActive {
						tabTransport.BorderWidth(0, 0, 2, 0).BorderColor(theme.RimFocus)
					}
					if tabTransport.Clicked() {
						state.SetVaultModalTab("transport")
					}
					tabTransport.Children(func() {
						ui.Text(c, "Transport Policy").
							Font(theme.FontSans).
							FontSize(theme.SizeBody).
							Bold().
							TextColor(transportTextColor)
					})
				})

			// Tab Content Area
			if activeTab == "transport" {
				renderTransportPolicyTab(c, state, target)
			} else {
				renderVariablesGridTab(c, state, target)
			}
		})
}

func renderVariablesGridTab(c *ui.Context, state *models.AppState, target models.Environment) {
	ui.Column(c).
		Grow(1).
		Gap(8).
		Children(func() {
			// Table Header strictly following Brilliant Board 05:
			// Variable name, Initial value, Current value, Secret toggle, Scope.
			ui.Row(c).
				Height(26).
				PaddingX(8).
				Background(theme.SurfaceSubPlane).
				Border(1, theme.BorderSubtle).
				Radius(4).
				AlignItems(ui.Center).
				Gap(6).
				Children(func() {
					ui.Box(c).Width(16) // Checkbox spacer (14 + 2)
					ui.Text(c, "VARIABLE NAME").Font(theme.FontMono).FontSize(9).Bold().TextColor(theme.TextDim).Width(136)
					ui.Text(c, "INITIAL VALUE").Font(theme.FontMono).FontSize(9).Bold().TextColor(theme.TextDim).Width(126)
					ui.Text(c, "CURRENT VALUE").Font(theme.FontMono).FontSize(9).Bold().TextColor(theme.TextDim).Grow(1)
					ui.Text(c, "TYPE").Font(theme.FontMono).FontSize(9).Bold().TextColor(theme.TextDim).Width(58)
					ui.Text(c, "SECRET").Font(theme.FontMono).FontSize(9).Bold().TextColor(theme.TextDim).Width(44)
					ui.Text(c, "SCOPE").Font(theme.FontMono).FontSize(9).Bold().TextColor(theme.TextDim).Width(34)
					ui.Box(c).Width(20) // Delete button spacer
				})

			// Variable Rows
			ui.Column(c).
				Grow(1).
				Gap(4).
				Children(func() {
					for _, variableItem := range target.Variables {
						v := variableItem
						isSecret := v.Type == models.VariableTypeSecret
						isRevealed := state.IsSecretRevealedForEnv(target.Name, v.Key)

						row := ui.Row(c).
							Height(32).
							PaddingX(8).
							Radius(6).
							Border(1, theme.BorderSubtle).
							AlignItems(ui.Center).
							Gap(6)

						if v.Enabled {
							row.Background(theme.SurfaceCardTop)
						} else {
							row.Background(ui.RGB(0x12, 0x14, 0x1C))
						}

						row.Children(func() {
							// 1. Enabled Checkbox
							chk := ui.Row(c).
								Width(14).
								Height(14).
								Radius(3).
								Center().
								Focusable()

							if v.Enabled {
								chk.Background(theme.RimFocus)
							} else {
								chk.Border(1, theme.BorderSubtle)
							}

							if chk.Clicked() {
								updated := v
								updated.Enabled = !updated.Enabled
								state.UpdateVariableInEnv(target.Name, v.Key, updated)
							}

							chk.Children(func() {
								if v.Enabled {
									ui.Text(c, "✓").
										Font(theme.FontSans).
										FontSize(9).
										Bold().
										TextColor(ui.RGB(0xFF, 0xFF, 0xFF))
								}
							})

							ui.Box(c).Width(2)

							// 2. Variable Key (Editable Text)
							keyVal := v.Key
							keyEl := ui.EditableText(c, &keyVal).
								Font(theme.FontMono).
								FontSize(theme.SizeBadge).
								TextColor(theme.RimFocus).
								Width(136)

							if keyEl.Changed() && keyVal != v.Key && keyVal != "" {
								updated := v
								updated.Key = keyVal
								state.UpdateVariableInEnv(target.Name, v.Key, updated)
							}

							// 3. Initial Value (Editable Text)
							initVal := v.InitialValue
							initEl := ui.EditableText(c, &initVal).
								Font(theme.FontMono).
								FontSize(theme.SizeBadge).
								TextColor(theme.TextSecondary).
								Width(126)

							if initEl.Changed() && initVal != v.InitialValue {
								updated := v
								updated.InitialValue = initVal
								state.UpdateVariableInEnv(target.Name, v.Key, updated)
							}

							// 4. Current Value (Editable Text, Masked when hidden)
							currentVal := v.CurrentValue
							if isSecret {
								if dec, ok := state.GetSecretPlaintext(target.Name, v.Key); ok && dec != "" {
									currentVal = dec
								}
							}
							if currentVal == "" {
								currentVal = v.InitialValue
							}

							displayVal := currentVal
							if isSecret && !isRevealed {
								displayVal = env.MaskSecret(displayVal)
							}

							valEl := ui.EditableText(c, &displayVal).
								Font(theme.FontMono).
								FontSize(theme.SizeBadge).
								TextColor(theme.TextPrimary).
								Grow(1)
							if isSecret && !isRevealed {
								valEl.TextColor(theme.TextDim)
							}

							if valEl.Changed() && displayVal != "" {
								updated := v
								if isSecret {
									if isRevealed {
										updated.CurrentValue = displayVal
										state.UpdateVariableInEnvWithSecret(target.Name, v.Key, updated, displayVal)
									}
								} else {
									updated.CurrentValue = displayVal
									state.UpdateVariableInEnv(target.Name, v.Key, updated)
								}
							}

							// 5. Type Badge (TEXT / SECRET toggle)
							typeBtn := ui.Row(c).
								Width(58).
								Height(20).
								Radius(3).
								Center().
								Focusable()

							if isSecret {
								typeBtn.Background(ui.RGBA(0xF5, 0x9E, 0x0B, 0.18)).
									Border(1, ui.RGBA(0xF5, 0x9E, 0x0B, 0.40))
							} else {
								typeBtn.Background(ui.RGBA(0x38, 0xBD, 0xF8, 0.14)).
									Border(1, ui.RGBA(0x38, 0xBD, 0xF8, 0.30))
							}

							if typeBtn.Clicked() {
								updated := v
								if isSecret {
									updated.Type = models.VariableTypeText
									if dec, ok := state.GetSecretPlaintext(target.Name, v.Key); ok && dec != "" {
										updated.CurrentValue = dec
									}
								} else {
									updated.Type = models.VariableTypeSecret
								}
								state.UpdateVariableInEnv(target.Name, v.Key, updated)
							}

							typeBtn.Children(func() {
								typeText := "TEXT"
								textColor := theme.RimFocus
								if isSecret {
									typeText = "SECRET"
									textColor = ui.RGB(0xF5, 0x9E, 0x0B)
								}
								ui.Text(c, typeText).
									Font(theme.FontMono).
									FontSize(8).
									Bold().
									TextColor(textColor)
							})

							// 6. Reveal / Mask Action Button for Secrets
							if isSecret {
								revealBtn := ui.Row(c).
									Width(44).
									Height(20).
									Radius(3).
									Background(theme.SurfaceSubPlane).
									Border(1, theme.BorderSubtle).
									Center().
									Focusable()

								if revealBtn.Clicked() {
									state.ToggleSecretRevealForEnv(target.Name, v.Key)
								}

								revealBtn.Children(func() {
									btnLabel := "SHOW"
									if isRevealed {
										btnLabel = "HIDE"
									}
									ui.Text(c, btnLabel).
										Font(theme.FontMono).
										FontSize(8).
										Bold().
										TextColor(theme.TextMuted)
								})
							} else {
								ui.Box(c).Width(44)
							}

							// 7. Scope indicator
							ui.Row(c).
								Width(34).
								Height(18).
								Radius(3).
								Center().
								Background(ui.RGBA(255, 255, 255, 0.05)).
								Children(func() {
									ui.Text(c, "Env").
										Font(theme.FontMono).
										FontSize(8).
										TextColor(theme.TextDim)
								})

							// 8. Delete Variable Row Button
							delRowBtn := ui.Row(c).
								Width(20).
								Height(20).
								Radius(3).
								Center().
								Focusable()

							if delRowBtn.Clicked() {
								state.DeleteVariableFromEnv(target.Name, v.Key)
							}

							delRowBtn.Children(func() {
								ui.Text(c, "✕").
									Font(theme.FontSans).
									FontSize(10).
									TextColor(theme.TextDim)
							})
						})
					}
				})

			// Add Variable Action Button (clean tactile button, no empty full-width row)
			ui.Row(c).
				PaddingX(2).
				Children(func() {
					addBtn := ui.Row(c).
						Height(24).
						PaddingX(10).
						Radius(4).
						Background(theme.SurfaceSubPlane).
						Border(1, theme.BorderSubtle).
						AlignItems(ui.Center).
						Focusable()

					if addBtn.Clicked() {
						nextKey := fmt.Sprintf("VAR_%d", len(target.Variables)+1)
						state.AddVariableToEnv(target.Name, models.Variable{
							Key:          nextKey,
							InitialValue: "value",
							CurrentValue: "value",
							Type:         models.VariableTypeText,
							Enabled:      true,
						})
					}

					addBtn.Children(func() {
						ui.Text(c, "+ Add Variable").
							Font(theme.FontSans).
							FontSize(theme.SizeBadge).
							TextColor(theme.RimFocus)
					})
				})
		})
}

func renderTransportPolicyTab(c *ui.Context, state *models.AppState, target models.Environment) {
	policy := target.TransportPolicy
	if policy.Protocol == "" {
		policy.Protocol = "HTTP/2"
	}

	ui.Column(c).
		Grow(1).
		Gap(14).
		Children(func() {
			// Protocol Radio Selector
			ui.Column(c).
				Gap(6).
				Children(func() {
					ui.Text(c, "HTTP PROTOCOL & MULTIPLEXING").
						Font(theme.FontMono).
						FontSize(9).
						Bold().
						TextColor(theme.TextDim)

					ui.Row(c).
						Gap(8).
						Children(func() {
							protocols := []string{"HTTP/2", "HTTP/3", "HTTP/1.1"}
							for _, proto := range protocols {
								pr := proto
								isSelected := policy.Protocol == pr
								btn := ui.Row(c).
									Height(28).
									PaddingX(12).
									Radius(5).
									AlignItems(ui.Center).
									Gap(6).
									Focusable()

								if isSelected {
									btn.Background(theme.SurfaceActive).
										Border(1, theme.RimFocus)
								} else {
									btn.Background(theme.SurfaceCardTop).
										Border(1, theme.BorderSubtle)
								}

								if btn.Clicked() {
									updated := policy
									updated.Protocol = pr
									state.UpdateTransportPolicy(target.Name, updated)
								}

								btn.Children(func() {
									dotColor := theme.TextDim
									if isSelected {
										dotColor = theme.RimFocus
									}
									primitives.StatusDot(c, dotColor)

									label := pr
									if pr == "HTTP/2" {
										label = "HTTP/2 Multiplexed"
									} else if pr == "HTTP/3" {
										label = "HTTP/3 (QUIC UDP 0-RTT)"
									}
									ui.Text(c, label).
										Font(theme.FontSans).
										FontSize(theme.SizeBadge).
										TextColor(theme.TextPrimary)
								})
							}
						})
				})

			// Security Toggles
			ui.Column(c).
				Gap(6).
				Children(func() {
					ui.Row(c).
						AlignItems(ui.Center).
						Gap(8).
						Children(func() {
							ui.Text(c, "SSL & CERTIFICATE VERIFICATION").
								Font(theme.FontMono).
								FontSize(9).
								Bold().
								TextColor(theme.TextDim)

							if policy.ClientCertPEM != "" {
								ui.Row(c).
									Height(18).
									PaddingX(6).
									Radius(3).
									Background(ui.RGB(0x0E, 0x26, 0x20)).
									Border(1, ui.RGB(0x1E, 0x52, 0x43)).
									AlignItems(ui.Center).
									Children(func() {
										ui.Text(c, "mTLS ENFORCED").
											Font(theme.FontMono).
											FontSize(8).
											Bold().
											TextColor(ui.RGB(0x34, 0xD3, 0x99))
									})
							}
						})

					sslToggle := ui.Row(c).
						Height(28).
						PaddingX(10).
						Radius(5).
						Background(theme.SurfaceCardTop).
						Border(1, theme.BorderSubtle).
						AlignItems(ui.Center).
						Gap(8).
						Focusable()

					if sslToggle.Clicked() {
						updated := policy
						updated.StrictSSL = !updated.StrictSSL
						state.UpdateTransportPolicy(target.Name, updated)
					}

					sslToggle.Children(func() {
						status := "OFF (Insecure)"
						statusColor := theme.TextDim
						if policy.StrictSSL {
							status = "ON (Strict Verification)"
							statusColor = ui.RGB(0x34, 0xD3, 0x99)
						}

						ui.Text(c, "Strict SSL Verification:").
							Font(theme.FontSans).
							FontSize(theme.SizeBadge).
							TextColor(theme.TextPrimary)

						ui.Text(c, status).
							Font(theme.FontMono).
							FontSize(theme.SizeBadge).
							Bold().
							TextColor(statusColor)
					})
				})

			// Pinned Root CA Certificate
			caBadge := ""
			caBadgeColor := ui.RGB(0x34, 0xD3, 0x99)
			if policy.PinnedRootCA != "" {
				if summary, err := enginehttp.InspectRootCA(policy.PinnedRootCA); err == nil {
					caBadge = summary
				} else {
					caBadge = "INVALID PEM"
					caBadgeColor = theme.RimDanger
				}
			}
			renderCertPickerRow(
				c,
				"PINNED ROOT CA (.PEM / .CRT)",
				policy.PinnedRootCA,
				"Select Root CA certificate (click to browse PEM/CRT)",
				caBadge,
				caBadgeColor,
				func() {
					state.PickFile("Select Pinned Root CA", []string{"pem", "crt", "cer"}, func(path string) {
						updated := policy
						updated.PinnedRootCA = path
						state.UpdateTransportPolicy(target.Name, updated)
					})
				},
				func() {
					updated := policy
					updated.PinnedRootCA = ""
					state.UpdateTransportPolicy(target.Name, updated)
				},
			)

			// mTLS Client Identity
			renderCertPickerRow(
				c,
				"mTLS CLIENT IDENTITY (.PEM / .P12)",
				policy.ClientCertPEM,
				"No client certificate configured (click to select PEM/P12)",
				"",
				theme.TextDim,
				func() {
					state.PickFile("Select mTLS Client Certificate", []string{"pem", "crt", "p12", "pfx"}, func(path string) {
						updated := policy
						updated.ClientCertPEM = path
						ext := strings.ToLower(filepath.Ext(path))
						if ext == ".p12" || ext == ".pfx" {
							updated.ClientKeyPEM = ""
						}
						state.UpdateTransportPolicy(target.Name, updated)
					})
				},
				func() {
					updated := policy
					updated.ClientCertPEM = ""
					updated.ClientKeyPEM = ""
					state.UpdateTransportPolicy(target.Name, updated)
				},
			)

			// Optional separate mTLS Private Key for non-PKCS12 PEM certificates
			certExt := strings.ToLower(filepath.Ext(policy.ClientCertPEM))
			if policy.ClientCertPEM != "" && certExt != ".p12" && certExt != ".pfx" {
				renderCertPickerRow(
					c,
					"mTLS CLIENT PRIVATE KEY (.KEY / .PEM)",
					policy.ClientKeyPEM,
					"Optional separate private key (click to browse KEY/PEM)",
					"",
					theme.TextDim,
					func() {
						state.PickFile("Select mTLS Private Key", []string{"key", "pem"}, func(path string) {
							updated := policy
							updated.ClientKeyPEM = path
							state.UpdateTransportPolicy(target.Name, updated)
						})
					},
					func() {
						updated := policy
						updated.ClientKeyPEM = ""
						state.UpdateTransportPolicy(target.Name, updated)
					},
				)
			}
		})
}

func renderCertPickerRow(
	c *ui.Context,
	heading string,
	selectedPath string,
	placeholder string,
	badge string,
	badgeColor ui.Color,
	onPick func(),
	onClear func(),
) {
	ui.Column(c).
		Gap(6).
		Children(func() {
			ui.Text(c, heading).
				Font(theme.FontMono).
				FontSize(9).
				Bold().
				TextColor(theme.TextDim)

			ui.Row(c).
				Gap(6).
				AlignItems(ui.Center).
				Children(func() {
					pickBtn := ui.Row(c).
						Grow(1).
						Height(32).
						PaddingX(10).
						Radius(5).
						Background(theme.SurfaceCardTop).
						Border(1, theme.BorderSubtle).
						AlignItems(ui.Center).
						Gap(8).
						Focusable()

					if pickBtn.Clicked() && onPick != nil {
						onPick()
					}

					pickBtn.Children(func() {
						displayText := placeholder
						textColor := theme.TextMuted
						if selectedPath != "" {
							displayText = filepath.Base(selectedPath)
							textColor = theme.TextPrimary
						}
						ui.Text(c, displayText).
							Font(theme.FontMono).
							FontSize(theme.SizeBadge).
							TextColor(textColor)

						ui.Spacer(c)

						if badge != "" {
							ui.Row(c).
								Height(18).
								PaddingX(6).
								Radius(3).
								Background(theme.SurfaceSubPlane).
								Border(1, theme.BorderSubtle).
								AlignItems(ui.Center).
								Children(func() {
									ui.Text(c, badge).
										Font(theme.FontMono).
										FontSize(8).
										Bold().
										TextColor(badgeColor)
								})
						}
					})

					if selectedPath != "" {
						clearBtn := ui.Row(c).
							Height(32).
							PaddingX(10).
							Radius(5).
							Background(theme.SurfaceCardTop).
							Border(1, theme.BorderSubtle).
							AlignItems(ui.Center).
							Focusable()

						if clearBtn.Clicked() && onClear != nil {
							onClear()
						}

						clearBtn.Children(func() {
							ui.Text(c, "Clear").
								Font(theme.FontSans).
								FontSize(theme.SizeBadge).
								TextColor(theme.TextMuted)
						})
					}
				})
		})
}
