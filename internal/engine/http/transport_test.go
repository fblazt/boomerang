package http

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fblazt/boomerang/internal/models"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

type testPKI struct {
	caPool            *x509.CertPool
	caPath            string
	expiredCAPath     string
	serverCert        tls.Certificate
	clientCertPath    string
	clientKeyPath     string
	clientCombinedPEM string
	rogueCertPath     string
	rogueKeyPath      string
}

func generateTestPKI(t *testing.T) testPKI {
	t.Helper()
	dir := t.TempDir()
	now := time.Now()

	caCert, caKey, caPEM := mustIssueCert(t, nil, nil, certSpec{
		commonName: "Boomerang Test Root CA",
		isCA:       true,
		notBefore:  now.Add(-time.Hour),
		notAfter:   now.Add(10 * 24 * time.Hour),
	})
	caPath := filepath.Join(dir, "ca.pem")
	mustWriteFile(t, caPath, caPEM)

	_, _, expiredCAPEM := mustIssueCert(t, nil, nil, certSpec{
		commonName: "Boomerang Expired CA",
		isCA:       true,
		notBefore:  now.Add(-48 * time.Hour),
		notAfter:   now.Add(-24 * time.Hour),
	})
	expiredCAPath := filepath.Join(dir, "expired-ca.pem")
	mustWriteFile(t, expiredCAPath, expiredCAPEM)

	srvCertParsed, srvKey, srvPEM := mustIssueCert(t, caCert, caKey, certSpec{
		commonName: "localhost",
		dnsNames:   []string{"localhost", "example.internal"},
		extUsages:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		notBefore:  now.Add(-time.Hour),
		notAfter:   now.Add(24 * time.Hour),
	})
	_ = srvCertParsed
	srvKeyPEM := mustMarshalECKeyPEM(t, srvKey)
	srvTLSCert, err := tls.X509KeyPair(srvPEM, srvKeyPEM)
	if err != nil {
		t.Fatalf("server X509KeyPair: %v", err)
	}

	_, cliKey, cliPEM := mustIssueCert(t, caCert, caKey, certSpec{
		commonName: "boomerang-mtls-client",
		extUsages:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		notBefore:  now.Add(-time.Hour),
		notAfter:   now.Add(24 * time.Hour),
	})
	cliKeyPEM := mustMarshalECKeyPEM(t, cliKey)
	cliCertPath := filepath.Join(dir, "client.pem")
	cliKeyPath := filepath.Join(dir, "client.key")
	cliCombinedPath := filepath.Join(dir, "client-bundle.pem")
	mustWriteFile(t, cliCertPath, cliPEM)
	mustWriteFile(t, cliKeyPath, cliKeyPEM)
	mustWriteFile(t, cliCombinedPath, append(append([]byte(nil), cliPEM...), cliKeyPEM...))

	rogueCA, rogueCAKey, _ := mustIssueCert(t, nil, nil, certSpec{
		commonName: "Rogue Untrusted CA",
		isCA:       true,
		notBefore:  now.Add(-time.Hour),
		notAfter:   now.Add(24 * time.Hour),
	})
	_, rogueKey, roguePEM := mustIssueCert(t, rogueCA, rogueCAKey, certSpec{
		commonName: "rogue-client",
		extUsages:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		notBefore:  now.Add(-time.Hour),
		notAfter:   now.Add(24 * time.Hour),
	})
	rogueCertPath := filepath.Join(dir, "rogue.pem")
	rogueKeyPath := filepath.Join(dir, "rogue.key")
	mustWriteFile(t, rogueCertPath, roguePEM)
	mustWriteFile(t, rogueKeyPath, mustMarshalECKeyPEM(t, rogueKey))

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		t.Fatal("failed to append test CA")
	}

	return testPKI{
		caPool:            pool,
		caPath:            caPath,
		expiredCAPath:     expiredCAPath,
		serverCert:        srvTLSCert,
		clientCertPath:    cliCertPath,
		clientKeyPath:     cliKeyPath,
		clientCombinedPEM: cliCombinedPath,
		rogueCertPath:     rogueCertPath,
		rogueKeyPath:      rogueKeyPath,
	}
}

