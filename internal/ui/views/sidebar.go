package views

import (
	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/primitives"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

// SidebarView renders the left navigation sidebar with collections, drafts, and realtime streams.
func SidebarView(c *ui.Context, state *models.AppState) ui.Element {
	return ui.Column(c).
		Width(260).
		Background(theme.SurfaceSubPlane).
		BorderWidth(0, 1, 0, 0).
		BorderColor(theme.BorderSubtle).
		Padding(12).
		Gap(16).
		Children(func() {
			// Section 1: Collections Header
			ui.Row(c).
				AlignItems(ui.Center).
				Children(func() {
					ui.Text(c, "COLLECTIONS").
						Font(theme.FontSans).
						FontSize(theme.SizeBadge).
						TextColor(theme.TextDim).
						Bold()

					ui.Box(c).Grow(1)

					// New Request Icon Button
					primitives.IconButton(c, "+", "New Request", func() {
						state.Mutate(func(s *models.AppState) {
							s.ActiveRequest.Name = "Untitled Request"
							s.ActiveRequest.Method = "GET"
							s.ActiveRequest.URL = "https://api.internal.net/v1/health"
						})
					})
				})

			// Collection Items List
			ui.Column(c).
				Gap(4).
				Children(func() {
					renderCollectionItem(c, state, "Users Service", "GET", "https://api.internal.net/v1/users")
					renderCollectionItem(c, state, "Auth Verify", "POST", "https://api.internal.net/v1/auth/login")
					renderCollectionItem(c, state, "Delete Session", "DEL", "https://api.internal.net/v1/session")
					renderCollectionItem(c, state, "Product Feed", "GQL", "https://api.internal.net/graphql")
				})

			// Section 2: Realtime Streams
			ui.Row(c).
				AlignItems(ui.Center).
				Children(func() {
					ui.Text(c, "REALTIME STREAMS").
						Font(theme.FontSans).
						FontSize(theme.SizeBadge).
						TextColor(theme.TextDim).
						Bold()
				})

			ui.Column(c).
				Gap(4).
				Children(func() {
					renderStreamItem(c, state, "Order Live Feed", "WSS", "wss://stream.internal.net/orders")
					renderStreamItem(c, state, "Telemetry SSE", "SSE", "https://stream.internal.net/sse")
				})

			// Flexible Spacer
			ui.Box(c).Grow(1)

			// Section 3: History & Runner Quick Launch
			ui.Column(c).
				Gap(6).
				Children(func() {
					ui.Row(c).
						Height(28).
						PaddingX(8).
						Radius(4).
						AlignItems(ui.Center).
						Gap(8).
						Focusable().
						Children(func() {
							ui.Text(c, "⏱").Font(theme.FontSans).FontSize(theme.SizeBase)
							ui.Text(c, "Execution History").
								Font(theme.FontSans).
								FontSize(theme.SizeBase).
								TextColor(theme.TextSecondary)
							ui.Box(c).Grow(1)
							primitives.ShortcutCap(c, "⌘H")
						})

					ui.Row(c).
						Height(28).
						PaddingX(8).
						Radius(4).
						AlignItems(ui.Center).
						Gap(8).
						Focusable().
						Children(func() {
							ui.Text(c, "▶").Font(theme.FontSans).FontSize(theme.SizeBase)
							ui.Text(c, "Suite Runner").
								Font(theme.FontSans).
								FontSize(theme.SizeBase).
								TextColor(theme.TextSecondary)
							ui.Box(c).Grow(1)
							primitives.ShortcutCap(c, "⌘R")
						})
				})
		})
}

func renderCollectionItem(c *ui.Context, state *models.AppState, name, verb, url string) {
	isSelected := false
	state.View(func(s *models.AppState) {
		isSelected = (s.ActiveRequest.Name == name)
	})

	bg := theme.SurfaceSubPlane
	if isSelected {
		bg = theme.SurfaceActive
	}

	row := ui.Row(c).
		Height(28).
		PaddingX(8).
		Radius(5).
		Background(bg).
		AlignItems(ui.Center).
		Gap(8).
		Focusable()

	if row.Clicked() {
		state.Mutate(func(s *models.AppState) {
			s.ActiveRequest.Name = name
			s.ActiveRequest.Method = verb
			s.ActiveRequest.URL = url
		})
	}

	row.Children(func() {
		primitives.KeycapBadge(c, verb)
		ui.Text(c, name).
			Font(theme.FontSans).
			FontSize(theme.SizeBase).
			TextColor(theme.TextPrimary)
	})
}

func renderStreamItem(c *ui.Context, state *models.AppState, name, proto, url string) {
	row := ui.Row(c).
		Height(28).
		PaddingX(8).
		Radius(5).
		Background(theme.SurfaceSubPlane).
		AlignItems(ui.Center).
		Gap(8).
		Focusable()

	if row.Clicked() {
		state.Mutate(func(s *models.AppState) {
			s.ActiveRequest.Name = name
			s.ActiveRequest.Method = proto
			s.ActiveRequest.URL = url
		})
	}

	row.Children(func() {
		primitives.KeycapBadge(c, proto)
		ui.Text(c, name).
			Font(theme.FontSans).
			FontSize(theme.SizeBase).
			TextColor(theme.TextPrimary)
	})
}
