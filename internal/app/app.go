package app

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/engine/env"
	enginehttp "github.com/fblazt/boomerang/internal/engine/http"
	"github.com/fblazt/boomerang/internal/models"
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
	st := NewAppState(nil)
	st.SetExecutor(enginehttp.NewClient(30 * time.Second))
	st.SetInterpolator(env.InterpolateRequest, env.Interpolate)

	// Initialize encrypted secret vault with disk persistence (or in-memory fallback)
	var vault *env.SecretVault
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		vaultPath := filepath.Join(home, ".boomerang", "vault.enc")
		vault, _ = env.NewSecretVault(vaultPath, "boomerang-vault-key")
	}
	if vault == nil {
		vault, _ = env.NewInMemorySecretVault("boomerang-vault-key")
	}

	eng := env.NewEngine(vault, models.DefaultEnvironments())
	st.SetEnvEngine(eng)

	app := &App{
		opts:  opts,
		state: st,
	}

	st.SetFilePicker(func(title string, extensions []string, onPicked func(path string)) {
		go func() {
			var filters []mygo.FileFilter
			if len(extensions) > 0 {
				filters = []mygo.FileFilter{{
					Name:       "Certificates",
					Extensions: extensions,
				}}
			}
			paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
				Parent:  app.win,
				Title:   title,
				Filters: filters,
			})
			if err != nil || len(paths) == 0 {
				return
			}
			onPicked(paths[0])
		}()
	})

	return app
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
