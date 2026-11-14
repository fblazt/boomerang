package models

import (
	"context"
	"strings"
)

// SetGraphQLRunner attaches the GraphQL introspection & execution client.
func (s *AppState) SetGraphQLRunner(runner GraphQLRunner) {
	s.mu.Lock()
	s.graphqlRunner = runner
	s.mu.Unlock()
}

// OpenGraphQLExplorer switches the center workspace to GraphQL Explorer view.
func (s *AppState) OpenGraphQLExplorer(endpoint string) {
	s.Mutate(func(state *AppState) {
		state.WorkspaceMode = "graphql"
		if strings.TrimSpace(endpoint) != "" {
			state.GQLRequest.Endpoint = endpoint
		}
	})
}

// SetGraphQLEndpoint updates the GraphQL endpoint URL.
func (s *AppState) SetGraphQLEndpoint(endpoint string) {
	s.mu.Lock()
	s.GQLRequest.Endpoint = endpoint
	s.mu.Unlock()
}

// SetGraphQLQuery updates the GraphQL operation document and refreshes field autocomplete suggestions.
func (s *AppState) SetGraphQLQuery(query string) {
	s.mu.Lock()
	s.GQLRequest.Query = query
	s.updateGraphQLSuggestionsLocked(query)
	s.mu.Unlock()
	s.RequestRedraw()
}

// ApplyGraphQLSuggestion inserts or completes a schema field suggestion in the active query document.
func (s *AppState) ApplyGraphQLSuggestion(fieldName string) {
	fieldName = strings.TrimSpace(fieldName)
	if fieldName == "" {
		return
	}
	s.Mutate(func(state *AppState) {
		var matchedField *GraphQLField
		if state.GQLSchema != nil {
			for _, t := range state.GQLSchema.Types {
				for _, f := range t.Fields {
					if f.Name == fieldName {
						fc := f
						matchedField = &fc
						break
					}
				}
				if matchedField != nil {
					break
				}
			}
		}

		snippet := fieldName
		if matchedField != nil && len(matchedField.Args) > 0 {
			var argPairs []string
			for _, a := range matchedField.Args {
				argPairs = append(argPairs, a.Name+": null")
			}
			snippet += "(" + strings.Join(argPairs, ", ") + ")"
		}

		cur := state.GQLRequest.Query
		trimmed := strings.TrimSpace(cur)
		if trimmed == "" {
			state.GQLRequest.Query = "query {\n  " + snippet + "\n}"
		} else if idx := strings.LastIndex(cur, "}"); idx > 0 {
			prefixBeforeClose := strings.TrimRight(cur[:idx], " \t")
			lines := strings.Split(prefixBeforeClose, "\n")
			lastLine := strings.TrimSpace(lines[len(lines)-1])
			if lastLine != "" && !strings.Contains(lastLine, "{") && strings.HasPrefix(strings.ToLower(fieldName), strings.ToLower(lastLine)) {
				lastLineStart := strings.LastIndex(prefixBeforeClose, lines[len(lines)-1])
				state.GQLRequest.Query = cur[:lastLineStart] + snippet + "\n" + cur[idx:]
			} else {
				sep := "\n  "
				if strings.HasSuffix(prefixBeforeClose, "\n") {
					sep = "  "
				}
				state.GQLRequest.Query = prefixBeforeClose + sep + snippet + "\n" + cur[idx:]
			}
		} else {
			state.GQLRequest.Query = strings.TrimRight(cur, "\n") + "\n  " + snippet
		}
		state.updateGraphQLSuggestionsLocked(state.GQLRequest.Query)
	})
}

// SetGraphQLVariables updates the GraphQL variables JSON text.
func (s *AppState) SetGraphQLVariables(varsJSON string) {
	s.mu.Lock()
	s.GQLRequest.VariablesJSON = varsJSON
	s.mu.Unlock()
}

// SelectGraphQLSchemaType navigates the schema documentation drawer to typeName.
func (s *AppState) SelectGraphQLSchemaType(typeName string) {
	s.Mutate(func(state *AppState) {
		state.GQLSelectedType = typeName
	})
}

// SetGraphQLSchemaSearch updates the filter query inside the Schema Explorer drawer.
func (s *AppState) SetGraphQLSchemaSearch(query string) {
	s.Mutate(func(state *AppState) {
		state.GQLSchemaSearch = query
	})
}

// ToggleGraphQLSchemaDrawer shows or hides the Schema Documentation drawer.
func (s *AppState) ToggleGraphQLSchemaDrawer() {
	s.Mutate(func(state *AppState) {
		state.GQLSchemaDrawerOpen = !state.GQLSchemaDrawerOpen
	})
}

// SetGraphQLResponseTab selects the GraphQL response payload viewer tab ("data", "extensions", "raw").
func (s *AppState) SetGraphQLResponseTab(tab string) {
	switch tab {
	case "data", "extensions", "raw":
	default:
		tab = "data"
	}
	s.Mutate(func(state *AppState) {
		state.GQLResponseTab = tab
	})
}

