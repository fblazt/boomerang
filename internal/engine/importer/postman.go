package importer

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fblazt/boomerang/internal/engine/storage"
	"github.com/fblazt/boomerang/internal/models"
)

type postmanDoc struct {
	Info struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Schema      string `json:"schema"`
	} `json:"info"`
	Item []postmanItem `json:"item"`
}

type postmanItem struct {
	Name    string          `json:"name"`
	Item    []postmanItem   `json:"item,omitempty"`
	Request *postmanRequest `json:"request,omitempty"`
	Event   []postmanEvent  `json:"event,omitempty"`
}

type postmanEvent struct {
	Listen string `json:"listen"`
	Script struct {
		Exec []string `json:"exec"`
	} `json:"script"`
}

type postmanRequest struct {
	Method string          `json:"method"`
	Header []postmanKV     `json:"header,omitempty"`
	URL    json.RawMessage `json:"url"`
	Body   *postmanBody    `json:"body,omitempty"`
	Auth   *postmanAuth    `json:"auth,omitempty"`
}

type postmanKV struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	Disabled    bool   `json:"disabled,omitempty"`
}

type postmanBody struct {
	Mode       string      `json:"mode"`
	Raw        string      `json:"raw,omitempty"`
	URLEncoded []postmanKV `json:"urlencoded,omitempty"`
	FormData   []postmanKV `json:"formdata,omitempty"`
	Options    struct {
		Raw struct {
			Language string `json:"language"`
		} `json:"raw"`
	} `json:"options,omitempty"`
}

type postmanAuth struct {
	Type   string      `json:"type"`
	Bearer []postmanKV `json:"bearer,omitempty"`
	Basic  []postmanKV `json:"basic,omitempty"`
	APIKey []postmanKV `json:"apikey,omitempty"`
}

type postmanURLObj struct {
	Raw   string      `json:"raw"`
	Query []postmanKV `json:"query,omitempty"`
}

func parsePostmanCollection(input string, now time.Time) (models.Collection, []string, error) {
	var doc postmanDoc
	if err := json.Unmarshal([]byte(input), &doc); err != nil {
		return models.Collection{}, nil, fmt.Errorf("invalid Postman JSON: %w", err)
	}
	if len(doc.Item) == 0 && doc.Info.Name == "" {
		return models.Collection{}, nil, fmt.Errorf("JSON is not a Postman collection")
	}

	col := models.Collection{
		ID:          storage.NewID("col"),
		Name:        strings.TrimSpace(doc.Info.Name),
		Description: doc.Info.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if col.Name == "" {
		col.Name = "Postman Collection"
	}

	var warnings []string
	col.Folders, col.Items = convertPostmanItems(doc.Item, now, &warnings)
	return col, warnings, nil
}

func convertPostmanItems(items []postmanItem, now time.Time, warnings *[]string) ([]models.Folder, []models.CollectionItem) {
	var (
		folders []models.Folder
		colItems []models.CollectionItem
	)

	for _, it := range items {
		if len(it.Item) > 0 && it.Request == nil {
			subFolders, subItems := convertPostmanItems(it.Item, now, warnings)
			folders = append(folders, models.Folder{
				ID:        storage.NewID("fld"),
				Name:      it.Name,
				Order:     len(folders),
				Folders:   subFolders,
				Items:     subItems,
				CreatedAt: now,
				UpdatedAt: now,
			})
			continue
		}

		if it.Request != nil {
			req := convertPostmanRequest(it.Name, *it.Request, it.Event)
			colItems = append(colItems, models.CollectionItem{
				ID:        storage.NewID("item"),
				Name:      it.Name,
				Order:     len(colItems),
				Request:   req,
				CreatedAt: now,
				UpdatedAt: now,
			})
			continue
		}

		// Empty folder without children
		folders = append(folders, models.Folder{
			ID:        storage.NewID("fld"),
			Name:      it.Name,
			Order:     len(folders),
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	return folders, colItems
}

func convertPostmanRequest(name string, pr postmanRequest, events []postmanEvent) models.RequestModel {
	method := strings.ToUpper(strings.TrimSpace(pr.Method))
	if method == "" {
		method = "GET"
	}

	rawURL, params := decodePostmanURL(pr.URL)
	if len(params) == 0 && rawURL != "" {
		params = models.SyncURLToParams(rawURL)
	}

	req := models.RequestModel{
		ID:       storage.NewID("req"),
		Name:     name,
		Method:   method,
		URL:      rawURL,
		Params:   params,
		BodyType: "none",
		AuthType: "none",
	}

	for _, h := range pr.Header {
		req.Headers = append(req.Headers, models.KeyValuePair{
			Key:         h.Key,
			Value:       h.Value,
			Description: h.Description,
			Enabled:     !h.Disabled,
		})
	}

	if pr.Body != nil {
		switch pr.Body.Mode {
		case "raw":
			req.Body = pr.Body.Raw
			lang := strings.ToLower(pr.Body.Options.Raw.Language)
			trimmed := strings.TrimSpace(pr.Body.Raw)
			if lang == "json" || strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
				req.BodyType = "json"
			} else {
				req.BodyType = "text"
			}
		case "urlencoded":
			req.BodyType = "urlencoded"
			for _, kv := range pr.Body.URLEncoded {
				req.URLEncoded = append(req.URLEncoded, models.KeyValuePair{
					Key:     kv.Key,
					Value:   kv.Value,
					Enabled: !kv.Disabled,
				})
			}
		case "formdata":
			req.BodyType = "form-data"
			for _, kv := range pr.Body.FormData {
				req.FormData = append(req.FormData, models.KeyValuePair{
					Key:     kv.Key,
					Value:   kv.Value,
					Enabled: !kv.Disabled,
				})
			}
		}
	}

	if pr.Auth != nil {
		switch strings.ToLower(pr.Auth.Type) {
		case "bearer":
			req.AuthType = "bearer"
			req.Token = postmanAttr(pr.Auth.Bearer, "token")
		case "basic":
			req.AuthType = "basic"
			req.BasicUser = postmanAttr(pr.Auth.Basic, "username")
			req.BasicPassword = postmanAttr(pr.Auth.Basic, "password")
		case "apikey":
			req.AuthType = "apikey"
			req.APIKeyKey = postmanAttr(pr.Auth.APIKey, "key")
			req.APIKeyValue = postmanAttr(pr.Auth.APIKey, "value")
			req.APIKeyIn = postmanAttr(pr.Auth.APIKey, "in")
			if req.APIKeyIn == "" {
				req.APIKeyIn = "header"
			}
		}
	}

	for _, ev := range events {
		if strings.EqualFold(ev.Listen, "test") && len(ev.Script.Exec) > 0 {
			req.TestScript = strings.Join(ev.Script.Exec, "\n")
			break
		}
	}

	return req
}

func decodePostmanURL(raw json.RawMessage) (string, []models.KeyValuePair) {
	if len(raw) == 0 {
		return "", nil
	}
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		return str, models.SyncURLToParams(str)
	}
	var obj postmanURLObj
	if err := json.Unmarshal(raw, &obj); err == nil {
		var params []models.KeyValuePair
		for _, q := range obj.Query {
			params = append(params, models.KeyValuePair{
				Key:         q.Key,
				Value:       q.Value,
				Description: q.Description,
				Enabled:     !q.Disabled,
			})
		}
		return obj.Raw, params
	}
	return "", nil
}

func postmanAttr(attrs []postmanKV, key string) string {
	for _, a := range attrs {
		if strings.EqualFold(a.Key, key) {
			return a.Value
		}
	}
	return ""
}
