package realtime

import (
	"context"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/fblazt/boomerang/internal/models"
	"github.com/gorilla/websocket"
)

type pipeListener struct {
	conns chan net.Conn
	done  chan struct{}
	once  sync.Once
}

func newPipeListener() *pipeListener {
	return &pipeListener{
		conns: make(chan net.Conn, 8),
		done:  make(chan struct{}),
	}
}

func (l *pipeListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

func (l *pipeListener) Close() error {
	l.once.Do(func() { close(l.done) })
	return nil
}

func (l *pipeListener) Addr() net.Addr { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8080} }

func (l *pipeListener) DialContext(_ context.Context, _, _ string) (net.Conn, error) {
	clientConn, serverConn := net.Pipe()
	select {
	case l.conns <- serverConn:
		return clientConn, nil
	case <-l.done:
		_ = clientConn.Close()
		_ = serverConn.Close()
		return nil, net.ErrClosed
	}
}

func TestWSClient_FramesAndKeepAlivePingPong(t *testing.T) {
	ln := newPipeListener()
	defer ln.Close()

	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()
			for {
				mt, msg, err := conn.ReadMessage()
				if err != nil {
					return
				}
				if err := conn.WriteMessage(mt, msg); err != nil {
					return
				}
			}
		}),
	}
	go func() { _ = srv.Serve(ln) }()
	defer srv.Close()

	var (
		mu     sync.Mutex
		frames []models.StreamFrame
	)
	client := NewWSClient()
	err := client.Connect(context.Background(), WSOptions{
		URL:          "ws://inmemory.local/ws",
		PingInterval: 80 * time.Millisecond,
		DialContext:  ln.DialContext,
		OnFrame: func(f models.StreamFrame) {
			mu.Lock()
			frames = append(frames, f)
			mu.Unlock()
		},
	})
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer client.Disconnect()

	if err := client.SendText(`{"hello":"ws"}`); err != nil {
		t.Fatalf("SendText failed: %v", err)
	}
	if err := client.SendBinary([]byte{0xDE, 0xAD, 0xBE, 0xEF}); err != nil {
		t.Fatalf("SendBinary failed: %v", err)
	}
	if err := client.SendPing(); err != nil {
		t.Fatalf("SendPing failed: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		var gotTextRecv, gotBinRecv, gotPong bool
		for _, f := range frames {
			if f.Direction == models.DirectionRecv && f.Opcode == models.OpcodeText && f.Payload == `{"hello":"ws"}` {
				gotTextRecv = true
			}
			if f.Direction == models.DirectionRecv && f.Opcode == models.OpcodeBinary && f.Payload == "0xdeadbeef" {
				gotBinRecv = true
			}
			if f.Direction == models.DirectionPing && f.Opcode == models.OpcodePong {
				gotPong = true
			}
		}
		mu.Unlock()
		if gotTextRecv && gotBinRecv && gotPong {
			break
		}
		time.Sleep(15 * time.Millisecond)
	}

	if err := client.Disconnect(); err != nil {
		t.Fatalf("Disconnect failed: %v", err)
	}
	if client.Status() != models.StreamDisconnected {
		t.Fatalf("expected disconnected status, got %s", client.Status())
	}
	if client.PingRTT() <= 0 {
		t.Fatalf("expected PingRTT > 0 after pong, got %d", client.PingRTT())
	}

	mu.Lock()
	defer mu.Unlock()
	var seenText, seenBin, seenPong bool
	for _, f := range frames {
		switch {
		case f.Direction == models.DirectionRecv && f.OpcodeCode == websocket.TextMessage:
			seenText = true
		case f.Direction == models.DirectionRecv && f.OpcodeCode == websocket.BinaryMessage:
			seenBin = true
		case f.Direction == models.DirectionPing && f.Opcode == models.OpcodePong:
			seenPong = true
		}
	}
	if !seenText || !seenBin || !seenPong {
		t.Fatalf("missing expected frames text=%v bin=%v pong=%v frames=%+v", seenText, seenBin, seenPong, frames)
	}
}
