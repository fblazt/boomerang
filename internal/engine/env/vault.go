package env

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fblazt/boomerang/internal/models"
)

// MaskSecret masks a secret string for safe UI presentation (e.g. bk_live_••••••••0f12).
func MaskSecret(secret string) string {
	if secret == "" {
		return ""
	}
	l := len(secret)
	if l <= 4 {
		return "••••"
	}

	// Detect prefix with underscores like bk_live_, sk_live_, ghp_, etc.
	if idx := strings.LastIndex(secret, "_"); idx != -1 && idx+1 <= l-4 {
		prefix := secret[:idx+1]
		suffix := secret[l-4:]
		return prefix + "••••••••" + suffix
	}

	if l <= 8 {
		return secret[:1] + "••••••••" + secret[l-1:]
	}

	return secret[:3] + "••••••••" + secret[l-3:]
}

// SecretVault manages local encrypted secrets using AES-256-GCM.
type SecretVault struct {
	mu          sync.RWMutex
	key         []byte                       // 32-byte AES key
	storagePath string                       // optional file path for persistence
	secrets     map[string]map[string]string // envName -> key -> encrypted_hex
}

// NewInMemorySecretVault creates a vault without file persistence.
func NewInMemorySecretVault(passphrase string) (*SecretVault, error) {
	if passphrase == "" {
		passphrase = "boomerang-default-vault-key"
	}
	hash := sha256.Sum256([]byte(passphrase))
	return &SecretVault{
		key:     hash[:],
		secrets: make(map[string]map[string]string),
	}, nil
}

// NewSecretVault creates a vault backed by an encrypted file on disk.
func NewSecretVault(storagePath string, passphrase string) (*SecretVault, error) {
	vault, err := NewInMemorySecretVault(passphrase)
	if err != nil {
		return nil, err
	}
	vault.storagePath = storagePath

	if storagePath != "" {
		if err := vault.Load(); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load vault: %w", err)
		}
	}
	return vault, nil
}

// encrypt encrypts plaintext using AES-256-GCM and returns hex-encoded (nonce + ciphertext).
func (v *SecretVault) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(v.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// decrypt decrypts hex-encoded (nonce + ciphertext) using AES-256-GCM.
func (v *SecretVault) decrypt(hexStr string) (string, error) {
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(v.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return string(plaintext), nil
}

// SetSecret stores an encrypted secret for the given environment and key.
func (v *SecretVault) SetSecret(envName, key, plaintext string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	enc, err := v.encrypt(plaintext)
	if err != nil {
		return fmt.Errorf("encryption error: %w", err)
	}

	if v.secrets[envName] == nil {
		v.secrets[envName] = make(map[string]string)
	}
	v.secrets[envName][key] = enc

	if v.storagePath != "" {
		_ = v.saveLocked()
	}
	return nil
}

// GetSecret retrieves and decrypts a secret value.
func (v *SecretVault) GetSecret(envName, key string) (string, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	envSecrets, ok := v.secrets[envName]
	if !ok {
		return "", fmt.Errorf("environment %q not found in vault", envName)
	}

	enc, ok := envSecrets[key]
	if !ok {
		return "", fmt.Errorf("secret %q not found in environment %q", key, envName)
	}

	return v.decrypt(enc)
}

// DeleteSecret removes a secret from the vault.
func (v *SecretVault) DeleteSecret(envName, key string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if envSecrets, ok := v.secrets[envName]; ok {
		delete(envSecrets, key)
		if v.storagePath != "" {
			_ = v.saveLocked()
		}
	}
	return nil
}

// ListSecrets returns the list of secret keys stored for the given environment.
func (v *SecretVault) ListSecrets(envName string) []string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	envSecrets, ok := v.secrets[envName]
	if !ok {
		return nil
	}

	keys := make([]string, 0, len(envSecrets))
	for k := range envSecrets {
		keys = append(keys, k)
	}
	return keys
}

// Save persists the encrypted vault to disk.
func (v *SecretVault) Save() error {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.saveLocked()
}

func (v *SecretVault) saveLocked() error {
	if v.storagePath == "" {
		return nil
	}

	dir := filepath.Dir(v.storagePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(v.secrets, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(v.storagePath, data, 0600)
}

// Load reads and parses the encrypted vault from disk.
func (v *SecretVault) Load() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.storagePath == "" {
		return nil
	}

	data, err := os.ReadFile(v.storagePath)
	if err != nil {
		return err
	}

	var loaded map[string]map[string]string
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}

	v.secrets = loaded
	return nil
}

// ExportDotEnv exports an Environment to standard .env format.
func ExportDotEnv(env models.Environment, vault *SecretVault, maskSecrets bool) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Boomerang Environment: %s\n", env.Name))
	if env.ID != "" {
		sb.WriteString(fmt.Sprintf("# ID: %s\n", env.ID))
	}
	sb.WriteString("\n")

	for _, v := range env.Variables {
		val := v.CurrentValue
		if val == "" {
			val = v.InitialValue
		}

		if v.Type == models.VariableTypeSecret {
			if vault != nil {
				if dec, err := vault.GetSecret(env.Name, v.Key); err == nil && dec != "" {
					val = dec
				}
			}
			if maskSecrets {
				val = MaskSecret(val)
			}
			sb.WriteString(fmt.Sprintf("# secret\n%s=%q\n", v.Key, val))
		} else {
			sb.WriteString(fmt.Sprintf("%s=%q\n", v.Key, val))
		}
	}

	return sb.String()
}

// ImportDotEnv parses standard .env content into a models.Environment and extracts secret values.
func ImportDotEnv(envName string, content string) (*models.Environment, map[string]string, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var vars []models.Variable
	secrets := make(map[string]string)

	isNextSecret := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Comment inspection
		if strings.HasPrefix(line, "#") {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "secret") {
				isNextSecret = true
			}
			continue
		}

		// Strip export prefix if present (e.g. 'export KEY=VAL')
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// Strip quotes if present
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}

		// Detect secret by name heuristic or prior comment
		upperKey := strings.ToUpper(key)
		isSecret := isNextSecret ||
			strings.Contains(upperKey, "SECRET") ||
			strings.Contains(upperKey, "KEY") ||
			strings.Contains(upperKey, "TOKEN") ||
			strings.Contains(upperKey, "PASSWORD") ||
			strings.Contains(upperKey, "AUTH") ||
			strings.Contains(upperKey, "PRIVATE")

		varType := models.VariableTypeText
		if isSecret {
			varType = models.VariableTypeSecret
			secrets[key] = val
		}

		vars = append(vars, models.Variable{
			Key:          key,
			InitialValue: val,
			CurrentValue: val,
			Type:         varType,
			Enabled:      true,
		})

		isNextSecret = false
	}

	env := &models.Environment{
		ID:        "env_" + strings.ToLower(envName),
		Name:      envName,
		Color:     "#38BDF8",
		Variables: vars,
		TransportPolicy: models.TransportPolicy{
			Protocol:  "HTTP/2",
			StrictSSL: true,
		},
	}

	return env, secrets, nil
}
