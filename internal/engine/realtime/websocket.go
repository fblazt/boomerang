package realtime

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fblazt/boomerang/internal/models"
	"github.com/gorilla/websocket"
)

// WSOptions configures a WebSocket connection session.
type WSOptions struct {
	URL          string
	Headers      http.Header
	TLSConfig    *tls.Config
	PingInterval time.Duration
	DialContext  func(ctx context.Context, network, addr string) (net.Conn, error)
	OnFrame      func(models.StreamFrame)
	OnStatus     func(status models.StreamStatus, pingRTTMs int64, err error)
}

// WSClient manages an interactive WebSocket connection with keep-alives, latency tracking, and thread-safe writes.
type WSClient struct {
	mu      sync.Mutex
	writeMu sync.Mutex

	conn   *websocket.Conn
	opts   WSOptions
	cancel context.CancelFunc
	wg     sync.WaitGroup

	status    models.StreamStatus
	pingRTTMs atomic.Int64
	lastPing  atomic.Int64
	frameSeq  atomic.Uint64
}

// NewWSClient returns an idle WebSocket client.
func NewWSClient() *WSClient {
	return &WSClient{status: models.StreamDisconnected}
}

// Connect opens a WebSocket connection using opts and starts read + keep-alive loops.
func (w *WSClient) Connect(ctx context.Context, opts WSOptions) error {
	_ = w.Disconnect()

	if opts.PingInterval <= 0 {
		opts.PingInterval = 5 * time.Second
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig:  opts.TLSConfig,
		NetDialContext:   opts.DialContext,
	}

	w.emitStatus(models.StreamConnecting, 0, nil)

	conn, resp, err := dialer.DialContext(ctx, opts.URL, opts.Headers)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		w.emitStatus(models.StreamError, 0, err)
		return fmt.Errorf("websocket dial %s: %w", opts.URL, err)
	}

	conn.SetPongHandler(func(appData string) error {
		sentNano := w.lastPing.Load()
		var rttMs int64
		if sentNano > 0 {
			rttMs = time.Since(time.Unix(0, sentNano)).Milliseconds()
			if rttMs < 1 {
				rttMs = 1
			}
		}
		w.pingRTTMs.Store(rttMs)
		w.emitFrame(models.StreamFrame{
			Direction:  models.DirectionPing,
			Opcode:     models.OpcodePong,
			OpcodeCode: websocket.PongMessage,
			Payload:    "PONG " + appData,
			SizeBytes:  len(appData),
			LatencyMs:  rttMs,
			Timestamp:  time.Now().UTC(),
		})
		w.emitStatus(models.StreamConnected, rttMs, nil)
		return nil
	})

	runCtx, cancel := context.WithCancel(context.Background())

	w.mu.Lock()
	w.conn = conn
	w.opts = opts
	w.cancel = cancel
	w.status = models.StreamConnected
	w.mu.Unlock()

	w.emitStatus(models.StreamConnected, 0, nil)
	w.emitFrame(models.StreamFrame{
		Direction:  models.DirectionSystem,
		Opcode:     models.OpcodeSystem,
		OpcodeCode: 0,
		Payload:    "Connected to " + opts.URL,
		Timestamp:  time.Now().UTC(),
	})

	w.wg.Add(2)
	go w.readLoop(runCtx, conn)
	go w.keepAliveLoop(runCtx, conn, opts.PingInterval)
	return nil
}

// SendText transmits a UTF-8 text frame and records it in the stream log.
func (w *WSClient) SendText(payload string) error {
	return w.sendData(websocket.TextMessage, models.OpcodeText, []byte(payload), payload)
}

// SendBinary transmits a binary frame and logs a hex preview in the stream log.
func (w *WSClient) SendBinary(data []byte) error {
	return w.sendData(websocket.BinaryMessage, models.OpcodeBinary, data, "0x"+hex.EncodeToString(data))
}

// SendPing transmits an explicit keep-alive Ping control frame for RTT measurement.
func (w *WSClient) SendPing() error {
	w.mu.Lock()
	conn := w.conn
	w.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("websocket is not connected")
	}

	now := time.Now().UTC()
	w.lastPing.Store(now.UnixNano())

	w.writeMu.Lock()
	err := conn.WriteControl(websocket.PingMessage, []byte("ping"), now.Add(3*time.Second))
	w.writeMu.Unlock()
	if err != nil {
		return fmt.Errorf("write ping: %w", err)
	}

	w.emitFrame(models.StreamFrame{
		Direction:  models.DirectionPing,
		Opcode:     models.OpcodePing,
		OpcodeCode: websocket.PingMessage,
		Payload:    "PING keep-alive",
		SizeBytes:  4,
		Timestamp:  now,
	})
	return nil
}

