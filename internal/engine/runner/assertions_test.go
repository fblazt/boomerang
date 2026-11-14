package runner

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fblazt/boomerang/internal/models"
)

func TestEvaluateAssertions_StatusTimingAndSchema(t *testing.T) {
	req := models.RequestModel{
		Method: "GET",
		URL:    "https://api.example.com/users",
	}
	resp := &models.ResponseModel{
		StatusCode: 200,
		StatusText: "200 OK",
		Duration:   42 * time.Millisecond,
		SizeBytes:  128,
		Headers: map[string]string{
			"Content-Type": "application/json; charset=utf-8",
		},
		Body: `{
			"ok": true,
			"meta": {"version": 2, "trace_id": "tr_991a"},
			"users": [
				{"id": "u_1", "email": "ada@boomerang.dev"},
				{"id": "u_2", "email": "grace@boomerang.dev"}
			]
		}`,
	}

	script := `
// Standard status, timing, header & JSON schema assertions
pm.test("Status is 200", function () {
  pm.response.to.have.status(200);
});

pm.test("Latency under 100ms", () => {
  pm.expect(pm.response.responseTime).to.be.below(100);
});

pm.test("Response payload matches schema", () => {
  const data = pm.response.json();
  pm.expect(data.ok).to.be.true;
  pm.expect(data.users).to.be.a("array");
  pm.expect(data.users).to.have.lengthOf(2);
  pm.expect(data.users[1].email).to.include("@boomerang.dev");
  pm.expect(data.meta).to.have.property("trace_id");
  pm.expect(data.meta.trace_id).to.match(/^tr_[0-9a-z]+$/);
});

pm.test("Content-Type JSON header", () => {
  pm.response.to.have.header("content-type", "application/json");
});
`

	results := EvaluateAssertions(script, req, resp)
	if len(results) != 4 {
		t.Fatalf("expected 4 assertion results, got %d: %+v", len(results), results)
	}
	for _, r := range results {
		if !r.Passed {
			t.Fatalf("expected assertion %q to pass, failed with: %s", r.Name, r.Message)
		}
	}
}

func TestEvaluateAssertions_FailuresAndNegations(t *testing.T) {
	req := models.RequestModel{Method: "POST", URL: "https://api.example.com/orders"}
	resp := &models.ResponseModel{
		StatusCode: 503,
		Duration:   350 * time.Millisecond,
		Body:       `{"error": "upstream timeout"}`,
	}

	script := `
pm.test("Expect created status", () => {
  pm.response.to.have.status(201);
});

pm.test("Fast execution", () => {
  pm.expect(pm.response.responseTime).to.be.below(200);
});

pm.test("No server error field", () => {
  pm.expect(pm.response.json().error).to.equal("none");
});

pm.test("Status is not 200", () => {
  pm.expect(pm.response.code).to.not.equal(200);
});
`

	results := EvaluateAssertions(script, req, resp)
	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}
	if results[0].Passed || !strings.Contains(results[0].Message, "201") {
		t.Fatalf("expected status failure, got %+v", results[0])
	}
	if results[1].Passed || !strings.Contains(results[1].Message, "below 200") {
		t.Fatalf("expected timing failure, got %+v", results[1])
	}
	if results[2].Passed || !strings.Contains(results[2].Message, "upstream timeout") {
		t.Fatalf("expected JSON equality failure, got %+v", results[2])
	}
	if !results[3].Passed {
		t.Fatalf("expected negated assertion to pass, got %+v", results[3])
	}
}

func TestPoolRunner_WithTestAssertions(t *testing.T) {
	exec := &stubExecutor{
		statusByURL: map[string]int{
			"https://api.test/pass": 200,
			"https://api.test/fail": 200,
		},
	}
	r := NewPoolRunner(exec, nil, EvaluateAssertions)

	col := models.Collection{
		ID:   "col_assert",
		Name: "Assertions Suite",
		Items: []models.CollectionItem{
			{
				ID:   "i_pass",
				Name: "Pass Step",
				Request: models.RequestModel{
					Method:     "GET",
					URL:        "https://api.test/pass",
					TestScript: `pm.test("OK", () => { pm.response.to.have.status(200); pm.expect(pm.response.json().ok).to.be.true; });`,
				},
			},
			{
				ID:   "i_fail",
				Name: "Fail Assertion Step",
				Request: models.RequestModel{
					Method:     "GET",
					URL:        "https://api.test/fail",
					TestScript: `pm.test("Needs 201", () => { pm.response.to.have.status(201); });`,
				},
			},
		},
	}

	report, err := r.RunSuite(context.Background(), col, models.SuiteRunConfig{Concurrency: 2}, nil)
	if err != nil {
		t.Fatalf("RunSuite failed: %v", err)
	}
	if report.TotalAssertions != 2 || report.PassedAssertions != 1 || report.FailedAssertions != 1 {
		t.Fatalf("expected 2 total / 1 passed / 1 failed assertions, got %+v", report)
	}
	if report.PassedRequests != 1 || report.FailedRequests != 1 {
		t.Fatalf("expected request pass/fail derived from assertions, got %+v", report)
	}
}