type certSpec struct {
	commonName string
	isCA       bool
	dnsNames   []string
	extUsages  []x509.ExtKeyUsage
	notBefore  time.Time
	notAfter   time.Time
}

func mustIssueCert(
	t *testing.T,
	issuer *x509.Certificate,
	issuerKey *ecdsa.PrivateKey,
	spec certSpec,
) (*x509.Certificate, *ecdsa.PrivateKey, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 62))
	if err != nil {
		t.Fatalf("serial: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: spec.commonName},
		NotBefore:             spec.notBefore,
		NotAfter:              spec.notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           spec.extUsages,
		DNSNames:              spec.dnsNames,
		BasicConstraintsValid: true,
		IsCA:                  spec.isCA,
	}
	if spec.isCA {
		tmpl.KeyUsage |= x509.KeyUsageCertSign
	}

	parent := tmpl
	signKey := key
	if issuer != nil && issuerKey != nil {
		parent = issuer
		signKey = issuerKey
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, parent, &key.PublicKey, signKey)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	return parsed, key, pemBytes
}

func mustMarshalECKeyPEM(t *testing.T, key *ecdsa.PrivateKey) []byte {
	t.Helper()
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("MarshalECPrivateKey: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestBuildTLSConfig_AndCertLoaders(t *testing.T) {
	pki := generateTestPKI(t)

	// Nil policy defaults
	cfg, err := BuildTLSConfig(nil)
	if err != nil {
		t.Fatalf("BuildTLSConfig(nil) error: %v", err)
	}
	if cfg.InsecureSkipVerify {
		t.Fatal("expected strict SSL by default when policy is nil")
	}

	// Non-strict SSL + Pinned Root CA + separate mTLS PEM pair
	cfg, err = BuildTLSConfig(&models.TransportPolicy{
		Protocol:      "HTTP/2",
		StrictSSL:     false,
		PinnedRootCA:  pki.caPath,
		ClientCertPEM: pki.clientCertPath,
		ClientKeyPEM:  pki.clientKeyPath,
	})
	if err != nil {
		t.Fatalf("BuildTLSConfig error: %v", err)
	}
	if !cfg.InsecureSkipVerify {
		t.Fatal("expected InsecureSkipVerify=true when StrictSSL=false")
	}
	if cfg.RootCAs == nil || len(cfg.Certificates) != 1 {
		t.Fatalf("expected RootCAs and 1 client cert, got pool=%v certs=%d", cfg.RootCAs != nil, len(cfg.Certificates))
	}

	// Combined PEM bundle without explicit key path
	cert, err := LoadClientCertificate(pki.clientCombinedPEM, "")
	if err != nil || len(cert.Certificate) == 0 {
		t.Fatalf("expected combined PEM to load cleanly, got err=%v", err)
	}

	// InspectRootCA valid & expired & invalid
	validSummary, err := InspectRootCA(pki.caPath)
	if err != nil || !strings.HasPrefix(validSummary, "VALID · ") {
		t.Fatalf("expected VALID summary, got %q err=%v", validSummary, err)
	}
	expiredSummary, err := InspectRootCA(pki.expiredCAPath)
	if err != nil || !strings.HasPrefix(expiredSummary, "EXPIRED") {
		t.Fatalf("expected EXPIRED summary, got %q err=%v", expiredSummary, err)
	}

	badFile := filepath.Join(t.TempDir(), "not-a-cert.p12")
	mustWriteFile(t, badFile, []byte("invalid-der"))
	if _, err := BuildTLSConfig(&models.TransportPolicy{PinnedRootCA: badFile}); err == nil {
		t.Fatal("expected error for invalid Root CA PEM")
	}
	if _, err := LoadClientCertificate(badFile, ""); err == nil {
		t.Fatal("expected error for invalid PKCS#12 file")
	}
}

func runPipeTLSHandshake(t *testing.T, serverCfg, clientCfg *tls.Config) error {
	t.Helper()
	srvConn, cliConn := net.Pipe()
	deadline := time.Now().Add(2 * time.Second)
	_ = srvConn.SetDeadline(deadline)
	_ = cliConn.SetDeadline(deadline)

	srvTLS := tls.Server(srvConn, serverCfg)
	cliTLS := tls.Client(cliConn, clientCfg)

	errCh := make(chan error, 1)
	go func() {
		defer srvConn.Close()
		if err := srvTLS.Handshake(); err != nil {
			errCh <- err
			return
		}
		_, err := srvTLS.Write([]byte("ok"))
		errCh <- err
	}()

	cliErr := cliTLS.Handshake()
	if cliErr == nil {
		var buf [2]byte
		_, cliErr = io.ReadFull(cliTLS, buf[:])
	}
	_ = cliConn.Close()
	srvErr := <-errCh
	if cliErr != nil {
		return cliErr
	}
	return srvErr
}

func TestTLS_RootCAPinning_And_mTLS_InMemoryPipe(t *testing.T) {
	pki := generateTestPKI(t)

	serverCfg := &tls.Config{
		Certificates: []tls.Certificate{pki.serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pki.caPool,
	}

	// 1. Missing pinned CA must reject unknown server authority when StrictSSL=true
	noCACfg, err := BuildTLSConfig(&models.TransportPolicy{
		StrictSSL:     true,
		ClientCertPEM: pki.clientCombinedPEM,
	})
	if err != nil {
		t.Fatalf("BuildTLSConfig: %v", err)
	}
	noCACfg.ServerName = "localhost"
	if err := runPipeTLSHandshake(t, serverCfg, noCACfg); err == nil {
		t.Fatal("expected handshake failure without pinned Root CA")
	}

	// 2. Pinned CA without client cert must fail mTLS server check
	noClientCertCfg, err := BuildTLSConfig(&models.TransportPolicy{
		StrictSSL:    true,
		PinnedRootCA: pki.caPath,
	})
	if err != nil {
		t.Fatalf("BuildTLSConfig: %v", err)
	}
	noClientCertCfg.ServerName = "localhost"
	if err := runPipeTLSHandshake(t, serverCfg, noClientCertCfg); err == nil {
		t.Fatal("expected handshake failure without mTLS client cert")
	}

	// 3. Rogue client cert signed by another CA must fail mTLS check
	rogueCfg, err := BuildTLSConfig(&models.TransportPolicy{
		StrictSSL:     true,
		PinnedRootCA:  pki.caPath,
		ClientCertPEM: pki.rogueCertPath,
		ClientKeyPEM:  pki.rogueKeyPath,
	})
	if err != nil {
		t.Fatalf("BuildTLSConfig: %v", err)
	}
	rogueCfg.ServerName = "localhost"
	if err := runPipeTLSHandshake(t, serverCfg, rogueCfg); err == nil {
		t.Fatal("expected handshake failure with untrusted client cert")
	}

	// 4. Valid Pinned Root CA + valid mTLS Client Cert succeeds
	validCfg, err := BuildTLSConfig(&models.TransportPolicy{
		StrictSSL:     true,
		PinnedRootCA:  pki.caPath,
		ClientCertPEM: pki.clientCertPath,
		ClientKeyPEM:  pki.clientKeyPath,
	})
	if err != nil {
		t.Fatalf("BuildTLSConfig: %v", err)
	}
	validCfg.ServerName = "localhost"
	if err := runPipeTLSHandshake(t, serverCfg, validCfg); err != nil {
		t.Fatalf("expected mTLS handshake to succeed, got: %v", err)
	}
}

func TestBuildTransport_ProtocolSelection(t *testing.T) {
	pki := generateTestPKI(t)

	// HTTP/1.1
	rt1, cleanup1, err := BuildTransport(&models.TransportPolicy{
		Protocol:  "HTTP/1.1",
		StrictSSL: true,
	})
	if err != nil {
		t.Fatalf("BuildTransport HTTP/1.1: %v", err)
	}
	defer cleanup1()
	tr1, ok := rt1.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport for HTTP/1.1, got %T", rt1)
	}
	if !tr1.Protocols.HTTP1() || tr1.Protocols.HTTP2() {
		t.Fatalf("expected HTTP/1.1 only, got HTTP1=%v HTTP2=%v", tr1.Protocols.HTTP1(), tr1.Protocols.HTTP2())
	}

	// HTTP/2
	rt2, cleanup2, err := BuildTransport(&models.TransportPolicy{
		Protocol:     "HTTP/2",
		StrictSSL:    true,
		PinnedRootCA: pki.caPath,
	})
	if err != nil {
		t.Fatalf("BuildTransport HTTP/2: %v", err)
	}
	defer cleanup2()
	tr2, ok := rt2.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport for HTTP/2, got %T", rt2)
	}
	if !tr2.Protocols.HTTP1() || !tr2.Protocols.HTTP2() {
		t.Fatalf("expected HTTP/1.1+HTTP/2 enabled, got HTTP1=%v HTTP2=%v", tr2.Protocols.HTTP1(), tr2.Protocols.HTTP2())
	}

	// HTTP/3
	rt3, cleanup3, err := BuildTransport(&models.TransportPolicy{
		Protocol:      "HTTP/3",
		StrictSSL:     true,
		PinnedRootCA:  pki.caPath,
		ClientCertPEM: pki.clientCombinedPEM,
	})
	if err != nil {
		t.Fatalf("BuildTransport HTTP/3: %v", err)
	}
	defer cleanup3()
	h3Tr, ok := rt3.(*http3.Transport)
	if !ok {
		t.Fatalf("expected *http3.Transport for HTTP/3, got %T", rt3)
	}
	if h3Tr.TLSClientConfig == nil || h3Tr.TLSClientConfig.RootCAs == nil || len(h3Tr.TLSClientConfig.Certificates) != 1 {
		t.Fatal("expected HTTP/3 transport to carry RootCAs and client cert")
	}
}

