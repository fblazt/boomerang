package models

import "time"

// GraphQLTypeKind identifies the GraphQL schema type classification.
type GraphQLTypeKind string

const (
	GQLKindObject      GraphQLTypeKind = "OBJECT"
	GQLKindInterface   GraphQLTypeKind = "INTERFACE"
	GQLKindUnion       GraphQLTypeKind = "UNION"
	GQLKindEnum        GraphQLTypeKind = "ENUM"
	GQLKindInputObject GraphQLTypeKind = "INPUT_OBJECT"
	GQLKindScalar      GraphQLTypeKind = "SCALAR"
	GQLKindList        GraphQLTypeKind = "LIST"
	GQLKindNonNull     GraphQLTypeKind = "NON_NULL"
)

// GraphQLTypeRef represents a nested GraphQL type reference wrapper (NON_NULL / LIST / named).
type GraphQLTypeRef struct {
	Kind   GraphQLTypeKind `json:"kind"`
	Name   string          `json:"name,omitempty"`
	OfType *GraphQLTypeRef `json:"ofType,omitempty"`
}

// Format renders a concise GraphQL SDL type expression (e.g. "[User!]!").
func (r GraphQLTypeRef) Format() string {
	switch r.Kind {
	case GQLKindNonNull:
		if r.OfType != nil {
			return r.OfType.Format() + "!"
		}
		return "!"
	case GQLKindList:
		if r.OfType != nil {
			return "[" + r.OfType.Format() + "]"
		}
		return "[]"
	default:
		if r.Name != "" {
			return r.Name
		}
		return string(r.Kind)
	}
}

// NamedBase extracts the underlying named GraphQL type unwrapping NON_NULL/LIST wrappers.
func (r GraphQLTypeRef) NamedBase() string {
	cur := &r
	for cur != nil {
		if cur.Name != "" {
			return cur.Name
		}
		cur = cur.OfType
	}
	return ""
}

// GraphQLInputValue models an argument or input object field in an introspected schema.
type GraphQLInputValue struct {
	Name         string         `json:"name"`
	Description  string         `json:"description,omitempty"`
	Type         GraphQLTypeRef `json:"type"`
	DefaultValue *string        `json:"defaultValue,omitempty"`
}

// GraphQLField models a field on an OBJECT or INTERFACE type.
type GraphQLField struct {
	Name              string              `json:"name"`
	Description       string              `json:"description,omitempty"`
	Args              []GraphQLInputValue `json:"args,omitempty"`
	Type              GraphQLTypeRef      `json:"type"`
	IsDeprecated      bool                `json:"isDeprecated,omitempty"`
	DeprecationReason string              `json:"deprecationReason,omitempty"`
}

// GraphQLEnumValue models an enum member in an introspected schema.
type GraphQLEnumValue struct {
	Name              string `json:"name"`
	Description       string `json:"description,omitempty"`
	IsDeprecated      bool   `json:"isDeprecated,omitempty"`
	DeprecationReason string `json:"deprecationReason,omitempty"`
}

// GraphQLType represents one complete named type in an introspected GraphQL schema.
type GraphQLType struct {
	Kind        GraphQLTypeKind     `json:"kind"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Fields      []GraphQLField      `json:"fields,omitempty"`
	InputFields []GraphQLInputValue `json:"inputFields,omitempty"`
	EnumValues  []GraphQLEnumValue  `json:"enumValues,omitempty"`
	Interfaces  []GraphQLTypeRef    `json:"interfaces,omitempty"`
}

// GraphQLSchema holds the parsed AST/type registry from a GraphQL __schema introspection query.
type GraphQLSchema struct {
	QueryTypeName        string        `json:"query_type_name"`
	MutationTypeName     string        `json:"mutation_type_name,omitempty"`
	SubscriptionTypeName string        `json:"subscription_type_name,omitempty"`
	Types                []GraphQLType `json:"types"`
	UpdatedAt            time.Time     `json:"updated_at"`
}

// FindType locates a named type in the schema registry.
func (s *GraphQLSchema) FindType(name string) (GraphQLType, bool) {
	if s == nil {
		return GraphQLType{}, false
	}
	for _, t := range s.Types {
		if t.Name == name {
			return t, true
		}
	}
	return GraphQLType{}, false
}

// RootOperations extracts root Query, Mutation, and Subscription fields when defined in the schema.
func (s *GraphQLSchema) RootOperations() (queries []GraphQLField, mutations []GraphQLField, subscriptions []GraphQLField) {
	if s == nil {
		return nil, nil, nil
	}
	if s.QueryTypeName != "" {
		if qt, ok := s.FindType(s.QueryTypeName); ok {
			queries = qt.Fields
		}
	}
	if s.MutationTypeName != "" {
		if mt, ok := s.FindType(s.MutationTypeName); ok {
			mutations = mt.Fields
		}
	}
	if s.SubscriptionTypeName != "" {
		if st, ok := s.FindType(s.SubscriptionTypeName); ok {
			subscriptions = st.Fields
		}
	}
	return queries, mutations, subscriptions
}

// GraphQLRequest specifies the query document, variables JSON, and endpoint to execute.
type GraphQLRequest struct {
	Endpoint      string         `json:"endpoint"`
	OperationName string         `json:"operation_name,omitempty"`
	Query         string         `json:"query"`
	VariablesJSON string         `json:"variables_json"`
	Headers       []KeyValuePair `json:"headers,omitempty"`
}

// GraphQLLocation stores source coordinates of a GraphQL execution error.
type GraphQLLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// GraphQLError captures a GraphQL spec error item returned in response.errors.
type GraphQLError struct {
	Message   string            `json:"message"`
	Locations []GraphQLLocation `json:"locations,omitempty"`
	Path      []any             `json:"path,omitempty"`
}

// GraphQLResponse captures execution telemetry and formatted JSON data returned by a GraphQL server.
type GraphQLResponse struct {
	StatusCode     int            `json:"status_code"`
	StatusText     string         `json:"status_text"`
	DurationMs     int64          `json:"duration_ms"`
	ResolverTimeMs float64        `json:"resolver_time_ms"`
	DataJSON       string         `json:"data_json"`
	RawJSON        string         `json:"raw_json"`
	ExtensionsJSON string         `json:"extensions_json,omitempty"`
	Errors         []GraphQLError `json:"errors,omitempty"`
	RequestError   string         `json:"request_error,omitempty"`
}
