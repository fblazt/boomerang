package models

import (
	"fmt"
	"os"
)

// OpenHistoryModal opens the execution history dialog and refreshes history entries from storage.
func (s *AppState) OpenHistoryModal() {
	s.mu.Lock()
	s.ShowHistoryModal = true
	s.HistoryStatusMessage = ""
	s.mu.Unlock()
	s.RefreshHistory()
}

// CloseHistoryModal closes the execution history dialog.
func (s *AppState) CloseHistoryModal() {
	s.Mutate(func(state *AppState) {
		state.ShowHistoryModal = false
	})
}

// ToggleHistoryModal opens or closes the execution history dialog.
func (s *AppState) ToggleHistoryModal() {
	s.mu.RLock()
	open := s.ShowHistoryModal
	s.mu.RUnlock()
	if open {
		s.CloseHistoryModal()
		return
	}
	s.OpenHistoryModal()
}

// RefreshHistory queries the configured HistoryStore using the active filter.
func (s *AppState) RefreshHistory() {
	s.mu.RLock()
	store := s.historyStore
	filter := s.HistoryFilter
	s.mu.RUnlock()

	if store == nil {
		s.RequestRedraw()
		return
	}

	entries, err := store.ListHistory(filter)
	if err != nil {
		return
	}
	totalCount, err := store.CountHistory(HistoryFilter{})
	if err != nil {
		totalCount = len(entries)
	}

	s.mu.Lock()
	s.HistoryEntries = entries
	s.HistoryTotalCount = totalCount
	s.mu.Unlock()

	s.RequestRedraw()
}

// SetHistorySearch updates the history text query and refreshes results.
func (s *AppState) SetHistorySearch(query string) {
	s.mu.Lock()
	s.HistoryFilter.Search = query
	s.mu.Unlock()
	s.RefreshHistory()
}

// SetHistoryMethodFilter updates the active HTTP method filter ("", "GET", "POST", ...) and refreshes.
func (s *AppState) SetHistoryMethodFilter(method string) {
	s.mu.Lock()
	if s.HistoryFilter.Method == method {
		s.HistoryFilter.Method = ""
	} else {
		s.HistoryFilter.Method = method
	}
	s.mu.Unlock()
	s.RefreshHistory()
}

// SetHistoryStatusFilter updates the active status filter ("", "2xx", "4xx", "5xx", "err") and refreshes.
func (s *AppState) SetHistoryStatusFilter(statusClass string) {
	s.mu.Lock()
	if s.HistoryFilter.StatusClass == statusClass {
		s.HistoryFilter.StatusClass = ""
	} else {
		s.HistoryFilter.StatusClass = statusClass
	}
	s.mu.Unlock()
	s.RefreshHistory()
}

// SelectAndReplayHistory loads a historical call's request & response snapshot into the workspace.
func (s *AppState) SelectAndReplayHistory(id string) bool {
	s.mu.Lock()
	var found *HistoryEntry
	for i := range s.HistoryEntries {
		if s.HistoryEntries[i].ID == id {
			cp := s.HistoryEntries[i]
			found = &cp
			break
		}
	}
	if found == nil {
		s.mu.Unlock()
		return false
	}

	s.SelectedHistoryID = found.ID
	s.ActiveCollectionID = ""
	s.ActiveItemID = ""
	s.ActiveDraftID = ""
	s.ActiveRequest = CloneRequestModel(found.Request)
	if found.Response != nil {
		respCopy := *found.Response
		s.ActiveResponse = &respCopy
	} else {
		s.ActiveResponse = nil
	}
	s.ShowHistoryModal = false
	s.mu.Unlock()

	s.RequestRedraw()
	return true
}

// ClearHistoryLog removes all persisted history rows and clears the timeline list.
func (s *AppState) ClearHistoryLog() error {
	s.mu.RLock()
	store := s.historyStore
	s.mu.RUnlock()

	if store != nil {
		if err := store.ClearHistory(); err != nil {
			return err
		}
	}

	s.mu.Lock()
	s.HistoryEntries = nil
	s.HistoryTotalCount = 0
	s.SelectedHistoryID = ""
	s.HistoryStatusMessage = "Cleared history log"
	s.mu.Unlock()

	s.RequestRedraw()
	return nil
}

// ExportHistoryHAR prompts for a save path via PickSaveFile and writes HAR 1.2 JSON via buildHAR.
func (s *AppState) ExportHistoryHAR(buildHAR func([]HistoryEntry) ([]byte, error)) {
	if buildHAR == nil {
		return
	}
	s.mu.RLock()
	entries := make([]HistoryEntry, len(s.HistoryEntries))
	copy(entries, s.HistoryEntries)
	s.mu.RUnlock()

	s.PickSaveFile("Export HAR 1.2 Archive", "boomerang-history.har", []string{"har", "json"}, func(path string) {
		data, err := buildHAR(entries)
		if err != nil {
			s.setHistoryStatusMessage(fmt.Sprintf("HAR export failed: %v", err))
			return
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			s.setHistoryStatusMessage(fmt.Sprintf("Write HAR failed: %v", err))
			return
		}
		s.setHistoryStatusMessage(fmt.Sprintf("Exported %d calls to %s", len(entries), path))
	})
}

func (s *AppState) setHistoryStatusMessage(msg string) {
	s.mu.Lock()
	s.HistoryStatusMessage = msg
	s.mu.Unlock()
	s.RequestRedraw()
}
