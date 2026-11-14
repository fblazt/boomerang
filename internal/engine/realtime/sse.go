package realtime

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fblazt/boomerang/internal/models"
)

// SSEOptions configures a Server-Sent Events stream consumer.
type SSEOptions struct {
	URL         string
	Headers     http.Header
	TLSConfig   *tls.Config
	LastEventID string
	DialContext func(ctx context.Context, network, addr string) (net.Conn, error)
	OnFrame     func(models.StreamFrame)
	OnStatus    func(status models.StreamStatus, latencyMs int64, err error)
}

// SSEClient connects to an HTTP text/event-stream endpoint and parses real-time events.
type SSEClient struct {
	mu          sync.Mutex
	opts        SSEOptions
	status      models.StreamStatus
	cancel      context.CancelFunc
	respBody    io.ReadCloser
	lastEventID string
	retryMs     int64
	wg          sync.WaitGroup
	seq         atomic.Uint64
}

// NewSSEClient returns an idle Server-Sent Events client.
func NewSSEClient() *SSEClient {
	return &SSEClient{status: models.StreamDisconnected}
}

// Connect establishes the streaming HTTP SSE connection and begins parsing events in the background.
func (s *SSEClient) Connect(ctx context.Context, opts SSEOptions) error {
	_ = s.Disconnect()

	s.emitStatus(models.StreamConnecting, 0, nil)

	runCtx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(runCtx, http.MethodGet, opts.URL, nil)
	if err != nil {
		cancel()
		s.emitStatus(models.StreamError, 0, err)
		return fmt.Errorf("build sse request: %w", err)
	}

	for k, vals := range opts.Headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	if opts.LastEventID != "" {
		req.Header.Set("Last-Event-ID", opts.LastEventID)
	}

	tr := &http.Transport{
		TLSClientConfig: opts.TLSConfig,
		DialContext:     opts.DialContext,
	}
	client := &http.Client{Transport: tr}

	dialCtx, dialCancel := context.WithTimeout(ctx, 10*time.Second)
	defer dialCancel()
	go func() {
		<-dialCtx.Done()
		if dialCtx.Err() == context.DeadlineExceeded || ctx.Err() != nil {
			cancel()
		}
	}()

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		cancel()
		s.emitStatus(models.StreamError, 0, err)
		return fmt.Errorf("connect sse %s: %w", opts.URL, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = resp.Body.Close()
		cancel()
		statusErr := fmt.Errorf("sse unexpected status %s", resp.Status)
		s.emitStatus(models.StreamError, 0, statusErr)
		return statusErr
	}

	handshakeMs := time.Since(start).Milliseconds()

	s.mu.Lock()
	s.opts = opts
	s.cancel = cancel
	s.respBody = resp.Body
	s.lastEventID = opts.LastEventID
	s.status = models.StreamConnected
	s.mu.Unlock()

	s.emitStatus(models.StreamConnected, handshakeMs, nil)
	s.emitFrame(models.StreamFrame{
		Direction: models.DirectionSystem,
		Opcode:    models.OpcodeSystem,
		Payload:   "Connected SSE stream " + opts.URL,
		LatencyMs: handshakeMs,
		Timestamp: time.Now().UTC(),
	})

	s.wg.Add(1)
	go s.readStream(runCtx, resp.Body, tr)
	return nil
}

// Disconnect terminates an active SSE stream and cleans up background goroutines.
func (s *SSEClient) Disconnect() error {
	s.mu.Lock()
	cancel := s.cancel
	body := s.respBody
	wasActive := s.status == models.StreamConnected || s.status == models.StreamConnecting
	s.cancel = nil
	s.respBody = nil
	s.status = models.StreamDisconnected
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if body != nil {
		_ = body.Close()
	}
	s.wg.Wait()

	if wasActive {
		s.emitFrame(models.StreamFrame{
			Direction: models.DirectionSystem,
			Opcode:    models.OpcodeClose,
			Payload:   "SSE stream disconnected",
			Timestamp: time.Now().UTC(),
		})
		s.emitStatus(models.StreamDisconnected, 0, nil)
	}
	return nil
}

// Status returns the current SSE stream status.
func (s *SSEClient) Status() models.StreamStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

// LastEventID returns the most recent event ID received from the SSE stream.
func (s *SSEClient) LastEventID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastEventID
}

func (s *SSEClient) readStream(ctx context.Context, body io.ReadCloser, tr *http.Transport) {
	defer s.wg.Done()
	defer tr.CloseIdleConnections()
	defer body.Close()

	reader := bufio.NewReaderSize(body, 64*1024)
	var (
		eventName string
		eventID   string
		dataLines []string
	)

	flush := func() {
		if len(dataLines) == 0 && eventName == "" {
			eventID = ""
			return
		}
		if eventName == "" {
			eventName = "message"
		}
		payload := strings.Join(dataLines, "\n")
		if eventID != "" {
			s.mu.Lock()
			s.lastEventID = eventID
			s.mu.Unlock()
		} else {
			eventID = s.LastEventID()
		}

		s.emitFrame(models.StreamFrame{
			Direction: models.DirectionRecv,
			Opcode:    models.OpcodeEvent,
			Payload:   payload,
			EventName: eventName,
			EventID:   eventID,
			SizeBytes: len(payload),
			Timestamp: time.Now().UTC(),
		})

		eventName = ""
		eventID = ""
		dataLines = nil
	}

	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			trimmed := strings.TrimRight(line, "\r\n")
			switch {
			case trimmed == "":
				flush()
			case strings.HasPrefix(trimmed, ":"):
				// SSE heartbeat comment line
			default:
				field, val, hasColon := strings.Cut(trimmed, ":")
				if hasColon {
					val = strings.TrimPrefix(val, " ")
				}
				switch field {
				case "event":
					eventName = val
				case "data":
					dataLines = append(dataLines, val)
				case "id":
					if !strings.ContainsRune(val, '\x00') {
						eventID = val
					}
				case "retry":
					if ms, parseErr := strconv.ParseInt(val, 10, 64); parseErr == nil && ms >= 0 {
						s.mu.Lock()
						s.retryMs = ms
						s.mu.Unlock()
					}
				}
			}
		}

		if err != nil {
			flush()
			select {
			case <-ctx.Done():
				return
			default:
			}
			s.mu.Lock()
			stillActive := s.respBody == body
			if stillActive {
				s.respBody = nil
				if err == io.EOF {
					s.status = models.StreamDisconnected
				} else {
					s.status = models.StreamError
				}
			}
			s.mu.Unlock()
			if stillActive {
				if err == io.EOF {
					s.emitFrame(models.StreamFrame{
						Direction: models.DirectionSystem,
						Opcode:    models.OpcodeClose,
						Payload:   "SSE stream closed by server",
						Timestamp: time.Now().UTC(),
					})
					s.emitStatus(models.StreamDisconnected, 0, nil)
				} else {
					s.emitStatus(models.StreamError, 0, err)
				}
			}
			return
		}
	}
}

func (s *SSEClient) emitFrame(f models.StreamFrame) {
	seq := s.seq.Add(1)
	if f.ID == "" {
		f.ID = fmt.Sprintf("sse_%d", seq)
	}
	s.mu.Lock()
	cb := s.opts.OnFrame
	s.mu.Unlock()
	if cb != nil {
		cb(f)
	}
}

func (s *SSEClient) emitStatus(status models.StreamStatus, latencyMs int64, err error) {
	s.mu.Lock()
	cb := s.opts.OnStatus
	s.status = status
	s.mu.Unlock()
	if cb != nil {
		cb(status, latencyMs, err)
	}
}
