package models

import (
	"context"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
)

// RequestExecutor defines the interface for executing HTTP requests.
type RequestExecutor interface {
	Execute(ctx context.Context, req RequestModel) (*ResponseModel, error)
}

// AppState holds the thread-safe global reactive state of the Boomerang application.
type AppState struct {
	mu sync.RWMutex

	// UI Redraw trigger callback (e.g. services.Invalidate or test hook)
	redrawFn func()

	// HTTP Client Executor
	executor RequestExecutor
	cancelFn context.CancelFunc

	// Active Request & Response
	ActiveRequest  RequestModel
	ActiveResponse *ResponseModel
	IsLoading      bool
	ElapsedMs      int64

	// Active Environment
	ActiveEnv string

	// Navigation & View Mode
	ActiveTab      string // params, headers, body, auth, settings
	SidebarOpen    bool
	CommandPalOpen bool
}

// NewAppState initializes an AppState with default values.
func NewAppState(redrawFn func()) *AppState {
	return &AppState{
		redrawFn: redrawFn,
		ActiveRequest: RequestModel{
			ID:     "req_default",
			Name:   "Health Check",
			Method: "GET",
			URL:    "https://api.internal.net/v1/health",
			Params: []KeyValuePair{},
			Headers: []KeyValuePair{
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

// SetExecutor configures the HTTP request execution engine.
func (s *AppState) SetExecutor(exec RequestExecutor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.executor = exec
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

// SetURL updates the URL string and syncs extracted parameters.
func (s *AppState) SetURL(rawURL string) {
	s.Mutate(func(state *AppState) {
		state.ActiveRequest.URL = rawURL
		state.ActiveRequest.Params = SyncURLToParams(rawURL)
	})
}

// SetParams updates parameters and syncs back to the URL.
func (s *AppState) SetParams(params []KeyValuePair) {
	s.Mutate(func(state *AppState) {
		state.ActiveRequest.Params = params
		state.ActiveRequest.URL = SyncParamsToURL(state.ActiveRequest.URL, params)
	})
}

// AddParam adds a new parameter and updates the URL.
func (s *AppState) AddParam(key, value, desc string) {
	s.Mutate(func(state *AppState) {
		state.ActiveRequest.Params = append(state.ActiveRequest.Params, KeyValuePair{
			Key:         key,
			Value:       value,
			Description: desc,
			Enabled:     true,
		})
		state.ActiveRequest.URL = SyncParamsToURL(state.ActiveRequest.URL, state.ActiveRequest.Params)
	})
}

// RemoveParam removes a parameter by index and updates the URL.
func (s *AppState) RemoveParam(index int) {
	s.Mutate(func(state *AppState) {
		if index >= 0 && index < len(state.ActiveRequest.Params) {
			state.ActiveRequest.Params = append(state.ActiveRequest.Params[:index], state.ActiveRequest.Params[index+1:]...)
			state.ActiveRequest.URL = SyncParamsToURL(state.ActiveRequest.URL, state.ActiveRequest.Params)
		}
	})
}

// ToggleParam flips the enabled status of a parameter and updates the URL.
func (s *AppState) ToggleParam(index int) {
	s.Mutate(func(state *AppState) {
		if index >= 0 && index < len(state.ActiveRequest.Params) {
			state.ActiveRequest.Params[index].Enabled = !state.ActiveRequest.Params[index].Enabled
			state.ActiveRequest.URL = SyncParamsToURL(state.ActiveRequest.URL, state.ActiveRequest.Params)
		}
	})
}

// SetHeaders updates request headers.
func (s *AppState) SetHeaders(headers []KeyValuePair) {
	s.Mutate(func(state *AppState) {
		state.ActiveRequest.Headers = headers
	})
}

// AddHeader appends a new header row.
func (s *AppState) AddHeader(key, value, desc string) {
	s.Mutate(func(state *AppState) {
		state.ActiveRequest.Headers = append(state.ActiveRequest.Headers, KeyValuePair{
			Key:         key,
			Value:       value,
			Description: desc,
			Enabled:     true,
		})
	})
}

// RemoveHeader deletes a header row by index.
func (s *AppState) RemoveHeader(index int) {
	s.Mutate(func(state *AppState) {
		if index >= 0 && index < len(state.ActiveRequest.Headers) {
			state.ActiveRequest.Headers = append(state.ActiveRequest.Headers[:index], state.ActiveRequest.Headers[index+1:]...)
		}
	})
}

// ToggleHeader flips the enabled status of a header row.
func (s *AppState) ToggleHeader(index int) {
	s.Mutate(func(state *AppState) {
		if index >= 0 && index < len(state.ActiveRequest.Headers) {
			state.ActiveRequest.Headers[index].Enabled = !state.ActiveRequest.Headers[index].Enabled
		}
	})
}

// SetActiveTab updates the active configuration tab.
func (s *AppState) SetActiveTab(tab string) {
	s.Mutate(func(state *AppState) {
		state.ActiveTab = tab
	})
}

// SetLoading updates loading state and elapsed duration.
func (s *AppState) SetLoading(loading bool, elapsedMs int64) {
	s.Mutate(func(state *AppState) {
		state.IsLoading = loading
		state.ElapsedMs = elapsedMs
	})
}

// SetResponse sets the execution response and requests redraw.
func (s *AppState) SetResponse(resp *ResponseModel) {
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

// SendRequest triggers dispatching ActiveRequest asynchronously with live timer updates.
func (s *AppState) SendRequest() {
	s.mu.Lock()
	if s.IsLoading {
		s.mu.Unlock()
		return
	}
	s.IsLoading = true
	s.ElapsedMs = 0
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFn = cancel
	exec := s.executor
	reqCopy := s.ActiveRequest
	s.mu.Unlock()

	s.RequestRedraw()

	go func() {
		startTime := time.Now()
		done := make(chan struct{})

		// Background ticker to update elapsed duration in real time
		go func() {
			ticker := time.NewTicker(30 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					s.mu.Lock()
					if !s.IsLoading {
						s.mu.Unlock()
						return
					}
					s.ElapsedMs = time.Since(startTime).Milliseconds()
					s.mu.Unlock()
					s.RequestRedraw()
				}
			}
		}()

		var resp *ResponseModel
		var err error
		if exec != nil {
			resp, err = exec.Execute(ctx, reqCopy)
		} else {
			// Built-in simulated response for testing without mock server
			time.Sleep(30 * time.Millisecond)
			resp = &ResponseModel{
				StatusCode: 200,
				StatusText: "200 OK",
				Duration:   30 * time.Millisecond,
				SizeBytes:  1024,
				Proto:      "HTTP/2.0",
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       "{\n  \"status\": \"success\"\n}",
				Telemetry: Telemetry{
					DNS:      5,
					TCP:      10,
					TLS:      15,
					TTFB:     20,
					Download: 5,
					Total:    55,
				},
			}
		}
		close(done)

		if err != nil && resp == nil {
			resp = &ResponseModel{
				Error: err.Error(),
			}
		}

		s.mu.Lock()
		s.ActiveResponse = resp
		s.IsLoading = false
		s.cancelFn = nil
		s.ElapsedMs = time.Since(startTime).Milliseconds()
		s.mu.Unlock()

		s.RequestRedraw()
	}()
}

// CancelRequest cancels the currently active request if any.
func (s *AppState) CancelRequest() {
	s.mu.Lock()
	if s.cancelFn != nil {
		s.cancelFn()
		s.cancelFn = nil
	}
	s.IsLoading = false
	s.mu.Unlock()

	s.RequestRedraw()
}

// BindServices wires mygo Context Services to trigger Invalidate on state mutations.
func (s *AppState) BindServices(services ui.Services) {
	s.SetRedrawFunc(func() {
		services.Invalidate()
	})
}
