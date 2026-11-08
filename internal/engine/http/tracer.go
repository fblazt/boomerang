package http

import (
	"context"
	"crypto/tls"
	"net/http/httptrace"
	"time"

	"github.com/fblazt/boomerang/internal/models"
)

// TraceCollector tracks network timing hooks during an HTTP request.
type TraceCollector struct {
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
			tc.dnsStart = time.Now()
		},
		DNSDone: func(_ httptrace.DNSDoneInfo) {
			tc.dnsDone = time.Now()
		},
		ConnectStart: func(_, _ string) {
			tc.tcpStart = time.Now()
		},
		ConnectDone: func(_, _ string, _ error) {
			tc.tcpDone = time.Now()
		},
		TLSHandshakeStart: func() {
			tc.tlsStart = time.Now()
		},
		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			tc.tlsDone = time.Now()
		},
		WroteRequest: func(_ httptrace.WroteRequestInfo) {
			tc.wroteReq = time.Now()
		},
		GotFirstResponseByte: func() {
			tc.firstByte = time.Now()
		},
	}
}

// MarkBodyDone records when the response body reading completes.
func (tc *TraceCollector) MarkBodyDone() {
	tc.bodyDone = time.Now()
}

// Telemetry calculates durations and returns a models.Telemetry struct.
func (tc *TraceCollector) Telemetry() models.Telemetry {
	now := time.Now()
	if tc.bodyDone.IsZero() {
		tc.bodyDone = now
	}

	var dns int64
	if !tc.dnsStart.IsZero() && !tc.dnsDone.IsZero() && tc.dnsDone.After(tc.dnsStart) {
		dns = tc.dnsDone.Sub(tc.dnsStart).Milliseconds()
	}

	var tcp int64
	if !tc.tcpStart.IsZero() && !tc.tcpDone.IsZero() && tc.tcpDone.After(tc.tcpStart) {
		tcp = tc.tcpDone.Sub(tc.tcpStart).Milliseconds()
	}

	var tlsHandshake int64
	if !tc.tlsStart.IsZero() && !tc.tlsDone.IsZero() && tc.tlsDone.After(tc.tlsStart) {
		tlsHandshake = tc.tlsDone.Sub(tc.tlsStart).Milliseconds()
	}

	var ttfb int64
	refStart := tc.wroteReq
	if refStart.IsZero() {
		refStart = tc.totalStart
	}
	if !tc.firstByte.IsZero() && tc.firstByte.After(refStart) {
		ttfb = tc.firstByte.Sub(refStart).Milliseconds()
	}

	var dl int64
	if !tc.firstByte.IsZero() && !tc.bodyDone.IsZero() && tc.bodyDone.After(tc.firstByte) {
		dl = tc.bodyDone.Sub(tc.firstByte).Milliseconds()
	}

	total := tc.bodyDone.Sub(tc.totalStart).Milliseconds()
	if total < 0 {
		total = 0
	}

	return models.Telemetry{
		DNS:      dns,
		TCP:      tcp,
		TLS:      tlsHandshake,
		TTFB:     ttfb,
		Download: dl,
		Total:    total,
	}
}

// WithTrace hooks a TraceCollector into ctx using httptrace.
func WithTrace(ctx context.Context, tc *TraceCollector) context.Context {
	return httptrace.WithClientTrace(ctx, tc.ClientTrace())
}
