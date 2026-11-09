package env

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fblazt/boomerang/internal/models"
)

var varRegex = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_$.-]+)\s*\}\}`)

// DefaultMaxInterpolationDepth limits recursion to prevent infinite loops from circular references.
const DefaultMaxInterpolationDepth = 10

// GenerateUUID generates a standard RFC 4122 v4 UUID using crypto/rand.
func GenerateUUID() string {
	var uuid [16]byte
	_, _ = rand.Read(uuid[:])
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant RFC 4122
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}

// GenerateRandomInt returns a cryptographically uniform random int in [0, max).
func GenerateRandomInt(max int64) int64 {
	if max <= 0 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return time.Now().UnixNano() % max
	}
	return n.Int64()
}

// resolveDynamicVariable evaluates built-in dynamic variables prefixed with '$'.
func resolveDynamicVariable(name string) (string, bool) {
	switch strings.ToLower(name) {
	case "$guid", "$uuid":
		return GenerateUUID(), true
	case "$timestamp":
		return strconv.FormatInt(time.Now().Unix(), 10), true
	case "$timestampms":
		return strconv.FormatInt(time.Now().UnixMilli(), 10), true
	case "$isotimestamp":
		return time.Now().UTC().Format(time.RFC3339), true
	case "$randomint":
		return strconv.FormatInt(GenerateRandomInt(1000), 10), true
	case "$randomemail":
		return fmt.Sprintf("user_%d@example.com", GenerateRandomInt(90000)+10000), true
	default:
		return "", false
	}
}

// Interpolate recursively replaces template variables {{var_name}} in a string using vars and dynamic variables.
func Interpolate(template string, vars map[string]string) string {
	return InterpolateWithMaxDepth(template, vars, DefaultMaxInterpolationDepth)
}

// InterpolateWithMaxDepth performs variable substitution with a specified maximum recursion depth.
func InterpolateWithMaxDepth(template string, vars map[string]string, maxDepth int) string {
	if template == "" || !strings.Contains(template, "{{") {
		return template
	}

	current := template
	for depth := 0; depth < maxDepth; depth++ {
		changed := false
		replaced := varRegex.ReplaceAllStringFunc(current, func(match string) string {
			sub := varRegex.FindStringSubmatch(match)
			if len(sub) < 2 {
				return match
			}
			key := strings.TrimSpace(sub[1])

			// 1. Check for dynamic variables first ($guid, $timestamp, etc.)
			if val, ok := resolveDynamicVariable(key); ok {
				changed = true
				return val
			}

			// 2. Check in provided variables map
			if vars != nil {
				if val, exists := vars[key]; exists {
					changed = true
					return val
				}
			}

			// Unresolved variable: retain original match
			return match
		})

		if !changed || replaced == current {
			break
		}
		current = replaced
		if !strings.Contains(current, "{{") {
			break
		}
	}

	return current
}

// ExtractVariables scans a string and returns all unique variable names enclosed in {{...}}.
func ExtractVariables(template string) []string {
	if template == "" || !strings.Contains(template, "{{") {
		return nil
	}

	matches := varRegex.FindAllStringSubmatch(template, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	var result []string
	for _, m := range matches {
		if len(m) > 1 {
			name := strings.TrimSpace(m[1])
			if !seen[name] {
				seen[name] = true
				result = append(result, name)
			}
		}
	}
	return result
}

// InterpolateKeyValuePairs interpolates keys and values for a list of KeyValuePair items.
func InterpolateKeyValuePairs(pairs []models.KeyValuePair, vars map[string]string) []models.KeyValuePair {
	if len(pairs) == 0 {
		return nil
	}
	out := make([]models.KeyValuePair, len(pairs))
	for i, p := range pairs {
		out[i] = models.KeyValuePair{
			Key:         Interpolate(p.Key, vars),
			Value:       Interpolate(p.Value, vars),
			Enabled:     p.Enabled,
			Description: p.Description,
			Secret:      p.Secret,
		}
	}
	return out
}

// InterpolateRequest produces a deep copy of RequestModel with all templated strings interpolated.
func InterpolateRequest(req models.RequestModel, vars map[string]string) models.RequestModel {
	out := req

	out.URL = Interpolate(req.URL, vars)
	out.Params = InterpolateKeyValuePairs(req.Params, vars)
	out.Headers = InterpolateKeyValuePairs(req.Headers, vars)
	if req.Body != "" {
		out.Body = Interpolate(req.Body, vars)
	}
	out.FormData = InterpolateKeyValuePairs(req.FormData, vars)
	out.URLEncoded = InterpolateKeyValuePairs(req.URLEncoded, vars)

	out.Token = Interpolate(req.Token, vars)
	out.BasicUser = Interpolate(req.BasicUser, vars)
	out.BasicPassword = Interpolate(req.BasicPassword, vars)
	out.APIKeyKey = Interpolate(req.APIKeyKey, vars)
	out.APIKeyValue = Interpolate(req.APIKeyValue, vars)

	return out
}
