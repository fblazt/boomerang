package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fblazt/boomerang/internal/models"
)

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"abcd", "••••"},
		{"12", "••••"},
		{"bk_live_94aB882c0f12", "bk_live_••••••••0f12"},
		{"sk_test_supersecret1234", "sk_test_••••••••1234"},
		{"plainpassword123", "pla••••••••123"},
		{"short1", "s••••••••1"},
		// Edge cases where prefix and suffix meet or overlap: must NOT leak the secret!
		{"12345_6789", "123••••••••789"},
		{"sk_live_1234", "sk_••••••••234"},
		// Multi-byte UTF-8 rune safety
		{"日本語_秘密_値12345678", "日本語_秘密_••••••••5678"},
	}

	for _, tt := range tests {
		got := MaskSecret(tt.input)
		if got != tt.expected {
			t.Errorf("MaskSecret(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestSecretVault_EncryptDecrypt(t *testing.T) {
	vault, err := NewInMemorySecretVault("super-safe-passphrase")
	if err != nil {
		t.Fatalf("failed to create vault: %v", err)
	}

	envName := "PROD"
	key := "api_key"
	plaintext := "bk_live_sec_9999999999"

	if err := vault.SetSecret(envName, key, plaintext); err != nil {
		t.Fatalf("SetSecret failed: %v", err)
	}

	decrypted, err := vault.GetSecret(envName, key)
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("decrypted %q, expected %q", decrypted, plaintext)
	}

	// Missing secret
	if _, err := vault.GetSecret(envName, "non_existent"); err == nil {
		t.Fatal("expected error for non_existent secret")
	}

	// Missing env
	if _, err := vault.GetSecret("NON_ENV", key); err == nil {
		t.Fatal("expected error for non_existent env")
	}

	// List secrets
	keys := vault.ListSecrets(envName)
	if len(keys) != 1 || keys[0] != key {
		t.Fatalf("unexpected keys: %v", keys)
	}

	// Delete secret
	if err := vault.DeleteSecret(envName, key); err != nil {
		t.Fatalf("DeleteSecret failed: %v", err)
	}
	if _, err := vault.GetSecret(envName, key); err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestSecretVault_FilePersistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vault-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	storageFile := filepath.Join(tempDir, "vault.enc")
	vault1, err := NewSecretVault(storageFile, "persisted-pass")
	if err != nil {
		t.Fatalf("failed to create vault: %v", err)
	}

	if err := vault1.SetSecret("STG", "db_password", "postgres_secret_123"); err != nil {
		t.Fatalf("failed to set secret: %v", err)
	}

	// Reload from same file with new vault instance
	vault2, err := NewSecretVault(storageFile, "persisted-pass")
	if err != nil {
		t.Fatalf("failed to reload vault: %v", err)
	}

	got, err := vault2.GetSecret("STG", "db_password")
	if err != nil {
		t.Fatalf("failed to get reloaded secret: %v", err)
	}
	if got != "postgres_secret_123" {
		t.Fatalf("expected 'postgres_secret_123', got %q", got)
	}

	// Reload with wrong passphrase should fail on decryption
	vaultWrong, err := NewSecretVault(storageFile, "WRONG-pass")
	if err != nil {
		t.Fatalf("failed to create vault with wrong pass: %v", err)
	}
	if _, err := vaultWrong.GetSecret("STG", "db_password"); err == nil {
		t.Fatal("expected error decrypting with wrong key")
	}
}

func TestDotEnv_ExportAndImport(t *testing.T) {
	vault, err := NewInMemorySecretVault("test-pass")
	if err != nil {
		t.Fatal(err)
	}

	env := models.Environment{
		Name: "TEST",
		Variables: []models.Variable{
			{Key: "BASE_URL", CurrentValue: "https://example.com/api", Type: models.VariableTypeText, Enabled: true},
			{Key: "AUTH_TOKEN", CurrentValue: "my_secret_token_12345", Type: models.VariableTypeSecret, Enabled: true},
		},
	}

	_ = vault.SetSecret("TEST", "AUTH_TOKEN", "my_secret_token_12345")

	// Export with masked secrets
	exportedMasked := ExportDotEnv(env, vault, true)
	if !strings.Contains(exportedMasked, `BASE_URL="https://example.com/api"`) {
		t.Fatalf("missing BASE_URL in export: %s", exportedMasked)
	}
	if !strings.Contains(exportedMasked, MaskSecret("my_secret_token_12345")) {
		t.Fatalf("secret not masked in export: %s", exportedMasked)
	}

	// Export with unmasked secrets
	exportedRaw := ExportDotEnv(env, vault, false)
	if !strings.Contains(exportedRaw, `AUTH_TOKEN="my_secret_token_12345"`) {
		t.Fatalf("expected raw secret in unmasked export: %s", exportedRaw)
	}

	// Import back
	importedEnv, secrets, err := ImportDotEnv("IMPORTED", exportedRaw)
	if err != nil {
		t.Fatalf("ImportDotEnv failed: %v", err)
	}

	if importedEnv.Name != "IMPORTED" {
		t.Fatalf("unexpected imported name: %s", importedEnv.Name)
	}
	if len(importedEnv.Variables) != 2 {
		t.Fatalf("expected 2 variables, got %d", len(importedEnv.Variables))
	}
	if secrets["AUTH_TOKEN"] != "my_secret_token_12345" {
		t.Fatalf("expected AUTH_TOKEN secret to be extracted, got %q", secrets["AUTH_TOKEN"])
	}
}

func TestDotEnv_ImportAdvancedFormatting(t *testing.T) {
	content := `
# Environment comment
export FOO=bar # inline comment
GREETING="Hello\nWorld"
SINGLE_QUOTED='single "quoted" value'
# secret
DB_PASS="p@ss#word"
`
	env, secrets, err := ImportDotEnv("ADVANCED", content)
	if err != nil {
		t.Fatalf("ImportDotEnv failed: %v", err)
	}

	varMap := make(map[string]string)
	for _, v := range env.Variables {
		varMap[v.Key] = v.CurrentValue
	}

	if varMap["FOO"] != "bar" {
		t.Errorf("expected FOO to be 'bar' (stripped inline comment), got %q", varMap["FOO"])
	}
	if varMap["GREETING"] != "Hello\nWorld" {
		t.Errorf("expected GREETING to be unescaped with newline, got %q", varMap["GREETING"])
	}
	if varMap["SINGLE_QUOTED"] != `single "quoted" value` {
		t.Errorf("expected SINGLE_QUOTED value, got %q", varMap["SINGLE_QUOTED"])
	}
	if secrets["DB_PASS"] != "p@ss#word" {
		t.Errorf("expected DB_PASS secret to be extracted, got %q", secrets["DB_PASS"])
	}
}