func TestHTTP3Transport_InMemoryQUIC_WithCustomCA_And_mTLS(t *testing.T) {
	pki := generateTestPKI(t)
	srvConn, cliConn := newInMemPacketConnPair()

	srvQuicTr := &quic.Transport{Conn: srvConn}
	cliQuicTr := &quic.Transport{Conn: cliConn}
	// Close underlying PacketConns before quic.Transport.Close so listen goroutines unblock cleanly (LIFO).
	defer srvQuicTr.Close()
	defer cliQuicTr.Close()
	defer srvConn.Close()
	defer cliConn.Close()

	srvTLS := http3.ConfigureTLSConfig(&tls.Config{
		Certificates: []tls.Certificate{pki.serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pki.caPool,
	})

	ln, err := srvQuicTr.ListenEarly(srvTLS, nil)
	if err != nil {
		t.Fatalf("ListenEarly: %v", err)
	}

	h3Server := &http3.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cn := ""
			if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
				cn = r.TLS.PeerCertificates[0].Subject.CommonName
			}
			_, _ = w.Write([]byte(r.Proto + "|" + cn))
		}),
	}
	go func() {
		_ = h3Server.ServeListener(ln)
	}()

	cliTLS, err := BuildTLSConfig(&models.TransportPolicy{
		Protocol:      "HTTP/3",
		StrictSSL:     true,
		PinnedRootCA:  pki.caPath,
		ClientCertPEM: pki.clientCombinedPEM,
	})
	if err != nil {
		t.Fatalf("BuildTLSConfig: %v", err)
	}

	h3Tr := NewHTTP3Transport(cliTLS)
	h3Tr.Dial = func(ctx context.Context, _ string, tlsCfg *tls.Config, cfg *quic.Config) (*quic.Conn, error) {
		return cliQuicTr.Dial(ctx, srvConn.LocalAddr(), tlsCfg, cfg)
	}
	defer h3Tr.Close()

	client := &http.Client{
		Transport: h3Tr,
		Timeout:   5 * time.Second,
	}
	resp, err := client.Get("https://localhost/h3-test")
	if err != nil {
		t.Fatalf("HTTP/3 GET failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	if string(body) != "HTTP/3.0|boomerang-mtls-client" {
		t.Fatalf("unexpected HTTP/3 response body: %q", string(body))
	}
}

