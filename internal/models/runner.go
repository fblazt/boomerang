package models

import (
	"context"
	"time"
)

// TestAssertionResult records the outcome of a single test assertion evaluated against a response.
type TestAssertionResult struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
}

// SuiteRunConfig configures a concurrent collection suite execution run.
type SuiteRunConfig struct {
	CollectionID   string            `json:"collection_id"`
	CollectionName string            `json:"collection_name"`
	Concurrency    int               `json:"concurrency"`
	StopOnError    bool              `json:"stop_on_error,omitempty"`
	Variables      map[string]string `json:"variables,omitempty"`
}

// SuiteStepResult stores telemetry, status, and assertions for a single executed request step.
type SuiteStepResult struct {
	Index      int                   `json:"index"`
	ItemID     string                `json:"item_id"`
	Name       string                `json:"name"`
	Method     string                `json:"method"`
	URL        string                `json:"url"`
	StatusCode int                   `json:"status_code"`
	StatusText string                `json:"status_text"`
	DurationMs int64                 `json:"duration_ms"`
	Error      string                `json:"error,omitempty"`
	Passed     bool                  `json:"passed"`
	Assertions []TestAssertionResult `json:"assertions,omitempty"`
}

// SuiteRunProgress delivers real-time worker execution progress to the UI.
type SuiteRunProgress struct {
	Completed     int             `json:"completed"`
	Total         int             `json:"total"`
	ActiveWorkers int             `json:"active_workers"`
	LastStep      SuiteStepResult `json:"last_step"`
}

// SuiteRunReport summarizes the completed suite execution with aggregate assertion metrics.
type SuiteRunReport struct {
	RunID            string            `json:"run_id"`
	CollectionID     string            `json:"collection_id"`
	CollectionName   string            `json:"collection_name"`
	Concurrency      int               `json:"concurrency"`
	StartedAt        time.Time         `json:"started_at"`
	FinishedAt       time.Time         `json:"finished_at"`
	TotalDurationMs  int64             `json:"total_duration_ms"`
	TotalRequests    int               `json:"total_requests"`
	PassedRequests   int               `json:"passed_requests"`
	FailedRequests   int               `json:"failed_requests"`
	TotalAssertions  int               `json:"total_assertions"`
	PassedAssertions int               `json:"passed_assertions"`
	FailedAssertions int               `json:"failed_assertions"`
	Cancelled        bool              `json:"cancelled,omitempty"`
	Steps            []SuiteStepResult `json:"steps"`
}

// SuiteRunner executes a collection concurrently across a worker pool.
type SuiteRunner interface {
	RunSuite(ctx context.Context, col Collection, cfg SuiteRunConfig, onProgress func(SuiteRunProgress)) (SuiteRunReport, error)
}
