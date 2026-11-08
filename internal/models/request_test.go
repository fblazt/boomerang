package models

import (
	"testing"
)

func TestSyncURLToParams(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected []KeyValuePair
	}{
		{
			name:     "empty url",
			url:      "",
			expected: []KeyValuePair{},
		},
		{
			name:     "no query",
			url:      "https://api.domain.com/v1/users",
			expected: []KeyValuePair{},
		},
		{
			name: "single param",
			url:  "https://api.domain.com/v1/users?page=1",
			expected: []KeyValuePair{
				{Key: "page", Value: "1", Enabled: true},
			},
		},
		{
			name: "multiple params and fragment",
			url:  "{{base_url}}/api/items?search=hello+world&limit=50#section",
			expected: []KeyValuePair{
				{Key: "search", Value: "hello world", Enabled: true},
				{Key: "limit", Value: "50", Enabled: true},
			},
		},
		{
			name: "param without value",
			url:  "http://localhost:8080/test?flag&debug=true",
			expected: []KeyValuePair{
				{Key: "flag", Value: "", Enabled: true},
				{Key: "debug", Value: "true", Enabled: true},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SyncURLToParams(tc.url)
			if len(got) != len(tc.expected) {
				t.Fatalf("expected %d params, got %d", len(tc.expected), len(got))
			}
			for i := range got {
				if got[i].Key != tc.expected[i].Key || got[i].Value != tc.expected[i].Value || got[i].Enabled != tc.expected[i].Enabled {
					t.Errorf("item %d mismatch: got %+v, expected %+v", i, got[i], tc.expected[i])
				}
			}
		})
	}
}

func TestSyncParamsToURL(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		params   []KeyValuePair
		expected string
	}{
		{
			name:     "empty params on clean url",
			rawURL:   "https://api.internal.com/v1/health",
			params:   []KeyValuePair{},
			expected: "https://api.internal.com/v1/health",
		},
		{
			name:   "sync enabled params",
			rawURL: "https://api.internal.com/v1/users",
			params: []KeyValuePair{
				{Key: "page", Value: "2", Enabled: true},
				{Key: "sort", Value: "desc", Enabled: false},
				{Key: "filter", Value: "active", Enabled: true},
			},
			expected: "https://api.internal.com/v1/users?page=2&filter=active",
		},
		{
			name:   "replace existing query and preserve fragment",
			rawURL: "{{base_url}}/items?old=1#hash",
			params: []KeyValuePair{
				{Key: "new", Value: "val", Enabled: true},
			},
			expected: "{{base_url}}/items?new=val#hash",
		},
		{
			name:   "remove query when all disabled",
			rawURL: "https://example.com/api?a=1&b=2",
			params: []KeyValuePair{
				{Key: "a", Value: "1", Enabled: false},
				{Key: "b", Value: "2", Enabled: false},
			},
			expected: "https://example.com/api",
		},
		{
			name:   "preserve variable template tokens",
			rawURL: "{{base_url}}/items",
			params: []KeyValuePair{
				{Key: "token", Value: "{{jwt_token}}", Enabled: true},
			},
			expected: "{{base_url}}/items?token={{jwt_token}}",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SyncParamsToURL(tc.rawURL, tc.params)
			if got != tc.expected {
				t.Errorf("got %q, expected %q", got, tc.expected)
			}
		})
	}
}

func TestMergeURLParams(t *testing.T) {
	existing := []KeyValuePair{
		{Key: "page", Value: "1", Enabled: true, Description: "Current page"},
		{Key: "debug", Value: "true", Enabled: false, Description: "Debug mode flag"},
	}

	// 1. Update existing key and add new key from URL
	newURL := "https://api.internal.com/v1/users?page=2&limit=50"
	merged := MergeURLParams(existing, newURL)

	if len(merged) != 3 {
		t.Fatalf("expected 3 params (page, limit, and preserved debug), got %d", len(merged))
	}

	// Check page (value updated, description preserved)
	if merged[0].Key != "page" || merged[0].Value != "2" || merged[0].Description != "Current page" || !merged[0].Enabled {
		t.Errorf("page param not correctly merged: %+v", merged[0])
	}

	// Check limit (new enabled param)
	if merged[1].Key != "limit" || merged[1].Value != "50" || !merged[1].Enabled {
		t.Errorf("limit param not correctly added: %+v", merged[1])
	}

	// Check debug (disabled param preserved)
	if merged[2].Key != "debug" || merged[2].Value != "true" || merged[2].Enabled || merged[2].Description != "Debug mode flag" {
		t.Errorf("debug disabled param not preserved: %+v", merged[2])
	}
}
