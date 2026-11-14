package views

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

type fakeGraphQLRunner struct{}

func (fakeGraphQLRunner) Introspect(_ context.Context, _ string, _ map[string]string) (*models.GraphQLSchema, error) {
	return &models.GraphQLSchema{
		QueryTypeName:    "Query",
		MutationTypeName: "Mutation",
		Types: []models.GraphQLType{
			{
				Kind:        models.GQLKindObject,
				Name:        "Query",
				Description: "Root read queries",
				Fields: []models.GraphQLField{
					{
						Name:        "users",
						Description: "Fetch workspace accounts",
						Args: []models.GraphQLInputValue{
							{Name: "limit", Type: models.GraphQLTypeRef{Kind: models.GQLKindScalar, Name: "Int"}},
						},
						Type: models.GraphQLTypeRef{
							Kind: models.GQLKindNonNull,
							OfType: &models.GraphQLTypeRef{
								Kind:   models.GQLKindList,
								OfType: &models.GraphQLTypeRef{Kind: models.GQLKindObject, Name: "User"},
							},
						},
					},
				},
			},
			{
				Kind:        models.GQLKindObject,
				Name:        "User",
				Description: "Registered workspace member",
				Fields: []models.GraphQLField{
					{Name: "id", Type: models.GraphQLTypeRef{Kind: models.GQLKindScalar, Name: "ID"}},
					{Name: "email", Type: models.GraphQLTypeRef{Kind: models.GQLKindScalar, Name: "String"}},
				},
			},
			{
				Kind:        models.GQLKindScalar,
				Name:        "String",
				Description: "UTF-8 character sequence",
			},
		},
	}, nil
}

func (fakeGraphQLRunner) ExecuteQuery(_ context.Context, _ models.GraphQLRequest) (*models.GraphQLResponse, error) {
	return &models.GraphQLResponse{
		StatusCode:     200,
		StatusText:     "200 OK",
		DurationMs:     18,
		ResolverTimeMs: 11.4,
		DataJSON:       "{\n  \"users\": [\n    {\n      \"id\": \"u_1\",\n      \"email\": \"ada@boomerang.dev\"\n    }\n  ]\n}",
		RawJSON:        "{\n  \"data\": {\n    \"users\": [\n      {\n        \"id\": \"u_1\"\n      }\n    ]\n  }\n}",
	}, nil
}

