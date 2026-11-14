package collection

import (
	"fmt"
	"strings"
	"time"

	"github.com/fblazt/boomerang/internal/engine/storage"
	"github.com/fblazt/boomerang/internal/models"
)

// FindFolder locates a folder by ID anywhere in col and returns its pointer along with its containing slice.
func FindFolder(col *models.Collection, folderID string) (*models.Folder, *[]models.Folder, int, bool) {
	if col == nil || folderID == "" {
		return nil, nil, -1, false
	}
	return findFolderInSlice(&col.Folders, folderID)
}

func findFolderInSlice(folders *[]models.Folder, folderID string) (*models.Folder, *[]models.Folder, int, bool) {
	for i := range *folders {
		if (*folders)[i].ID == folderID {
			return &(*folders)[i], folders, i, true
		}
		if found, slice, idx, ok := findFolderInSlice(&(*folders)[i].Folders, folderID); ok {
			return found, slice, idx, true
		}
	}
	return nil, nil, -1, false
}

// FindItem locates a CollectionItem by ID anywhere inside col and returns its slice and index.
func FindItem(col *models.Collection, itemID string) (*[]models.CollectionItem, int, bool) {
	if col == nil || itemID == "" {
		return nil, -1, false
	}
	for i := range col.Items {
		if col.Items[i].ID == itemID {
			return &col.Items, i, true
		}
	}
	return findItemInFolders(col.Folders, itemID)
}

func findItemInFolders(folders []models.Folder, itemID string) (*[]models.CollectionItem, int, bool) {
	for i := range folders {
		for j := range folders[i].Items {
			if folders[i].Items[j].ID == itemID {
				return &folders[i].Items, j, true
			}
		}
		if slice, idx, ok := findItemInFolders(folders[i].Folders, itemID); ok {
			return slice, idx, true
		}
	}
	return nil, -1, false
}

// AddFolder appends a new folder either at the collection root (parentFolderID == "") or inside parentFolderID.
func AddFolder(col *models.Collection, parentFolderID, name string, now time.Time) (models.Folder, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return models.Folder{}, fmt.Errorf("folder name cannot be empty")
	}

	target := &col.Folders
	if parentFolderID != "" {
		parent, _, _, ok := FindFolder(col, parentFolderID)
		if !ok {
			return models.Folder{}, fmt.Errorf("parent folder %q not found", parentFolderID)
		}
		target = &parent.Folders
	}

	folder := models.Folder{
		ID:        storage.NewID("fld"),
		Name:      trimmed,
		Order:     len(*target),
		CreatedAt: now,
		UpdatedAt: now,
	}
	*target = append(*target, folder)
	col.UpdatedAt = now
	return models.CloneFolder(folder), nil
}

// RenameFolder renames a nested or root folder inside col.
func RenameFolder(col *models.Collection, folderID, newName string, now time.Time) error {
	trimmed := strings.TrimSpace(newName)
	if trimmed == "" {
		return fmt.Errorf("folder name cannot be empty")
	}
	folder, _, _, ok := FindFolder(col, folderID)
	if !ok {
		return fmt.Errorf("folder %q not found", folderID)
	}
	folder.Name = trimmed
	folder.UpdatedAt = now
	col.UpdatedAt = now
	return nil
}

// DeleteFolder removes a folder and its subtree from col.
func DeleteFolder(col *models.Collection, folderID string, now time.Time) error {
	_, slice, idx, ok := FindFolder(col, folderID)
	if !ok {
		return fmt.Errorf("folder %q not found", folderID)
	}
	*slice = append((*slice)[:idx], (*slice)[idx+1:]...)
	reindexFolders(*slice)
	col.UpdatedAt = now
	return nil
}

// AddItem saves a new request item into the collection root (folderID == "") or specified folder.
func AddItem(col *models.Collection, folderID, name string, req models.RequestModel, now time.Time) (models.CollectionItem, error) {
	items, _, err := itemSliceTarget(col, folderID)
	if err != nil {
		return models.CollectionItem{}, err
	}

	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		trimmed = strings.TrimSpace(req.Name)
	}
	if trimmed == "" {
		trimmed = "Untitled Request"
	}

	cloned := models.CloneRequestModel(req)
	itemID := storage.NewID("req")
	cloned.ID = itemID
	cloned.Name = trimmed

	item := models.CollectionItem{
		ID:        itemID,
		Name:      trimmed,
		Order:     len(*items),
		Request:   cloned,
		CreatedAt: now,
		UpdatedAt: now,
	}
	*items = append(*items, item)
	col.UpdatedAt = now
	return models.CloneCollectionItem(item), nil
}

// UpdateItemRequest replaces the saved request payload of an existing CollectionItem.
func UpdateItemRequest(col *models.Collection, itemID string, req models.RequestModel, now time.Time) (models.CollectionItem, error) {
	items, idx, ok := FindItem(col, itemID)
	if !ok {
		return models.CollectionItem{}, fmt.Errorf("request item %q not found", itemID)
	}
	cloned := models.CloneRequestModel(req)
	cloned.ID = (*items)[idx].ID
	if strings.TrimSpace(cloned.Name) == "" {
		cloned.Name = (*items)[idx].Name
	} else {
		(*items)[idx].Name = strings.TrimSpace(cloned.Name)
	}
	(*items)[idx].Request = cloned
	(*items)[idx].UpdatedAt = now
	col.UpdatedAt = now
	return models.CloneCollectionItem((*items)[idx]), nil
}

