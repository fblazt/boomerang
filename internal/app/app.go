package app

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/ui/catalog"
	"github.com/fblazt/boomerang/internal/ui/theme"
	"github.com/fblazt/boomerang/internal/ui/views"
)

// Options defines desktop application bootstrap options.
type Options struct {
	Title     string
	Width     int
	Height    int
	MinWidth  int
	MinHeight int
	Preview   string
}

// DefaultOptions returns standard 1440x900 native desktop options.
func DefaultOptions() Options {
	return Options{
		Title:     "Boomerang",
		Width:     1440,
		Height:    900,
		MinWidth:  1024,
		MinHeight: 640,
	}
}

// App encapsulates the desktop application instance and lifecycle.
type App struct {
	opts  Options
	state *AppState
	win   *mygo.Window
}

// NewApp creates a new application instance.
func NewApp(opts Options) *App {
	return &App{
		opts:  opts,
		state: NewAppState(nil),
	}
}

// State returns the application's thread-safe state.
func (a *App) State() *AppState {
	return a.state
}

// Window returns the native OS window instance.
func (a *App) Window() *mygo.Window {
	return a.win
}

// BuildWindowOptions returns the configured WindowOptions for mygo.
func (a *App) BuildWindowOptions() mygo.WindowOptions {
	return mygo.WindowOptions{
		Title:         a.opts.Title,
		Width:         a.opts.Width,
		Height:        a.opts.Height,
		MinWidth:      a.opts.MinWidth,
		MinHeight:     a.opts.MinHeight,
		TitleBarStyle: mygo.TitleBarHiddenInset,
		BackgroundColor: "#090A0F",
		Content:       ui.View(a.View),
	}
}

// View is the root immediate GPU renderer callback.
func (a *App) View(c *ui.Context) {
	// Apply Boomerang theme tokens
	c.SetTheme(theme.NewBoomerangTheme())

	// Bind reactive services for async background redraws
	a.state.BindServices(c.Services())

	// If preview mode is activated, display Component Catalog
	if a.opts.Preview == "components" {
		catalog.ComponentCatalogView(c)
		return
	}

	// Full three-pane application shell
	views.ShellView(c, a.state)
}

func (a *App) renderTitlebar(c *ui.Context) {
	ui.Row(c).
		Height(44).
		PaddingX(16).
		Background(theme.SurfaceSubPlane).
		BorderWidth(0, 0, 1, 0).
		BorderColor(theme.BorderSubtle).
		Children(func() {
			// Drag area / title offset for macOS traffic lights
			ui.Row(c).
				Width(70).
				Children(func() {})

			ui.Text(c, "Boomerang").
				Font(theme.FontSans).
				FontSize(theme.SizeHeading).
				Bold().
				TextColor(theme.TextPrimary)

			ui.Row(c).
				Grow(1).
				Children(func() {})

			// Environment badge indicator
			a.state.View(func(s *AppState) {
				ui.Row(c).
					PaddingX(8).
					PaddingY(4).
					Radius(4).
					Background(theme.SurfaceCardTop).
					Border(1, theme.BorderSubtle).
					Children(func() {
						ui.Text(c, s.ActiveEnv).
							Font(theme.FontMono).
							FontSize(theme.SizeBadge).
							TextColor(theme.RimFocus)
					})
			})
		})
}

func (a *App) renderWorkspace(c *ui.Context) {
	ui.Row(c).
		Grow(1).
		Background(theme.CanvasDeep).
		Children(func() {
			// Left Sidebar placeholder for Phase 1
			ui.Column(c).
				Width(260).
				Background(theme.SurfaceSubPlane).
				BorderWidth(0, 1, 0, 0).
				BorderColor(theme.BorderSubtle).
				Padding(16).
				Children(func() {
					ui.Text(c, "COLLECTIONS").
						Font(theme.FontSans).
						FontSize(theme.SizeBadge).
						TextColor(theme.TextDim).
						Bold()
				})

			// Main Center Stage
			ui.Column(c).
				Grow(1).
				Padding(24).
				Children(func() {
					a.state.View(func(s *AppState) {
						ui.Text(c, "Endpoint: "+s.ActiveRequest.URL).
							Font(theme.FontMono).
							FontSize(theme.SizeBody).
							TextColor(theme.TextSecondary)
					})
				})
		})
}

// Run configures the native lifecycle and starts the OS event loop.
func (a *App) Run() error {
	mygo.App.SetName(a.opts.Title)

	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(a.BuildWindowOptions())
	})

	if err := mygo.App.Run(); err != nil {
		log.Printf("app error: %v", err)
		return err
	}
	return nil
}
