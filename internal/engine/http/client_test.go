package http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"os"
	"testing"
	"time"

	"github.com/fblazt/boomerang/internal/models"
)

type mockTransport struct {
	handler http.Handler
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := req.Context().Err(); err != nil {
		return nil, err
	}
	rec := httptest.NewRecorder()
	m.handler.ServeHTTP(rec, req)
	resp := rec.Result()
	resp.Request = req
	return resp, nil
}

func TestClient_GET_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("filter") != "active" {
			t.Errorf("expected filter=active query param, got %s", r.URL.Query().Get("filter"))
		}
		if r.Header.Get("X-Custom") != "Boomerang" {
			t.Errorf("expected X-Custom header, got %s", r.Header.Get("X-Custom"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	client := NewClientWithTransport(&mockTransport{handler: handler}, 5*time.Second)
	req := models.RequestModel{
		Method: "GET",
		URL:    "http://api.test/v1/items",
		Params: []models.KeyValuePair{
			{Key: "filter", Value: "active", Enabled: true},
			{Key: "ignored", Value: "no", Enabled: false},
		},
		Headers: []models.KeyValuePair{
			{Key: "X-Custom", Value: "Boomerang", Enabled: true},
		},
	}

	resp, err := client.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	if resp.Body != `{"status":"ok"}` {
		t.Errorf("expected body, got %q", resp.Body)
	}
	if resp.SizeBytes != int64(len(`{"status":"ok"}`)) {
		t.Errorf("expected size %d, got %d", len(`{"status":"ok"}`), resp.SizeBytes)
	}
}

func TestClient_POST_JSON(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"user":"admin"}` {
			t.Errorf("unexpected body: %s", string(body))
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":123}`))
	})

	client := NewClientWithTransport(&mockTransport{handler: handler}, 5*time.Second)
	req := models.RequestModel{
		Method:   "POST",
		URL:      "http://api.test/users",
		BodyType: "json",
		Body:     `{"user":"admin"}`,
	}

	resp, err := client.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
	if resp.Body != `{"id":123}` {
		t.Errorf("expected body, got %q", resp.Body)
	}
}

func TestClient_AuthTypes(t *testing.T) {
	tests := []struct {
		name     string
		authType string
		setupReq func(r *models.RequestModel)
		checkReq func(t *testing.T, r *http.Request)
	}{
		{
			name:     "bearer token",
			authType: "bearer",
			setupReq: func(r *models.RequestModel) {
				r.Token = "jwt_secret_token_123"
			},
			checkReq: func(t *testing.T, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer jwt_secret_token_123" {
					t.Errorf("expected Bearer header, got %s", r.Header.Get("Authorization"))
				}
			},
		},
		{
			name:     "basic auth",
			authType: "basic",
			setupReq: func(r *models.RequestModel) {
				r.BasicUser = "admin"
				r.BasicPassword = "secretpassword"
			},
			checkReq: func(t *testing.T, r *http.Request) {
				user, pass, ok := r.BasicAuth()
				if !ok || user != "admin" || pass != "secretpassword" {
					t.Errorf("basic auth failed: user=%s pass=%s ok=%v", user, pass, ok)
				}
			},
		},
		{
			name:     "apikey header",
			authType: "apikey",
			setupReq: func(r *models.RequestModel) {
				r.APIKeyIn = "header"
				r.APIKeyKey = "X-API-Key"
				r.APIKeyValue = "key_999"
			},
			checkReq: func(t *testing.T, r *http.Request) {
				if r.Header.Get("X-API-Key") != "key_999" {
					t.Errorf("expected X-API-Key key_999, got %s", r.Header.Get("X-API-Key"))
				}
			},
		},
		{
			name:     "apikey query",
			authType: "apikey",
			setupReq: func(r *models.RequestModel) {
				r.APIKeyIn = "query"
				r.APIKeyKey = "api_key"
				r.APIKeyValue = "param_999"
			},
			checkReq: func(t *testing.T, r *http.Request) {
				if r.URL.Query().Get("api_key") != "param_999" {
					t.Errorf("expected api_key param_999, got %s", r.URL.Query().Get("api_key"))
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tc.checkReq(t, r)
				w.WriteHeader(http.StatusOK)
			})

			client := NewClientWithTransport(&mockTransport{handler: handler}, 5*time.Second)
			req := models.RequestModel{
				Method:   "GET",
				URL:      "http://api.test/auth",
				AuthType: tc.authType,
			}
			tc.setupReq(&req)

			resp, err := client.Execute(context.Background(), req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected 200, got %d", resp.StatusCode)
			}
		})
	}
}

