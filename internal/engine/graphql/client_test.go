package graphql

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/fblazt/boomerang/internal/models"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

const sampleIntrospectionJSON = `{
  "data": {
    "__schema": {
      "queryType": { "name": "Query" },
      "mutationType": { "name": "Mutation" },
      "subscriptionType": null,
      "types": [
        { "kind": "OBJECT", "name": "__Schema", "description": "Internal" },
        {
          "kind": "OBJECT",
          "name": "Query",
          "fields": [
            {
              "name": "users",
              "description": "Fetch active users",
              "args": [
                { "name": "limit", "type": { "kind": "SCALAR", "name": "Int" } }
              ],
              "type": {
                "kind": "NON_NULL",
                "ofType": {
                  "kind": "LIST",
                  "ofType": {
                    "kind": "NON_NULL",
                    "ofType": { "kind": "OBJECT", "name": "User" }
                  }
                }
              }
            }
          ]
        },
        {
          "kind": "OBJECT",
          "name": "Mutation",
          "fields": [
            {
              "name": "createUser",
              "args": [],
              "type": { "kind": "OBJECT", "name": "User" }
            }
          ]
        },
        {
          "kind": "OBJECT",
          "name": "User",
          "description": "Registered account",
          "fields": [
            {
              "name": "id",
              "type": { "kind": "NON_NULL", "ofType": { "kind": "SCALAR", "name": "ID" } }
            },
            {
              "name": "email",
              "type": { "kind": "SCALAR", "name": "String" }
            }
          ]
        },
        {
          "kind": "SCALAR",
          "name": "String",
          "description": "UTF-8 string"
        }
      ]
    }
  }
}`

func TestClient_IntrospectAndExecuteQuery(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			rawBody, _ := io.ReadAll(r.Body)
			var payload map[string]any
			_ = json.Unmarshal(rawBody, &payload)
			q, _ := payload["query"].(string)

			var respJSON string
			if strings.Contains(q, "__schema") {
				respJSON = sampleIntrospectionJSON
			} else {
				respJSON = `{
				  "data": { "users": [{ "id": "u_1", "email": "ada@boomerang.dev" }] },
				  "extensions": { "resolver_time": 11.4 }
				}`
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(respJSON)),
			}, nil
		}),
	}

	client := NewClient(httpClient)
	schema, err := client.Introspect(context.Background(), "https://api.internal.net/graphql", nil)
	if err != nil {
		t.Fatalf("Introspect failed: %v", err)
	}

	if schema.QueryTypeName != "Query" || schema.MutationTypeName != "Mutation" {
		t.Fatalf("unexpected root types: query=%q mutation=%q", schema.QueryTypeName, schema.MutationTypeName)
	}
	if _, internalFound := schema.FindType("__Schema"); internalFound {
		t.Fatal("expected internal __Schema type excluded from registry")
	}

	queries, mutations, _ := schema.RootOperations()
	if len(queries) != 1 || queries[0].Name != "users" || queries[0].Type.Format() != "[User!]!" {
		t.Fatalf("unexpected root query signature: %+v", queries)
	}
	if len(mutations) != 1 || mutations[0].Name != "createUser" {
		t.Fatalf("unexpected root mutations: %+v", mutations)
	}

	resp, err := client.ExecuteQuery(context.Background(), models.GraphQLRequest{
		Endpoint:      "https://api.internal.net/graphql",
		Query:         `query { users(limit: 5) { id email } }`,
		VariablesJSON: `{"limit": 5}`,
	})
	if err != nil {
		t.Fatalf("ExecuteQuery failed: %v", err)
	}
	if resp.ResolverTimeMs != 11.4 {
		t.Fatalf("expected ResolverTimeMs 11.4, got %v", resp.ResolverTimeMs)
	}
	if !strings.Contains(resp.DataJSON, `"ada@boomerang.dev"`) {
		t.Fatalf("expected formatted data JSON, got:\n%s", resp.DataJSON)
	}
}

func TestClient_RejectsInvalidVariablesJSON(t *testing.T) {
	client := NewClient(nil)
	resp, err := client.ExecuteQuery(context.Background(), models.GraphQLRequest{
		Endpoint:      "https://api.internal.net/graphql",
		Query:         `{ health }`,
		VariablesJSON: `{invalid`,
	})
	if err == nil || resp == nil || resp.RequestError == "" {
		t.Fatalf("expected validation error for malformed variables JSON, got resp=%+v err=%v", resp, err)
	}
}
