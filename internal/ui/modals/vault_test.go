package modals

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/engine/env"
	"github.com/fblazt/boomerang/internal/models"
)

func TestVaultManagerModal_DirectRender(t *testing.T) {
	state := models.NewAppState(nil)
	state.ActiveVaultEnv = "PROD"
	state.ShowVaultModal = true

	tt := ui.NewTester(func(c *ui.Context) {
		VaultManagerModal(c, state)
	}, 960, 640)
	tt.Frame()

	// Title & Subtitle
	if !tt.HasText("Workspace Environment & Secret Vault Manager") {
		t.Fatal("expected modal title")
	}
	if !tt.HasText("Encrypted secrets storage, variable scopes, and transport policies") {
		t.Fatal("expected modal subtitle")
	}

	// Environment List in sidebar
	if !tt.HasText("ENVIRONMENTS") {
		t.Fatal("expected ENVIRONMENTS section")
	}
	if !tt.HasText("PROD") || !tt.HasText("STG") || !tt.HasText("DEV") || !tt.HasText("LOCAL") {
		t.Fatal("expected default environments listed")
	}

	// Active Tab default
	if !tt.HasText("Variables (3)") {
		t.Fatal("expected Variables (3) tab")
	}
	if !tt.HasText("Transport Policy") {
		t.Fatal("expected Transport Policy tab")
	}

	// Brilliant Board 05 Table columns
	if !tt.HasText("INITIAL VALUE") || !tt.HasText("SCOPE") {
		t.Fatal("expected INITIAL VALUE and SCOPE columns in table header")
	}

	// Export / Import .env actions
	if !tt.HasText("Export .env") || !tt.HasText("Import .env") {
		t.Fatal("expected Export .env and Import .env action buttons")
	}

	// Variable Rows
	if !tt.HasText("base_url") || !tt.HasText("api_key") || !tt.HasText("timeout") {
		t.Fatal("expected default variable rows")
	}

	// Secret masking: api_key is secret, so it should have SHOW button
	if !tt.HasText("SHOW") {
		t.Fatal("expected SHOW button for masked api_key secret")
	}

	// Click SHOW button
	if err := tt.Click("SHOW"); err != nil {
		t.Fatalf("failed to click SHOW: %v", err)
	}
	tt.Frame()

	// Should now show HIDE button
	if !tt.HasText("HIDE") {
		t.Fatal("expected HIDE button after revealing secret")
	}
	if !tt.HasText("bk_live_94aB882c0f12") {
		t.Fatal("expected revealed secret value")
	}

	// Add variable test
	if err := tt.Click("+ Add Variable"); err != nil {
		t.Fatalf("failed to click + Add Variable: %v", err)
	}
	tt.Frame()

	if !tt.HasText("VAR_4") {
		t.Fatal("expected new variable row VAR_4")
	}

	// Switch tab to Transport Policy
	if err := tt.Click("Transport Policy"); err != nil {
		t.Fatalf("failed to click Transport Policy tab: %v", err)
	}
	tt.Frame()

	if !tt.HasText("HTTP PROTOCOL & MULTIPLEXING") {
		t.Fatal("expected HTTP PROTOCOL section in Transport Policy")
	}
	if !tt.HasText("HTTP/2 Multiplexed") || !tt.HasText("HTTP/3 (QUIC UDP 0-RTT)") {
		t.Fatal("expected HTTP protocols")
	}
	if !tt.HasText("Strict SSL Verification:") {
		t.Fatal("expected Strict SSL Verification toggle")
	}

	// Close modal button
	if err := tt.Click("Close"); err != nil {
		t.Fatalf("failed to click Close button: %v", err)
	}
	tt.Frame()

	if state.ShowVaultModal {
		t.Fatal("expected ShowVaultModal to be false after clicking Close")
	}
}

func TestVaultModalHost_ModalDismiss(t *testing.T) {
	state := models.NewAppState(nil)
	state.ShowVaultModal = true

	tt := ui.NewTester(func(c *ui.Context) {
		VaultModalHost(c, state)
	}, 1200, 800)
	tt.Frame()

	if !tt.HasText("Workspace Environment & Secret Vault Manager") {
		t.Fatal("expected modal to be rendered when ShowVaultModal is true")
	}

	// Close via Close button
	if err := tt.Click("Close"); err != nil {
		t.Fatalf("failed to click Close: %v", err)
	}
	tt.Frame()

	if state.ShowVaultModal {
		t.Fatal("expected ShowVaultModal to be false after Close")
	}
}

func TestVaultModalHost_EscapeDismiss(t *testing.T) {
	state := models.NewAppState(nil)
	state.ShowVaultModal = true

	tt := ui.NewTester(func(c *ui.Context) {
		VaultModalHost(c, state)
	}, 1200, 800)
	tt.Frame()

	if !tt.HasText("Workspace Environment & Secret Vault Manager") {
		t.Fatal("expected modal to be rendered when ShowVaultModal is true")
	}

	// Press Escape to dismiss modal
	tt.Key(0, ui.KeyEscape)
	tt.Frame()

	if state.ShowVaultModal {
		t.Fatal("expected ShowVaultModal to be false after pressing Escape")
	}
}

