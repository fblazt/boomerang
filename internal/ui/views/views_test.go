package views

import (
	"context"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
)

func TestShellView(t *testing.T) {
	state := models.NewAppState(nil)
	state.ActiveRequest.Name = "Test Request"
	state.ActiveRequest.Method = "GET"
	state.ActiveRequest.URL = "https://api.internal.net/v1/test"

	tt := ui.NewTester(func(c *ui.Context) {
		ShellView(c, state)
	}, 1440, 900)
	tt.Frame()

	// Assert Header
	if !tt.HasText("Boomerang") {
		t.Fatal("expected Boomerang header text")
	}
	if !tt.HasText("PROD") {
		t.Fatal("expected PROD environment badge")
	}
	if !tt.HasText("⌘K") {
		t.Fatal("expected ⌘K shortcut cap")
	}

	// Assert Sidebar
	if !tt.HasText("COLLECTIONS") {
		t.Fatal("expected COLLECTIONS sidebar section")
	}
	if !tt.HasText("REALTIME STREAMS") {
		t.Fatal("expected REALTIME STREAMS sidebar section")
	}
	if !tt.HasText("Users Service") {
		t.Fatal("expected Users Service collection item")
	}

	// Assert Main Workspace
	if !tt.HasText("Send") {
		t.Fatal("expected Send button")
	}
	if !tt.HasText("{{base_url}}") {
		t.Fatal("expected {{base_url}} variable chip")
	}

	// Assert Status Bar
	if !tt.HasText("Engine Online") {
		t.Fatal("expected Engine Online status indicator")
	}
	if !tt.HasText("Sockets:") {
		t.Fatal("expected Sockets: label")
	}
	if !tt.HasText("Ping:") {
		t.Fatal("expected Ping: label")
	}
}

func TestHeaderView_EnvToggle(t *testing.T) {
	state := models.NewAppState(nil)
	state.ActiveEnv = "PROD"

	tt := ui.NewTester(func(c *ui.Context) {
		HeaderView(c, state)
	}, 1000, 44)
	tt.Frame()

	if !tt.HasText("PROD") {
		t.Fatal("expected PROD")
	}

	if err := tt.Click("PROD"); err != nil {
		t.Fatalf("failed to click PROD: %v", err)
	}
	tt.Frame()

	if state.ActiveEnv != "STG" {
		t.Fatalf("expected ActiveEnv to change to STG, got %s", state.ActiveEnv)
	}
	if !tt.HasText("STG") {
		t.Fatal("expected UI to show STG")
	}
}

func TestSidebarView_ItemSelection(t *testing.T) {
	state := models.NewAppState(nil)

	tt := ui.NewTester(func(c *ui.Context) {
		SidebarView(c, state)
	}, 260, 600)
	tt.Frame()

	if err := tt.Click("Auth Verify"); err != nil {
		t.Fatalf("failed to click Auth Verify: %v", err)
	}
	tt.Frame()

	if state.ActiveRequest.Name != "Auth Verify" {
		t.Fatalf("expected ActiveRequest.Name to be 'Auth Verify', got %q", state.ActiveRequest.Name)
	}
	if state.ActiveRequest.Method != "POST" {
		t.Fatalf("expected ActiveRequest.Method to be 'POST', got %q", state.ActiveRequest.Method)
	}
}

func TestRequestTabsView_Switching(t *testing.T) {
	state := models.NewAppState(nil)
	state.ActiveRequest.Params = []models.KeyValuePair{
		{Key: "page", Value: "1", Enabled: true},
	}
	state.ActiveRequest.Headers = []models.KeyValuePair{
		{Key: "Content-Type", Value: "application/json", Enabled: true},
	}

	tt := ui.NewTester(func(c *ui.Context) {
		RequestTabsView(c, state)
	}, 900, 400)
	tt.Frame()

	// Initial tab is Params
	if !tt.HasText("+ Add Parameter") {
		t.Fatal("expected params table with + Add Parameter button")
	}

	// Click Headers tab
	if err := tt.Click("Headers"); err != nil {
		t.Fatalf("failed to click Headers tab: %v", err)
	}
	tt.Frame()

	if state.ActiveTab != "headers" {
		t.Fatalf("expected ActiveTab to be headers, got %s", state.ActiveTab)
	}
	if !tt.HasText("+ Add Header") {
		t.Fatal("expected headers table with + Add Header button")
	}

	// Click Body tab
	if err := tt.Click("Body"); err != nil {
		t.Fatalf("failed to click Body tab: %v", err)
	}
	tt.Frame()

	if state.ActiveTab != "body" {
		t.Fatalf("expected ActiveTab to be body, got %s", state.ActiveTab)
	}
	if !tt.HasText("JSON") || !tt.HasText("Form-Data") || !tt.HasText("x-www-form-urlencoded") {
		t.Fatal("expected body format pills")
	}

	// Click Auth tab
	if err := tt.Click("Auth"); err != nil {
		t.Fatalf("failed to click Auth tab: %v", err)
	}
	tt.Frame()

	if state.ActiveTab != "auth" {
		t.Fatalf("expected ActiveTab to be auth, got %s", state.ActiveTab)
	}
	if !tt.HasText("Bearer Token") || !tt.HasText("Basic Auth") || !tt.HasText("API Key") {
		t.Fatal("expected auth type selectors")
	}

	// Select Bearer Token
	if err := tt.Click("Bearer Token"); err != nil {
		t.Fatalf("failed to click Bearer Token: %v", err)
	}
	tt.Frame()

	if state.ActiveRequest.AuthType != "bearer" {
		t.Fatalf("expected AuthType bearer, got %s", state.ActiveRequest.AuthType)
	}
}

