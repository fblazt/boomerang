package models

import (
	"fmt"
	"strings"
)

// SetImporterService attaches the universal API spec & cURL importer.
func (s *AppState) SetImporterService(svc ImporterService) {
	s.mu.Lock()
	s.importer = svc
	s.mu.Unlock()
}

// OpenImportModal opens the single-step Universal Import modal.
func (s *AppState) OpenImportModal() {
	s.Mutate(func(state *AppState) {
		state.ShowImportModal = true
		state.ImportError = ""
		state.ImportStatusMessage = ""
	})
}

// ToggleImportModal toggles the Universal Import modal visibility.
func (s *AppState) ToggleImportModal() {
	s.mu.RLock()
	show := s.ShowImportModal
	s.mu.RUnlock()
	if show {
		s.CloseImportModal()
		return
	}
	s.OpenImportModal()
}

// CloseImportModal closes the Universal Import modal.
func (s *AppState) CloseImportModal() {
	s.Mutate(func(state *AppState) {
		state.ShowImportModal = false
	})
}

// SetImportRawText stores pasted text and runs immediate auto-detection and preview parsing.
func (s *AppState) SetImportRawText(raw string) {
	s.mu.Lock()
	svc := s.importer
	s.ImportRawText = raw
	s.ImportFilePath = ""
	s.ImportError = ""
	if strings.TrimSpace(raw) == "" {
		s.ImportDetectedFormat = ImportFormatUnknown
		s.ImportPreviewData = nil
		s.ImportTargetName = ""
		s.mu.Unlock()
		s.RequestRedraw()
		return
	}
	if svc == nil {
		s.ImportError = "Importer service not configured."
		s.mu.Unlock()
		s.RequestRedraw()
		return
	}
	detected := svc.DetectFormat(raw)
	s.ImportDetectedFormat = detected
	prev, err := svc.Parse(raw, "paste")
	if err != nil {
		s.ImportPreviewData = nil
		s.ImportTargetName = ""
		s.ImportError = err.Error()
	} else {
		pc := prev
		s.ImportPreviewData = &pc
		s.ImportTargetName = pc.Collection.Name
		s.ImportDetectedFormat = pc.Format
		s.ImportError = ""
	}
	s.mu.Unlock()
	s.RequestRedraw()
}

// LoadImportPath parses a file or Bruno directory from disk and updates the modal preview.
func (s *AppState) LoadImportPath(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	s.mu.Lock()
	svc := s.importer
	s.ImportFilePath = path
	if svc == nil {
		s.ImportError = "Importer service not configured."
		s.mu.Unlock()
		s.RequestRedraw()
		return
	}
	prev, err := svc.ParseFile(path)
	if err != nil {
		s.ImportPreviewData = nil
		s.ImportError = err.Error()
	} else {
		pc := prev
		s.ImportPreviewData = &pc
		s.ImportTargetName = pc.Collection.Name
		s.ImportDetectedFormat = pc.Format
		s.ImportError = ""
	}
	s.mu.Unlock()
	s.RequestRedraw()
}

// SetImportTargetName overrides the imported collection display name before confirmation.
func (s *AppState) SetImportTargetName(name string) {
	s.Mutate(func(state *AppState) {
		state.ImportTargetName = name
	})
}

// ConfirmImport commits the parsed collection preview into workspace storage and selects its first request.
func (s *AppState) ConfirmImport() error {
	s.mu.Lock()
	if s.ImportPreviewData == nil {
		s.ImportError = "Paste a cURL/spec or choose a file to import."
		s.mu.Unlock()
		s.RequestRedraw()
		return fmt.Errorf("nothing to import")
	}

	col := CloneCollection(s.ImportPreviewData.Collection)
	if trimmed := strings.TrimSpace(s.ImportTargetName); trimmed != "" {
		col.Name = trimmed
	}
	store := s.collectionStore
	s.mu.Unlock()

	if store != nil {
		saved, err := store.ImportCollection(col)
		if err != nil {
			s.Mutate(func(state *AppState) {
				state.ImportError = err.Error()
			})
			return err
		}
		col = saved
	}

	var firstItem *CollectionItem
	if len(col.Items) > 0 {
		it := CloneCollectionItem(col.Items[0])
		firstItem = &it
	} else if first := findFirstFolderItem(col.Folders); first != nil {
		it := CloneCollectionItem(*first)
		firstItem = &it
	}

	s.Mutate(func(state *AppState) {
		if store != nil {
			state.Collections = store.Collections()
		} else {
			col.Order = len(state.Collections)
			state.Collections = append(state.Collections, col)
		}
		state.ActiveCollectionID = col.ID
		state.SelectedFolderID = ""
		if firstItem != nil {
			state.ActiveItemID = firstItem.ID
			state.ActiveRequest = CloneRequestModel(firstItem.Request)
			state.ActiveDraftID = ""
			state.WorkspaceMode = "rest"
		}
		state.ShowImportModal = false
		state.ImportError = ""
		state.ImportStatusMessage = fmt.Sprintf("Imported %q (%d requests)", col.Name, s.countImportRequests(col))
	})
	return nil
}

func (s *AppState) countImportRequests(col Collection) int {
	total := len(col.Items)
	var walk func([]Folder)
	walk = func(folders []Folder) {
		for _, f := range folders {
			total += len(f.Items)
			walk(f.Folders)
		}
	}
	walk(col.Folders)
	return total
}

func findFirstFolderItem(folders []Folder) *CollectionItem {
	for i := range folders {
		if len(folders[i].Items) > 0 {
			return &folders[i].Items[0]
		}
		if sub := findFirstFolderItem(folders[i].Folders); sub != nil {
			return sub
		}
	}
	return nil
}
