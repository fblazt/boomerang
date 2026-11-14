package models

import (
	"net/url"
	"strings"
	"time"
)

// Telemetry holds network breakdown timing and connection diagnostics.
type Telemetry struct {
	DNS         int64  `json:"dns"`                    // DNS lookup duration in ms
	TCP         int64  `json:"tcp"`                    // TCP connect duration in ms
	TLS         int64  `json:"tls"`                    // TLS handshake duration in ms
	TTFB        int64  `json:"ttfb"`                   // Time to first byte in ms
	Download    int64  `json:"download"`               // Content download duration in ms
	Total       int64  `json:"total"`                  // Total roundtrip time in ms
	TotalMicros int64  `json:"total_micros,omitempty"` // Total roundtrip time in microseconds
	RemoteAddr  string `json:"remote_addr,omitempty"`  // Remote IP:Port dialed
	TLSVersion  string `json:"tls_version,omitempty"`  // Negotiated TLS version (e.g. TLS 1.3)
	TLSCipher   string `json:"tls_cipher,omitempty"`   // Negotiated TLS cipher suite
	ConnReused  bool   `json:"conn_reused,omitempty"`  // Whether pooled TCP/TLS connection was reused
}

// KeyValuePair represents query parameters, headers, or form fields.
type KeyValuePair struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description,omitempty"`
	Secret      bool   `json:"secret,omitempty"`
}

// RequestModel represents an in-memory HTTP or API request draft.
type RequestModel struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Method        string         `json:"method"`
	URL           string         `json:"url"`
	Params        []KeyValuePair `json:"params"`
	Headers       []KeyValuePair `json:"headers"`
	BodyType      string         `json:"body_type"` // none, json, text, form-data, urlencoded, binary
	Body          string         `json:"body"`
	FormData      []KeyValuePair `json:"form_data,omitempty"`
	URLEncoded    []KeyValuePair `json:"urlencoded,omitempty"`
	AuthType      string         `json:"auth_type"` // none, bearer, basic, apikey
	Token         string         `json:"token,omitempty"`
	BasicUser     string         `json:"basic_user,omitempty"`
	BasicPassword string         `json:"basic_password,omitempty"`
	APIKeyKey       string           `json:"apikey_key,omitempty"`
	APIKeyValue     string           `json:"apikey_value,omitempty"`
	APIKeyIn        string           `json:"apikey_in,omitempty"` // header, query
	StrictSSL       *bool            `json:"strict_ssl,omitempty"`
	FollowRedirects *bool            `json:"follow_redirects,omitempty"`
	EnableHTTP2     *bool            `json:"enable_http2,omitempty"`
	TransportPolicy *TransportPolicy `json:"transport_policy,omitempty"`
	TestScript      string           `json:"test_script,omitempty"`
}

// EffectiveFollowRedirects returns whether HTTP 3xx redirects should be followed (defaults to true).
func (r RequestModel) EffectiveFollowRedirects() bool {
	if r.FollowRedirects != nil {
		return *r.FollowRedirects
	}
	return true
}

// EffectiveTransportPolicy merges per-request overrides over an environment TransportPolicy.
func (r RequestModel) EffectiveTransportPolicy(envPolicy *TransportPolicy) TransportPolicy {
	policy := TransportPolicy{
		Protocol:  "HTTP/2",
		StrictSSL: true,
	}
	if envPolicy != nil {
		policy = *envPolicy
		if policy.Protocol == "" {
			policy.Protocol = "HTTP/2"
		}
	}
	if r.StrictSSL != nil {
		policy.StrictSSL = *r.StrictSSL
	}
	if r.EnableHTTP2 != nil && policy.Protocol != "HTTP/3" {
		if *r.EnableHTTP2 {
			policy.Protocol = "HTTP/2"
		} else {
			policy.Protocol = "HTTP/1.1"
		}
	}
	return policy
}

