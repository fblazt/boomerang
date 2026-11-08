package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fblazt/boomerang/internal/models"
)

// Client executes HTTP requests and collects telemetry metrics.
type Client struct {
	httpClient *http.Client
}

// NewClient returns a new configured HTTP Client with default transport.
func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		DisableKeepAlives: false,
		MaxIdleConns:      100,
		IdleConnTimeout:   90 * time.Second,
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
	}
}

// NewClientWithTransport creates a Client using a custom http.RoundTripper.
func NewClientWithTransport(transport http.RoundTripper, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
	}
}

// Execute dispatches the request model through net/http with httptrace.
func (c *Client) Execute(ctx context.Context, req models.RequestModel) (*models.ResponseModel, error) {
	targetURL := req.URL
	if targetURL == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}

	// Default scheme if omitted
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "http://" + targetURL
	}

	// Apply enabled query params
	if len(req.Params) > 0 {
		targetURL = models.SyncParamsToURL(targetURL, req.Params)
	}

	// If API Key in query param
	if req.AuthType == "apikey" && req.APIKeyIn == "query" && req.APIKeyKey != "" {
		param := []models.KeyValuePair{{Key: req.APIKeyKey, Value: req.APIKeyValue, Enabled: true}}
		// Parse existing and append
		existing := models.SyncURLToParams(targetURL)
		existing = append(existing, param...)
		targetURL = models.SyncParamsToURL(targetURL, existing)
	}

	// Build Body
	var bodyReader io.Reader
	var autoContentType string

	switch req.BodyType {
	case "json":
		bodyReader = strings.NewReader(req.Body)
		autoContentType = "application/json"
	case "text":
		bodyReader = strings.NewReader(req.Body)
		autoContentType = "text/plain"
	case "form-data":
		var b bytes.Buffer
		mw := multipart.NewWriter(&b)
		for _, item := range req.FormData {
			if item.Enabled && item.Key != "" {
				_ = mw.WriteField(item.Key, item.Value)
			}
		}
		_ = mw.Close()
		bodyReader = &b
		autoContentType = mw.FormDataContentType()
	case "urlencoded":
		vals := url.Values{}
		for _, item := range req.URLEncoded {
			if item.Enabled && item.Key != "" {
				vals.Add(item.Key, item.Value)
			}
		}
		encoded := vals.Encode()
		bodyReader = strings.NewReader(encoded)
		autoContentType = "application/x-www-form-urlencoded"
	case "binary":
		bodyReader = strings.NewReader(req.Body)
	default:
		// "none" or empty
		bodyReader = nil
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = "GET"
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, targetURL, bodyReader)
	if err != nil {
		return &models.ResponseModel{
			Error: err.Error(),
		}, err
	}

	// Set content-type if prepared
	if autoContentType != "" {
		httpReq.Header.Set("Content-Type", autoContentType)
	}

	// Apply custom headers
	for _, h := range req.Headers {
		if h.Enabled && strings.TrimSpace(h.Key) != "" {
			httpReq.Header.Set(h.Key, h.Value)
		}
	}

	// Apply Auth
	switch req.AuthType {
	case "bearer":
		if req.Token != "" {
			httpReq.Header.Set("Authorization", "Bearer "+req.Token)
		}
	case "basic":
		httpReq.SetBasicAuth(req.BasicUser, req.BasicPassword)
	case "apikey":
		if req.APIKeyIn == "header" && req.APIKeyKey != "" {
			httpReq.Header.Set(req.APIKeyKey, req.APIKeyValue)
		}
	}

	// Trace collector
	traceCollector := NewTraceCollector()
	tracedCtx := WithTrace(httpReq.Context(), traceCollector)
	httpReq = httpReq.WithContext(tracedCtx)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		traceCollector.MarkBodyDone()
		telemetry := traceCollector.Telemetry()
		return &models.ResponseModel{
			Error:     err.Error(),
			Telemetry: telemetry,
		}, err
	}
	defer resp.Body.Close()

	bodyBytes, readErr := io.ReadAll(resp.Body)
	traceCollector.MarkBodyDone()
	telemetry := traceCollector.Telemetry()

	if readErr != nil {
		return &models.ResponseModel{
			StatusCode: resp.StatusCode,
			StatusText: resp.Status,
			Proto:      resp.Proto,
			Error:      readErr.Error(),
			Telemetry:  telemetry,
		}, readErr
	}

	respHeaders := make(map[string]string)
	for k, v := range resp.Header {
		respHeaders[k] = strings.Join(v, ", ")
	}

	return &models.ResponseModel{
		StatusCode: resp.StatusCode,
		StatusText: resp.Status,
		Duration:   time.Duration(telemetry.Total) * time.Millisecond,
		SizeBytes:  int64(len(bodyBytes)),
		Proto:      resp.Proto,
		Headers:    respHeaders,
		Body:       string(bodyBytes),
		Telemetry:  telemetry,
	}, nil
}
