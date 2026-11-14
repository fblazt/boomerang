package models

import "time"

// HistoryEntry captures an executed API call along with full request/response payloads and telemetry.
type HistoryEntry struct {
	ID              string         `json:"id"`
	RequestID       string         `json:"request_id,omitempty"`
	Method          string         `json:"method"`
	URL             string         `json:"url"`
	StatusCode      int            `json:"status_code"`
	StatusText      string         `json:"status_text"`
	DurationMs      int64          `json:"duration_ms"`
	SizeBytes       int64          `json:"size_bytes"`
	Protocol        string         `json:"protocol"`
	EnvironmentName string         `json:"environment_name,omitempty"`
	Error           string         `json:"error,omitempty"`
	ExecutedAt      time.Time      `json:"executed_at"`
	Request         RequestModel   `json:"request"`
	Response        *ResponseModel `json:"response,omitempty"`
	Telemetry       Telemetry      `json:"telemetry"`
}

// HistoryFilter configures queries against stored request execution history.
type HistoryFilter struct {
	Search      string // Case-insensitive substring match across URL, method, name, status, error
	Method      string // Optional HTTP verb filter (GET, POST, ...)
	StatusClass string // Optional filter: "2xx", "3xx", "4xx", "5xx", "err"
	Since       time.Time
	Until       time.Time
	Limit       int
	Offset      int
}

// ScratchpadDraft stores an uncommitted ad-hoc request persisted in SQLite.
type ScratchpadDraft struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Order     int          `json:"order"`
	Request   RequestModel `json:"request"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}