// ResponseModel represents the execution response.
type ResponseModel struct {
	StatusCode int               `json:"status_code"`
	StatusText string            `json:"status_text"`
	Duration   time.Duration     `json:"duration"`
	SizeBytes  int64             `json:"size_bytes"`
	Proto      string            `json:"proto"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Telemetry  Telemetry             `json:"telemetry"`
	Assertions []TestAssertionResult `json:"assertions,omitempty"`
	Error      string                `json:"error,omitempty"`
}

// SyncParamsToURL rebuilds the rawURL's query string using the provided params slice.
// Only params where Enabled is true and Key is non-empty are included.
// It is token-safe (preserves {{variable}} tokens in the base path and fragments).
func SyncParamsToURL(rawURL string, params []KeyValuePair) string {
	basePart := rawURL
	fragmentPart := ""

	if idx := strings.Index(basePart, "#"); idx != -1 {
		fragmentPart = basePart[idx:]
		basePart = basePart[:idx]
	}

	if idx := strings.Index(basePart, "?"); idx != -1 {
		basePart = basePart[:idx]
	}

	var pairs []string
	for _, p := range params {
		if !p.Enabled || strings.TrimSpace(p.Key) == "" {
			continue
		}
		k := url.QueryEscape(p.Key)
		v := url.QueryEscape(p.Value)
		// Preserve template variable tokens {{...}} without escaping braces
		k = strings.ReplaceAll(strings.ReplaceAll(k, "%7B%7B", "{{"), "%7D%7D", "}}")
		v = strings.ReplaceAll(strings.ReplaceAll(v, "%7B%7B", "{{"), "%7D%7D", "}}")
		pairs = append(pairs, k+"="+v)
	}

	if len(pairs) == 0 {
		return basePart + fragmentPart
	}

	return basePart + "?" + strings.Join(pairs, "&") + fragmentPart
}

// SyncURLToParams extracts query parameters from a rawURL into KeyValuePair structs.
// It safely parses parameters without mangling template tokens in the path.
func SyncURLToParams(rawURL string) []KeyValuePair {
	basePart := rawURL
	if idx := strings.Index(basePart, "#"); idx != -1 {
		basePart = basePart[:idx]
	}

	idx := strings.Index(basePart, "?")
	if idx == -1 || idx == len(basePart)-1 {
		return []KeyValuePair{}
	}

	queryString := basePart[idx+1:]
	parts := strings.Split(queryString, "&")
	var result []KeyValuePair

	for _, part := range parts {
		if part == "" {
			continue
		}
		eqIdx := strings.Index(part, "=")
		var k, v string
		if eqIdx != -1 {
			k, _ = url.QueryUnescape(part[:eqIdx])
			v, _ = url.QueryUnescape(part[eqIdx+1:])
		} else {
			k, _ = url.QueryUnescape(part)
			v = ""
		}

		result = append(result, KeyValuePair{
			Key:     k,
			Value:   v,
			Enabled: true,
		})
	}

	return result
}

// MergeURLParams parses query parameters from rawURL and merges them with existingParams,
// preserving custom descriptions, secret flags, and disabled parameters that are not in the URL.
func MergeURLParams(existingParams []KeyValuePair, rawURL string) []KeyValuePair {
	parsed := SyncURLToParams(rawURL)
	if len(parsed) == 0 && len(existingParams) == 0 {
		return []KeyValuePair{}
	}

	result := make([]KeyValuePair, 0, len(parsed)+len(existingParams))
	matchedExisting := make(map[int]bool)

	for _, p := range parsed {
		found := false
		for i, ex := range existingParams {
			if !matchedExisting[i] && ex.Key == p.Key {
				matchedExisting[i] = true
				result = append(result, KeyValuePair{
					Key:         p.Key,
					Value:       p.Value,
					Enabled:     true,
					Description: ex.Description,
					Secret:      ex.Secret,
				})
				found = true
				break
			}
		}
		if !found {
			result = append(result, p)
		}
	}

	// Preserve any disabled parameters that were previously in the table
	for i, ex := range existingParams {
		if !matchedExisting[i] && !ex.Enabled && ex.Key != "" {
			result = append(result, ex)
		}
	}

	return result
}