// Disconnect gracefully closes the active WebSocket connection and waits for background loops to exit.
func (w *WSClient) Disconnect() error {
	w.mu.Lock()
	conn := w.conn
	cancel := w.cancel
	w.conn = nil
	w.cancel = nil
	wasConnected := w.status == models.StreamConnected || w.status == models.StreamConnecting
	w.status = models.StreamDisconnected
	w.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if conn != nil {
		w.writeMu.Lock()
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "client disconnect"),
			time.Now().Add(time.Second),
		)
		w.writeMu.Unlock()
		_ = conn.Close()
	}
	w.wg.Wait()

	if wasConnected {
		w.emitFrame(models.StreamFrame{
			Direction:  models.DirectionSystem,
			Opcode:     models.OpcodeClose,
			OpcodeCode: websocket.CloseMessage,
			Payload:    "Disconnected",
			Timestamp:  time.Now().UTC(),
		})
		w.emitStatus(models.StreamDisconnected, w.pingRTTMs.Load(), nil)
	}
	return nil
}

// Status returns the current WebSocket connection status.
func (w *WSClient) Status() models.StreamStatus {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.status
}

// PingRTT returns the most recently observed WebSocket Ping/Pong round-trip latency in milliseconds.
func (w *WSClient) PingRTT() int64 {
	return w.pingRTTMs.Load()
}

func (w *WSClient) sendData(msgType int, op models.StreamOpcode, raw []byte, display string) error {
	w.mu.Lock()
	conn := w.conn
	w.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("websocket is not connected")
	}

	w.writeMu.Lock()
	err := conn.WriteMessage(msgType, raw)
	w.writeMu.Unlock()
	if err != nil {
		return fmt.Errorf("write frame: %w", err)
	}

	w.emitFrame(models.StreamFrame{
		Direction:  models.DirectionSent,
		Opcode:     op,
		OpcodeCode: msgType,
		Payload:    display,
		SizeBytes:  len(raw),
		Timestamp:  time.Now().UTC(),
	})
	return nil
}

func (w *WSClient) readLoop(ctx context.Context, conn *websocket.Conn) {
	defer w.wg.Done()
	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
			}
			w.mu.Lock()
			stillActive := w.conn == conn
			if stillActive {
				w.conn = nil
				w.status = models.StreamError
			}
			w.mu.Unlock()
			if stillActive {
				w.emitFrame(models.StreamFrame{
					Direction:  models.DirectionSystem,
					Opcode:     models.OpcodeClose,
					OpcodeCode: websocket.CloseMessage,
					Payload:    err.Error(),
					Timestamp:  time.Now().UTC(),
				})
				w.emitStatus(models.StreamError, w.pingRTTMs.Load(), err)
			}
			return
		}

		op := models.OpcodeText
		payload := string(data)
		if msgType == websocket.BinaryMessage {
			op = models.OpcodeBinary
			payload = "0x" + hex.EncodeToString(data)
		}

		w.emitFrame(models.StreamFrame{
			Direction:  models.DirectionRecv,
			Opcode:     op,
			OpcodeCode: msgType,
			Payload:    payload,
			SizeBytes:  len(data),
			Timestamp:  time.Now().UTC(),
		})
	}
}

func (w *WSClient) keepAliveLoop(ctx context.Context, conn *websocket.Conn, interval time.Duration) {
	defer w.wg.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.mu.Lock()
			same := w.conn == conn
			w.mu.Unlock()
			if !same {
				return
			}
			_ = w.SendPing()
		}
	}
}

func (w *WSClient) emitFrame(f models.StreamFrame) {
	seq := w.frameSeq.Add(1)
	if f.ID == "" {
		f.ID = fmt.Sprintf("frm_%d", seq)
	}
	w.mu.Lock()
	cb := w.opts.OnFrame
	w.mu.Unlock()
	if cb != nil {
		cb(f)
	}
}

func (w *WSClient) emitStatus(status models.StreamStatus, rtt int64, err error) {
	w.mu.Lock()
	cb := w.opts.OnStatus
	w.status = status
	w.mu.Unlock()
	if cb != nil {
		cb(status, rtt, err)
	}
}
