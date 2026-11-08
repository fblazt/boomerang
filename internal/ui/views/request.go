package views

import (
	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/primitives"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

// UrlBarView renders the elevated URL bar row with method selector badge,
// variable token chip, text input, active environment indicator, and Send/Cancel/Save actions.
func UrlBarView(c *ui.Context, state *models.AppState) ui.Element {
	return ui.Row(c).
		AlignItems(ui.Center).
		Gap(8).
		Children(func() {
			// Elevated URL Bar Container with Method inside and PROD chip (Brilliant Frame 304-308)
			primitives.UrlBarInputPlane(c, func() {
				var currentMethod string
				var currentURL string
				var activeEnv string

				state.View(func(s *models.AppState) {
					currentMethod = s.ActiveRequest.Method
					currentURL = s.ActiveRequest.URL
					activeEnv = s.ActiveEnv
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

				urlInput := ui.EditableText(c, &currentURL).
					Font(theme.FontMono).
					FontSize(theme.SizeBody).
					TextColor(theme.TextPrimary).
					Grow(1)

				if urlInput.Changed() {
					state.SetURL(currentURL)
				}

				// Environment indicator chip on the right inside input bar (Brilliant Frame 308)
				ui.Row(c).
					AlignItems(ui.Center).
					Gap(5).
					Children(func() {
						ui.Box(c).Size(6, 6).Radius(3).Background(theme.RimFocus)
						ui.Text(c, activeEnv).
							Font(theme.FontMono).
							FontSize(theme.SizeBadge).
							TextColor(theme.TextMuted)
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
