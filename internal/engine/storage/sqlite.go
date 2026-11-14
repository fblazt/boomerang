package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fblazt/boomerang/internal/models"
	_ "modernc.org/sqlite"
)

const sqliteSchema = `
CREATE TABLE IF NOT EXISTS history_entries (
	id TEXT PRIMARY KEY,
	request_id TEXT NOT NULL DEFAULT '',
	method TEXT NOT NULL,
	url TEXT NOT NULL,
	status_code INTEGER NOT NULL DEFAULT 0,
	status_text TEXT NOT NULL DEFAULT '',
	duration_ms INTEGER NOT NULL DEFAULT 0,
	size_bytes INTEGER NOT NULL DEFAULT 0,
	protocol TEXT NOT NULL DEFAULT '',
	environment_name TEXT NOT NULL DEFAULT '',
	error_text TEXT NOT NULL DEFAULT '',
	executed_at INTEGER NOT NULL,
	request_json TEXT NOT NULL,
	response_json TEXT NOT NULL DEFAULT '',
	telemetry_json TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_history_executed_at ON history_entries(executed_at DESC);
CREATE INDEX IF NOT EXISTS idx_history_method ON history_entries(method);
CREATE INDEX IF NOT EXISTS idx_history_status ON history_entries(status_code);

CREATE TABLE IF NOT EXISTS scratchpad_drafts (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	sort_order INTEGER NOT NULL DEFAULT 0,
	request_json TEXT NOT NULL,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_drafts_order ON scratchpad_drafts(sort_order ASC, updated_at DESC);
`

// DefaultDBPath returns the default SQLite path (~/.boomerang/boomerang.db).
func DefaultDBPath() (string, error) {
	dir, err := DefaultDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "boomerang.db"), nil
}

// SQLiteStore provides persistent SQLite storage for execution history logs and scratchpad drafts.
type SQLiteStore struct {
	mu sync.Mutex
	db *sql.DB
}

// OpenSQLite opens (or creates) a SQLite database at dbPath (or ":memory:") and applies migrations.
func OpenSQLite(dbPath string) (*SQLiteStore, error) {
	if strings.TrimSpace(dbPath) == "" {
		return nil, fmt.Errorf("sqlite db path cannot be empty")
	}

	isMemory := dbPath == ":memory:" || strings.HasPrefix(dbPath, "file::memory:")
	if !isMemory {
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", dbPath, err)
	}
	// SQLite handles concurrent writers cleanly with a single pooled writer connection + busy timeout.
	db.SetMaxOpenConns(1)

	if !isMemory {
		if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;`); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("configure sqlite pragmas: %w", err)
		}
	}
	if _, err := db.Exec(sqliteSchema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate sqlite schema: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

// Close closes the underlying SQLite connection.
func (s *SQLiteStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

// RecordHistory persists an executed HTTP call along with its request, response, and telemetry snapshot.
func (s *SQLiteStore) RecordHistory(entry models.HistoryEntry) (models.HistoryEntry, error) {
	if entry.ID == "" {
		entry.ID = NewID("hist")
	}
	if entry.ExecutedAt.IsZero() {
		entry.ExecutedAt = time.Now().UTC()
	}

	reqJSON, err := json.Marshal(entry.Request)
	if err != nil {
		return models.HistoryEntry{}, fmt.Errorf("marshal history request: %w", err)
	}
	var respJSON string
	if entry.Response != nil {
		b, err := json.Marshal(entry.Response)
		if err != nil {
			return models.HistoryEntry{}, fmt.Errorf("marshal history response: %w", err)
		}
		respJSON = string(b)
	}
	telJSON, err := json.Marshal(entry.Telemetry)
	if err != nil {
		return models.HistoryEntry{}, fmt.Errorf("marshal history telemetry: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return models.HistoryEntry{}, fmt.Errorf("sqlite store is closed")
	}

	_, err = s.db.Exec(
		`INSERT INTO history_entries (
			id, request_id, method, url, status_code, status_text, duration_ms, size_bytes,
			protocol, environment_name, error_text, executed_at, request_json, response_json, telemetry_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID,
		entry.RequestID,
		strings.ToUpper(strings.TrimSpace(entry.Method)),
		entry.URL,
		entry.StatusCode,
		entry.StatusText,
		entry.DurationMs,
		entry.SizeBytes,
		entry.Protocol,
		entry.EnvironmentName,
		entry.Error,
		entry.ExecutedAt.UTC().UnixNano(),
		string(reqJSON),
		respJSON,
		string(telJSON),
	)
	if err != nil {
		return models.HistoryEntry{}, fmt.Errorf("insert history entry: %w", err)
	}
	return entry, nil
}

// ListHistory queries persisted execution history ordered newest-first using filter.
func (s *SQLiteStore) ListHistory(filter models.HistoryFilter) ([]models.HistoryEntry, error) {
	whereClause, args := buildHistoryWhere(filter)
	limit := filter.Limit
	if limit <= 0 {
		limit = 200
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(
		`SELECT id, request_id, method, url, status_code, status_text, duration_ms, size_bytes,
		        protocol, environment_name, error_text, executed_at, request_json, response_json, telemetry_json
		 FROM history_entries %s
		 ORDER BY executed_at DESC
		 LIMIT ? OFFSET ?`,
		whereClause,
	)
	args = append(args, limit, offset)

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil, fmt.Errorf("sqlite store is closed")
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()

	var entries []models.HistoryEntry
	for rows.Next() {
		entry, err := scanHistoryRow(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate history rows: %w", err)
	}
	return entries, nil
}

// CountHistory returns total matching history entries for filter.
func (s *SQLiteStore) CountHistory(filter models.HistoryFilter) (int, error) {
	whereClause, args := buildHistoryWhere(filter)
	query := fmt.Sprintf(`SELECT COUNT(*) FROM history_entries %s`, whereClause)

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return 0, fmt.Errorf("sqlite store is closed")
	}

	var count int
	if err := s.db.QueryRow(query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count history: %w", err)
	}
	return count, nil
}

// GetHistory retrieves a single persisted HistoryEntry by ID.
func (s *SQLiteStore) GetHistory(id string) (models.HistoryEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return models.HistoryEntry{}, fmt.Errorf("sqlite store is closed")
	}

	row := s.db.QueryRow(
		`SELECT id, request_id, method, url, status_code, status_text, duration_ms, size_bytes,
		        protocol, environment_name, error_text, executed_at, request_json, response_json, telemetry_json
		 FROM history_entries WHERE id = ?`,
		id,
	)
	return scanHistoryRow(row)
}

// DeleteHistory removes a single history row by ID.
func (s *SQLiteStore) DeleteHistory(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return fmt.Errorf("sqlite store is closed")
	}
	_, err := s.db.Exec(`DELETE FROM history_entries WHERE id = ?`, id)
	return err
}