func TestURLBar_SendAndCancelTransition(t *testing.T) {
	state := models.NewAppState(nil)
	state.ActiveRequest.URL = "https://api.internal.net/v1/users"

	// Mock slow executor so we can observe in-flight loading state
	slowExec := &blockingExecutor{started: make(chan struct{}), block: make(chan struct{})}
	state.SetExecutor(slowExec)

	tt := ui.NewTester(func(c *ui.Context) {
		MainWorkspaceView(c, state)
	}, 1200, 800)
	tt.Frame()

	// Initially Send button is visible
	if !tt.HasText("Send") {
		t.Fatal("expected Send button")
	}

	// Click Send
	if err := tt.Click("Send"); err != nil {
		t.Fatalf("failed to click Send: %v", err)
	}
	<-slowExec.started
	tt.Frame()

	// Should transition to Cancel button and in-flight state
	if !state.IsLoading {
		t.Fatal("expected IsLoading to be true")
	}
	if !tt.HasText("Cancel") {
		t.Fatal("expected Cancel button during in-flight request")
	}
	if !tt.HasText("Request in-flight") {
		t.Fatal("expected in-flight status text")
	}

	// Click Cancel
	if err := tt.Click("Cancel"); err != nil {
		t.Fatalf("failed to click Cancel: %v", err)
	}
	tt.Frame()

	if state.IsLoading {
		t.Fatal("expected IsLoading to be false after cancel")
	}
	if !tt.HasText("Send") {
		t.Fatal("expected Send button restored after cancel")
	}

	close(slowExec.block)
}

type blockingExecutor struct {
	started chan struct{}
	block   chan struct{}
}

func (b *blockingExecutor) Execute(ctx context.Context, req models.RequestModel) (*models.ResponseModel, error) {
	close(b.started)
	select {
	case <-ctx.Done():
		return &models.ResponseModel{Error: ctx.Err().Error()}, ctx.Err()
	case <-b.block:
		return &models.ResponseModel{
			StatusCode: 200,
			StatusText: "OK",
		}, nil
	}
}

func TestResponseView_TelemetryWaterfallAndError(t *testing.T) {
	// 1. Successful response with telemetry
	state := models.NewAppState(nil)
	state.SetResponse(&models.ResponseModel{
		StatusCode: 200,
		StatusText: "200 OK",
		Duration:   120 * time.Millisecond,
		SizeBytes:  1024,
		Proto:      "HTTP/2.0",
		Body:       "{\n  \"status\": \"active\"\n}",
		Telemetry: models.Telemetry{
			DNS:      12,
			TCP:      19,
			TLS:      28,
			TTFB:     42,
			Download: 19,
			Total:    120,
		},
	})

	tt := ui.NewTester(func(c *ui.Context) {
		ResponseView(c, state)
	}, 1000, 500)
	tt.Frame()

	if !tt.HasText("200 200 OK") {
		t.Fatal("expected status 200 badge")
	}
	if !tt.HasText("DNS 12ms") || !tt.HasText("TCP 19ms") || !tt.HasText("TLS 28ms") || !tt.HasText("TTFB 42ms") || !tt.HasText("DL 19ms") {
		t.Fatal("expected telemetry waterfall breakdown legend")
	}
	if !tt.HasText("Copy") {
		t.Fatal("expected copy button")
	}

	// 2. Error response diagnostic
	state.SetResponse(&models.ResponseModel{
		Error: "connection refused: dial tcp 127.0.0.1:8080",
	})
	tt.Frame()

	if !tt.HasText("Request Failed") {
		t.Fatal("expected Request Failed title")
	}
	if !tt.HasText("connection refused: dial tcp 127.0.0.1:8080") {
		t.Fatal("expected diagnostic error message")
	}
}

func TestMainWorkspaceView_QuerySync(t *testing.T) {
	state := models.NewAppState(nil)
	state.ActiveRequest.URL = "https://api.internal.net/v1/users"

	tt := ui.NewTester(func(c *ui.Context) {
		MainWorkspaceView(c, state)
	}, 1200, 800)
	tt.Frame()

	// Add parameter programmatically and verify URL synced
	state.AddParam("sort", "desc", "sorting order")
	tt.Frame()

	if state.ActiveRequest.URL != "https://api.internal.net/v1/users?sort=desc" {
		t.Fatalf("expected synced URL, got %s", state.ActiveRequest.URL)
	}

	// Cycle method
	if err := tt.Click("GET"); err != nil {
		t.Fatalf("failed to click method badge: %v", err)
	}
	tt.Frame()

	if state.ActiveRequest.Method != "POST" {
		t.Fatalf("expected method POST after clicking GET, got %s", state.ActiveRequest.Method)
	}
}
