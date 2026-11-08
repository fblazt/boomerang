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

	total := tc.bodyDone.Sub(tc.totalStart).Milliseconds()
	if total < 0 {
		total = 0
	} else if total == 0 && tc.bodyDone.After(tc.totalStart) {
		total = 1
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
