package importer

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fblazt/boomerang/internal/engine/storage"
	"github.com/fblazt/boomerang/internal/models"
	"gopkg.in/yaml.v3"
)

type openAPIDoc struct {
	OpenAPI string `json:"openapi" yaml:"openapi"`
	Info    struct {
		Title       string `json:"title" yaml:"title"`
		Description string `json:"description" yaml:"description"`
		Version     string `json:"version" yaml:"version"`
	} `json:"info" yaml:"info"`
	Servers []struct {
		URL string `json:"url" yaml:"url"`
	} `json:"servers" yaml:"servers"`
	Paths map[string]map[string]openAPIOp `json:"paths" yaml:"paths"`
}

type openAPIOp struct {
	Summary     string              `json:"summary" yaml:"summary"`
	OperationID string              `json:"operationId" yaml:"operationId"`
	Tags        []string            `json:"tags" yaml:"tags"`
	Parameters  []openAPIParam      `json:"parameters" yaml:"parameters"`
	RequestBody *openAPIRequestBody `json:"requestBody" yaml:"requestBody"`
	Security    []map[string][]any  `json:"security" yaml:"security"`
}

type openAPIParam struct {
	Name        string `json:"name" yaml:"name"`
	In          string `json:"in" yaml:"in"`
	Description string `json:"description" yaml:"description"`
	Required    bool   `json:"required" yaml:"required"`
	Example     any    `json:"example" yaml:"example"`
	Schema      *struct {
		Default any    `json:"default" yaml:"default"`
		Example any    `json:"example" yaml:"example"`
		Type    string `json:"type" yaml:"type"`
	} `json:"schema" yaml:"schema"`
}

type openAPIRequestBody struct {
	Content map[string]openAPIMedia `json:"content" yaml:"content"`
}

type openAPIMedia struct {
	Example any            `json:"example" yaml:"example"`
	Schema  *openAPISchema `json:"schema" yaml:"schema"`
}

type openAPISchema struct {
	Type       string                    `json:"type" yaml:"type"`
	Example    any                       `json:"example" yaml:"example"`
	Properties map[string]*openAPISchema `json:"properties" yaml:"properties"`
	Items      *openAPISchema            `json:"items" yaml:"items"`
}

