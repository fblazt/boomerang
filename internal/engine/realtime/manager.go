package realtime

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync"

	"github.com/fblazt/boomerang/internal/models"
)

// Manager implements models.StreamController across both WebSocket and SSE protocols.
type Manager struct {
	mu          sync.Mutex
	proto       models.StreamProtocol
	ws          *WSClient
	sse         *SSEClient
	dialContext func(ctx context.Context, network, addr string) (net.Conn, error)
	tlsConfig   *tls.Config
}

// NewManager creates a unified WebSocket and SSE realtime stream controller.
func NewManager() *Manager {
	return &Manager{
		ws:  NewWSClient(),
		sse: NewSSEClient(),
	}
}

// SetDialContext configures a custom network dialer (useful for custom routing or in-memory testing).
func (m *Manager) SetDialContext(fn func(ctx context.Context, network, addr string) (net.Conn, error)) {
	m.mu.Lock()
	m.dialContext = fn
	m.mu.Unlock()
}

// SetTLSConfig configures TLS parameters used by subsequent WebSocket and SSE sessions.
func (m *Manager) SetTLSConfig(cfg *tls.Config) {
	m.mu.Lock()
	m.tlsConfig = cfg
	m.mu.Unlock()
}

// Connect establishes a streaming session for proto against targetURL.
func (m *Manager) Connect(
	ctx context.Context,
	proto models.StreamProtocol,
	targetURL string,
	headers map[string]string,
	onFrame func(models.StreamFrame),
	onStatus func(models.StreamStatus, int64, error),
) error {
	_ = m.Disconnect()

	hdr := make(http.Header)
	for k, v := range headers {
		hdr.Set(k, v)
	}

	m.mu.Lock()
	m.proto = proto
	dialFn := m.dialContext
	tlsCfg := m.tlsConfig
	m.mu.Unlock()

	if proto == models.StreamProtocolSSE {
		return m.sse.Connect(ctx, SSEOptions{
			URL:         targetURL,
			Headers:     hdr,
			TLSConfig:   tlsCfg,
			DialContext: dialFn,
			OnFrame:     onFrame,
			OnStatus:    onStatus,
		})
	}

	return m.ws.Connect(ctx, WSOptions{
		URL:         targetURL,
		Headers:     hdr,
		TLSConfig:   tlsCfg,
		DialContext: dialFn,
		OnFrame:     onFrame,
		OnStatus:    onStatus,
	})
}

// Disconnect terminates the currently active WebSocket or SSE session.
func (m *Manager) Disconnect() error {
	_ = m.ws.Disconnect()
	_ = m.sse.Disconnect()
	return nil
}

// SendText transmits a WebSocket text frame.
func (m *Manager) SendText(payload string) error {
	return m.ws.SendText(payload)
}

// SendBinary transmits a WebSocket binary frame.
func (m *Manager) SendBinary(data []byte) error {
	return m.ws.SendBinary(data)
}

// SendPing transmits an explicit WebSocket ping frame.
func (m *Manager) SendPing() error {
	return m.ws.SendPing()
}
