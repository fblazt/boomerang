package http

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fblazt/boomerang/internal/models"
	"golang.org/x/crypto/pkcs12"
)

// BuildTLSConfig constructs a *tls.Config from an environment TransportPolicy.
// It supports StrictSSL verification, custom Root CA pinning, and mTLS client identities (PEM / PKCS#12).
func BuildTLSConfig(policy *models.TransportPolicy) (*tls.Config, error) {
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if policy == nil {
		return cfg, nil
	}

	cfg.InsecureSkipVerify = !policy.StrictSSL

	if caPath := strings.TrimSpace(policy.PinnedRootCA); caPath != "" {
		pool, err := LoadRootCAPool(caPath)
		if err != nil {
			return nil, fmt.Errorf("load pinned root CA %q: %w", caPath, err)
		}
		cfg.RootCAs = pool
	}

	if certPath := strings.TrimSpace(policy.ClientCertPEM); certPath != "" {
		cert, err := LoadClientCertificate(certPath, strings.TrimSpace(policy.ClientKeyPEM))
		if err != nil {
			return nil, fmt.Errorf("load mTLS client certificate %q: %w", certPath, err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}

	return cfg, nil
}

// LoadRootCAPool parses one or more PEM-encoded CA certificates into an isolated x509.CertPool.
func LoadRootCAPool(pemPath string) (*x509.CertPool, error) {
	pemBytes, err := os.ReadFile(pemPath)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pemBytes) {
		return nil, fmt.Errorf("no valid PEM certificates found in %s", filepath.Base(pemPath))
	}
	return pool, nil
}

// LoadClientCertificate loads an mTLS client certificate from separate PEM cert/key files,
// a combined PEM file (cert+key), or an unencrypted PKCS#12 (.p12/.pfx) archive.
func LoadClientCertificate(certPath, keyPath string) (tls.Certificate, error) {
	ext := strings.ToLower(filepath.Ext(certPath))
	if ext == ".p12" || ext == ".pfx" {
		pfxBytes, err := os.ReadFile(certPath)
		if err != nil {
			return tls.Certificate{}, err
		}
		blocks, err := pkcs12.ToPEM(pfxBytes, "")
		if err != nil {
			return tls.Certificate{}, fmt.Errorf("decode pkcs12: %w", err)
		}
		var pemData []byte
		for _, b := range blocks {
			pemData = append(pemData, pem.EncodeToMemory(b)...)
		}
		return tls.X509KeyPair(pemData, pemData)
	}

	if keyPath != "" {
		return tls.LoadX509KeyPair(certPath, keyPath)
	}

	// Combined PEM containing both CERTIFICATE and PRIVATE KEY blocks.
	pemBytes, err := os.ReadFile(certPath)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.X509KeyPair(pemBytes, pemBytes)
}

// InspectRootCA returns a concise status summary (CN and validity days) for a PEM Root CA file.
func InspectRootCA(pemPath string) (string, error) {
	pemBytes, err := os.ReadFile(pemPath)
	if err != nil {
		return "", err
	}
	for len(pemBytes) > 0 {
		var block *pem.Block
		block, pemBytes = pem.Decode(pemBytes)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return "", err
		}
		days := int(time.Until(cert.NotAfter).Hours() / 24)
		if days < 0 {
			return fmt.Sprintf("EXPIRED (%s)", cert.Subject.CommonName), nil
		}
		return fmt.Sprintf("VALID · %d DAYS", days), nil
	}
	return "", fmt.Errorf("no certificate block in %s", filepath.Base(pemPath))
}
