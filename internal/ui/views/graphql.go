package views

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/primitives"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

// GraphQLView renders the GraphQL Explorer workspace with endpoint toolbar,
// schema introspection documentation drawer, split query/variables panes, and telemetry response.
func GraphQLView(c *ui.Context, state *models.AppState) ui.Element {
	var (
		req            models.GraphQLRequest
		schema         *models.GraphQLSchema
		selectedType   string
		schemaSearch   string
		drawerOpen     bool
		introspecting  bool
		loading        bool
		resp           *models.GraphQLResponse
		gqlErr         string
	)
	state.View(func(s *models.AppState) {
		req = s.GQLRequest
		schema = s.GQLSchema
		selectedType = s.GQLSelectedType
		schemaSearch = s.GQLSchemaSearch
		drawerOpen = s.GQLSchemaDrawerOpen
		introspecting = s.GQLIntrospecting
		loading = s.GQLLoading
		resp = s.GQLResponse
		gqlErr = s.GQLError
	})

	if c.Shortcut(ui.Cmd, ui.KeyEnter) && !loading {
		state.ExecuteGraphQL()
	}

	return ui.Column(c).
		Grow(1).
		Padding(16).
		Gap(12).
		Background(theme.CanvasDeep).
		Children(func() {
			renderGraphQLToolbar(c, state, req, drawerOpen, introspecting, loading)

			if gqlErr != "" {
				ui.Row(c).
					Padding(8, 12, 8, 12).
					Radius(6).
					Background(ui.RGBA(0xEF, 0x44, 0x44, 0.12)).
					Border(1, theme.RimDanger).
					Children(func() {
						ui.Text(c, gqlErr).
							Font(theme.FontMono).
							FontSize(theme.SizeBase).
							TextColor(theme.RimDanger)
					})
			}

			ui.Row(c).
				Grow(1).
				Gap(12).
				AlignItems(ui.Stretch).
				Children(func() {
					// Workspace Document & Response placeholder (enhanced in Commits #29 & #30)
					primitives.ElevatedPlane(c, func() {
						ui.Column(c).
							Grow(1).
							Padding(14).
							Gap(10).
							Children(func() {
								ui.Text(c, "GraphQL Workspace").
									Font(theme.FontSans).
									FontSize(theme.SizeHeading).
									Bold().
									TextColor(theme.TextPrimary)
								ui.Text(c, req.Query).
									Font(theme.FontMono).
									FontSize(theme.SizeBase).
									TextColor(theme.TextSecondary)
								if resp != nil {
									ui.Text(c, resp.DataJSON).
										Font(theme.FontMono).
										FontSize(theme.SizeBase).
										TextColor(theme.TextPrimary)
								}
							})
					}).Grow(1)

					if drawerOpen {
						renderSchemaDrawer(c, state, schema, selectedType, schemaSearch, introspecting)
					}
				})
		})
}

func renderGraphQLToolbar(
	c *ui.Context,
	state *models.AppState,
	req models.GraphQLRequest,
	drawerOpen, introspecting, loading bool,
) {
	endpointDraft := req.Endpoint
	primitives.ElevatedPlane(c, func() {
		ui.Row(c).
			Height(48).
			PaddingX(10).
			Gap(10).
			AlignItems(ui.Center).
			Children(func() {
				primitives.KeycapBadge(c, "GQL")

				primitives.InputPlane(c, false, false, func() {
					in := ui.EditableText(c, &endpointDraft).
						Font(theme.FontMono).
						FontSize(theme.SizeBody).
						TextColor(theme.TextPrimary).
						Grow(1)
					if in.Changed() {
						state.SetGraphQLEndpoint(endpointDraft)
					}
					if in.Submitted() {
						state.IntrospectGraphQL()
					}
				}).Grow(1)

				syncLabel := "Introspect"
				if introspecting {
					syncLabel = "Syncing..."
				}
				renderStreamPillBtn(c, syncLabel, false, func() {
					state.IntrospectGraphQL()
				})

				renderStreamPillBtn(c, "Schema Docs", drawerOpen, func() {
					state.ToggleGraphQLSchemaDrawer()
				})

				execLabel := "Execute Query"
				if loading {
					execLabel = "Running..."
				}
				primitives.PrimaryButton(c, execLabel, func() {
					state.ExecuteGraphQL()
				})

				renderStreamPillBtn(c, "REST", false, func() {
					state.SetWorkspaceMode("rest")
				})
			})
	})
}