func TestVaultModalHost_BackdropClickDismiss(t *testing.T) {
	state := models.NewAppState(nil)
	state.ShowVaultModal = true

	tt := ui.NewTester(func(c *ui.Context) {
		VaultModalHost(c, state)
	}, 1200, 800)
	tt.Frame()

	if !tt.HasText("Workspace Environment & Secret Vault Manager") {
		t.Fatal("expected modal to be rendered when ShowVaultModal is true")
	}

	// Click on backdrop outside modal (e.g. top-left corner at 10, 10)
	tt.ClickAt(10, 10)
	tt.Frame()

	if state.ShowVaultModal {
		t.Fatal("expected ShowVaultModal to be false after clicking backdrop")
	}
}

func TestVaultManagerModal_WithEngine(t *testing.T) {
	state := models.NewAppState(nil)
	vault, err := env.NewInMemorySecretVault("modal-vault-pass")
	if err != nil {
		t.Fatal(err)
	}
	engine := env.NewEngine(vault, models.DefaultEnvironments())
	state.SetEnvEngine(engine)

	state.ActiveVaultEnv = "PROD"
	state.ShowVaultModal = true

	tt := ui.NewTester(func(c *ui.Context) {
		VaultManagerModal(c, state)
	}, 1000, 700)
	tt.Frame()

	// Click Export .env
	if err := tt.Click("Export .env"); err != nil {
		t.Fatalf("failed to click Export .env: %v", err)
	}
	tt.Frame()

	clip := tt.Clipboard()
	if !strings.Contains(clip, "base_url") || !strings.Contains(clip, "https://api.internal.net/v1") {
		t.Fatalf("expected exported .env in clipboard, got: %s", clip)
	}

	// Click SHOW on api_key: should decrypt from vault
	if err := tt.Click("SHOW"); err != nil {
		t.Fatalf("failed to click SHOW: %v", err)
	}
	tt.Frame()

	if !tt.HasText("bk_live_94aB882c0f12") {
		t.Fatal("expected decrypted secret from vault to be displayed")
	}
}

func TestVaultManagerModal_TransportPolicyCertPickers(t *testing.T) {
	state := models.NewAppState(nil)
	state.ActiveVaultEnv = "PROD"
	state.ShowVaultModal = true

	var pickedTitles []string
	state.SetFilePicker(func(title string, extensions []string, onPicked func(path string)) {
		pickedTitles = append(pickedTitles, title)
		if strings.Contains(title, "Root CA") {
			onPicked("/tmp/certs/corp-root-ca.pem")
			return
		}
		if strings.Contains(title, "Client Certificate") {
			onPicked("/tmp/certs/service-client.pem")
			return
		}
		onPicked("/tmp/certs/service-client.key")
	})

	tt := ui.NewTester(func(c *ui.Context) {
		VaultManagerModal(c, state)
	}, 960, 720)
	tt.Frame()

	if err := tt.Click("Transport Policy"); err != nil {
		t.Fatalf("failed to click Transport Policy tab: %v", err)
	}
	tt.Frame()

	if err := tt.Click("Select Root CA certificate (click to browse PEM/CRT)"); err != nil {
		t.Fatalf("failed to click Pinned Root CA picker: %v", err)
	}
	tt.Frame()

	if !tt.HasText("corp-root-ca.pem") {
		t.Fatal("expected selected Root CA filename to render")
	}

	if err := tt.Click("No client certificate configured (click to select PEM/P12)"); err != nil {
		t.Fatalf("failed to click mTLS client cert picker: %v", err)
	}
	tt.Frame()

	if !tt.HasText("service-client.pem") || !tt.HasText("mTLS ENFORCED") {
		t.Fatal("expected mTLS certificate filename and mTLS ENFORCED badge")
	}

	if err := tt.Click("Optional separate private key (click to browse KEY/PEM)"); err != nil {
		t.Fatalf("failed to click mTLS private key picker: %v", err)
	}
	tt.Frame()

	if !tt.HasText("service-client.key") {
		t.Fatal("expected selected private key filename to render")
	}

	if len(pickedTitles) != 3 {
		t.Fatalf("expected 3 file picker calls, got %d (%v)", len(pickedTitles), pickedTitles)
	}

	// Clear buttons should clear configured paths
	if err := tt.Click("Clear"); err != nil {
		t.Fatalf("failed to click Clear: %v", err)
	}
	tt.Frame()

	state.View(func(s *models.AppState) {
		for _, e := range s.Environments {
			if e.Name == "PROD" && e.TransportPolicy.PinnedRootCA != "" {
				t.Fatalf("expected PinnedRootCA cleared, got %q", e.TransportPolicy.PinnedRootCA)
			}
		}
	})
}