func TestClient_FormData_And_UrlEncoded(t *testing.T) {
	// 1. Form-Data
	formHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1024 * 1024); err != nil {
			t.Fatalf("parse multipart error: %v", err)
		}
		if r.FormValue("name") != "John" {
			t.Errorf("expected John, got %s", r.FormValue("name"))
		}
		w.WriteHeader(http.StatusOK)
	})

	clientForm := NewClientWithTransport(&mockTransport{handler: formHandler}, 5*time.Second)
	resp, err := clientForm.Execute(context.Background(), models.RequestModel{
		Method:   "POST",
		URL:      "http://api.test/form",
		BodyType: "form-data",
		FormData: []models.KeyValuePair{
			{Key: "name", Value: "John", Enabled: true},
			{Key: "ignored", Value: "skip", Enabled: false},
		},
	})
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("form-data failed: err=%v status=%d", err, resp.StatusCode)
	}

	// 2. UrlEncoded
	urlHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form error: %v", err)
		}
		if r.FormValue("item") != "gadget" {
			t.Errorf("expected gadget, got %s", r.FormValue("item"))
		}
		w.WriteHeader(http.StatusOK)
	})

	clientURL := NewClientWithTransport(&mockTransport{handler: urlHandler}, 5*time.Second)
	resp, err = clientURL.Execute(context.Background(), models.RequestModel{
		Method:     "POST",
		URL:        "http://api.test/urlencoded",
		BodyType:   "urlencoded",
		URLEncoded: []models.KeyValuePair{{Key: "item", Value: "gadget", Enabled: true}},
	})
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("urlencoded failed: err=%v status=%d", err, resp.StatusCode)
	}
}

func TestClient_FormDataFileAttachment(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-upload-*.bin")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	_, _ = tmpFile.Write([]byte("file-attachment-bytes"))
	tmpFile.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("failed to parse multipart: %v", err)
		}
		file, _, err := r.FormFile("document")
		if err != nil {
			t.Fatalf("failed to get form file: %v", err)
		}
		defer file.Close()
		content, _ := io.ReadAll(file)
		if string(content) != "file-attachment-bytes" {
			t.Fatalf("unexpected content: %s", string(content))
		}
		w.WriteHeader(http.StatusOK)
	})

	client := NewClientWithTransport(&mockTransport{handler: handler}, 5*time.Second)
	resp, err := client.Execute(context.Background(), models.RequestModel{
		Method:   "POST",
		URL:      "http://api.test/upload",
		BodyType: "form-data",
		FormData: []models.KeyValuePair{
			{Key: "document", Value: "@" + tmpFile.Name(), Enabled: true},
		},
	})
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("file attachment upload failed: err=%v, status=%d", err, resp.StatusCode)
	}
}

func TestClient_BinaryFileBody(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-binary-*.bin")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	_, _ = tmpFile.Write([]byte("raw disk binary content"))
	tmpFile.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != "raw disk binary content" {
			t.Fatalf("unexpected binary body: %s", string(body))
		}
		w.WriteHeader(http.StatusOK)
	})

	client := NewClientWithTransport(&mockTransport{handler: handler}, 5*time.Second)
	resp, err := client.Execute(context.Background(), models.RequestModel{
		Method:   "POST",
		URL:      "http://api.test/binary",
		BodyType: "binary",
		Body:     tmpFile.Name(),
	})
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("binary file execution failed: err=%v, status=%d", err, resp.StatusCode)
	}
}

func TestClient_APIKeyQueryDeduplication(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys := r.URL.Query()["api_key"]
		if len(keys) != 1 || keys[0] != "new_token" {
			t.Fatalf("expected single api_key with value new_token, got: %v", keys)
		}
		w.WriteHeader(http.StatusOK)
	})

	client := NewClientWithTransport(&mockTransport{handler: handler}, 5*time.Second)
	resp, err := client.Execute(context.Background(), models.RequestModel{
		Method:      "GET",
		URL:         "http://api.test/data?api_key=old_token",
		AuthType:    "apikey",
		APIKeyIn:    "query",
		APIKeyKey:   "api_key",
		APIKeyValue: "new_token",
	})
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("apikey query deduplication failed: err=%v status=%d", err, resp.StatusCode)
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	canceledTransport := &cancelRoundTripper{}
	client := NewClientWithTransport(canceledTransport, 5*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // canceled immediately

	resp, err := client.Execute(ctx, models.RequestModel{
		Method: "GET",
		URL:    "http://api.test/cancel",
	})

	if err == nil {
		t.Fatal("expected context canceled error, got nil")
	}
	if resp.Error == "" {
		t.Fatal("expected ResponseModel.Error to be populated")
	}
}

type cancelRoundTripper struct{}

func (c *cancelRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	<-req.Context().Done()
	return nil, req.Context().Err()
}

func TestTracer_Durations(t *testing.T) {
	tc := NewTraceCollector()
	tr := tc.ClientTrace()

	tr.DNSStart(httptrace.DNSStartInfo{})
	time.Sleep(2 * time.Millisecond)
	tr.DNSDone(httptrace.DNSDoneInfo{})

	tr.ConnectStart("", "")
	time.Sleep(2 * time.Millisecond)
	tr.ConnectDone("", "", nil)

	tr.GotFirstResponseByte()
	tc.MarkBodyDone()

	telem := tc.Telemetry()
	if telem.Total < 0 {
		t.Errorf("expected total >= 0, got %d", telem.Total)
	}
}
