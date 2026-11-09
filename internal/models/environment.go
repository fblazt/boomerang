package models

// VariableType defines whether a variable is plaintext or an encrypted secret.
type VariableType string

const (
	VariableTypeText   VariableType = "text"
	VariableTypeSecret VariableType = "secret"
)

// Variable represents a workspace environment key-value pair.
type Variable struct {
	Key          string       `json:"key"`
	InitialValue string       `json:"initial_value"`
	CurrentValue string       `json:"current_value"`
	Type         VariableType `json:"type"` // "text" or "secret"
	Enabled      bool         `json:"enabled"`
	Description  string       `json:"description,omitempty"`
}

// TransportPolicy configures HTTP protocol, TLS verification, and mTLS credentials.
type TransportPolicy struct {
	Protocol      string `json:"protocol"` // "HTTP/2", "HTTP/3", "HTTP/1.1"
	StrictSSL     bool   `json:"strict_ssl"`
	PinnedRootCA  string `json:"pinned_root_ca,omitempty"`
	ClientCertPEM string `json:"client_cert_pem,omitempty"`
	ClientKeyPEM  string `json:"client_key_pem,omitempty"`
}

// Environment represents an environment profile (e.g., PROD, STG, DEV, LOCAL).
type Environment struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Color           string          `json:"color,omitempty"`
	Variables       []Variable      `json:"variables"`
	TransportPolicy TransportPolicy `json:"transport_policy"`
	IsDefault       bool            `json:"is_default,omitempty"`
}

// DefaultEnvironments returns standard default workspace environments.
func DefaultEnvironments() []Environment {
	return []Environment{
		{
			ID:    "env_prod",
			Name:  "PROD",
			Color: "#34D399",
			Variables: []Variable{
				{Key: "base_url", InitialValue: "https://api.internal.net/v1", CurrentValue: "https://api.internal.net/v1", Type: VariableTypeText, Enabled: true},
				{Key: "api_key", InitialValue: "bk_live_94aB882c0f12", CurrentValue: "bk_live_94aB882c0f12", Type: VariableTypeSecret, Enabled: true},
				{Key: "timeout", InitialValue: "5000", CurrentValue: "5000", Type: VariableTypeText, Enabled: true},
			},
			TransportPolicy: TransportPolicy{
				Protocol:  "HTTP/2",
				StrictSSL: true,
			},
			IsDefault: true,
		},
		{
			ID:    "env_stg",
			Name:  "STG",
			Color: "#FBBF24",
			Variables: []Variable{
				{Key: "base_url", InitialValue: "https://stg-api.internal.net/v1", CurrentValue: "https://stg-api.internal.net/v1", Type: VariableTypeText, Enabled: true},
				{Key: "api_key", InitialValue: "bk_test_33dE441a9901", CurrentValue: "bk_test_33dE441a9901", Type: VariableTypeSecret, Enabled: true},
				{Key: "timeout", InitialValue: "10000", CurrentValue: "10000", Type: VariableTypeText, Enabled: true},
			},
			TransportPolicy: TransportPolicy{
				Protocol:  "HTTP/2",
				StrictSSL: true,
			},
		},
		{
			ID:    "env_dev",
			Name:  "DEV",
			Color: "#38BDF8",
			Variables: []Variable{
				{Key: "base_url", InitialValue: "http://localhost:8080/v1", CurrentValue: "http://localhost:8080/v1", Type: VariableTypeText, Enabled: true},
				{Key: "api_key", InitialValue: "dev_key_local", CurrentValue: "dev_key_local", Type: VariableTypeSecret, Enabled: true},
				{Key: "timeout", InitialValue: "30000", CurrentValue: "30000", Type: VariableTypeText, Enabled: true},
			},
			TransportPolicy: TransportPolicy{
				Protocol:  "HTTP/1.1",
				StrictSSL: false,
			},
		},
		{
			ID:    "env_local",
			Name:  "LOCAL",
			Color: "#A855F7",
			Variables: []Variable{
				{Key: "base_url", InitialValue: "http://127.0.0.1:3000", CurrentValue: "http://127.0.0.1:3000", Type: VariableTypeText, Enabled: true},
				{Key: "debug", InitialValue: "true", CurrentValue: "true", Type: VariableTypeText, Enabled: true},
			},
			TransportPolicy: TransportPolicy{
				Protocol:  "HTTP/1.1",
				StrictSSL: false,
			},
		},
	}
}