func renderSchemaDrawer(
	c *ui.Context,
	state *models.AppState,
	schema *models.GraphQLSchema,
	selectedType string,
	search string,
	introspecting bool,
) {
	searchDraft := search
	ui.Column(c).
		Width(340).
		Radius(8).
		Background(theme.SurfaceSubPlane).
		Border(1, theme.BorderMedium).
		Children(func() {
			// Drawer Header
			ui.Row(c).
				Height(40).
				PaddingX(12).
				Background(theme.SurfaceCardTop).
				BorderWidth(0, 0, 1, 0).
				BorderColor(theme.BorderSubtle).
				Radius(8, 8, 0, 0).
				AlignItems(ui.Center).
				Gap(8).
				Children(func() {
					ui.Text(c, "SCHEMA EXPLORER").
						Font(theme.FontMono).
						FontSize(theme.SizeBadge).
						Bold().
						TextColor(theme.TextSecondary)
					ui.Box(c).Grow(1)
					if schema != nil {
						ui.Text(c, fmt.Sprintf("%d types", len(schema.Types))).
							Font(theme.FontMono).
							FontSize(theme.SizeShortcut).
							TextColor(theme.TextDim)
					}
				})

			// Filter Box
			ui.Row(c).
				Padding(8, 10, 8, 10).
				BorderWidth(0, 0, 1, 0).
				BorderColor(theme.BorderSubtle).
				Children(func() {
					ui.Row(c).
						Height(28).
						PaddingX(8).
						Radius(5).
						Background(theme.CanvasDeep).
						Border(1, theme.BorderSubtle).
						AlignItems(ui.Center).
						Grow(1).
						Children(func() {
							in := ui.EditableText(c, &searchDraft).
								Placeholder("Search schema types & fields...").
								PlaceholderColor(theme.TextDim).
								Font(theme.FontSans).
								FontSize(theme.SizeBase).
								TextColor(theme.TextPrimary).
								Grow(1)
							if in.Changed() {
								state.SetGraphQLSchemaSearch(searchDraft)
							}
						})
				})

			if schema == nil {
				ui.Column(c).
					Grow(1).
					Padding(16).
					Center().
					Gap(8).
					Children(func() {
						msg := "Click Introspect to load endpoint schema."
						if introspecting {
							msg = "Running __schema introspection..."
						}
						ui.Text(c, msg).
							Font(theme.FontSans).
							FontSize(theme.SizeBase).
							TextColor(theme.TextMuted)
					})
				return
			}

			ui.Row(c).
				Grow(1).
				AlignItems(ui.Stretch).
				Children(func() {
					// Left Type Hierarchy List
					ui.Scroll(c).
						Width(136).
						Padding(8).
						Gap(4).
						BorderWidth(0, 1, 0, 0).
						BorderColor(theme.BorderSubtle).
						Children(func() {
							q := strings.ToLower(strings.TrimSpace(search))
							for _, t := range schema.Types {
								if q != "" && !typeMatchesSearch(t, q) {
									continue
								}
								typeItem := t
								isSel := typeItem.Name == selectedType
								bg := theme.SurfaceSubPlane
								if isSel {
									bg = theme.SurfaceActive
								}
								row := ui.Row(c).
									Height(26).
									PaddingX(6).
									Radius(4).
									Background(bg).
									AlignItems(ui.Center).
									Gap(6).
									Focusable()
								if isSel {
									row.Border(1, theme.BorderMedium)
								}
								if row.Clicked() {
									state.SelectGraphQLSchemaType(typeItem.Name)
								}
								row.Children(func() {
									ui.Text(c, shortKindTag(typeItem.Kind)).
										Font(theme.FontMono).
										FontSize(theme.SizeShortcut).
										TextColor(kindBadgeColor(typeItem.Kind))
									ui.Text(c, typeItem.Name).
										Font(theme.FontMono).
										FontSize(theme.SizeBadge).
										TextColor(theme.TextPrimary)
								})
							}
						})

					// Right Type Docs Detail Pane
					ui.Scroll(c).
						Grow(1).
						Padding(10).
						Gap(8).
						Children(func() {
							target, ok := schema.FindType(selectedType)
							if !ok {
								ui.Text(c, "Select a schema type to inspect fields and arguments.").
									Font(theme.FontSans).
									FontSize(theme.SizeBase).
									TextColor(theme.TextDim)
								return
							}
							renderSchemaTypeDetails(c, state, target)
						})
				})
		})
}