// InsertGraphQLFieldSnippet appends a field selection snippet into the active query document.
func (s *AppState) InsertGraphQLFieldSnippet(field GraphQLField) {
	s.Mutate(func(state *AppState) {
		snippet := field.Name
		if len(field.Args) > 0 {
			var argPairs []string
			for _, a := range field.Args {
				argPairs = append(argPairs, a.Name+": null")
			}
			snippet += "(" + strings.Join(argPairs, ", ") + ")"
		}
		cur := strings.TrimSpace(state.GQLRequest.Query)
		if cur == "" {
			state.GQLRequest.Query = "query {\n  " + snippet + "\n}"
		} else if idx := strings.LastIndex(cur, "}"); idx > 0 {
			state.GQLRequest.Query = cur[:idx] + "  " + snippet + "\n}"
		} else {
			state.GQLRequest.Query = cur + "\n  " + snippet
		}
		state.updateGraphQLSuggestionsLocked(state.GQLRequest.Query)
	})
}

// IntrospectGraphQL runs a background __schema query against GQLRequest.Endpoint.
func (s *AppState) IntrospectGraphQL() {
	s.mu.Lock()
	runner := s.graphqlRunner
	if runner == nil || s.GQLIntrospecting {
		s.mu.Unlock()
		return
	}
	endpoint := s.GQLRequest.Endpoint
	vars := s.getActiveVariablesLocked()
	strFn := s.interpolateStrFn
	s.GQLIntrospecting = true
	s.GQLError = ""
	s.mu.Unlock()

	if strFn != nil {
		endpoint = strFn(endpoint, vars)
	}
	s.RequestRedraw()

	go func() {
		schema, err := runner.Introspect(context.Background(), endpoint, nil)
		s.mu.Lock()
		s.GQLIntrospecting = false
		if err != nil {
			s.GQLError = err.Error()
		} else {
			s.GQLSchema = schema
			s.GQLError = ""
			if s.GQLSelectedType == "" && schema != nil {
				if schema.QueryTypeName != "" {
					s.GQLSelectedType = schema.QueryTypeName
				} else if len(schema.Types) > 0 {
					s.GQLSelectedType = schema.Types[0].Name
				}
			}
			s.updateGraphQLSuggestionsLocked(s.GQLRequest.Query)
		}
		s.mu.Unlock()
		s.RequestRedraw()
	}()
}

// ExecuteGraphQL dispatches the current GraphQL query + variables asynchronously.
func (s *AppState) ExecuteGraphQL() {
	s.mu.Lock()
	runner := s.graphqlRunner
	if runner == nil || s.GQLLoading {
		s.mu.Unlock()
		return
	}
	req := s.GQLRequest
	vars := s.getActiveVariablesLocked()
	strFn := s.interpolateStrFn
	s.GQLLoading = true
	s.GQLError = ""
	s.mu.Unlock()

	if strFn != nil {
		req.Endpoint = strFn(req.Endpoint, vars)
		req.VariablesJSON = strFn(req.VariablesJSON, vars)
	}
	s.RequestRedraw()

	go func() {
		resp, err := runner.ExecuteQuery(context.Background(), req)
		s.mu.Lock()
		s.GQLLoading = false
		if err != nil && resp == nil {
			s.GQLError = err.Error()
		} else {
			s.GQLResponse = resp
			if err != nil && resp != nil && resp.RequestError == "" {
				s.GQLError = err.Error()
			}
		}
		s.mu.Unlock()
		s.RequestRedraw()
	}()
}

func (s *AppState) updateGraphQLSuggestionsLocked(query string) {
	if s.GQLSchema == nil {
		s.GQLAutocompleteItems = nil
		return
	}
	prefix := ""
	lines := strings.Split(query, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" || line == "}" {
			continue
		}
		if strings.HasSuffix(line, "{") {
			// Cursor is on an opening brace header; suggest all root/selected fields.
			prefix = ""
			break
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			cand := strings.TrimRight(fields[len(fields)-1], "{}")
			if cand != "" && !strings.Contains(cand, "(") {
				prefix = strings.ToLower(cand)
			}
		}
		break
	}

	seen := make(map[string]bool)
	var suggestions []string
	addCandidate := func(name string) {
		if name == "" || seen[name] {
			return
		}
		if prefix == "" || strings.HasPrefix(strings.ToLower(name), prefix) {
			seen[name] = true
			suggestions = append(suggestions, name)
		}
	}

	for _, t := range s.GQLSchema.Types {
		if t.Name == s.GQLSchema.QueryTypeName || t.Name == s.GQLSchema.MutationTypeName || t.Name == s.GQLSelectedType {
			for _, f := range t.Fields {
				addCandidate(f.Name)
			}
		}
	}
	if len(suggestions) == 0 && prefix != "" {
		// Fall back to full field list when prefix is an already-complete field name.
		prefix = ""
		for _, t := range s.GQLSchema.Types {
			if t.Name == s.GQLSchema.QueryTypeName || t.Name == s.GQLSchema.MutationTypeName || t.Name == s.GQLSelectedType {
				for _, f := range t.Fields {
					addCandidate(f.Name)
				}
			}
		}
	}
	if len(suggestions) > 8 {
		suggestions = suggestions[:8]
	}
	s.GQLAutocompleteItems = suggestions
}
