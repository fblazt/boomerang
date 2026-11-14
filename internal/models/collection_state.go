package models

import (
	"strings"
)

// SetCollectionStore attaches a collection store and synchronizes Collections in AppState.
func (s *AppState) SetCollectionStore(store CollectionStore) {
	s.mu.Lock()
	s.collectionStore = store
	if store != nil {
		s.Collections = store.Collections()
	}
	s.mu.Unlock()
	s.RequestRedraw()
}

// CollectionStore returns the configured CollectionStore, if any.
func (s *AppState) CollectionStore() CollectionStore {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.collectionStore
}

// SelectCollectionItem loads a saved collection request into ActiveRequest for editing/execution.
func (s *AppState) SelectCollectionItem(collectionID, itemID string) bool {
	s.mu.Lock()
	var found *CollectionItem
	for _, col := range s.Collections {
		if collectionID != "" && col.ID != collectionID {
			continue
		}
		if item := findCollectionItem(col.Items, col.Folders, itemID); item != nil {
			collectionID = col.ID
			found = item
			break
		}
	}
	if found == nil {
		s.mu.Unlock()
		return false
	}

	s.ActiveCollectionID = collectionID
	s.ActiveItemID = found.ID
	s.ActiveDraftID = ""
	s.WorkspaceMode = "rest"
	s.ActiveRequest = CloneRequestModel(found.Request)
	s.mu.Unlock()

	s.RequestRedraw()
	return true
}

// SaveActiveRequest saves edits back to the linked collection item or saves an ad-hoc request to the workspace collection.
func (s *AppState) SaveActiveRequest() error {
	s.mu.Lock()
	store := s.collectionStore
	colID := s.ActiveCollectionID
	itemID := s.ActiveItemID
	folderID := s.SelectedFolderID
	reqCopy := CloneRequestModel(s.ActiveRequest)
	var existingCols []Collection
	if len(s.Collections) > 0 {
		existingCols = s.Collections
	}
	s.mu.Unlock()

	if store == nil {
		return nil
	}

	if colID != "" && itemID != "" {
		if _, err := store.UpdateItemRequest(colID, itemID, reqCopy); err == nil {
			s.syncCollectionsFromStore(store)
			return nil
		}
	}

	if colID == "" {
		if len(existingCols) > 0 {
			colID = existingCols[0].ID
		} else {
			createdCol, err := store.CreateCollection("Workspace")
			if err != nil {
				return err
			}
			colID = createdCol.ID
		}
	}

	name := strings.TrimSpace(reqCopy.Name)
	if name == "" {
		name = "Untitled Request"
	}
	saved, err := store.SaveItem(colID, folderID, name, reqCopy)
	if err != nil {
		// SelectedFolderID may belong to a different collection; retry at collection root.
		saved, err = store.SaveItem(colID, "", name, reqCopy)
		if err != nil {
			return err
		}
	}

	s.mu.Lock()
	promotedDraftID := s.ActiveDraftID
	draftStore := s.draftStore
	s.Collections = store.Collections()
	s.ActiveCollectionID = colID
	s.ActiveItemID = saved.ID
	s.ActiveDraftID = ""
	s.ActiveRequest = CloneRequestModel(saved.Request)
	if promotedDraftID != "" {
		for i := range s.Drafts {
			if s.Drafts[i].ID == promotedDraftID {
				s.Drafts = append(s.Drafts[:i], s.Drafts[i+1:]...)
				break
			}
		}
	}
	s.mu.Unlock()

	if promotedDraftID != "" && draftStore != nil {
		_ = draftStore.DeleteDraft(promotedDraftID)
	}

	s.RequestRedraw()
	return nil
}

// CreateCollection adds a new root collection and expands it.
func (s *AppState) CreateCollection(name string) (Collection, error) {
	store := s.CollectionStore()
	if store == nil {
		return Collection{}, nil
	}
	col, err := store.CreateCollection(name)
	if err != nil {
		return Collection{}, err
	}
	s.syncCollectionsFromStore(store)
	return col, nil
}

