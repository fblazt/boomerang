package models

import (
	"slices"
	"time"
)

// CollectionItem represents a saved API request within a collection or nested folder.
type CollectionItem struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Order     int          `json:"order"`
	Request   RequestModel `json:"request"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// Folder groups requests and child folders inside a collection.
type Folder struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Order     int              `json:"order"`
	Folders   []Folder         `json:"folders,omitempty"`
	Items     []CollectionItem `json:"items,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// Collection represents a workspace collection persisted to local storage as JSON.
type Collection struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Order       int              `json:"order"`
	Folders     []Folder         `json:"folders,omitempty"`
	Items       []CollectionItem `json:"items,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// CloneRequestModel returns a deep copy of req so edits never mutate shared slices or pointers.
func CloneRequestModel(req RequestModel) RequestModel {
	out := req
	out.Params = slices.Clone(req.Params)
	out.Headers = slices.Clone(req.Headers)
	out.FormData = slices.Clone(req.FormData)
	out.URLEncoded = slices.Clone(req.URLEncoded)
	if req.StrictSSL != nil {
		v := *req.StrictSSL
		out.StrictSSL = &v
	}
	if req.FollowRedirects != nil {
		v := *req.FollowRedirects
		out.FollowRedirects = &v
	}
	if req.EnableHTTP2 != nil {
		v := *req.EnableHTTP2
		out.EnableHTTP2 = &v
	}
	if req.TransportPolicy != nil {
		p := *req.TransportPolicy
		out.TransportPolicy = &p
	}
	return out
}

// CloneCollectionItem returns a deep copy of a CollectionItem.
func CloneCollectionItem(item CollectionItem) CollectionItem {
	out := item
	out.Request = CloneRequestModel(item.Request)
	return out
}

// CloneFolder returns a deep copy of a Folder and its nested subtree.
func CloneFolder(f Folder) Folder {
	out := f
	if len(f.Folders) > 0 {
		out.Folders = make([]Folder, len(f.Folders))
		for i, sub := range f.Folders {
			out.Folders[i] = CloneFolder(sub)
		}
	}
	if len(f.Items) > 0 {
		out.Items = make([]CollectionItem, len(f.Items))
		for i, item := range f.Items {
			out.Items[i] = CloneCollectionItem(item)
		}
	}
	return out
}

// CloneCollection returns a deep copy of a Collection and its nested subtree.
func CloneCollection(c Collection) Collection {
	out := c
	if len(c.Folders) > 0 {
		out.Folders = make([]Folder, len(c.Folders))
		for i, sub := range c.Folders {
			out.Folders[i] = CloneFolder(sub)
		}
	}
	if len(c.Items) > 0 {
		out.Items = make([]CollectionItem, len(c.Items))
		for i, item := range c.Items {
			out.Items[i] = CloneCollectionItem(item)
		}
	}
	return out
}

// DefaultCollections returns initial starter collections when no workspace collections exist on disk.
func DefaultCollections() []Collection {
	now := time.Now().UTC()
	return []Collection{
		{
			ID:          "col_core_api",
			Name:        "Core API",
			Description: "Starter workspace API endpoints",
			Order:       0,
			CreatedAt:   now,
			UpdatedAt:   now,
			Items: []CollectionItem{
				{
					ID:        "req_users_service",
					Name:      "Users Service",
					Order:     0,
					CreatedAt: now,
					UpdatedAt: now,
					Request: RequestModel{
						ID:       "req_users_service",
						Name:     "Users Service",
						Method:   "GET",
						URL:      "{{base_url}}/v2/users?limit=25&active=true",
						Params:   []KeyValuePair{{Key: "limit", Value: "25", Enabled: true}, {Key: "active", Value: "true", Enabled: true}},
						Headers:  []KeyValuePair{{Key: "Accept", Value: "application/json", Enabled: true}},
						BodyType: "none",
						AuthType: "bearer",
						Token:    "{{api_token}}",
					},
				},
				{
					ID:        "req_auth_verify",
					Name:      "Auth Verify",
					Order:     1,
					CreatedAt: now,
					UpdatedAt: now,
					Request: RequestModel{
						ID:       "req_auth_verify",
						Name:     "Auth Verify",
						Method:   "POST",
						URL:      "{{base_url}}/v2/auth/verify",
						Headers:  []KeyValuePair{{Key: "Content-Type", Value: "application/json", Enabled: true}},
						BodyType: "json",
						Body:     `{"scope":"workspace:read"}`,
						AuthType: "none",
					},
				},
				{
					ID:        "req_delete_session",
					Name:      "Delete Session",
					Order:     2,
					CreatedAt: now,
					UpdatedAt: now,
					Request: RequestModel{
						ID:       "req_delete_session",
						Name:     "Delete Session",
						Method:   "DELETE",
						URL:      "{{base_url}}/v2/sessions/current",
						BodyType: "none",
						AuthType: "bearer",
						Token:    "{{api_token}}",
					},
				},
			},
			Folders: []Folder{
				{
					ID:        "fld_catalog",
					Name:      "Catalog",
					Order:     0,
					CreatedAt: now,
					UpdatedAt: now,
					Items: []CollectionItem{
						{
							ID:        "req_product_feed",
							Name:      "Product Feed",
							Order:     0,
							CreatedAt: now,
							UpdatedAt: now,
							Request: RequestModel{
								ID:       "req_product_feed",
								Name:     "Product Feed",
								Method:   "PUT",
								URL:      "{{base_url}}/v2/catalog/feed",
								Headers:  []KeyValuePair{{Key: "Content-Type", Value: "application/json", Enabled: true}},
								BodyType: "json",
								Body:     `{"batchId":"2026-q4","publish":true}`,
								AuthType: "none",
							},
						},
					},
				},
			},
		},
	}
}