func TestClientExecute_InvalidPolicyReturnsError(t *testing.T) {
	c := NewClient(2 * time.Second)
	resp, err := c.Execute(context.Background(), models.RequestModel{
		Method: "GET",
		URL:    "https://example.internal/health",
		TransportPolicy: &models.TransportPolicy{
			Protocol:     "HTTP/2",
			StrictSSL:    true,
			PinnedRootCA: filepath.Join(t.TempDir(), "missing-ca.pem"),
		},
	})
	if err == nil {
		t.Fatal("expected error for missing pinned Root CA")
	}
	if resp == nil || resp.Error == "" {
		t.Fatal("expected ResponseModel error details")
	}
}

// memPacketAddr implements net.Addr for in-memory QUIC packet connections.
type memPacketAddr string

func (a memPacketAddr) Network() string { return "udp" }
func (a memPacketAddr) String() string  { return string(a) }

// memPacket represents an in-memory UDP datagram.
type memPacket struct {
	data []byte
	from net.Addr
}

// memPacketConn implements net.PacketConn in memory without opening host OS sockets.
type memPacketConn struct {
	local      memPacketAddr
	peer       *memPacketConn
	inbox      chan memPacket
	closeOnce  sync.Once
	closed     chan struct{}
	mu         sync.Mutex
	readDead   time.Time
	writeDead  time.Time
}

