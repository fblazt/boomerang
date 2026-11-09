package app_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/app"
	"github.com/fblazt/boomerang/internal/engine/env"
	"github.com/fblazt/boomerang/internal/models"
)

func TestAppState_ThreadSafetyAndMutations(t *testing.T) {
	redrawCount := 0
	var mu sync.Mutex

	state := app.NewAppState(func() {
		mu.Lock()
		redrawCount++
		mu.Unlock()
	})

	if state.ActiveRequest.Method != "GET" {
		t.Fatalf("expected GET, got %s", state.ActiveRequest.Method)
	}

	state.SetMethod("POST")
	state.SetURL("https://api.test/v1/resource")
	state.SetActiveEnv("DEV")

	state.View(func(s *app.AppState) {
		if s.ActiveRequest.Method != "POST" {
			t.Fatalf("expected POST, got %s", s.ActiveRequest.Method)
		}
		if s.ActiveRequest.URL != "https://api.test/v1/resource" {
			t.Fatalf("expected https://api.test/v1/resource, got %s", s.ActiveRequest.URL)
		}
		if s.ActiveEnv != "DEV" {
			t.Fatalf("expected DEV, got %s", s.ActiveEnv)
		}
	})

	mu.Lock()
	count := redrawCount
	mu.Unlock()

	if count < 3 {
		t.Fatalf("expected at least 3 redraw triggers, got %d", count)
	}
}

func TestAppState_ConcurrentAccess(t *testing.T) {
	state := app.NewAppState(nil)
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			state.SetLoading(true, int64(idx*10))
			state.SetResponse(&models.ResponseModel{
				StatusCode: 200,
				StatusText: "OK",
				Duration:   time.Duration(idx) * time.Millisecond,
			})
		}(i)

		go func() {
			defer wg.Done()
			state.View(func(s *app.AppState) {
				_ = s.ActiveRequest.Method
				_ = s.IsLoading
				if s.ActiveResponse != nil {
					_ = s.ActiveResponse.StatusCode
				}
			})
		}()
	}

	wg.Wait()
}

func TestAppState_HeadlessTesterReactive(t *testing.T) {
	state := app.NewAppState(nil)

	tt := ui.NewTester(func(c *ui.Context) {
		// Wire services on first frame
		state.BindServices(c.Services())

		state.View(func(s *app.AppState) {
			ui.Column(c).Children(func() {
				ui.Text(c, "Method: "+s.ActiveRequest.Method)
				if s.IsLoading {
					ui.Text(c, "Status: Sending...")
				} else {
					ui.Text(c, "Status: Idle")
				}
			})
		})
	}, 800, 600)

	tt.Frame()
	if !tt.HasText("Method: GET") {
		t.Fatal("expected 'Method: GET'")
	}
	if !tt.HasText("Status: Idle") {
		t.Fatal("expected 'Status: Idle'")
	}

	// Trigger state mutation
	state.SetMethod("POST")
	state.SetLoading(true, 42)

	tt.Frame()
	if !tt.HasText("Method: POST") {
		t.Fatal("expected 'Method: POST'")
	}
	if !tt.HasText("Status: Sending...") {
		t.Fatal("expected 'Status: Sending...'")
	}
}

type mockExecutor struct {
	lastReq models.RequestModel
	done    chan struct{}
}

func (m *mockExecutor) Execute(ctx context.Context, req models.RequestModel) (*models.ResponseModel, error) {
	m.lastReq = req
	close(m.done)
	return &models.ResponseModel{
		StatusCode: 200,
		StatusText: "200 OK",
	}, nil
}

func TestAppState_SendRequest_WithInterpolation(t *testing.T) {
	state := app.NewAppState(nil)
	state.SetInterpolator(env.InterpolateRequest, env.Interpolate)

	state.ActiveEnv = "PROD"
	state.ActiveRequest.URL = "{{base_url}}/users"
	state.ActiveRequest.Headers = []models.KeyValuePair{
		{Key: "Authorization", Value: "Bearer {{api_key}}", Enabled: true},
	}

	mockExec := &mockExecutor{done: make(chan struct{})}
	state.SetExecutor(mockExec)

	state.SendRequest()

	select {
	case <-mockExec.done:
	case <-time.After(2 * time.Second):
		t.Fatal("mock execution timed out")
	}

	if mockExec.lastReq.URL != "https://api.internal.net/v1/users" {
		t.Fatalf("expected interpolated URL 'https://api.internal.net/v1/users', got %q", mockExec.lastReq.URL)
	}
	if len(mockExec.lastReq.Headers) != 1 || mockExec.lastReq.Headers[0].Value != "Bearer bk_live_94aB882c0f12" {
		t.Fatalf("expected interpolated header 'Bearer bk_live_94aB882c0f12', got %v", mockExec.lastReq.Headers)
	}
	// ActiveRequest should remain untouched (templated)
	if state.ActiveRequest.URL != "{{base_url}}/users" {
		t.Fatalf("ActiveRequest.URL should remain templated, got %q", state.ActiveRequest.URL)
	}
}
