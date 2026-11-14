package graphql

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/fblazt/boomerang/internal/models"
)

// IntrospectionQuery is the standard GraphQL schema introspection query sent to endpoints.
const IntrospectionQuery = `query IntrospectionQuery {
  __schema {
    queryType { name }
    mutationType { name }
    subscriptionType { name }
    types {
      kind
      name
      description
      fields(includeDeprecated: true) {
        name
        description
        args {
          name
          description
          type { ...TypeRef }
          defaultValue
        }
        type { ...TypeRef }
        isDeprecated
        deprecationReason
      }
      inputFields {
        name
        description
        type { ...TypeRef }
        defaultValue
      }
      interfaces { ...TypeRef }
      enumValues(includeDeprecated: true) {
        name
        description
        isDeprecated
        deprecationReason
      }
    }
  }
}

fragment TypeRef on __Type {
  kind
  name
  ofType {
    kind
    name
    ofType {
      kind
      name
      ofType {
        kind
        name
      }
    }
  }
}`

type rawIntrospectionEnvelope struct {
	Data   *rawIntrospectionData `json:"data"`
	Schema *rawIntrospectionBody  `json:"__schema"`
	Errors []models.GraphQLError `json:"errors"`
}

type rawIntrospectionData struct {
	Schema *rawIntrospectionBody `json:"__schema"`
}

type rawIntrospectionBody struct {
	QueryType        *namedRef            `json:"queryType"`
	MutationType     *namedRef            `json:"mutationType"`
	SubscriptionType *namedRef            `json:"subscriptionType"`
	Types            []models.GraphQLType `json:"types"`
}

type namedRef struct {
	Name string `json:"name"`
}

// ParseIntrospection parses a GraphQL __schema JSON response into a structured GraphQLSchema registry.
func ParseIntrospection(rawJSON []byte) (*models.GraphQLSchema, error) {
	var env rawIntrospectionEnvelope
	if err := json.Unmarshal(rawJSON, &env); err != nil {
		return nil, fmt.Errorf("decode introspection response: %w", err)
	}
	if len(env.Errors) > 0 && (env.Data == nil || env.Data.Schema == nil) && env.Schema == nil {
		return nil, fmt.Errorf("graphql introspection error: %s", env.Errors[0].Message)
	}

	body := env.Schema
	if body == nil && env.Data != nil {
		body = env.Data.Schema
	}
	if body == nil {
		return nil, fmt.Errorf("introspection payload missing data.__schema")
	}

	schema := &models.GraphQLSchema{
		UpdatedAt: time.Now().UTC(),
	}
	if body.QueryType != nil {
		schema.QueryTypeName = body.QueryType.Name
	}
	if body.MutationType != nil {
		schema.MutationTypeName = body.MutationType.Name
	}
	if body.SubscriptionType != nil {
		schema.SubscriptionTypeName = body.SubscriptionType.Name
	}

	schema.Types = make([]models.GraphQLType, 0, len(body.Types))
	for _, t := range body.Types {
		if strings.HasPrefix(t.Name, "__") {
			continue
		}
		schema.Types = append(schema.Types, t)
	}
	slices.SortStableFunc(schema.Types, func(a, b models.GraphQLType) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return schema, nil
}