func newInMemPacketConnPair() (*memPacketConn, *memPacketConn) {
	a := &memPacketConn{
		local:  memPacketAddr("127.0.0.1:44301"),
		inbox:  make(chan memPacket, 128),
		closed: make(chan struct{}),
	}
	b := &memPacketConn{
		local:  memPacketAddr("127.0.0.1:44302"),
		peer:   a,
		inbox:  make(chan memPacket, 128),
		closed: make(chan struct{}),
	}
	a.peer = b
	return a, b
}

func (c *memPacketConn) ReadFrom(p []byte) (int, net.Addr, error) {
	c.mu.Lock()
	dead := c.readDead
	c.mu.Unlock()

	var timerCh <-chan time.Time
	if !dead.IsZero() {
		d := time.Until(dead)
		if d <= 0 {
			return 0, nil, os.ErrDeadlineExceeded
		}
		t := time.NewTimer(d)
		defer t.Stop()
		timerCh = t.C
	}

	select {
	case <-c.closed:
		return 0, nil, net.ErrClosed
	case <-timerCh:
		return 0, nil, os.ErrDeadlineExceeded
	case pkt := <-c.inbox:
		n := copy(p, pkt.data)
		return n, pkt.from, nil
	}
}

func (c *memPacketConn) WriteTo(p []byte, _ net.Addr) (int, error) {
	select {
	case <-c.closed:
		return 0, net.ErrClosed
	default:
	}

	cp := append([]byte(nil), p...)
	select {
	case <-c.peer.closed:
		return 0, errors.New("peer closed")
	case c.peer.inbox <- memPacket{data: cp, from: c.local}:
		return len(p), nil
	default:
		// Drop packet on full buffer just like UDP.
		return len(p), nil
	}
}

func (c *memPacketConn) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
	})
	return nil
}

func (c *memPacketConn) LocalAddr() net.Addr { return c.local }

func (c *memPacketConn) SetDeadline(t time.Time) error {
	_ = c.SetReadDeadline(t)
	return c.SetWriteDeadline(t)
}

func (c *memPacketConn) SetReadDeadline(t time.Time) error {
	c.mu.Lock()
	c.readDead = t
	c.mu.Unlock()
	return nil
}

func (c *memPacketConn) SetWriteDeadline(t time.Time) error {
	c.mu.Lock()
	c.writeDead = t
	c.mu.Unlock()
	return nil
}
