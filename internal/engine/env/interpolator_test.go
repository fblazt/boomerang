package env

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fblazt/boomerang/internal/models"
)

func TestInterpolate_Basic(t *testing.T) {
	vars := map[string]string{
		"base_url": "https://api.internal.net/v1",
		"version":  "v2",
		"userId":   "usr_42",
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"{{base_url}}/users/{{userId}}", "https://api.internal.net/v1/users/usr_42"},
		{"https://api.internal.net/{{version}}/ping", "https://api.internal.net/v2/ping"},
		{"No variables here", "No variables here"},
		{"", ""},
		{"{{unresolved_var}}", "{{unresolved_var}}"},
		{"Bearer {{ api_key }}", "Bearer {{ api_key }}"}, // not in map
	}

	for _, tt := range tests {
		got := Interpolate(tt.input, vars)
		if got != tt.expected {
			t.Errorf("Interpolate(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestInterpolate_WhitespaceInsideBraces(t *testing.T) {
	vars := map[string]string{
		"token": "secret_abc",
	}
	got := Interpolate("Bearer {{ token }}", vars)
	if got != "Bearer secret_abc" {
		t.Errorf("expected whitespace inside braces to resolve, got %q", got)
	}
}

func TestInterpolate_NestedAndRecursive(t *testing.T) {
	vars := map[string]string{
		"host":      "internal.net",
		"port":      "8080",
		"domain":    "{{host}}:{{port}}",
		"base_url":  "https://{{domain}}/v1",
		"users_url": "{{base_url}}/users",
	}

	got := Interpolate("{{users_url}}?limit=10", vars)
	expected := "https://internal.net:8080/v1/users?limit=10"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestInterpolate_CircularReferenceSafe(t *testing.T) {
	vars := map[string]string{
		"a": "{{b}}",
		"b": "{{a}}",
	}

	// Should terminate safely without hanging or stack overflow
	done := make(chan string)
	go func() {
		done <- Interpolate("{{a}}", vars)
	}()

	select {
	case res := <-done:
		if res != "{{b}}" && res != "{{a}}" {
			t.Errorf("unexpected circular result: %q", res)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Interpolate hung on circular reference")
	}
}

func TestInterpolate_SelfReferenceSafe(t *testing.T) {
	vars := map[string]string{
		"loop": "{{loop}}",
	}
	res := Interpolate("value: {{loop}}", vars)
	if res != "value: {{loop}}" {
		t.Errorf("expected 'value: {{loop}}', got %q", res)
	}
}

func TestInterpolate_DynamicVariables(t *testing.T) {
	guidStr := Interpolate("{{$guid}}", nil)
	guidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !guidRegex.MatchString(guidStr) {
		t.Fatalf("{{$guid}} produced invalid UUID v4: %q", guidStr)
	}

	uuidStr := Interpolate("{{$uuid}}", nil)
	if !guidRegex.MatchString(uuidStr) {
		t.Fatalf("{{$uuid}} produced invalid UUID v4: %q", uuidStr)
	}

	tsStr := Interpolate("{{$timestamp}}", nil)
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil || ts <= 0 {
		t.Fatalf("{{$timestamp}} produced invalid unix timestamp: %q", tsStr)
	}

	tsMsStr := Interpolate("{{$timestampMs}}", nil)
	tsMs, err := strconv.ParseInt(tsMsStr, 10, 64)
	if err != nil || tsMs < ts*1000 {
		t.Fatalf("{{$timestampMs}} produced invalid ms timestamp: %q", tsMsStr)
	}

	isoStr := Interpolate("{{$isoTimestamp}}", nil)
	_, err = time.Parse(time.RFC3339, isoStr)
	if err != nil {
		t.Fatalf("{{$isoTimestamp}} produced invalid RFC3339 time: %q", isoStr)
	}

	randIntStr := Interpolate("{{$randomInt}}", nil)
	randInt, err := strconv.Atoi(randIntStr)
	if err != nil || randInt < 0 || randInt >= 1000 {
		t.Fatalf("{{$randomInt}} produced out-of-range integer: %q", randIntStr)
	}

	emailStr := Interpolate("{{$randomEmail}}", nil)
	if !strings.HasPrefix(emailStr, "user_") || !strings.HasSuffix(emailStr, "@example.com") {
		t.Fatalf("{{$randomEmail}} produced invalid format: %q", emailStr)
	}
}

func TestExtractVariables(t *testing.T) {
	input := "https://{{host}}:{{port}}/v1/{{resource}}?token={{$guid}}&limit={{limit}}&dup={{host}}"
	extracted := ExtractVariables(input)

	expected := []string{"host", "port", "resource", "$guid", "limit"}
	if len(extracted) != len(expected) {
		t.Fatalf("expected %d extracted vars, got %d: %v", len(expected), len(extracted), extracted)
	}

	for i, exp := range expected {
		if extracted[i] != exp {
			t.Errorf("at index %d: expected %q, got %q", i, exp, extracted[i])
		}
	}

	if ExtractVariables("no braces") != nil {
		t.Error("expected nil for string without braces")
	}
	if ExtractVariables("") != nil {
		t.Error("expected nil for empty string")
	}
}

func TestInterpolateRequest(t *testing.T) {
	vars := map[string]string{
		"base_url": "https://api.internal.net",
		"endpoint": "users",
		"auth_tok": "eyJh...test",
		"q_key":    "role",
		"q_val":    "admin",
		"body_id":  "usr_99",
	}

	req := models.RequestModel{
		ID:     "req_1",
		Name:   "Test",
		Method: "POST",
		URL:    "{{base_url}}/v1/{{endpoint}}",
		Params: []models.KeyValuePair{
			{Key: "{{q_key}}", Value: "{{q_val}}", Enabled: true},
		},
		Headers: []models.KeyValuePair{
			{Key: "Authorization", Value: "Bearer {{auth_tok}}", Enabled: true},
		},
		Body:      `{"userId": "{{body_id}}"}`,
		AuthType:  "bearer",
		Token:     "{{auth_tok}}",
		APIKeyKey: "X-Key-{{q_key}}",
	}

	result := InterpolateRequest(req, vars)

	if result.URL != "https://api.internal.net/v1/users" {
		t.Errorf("unexpected URL: %s", result.URL)
	}
	if len(result.Params) != 1 || result.Params[0].Key != "role" || result.Params[0].Value != "admin" {
		t.Errorf("unexpected params: %+v", result.Params)
	}
	if len(result.Headers) != 1 || result.Headers[0].Value != "Bearer eyJh...test" {
		t.Errorf("unexpected headers: %+v", result.Headers)
	}
	if result.Body != `{"userId": "usr_99"}` {
		t.Errorf("unexpected body: %s", result.Body)
	}
	if result.Token != "eyJh...test" {
		t.Errorf("unexpected token: %s", result.Token)
	}
	if result.APIKeyKey != "X-Key-role" {
		t.Errorf("unexpected APIKeyKey: %s", result.APIKeyKey)
	}
}