// RenameItem updates both item.Name and item.Request.Name.
func RenameItem(col *models.Collection, itemID, newName string, now time.Time) error {
	trimmed := strings.TrimSpace(newName)
	if trimmed == "" {
		return fmt.Errorf("request name cannot be empty")
	}
	items, idx, ok := FindItem(col, itemID)
	if !ok {
		return fmt.Errorf("request item %q not found", itemID)
	}
	(*items)[idx].Name = trimmed
	(*items)[idx].Request.Name = trimmed
	(*items)[idx].UpdatedAt = now
	col.UpdatedAt = now
	return nil
}

// DuplicateItem inserts a deep copy immediately after itemID in the same parent container.
func DuplicateItem(col *models.Collection, itemID string, now time.Time) (models.CollectionItem, error) {
	items, idx, ok := FindItem(col, itemID)
	if !ok {
		return models.CollectionItem{}, fmt.Errorf("request item %q not found", itemID)
	}
	dup := models.CloneCollectionItem((*items)[idx])
	newID := storage.NewID("req")
	dup.ID = newID
	dup.Request.ID = newID
	dup.Name = dup.Name + " Copy"
	dup.Request.Name = dup.Name
	dup.CreatedAt = now
	dup.UpdatedAt = now

	insertAt := idx + 1
	*items = append(*items, models.CollectionItem{})
	copy((*items)[insertAt+1:], (*items)[insertAt:])
	(*items)[insertAt] = dup
	reindexItems(*items)
	col.UpdatedAt = now
	return models.CloneCollectionItem(dup), nil
}

// ReorderItem shifts an item within its sibling slice by delta positions (-1 moves up, +1 moves down).
func ReorderItem(col *models.Collection, itemID string, delta int, now time.Time) error {
	if delta == 0 {
		return nil
	}
	items, idx, ok := FindItem(col, itemID)
	if !ok {
		return fmt.Errorf("request item %q not found", itemID)
	}
	target := idx + delta
	if target < 0 {
		target = 0
	}
	if target >= len(*items) {
		target = len(*items) - 1
	}
	if target == idx {
		return nil
	}
	item := (*items)[idx]
	*items = append((*items)[:idx], (*items)[idx+1:]...)
	*items = append(*items, models.CollectionItem{})
	copy((*items)[target+1:], (*items)[target:])
	(*items)[target] = item
	reindexItems(*items)
	col.UpdatedAt = now
	return nil
}

// MoveItem moves an existing request item to targetFolderID ("" for collection root) at targetIdx.
func MoveItem(col *models.Collection, itemID, targetFolderID string, targetIdx int, now time.Time) error {
	srcSlice, srcIdx, ok := FindItem(col, itemID)
	if !ok {
		return fmt.Errorf("request item %q not found", itemID)
	}
	dstSlice, _, err := itemSliceTarget(col, targetFolderID)
	if err != nil {
		return err
	}

	item := (*srcSlice)[srcIdx]
	*srcSlice = append((*srcSlice)[:srcIdx], (*srcSlice)[srcIdx+1:]...)
	reindexItems(*srcSlice)

	if dstSlice == srcSlice && srcIdx < targetIdx {
		targetIdx--
	}
	if targetIdx < 0 || targetIdx > len(*dstSlice) {
		targetIdx = len(*dstSlice)
	}
	*dstSlice = append(*dstSlice, models.CollectionItem{})
	copy((*dstSlice)[targetIdx+1:], (*dstSlice)[targetIdx:])
	(*dstSlice)[targetIdx] = item
	reindexItems(*dstSlice)

	item.UpdatedAt = now
	(*dstSlice)[targetIdx].UpdatedAt = now
	col.UpdatedAt = now
	return nil
}

// DeleteItem deletes a request item from col.
func DeleteItem(col *models.Collection, itemID string, now time.Time) error {
	items, idx, ok := FindItem(col, itemID)
	if !ok {
		return fmt.Errorf("request item %q not found", itemID)
	}
	*items = append((*items)[:idx], (*items)[idx+1:]...)
	reindexItems(*items)
	col.UpdatedAt = now
	return nil
}

// FlattenItems returns every request item in col in display traversal order.
func FlattenItems(col models.Collection) []models.CollectionItem {
	var out []models.CollectionItem
	for _, item := range col.Items {
		out = append(out, models.CloneCollectionItem(item))
	}
	collectFolderItems(col.Folders, &out)
	return out
}

func collectFolderItems(folders []models.Folder, out *[]models.CollectionItem) {
	for _, f := range folders {
		for _, item := range f.Items {
			*out = append(*out, models.CloneCollectionItem(item))
		}
		collectFolderItems(f.Folders, out)
	}
}

// CountTotals returns total collections and total request items across cols.
func CountTotals(cols []models.Collection) (collectionCount int, requestCount int) {
	collectionCount = len(cols)
	for _, col := range cols {
		requestCount += len(FlattenItems(col))
	}
	return collectionCount, requestCount
}

func itemSliceTarget(col *models.Collection, folderID string) (*[]models.CollectionItem, *time.Time, error) {
	if col == nil {
		return nil, nil, fmt.Errorf("collection cannot be nil")
	}
	if folderID == "" {
		return &col.Items, &col.UpdatedAt, nil
	}
	folder, _, _, ok := FindFolder(col, folderID)
	if !ok {
		return nil, nil, fmt.Errorf("folder %q not found", folderID)
	}
	return &folder.Items, &folder.UpdatedAt, nil
}

func reindexItems(items []models.CollectionItem) {
	for i := range items {
		items[i].Order = i
	}
}

func reindexFolders(folders []models.Folder) {
	for i := range folders {
		folders[i].Order = i
	}
}
