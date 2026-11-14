package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fblazt/boomerang/internal/engine/storage"
	"github.com/fblazt/boomerang/internal/models"
)

func parseBrunoCollection(input, sourceHint string, now time.Time) (models.Collection, []string, error) {
	item, err := parseSingleBru(input, sourceHint, now)
	if err != nil {
		return models.Collection{}, nil, err
	}

	colName := "Bruno Imports"
	if strings.TrimSpace(sourceHint) != "" && sourceHint != "paste" {
		colName = strings.TrimSuffix(sourceHint, filepath.Ext(sourceHint))
	}

	return models.Collection{
		ID:        storage.NewID("col"),
		Name:      colName,
		CreatedAt: now,
		UpdatedAt: now,
		Items:     []models.CollectionItem{item},
	}, nil, nil
}

func parseBrunoDirectory(dir string) (models.ImportPreview, error) {
	now := time.Now().UTC()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return models.ImportPreview{}, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	col := models.Collection{
		ID:        storage.NewID("col"),
		Name:      filepath.Base(dir),
		CreatedAt: now,
		UpdatedAt: now,
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".bru") {
			continue
		}
		if strings.EqualFold(entry.Name(), "bruno.bru") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		item, err := parseSingleBru(string(raw), entry.Name(), now)
		if err == nil {
			item.Order = len(col.Items)
			col.Items = append(col.Items, item)
		}
	}

	if len(col.Items) == 0 {
		return models.ImportPreview{}, fmt.Errorf("no valid .bru request files found in %s", dir)
	}

	folders, reqs := countTreeTotals(col)
	return models.ImportPreview{
		Format:       models.ImportFormatBruno,
		FormatLabel:  models.ImportFormatBruno.DisplayName(),
		Collection:   col,
		RequestCount: reqs,
		FolderCount:  folders,
	}, nil
}

func parseSingleBru(input, sourceHint string, now time.Time) (models.CollectionItem, error) {
	blocks := parseBruBlocks(input)
	if len(blocks) == 0 {
		return models.CollectionItem{}, fmt.Errorf("no Bruno DSL blocks found")
	}

	req := models.RequestModel{
		ID:       storage.NewID("req"),
		BodyType: "none",
		AuthType: "none",
	}

	if meta, ok := blocks["meta"]; ok {
		if kv := parseBruKVLines(meta); kv["name"] != "" {
			req.Name = kv["name"]
		}
	}

	for _, m := range []string{"get", "post", "put", "patch", "delete", "head", "options"} {
		if blk, ok := blocks[m]; ok {
			req.Method = strings.ToUpper(m)
			kv := parseBruKVLines(blk)
			req.URL = kv["url"]
			if req.BodyType == "none" && kv["body"] != "" && kv["body"] != "none" {
				req.BodyType = kv["body"]
			}
			break
		}
	}
	if req.Method == "" {
		req.Method = "GET"
	}
	if req.URL == "" {
		return models.CollectionItem{}, fmt.Errorf("Bruno file missing HTTP block URL")
	}
	if req.Name == "" {
		req.Name = strings.TrimSuffix(sourceHint, filepath.Ext(sourceHint))
		if req.Name == "" || req.Name == "paste" {
			req.Name = deriveNameFromURL(req.Method, req.URL)
		}
	}

	if qBlk, ok := blocks["params:query"]; ok {
		for _, pair := range parseBruOrderedKVs(qBlk) {
			req.Params = append(req.Params, pair)
		}
	} else {
		req.Params = models.SyncURLToParams(req.URL)
	}

	if hBlk, ok := blocks["headers"]; ok {
		req.Headers = append(req.Headers, parseBruOrderedKVs(hBlk)...)
	}

	if bBlk, ok := blocks["body:json"]; ok {
		req.BodyType = "json"
		req.Body = strings.TrimSpace(bBlk)
	} else if tBlk, ok := blocks["body:text"]; ok {
		req.BodyType = "text"
		req.Body = strings.TrimSpace(tBlk)
	} else if uBlk, ok := blocks["body:form-urlencoded"]; ok {
		req.BodyType = "urlencoded"
		req.URLEncoded = parseBruOrderedKVs(uBlk)
	}

	if authBlk, ok := blocks["auth:bearer"]; ok {
		kv := parseBruKVLines(authBlk)
		req.AuthType = "bearer"
		req.Token = kv["token"]
	} else if authBlk, ok := blocks["auth:basic"]; ok {
		kv := parseBruKVLines(authBlk)
		req.AuthType = "basic"
		req.BasicUser = kv["username"]
		req.BasicPassword = kv["password"]
	}

	if tBlk, ok := blocks["tests"]; ok {
		req.TestScript = strings.TrimSpace(tBlk)
	}

	return models.CollectionItem{
		ID:        storage.NewID("item"),
		Name:      req.Name,
		Request:   req,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func parseBruBlocks(input string) map[string]string {
	blocks := make(map[string]string)
	lines := strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")

	var curName string
	var buf strings.Builder
	depth := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if depth == 0 {
			if strings.HasSuffix(trimmed, "{") {
				curName = strings.ToLower(strings.TrimSpace(strings.TrimSuffix(trimmed, "{")))
				buf.Reset()
				depth = 1
			}
			continue
		}

		if trimmed == "}" && depth == 1 {
			blocks[curName] = strings.TrimSuffix(buf.String(), "\n")
			curName = ""
			depth = 0
			continue
		}

		// Track nested braces inside JSON/tests blocks accurately.
		for i := 0; i < len(line); i++ {
			if line[i] == '{' {
				depth++
			} else if line[i] == '}' {
				depth--
				if depth == 0 {
					blocks[curName] = strings.TrimSuffix(buf.String(), "\n")
					curName = ""
					goto nextLine
				}
			}
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
	nextLine:
	}

	return blocks
}

func parseBruKVLines(body string) map[string]string {
	out := make(map[string]string)
	for _, p := range parseBruOrderedKVs(body) {
		out[strings.ToLower(p.Key)] = p.Value
	}
	return out
}

func parseBruOrderedKVs(body string) []models.KeyValuePair {
	var out []models.KeyValuePair
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		enabled := true
		if strings.HasPrefix(t, "~") {
			enabled = false
			t = strings.TrimSpace(t[1:])
		}
		k, v, ok := strings.Cut(t, ":")
		if !ok {
			continue
		}
		out = append(out, models.KeyValuePair{
			Key:     strings.TrimSpace(k),
			Value:   strings.TrimSpace(v),
			Enabled: enabled,
		})
	}
	return out
}