// CreateFolder creates a folder inside collectionID (optionally nested in parentFolderID).
func (s *AppState) CreateFolder(collectionID, parentFolderID, name string) (Folder, error) {
	store := s.CollectionStore()
	if store == nil {
		return Folder{}, nil
	}
	if collectionID == "" {
		s.mu.RLock()
		if len(s.Collections) > 0 {
			collectionID = s.Collections[0].ID
		}
		s.mu.RUnlock()
	}
	folder, err := store.CreateFolder(collectionID, parentFolderID, name)
	if err != nil {
		return Folder{}, err
	}
	s.mu.Lock()
	delete(s.CollapsedNodes, collectionID)
	if parentFolderID != "" {
		delete(s.CollapsedNodes, parentFolderID)
	}
	s.Collections = store.Collections()
	s.mu.Unlock()
	s.RequestRedraw()
	return folder, nil
}

// CreateRequestInCollection appends a new request inside collectionID/folderID and focuses it in the editor.
func (s *AppState) CreateRequestInCollection(collectionID, folderID, name string) (CollectionItem, error) {
	store := s.CollectionStore()
	if store == nil {
		return CollectionItem{}, nil
	}
	if collectionID == "" {
		s.mu.RLock()
		if len(s.Collections) > 0 {
			collectionID = s.Collections[0].ID
		}
		s.mu.RUnlock()
		if collectionID == "" {
			created, err := store.CreateCollection("Workspace")
			if err != nil {
				return CollectionItem{}, err
			}
			collectionID = created.ID
		}
	}
	if strings.TrimSpace(name) == "" {
		name = "New Request"
	}
	item, err := store.SaveItem(collectionID, folderID, name, RequestModel{
		Name:     name,
		Method:   "GET",
		URL:      "https://api.internal.net/v1/resource",
		Params:   []KeyValuePair{},
		Headers:  []KeyValuePair{{Key: "Accept", Value: "application/json", Enabled: true}},
		BodyType: "none",
		AuthType: "none",
	})
	if err != nil {
		return CollectionItem{}, err
	}

	s.mu.Lock()
	delete(s.CollapsedNodes, collectionID)
	if folderID != "" {
		delete(s.CollapsedNodes, folderID)
	}
	s.Collections = store.Collections()
	s.ActiveCollectionID = collectionID
	s.ActiveItemID = item.ID
	s.ActiveDraftID = ""
	s.WorkspaceMode = "rest"
	s.ActiveRequest = CloneRequestModel(item.Request)
	s.mu.Unlock()

	s.RequestRedraw()
	return item, nil
}

// DuplicateCollectionItem duplicates an existing saved request immediately below the original.
func (s *AppState) DuplicateCollectionItem(collectionID, itemID string) (CollectionItem, error) {
	store := s.CollectionStore()
	if store == nil {
		return CollectionItem{}, nil
	}
	dup, err := store.DuplicateItem(collectionID, itemID)
	if err != nil {
		return CollectionItem{}, err
	}
	s.syncCollectionsFromStore(store)
	return dup, nil
}

// ReorderCollectionItem shifts an item up (-1) or down (+1) within its sibling folder/root list.
func (s *AppState) ReorderCollectionItem(collectionID, itemID string, delta int) error {
	store := s.CollectionStore()
	if store == nil {
		return nil
	}
	if err := store.ReorderItem(collectionID, itemID, delta); err != nil {
		return err
	}
	s.syncCollectionsFromStore(store)
	return nil
}

// MoveCollectionItem moves a request item into targetFolderID ("" for collection root).
func (s *AppState) MoveCollectionItem(collectionID, itemID, targetFolderID string, targetIdx int) error {
	store := s.CollectionStore()
	if store == nil {
		return nil
	}
	if err := store.MoveItem(collectionID, itemID, targetFolderID, targetIdx); err != nil {
		return err
	}
	s.syncCollectionsFromStore(store)
	return nil
}

// DeleteCollectionItem removes a request item from its collection.
func (s *AppState) DeleteCollectionItem(collectionID, itemID string) error {
	store := s.CollectionStore()
	if store == nil {
		return nil
	}
	if err := store.DeleteItem(collectionID, itemID); err != nil {
		return err
	}
	s.mu.Lock()
	if s.ActiveItemID == itemID {
		s.ActiveItemID = ""
		s.ActiveCollectionID = ""
	}
	s.Collections = store.Collections()
	s.mu.Unlock()
	s.RequestRedraw()
	return nil
}

// DeleteCollectionFolder removes a folder and its subtree from collectionID.
func (s *AppState) DeleteCollectionFolder(collectionID, folderID string) error {
	store := s.CollectionStore()
	if store == nil {
		return nil
	}
	if err := store.DeleteFolder(collectionID, folderID); err != nil {
		return err
	}
	s.syncCollectionsFromStore(store)
	return nil
}

