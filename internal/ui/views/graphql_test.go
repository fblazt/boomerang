package views

import (
	"context"
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
