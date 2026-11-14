package collection

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fblazt/boomerang/internal/engine/storage"
	"github.com/fblazt/boomerang/internal/models"
)

// Store manages workspace collections and persists each collection as readable JSON on local disk.
type Store struct {
	mu          sync.RWMutex
	dir         string
	collections []models.Collection
	paths       map[string]string
}

// NewStore opens or initializes a collection store rooted at dir and loads any existing collections.
func NewStore(dir string) (*Store, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("collection directory cannot be empty")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create collection dir %s: %w", dir, err)
	}

	s := &Store{
		dir:   dir,
		paths: make(map[string]string),
	}
	if err := s.Reload(); err != nil {
		return nil, err
	}
	return s, nil
}

// NewInMemoryStore returns an in-memory collection store without filesystem persistence.
func NewInMemoryStore() *Store {
	return &Store{
		paths: make(map[string]string),
	}
}

// Reload discards cached collections and reads all *.json collection files from disk.
func (s *Store) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.dir == "" {
		return nil
	}

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return fmt.Errorf("read collections dir %s: %w", s.dir, err)
	}

	cols := make([]models.Collection, 0, len(entries))
	paths := make(map[string]string)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		fullPath := filepath.Join(s.dir, entry.Name())
		var col models.Collection
		if err := storage.ReadJSON(fullPath, &col); err != nil {
			return err
		}
		if col.ID == "" {
			col.ID = storage.NewID("col")
		}
		cols = append(cols, col)
		paths[col.ID] = fullPath
	}

	sortCollections(cols)
	s.collections = cols
	s.paths = paths
	return nil
}

// SeedDefaultsIfEmpty writes starter collections once when no collections exist on disk.
func (s *Store) SeedDefaultsIfEmpty() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.collections) > 0 {
		return nil
	}

	defaults := models.DefaultCollections()
	for _, col := range defaults {
		if err := s.writeCollectionLocked(col); err != nil {
			return err
		}
		s.collections = append(s.collections, models.CloneCollection(col))
	}
	sortCollections(s.collections)
	return nil
}

// Collections returns deep copies of all collections in display order.
func (s *Store) Collections() []models.Collection {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]models.Collection, len(s.collections))
	for i, col := range s.collections {
		out[i] = models.CloneCollection(col)
	}
	return out
}

// GetItem looks up a single request item across collections.
func (s *Store) GetItem(collectionID, itemID string) (models.CollectionItem, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.collections {
		if collectionID != "" && s.collections[i].ID != collectionID {
			continue
		}
		if items, idx, ok := FindItem(&s.collections[i], itemID); ok {
			return models.CloneCollectionItem((*items)[idx]), true
		}
	}
	return models.CollectionItem{}, false
}