func TestGraphQLView_SchemaDrawerNavigation(t *testing.T) {
	state := models.NewAppState(nil)
	state.SetGraphQLRunner(fakeGraphQLRunner{})

	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(theme.NewBoomerangTheme())
		ShellView(c, state)
	}, 1280, 820)
	tt.Frame()

	// Launch GraphQL Explorer from sidebar
	if err := tt.Click("GraphQL Explorer"); err != nil {
		t.Fatalf("click GraphQL Explorer: %v", err)
	}
	tt.Frame()

	// Trigger Introspect
	if err := tt.Click("Introspect"); err != nil {
		t.Fatalf("click Introspect: %v", err)
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		var hasSchema bool
		state.View(func(s *models.AppState) {
			hasSchema = s.GQLSchema != nil
		})
		if hasSchema {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	tt.Frame()

	for _, want := range []string{"SCHEMA EXPLORER", "3 types", "Root read queries", "args(limit: Int)", "[User]!"} {
		if !tt.HasText(want) {
			t.Fatalf("expected %q in schema drawer", want)
		}
	}

	// Click return type link [User]! to navigate to User type docs
	if err := tt.Click("[User]!"); err != nil {
		t.Fatalf("click return type [User]!: %v", err)
	}
	tt.Frame()

	if !tt.HasText("Registered workspace member") || !tt.HasText("email") {
		t.Fatal("expected User type documentation and fields displayed")
	}

	// Navigate to String scalar doc
	if err := tt.Click("String"); err != nil {
		t.Fatalf("click String scalar type: %v", err)
	}
	tt.Frame()
	if !tt.HasText("UTF-8 character sequence") {
		t.Fatal("expected scalar String documentation displayed")
	}
}

func TestGraphQLView_SplitQueryVariablesAndAutocomplete(t *testing.T) {
	state := models.NewAppState(nil)
	state.SetGraphQLRunner(fakeGraphQLRunner{})
	state.OpenGraphQLExplorer("https://api.example.com/graphql")

	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(theme.NewBoomerangTheme())
		ShellView(c, state)
	}, 1280, 820)
	tt.Frame()

	// Introspect schema so autocomplete suggestions populate
	if err := tt.Click("Introspect"); err != nil {
		t.Fatalf("click Introspect: %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		var hasSchema bool
		state.View(func(s *models.AppState) {
			hasSchema = s.GQLSchema != nil
		})
		if hasSchema {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	tt.Frame()

	for _, want := range []string{"QUERY DOCUMENT", "QUERY VARIABLES (JSON)", "SUGGESTIONS"} {
		if !tt.HasText(want) {
			t.Fatalf("expected %q in split query/variables workspace", want)
		}
	}

	// Simulate user typing a partial field prefix in Query Document
	state.SetGraphQLQuery("query {\n  us\n}")
	tt.Frame()

	qNode, ok := tt.Access("GraphQL Query Editor")
	if !ok || !strings.Contains(qNode.Value, "us") {
		t.Fatalf("expected GraphQL Query Editor synced, got ok=%v node=%+v", ok, qNode)
	}

	// Autocomplete suggestion chip "users" should appear and insert field snippet cleanly
	if err := tt.Click("users"); err != nil {
		t.Fatalf("click autocomplete suggestion users: %v", err)
	}
	tt.Frame()

	var queryAfter string
	state.View(func(s *models.AppState) {
		queryAfter = s.GQLRequest.Query
	})
	if !strings.Contains(queryAfter, "users(limit: null)") {
		t.Fatalf("expected autocomplete to complete users(limit: null), got %q", queryAfter)
	}

	// Variables editor reflects state updates and Reset {} clears it cleanly
	state.Mutate(func(s *models.AppState) {
		s.GQLRequest.VariablesJSON = "{\n  \"limit\": 25\n}"
	})
	tt.Frame()
	vNode, ok := tt.Access("GraphQL Variables Editor")
	if !ok || !strings.Contains(vNode.Value, "\"limit\": 25") {
		t.Fatalf("expected GraphQL Variables Editor synced, got ok=%v node=%+v", ok, vNode)
	}

	if err := tt.Click("Reset {}"); err != nil {
		t.Fatalf("click Reset {}: %v", err)
	}
	tt.Frame()
	var varsAfter string
	state.View(func(s *models.AppState) {
		varsAfter = s.GQLRequest.VariablesJSON
	})
	if varsAfter != "{}" {
		t.Fatalf("expected Reset {} to set VariablesJSON to \"{}\", got %q", varsAfter)
	}
}

func TestGraphQLView_ResolverTelemetryAndResponseTabs(t *testing.T) {
	state := models.NewAppState(nil)
	state.SetGraphQLRunner(fakeGraphQLRunner{})
	state.OpenGraphQLExplorer("https://api.example.com/graphql")

	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(theme.NewBoomerangTheme())
		ShellView(c, state)
	}, 1280, 820)
	tt.Frame()

	// Execute GraphQL operation via Execute Query button
	if err := tt.Click("Execute Query"); err != nil {
		t.Fatalf("click Execute Query: %v", err)
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		var hasResp bool
		state.View(func(s *models.AppState) {
			hasResp = s.GQLResponse != nil
		})
		if hasResp {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	tt.Frame()

	for _, want := range []string{"200 OK", "resolver_time: 11.4 ms", "18 ms"} {
		if !tt.HasText(want) {
			t.Fatalf("expected telemetry chip %q in GraphQL response pane", want)
		}
	}

	// Verify formatted JSON response inside GraphQL Response Viewer
	respNode, ok := tt.Access("GraphQL Response Viewer")
	if !ok || !strings.Contains(respNode.Value, "ada@boomerang.dev") {
		t.Fatalf("expected GraphQL Response Viewer to contain formatted DataJSON, got ok=%v node=%+v", ok, respNode)
	}

	// Switch to Raw tab
	if err := tt.Click("Raw"); err != nil {
		t.Fatalf("click Raw tab: %v", err)
	}
	tt.Frame()
	rawNode, ok := tt.Access("GraphQL Response Viewer")
	if !ok || !strings.Contains(rawNode.Value, "\"data\"") {
		t.Fatalf("expected RawJSON displayed on Raw tab, got ok=%v node=%+v", ok, rawNode)
	}

	// Inject GraphQL execution errors and verify error diagnostic panel renders
	state.Mutate(func(s *models.AppState) {
		s.GQLResponse.Errors = []models.GraphQLError{
			{
				Message:   "Field 'secret' requires scope",
				Locations: []models.GraphQLLocation{{Line: 2, Column: 4}},
			},
		}
	})
	tt.Frame()

	if !tt.HasText("GRAPHQL ERRORS (1)") || !tt.HasText("Field 'secret' requires scope (line 2:4)") {
		t.Fatal("expected GraphQL error diagnostics panel with line:column location")
	}
}
