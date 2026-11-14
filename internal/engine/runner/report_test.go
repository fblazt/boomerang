package runner

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/fblazt/boomerang/internal/models"
)

func TestReportFormatters_JSONAndMarkdown(t *testing.T) {
	report := models.SuiteRunReport{
		RunID:            "run_42",
		CollectionID:     "col_1",
		CollectionName:   "Auth & Billing",
		Concurrency:      8,
		StartedAt:        time.Now().UTC(),
		FinishedAt:       time.Now().UTC(),
		TotalDurationMs:  124,
		TotalRequests:    2,
		PassedRequests:   1,
		FailedRequests:   1,
		TotalAssertions:  2,
		PassedAssertions: 1,
		FailedAssertions: 1,
		Steps: []models.SuiteStepResult{
			{
				Index:      0,
				Name:       "Login",
				Method:     "POST",
				URL:        "https://api.test/login",
				StatusCode: 200,
				DurationMs: 42,
				Passed:     true,
				Assertions: []models.TestAssertionResult{
					{Name: "Status 200", Passed: true},
				},
			},
			{
				Index:      1,
				Name:       "Invoice",
				Method:     "GET",
				URL:        "https://api.test/invoice",
				StatusCode: 500,
				DurationMs: 82,
				Passed:     false,
				Assertions: []models.TestAssertionResult{
					{Name: "Status 200", Passed: false, Message: "expected status 200, got 500"},
				},
			},
		},
	}

	raw, err := FormatReportJSON(report)
	if err != nil {
		t.Fatalf("FormatReportJSON failed: %v", err)
	}
	var decoded models.SuiteRunReport
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal report json: %v", err)
	}
	if decoded.RunID != "run_42" || decoded.FailedAssertions != 1 || len(decoded.Steps) != 2 {
		t.Fatalf("unexpected decoded JSON report: %+v", decoded)
	}

	md := FormatReportMarkdown(report)
	for _, want := range []string{"# Suite Run Report: Auth & Billing", "8 worker threads", "| 1 | PASS | POST | Login | 200 | 42 ms | 1/1 |", "| 2 | FAIL | GET | Invoice | 500 | 82 ms | 0/1 |"} {
		if !strings.Contains(md, want) {
			t.Fatalf("expected %q in markdown report:\n%s", want, md)
		}
	}
}
