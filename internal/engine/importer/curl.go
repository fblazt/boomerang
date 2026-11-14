package importer

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/fblazt/boomerang/internal/engine/storage"
	"github.com/fblazt/boomerang/internal/models"
)

func parseCurlCollection(input, sourceHint string, now time.Time) (models.Collection, []string, error) {
	tokens, err := tokenizeShell(input)
	if err != nil {
		return models.Collection{}, nil, err
	}
	if len(tokens) == 0 || !strings.EqualFold(tokens[0], "curl") {
		return models.Collection{}, nil, fmt.Errorf("input does not start with curl")
	}

	req := models.RequestModel{
		ID:       storage.NewID("req"),
		Method:   "",
		BodyType: "none",
		AuthType: "none",
	}

	var (
		warnings  []string
		rawURL    string
		dataParts []string
		isForm    bool
		isGetFlag bool
		formPairs []models.KeyValuePair
	)

	for i := 1; i < len(tokens); i++ {
		tok := tokens[i]
		nextArg := func() (string, bool) {
			if i+1 < len(tokens) {
				i++
				return tokens[i], true
			}
			return "", false
		}

		switch {
		case tok == "-X" || tok == "--request":
			if v, ok := nextArg(); ok {
				req.Method = strings.ToUpper(strings.TrimSpace(v))
			}
		case strings.HasPrefix(tok, "-X") && len(tok) > 2:
			req.Method = strings.ToUpper(strings.TrimSpace(tok[2:]))
		case strings.HasPrefix(tok, "--request="):
			req.Method = strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(tok, "--request=")))

		case tok == "-H" || tok == "--header":
			if v, ok := nextArg(); ok {
				appendHeaderLine(&req, v)
			}
		case strings.HasPrefix(tok, "--header="):
			appendHeaderLine(&req, strings.TrimPrefix(tok, "--header="))

		case tok == "-d" || tok == "--data" || tok == "--data-raw" || tok == "--data-binary" || tok == "--data-urlencode" || tok == "--data-ascii":
			if v, ok := nextArg(); ok {
				dataParts = append(dataParts, v)
			}
		case strings.HasPrefix(tok, "--data-raw="):
			dataParts = append(dataParts, strings.TrimPrefix(tok, "--data-raw="))
		case strings.HasPrefix(tok, "--data="):
			dataParts = append(dataParts, strings.TrimPrefix(tok, "--data="))

		case tok == "-F" || tok == "--form":
			if v, ok := nextArg(); ok {
				isForm = true
				k, val, _ := strings.Cut(v, "=")
				formPairs = append(formPairs, models.KeyValuePair{Key: k, Value: val, Enabled: true})
			}

		case tok == "-u" || tok == "--user":
			if v, ok := nextArg(); ok {
				user, pass, _ := strings.Cut(v, ":")
				req.AuthType = "basic"
				req.BasicUser = user
				req.BasicPassword = pass
			}

		case tok == "-A" || tok == "--user-agent":
			if v, ok := nextArg(); ok {
				req.Headers = append(req.Headers, models.KeyValuePair{Key: "User-Agent", Value: v, Enabled: true})
			}

		case tok == "-G" || tok == "--get":
			isGetFlag = true

		case tok == "--url":
			if v, ok := nextArg(); ok {
				rawURL = v
			}

		case tok == "-k" || tok == "--insecure":
			f := false
			req.StrictSSL = &f

		case tok == "-L" || tok == "--location":
			t := true
			req.FollowRedirects = &t

		case tok == "-i" || tok == "--include" || tok == "-s" || tok == "--silent" || tok == "-v" || tok == "--verbose" || tok == "--compressed":
			// Valid curl flags with no argument

		case strings.HasPrefix(tok, "-"):
			warnings = append(warnings, "ignored curl flag: "+tok)

		default:
			if rawURL == "" {
				rawURL = tok
			}
		}
	}

	if rawURL == "" {
		return models.Collection{}, nil, fmt.Errorf("curl command has no URL")
	}

	req.URL = rawURL
	req.Params = models.SyncURLToParams(rawURL)

	// Inspect Authorization header if not already set via -u
	for _, h := range req.Headers {
		if strings.EqualFold(h.Key, "Authorization") {
			if strings.HasPrefix(h.Value, "Bearer ") {
				req.AuthType = "bearer"
				req.Token = strings.TrimSpace(strings.TrimPrefix(h.Value, "Bearer "))
			} else if strings.HasPrefix(h.Value, "Basic ") && req.AuthType == "none" {
				rawCred := strings.TrimSpace(strings.TrimPrefix(h.Value, "Basic "))
				if dec, err := base64.StdEncoding.DecodeString(rawCred); err == nil {
					if u, p, ok := strings.Cut(string(dec), ":"); ok {
						req.AuthType = "basic"
						req.BasicUser = u
						req.BasicPassword = p
					}
				}
			}
		}
	}

	if isForm {
		req.BodyType = "form-data"
		req.FormData = formPairs
	} else if len(dataParts) > 0 {
		combined := strings.Join(dataParts, "&")
		if isGetFlag {
			sep := "?"
			if strings.Contains(req.URL, "?") {
				sep = "&"
			}
			req.URL += sep + combined
			req.Params = models.SyncURLToParams(req.URL)
		} else {
			req.Body = combined
			trimmed := strings.TrimSpace(combined)
			ct := headerValue(req.Headers, "Content-Type")
			switch {
			case strings.Contains(strings.ToLower(ct), "application/json") || strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "["):
				req.BodyType = "json"
			case strings.Contains(strings.ToLower(ct), "application/x-www-form-urlencoded"):
				req.BodyType = "urlencoded"
				if vals, err := url.ParseQuery(combined); err == nil {
					for k, list := range vals {
						for _, v := range list {
							req.URLEncoded = append(req.URLEncoded, models.KeyValuePair{Key: k, Value: v, Enabled: true})
						}
					}
				}
			default:
				req.BodyType = "text"
			}
		}
	}

	if req.Method == "" {
		if len(dataParts) > 0 && !isGetFlag {
			req.Method = "POST"
		} else {
			req.Method = "GET"
		}
	}

	reqName := deriveNameFromURL(req.Method, req.URL)
	req.Name = reqName

	colName := "cURL Imports"
	if strings.TrimSpace(sourceHint) != "" && sourceHint != "paste" {
		colName = sourceHint
	}

	return models.Collection{
		ID:        storage.NewID("col"),
		Name:      colName,
		CreatedAt: now,
		UpdatedAt: now,
		Items: []models.CollectionItem{
			{
				ID:        storage.NewID("item"),
				Name:      reqName,
				Order:     0,
				Request:   req,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}, warnings, nil
}

func appendHeaderLine(req *models.RequestModel, line string) {
	k, v, ok := strings.Cut(line, ":")
	if !ok {
		return
	}
	req.Headers = append(req.Headers, models.KeyValuePair{
		Key:     strings.TrimSpace(k),
		Value:   strings.TrimSpace(v),
		Enabled: true,
	})
}

func headerValue(headers []models.KeyValuePair, key string) string {
	for _, h := range headers {
		if strings.EqualFold(h.Key, key) {
			return h.Value
		}
	}
	return ""
}

func deriveNameFromURL(method, rawURL string) string {
	u, err := url.Parse(rawURL)
	if err == nil && u.Path != "" && u.Path != "/" {
		return method + " " + u.Path
	}
	if err == nil && u.Host != "" {
		return method + " " + u.Host
	}
	return method + " Request"
}

func tokenizeShell(input string) ([]string, error) {
	// Normalize backslash-newline line continuations.
	normalized := strings.ReplaceAll(input, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\\\n", " ")

	var (
		tokens  []string
		buf     strings.Builder
		inTok   bool
		quote   byte
		escaped bool
	)

	flush := func() {
		if inTok {
			tokens = append(tokens, buf.String())
			buf.Reset()
			inTok = false
		}
	}

	for i := 0; i < len(normalized); i++ {
		ch := normalized[i]
		if escaped {
			buf.WriteByte(ch)
			inTok = true
			escaped = false
			continue
		}

		if quote == '\'' {
			if ch == '\'' {
				quote = 0
				continue
			}
			buf.WriteByte(ch)
			inTok = true
			continue
		}

		if quote == '"' {
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				quote = 0
				continue
			}
			buf.WriteByte(ch)
			inTok = true
			continue
		}

		switch {
		case ch == '\'' || ch == '"':
			quote = ch
			inTok = true
		case ch == '\\':
			escaped = true
			inTok = true
		case ch == ' ' || ch == '\t' || ch == '\n':
			flush()
		default:
			if ch == '$' && i+1 < len(normalized) && normalized[i+1] == '\'' {
				quote = '\''
				inTok = true
				i++
				continue
			}
			buf.WriteByte(ch)
			inTok = true
		}
	}

	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote in cURL command")
	}
	flush()
	return tokens, nil
}
