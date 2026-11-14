package app

import (
	"log"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/engine/collection"
	"github.com/fblazt/boomerang/internal/engine/env"
	"github.com/fblazt/boomerang/internal/engine/graphql"
	enginehttp "github.com/fblazt/boomerang/internal/engine/http"
	"github.com/fblazt/boomerang/internal/engine/realtime"
	"github.com/fblazt/boomerang/internal/engine/runner"
	"github.com/fblazt/boomerang/internal/engine/storage"
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
	DataDir   string // Optional workspace persistence directory; defaults to ~/.boomerang
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
	db    *storage.SQLiteStore
}

// NewApp creates a new application instance.
func NewApp(opts Options) *App {
	st := NewAppState(nil)
	st.SetExecutor(enginehttp.NewClient(30 * time.Second))
	st.SetInterpolator(env.InterpolateRequest, env.Interpolate)

	dataDir := opts.DataDir
	if dataDir == "" {
		if dir, err := storage.DefaultDataDir(); err == nil {
			dataDir = dir
		}
	}

	// Initialize encrypted secret vault with disk persistence (or in-memory fallback)
	var vault *env.SecretVault
	if dataDir != "" {
		vaultPath := filepath.Join(dataDir, "vault.enc")
		vault, _ = env.NewSecretVault(vaultPath, "boomerang-vault-key")
	}
	if vault == nil {
		vault, _ = env.NewInMemorySecretVault("boomerang-vault-key")
	}

	// Initialize SQLite history & scratchpad database (or in-memory fallback)
	var dbStore *storage.SQLiteStore
	if dataDir != "" {
		dbStore, _ = storage.OpenSQLite(filepath.Join(dataDir, "boomerang.db"))
	}
	if dbStore == nil {
		dbStore, _ = storage.OpenSQLite(":memory:")
	}
	if dbStore != nil {
		st.SetHistoryStore(dbStore)
		st.SetDraftStore(dbStore)
	}

	// Initialize local filesystem collections store (or in-memory fallback)
	var colStore *collection.Store
	if dataDir != "" {
		colStore, _ = collection.NewStore(filepath.Join(dataDir, "collections"))
	}
	if colStore == nil {
		colStore = collection.NewInMemoryStore()
	}
	_ = colStore.SeedDefaultsIfEmpty()
	st.SetCollectionStore(colStore)

	eng := env.NewEngine(vault, models.DefaultEnvironments())
	st.SetEnvEngine(eng)
	st.SetStreamController(realtime.NewManager())
	st.SetGraphQLRunner(graphql.NewClient(nil))
	st.SetTestEvaluator(runner.EvaluateAssertions)
	st.SetSuiteRunner(runner.NewPoolRunner(enginehttp.NewClient(30*time.Second), env.InterpolateRequest, runner.EvaluateAssertions))

	app := &App{
		opts:  opts,
		state: st,
		db:    dbStore,
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

	st.SetSaveFilePicker(func(title, defaultName string, extensions []string, onPicked func(path string)) {
		go func() {
			var filters []mygo.FileFilter
			if len(extensions) > 0 {
				filters = []mygo.FileFilter{{
					Name:       "HAR Archive",
					Extensions: extensions,
				}}
			}
			savePath, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
				Parent:            app.win,
				Title:             title,
				DefaultPath:       defaultName,
				Filters:           filters,
				CreateDirectories: true,
			})
			if err != nil || savePath == "" {
				return
			}
			onPicked(savePath)
		}()
	})

	return app
}

// State returns the application's thread-safe state.
func (a *App) State() *AppState {
	return a.state
}

// Close releases persisted application resources and shuts down active connections.
func (a *App) Close() error {
	if a.state != nil {
		a.state.DisconnectStream()
	}
	if a.db != nil {
		return a.db.Close()
	}
	return nil
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