// ClearHistory removes all stored history rows.
func (s *SQLiteStore) ClearHistory() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return fmt.Errorf("sqlite store is closed")
	}
	_, err := s.db.Exec(`DELETE FROM history_entries`)
	return err
}

// SaveDraft creates or updates a scratchpad draft in SQLite.
func (s *SQLiteStore) SaveDraft(draft models.ScratchpadDraft) (models.ScratchpadDraft, error) {
	now := time.Now().UTC()
	if draft.ID == "" {
		draft.ID = NewID("draft")
	}
	if strings.TrimSpace(draft.Name) == "" {
		draft.Name = "Untitled Request"
	}
	if draft.CreatedAt.IsZero() {
		draft.CreatedAt = now
	}
	draft.UpdatedAt = now
	draft.Request.ID = draft.ID
	draft.Request.Name = draft.Name

	reqJSON, err := json.Marshal(draft.Request)
	if err != nil {
		return models.ScratchpadDraft{}, fmt.Errorf("marshal draft request: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return models.ScratchpadDraft{}, fmt.Errorf("sqlite store is closed")
	}

	_, err = s.db.Exec(
		`INSERT INTO scratchpad_drafts (id, name, sort_order, request_json, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			sort_order = excluded.sort_order,
			request_json = excluded.request_json,
			updated_at = excluded.updated_at`,
		draft.ID,
		draft.Name,
		draft.Order,
		string(reqJSON),
		draft.CreatedAt.UTC().UnixNano(),
		draft.UpdatedAt.UTC().UnixNano(),
	)
	if err != nil {
		return models.ScratchpadDraft{}, fmt.Errorf("upsert draft: %w", err)
	}
	return draft, nil
}

// ListDrafts returns all saved scratchpad drafts ordered by sort_order ASC, updated_at DESC.
func (s *SQLiteStore) ListDrafts() ([]models.ScratchpadDraft, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil, fmt.Errorf("sqlite store is closed")
	}

	rows, err := s.db.Query(
		`SELECT id, name, sort_order, request_json, created_at, updated_at
		 FROM scratchpad_drafts
		 ORDER BY sort_order ASC, updated_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query drafts: %w", err)
	}
	defer rows.Close()

	var drafts []models.ScratchpadDraft
	for rows.Next() {
		var (
			d         models.ScratchpadDraft
			reqJSON   string
			createdNs int64
			updatedNs int64
		)
		if err := rows.Scan(&d.ID, &d.Name, &d.Order, &reqJSON, &createdNs, &updatedNs); err != nil {
			return nil, fmt.Errorf("scan draft: %w", err)
		}
		if err := json.Unmarshal([]byte(reqJSON), &d.Request); err != nil {
			return nil, fmt.Errorf("decode draft request: %w", err)
		}
		d.CreatedAt = time.Unix(0, createdNs).UTC()
		d.UpdatedAt = time.Unix(0, updatedNs).UTC()
		drafts = append(drafts, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate drafts: %w", err)
	}
	return drafts, nil
}

// DeleteDraft deletes a scratchpad draft by ID.
func (s *SQLiteStore) DeleteDraft(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return fmt.Errorf("sqlite store is closed")
	}
	_, err := s.db.Exec(`DELETE FROM scratchpad_drafts WHERE id = ?`, id)
	return err
}

func buildHistoryWhere(f models.HistoryFilter) (string, []any) {
	var clauses []string
	var args []any

	if m := strings.ToUpper(strings.TrimSpace(f.Method)); m != "" && m != "ALL" {
		clauses = append(clauses, "method = ?")
		args = append(args, m)
	}

	switch strings.ToLower(strings.TrimSpace(f.StatusClass)) {
	case "2xx":
		clauses = append(clauses, "status_code >= 200 AND status_code < 300")
	case "3xx":
		clauses = append(clauses, "status_code >= 300 AND status_code < 400")
	case "4xx":
		clauses = append(clauses, "status_code >= 400 AND status_code < 500")
	case "5xx":
		clauses = append(clauses, "status_code >= 500 AND status_code < 600")
	case "err", "error":
		clauses = append(clauses, "(error_text != '' OR status_code = 0)")
	}

	if q := strings.TrimSpace(f.Search); q != "" {
		like := "%" + strings.ToLower(q) + "%"
		clauses = append(clauses, "(LOWER(url) LIKE ? OR LOWER(method) LIKE ? OR LOWER(status_text) LIKE ? OR LOWER(error_text) LIKE ? OR LOWER(request_json) LIKE ?)")
		args = append(args, like, like, like, like, like)
	}

	if !f.Since.IsZero() {
		clauses = append(clauses, "executed_at >= ?")
		args = append(args, f.Since.UTC().UnixNano())
	}
	if !f.Until.IsZero() {
		clauses = append(clauses, "executed_at <= ?")
		args = append(args, f.Until.UTC().UnixNano())
	}

	if len(clauses) == 0 {
		return "", nil
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanHistoryRow(scanner rowScanner) (models.HistoryEntry, error) {
	var (
		e          models.HistoryEntry
		executedNs int64
		reqJSON    string
		respJSON   string
		telJSON    string
	)
	if err := scanner.Scan(
		&e.ID,
		&e.RequestID,
		&e.Method,
		&e.URL,
		&e.StatusCode,
		&e.StatusText,
		&e.DurationMs,
		&e.SizeBytes,
		&e.Protocol,
		&e.EnvironmentName,
		&e.Error,
		&executedNs,
		&reqJSON,
		&respJSON,
		&telJSON,
	); err != nil {
		return models.HistoryEntry{}, err
	}

	e.ExecutedAt = time.Unix(0, executedNs).UTC()
	if err := json.Unmarshal([]byte(reqJSON), &e.Request); err != nil {
		return models.HistoryEntry{}, fmt.Errorf("decode history request: %w", err)
	}
	if respJSON != "" {
		var resp models.ResponseModel
		if err := json.Unmarshal([]byte(respJSON), &resp); err != nil {
			return models.HistoryEntry{}, fmt.Errorf("decode history response: %w", err)
		}
		e.Response = &resp
	}
	if telJSON != "" {
		if err := json.Unmarshal([]byte(telJSON), &e.Telemetry); err != nil {
			return models.HistoryEntry{}, fmt.Errorf("decode history telemetry: %w", err)
		}
	}
	return e, nil
}
