package models

import (
	"context"
	"strings"
	"time"
)

const maxStreamFrames = 500

// SetStreamController attaches the live WebSocket/SSE runtime controller to AppState.
func (s *AppState) SetStreamController(ctrl StreamController) {
	s.mu.Lock()
	s.streamCtrl = ctrl
	s.mu.Unlock()
}

// SetWorkspaceMode switches the center workspace mode ("rest", "stream", "graphql").
func (s *AppState) SetWorkspaceMode(mode string) {
	s.Mutate(func(state *AppState) {
		if mode == "" {
			mode = "rest"
		}
		state.WorkspaceMode = mode
	})
}

// OpenStreamEndpoint selects a WebSocket or SSE endpoint and opens the Realtime Stream Inspector.
func (s *AppState) OpenStreamEndpoint(proto StreamProtocol, name, url string) {
	s.Mutate(func(state *AppState) {
		state.WorkspaceMode = "stream"
		if proto != "" {
			state.Stream.Protocol = proto
		}
		if url != "" {
			state.Stream.URL = url
		}
		if name != "" {
			state.ActiveRequest.Name = name
		}
	})
}

// SetStreamProtocol switches the active realtime inspector protocol between WSS and SSE.
func (s *AppState) SetStreamProtocol(proto StreamProtocol) {
	s.Mutate(func(state *AppState) {
		state.Stream.Protocol = proto
		if proto == StreamProtocolSSE && strings.HasPrefix(state.Stream.URL, "wss://") {
			state.Stream.URL = "https://stream.internal.net/sse"
		} else if proto == StreamProtocolWSS && strings.HasPrefix(state.Stream.URL, "https://") {
			state.Stream.URL = "wss://stream.internal.net/orders"
		}
	})
}

// SetStreamURL updates the realtime stream target URL.
func (s *AppState) SetStreamURL(targetURL string) {
	s.mu.Lock()
	s.Stream.URL = targetURL
	s.mu.Unlock()
}

// SetStreamComposerBody updates the frame payload composer text.
func (s *AppState) SetStreamComposerBody(body string) {
	s.mu.Lock()
	s.Stream.ComposerBody = body
	s.mu.Unlock()
}

// SetStreamComposerType toggles between TEXT and BINARY outbound WebSocket frames.
func (s *AppState) SetStreamComposerType(op StreamOpcode) {
	s.Mutate(func(state *AppState) {
		state.Stream.ComposerType = string(op)
	})
}

// SetStreamDirectionFilter sets the frame log direction filter ("", "SENT", "RECV", "PING").
func (s *AppState) SetStreamDirectionFilter(dir string) {
	s.Mutate(func(state *AppState) {
		if state.StreamFilterDirection == dir {
			state.StreamFilterDirection = ""
		} else {
			state.StreamFilterDirection = dir
		}
	})
}

// ClearStreamFrames clears all logged realtime frames and counters.
func (s *AppState) ClearStreamFrames() {
	s.Mutate(func(state *AppState) {
		state.StreamFrames = nil
		state.Stream.SentCount = 0
		state.Stream.RecvCount = 0
	})
}

// ConnectStream resolves template variables and initiates an async WebSocket or SSE connection.
func (s *AppState) ConnectStream() {
	s.mu.Lock()
	ctrl := s.streamCtrl
	if ctrl == nil {
		s.mu.Unlock()
		return
	}
	proto := s.Stream.Protocol
	rawURL := s.Stream.URL
	vars := s.getActiveVariablesLocked()
	strFn := s.interpolateStrFn
	s.Stream.Status = StreamConnecting
	s.Stream.Error = ""
	s.Stream.PingRTTMs = 0
	s.mu.Unlock()

	if strFn != nil {
		rawURL = strFn(rawURL, vars)
	}
	s.RequestRedraw()

	go func() {
		err := ctrl.Connect(
			context.Background(),
			proto,
			rawURL,
			nil,
			func(f StreamFrame) {
				s.appendStreamFrame(f)
			},
			func(status StreamStatus, rttMs int64, stErr error) {
				s.updateStreamStatus(status, rttMs, stErr)
			},
		)
		if err != nil {
			s.updateStreamStatus(StreamError, 0, err)
		}
	}()
}

// DisconnectStream terminates the active realtime connection.
func (s *AppState) DisconnectStream() {
	s.mu.RLock()
	ctrl := s.streamCtrl
	s.mu.RUnlock()
	if ctrl == nil {
		s.updateStreamStatus(StreamDisconnected, 0, nil)
		return
	}
	go func() {
		_ = ctrl.Disconnect()
		s.updateStreamStatus(StreamDisconnected, 0, nil)
	}()
}

// SendStreamFrame sends the current composer payload over the connected WebSocket session.
func (s *AppState) SendStreamFrame() error {
	s.mu.RLock()
	ctrl := s.streamCtrl
	status := s.Stream.Status
	body := s.Stream.ComposerBody
	op := s.Stream.ComposerType
	vars := s.getActiveVariablesLocked()
	strFn := s.interpolateStrFn
	s.mu.RUnlock()

	if ctrl == nil || status != StreamConnected {
		return nil
	}
	if strFn != nil {
		body = strFn(body, vars)
	}
	if op == string(OpcodeBinary) {
		return ctrl.SendBinary([]byte(body))
	}
	return ctrl.SendText(body)
}

// SendStreamPing sends an explicit WebSocket Ping frame for live RTT measurement.
func (s *AppState) SendStreamPing() error {
	s.mu.RLock()
	ctrl := s.streamCtrl
	s.mu.RUnlock()
	if ctrl == nil {
		return nil
	}
	return ctrl.SendPing()
}

func (s *AppState) appendStreamFrame(f StreamFrame) {
	s.mu.Lock()
	s.StreamFrames = append(s.StreamFrames, f)
	if len(s.StreamFrames) > maxStreamFrames {
		s.StreamFrames = append([]StreamFrame(nil), s.StreamFrames[len(s.StreamFrames)-maxStreamFrames:]...)
	}
	switch f.Direction {
	case DirectionSent:
		s.Stream.SentCount++
	case DirectionRecv:
		s.Stream.RecvCount++
	}
	s.mu.Unlock()
	s.RequestRedraw()
}

func (s *AppState) updateStreamStatus(status StreamStatus, rttMs int64, err error) {
	s.mu.Lock()
	s.Stream.Status = status
	if rttMs > 0 {
		s.Stream.PingRTTMs = rttMs
	}
	if status == StreamConnected && s.Stream.ConnectedAt.IsZero() {
		s.Stream.ConnectedAt = time.Now().UTC()
	} else if status != StreamConnected {
		s.Stream.ConnectedAt = time.Time{}
	}
	if err != nil {
		s.Stream.Error = err.Error()
	} else if status == StreamConnected || status == StreamDisconnected {
		s.Stream.Error = ""
	}
	s.mu.Unlock()
	s.RequestRedraw()
}