// CreateCollection creates a new root collection and persists it to disk.
func (s *Store) CreateCollection(name string) (models.Collection, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return models.Collection{}, fmt.Errorf("collection name cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	col := models.Collection{
		ID:        storage.NewID("col"),
		Name:      trimmed,
		Order:     len(s.collections),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.writeCollectionLocked(col); err != nil {
		return models.Collection{}, err
	}
	s.collections = append(s.collections, col)
	sortCollections(s.collections)
	return models.CloneCollection(col), nil
}

// ImportCollection persists a pre-parsed external collection onto disk and adds it to the store.
func (s *Store) ImportCollection(col models.Collection) (models.Collection, error) {
	if strings.TrimSpace(col.Name) == "" {
		col.Name = "Imported Collection"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	cloned := models.CloneCollection(col)
	if cloned.ID == "" {
		cloned.ID = storage.NewID("col")
	}
	cloned.Order = len(s.collections)
	if cloned.CreatedAt.IsZero() {
		cloned.CreatedAt = now
	}
	cloned.UpdatedAt = now

	if err := s.writeCollectionLocked(cloned); err != nil {
		return models.Collection{}, err
	}
	s.collections = append(s.collections, cloned)
	sortCollections(s.collections)
	return models.CloneCollection(cloned), nil
}

// RenameCollection renames a collection and refreshes its readable JSON filename on disk.
func (s *Store) RenameCollection(collectionID, name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("collection name cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findCollectionIdxLocked(collectionID)
	if idx < 0 {
		return fmt.Errorf("collection %q not found", collectionID)
	}
	s.collections[idx].Name = trimmed
	s.collections[idx].UpdatedAt = time.Now().UTC()
	return s.writeCollectionLocked(s.collections[idx])
}

// DeleteCollection deletes a collection from memory and removes its JSON file from disk.
func (s *Store) DeleteCollection(collectionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findCollectionIdxLocked(collectionID)
	if idx < 0 {
		return fmt.Errorf("collection %q not found", collectionID)
	}
	if oldPath, ok := s.paths[collectionID]; ok {
		if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove collection file %s: %w", oldPath, err)
		}
		delete(s.paths, collectionID)
	}
	s.collections = append(s.collections[:idx], s.collections[idx+1:]...)
	for i := range s.collections {
		s.collections[i].Order = i
	}
	return nil
}

// CreateFolder creates a folder inside collectionID (optionally nested inside parentFolderID) and saves.
func (s *Store) CreateFolder(collectionID, parentFolderID, name string) (models.Folder, error) {
	var created models.Folder
	err := s.mutateCollection(collectionID, func(col *models.Collection) error {
		var err error
		created, err = AddFolder(col, parentFolderID, name, time.Now().UTC())
		return err
	})
	return created, err
}

// RenameFolder renames a folder inside collectionID and saves.
func (s *Store) RenameFolder(collectionID, folderID, newName string) error {
	return s.mutateCollection(collectionID, func(col *models.Collection) error {
		return RenameFolder(col, folderID, newName, time.Now().UTC())
	})
}

// DeleteFolder deletes a folder inside collectionID and saves.
func (s *Store) DeleteFolder(collectionID, folderID string) error {
	return s.mutateCollection(collectionID, func(col *models.Collection) error {
		return DeleteFolder(col, folderID, time.Now().UTC())
	})
}

// SaveItem adds a new saved request item to collectionID (and optional folderID) and saves.
func (s *Store) SaveItem(collectionID, folderID, name string, req models.RequestModel) (models.CollectionItem, error) {
	var saved models.CollectionItem
	err := s.mutateCollection(collectionID, func(col *models.Collection) error {
		var err error
		saved, err = AddItem(col, folderID, name, req, time.Now().UTC())
		return err
	})
	return saved, err
}

// UpdateItemRequest updates the saved request inside an existing CollectionItem and saves.
func (s *Store) UpdateItemRequest(collectionID, itemID string, req models.RequestModel) (models.CollectionItem, error) {
	var updated models.CollectionItem
	err := s.mutateCollection(collectionID, func(col *models.Collection) error {
		var err error
		updated, err = UpdateItemRequest(col, itemID, req, time.Now().UTC())
		return err
	})
	return updated, err
}

// RenameItem renames an existing request item inside collectionID and saves.
func (s *Store) RenameItem(collectionID, itemID, newName string) error {
	return s.mutateCollection(collectionID, func(col *models.Collection) error {
		return RenameItem(col, itemID, newName, time.Now().UTC())
	})
}

// DuplicateItem clones an existing request item inside collectionID right below the original and saves.
func (s *Store) DuplicateItem(collectionID, itemID string) (models.CollectionItem, error) {
	var dup models.CollectionItem
	err := s.mutateCollection(collectionID, func(col *models.Collection) error {
		var err error
		dup, err = DuplicateItem(col, itemID, time.Now().UTC())
		return err
	})
	return dup, err
}

// ReorderItem shifts an item up (delta < 0) or down (delta > 0) within its sibling list and saves.
func (s *Store) ReorderItem(collectionID, itemID string, delta int) error {
	return s.mutateCollection(collectionID, func(col *models.Collection) error {
		return ReorderItem(col, itemID, delta, time.Now().UTC())
	})
}

// MoveItem moves an item to targetFolderID ("" for root) at targetIdx and saves.
func (s *Store) MoveItem(collectionID, itemID, targetFolderID string, targetIdx int) error {
	return s.mutateCollection(collectionID, func(col *models.Collection) error {
		return MoveItem(col, itemID, targetFolderID, targetIdx, time.Now().UTC())
	})
}

// DeleteItem deletes a request item from collectionID and saves.
func (s *Store) DeleteItem(collectionID, itemID string) error {
	return s.mutateCollection(collectionID, func(col *models.Collection) error {
		return DeleteItem(col, itemID, time.Now().UTC())
	})
}

// Dir returns the filesystem directory where collection JSON files live.
func (s *Store) Dir() string {
	return s.dir
}

func (s *Store) mutateCollection(collectionID string, fn func(*models.Collection) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findCollectionIdxLocked(collectionID)
	if idx < 0 {
		return fmt.Errorf("collection %q not found", collectionID)
	}

	copyCol := models.CloneCollection(s.collections[idx])
	if err := fn(&copyCol); err != nil {
		return err
	}
	if err := s.writeCollectionLocked(copyCol); err != nil {
		return err
	}
	s.collections[idx] = copyCol
	return nil
}

func (s *Store) findCollectionIdxLocked(collectionID string) int {
	for i := range s.collections {
		if s.collections[i].ID == collectionID {
			return i
		}
	}
	return -1
}

func (s *Store) writeCollectionLocked(col models.Collection) error {
	if s.dir == "" {
		return nil
	}
	filename := fmt.Sprintf("%s-%s.json", storage.SafeSlug(col.Name), col.ID)
	targetPath := filepath.Join(s.dir, filename)

	if err := storage.WriteJSONAtomic(targetPath, col); err != nil {
		return err
	}
	if prevPath, ok := s.paths[col.ID]; ok && prevPath != targetPath {
		_ = os.Remove(prevPath)
	}
	s.paths[col.ID] = targetPath
	return nil
}

func sortCollections(cols []models.Collection) {
	slices.SortStableFunc(cols, func(a, b models.Collection) int {
		if c := cmp.Compare(a.Order, b.Order); c != 0 {
			return c
		}
		return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
}
