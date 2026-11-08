package primitives

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
)

func TestKVTable_Render(t *testing.T) {
	items := []models.KeyValuePair{
		{Key: "page", Value: "1", Description: "Page number", Enabled: true},
		{Key: "limit", Value: "50", Description: "Page size", Enabled: false},
	}
	changeCount := 0

	tt := ui.NewTester(func(c *ui.Context) {
		KVTable(c, KVTableProps{
			Items: &items,
			OnChange: func() {
				changeCount++
			},
			KeyPrompt:   "Param Key",
			ValuePrompt: "Param Value",
			DescPrompt:  "Param Desc",
			AddLabel:    "+ Add Param",
		})
	}, 800, 300)
	tt.Frame()

	if !tt.HasText("KEY") || !tt.HasText("VALUE") || !tt.HasText("DESCRIPTION") {
		t.Fatal("expected table headers")
	}
	if !tt.HasText("+ Add Param") {
		t.Fatal("expected add button")
	}
}

func TestKVTable_AddAndDelete(t *testing.T) {
	items := []models.KeyValuePair{
		{Key: "test_key", Value: "test_val", Enabled: true},
	}
	changeCount := 0

	tt := ui.NewTester(func(c *ui.Context) {
		KVTable(c, KVTableProps{
			Items: &items,
			OnChange: func() {
				changeCount++
			},
			AddLabel: "+ Add Row",
		})
	}, 800, 300)
	tt.Frame()

	// 1. Click Add Row
	if err := tt.Click("+ Add Row"); err != nil {
		t.Fatalf("failed to click add row: %v", err)
	}
	tt.Frame()

	if len(items) != 2 {
		t.Fatalf("expected 2 items after add, got %d", len(items))
	}
	if changeCount != 1 {
		t.Fatalf("expected changeCount 1, got %d", changeCount)
	}

	// 2. Click Delete (✕)
	if err := tt.Click("✕"); err != nil {
		t.Fatalf("failed to click delete: %v", err)
	}
	tt.Frame()

	if len(items) != 1 {
		t.Fatalf("expected 1 item after delete, got %d", len(items))
	}
	if changeCount != 2 {
		t.Fatalf("expected changeCount 2, got %d", changeCount)
	}
}

func TestKVTable_Toggle(t *testing.T) {
	items := []models.KeyValuePair{
		{Key: "active_key", Value: "active_val", Enabled: true},
	}
	changeCount := 0

	tt := ui.NewTester(func(c *ui.Context) {
		KVTable(c, KVTableProps{
			Items: &items,
			OnChange: func() {
				changeCount++
			},
		})
	}, 800, 300)
	tt.Frame()

	// Initial enabled item shows checkmark
	if !tt.HasText("✓") {
		t.Fatal("expected checkmark for enabled item")
	}

	// Click checkmark to toggle
	if err := tt.Click("✓"); err != nil {
		t.Fatalf("failed to click checkmark: %v", err)
	}
	tt.Frame()

	if items[0].Enabled {
		t.Fatal("expected item to be disabled after clicking checkmark")
	}
	if changeCount != 1 {
		t.Fatalf("expected changeCount 1, got %d", changeCount)
	}
}
