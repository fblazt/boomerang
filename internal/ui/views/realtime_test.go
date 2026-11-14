package views

import (
	"context"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

type fakeStreamCtrl struct {
	onFrame  func(models.StreamFrame)
	onStatus func(models.StreamStatus, int64, error)
}

func (f *fakeStreamCtrl) Connect(
	_ context.Context,
	proto models.StreamProtocol,
	url string,
	_ map[string]string,
	onFrame func(models.StreamFrame),
	onStatus func(models.StreamStatus, int64, error),
) error {
	f.onFrame = onFrame
	f.onStatus = onStatus
	onStatus(models.StreamConnected, 14, nil)
	onFrame(models.StreamFrame{
		ID:        "f_sys",
		Direction: models.DirectionSystem,
		Opcode:    models.OpcodeSystem,
		Payload:   "Connected " + string(proto) + " " + url,
		Timestamp: time.Now().UTC(),
	})
	return nil
}

func (f *fakeStreamCtrl) Disconnect() error {
	if f.onStatus != nil {
		f.onStatus(models.StreamDisconnected, 0, nil)
	}
	return nil
}

func (f *fakeStreamCtrl) SendText(payload string) error {
	if f.onFrame != nil {
		f.onFrame(models.StreamFrame{
			ID:         "f_sent",
			Direction:  models.DirectionSent,
			Opcode:     models.OpcodeText,
			OpcodeCode: 1,
			Payload:    payload,
			SizeBytes:  len(payload),
			Timestamp:  time.Now().UTC(),
		})
		f.onFrame(models.StreamFrame{
			ID:         "f_recv",
			Direction:  models.DirectionRecv,
			Opcode:     models.OpcodeText,
			OpcodeCode: 1,
			Payload:    `{"ack":true}`,
			SizeBytes:  12,
			Timestamp:  time.Now().UTC(),
		})
	}
	return nil
}

func (f *fakeStreamCtrl) SendBinary(data []byte) error {
	return f.SendText(string(data))
}

func (f *fakeStreamCtrl) SendPing() error {
	if f.onFrame != nil {
		f.onFrame(models.StreamFrame{
			ID:         "f_pong",
			Direction:  models.DirectionPing,
			Opcode:     models.OpcodePong,
			OpcodeCode: 10,
			Payload:    "PONG",
			LatencyMs:  11,
			Timestamp:  time.Now().UTC(),
		})
	}
	return nil
}

func TestRealtimeView_ConnectionBarFromSidebar(t *testing.T) {
	state := models.NewAppState(nil)
	state.SetStreamController(&fakeStreamCtrl{})

	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(theme.NewBoomerangTheme())
		ShellView(c, state)
	}, 1280, 800)
	tt.Frame()

	// Select Order Live Feed (WSS) in sidebar -> enters stream workspace
	if err := tt.Click("Order Live Feed"); err != nil {
		t.Fatalf("click Order Live Feed: %v", err)
	}
	tt.Frame()

	if state.WorkspaceMode != "stream" || !tt.HasText("DISCONNECTED") {
		t.Fatal("expected stream inspector open with DISCONNECTED connection bar")
	}

	// Connect
	if err := tt.Click("Connect"); err != nil {
		t.Fatalf("click Connect: %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		var status models.StreamStatus
		state.View(func(s *models.AppState) { status = s.Stream.Status })
		if status == models.StreamConnected {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	tt.Frame()

	if !tt.HasText("CONNECTED") || !tt.HasText("Ping: 14 ms") || !tt.HasText("Disconnect") {
		t.Fatal("expected CONNECTED, Ping: 14 ms, and Disconnect button")
	}

	// Push outbound frame via Frame Composer & Push Frame button
	state.SetStreamComposerBody(`{"subscribe":"trades"}`)
	tt.Frame()
	if err := tt.Click("↑ Push Frame"); err != nil {
		t.Fatalf("click Push Frame: %v", err)
	}
	tt.Frame()

	for _, want := range []string{"↑ SENT", "↓ RECV", "Text 1", `{"subscribe":"trades"}`, `{"ack":true}`} {
		if !tt.HasText(want) {
			t.Fatalf("expected %q in live direction timeline", want)
		}
	}

	// Filter timeline to SENT only
	if err := tt.Click("SENT"); err != nil {
		t.Fatalf("click SENT filter: %v", err)
	}
	tt.Frame()
	if tt.HasText(`{"ack":true}`) {
		t.Fatal("expected RECV frame hidden when SENT filter active")
	}

	// Clear filter, send manual Ping, and check PONG direction badge
	if err := tt.Click("SENT"); err != nil {
		t.Fatalf("toggle SENT filter off: %v", err)
	}
	tt.Frame()
	if err := tt.Click("Send Ping"); err != nil {
		t.Fatalf("click Send Ping: %v", err)
	}
	tt.Frame()
	if !tt.HasText("↕ PONG") || !tt.HasText("Pong 10") {
		t.Fatal("expected ↕ PONG and Pong 10 in timeline after Send Ping")
	}

	// Disconnect
	if err := tt.Click("Disconnect"); err != nil {
		t.Fatalf("click Disconnect: %v", err)
	}
	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		var status models.StreamStatus
		state.View(func(s *models.AppState) { status = s.Stream.Status })
		if status == models.StreamDisconnected {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	tt.Frame()
	if !tt.HasText("DISCONNECTED") || !tt.HasText("Connect") {
		t.Fatal("expected DISCONNECTED and Connect button after disconnect")
	}
}