// BeginRenameNode starts inline renaming for a collection, folder, or request item.
func (s *AppState) BeginRenameNode(nodeType, collectionID, nodeID, currentName string) {
	s.Mutate(func(state *AppState) {
		state.RenamingNodeType = nodeType
		state.RenamingCollection = collectionID
		state.RenamingNodeID = nodeID
		state.RenameDraft = currentName
	})
}

// SetRenameDraft updates the live inline rename text buffer without extra disk writes.
func (s *AppState) SetRenameDraft(text string) {
	s.mu.Lock()
	s.RenameDraft = text
	s.mu.Unlock()
}

// CommitRenameNode saves the pending inline rename to the store.
func (s *AppState) CommitRenameNode() error {
	s.mu.RLock()
	store := s.collectionStore
	nodeType := s.RenamingNodeType
	colID := s.RenamingCollection
	nodeID := s.RenamingNodeID
	newName := strings.TrimSpace(s.RenameDraft)
	s.mu.RUnlock()

	if nodeID == "" || store == nil || newName == "" {
		s.CancelRenameNode()
		return nil
	}

	var err error
	switch nodeType {
	case "collection":
		err = store.RenameCollection(nodeID, newName)
	case "folder":
		err = store.RenameFolder(colID, nodeID, newName)
	case "item":
		err = store.RenameItem(colID, nodeID, newName)
	}
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.Collections = store.Collections()
	if nodeType == "item" && s.ActiveItemID == nodeID {
		s.ActiveRequest.Name = newName
	}
	s.clearRenameLocked()
	s.mu.Unlock()
	s.RequestRedraw()
	return nil
}

// CancelRenameNode exits inline rename mode without applying edits.
func (s *AppState) CancelRenameNode() {
	s.Mutate(func(state *AppState) {
		state.clearRenameLocked()
	})
}

// ToggleNodeExpanded collapses or expands a collection or folder node in the sidebar tree.
func (s *AppState) ToggleNodeExpanded(nodeID string) {
	s.Mutate(func(state *AppState) {
		if state.CollapsedNodes == nil {
			state.CollapsedNodes = make(map[string]bool)
		}
		if state.CollapsedNodes[nodeID] {
			delete(state.CollapsedNodes, nodeID)
		} else {
			state.CollapsedNodes[nodeID] = true
		}
	})
}

// IsNodeExpanded returns true unless nodeID has been explicitly collapsed.
func (s *AppState) IsNodeExpanded(nodeID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.CollapsedNodes[nodeID]
}

// SetCollectionFilter updates the sidebar collections search filter.
func (s *AppState) SetCollectionFilter(query string) {
	s.Mutate(func(state *AppState) {
		state.CollectionFilter = query
	})
}

// SetSelectedFolder sets the active target folder for new requests/subfolders.
func (s *AppState) SetSelectedFolder(collectionID, folderID string) {
	s.Mutate(func(state *AppState) {
		state.ActiveCollectionID = collectionID
		state.SelectedFolderID = folderID
	})
}

// CollectionCounts returns total collections and total saved requests across AppState.Collections.
func (s *AppState) CollectionCounts() (collectionsCount, requestsCount int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	collectionsCount = len(s.Collections)
	for _, col := range s.Collections {
		requestsCount += countFolderItems(col.Items, col.Folders)
	}
	return collectionsCount, requestsCount
}

func (s *AppState) syncCollectionsFromStore(store CollectionStore) {
	s.mu.Lock()
	s.Collections = store.Collections()
	s.mu.Unlock()
	s.RequestRedraw()
}

func (s *AppState) clearRenameLocked() {
	s.RenamingNodeType = ""
	s.RenamingCollection = ""
	s.RenamingNodeID = ""
	s.RenameDraft = ""
}

func findCollectionItem(items []CollectionItem, folders []Folder, itemID string) *CollectionItem {
	for i := range items {
		if items[i].ID == itemID {
			cp := CloneCollectionItem(items[i])
			return &cp
		}
	}
	for _, f := range folders {
		if found := findCollectionItem(f.Items, f.Folders, itemID); found != nil {
			return found
		}
	}
	return nil
}

func countFolderItems(items []CollectionItem, folders []Folder) int {
	total := len(items)
	for _, f := range folders {
		total += countFolderItems(f.Items, f.Folders)
	}
	return total
}
