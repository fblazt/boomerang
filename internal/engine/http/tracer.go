package http

import (
	"context"
	"crypto/tls"
	"net/http/httptrace"
	"sync"
	"time"

	"github.com/fblazt/boomerang/internal/models"
)

// TraceCollector tracks network timing hooks and connection metadata during an HTTP request.
type TraceCollector struct {
	mu         sync.Mutex
	dnsStart   time.Time
	dnsDone    time.Time
	tcpStart   time.Time
	tcpDone    time.Time
	tlsStart   time.Time
	tlsDone    time.Time
	reqStart   time.Time
	wroteReq   time.Time
	firstByte  time.Time
	bodyDone   time.Time
	totalStart time.Time
	remoteAddr string
	tlsVersion string
	tlsCipher  string
	connReused bool
}

// NewTraceCollector creates a new TraceCollector.
func NewTraceCollector() *TraceCollector {
	return &TraceCollector{
		totalStart: time.Now(),
	}
}

// ClientTrace returns a configured *httptrace.ClientTrace.
func (tc *TraceCollector) ClientTrace() *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) {
			tc.mu.Lock()
			tc.dnsStart = time.Now()
			tc.mu.Unlock()
		},
		DNSDone: func(_ httptrace.DNSDoneInfo) {
			tc.mu.Lock()
			tc.dnsDone = time.Now()
			tc.mu.Unlock()
		},
		ConnectStart: func(_, _ string) {
			tc.mu.Lock()
			tc.tcpStart = time.Now()
			tc.mu.Unlock()
		},
		ConnectDone: func(_, addr string, err error) {
			tc.mu.Lock()
			tc.tcpDone = time.Now()
			if err == nil && addr != "" && tc.remoteAddr == "" {
				tc.remoteAddr = addr
			}
			tc.mu.Unlock()
		},
		TLSHandshakeStart: func() {
			tc.mu.Lock()
			tc.tlsStart = time.Now()
			tc.mu.Unlock()
		},
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			tc.mu.Lock()
			tc.tlsDone = time.Now()
			if err == nil && cs.Version != 0 {
				tc.tlsVersion = tls.VersionName(cs.Version)
				tc.tlsCipher = tls.CipherSuiteName(cs.CipherSuite)
			}
			tc.mu.Unlock()
		},
		GotConn: func(info httptrace.GotConnInfo) {
			tc.mu.Lock()
			tc.connReused = info.Reused
			if info.Conn != nil && info.Conn.RemoteAddr() != nil {
				tc.remoteAddr = info.Conn.RemoteAddr().String()
			}
			tc.mu.Unlock()
		},
		WroteRequest: func(_ httptrace.WroteRequestInfo) {
			tc.mu.Lock()
			tc.wroteReq = time.Now()
			tc.mu.Unlock()
		},
		GotFirstResponseByte: func() {
			tc.mu.Lock()
			tc.firstByte = time.Now()
			tc.mu.Unlock()
		},
	}
}

// MarkBodyDone records when the response body reading completes.
func (tc *TraceCollector) MarkBodyDone() {
	tc.mu.Lock()
	tc.bodyDone = time.Now()
	tc.mu.Unlock()
}

func durMs(start, end time.Time) int64 {
	if start.IsZero() || end.IsZero() || !end.After(start) {
		return 0
	}
	d := end.Sub(start)
	ms := d.Milliseconds()
	if ms == 0 && d > 0 {
		return 1
	}
	return ms
}

// Telemetry calculates durations and returns a models.Telemetry struct.
func (tc *TraceCollector) Telemetry() models.Telemetry {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	now := time.Now()
	if tc.bodyDone.IsZero() {
		tc.bodyDone = now
	}

	dns := durMs(tc.dnsStart, tc.dnsDone)
	tcp := durMs(tc.tcpStart, tc.tcpDone)
	tlsHandshake := durMs(tc.tlsStart, tc.tlsDone)

	refStart := tc.wroteReq
	if refStart.IsZero() {
		refStart = tc.totalStart
	}
	ttfb := durMs(refStart, tc.firstByte)
	dl := durMs(tc.firstByte, tc.bodyDone)

	elapsed := tc.bodyDone.Sub(tc.totalStart)
	total := elapsed.Milliseconds()
	totalMicros := elapsed.Microseconds()
	if total < 0 {
		total = 0
		totalMicros = 0
	} else if total == 0 && elapsed > 0 {
		total = 1
	}

	return models.Telemetry{
		DNS:         dns,
		TCP:         tcp,
		TLS:         tlsHandshake,
		TTFB:        ttfb,
		Download:    dl,
		Total:       total,
		TotalMicros: totalMicros,
		RemoteAddr:  tc.remoteAddr,
		TLSVersion:  tc.tlsVersion,
		TLSCipher:   tc.tlsCipher,
		ConnReused:  tc.connReused,
	}
}

// WithTrace hooks a TraceCollector into ctx using httptrace.
func WithTrace(ctx context.Context, tc *TraceCollector) context.Context {
	return httptrace.WithClientTrace(ctx, tc.ClientTrace())
}
