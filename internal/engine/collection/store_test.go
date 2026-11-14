package collection

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/fblazt/boomerang/internal/models"
)

func TestStorePersistenceAndTreeOperations(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	if err := store.SeedDefaultsIfEmpty(); err != nil {
		t.Fatalf("SeedDefaultsIfEmpty failed: %v", err)
	}
	cols := store.Collections()
	if len(cols) != 1 || cols[0].Name != "Core API" {
		t.Fatalf("expected 1 seeded collection 'Core API', got %+v", cols)
	}

	// Verify readable JSON file exists on disk
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected 1 collection JSON file on disk, got %d (err=%v)", len(entries), err)
	}

	// Create custom collection + nested folders + items
	billing, err := store.CreateCollection("Billing API")
	if err != nil {
		t.Fatalf("CreateCollection failed: %v", err)
	}

	rootFolder, err := store.CreateFolder(billing.ID, "", "Invoices")
	if err != nil {
		t.Fatalf("CreateFolder root failed: %v", err)
	}
	subFolder, err := store.CreateFolder(billing.ID, rootFolder.ID, "Webhooks")
	if err != nil {
		t.Fatalf("CreateFolder nested failed: %v", err)
	}

	itemA, err := store.SaveItem(billing.ID, subFolder.ID, "Create Invoice", models.RequestModel{
		Method:   "POST",
		URL:      "https://api.example.com/v1/invoices",
		BodyType: "json",
		Body:     `{"amount":4200}`,
		Headers:  []models.KeyValuePair{{Key: "Content-Type", Value: "application/json", Enabled: true}},
	})
	if err != nil {
		t.Fatalf("SaveItem failed: %v", err)
	}

	dup, err := store.DuplicateItem(billing.ID, itemA.ID)
	if err != nil {
		t.Fatalf("DuplicateItem failed: %v", err)
	}
	if dup.Name != "Create Invoice Copy" {
		t.Fatalf("expected duplicate name 'Create Invoice Copy', got %q", dup.Name)
	}

	if err := store.RenameItem(billing.ID, dup.ID, "Refund Invoice"); err != nil {
		t.Fatalf("RenameItem failed: %v", err)
	}
	// Move second item before first
	if err := store.ReorderItem(billing.ID, dup.ID, -1); err != nil {
		t.Fatalf("ReorderItem failed: %v", err)
	}
	// Move item from nested folder to root
	if err := store.MoveItem(billing.ID, itemA.ID, "", 0); err != nil {
		t.Fatalf("MoveItem to root failed: %v", err)
	}

	// Rename collection and ensure stale filename is replaced cleanly
	if err := store.RenameCollection(billing.ID, "Payments V2"); err != nil {
		t.Fatalf("RenameCollection failed: %v", err)
	}

	// Reload in a brand new Store from the same disk directory
	reloaded, err := NewStore(dir)
	if err != nil {
		t.Fatalf("Reload NewStore failed: %v", err)
	}
	reloadedCols := reloaded.Collections()
	if len(reloadedCols) != 2 {
		t.Fatalf("expected 2 collections after reload, got %d", len(reloadedCols))
	}

	var payments models.Collection
	for _, c := range reloadedCols {
		if c.ID == billing.ID {
			payments = c
		}
	}
	if payments.Name != "Payments V2" {
		t.Fatalf("expected renamed collection 'Payments V2', got %q", payments.Name)
	}
	if len(payments.Items) != 1 || payments.Items[0].Name != "Create Invoice" {
		t.Fatalf("expected moved root item 'Create Invoice', got %+v", payments.Items)
	}
	if len(payments.Folders) != 1 || len(payments.Folders[0].Folders) != 1 {
		t.Fatalf("expected nested folder hierarchy preserved, got %+v", payments.Folders)
	}
	if items := payments.Folders[0].Folders[0].Items; len(items) != 1 || items[0].Name != "Refund Invoice" {
		t.Fatalf("expected renamed item in nested folder, got %+v", items)
	}

	// Check stale file was removed when collection was renamed
	diskFiles, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	if len(diskFiles) != 2 {
		t.Fatalf("expected exactly 2 collection files on disk after rename, got %v", diskFiles)
	}

	// Update request payload in place & delete folder/collection checks
	updatedReq := models.CloneRequestModel(itemA.Request)
	updatedReq.URL = "https://api.example.com/v2/invoices"
	if _, err := reloaded.UpdateItemRequest(payments.ID, itemA.ID, updatedReq); err != nil {
		t.Fatalf("UpdateItemRequest failed: %v", err)
	}
	gotItem, ok := reloaded.GetItem(payments.ID, itemA.ID)
	if !ok || gotItem.Request.URL != "https://api.example.com/v2/invoices" {
		t.Fatalf("expected updated URL after UpdateItemRequest, got %+v (ok=%v)", gotItem, ok)
	}

	if err := reloaded.DeleteFolder(payments.ID, rootFolder.ID); err != nil {
		t.Fatalf("DeleteFolder failed: %v", err)
	}
	if err := reloaded.DeleteCollection(payments.ID); err != nil {
		t.Fatalf("DeleteCollection failed: %v", err)
	}
	if left := reloaded.Collections(); len(left) != 1 {
		t.Fatalf("expected 1 collection after DeleteCollection, got %d", len(left))
	}
}

func TestStoreConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	col, err := store.CreateCollection("Concurrent")
	if err != nil {
		t.Fatalf("CreateCollection failed: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := store.SaveItem(col.ID, "", "Req", models.RequestModel{
				Method: "GET",
				URL:    "https://example.com",
			})
			if err != nil {
				t.Errorf("concurrent SaveItem %d failed: %v", idx, err)
			}
		}(i)
	}
	wg.Wait()

	cols := store.Collections()
	_, total := CountTotals(cols)
	if total != 12 {
		t.Fatalf("expected 12 saved items after concurrent writes, got %d", total)
	}
}
