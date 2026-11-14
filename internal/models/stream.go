package models

import "time"

// StreamProtocol identifies the realtime streaming transport protocol.
type StreamProtocol string

const (
	StreamProtocolWSS StreamProtocol = "WSS"
	StreamProtocolSSE StreamProtocol = "SSE"
)

// StreamDirection indicates whether a frame was sent, received, a keep-alive probe, or system status.
type StreamDirection string

const (
	DirectionSent   StreamDirection = "SENT"
	DirectionRecv   StreamDirection = "RECV"
	DirectionPing   StreamDirection = "PING"
	DirectionSystem StreamDirection = "SYSTEM"
)

// StreamOpcode labels WebSocket opcodes and SSE frame types.
type StreamOpcode string

const (
	OpcodeText   StreamOpcode = "TEXT"
	OpcodeBinary StreamOpcode = "BINARY"
	OpcodePing   StreamOpcode = "PING"
	OpcodePong   StreamOpcode = "PONG"
	OpcodeClose  StreamOpcode = "CLOSE"
	OpcodeEvent  StreamOpcode = "EVENT"
	OpcodeSystem StreamOpcode = "SYSTEM"
)

// StreamStatus represents the lifecycle state of a realtime connection.
type StreamStatus string

const (
	StreamDisconnected StreamStatus = "disconnected"
	StreamConnecting   StreamStatus = "connecting"
	StreamConnected    StreamStatus = "connected"
	StreamError        StreamStatus = "error"
)

// StreamFrame represents one WebSocket frame, keep-alive event, SSE event, or lifecycle notice.
type StreamFrame struct {
	ID         string          `json:"id"`
	Direction  StreamDirection `json:"direction"`
	Opcode     StreamOpcode    `json:"opcode"`
	OpcodeCode int             `json:"opcode_code"` // RFC 6455: 1=Text, 2=Binary, 8=Close, 9=Ping, 10=Pong
	Payload    string          `json:"payload"`
	EventName  string          `json:"event_name,omitempty"`
	EventID    string          `json:"event_id,omitempty"`
	SizeBytes  int             `json:"size_bytes"`
	LatencyMs  int64           `json:"latency_ms,omitempty"`
	Timestamp  time.Time       `json:"timestamp"`
}

// StreamSession captures live metadata for an active or closed stream connection.
type StreamSession struct {
	Protocol     StreamProtocol `json:"protocol"`
	URL          string         `json:"url"`
	Status       StreamStatus   `json:"status"`
	PingRTTMs    int64          `json:"ping_rtt_ms"`
	ConnectedAt  time.Time      `json:"connected_at,omitempty"`
	SentCount    int            `json:"sent_count"`
	RecvCount    int            `json:"recv_count"`
	Error        string         `json:"error,omitempty"`
	ComposerBody string         `json:"composer_body,omitempty"`
	ComposerType string         `json:"composer_type,omitempty"` // TEXT or BINARY
}
