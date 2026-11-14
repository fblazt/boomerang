package runner

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fblazt/boomerang/internal/models"
)

type stubExecutor struct {
	delay         time.Duration
	curInFlight   atomic.Int64
	maxInFlight   atomic.Int64
	mu            sync.Mutex
	seenURLs      []string
	failItemURL   string
	statusByURL   map[string]int
}

func (s *stubExecutor) Execute(ctx context.Context, req models.RequestModel) (*models.ResponseModel, error) {
	inFlight := s.curInFlight.Add(1)
	for {
		prev := s.maxInFlight.Load()
		if inFlight <= prev || s.maxInFlight.CompareAndSwap(prev, inFlight) {
			break
		}
	}
	defer s.curInFlight.Add(-1)

	s.mu.Lock()
	s.seenURLs = append(s.seenURLs, req.URL)
	s.mu.Unlock()

	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if s.failItemURL != "" && req.URL == s.failItemURL {
		return nil, errors.New("connection refused")
	}

	code := 200
	text := "200 OK"
	if c, ok := s.statusByURL[req.URL]; ok {
		code = c
		text = fmt.Sprintf("%d", c)
	}

	return &models.ResponseModel{
		StatusCode: code,
		StatusText: text,
		Duration:   5 * time.Millisecond,
		Body:       `{"ok":true}`,
	}, nil
}

func makeTestCollection(n int) models.Collection {
	items := make([]models.CollectionItem, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, models.CollectionItem{
			ID:    fmt.Sprintf("item_%d", i),
			Name:  fmt.Sprintf("Endpoint %d", i),
			Order: i,
			Request: models.RequestModel{
				ID:     fmt.Sprintf("req_%d", i),
				Name:   fmt.Sprintf("Endpoint %d", i),
				Method: "GET",
				URL:    fmt.Sprintf("{{base}}/v1/r/%d", i),
			},
		})
	}
	return models.Collection{
		ID:    "col_perf",
		Name:  "Platform API",
		Items: items,
	}
}

func TestClampConcurrency(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{-5, 1},
		{0, 1},
		{1, 1},
		{8, 8},
		{32, 32},
		{99, 32},
	}
	for _, tc := range cases {
		if got := ClampConcurrency(tc.in); got != tc.want {
			t.Fatalf("ClampConcurrency(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestPoolRunner_ConcurrentExecutionAndOrder(t *testing.T) {
	exec := &stubExecutor{
		delay: 25 * time.Millisecond,
	}
	interp := func(req models.RequestModel, vars map[string]string) models.RequestModel {
		if base := vars["base"]; base != "" && len(req.URL) >= 8 && req.URL[:8] == "{{base}}" {
			req.URL = base + req.URL[8:]
		}
		return req
	}
	runner := NewPoolRunner(exec, interp, nil)

	col := makeTestCollection(12)
	var progressCount int
	report, err := runner.RunSuite(context.Background(), col, models.SuiteRunConfig{
		Concurrency: 6,
		Variables:   map[string]string{"base": "https://api.staging.net"},
	}, func(p models.SuiteRunProgress) {
		progressCount++
		if p.Total != 12 {
			t.Fatalf("expected progress total 12, got %d", p.Total)
		}
	})
	if err != nil {
		t.Fatalf("RunSuite failed: %v", err)
	}

	if report.TotalRequests != 12 || report.PassedRequests != 12 || report.FailedRequests != 0 {
		t.Fatalf("unexpected suite counts: %+v", report)
	}
	if progressCount != 12 {
		t.Fatalf("expected 12 progress updates, got %d", progressCount)
	}
	if exec.maxInFlight.Load() < 3 {
		t.Fatalf("expected concurrent worker execution >= 3 in-flight, got %d", exec.maxInFlight.Load())
	}
	if len(report.Steps) != 12 {
		t.Fatalf("expected 12 steps in report, got %d", len(report.Steps))
	}
	if report.Steps[0].URL != "https://api.staging.net/v1/r/0" || report.Steps[11].Index != 11 {
		t.Fatalf("expected ordered interpolated steps, got first=%+v last=%+v", report.Steps[0], report.Steps[11])
	}
}

func TestPoolRunner_FailuresAndCancellation(t *testing.T) {
	exec := &stubExecutor{
		failItemURL: "https://fail.local/err",
		statusByURL: map[string]int{
			"https://fail.local/404": 404,
		},
	}
	runner := NewPoolRunner(exec, nil, nil)

	col := models.Collection{
		ID:   "col_err",
		Name: "Errors",
		Items: []models.CollectionItem{
			{ID: "i1", Name: "OK", Request: models.RequestModel{Method: "GET", URL: "https://fail.local/ok"}},
			{ID: "i2", Name: "NetErr", Request: models.RequestModel{Method: "GET", URL: "https://fail.local/err"}},
		},
		Folders: []models.Folder{
			{
				ID:   "f1",
				Name: "Sub",
				Items: []models.CollectionItem{
					{ID: "i3", Name: "NotFound", Request: models.RequestModel{Method: "GET", URL: "https://fail.local/404"}},
				},
			},
		},
	}

	report, err := runner.RunSuite(context.Background(), col, models.SuiteRunConfig{Concurrency: 2}, nil)
	if err != nil {
		t.Fatalf("RunSuite failed: %v", err)
	}
	if report.TotalRequests != 3 || report.PassedRequests != 1 || report.FailedRequests != 2 {
		t.Fatalf("expected 1 passed / 2 failed, got %+v", report)
	}

	// Verify context cancellation flags Cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cancelledReport, err := runner.RunSuite(ctx, col, models.SuiteRunConfig{Concurrency: 2}, nil)
	if err != nil {
		t.Fatalf("cancelled RunSuite returned unexpected error: %v", err)
	}
	if !cancelledReport.Cancelled {
		t.Fatal("expected Cancelled=true when context already cancelled")
	}
}
