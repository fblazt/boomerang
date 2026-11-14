package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fblazt/boomerang/internal/models"
)

// Client executes GraphQL schema introspection queries and typed document operations.
type Client struct {
	httpClient *http.Client
}

// NewClient constructs a GraphQL executor using httpClient (or a 30s default if nil).
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{httpClient: httpClient}
}

// Introspect issues the standard GraphQL __schema query and parses the type hierarchy.
func (c *Client) Introspect(ctx context.Context, endpoint string, headers map[string]string) (*models.GraphQLSchema, error) {
	if strings.TrimSpace(endpoint) == "" {
		return nil, fmt.Errorf("graphql endpoint cannot be empty")
	}

	raw, statusCode, statusText, _, err := c.postGraphQL(ctx, endpoint, headers, map[string]any{
		"query": IntrospectionQuery,
	})
	if err != nil {
		return nil, err
	}
	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("graphql introspection failed: %s", statusText)
	}
	return ParseIntrospection(raw)
}

// ExecuteQuery sends a GraphQL query/mutation with parsed JSON variables and collects telemetry.
func (c *Client) ExecuteQuery(ctx context.Context, req models.GraphQLRequest) (*models.GraphQLResponse, error) {
	if strings.TrimSpace(req.Endpoint) == "" {
		return nil, fmt.Errorf("graphql endpoint cannot be empty")
	}

	var vars any = map[string]any{}
	if trimmed := strings.TrimSpace(req.VariablesJSON); trimmed != "" {
		if err := json.Unmarshal([]byte(trimmed), &vars); err != nil {
			return &models.GraphQLResponse{
				RequestError: fmt.Sprintf("invalid variables JSON: %v", err),
			}, err
		}
	}

	payload := map[string]any{
		"query":     req.Query,
		"variables": vars,
	}
	if strings.TrimSpace(req.OperationName) != "" {
		payload["operationName"] = strings.TrimSpace(req.OperationName)
	}

	hdrs := make(map[string]string, len(req.Headers))
	for _, h := range req.Headers {
		if h.Enabled && strings.TrimSpace(h.Key) != "" {
			hdrs[h.Key] = h.Value
		}
	}

	raw, statusCode, statusText, elapsed, err := c.postGraphQL(ctx, req.Endpoint, hdrs, payload)
	if err != nil {
		return &models.GraphQLResponse{
			DurationMs:   elapsed.Milliseconds(),
			RequestError: err.Error(),
		}, err
	}

	resp := &models.GraphQLResponse{
		StatusCode:     statusCode,
		StatusText:     statusText,
		DurationMs:     elapsed.Milliseconds(),
		ResolverTimeMs: float64(elapsed.Microseconds()) / 1000.0,
		RawJSON:        prettyJSON(raw),
	}

	var env struct {
		Data       json.RawMessage        `json:"data"`
		Errors     []models.GraphQLError  `json:"errors"`
		Extensions map[string]interface{} `json:"extensions"`
	}
	if json.Unmarshal(raw, &env) == nil {
		resp.Errors = env.Errors
		if len(env.Data) > 0 {
			resp.DataJSON = prettyJSON(env.Data)
		}
		if len(env.Extensions) > 0 {
			if extBytes, mErr := json.MarshalIndent(env.Extensions, "", "  "); mErr == nil {
				resp.ExtensionsJSON = string(extBytes)
			}
			if rt, ok := extractResolverTimeMs(env.Extensions); ok {
				resp.ResolverTimeMs = rt
			}
		}
	}
	if resp.DataJSON == "" {
		resp.DataJSON = resp.RawJSON
	}

	return resp, nil
}

func (c *Client) postGraphQL(
	ctx context.Context,
	endpoint string,
	headers map[string]string,
	body any,
) ([]byte, int, string, time.Duration, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, 0, "", 0, fmt.Errorf("encode graphql payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return nil, 0, "", 0, fmt.Errorf("build graphql request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	start := time.Now()
	httpResp, err := c.httpClient.Do(httpReq)
	elapsed := time.Since(start)
	if err != nil {
		return nil, 0, "", elapsed, err
	}
	defer httpResp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(httpResp.Body, 16<<20))
	if err != nil {
		return nil, httpResp.StatusCode, httpResp.Status, elapsed, fmt.Errorf("read graphql response: %w", err)
	}
	return raw, httpResp.StatusCode, httpResp.Status, elapsed, nil
}

func prettyJSON(raw []byte) string {
	var buf bytes.Buffer
	if json.Indent(&buf, raw, "", "  ") == nil {
		return buf.String()
	}
	return string(raw)
}

func extractResolverTimeMs(ext map[string]interface{}) (float64, bool) {
	for _, key := range []string{"resolver_time", "resolver_time_ms", "resolverTime"} {
		if raw, ok := ext[key]; ok {
			if ms, ok := parseDurationNumberMs(raw); ok {
				return ms, true
			}
		}
	}
	if tracing, ok := ext["tracing"].(map[string]interface{}); ok {
		if durNs, ok := parseDurationNumberMs(tracing["duration"]); ok {
			return durNs / 1e6, true
		}
	}
	return 0, false
}

func parseDurationNumberMs(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int64:
		return float64(val), true
	case string:
		clean := strings.TrimSuffix(strings.TrimSpace(val), "ms")
		f, err := strconv.ParseFloat(clean, 64)
		return f, err == nil
	default:
		return 0, false
	}
}
