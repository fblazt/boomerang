package http

import (
	"crypto/tls"
	"net/http"
	"strings"
	"time"

	"github.com/fblazt/boomerang/internal/models"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

// NewHTTP3Transport creates an HTTP/3 QUIC RoundTripper configured with the provided TLS settings.
func NewHTTP3Transport(tlsCfg *tls.Config) *http3.Transport {
	var h3TLS *tls.Config
	if tlsCfg != nil {
		h3TLS = tlsCfg.Clone()
	} else {
		h3TLS = &tls.Config{MinVersion: tls.VersionTLS13}
	}
	if len(h3TLS.NextProtos) == 0 {
		h3TLS.NextProtos = []string{http3.NextProtoH3}
	}

	return &http3.Transport{
		TLSClientConfig: h3TLS,
		QUICConfig: &quic.Config{
			HandshakeIdleTimeout: 5 * time.Second,
			MaxIdleTimeout:       30 * time.Second,
		},
	}
}

// BuildTransport constructs an http.RoundTripper and an optional cleanup function
// honoring the protocol, SSL verification, Root CA pinning, and mTLS settings in policy.
func BuildTransport(policy *models.TransportPolicy) (http.RoundTripper, func(), error) {
	tlsCfg, err := BuildTLSConfig(policy)
	if err != nil {
		return nil, nil, err
	}

	proto := "HTTP/2"
	if policy != nil && strings.TrimSpace(policy.Protocol) != "" {
		proto = strings.TrimSpace(policy.Protocol)
	}

	switch proto {
	case "HTTP/3":
		h3Tr := NewHTTP3Transport(tlsCfg)
		return h3Tr, func() { _ = h3Tr.Close() }, nil

	case "HTTP/1.1":
		var protocols http.Protocols
		protocols.SetHTTP1(true)
		tr := &http.Transport{
			TLSClientConfig:   tlsCfg,
			Protocols:         &protocols,
			DisableKeepAlives: false,
			MaxIdleConns:      100,
			IdleConnTimeout:   90 * time.Second,
		}
		return tr, tr.CloseIdleConnections, nil

	default: // "HTTP/2"
		var protocols http.Protocols
		protocols.SetHTTP1(true)
		protocols.SetHTTP2(true)
		tr := &http.Transport{
			TLSClientConfig:   tlsCfg,
			Protocols:         &protocols,
			ForceAttemptHTTP2: true,
			DisableKeepAlives: false,
			MaxIdleConns:      100,
			IdleConnTimeout:   90 * time.Second,
		}
		return tr, tr.CloseIdleConnections, nil
	}
}
