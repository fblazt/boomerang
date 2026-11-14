package storage

import (
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/fblazt/boomerang/internal/models"
)

// HARArchive represents a standard HTTP Archive (HAR 1.2) document.
type HARArchive struct {
	Log HARLog `json:"log"`
}

// HARLog is the root metadata & entries container in HAR 1.2.
type HARLog struct {
	Version string     `json:"version"`
	Creator HARCreator `json:"creator"`
	Entries []HAREntry `json:"entries"`
}

// HARCreator identifies the application that produced the HAR export.
type HARCreator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// HAREntry describes one executed HTTP request/response transaction in HAR 1.2.
type HAREntry struct {
	StartedDateTime string      `json:"startedDateTime"`
	Time            float64     `json:"time"`
	Request         HARRequest  `json:"request"`
	Response        HARResponse `json:"response"`
	Cache           struct{}    `json:"cache"`
	Timings         HARTimings  `json:"timings"`
	ServerIPAddress string      `json:"serverIPAddress,omitempty"`
	Comment         string      `json:"comment,omitempty"`
}

// HARRequest holds HTTP request attributes conforming to HAR 1.2.
type HARRequest struct {
	Method      string         `json:"method"`
	URL         string         `json:"url"`
	HTTPVersion string         `json:"httpVersion"`
	Cookies     []HARNameValue `json:"cookies"`
	Headers     []HARNameValue `json:"headers"`
	QueryString []HARNameValue `json:"queryString"`
	PostData    *HARPostData   `json:"postData,omitempty"`
	HeadersSize int            `json:"headersSize"`
	BodySize    int            `json:"bodySize"`
}

// HARResponse holds HTTP response attributes conforming to HAR 1.2.
type HARResponse struct {
	Status      int            `json:"status"`
	StatusText  string         `json:"statusText"`
	HTTPVersion string         `json:"httpVersion"`
	Cookies     []HARNameValue `json:"cookies"`
	Headers     []HARNameValue `json:"headers"`
	Content     HARContent     `json:"content"`
	RedirectURL string         `json:"redirectURL"`
	HeadersSize int            `json:"headersSize"`
	BodySize    int64          `json:"bodySize"`
}

// HARNameValue represents a header or query-parameter pair.
type HARNameValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HARPostData holds the request payload body and detected MIME type.
type HARPostData struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

// HARContent holds the response body text and MIME type.
type HARContent struct {
	Size     int64  `json:"size"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
}

// HARTimings captures detailed phase durations in milliseconds per HAR 1.2.
type HARTimings struct {
	DNS     int64 `json:"dns"`
	Connect int64 `json:"connect"`
	SSL     int64 `json:"ssl"`
	Send    int64 `json:"send"`
	Wait    int64 `json:"wait"`
	Receive int64 `json:"receive"`
}

// BuildHAR converts history entries into formatted HAR 1.2 JSON bytes.
func BuildHAR(entries []models.HistoryEntry) ([]byte, error) {
	harEntries := make([]HAREntry, 0, len(entries))
	for _, entry := range entries {
		harEntries = append(harEntries, toHAREntry(entry))
	}

	archive := HARArchive{
		Log: HARLog{
			Version: "1.2",
			Creator: HARCreator{
				Name:    "Boomerang",
				Version: "0.1.0",
			},
			Entries: harEntries,
		},
	}

	data, err := json.MarshalIndent(archive, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func toHAREntry(entry models.HistoryEntry) HAREntry {
	proto := entry.Protocol
	if proto == "" {
		proto = "HTTP/1.1"
	}

	reqHeaders := make([]HARNameValue, 0, len(entry.Request.Headers))
	reqMime := "application/octet-stream"
	for _, h := range entry.Request.Headers {
		if !h.Enabled || strings.TrimSpace(h.Key) == "" {
			continue
		}
		reqHeaders = append(reqHeaders, HARNameValue{Name: h.Key, Value: h.Value})
		if strings.EqualFold(h.Key, "content-type") {
			reqMime = h.Value
		}
	}
	if entry.Request.BodyType == "json" && reqMime == "application/octet-stream" {
		reqMime = "application/json"
	}

	queryPairs := extractHARQueryPairs(entry.URL, entry.Request.Params)

	var postData *HARPostData
	bodySize := len(entry.Request.Body)
	if bodySize > 0 {
		postData = &HARPostData{
			MimeType: reqMime,
			Text:     entry.Request.Body,
		}
	}

	harResp := HARResponse{
		Status:      entry.StatusCode,
		StatusText:  entry.StatusText,
		HTTPVersion: proto,
		Cookies:     []HARNameValue{},
		Headers:     []HARNameValue{},
		Content:     HARContent{MimeType: "application/octet-stream"},
		HeadersSize: -1,
		BodySize:    entry.SizeBytes,
	}
	if entry.Response != nil {
		respHeaders := make([]HARNameValue, 0, len(entry.Response.Headers))
		respMime := "text/plain"
		for k, v := range entry.Response.Headers {
			respHeaders = append(respHeaders, HARNameValue{Name: k, Value: v})
			if strings.EqualFold(k, "content-type") && v != "" {
				respMime = v
			}
		}
		harResp.Headers = respHeaders
		harResp.Content = HARContent{
			Size:     entry.Response.SizeBytes,
			MimeType: respMime,
			Text:     entry.Response.Body,
		}
	}

	return HAREntry{
		StartedDateTime: entry.ExecutedAt.UTC().Format(time.RFC3339Nano),
		Time:            float64(entry.DurationMs),
		Request: HARRequest{
			Method:      strings.ToUpper(entry.Method),
			URL:         entry.URL,
			HTTPVersion: proto,
			Cookies:     []HARNameValue{},
			Headers:     reqHeaders,
			QueryString: queryPairs,
			PostData:    postData,
			HeadersSize: -1,
			BodySize:    bodySize,
		},
		Response: harResp,
		Timings: HARTimings{
			DNS:     nonNeg(entry.Telemetry.DNS),
			Connect: nonNeg(entry.Telemetry.TCP),
			SSL:     nonNeg(entry.Telemetry.TLS),
			Send:    0,
			Wait:    nonNeg(entry.Telemetry.TTFB),
			Receive: nonNeg(entry.Telemetry.Download),
		},
		ServerIPAddress: entry.Telemetry.RemoteAddr,
		Comment:         entry.Error,
	}
}

func extractHARQueryPairs(rawURL string, params []models.KeyValuePair) []HARNameValue {
	var out []HARNameValue
	seen := make(map[string]bool)

	if u, err := url.Parse(rawURL); err == nil {
		for k, vals := range u.Query() {
			for _, v := range vals {
				out = append(out, HARNameValue{Name: k, Value: v})
				seen[k+"="+v] = true
			}
		}
	}
	for _, p := range params {
		if !p.Enabled || strings.TrimSpace(p.Key) == "" {
			continue
		}
		key := p.Key + "=" + p.Value
		if !seen[key] {
			out = append(out, HARNameValue{Name: p.Key, Value: p.Value})
		}
	}
	if out == nil {
		out = []HARNameValue{}
	}
	return out
}

func nonNeg(v int64) int64 {
	if v < 0 {
		return -1
	}
	return v
}
