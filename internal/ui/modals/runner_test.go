package modals

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/engine/runner"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

type fakeSuiteExecutor struct{}

func (fakeSuiteExecutor) Execute(_ context.Context, req models.RequestModel) (*models.ResponseModel, error) {
	status := 200
	body := `{"ok":true,"status":"active"}`
	if req.URL == "https://api.test/fail" {
		status = 500
		body = `{"ok":false,"error":"db unavailable"}`
	}
	return &models.ResponseModel{
		StatusCode: status,
		StatusText: "OK",
		Duration:   12 * time.Millisecond,
		Body:       body,
	}, nil
}

func setupRunnerTestState() *models.AppState {
	state := models.NewAppState(nil)
	state.SetSuiteRunner(runner.NewPoolRunner(fakeSuiteExecutor{}, nil, runner.EvaluateAssertions))
	state.Mutate(func(s *models.AppState) {
		s.Collections = []models.Collection{
			{
				ID:   "col_api",
				Name: "Core API Suite",
				Items: []models.CollectionItem{
					{
						ID:   "item_1",
						Name: "List Health",
						Request: models.RequestModel{
							ID:         "req_1",
							Name:       "List Health",
							Method:     "GET",
							URL:        "https://api.test/health",
							TestScript: `pm.test("Status 200", () => { pm.response.to.have.status(200); });`,
						},
					},
					{
						ID:   "item_2",
						Name: "Billing Charge",
						Request: models.RequestModel{
							ID:         "req_2",
							Name:       "Billing Charge",
							Method:     "POST",
							URL:        "https://api.test/fail",
							TestScript: `pm.test("Expect 200", () => { pm.response.to.have.status(200); });`,
						},
					},
				},
			},
		}
		s.RunnerCollectionID = "col_api"
	})
	return state
}

func TestRunnerModal_ConcurrencySliderAndBatchExecution(t *testing.T) {
	state := setupRunnerTestState()
	state.OpenRunnerModal()

	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(theme.NewBoomerangTheme())
		RunnerModalHost(c, state)
	}, 1024, 720)
	tt.Frame()

	for _, want := range []string{"Collection Suite Runner", "Core API Suite", "4 threads", "Run Suite"} {
		if !tt.HasText(want) {
			t.Fatalf("expected %q in Suite Runner modal", want)
		}
	}

	// Adjust concurrency via preset chip "8x" and fine stepper "+" -> "9 threads"
	if err := tt.Click("8x"); err != nil {
		t.Fatalf("click 8x preset: %v", err)
	}
	tt.Frame()
	if !tt.HasText("8 threads") {
		t.Fatal("expected 8 threads readout after 8x preset click")
	}

	if err := tt.Click("+"); err != nil {
		t.Fatalf("click + stepper: %v", err)
	}
	tt.Frame()
	if !tt.HasText("9 threads") {
		t.Fatal("expected 9 threads readout after + stepper click")
	}

	// Select 16x for batch launch
	if err := tt.Click("16x"); err != nil {
		t.Fatalf("click 16x preset: %v", err)
	}
	tt.Frame()

	// Launch batch suite execution
	if err := tt.Click("Run Suite"); err != nil {
		t.Fatalf("click Run Suite: %v", err)
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		var done bool
		state.View(func(s *models.AppState) {
			done = !s.RunnerRunning && s.RunnerReport != nil
		})
		if done {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	tt.Frame()

	var gotReport *models.SuiteRunReport
	state.View(func(s *models.AppState) {
		gotReport = s.RunnerReport
	})
	if gotReport == nil {
		t.Fatal("expected RunnerReport populated after execution")
	}
	if gotReport.Concurrency != 16 {
		t.Fatalf("expected Concurrency=16 in report, got %d", gotReport.Concurrency)
	}
	if !tt.HasText("PROGRESS 2 / 2") || !tt.HasText("1/2 ok") || !tt.HasText("1/2 pass") {
		t.Fatal("expected live progress and assertion summary counts in Suite Runner modal")
	}
}

func TestRunnerModal_AssertionFilterAndExportReport(t *testing.T) {
	state := setupRunnerTestState()
	exportPath := filepath.Join(t.TempDir(), "suite-report.json")
	state.SetSaveFilePicker(func(_ string, _ string, _ []string, onPicked func(string)) {
		onPicked(exportPath)
	})
	state.OpenRunnerModal()

	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(theme.NewBoomerangTheme())
		RunnerModalHost(c, state)
	}, 1024, 720)
	tt.Frame()

	if err := tt.Click("Run Suite"); err != nil {
		t.Fatalf("click Run Suite: %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		var done bool
		state.View(func(s *models.AppState) {
			done = !s.RunnerRunning && s.RunnerReport != nil
		})
		if done {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	tt.Frame()

	for _, want := range []string{"List Health", "Billing Charge", "✓ Status 200", "✗ Expect 200: expected status 200, got 500"} {
		if !tt.HasText(want) {
			t.Fatalf("expected %q in assertion breakdown", want)
		}
	}

	// Filter to Failed only
	if err := tt.Click("Failed"); err != nil {
		t.Fatalf("click Failed filter: %v", err)
	}
	tt.Frame()
	if tt.HasText("List Health") || !tt.HasText("Billing Charge") {
		t.Fatal("expected Failed filter to hide passing List Health and show Billing Charge")
	}

	// Export JSON report
	if err := tt.Click("Export JSON"); err != nil {
		t.Fatalf("click Export JSON: %v", err)
	}
	tt.Frame()

	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("read exported suite report: %v", err)
	}
	if !strings.Contains(string(data), "Billing Charge") || !strings.Contains(string(data), "failed_assertions") {
		t.Fatalf("unexpected exported suite JSON:\n%s", string(data))
	}
}
