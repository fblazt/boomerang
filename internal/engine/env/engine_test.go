package env

import (
	"fmt"
	"sync"
	"testing"

	"github.com/fblazt/boomerang/internal/models"
)

func TestEngine_InitializationAndDefaults(t *testing.T) {
	vault, err := NewInMemorySecretVault("test-passphrase")
	if err != nil {
		t.Fatal(err)
	}

	engine := NewEngine(vault, nil)
	envs := engine.GetEnvironments()

	if len(envs) != 4 {
		t.Fatalf("expected 4 default environments, got %d", len(envs))
	}

	active := engine.GetActiveEnvironment()
	if active == nil || active.Name != "PROD" {
		t.Fatalf("expected active environment to be PROD, got %+v", active)
	}

	// PROD api_key should be populated in vault
	secret, err := vault.GetSecret("PROD", "api_key")
	if err != nil || secret != "bk_live_94aB882c0f12" {
		t.Fatalf("expected PROD api_key in vault, got %q, err: %v", secret, err)
	}
}

func TestEngine_CRUDEnvironments(t *testing.T) {
	engine := NewEngine(nil, nil)

	// Create
	newEnv := models.Environment{
		Name:  "QA",
		Color: "#F472B6",
		Variables: []models.Variable{
			{Key: "base_url", CurrentValue: "https://qa.internal.net", Type: models.VariableTypeText, Enabled: true},
		},
	}
	if err := engine.CreateEnvironment(newEnv); err != nil {
		t.Fatalf("CreateEnvironment failed: %v", err)
	}

	// Duplicate creation should fail
	if err := engine.CreateEnvironment(newEnv); err == nil {
		t.Fatal("expected error on duplicate environment name")
	}

	// Read
	readEnv, err := engine.GetEnvironment("QA")
	if err != nil || readEnv.Name != "QA" {
		t.Fatalf("failed to read QA environment: %v", err)
	}

	// Update
	readEnv.Color = "#FFFFFF"
	if err := engine.UpdateEnvironment(*readEnv); err != nil {
		t.Fatalf("UpdateEnvironment failed: %v", err)
	}
	updated, _ := engine.GetEnvironment("QA")
	if updated.Color != "#FFFFFF" {
		t.Fatalf("expected color #FFFFFF, got %s", updated.Color)
	}

	// Delete
	if err := engine.DeleteEnvironment("QA"); err != nil {
		t.Fatalf("DeleteEnvironment failed: %v", err)
	}
	if _, err := engine.GetEnvironment("QA"); err == nil {
		t.Fatal("expected error getting deleted environment")
	}
}

func TestEngine_DuplicateEnvironment(t *testing.T) {
	vault, _ := NewInMemorySecretVault("pass")
	engine := NewEngine(vault, nil)

	dup, err := engine.DuplicateEnvironment("PROD", "PROD-COPY")
	if err != nil {
		t.Fatalf("DuplicateEnvironment failed: %v", err)
	}

	if dup.Name != "PROD-COPY" {
		t.Fatalf("expected name PROD-COPY, got %s", dup.Name)
	}
	if len(dup.Variables) != 3 {
		t.Fatalf("expected 3 variables, got %d", len(dup.Variables))
	}

	// Check vault cloned secret
	secret, err := vault.GetSecret("PROD-COPY", "api_key")
	if err != nil || secret != "bk_live_94aB882c0f12" {
		t.Fatalf("expected cloned secret, got %q, err: %v", secret, err)
	}
}

func TestEngine_VariableResolution(t *testing.T) {
	vault, _ := NewInMemorySecretVault("pass")
	engine := NewEngine(vault, nil)

	resolved, err := engine.GetResolvedVariables("PROD")
	if err != nil {
		t.Fatalf("GetResolvedVariables failed: %v", err)
	}

	if resolved["base_url"] != "https://api.internal.net/v1" {
		t.Errorf("expected base_url, got %q", resolved["base_url"])
	}
	if resolved["api_key"] != "bk_live_94aB882c0f12" {
		t.Errorf("expected decrypted api_key from vault, got %q", resolved["api_key"])
	}
	if resolved["timeout"] != "5000" {
		t.Errorf("expected timeout 5000, got %q", resolved["timeout"])
	}

	// Add variable
	newVar := models.Variable{
		Key:     "custom_secret",
		Type:    models.VariableTypeSecret,
		Enabled: true,
	}
	if err := engine.AddVariable("PROD", newVar, "super_unmasked_secret"); err != nil {
		t.Fatalf("AddVariable failed: %v", err)
	}

	resolved2, _ := engine.GetResolvedVariables("PROD")
	if resolved2["custom_secret"] != "super_unmasked_secret" {
		t.Fatalf("expected unmasked secret in resolved map, got %q", resolved2["custom_secret"])
	}

	// Delete variable
	if err := engine.DeleteVariable("PROD", "custom_secret"); err != nil {
		t.Fatalf("DeleteVariable failed: %v", err)
	}
	resolved3, _ := engine.GetResolvedVariables("PROD")
	if _, ok := resolved3["custom_secret"]; ok {
		t.Fatal("expected custom_secret to be deleted")
	}
}

func TestEngine_ConcurrentAccess(t *testing.T) {
	vault, _ := NewInMemorySecretVault("pass")
	engine := NewEngine(vault, nil)

	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(3)
		go func(idx int) {
			defer wg.Done()
			_ = engine.SetActiveEnvironment("STG")
			_ = engine.GetActiveEnvironment()
		}(i)

		go func(idx int) {
			defer wg.Done()
			_, _ = engine.GetResolvedVariables("PROD")
		}(i)

		go func(idx int) {
			defer wg.Done()
			v := models.Variable{
				Key:          fmt.Sprintf("k_%d", idx),
				CurrentValue: "val",
				Type:         models.VariableTypeText,
				Enabled:      true,
			}
			_ = engine.AddVariable("DEV", v, "")
		}(i)
	}
	wg.Wait()
}
