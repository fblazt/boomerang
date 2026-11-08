package app

import (
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
)

// AppState holds the thread-safe global reactive state of the Boomerang application.
type AppState struct {
	mu sync.RWMutex

	// UI Redraw trigger callback (e.g. services.Invalidate or test hook)
	redrawFn func()

	// Active Request & Response
	ActiveRequest  models.RequestModel
	ActiveResponse *models.ResponseModel
	IsLoading      bool
	ElapsedMs      int64

	// Active Environment
	ActiveEnv string

	// Navigation & View Mode
	ActiveTab    string // params, headers, body, auth, tests
	SidebarOpen  bool
	CommandPalOpen bool
}

// NewAppState initializes an AppState with default values.
func NewAppState(redrawFn func()) *AppState {
	return &AppState{
		redrawFn: redrawFn,
		ActiveRequest: models.RequestModel{
			Method: "GET",
			URL:    "https://api.internal.net/v1/health",
			Params: []models.KeyValuePair{},
			Headers: []models.KeyValuePair{
				{Key: "Accept", Value: "application/json", Enabled: true},
			},
			BodyType: "json",
			AuthType: "none",
		},
		ActiveEnv:   "PROD",
		ActiveTab:   "params",
		SidebarOpen: true,
	}
}

// SetRedrawFunc attaches a redraw callback to the state.
func (s *AppState) SetRedrawFunc(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.redrawFn = fn
}

// RequestRedraw safely triggers a frame re-render on the UI thread.
func (s *AppState) RequestRedraw() {
	s.mu.RLock()
	fn := s.redrawFn
	s.mu.RUnlock()

	if fn != nil {
		fn()
	}
}

// Mutate performs thread-safe state mutation and triggers a redraw.
func (s *AppState) Mutate(fn func(s *AppState)) {
	s.mu.Lock()
	fn(s)
	s.mu.Unlock()

	s.RequestRedraw()
}

// View executes a read-only closure with read-lock held.
func (s *AppState) View(fn func(s *AppState)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fn(s)
}

// SetMethod updates the HTTP verb and requests redraw.
func (s *AppState) SetMethod(method string) {
	s.Mutate(func(state *AppState) {
		state.ActiveRequest.Method = method
	})
}

// SetURL updates the URL string and requests redraw.
func (s *AppState) SetURL(url string) {
	s.Mutate(func(state *AppState) {
		state.ActiveRequest.URL = url
	})
}

// SetLoading updates loading state and elapsed duration.
func (s *AppState) SetLoading(loading bool, elapsedMs int64) {
	s.Mutate(func(state *AppState) {
		state.IsLoading = loading
		state.ElapsedMs = elapsedMs
	})
}

// SetResponse sets active response and marks loading finished.
func (s *AppState) SetResponse(resp *models.ResponseModel) {
	s.Mutate(func(state *AppState) {
		state.ActiveResponse = resp
		state.IsLoading = false
	})
}

// SetActiveEnv switches environment (PROD, STG, DEV, etc.).
func (s *AppState) SetActiveEnv(env string) {
	s.Mutate(func(state *AppState) {
		state.ActiveEnv = env
	})
}

// BindServices wires mygo/ui.Services to trigger c.Services().Invalidate() on state updates.
func (s *AppState) BindServices(services ui.Services) {
	s.SetRedrawFunc(func() {
		services.Invalidate()
	})
}