func parseOpenAPICollection(input, sourceHint string, now time.Time) (models.Collection, []string, error) {
	var doc openAPIDoc
	trimmed := strings.TrimSpace(input)
	if strings.HasPrefix(trimmed, "{") {
		if err := json.Unmarshal([]byte(trimmed), &doc); err != nil {
			return models.Collection{}, nil, fmt.Errorf("invalid OpenAPI JSON: %w", err)
		}
	} else {
		if err := yaml.Unmarshal([]byte(trimmed), &doc); err != nil {
			return models.Collection{}, nil, fmt.Errorf("invalid OpenAPI YAML: %w", err)
		}
	}

	if doc.OpenAPI == "" && len(doc.Paths) == 0 {
		return models.Collection{}, nil, fmt.Errorf("input is missing OpenAPI openapi/paths fields")
	}

	baseURL := "{{base_url}}"
	if len(doc.Servers) > 0 && strings.TrimSpace(doc.Servers[0].URL) != "" {
		baseURL = strings.TrimRight(strings.TrimSpace(doc.Servers[0].URL), "/")
	}

	colName := strings.TrimSpace(doc.Info.Title)
	if colName == "" {
		colName = strings.TrimSpace(sourceHint)
	}
	if colName == "" {
		colName = "OpenAPI Specification"
	}

	col := models.Collection{
		ID:          storage.NewID("col"),
		Name:        colName,
		Description: doc.Info.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Deterministic path & method iteration order.
	paths := make([]string, 0, len(doc.Paths))
	for p := range doc.Paths {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	folderIndex := make(map[string]int)
	methodOrder := []string{"get", "post", "put", "patch", "delete", "head", "options"}

	for _, path := range paths {
		ops := doc.Paths[path]
		for _, m := range methodOrder {
			op, ok := ops[m]
			if !ok {
				continue
			}

			method := strings.ToUpper(m)
			reqName := strings.TrimSpace(op.Summary)
			if reqName == "" {
				reqName = strings.TrimSpace(op.OperationID)
			}
			if reqName == "" {
				reqName = method + " " + path
			}

			fullURL := baseURL + path
			if !strings.HasPrefix(path, "/") {
				fullURL = baseURL + "/" + path
			}

			req := models.RequestModel{
				ID:       storage.NewID("req"),
				Name:     reqName,
				Method:   method,
				URL:      fullURL,
				BodyType: "none",
				AuthType: "none",
			}

			for _, p := range op.Parameters {
				val := paramDefaultValue(p)
				switch strings.ToLower(p.In) {
				case "query":
					req.Params = append(req.Params, models.KeyValuePair{
						Key:         p.Name,
						Value:       val,
						Description: p.Description,
						Enabled:     p.Required,
					})
				case "header":
					req.Headers = append(req.Headers, models.KeyValuePair{
						Key:         p.Name,
						Value:       val,
						Description: p.Description,
						Enabled:     p.Required,
					})
				}
			}
			if len(req.Params) > 0 {
				req.URL = models.SyncParamsToURL(req.URL, req.Params)
			}

			if len(op.Security) > 0 {
				for scheme := range op.Security[0] {
					if strings.Contains(strings.ToLower(scheme), "bearer") || strings.Contains(strings.ToLower(scheme), "jwt") {
						req.AuthType = "bearer"
						req.Token = "{{api_token}}"
						break
					}
				}
			}

			if op.RequestBody != nil {
				if media, ok := op.RequestBody.Content["application/json"]; ok {
					req.BodyType = "json"
					req.Headers = append(req.Headers, models.KeyValuePair{
						Key:     "Content-Type",
						Value:   "application/json",
						Enabled: true,
					})
					if example := buildMediaExampleJSON(media); example != "" {
						req.Body = example
					} else {
						req.Body = "{}"
					}
				}
			}

			item := models.CollectionItem{
				ID:        storage.NewID("item"),
				Name:      reqName,
				Request:   req,
				CreatedAt: now,
				UpdatedAt: now,
			}

			if len(op.Tags) > 0 && strings.TrimSpace(op.Tags[0]) != "" {
				tag := strings.TrimSpace(op.Tags[0])
				fIdx, exists := folderIndex[tag]
				if !exists {
					fIdx = len(col.Folders)
					folderIndex[tag] = fIdx
					col.Folders = append(col.Folders, models.Folder{
						ID:        storage.NewID("fld"),
						Name:      tag,
						Order:     fIdx,
						CreatedAt: now,
						UpdatedAt: now,
					})
				}
				item.Order = len(col.Folders[fIdx].Items)
				col.Folders[fIdx].Items = append(col.Folders[fIdx].Items, item)
			} else {
				item.Order = len(col.Items)
				col.Items = append(col.Items, item)
			}
		}
	}

	return col, nil, nil
}

func paramDefaultValue(p openAPIParam) string {
	if p.Example != nil {
		return fmt.Sprint(p.Example)
	}
	if p.Schema != nil {
		if p.Schema.Example != nil {
			return fmt.Sprint(p.Schema.Example)
		}
		if p.Schema.Default != nil {
			return fmt.Sprint(p.Schema.Default)
		}
	}
	return ""
}

func buildMediaExampleJSON(media openAPIMedia) string {
	var sample any
	if media.Example != nil {
		sample = media.Example
	} else if media.Schema != nil {
		sample = synthesizeSchemaSample(media.Schema)
	}
	if sample == nil {
		return ""
	}
	raw, err := json.MarshalIndent(sample, "", "  ")
	if err != nil {
		return ""
	}
	return string(raw)
}

func synthesizeSchemaSample(s *openAPISchema) any {
	if s == nil {
		return nil
	}
	if s.Example != nil {
		return s.Example
	}
	switch strings.ToLower(s.Type) {
	case "object":
		m := make(map[string]any, len(s.Properties))
		for k, prop := range s.Properties {
			m[k] = synthesizeSchemaSample(prop)
		}
		return m
	case "array":
		if s.Items != nil {
			return []any{synthesizeSchemaSample(s.Items)}
		}
		return []any{}
	case "integer", "number":
		return 0
	case "boolean":
		return false
	default:
		if len(s.Properties) > 0 {
			m := make(map[string]any, len(s.Properties))
			for k, prop := range s.Properties {
				m[k] = synthesizeSchemaSample(prop)
			}
			return m
		}
		return "string"
	}
}
