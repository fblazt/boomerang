package models

import (
	"fmt"
	"strings"
	"time"
)

// SetDraftStore attaches a SQLite-backed scratchpad store and restores any saved drafts.
func (s *AppState) SetDraftStore(store DraftStore) {
	var drafts []ScratchpadDraft
	if store != nil {
		drafts, _ = store.ListDrafts()
	}

	s.mu.Lock()
	s.draftStore = store
	s.Drafts = drafts
	if len(drafts) > 0 && s.ActiveItemID == "" {
		s.ActiveDraftID = drafts[0].ID
		s.ActiveRequest = CloneRequestModel(drafts[0].Request)
	}
	s.mu.Unlock()

	s.RequestRedraw()
}

// DraftStore returns the configured DraftStore, if any.
func (s *AppState) DraftStore() DraftStore {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.draftStore
}

// NewScratchpadDraft creates an independent uncommitted draft request and persists it.
func (s *AppState) NewScratchpadDraft() ScratchpadDraft {
	s.mu.Lock()
	store := s.draftStore
	idx := len(s.Drafts)
	name := "Untitled Request"
	if idx > 0 {
		name = fmt.Sprintf("Untitled Request %d", idx+1)
	}
	draft := ScratchpadDraft{
		ID:    fmt.Sprintf("draft_%d", time.Now().UTC().UnixNano()),
		Name:  name,
		Order: idx,
		Request: RequestModel{
			Name:     name,
			Method:   "GET",
			URL:      "https://api.internal.net/v1/health",
			Params:   []KeyValuePair{},
			Headers:  []KeyValuePair{{Key: "Accept", Value: "application/json", Enabled: true}},
			BodyType: "json",
			AuthType: "none",
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	s.ActiveCollectionID = ""
	s.ActiveItemID = ""
	s.ActiveDraftID = draft.ID
	s.WorkspaceMode = "rest"
	s.ActiveRequest = CloneRequestModel(draft.Request)
	s.Drafts = append(s.Drafts, draft)
	s.mu.Unlock()

	if store != nil {
		if saved, err := store.SaveDraft(draft); err == nil {
			s.upsertDraftSnapshot(saved)
			draft = saved
		}
	}

	s.RequestRedraw()
	return draft
}

// SelectDraft loads an existing uncommitted scratchpad draft into the active editor workspace.
func (s *AppState) SelectDraft(draftID string) bool {
	s.mu.Lock()
	var found *ScratchpadDraft
	for i := range s.Drafts {
		if s.Drafts[i].ID == draftID {
			cp := s.Drafts[i]
			cp.Request = CloneRequestModel(cp.Request)
			found = &cp
			break
		}
	}
	if found == nil {
		s.mu.Unlock()
		return false
	}

	s.ActiveCollectionID = ""
	s.ActiveItemID = ""
	s.ActiveDraftID = found.ID
	s.WorkspaceMode = "rest"
	s.ActiveRequest = CloneRequestModel(found.Request)
	s.mu.Unlock()

	s.RequestRedraw()
	return true
}

// DeleteDraft discards an uncommitted draft from memory and persistent SQLite storage.
func (s *AppState) DeleteDraft(draftID string) error {
	s.mu.Lock()
	store := s.draftStore
	removedIdx := -1
	for i := range s.Drafts {
		if s.Drafts[i].ID == draftID {
			removedIdx = i
			break
		}
	}
	if removedIdx >= 0 {
		s.Drafts = append(s.Drafts[:removedIdx], s.Drafts[removedIdx+1:]...)
	}
	if s.ActiveDraftID == draftID {
		s.ActiveDraftID = ""
	}
	s.mu.Unlock()

	if store != nil {
		if err := store.DeleteDraft(draftID); err != nil {
			return err
		}
	}

	s.RequestRedraw()
	return nil
}

func (s *AppState) prepareDraftSnapshotLocked() ScratchpadDraft {
	now := time.Now().UTC()
	if s.ActiveDraftID == "" {
		s.ActiveDraftID = fmt.Sprintf("draft_%d", now.UnixNano())
	}

	name := strings.TrimSpace(s.ActiveRequest.Name)
	if name == "" || name == "Health Check" {
		name = "Untitled Request"
	}

	var existing ScratchpadDraft
	found := false
	for _, d := range s.Drafts {
		if d.ID == s.ActiveDraftID {
			existing = d
			found = true
			break
		}
	}

	order := len(s.Drafts)
	createdAt := now
	if found {
		order = existing.Order
		createdAt = existing.CreatedAt
		if existing.Name != "" && existing.Name != "Untitled Request" {
			name = existing.Name
		}
	}

	reqCopy := CloneRequestModel(s.ActiveRequest)
	reqCopy.ID = s.ActiveDraftID
	reqCopy.Name = name
	s.ActiveRequest.ID = s.ActiveDraftID
	s.ActiveRequest.Name = name

	return ScratchpadDraft{
		ID:        s.ActiveDraftID,
		Name:      name,
		Order:     order,
		Request:   reqCopy,
		CreatedAt: createdAt,
		UpdatedAt: now,
	}
}

func (s *AppState) upsertDraftSnapshot(saved ScratchpadDraft) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.Drafts {
		if s.Drafts[i].ID == saved.ID {
			s.Drafts[i] = saved
			return
		}
	}
	s.Drafts = append(s.Drafts, saved)
}
