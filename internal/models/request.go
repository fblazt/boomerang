package models

import "time"

// Telemetry holds nanosecond-precision network breakdown metrics.
type Telemetry struct {
	DNS      int64 // DNS lookup duration in ms
	TCP      int64 // TCP connect duration in ms
	TLS      int64 // TLS handshake duration in ms
	TTFB     int64 // Time to first byte in ms
	Download int64 // Content download duration in ms
	Total    int64 // Total roundtrip time in ms
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
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Method   string         `json:"method"`
	URL      string         `json:"url"`
	Params   []KeyValuePair `json:"params"`
	Headers  []KeyValuePair `json:"headers"`
	BodyType string         `json:"body_type"` // none, json, form, text
	Body     string         `json:"body"`
	AuthType string         `json:"auth_type"` // none, bearer, basic
	Token    string         `json:"token"`
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
	Telemetry  Telemetry         `json:"telemetry"`
	Error      string            `json:"error,omitempty"`
}
