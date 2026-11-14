package runner

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fblazt/boomerang/internal/engine/collection"
	"github.com/fblazt/boomerang/internal/engine/storage"
	"github.com/fblazt/boomerang/internal/models"
)

const (
	MinConcurrency = 1
	MaxConcurrency = 32
)

// InterpolateFunc resolves environment variables in a request template before execution.
type InterpolateFunc func(req models.RequestModel, vars map[string]string) models.RequestModel

// AssertionEvalFunc evaluates test assertions against an executed request and response.
type AssertionEvalFunc func(script string, req models.RequestModel, resp *models.ResponseModel) []models.TestAssertionResult

// PoolRunner executes collection suites across a bounded worker goroutine pool.
type PoolRunner struct {
	executor      models.RequestExecutor
	interpolate   InterpolateFunc
	evaluateTests AssertionEvalFunc
}

// NewPoolRunner constructs a concurrent collection suite runner.
func NewPoolRunner(executor models.RequestExecutor, interpolate InterpolateFunc, evaluateTests AssertionEvalFunc) *PoolRunner {
	return &PoolRunner{
		executor:      executor,
		interpolate:   interpolate,
		evaluateTests: evaluateTests,
	}
}

// ClampConcurrency bounds worker pool size to [1, 32].
func ClampConcurrency(n int) int {
	if n < MinConcurrency {
		return MinConcurrency
	}
	if n > MaxConcurrency {
		return MaxConcurrency
	}
	return n
}

type suiteJob struct {
	index int
	item  models.CollectionItem
}

// RunSuite executes all requests inside col across cfg.Concurrency worker goroutines.
func (r *PoolRunner) RunSuite(
	ctx context.Context,
	col models.Collection,
	cfg models.SuiteRunConfig,
	onProgress func(models.SuiteRunProgress),
) (models.SuiteRunReport, error) {
	if r == nil || r.executor == nil {
		return models.SuiteRunReport{}, fmt.Errorf("suite runner requires a request executor")
	}

	items := collection.FlattenItems(col)
	concurrency := ClampConcurrency(cfg.Concurrency)
	startedAt := time.Now().UTC()

	colName := cfg.CollectionName
	if strings.TrimSpace(colName) == "" {
		colName = col.Name
	}
	colID := cfg.CollectionID
	if colID == "" {
		colID = col.ID
	}

	report := models.SuiteRunReport{
		RunID:          storage.NewID("run"),
		CollectionID:   colID,
		CollectionName: colName,
		Concurrency:    concurrency,
		StartedAt:      startedAt,
		TotalRequests:  len(items),
	}

	if len(items) == 0 {
		report.FinishedAt = time.Now().UTC()
		return report, nil
	}

	workerCount := concurrency
	if workerCount > len(items) {
		workerCount = len(items)
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan suiteJob, len(items))
	for i, item := range items {
		jobs <- suiteJob{index: i, item: item}
	}
	close(jobs)

	results := make([]models.SuiteStepResult, len(items))
	executed := make([]bool, len(items))

	var (
		completed     atomic.Int64
		activeWorkers atomic.Int64
		progressMu    sync.Mutex
		wg            sync.WaitGroup
	)

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if runCtx.Err() != nil {
					return
				}

				activeWorkers.Add(1)
				step := r.executeStep(runCtx, job, cfg.Variables)
				active := int(activeWorkers.Add(-1))

				results[job.index] = step
				executed[job.index] = true

				done := int(completed.Add(1))
				if onProgress != nil {
					progressMu.Lock()
					onProgress(models.SuiteRunProgress{
						Completed:     done,
						Total:         len(items),
						ActiveWorkers: active,
						LastStep:      step,
					})
					progressMu.Unlock()
				}

				if cfg.StopOnError && !step.Passed {
					cancel()
					return
				}
			}
		}()
	}

	wg.Wait()

	finishedAt := time.Now().UTC()
	report.FinishedAt = finishedAt
	report.TotalDurationMs = finishedAt.Sub(startedAt).Milliseconds()
	report.Cancelled = ctx.Err() != nil

	for i, item := range items {
		if !executed[i] {
			continue
		}
		step := results[i]
		report.Steps = append(report.Steps, step)
		if step.Passed {
			report.PassedRequests++
		} else {
			report.FailedRequests++
		}
		report.TotalAssertions += len(step.Assertions)
		for _, a := range step.Assertions {
			if a.Passed {
				report.PassedAssertions++
			} else {
				report.FailedAssertions++
			}
		}
		if cfg.StopOnError && !step.Passed {
			_ = item
		}
	}

	return report, nil
}

func (r *PoolRunner) executeStep(
	ctx context.Context,
	job suiteJob,
	vars map[string]string,
) models.SuiteStepResult {
	req := models.CloneRequestModel(job.item.Request)
	if req.Name == "" {
		req.Name = job.item.Name
	}
	if r.interpolate != nil {
		req = r.interpolate(req, vars)
	}

	stepStart := time.Now()
	resp, err := r.executor.Execute(ctx, req)
	durationMs := time.Since(stepStart).Milliseconds()

	step := models.SuiteStepResult{
		Index:      job.index,
		ItemID:     job.item.ID,
		Name:       req.Name,
		Method:     req.Method,
		URL:        req.URL,
		DurationMs: durationMs,
	}

	if resp != nil {
		step.StatusCode = resp.StatusCode
		step.StatusText = resp.StatusText
		if resp.Duration > 0 {
			step.DurationMs = resp.Duration.Milliseconds()
		}
		if resp.Error != "" {
			step.Error = resp.Error
		}
	}
	if err != nil && step.Error == "" {
		step.Error = err.Error()
	}

	if r.evaluateTests != nil && strings.TrimSpace(req.TestScript) != "" {
		step.Assertions = r.evaluateTests(req.TestScript, req, resp)
		if resp != nil {
			resp.Assertions = step.Assertions
		}
	}

	step.Passed = step.Error == "" && (step.StatusCode == 0 || step.StatusCode < 400)
	if len(step.Assertions) > 0 {
		step.Passed = step.Error == ""
		for _, a := range step.Assertions {
			if !a.Passed {
				step.Passed = false
				break
			}
		}
	}

	return step
}
