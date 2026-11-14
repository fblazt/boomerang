package models

import (
	"context"
	"fmt"
)

// SetSuiteRunner attaches the concurrent collection suite runner engine.
func (s *AppState) SetSuiteRunner(runner SuiteRunner) {
	s.mu.Lock()
	s.suiteRunner = runner
	s.mu.Unlock()
}

// OpenRunnerModal opens the Collection Suite Runner modal and ensures a valid default collection and thread count.
func (s *AppState) OpenRunnerModal() {
	s.Mutate(func(state *AppState) {
		state.ShowRunnerModal = true
		if state.RunnerConcurrency < 1 || state.RunnerConcurrency > 32 {
			state.RunnerConcurrency = 4
		}
		if state.RunnerFilter == "" {
			state.RunnerFilter = "all"
		}
		state.ensureRunnerCollectionSelectedLocked()
	})
}

// ToggleRunnerModal toggles the Suite Runner modal visibility.
func (s *AppState) ToggleRunnerModal() {
	s.mu.RLock()
	show := s.ShowRunnerModal
	s.mu.RUnlock()
	if show {
		s.CloseRunnerModal()
		return
	}
	s.OpenRunnerModal()
}

// CloseRunnerModal closes the Suite Runner modal.
func (s *AppState) CloseRunnerModal() {
	s.Mutate(func(state *AppState) {
		state.ShowRunnerModal = false
	})
}

// SetRunnerCollection selects which collection will be executed by the Suite Runner.
func (s *AppState) SetRunnerCollection(collectionID string) {
	s.Mutate(func(state *AppState) {
		state.RunnerCollectionID = collectionID
		state.RunnerStatusMessage = ""
	})
}

// SetRunnerConcurrency sets the worker pool size clamped to [1, 32].
func (s *AppState) SetRunnerConcurrency(threads int) {
	if threads < 1 {
		threads = 1
	}
	if threads > 32 {
		threads = 32
	}
	s.Mutate(func(state *AppState) {
		state.RunnerConcurrency = threads
	})
}

// ToggleRunnerStopOnError toggles halting suite execution after the first failed request.
func (s *AppState) ToggleRunnerStopOnError() {
	s.Mutate(func(state *AppState) {
		state.RunnerStopOnError = !state.RunnerStopOnError
	})
}

// SetRunnerFilter sets the result list filter in the Suite Runner summary ("all", "passed", "failed").
func (s *AppState) SetRunnerFilter(filter string) {
	switch filter {
	case "all", "passed", "failed":
	default:
		filter = "all"
	}
	s.Mutate(func(state *AppState) {
		state.RunnerFilter = filter
	})
}

// SetRunnerStatusMessage updates the informational status bar inside the Suite Runner modal.
func (s *AppState) SetRunnerStatusMessage(msg string) {
	s.Mutate(func(state *AppState) {
		state.RunnerStatusMessage = msg
	})
}

// StartSuiteRun launches a concurrent collection suite execution in the background.
func (s *AppState) StartSuiteRun() {
	s.mu.Lock()
	runner := s.suiteRunner
	if runner == nil || s.RunnerRunning {
		if runner == nil {
			s.RunnerStatusMessage = "Suite runner engine not configured."
		}
		s.mu.Unlock()
		s.RequestRedraw()
		return
	}
	s.ensureRunnerCollectionSelectedLocked()
	var targetCol *Collection
	for i := range s.Collections {
		if s.Collections[i].ID == s.RunnerCollectionID {
			c := CloneCollection(s.Collections[i])
			targetCol = &c
			break
		}
	}
	if targetCol == nil {
		s.RunnerStatusMessage = "Select a collection with requests to run."
		s.mu.Unlock()
		s.RequestRedraw()
		return
	}

	concurrency := s.RunnerConcurrency
	if concurrency < 1 {
		concurrency = 4
	}
	if concurrency > 32 {
		concurrency = 32
	}
	cfg := SuiteRunConfig{
		CollectionID:   targetCol.ID,
		CollectionName: targetCol.Name,
		Concurrency:    concurrency,
		StopOnError:    s.RunnerStopOnError,
		Variables:      s.getActiveVariablesLocked(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.runnerCancel = cancel
	s.RunnerRunning = true
	s.RunnerReport = nil
	s.RunnerProgress = SuiteRunProgress{}
	s.RunnerStatusMessage = fmt.Sprintf("Running %s across %d worker(s)...", targetCol.Name, concurrency)
	s.mu.Unlock()

	s.RequestRedraw()

	go func() {
		report, err := runner.RunSuite(ctx, *targetCol, cfg, func(prog SuiteRunProgress) {
			s.mu.Lock()
			s.RunnerProgress = prog
			s.mu.Unlock()
			s.RequestRedraw()
		})

		s.mu.Lock()
		s.RunnerRunning = false
		s.runnerCancel = nil
		if err != nil {
			s.RunnerStatusMessage = "Suite run failed: " + err.Error()
		} else {
			rc := report
			s.RunnerReport = &rc
			if report.Cancelled {
				s.RunnerStatusMessage = fmt.Sprintf("Cancelled after %d/%d requests.", len(report.Steps), report.TotalRequests)
			} else {
				s.RunnerStatusMessage = fmt.Sprintf("Completed %d requests: %d passed, %d failed in %d ms.",
					report.TotalRequests, report.PassedRequests, report.FailedRequests, report.TotalDurationMs)
			}
		}
		s.mu.Unlock()
		s.RequestRedraw()
	}()
}

// CancelSuiteRun aborts an active collection suite execution.
func (s *AppState) CancelSuiteRun() {
	s.mu.Lock()
	cancel := s.runnerCancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *AppState) ensureRunnerCollectionSelectedLocked() {
	if len(s.Collections) == 0 {
		s.RunnerCollectionID = ""
		return
	}
	for _, col := range s.Collections {
		if col.ID == s.RunnerCollectionID {
			return
		}
	}
	if s.ActiveCollectionID != "" {
		for _, col := range s.Collections {
			if col.ID == s.ActiveCollectionID {
				s.RunnerCollectionID = col.ID
				return
			}
		}
	}
	s.RunnerCollectionID = s.Collections[0].ID
}