func renderSchemaTypeDetails(c *ui.Context, state *models.AppState, t models.GraphQLType) {
	ui.Row(c).
		AlignItems(ui.Center).
		Gap(6).
		Children(func() {
			ui.Text(c, string(t.Kind)).
				Font(theme.FontMono).
				FontSize(theme.SizeShortcut).
				Bold().
				TextColor(kindBadgeColor(t.Kind))
			ui.Text(c, t.Name).
				Font(theme.FontMono).
				FontSize(theme.SizeBody).
				Bold().
				TextColor(theme.TextPrimary)
		})

	if t.Description != "" {
		ui.Text(c, t.Description).
			Font(theme.FontSans).
			FontSize(theme.SizeBadge).
			TextColor(theme.TextMuted)
	}

	if len(t.Fields) > 0 {
		ui.Text(c, "FIELDS").
			Font(theme.FontMono).
			FontSize(theme.SizeShortcut).
			Bold().
			TextColor(theme.TextDim)

		for _, f := range t.Fields {
			field := f
			ui.Column(c).
				Padding(6, 8, 6, 8).
				Radius(5).
				Background(theme.SurfaceCardTop).
				Border(1, theme.BorderSubtle).
				Gap(3).
				Children(func() {
					ui.Row(c).
						AlignItems(ui.Center).
						Gap(6).
						Children(func() {
							ui.Text(c, field.Name).
								Font(theme.FontMono).
								FontSize(theme.SizeBadge).
								Bold().
								TextColor(theme.TextPrimary)

							ui.Box(c).Grow(1)

							retType := field.Type.Format()
							retBase := field.Type.NamedBase()
							typeLink := ui.Row(c).
								Padding(1, 5, 1, 5).
								Radius(4).
								Background(theme.CanvasDeep).
								Focusable()
							if typeLink.Clicked() && retBase != "" {
								state.SelectGraphQLSchemaType(retBase)
							}
							typeLink.Children(func() {
								ui.Text(c, retType).
									Font(theme.FontMono).
									FontSize(theme.SizeShortcut).
									TextColor(theme.RimFocus)
							})

							renderStreamPillBtn(c, "+ Insert", false, func() {
								state.InsertGraphQLFieldSnippet(field)
							})
						})

					if len(field.Args) > 0 {
						var argsText []string
						for _, arg := range field.Args {
							argsText = append(argsText, fmt.Sprintf("%s: %s", arg.Name, arg.Type.Format()))
						}
						ui.Text(c, "args("+strings.Join(argsText, ", ")+")").
							Font(theme.FontMono).
							FontSize(theme.SizeShortcut).
							TextColor(theme.TextMuted)
					}

					if field.Description != "" {
						ui.Text(c, field.Description).
							Font(theme.FontSans).
							FontSize(theme.SizeShortcut).
							TextColor(theme.TextDim)
					}
				})
		}
	}

	if len(t.InputFields) > 0 {
		ui.Text(c, "INPUT FIELDS").
			Font(theme.FontMono).
			FontSize(theme.SizeShortcut).
			Bold().
			TextColor(theme.TextDim)
		for _, inf := range t.InputFields {
			ui.Text(c, fmt.Sprintf("%s: %s", inf.Name, inf.Type.Format())).
				Font(theme.FontMono).
				FontSize(theme.SizeBadge).
				TextColor(theme.TextSecondary)
		}
	}

	if len(t.EnumValues) > 0 {
		ui.Text(c, "ENUM VALUES").
			Font(theme.FontMono).
			FontSize(theme.SizeShortcut).
			Bold().
			TextColor(theme.TextDim)
		for _, ev := range t.EnumValues {
			ui.Text(c, ev.Name).
				Font(theme.FontMono).
				FontSize(theme.SizeBadge).
				TextColor(ui.RGB(0xFB, 0xBF, 0x24))
		}
	}
}

func typeMatchesSearch(t models.GraphQLType, q string) bool {
	if strings.Contains(strings.ToLower(t.Name), q) || strings.Contains(strings.ToLower(t.Description), q) {
		return true
	}
	for _, f := range t.Fields {
		if strings.Contains(strings.ToLower(f.Name), q) {
			return true
		}
	}
	return false
}

func shortKindTag(kind models.GraphQLTypeKind) string {
	switch kind {
	case models.GQLKindObject:
		return "O"
	case models.GQLKindInputObject:
		return "I"
	case models.GQLKindEnum:
		return "E"
	case models.GQLKindScalar:
		return "S"
	case models.GQLKindInterface:
		return "IF"
	case models.GQLKindUnion:
		return "U"
	default:
		return "T"
	}
}

func kindBadgeColor(kind models.GraphQLTypeKind) ui.Color {
	switch kind {
	case models.GQLKindObject:
		return theme.RimFocus
	case models.GQLKindScalar:
		return ui.RGB(0x34, 0xD3, 0x99)
	case models.GQLKindEnum, models.GQLKindInputObject:
		return ui.RGB(0xFB, 0xBF, 0x24)
	default:
		return ui.RGB(0xA8, 0x55, 0xF7)
	}
}
