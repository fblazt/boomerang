package realtime

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/fblazt/boomerang/internal/models"
)

func TestSSEClient_ParsesEventsAndMultiLinePayloads(t *testing.T) {
	ln := newPipeListener()
	defer ln.Close()

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			fl, _ := w.(http.Flusher)

			_, _ = fmt.Fprint(w, ": heartbeat comment\n\n")
			_, _ = fmt.Fprint(w, "id: evt_1\nevent: metric\ndata: {\"cpu\":42}\ndata: {\"mem\":64}\n\n")
			_, _ = fmt.Fprint(w, "id: evt_2\ndata: ping-ready\n\n")
			if fl != nil {
				fl.Flush()
			}
			<-r.Context().Done()
		}),
	}
	go func() { _ = srv.Serve(ln) }()
	defer srv.Close()

	var (
		mu     sync.Mutex
		frames []models.StreamFrame
	)
	client := NewSSEClient()
	err := client.Connect(context.Background(), SSEOptions{
		URL:         "http://inmemory.local/events",
		DialContext: ln.DialContext,
		OnFrame: func(f models.StreamFrame) {
			mu.Lock()
			frames = append(frames, f)
			mu.Unlock()
		},
	})
	if err != nil {
		t.Fatalf("SSE Connect failed: %v", err)
	}
	defer client.Disconnect()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		eventCount := 0
		for _, f := range frames {
			if f.Direction == models.DirectionRecv && f.Opcode == models.OpcodeEvent {
				eventCount++
			}
		}
		mu.Unlock()
		if eventCount >= 2 {
			break
		}
		time.Sleep(15 * time.Millisecond)
	}

	if err := client.Disconnect(); err != nil {
		t.Fatalf("SSE Disconnect failed: %v", err)
	}
	if client.LastEventID() != "evt_2" {
		t.Fatalf("expected LastEventID 'evt_2', got %q", client.LastEventID())
	}

	mu.Lock()
	defer mu.Unlock()

	var events []models.StreamFrame
	for _, f := range frames {
		if f.Opcode == models.OpcodeEvent {
			events = append(events, f)
		}
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 SSE event frames, got %d (%+v)", len(events), events)
	}
	if events[0].EventName != "metric" || events[0].EventID != "evt_1" || events[0].Payload != "{\"cpu\":42}\n{\"mem\":64}" {
		t.Fatalf("unexpected first SSE event: %+v", events[0])
	}
	if events[1].EventName != "message" || events[1].EventID != "evt_2" || events[1].Payload != "ping-ready" {
		t.Fatalf("unexpected second SSE event: %+v", events[1])
	}
}
