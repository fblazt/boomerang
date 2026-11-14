package models

import (
	"fmt"
	"sort"
	"strings"
)

// PaletteItem represents one actionable search result in the Global Command Palette (⌘K).
type PaletteItem struct {
	ID       string
	Category string // "Endpoint", "Scratchpad", "Environment", "Command"
	Badge    string
	Title    string
	Subtitle string
	Shortcut string
	Score    int

	collectionID string
	itemID       string
	draftID      string
	envName      string
	actionID     string
}

// OpenCommandPalette opens the global command palette modal and clears prior query state.
func (s *AppState) OpenCommandPalette() {
	s.Mutate(func(state *AppState) {
		state.CommandPalOpen = true
		state.PaletteQuery = ""
		state.PaletteSelectedIndex = 0
	})
}

// ToggleCommandPalette toggles visibility of the global command palette modal.
func (s *AppState) ToggleCommandPalette() {
	s.mu.RLock()
	open := s.CommandPalOpen
	s.mu.RUnlock()
	if open {
		s.CloseCommandPalette()
		return
	}
	s.OpenCommandPalette()
}

// CloseCommandPalette closes the command palette modal.
func (s *AppState) CloseCommandPalette() {
	s.Mutate(func(state *AppState) {
		state.CommandPalOpen = false
	})
}

// SetPaletteQuery updates the command palette search query and resets row selection to 0.
func (s *AppState) SetPaletteQuery(query string) {
	s.Mutate(func(state *AppState) {
		state.PaletteQuery = query
		state.PaletteSelectedIndex = 0
	})
}

// MovePaletteSelection moves the active highlight index within [0, total-1].
func (s *AppState) MovePaletteSelection(delta, total int) {
	if total <= 0 {
		return
	}
	s.Mutate(func(state *AppState) {
		next := (state.PaletteSelectedIndex + delta) % total
		if next < 0 {
			next += total
		}
		state.PaletteSelectedIndex = next
	})
}

// FilteredPaletteItems builds and fuzzy-ranks all endpoints, drafts, environments, and workspace commands.
func (s *AppState) FilteredPaletteItems() []PaletteItem {
	s.mu.RLock()
	query := strings.TrimSpace(s.PaletteQuery)
	cols := s.Collections
	drafts := s.Drafts
	envs := s.Environments
	activeEnv := s.ActiveEnv

	var candidates []PaletteItem

	// 1. Saved collection endpoints
	for _, col := range cols {
		collectPaletteEndpoints(&candidates, col.ID, col.Name, col.Items, col.Folders)
	}

	// 2. Scratchpad drafts
	for _, d := range drafts {
		title := d.Name
		if title == "" {
			title = d.Request.Name
		}
		candidates = append(candidates, PaletteItem{
			ID:       "draft:" + d.ID,
			Category: "Scratchpad",
			Badge:    d.Request.Method,
			Title:    title,
			Subtitle: "Scratchpad • " + d.Request.URL,
			draftID:  d.ID,
		})
	}

	// 3. Environments
	for idx, env := range envs {
		shortcut := ""
		if idx < 4 {
			shortcut = fmt.Sprintf("⌘%d", idx+1)
		}
		sub := "Switch environment"
		if strings.EqualFold(env.Name, activeEnv) {
			sub = "Active environment"
		}
		candidates = append(candidates, PaletteItem{
			ID:       "env:" + env.Name,
			Category: "Environment",
			Badge:    "ENV",
			Title:    env.Name,
			Subtitle: sub,
			Shortcut: shortcut,
			envName:  env.Name,
		})
	}

	// 4. Workspace & Modal Actions
	commands := []struct {
		id, badge, title, subtitle, shortcut string
	}{
		{"send", "RUN", "Send Active Request", "Execute active HTTP/REST request", "⌘↵"},
		{"new_draft", "NEW", "New Scratchpad Draft", "Open an uncommitted request tab", "⌘N"},
		{"import", "IMP", "Import API / cURL / OpenAPI", "Open single-step universal importer", "⌘I"},
		{"vault", "ENV", "Environment & Secret Vault", "Manage variables, TLS & proxies", "⌘E"},
		{"history", "LOG", "Execution History & HAR Export", "Inspect recent request timeline", "⌘H"},
		{"runner", "SUITE", "Collection Suite Runner", "Run collection tests concurrently", "⌘R"},
		{"graphql", "GQL", "GraphQL Explorer", "Schema introspection, query & variables", ""},
		{"stream", "WS", "Realtime Stream Inspector", "Open WebSocket / SSE inspector", ""},
		{"rest", "HTTP", "REST Request Builder", "Return to HTTP request workspace", ""},
	}
	for _, cmd := range commands {
		candidates = append(candidates, PaletteItem{
			ID:       "cmd:" + cmd.id,
			Category: "Command",
			Badge:    cmd.badge,
			Title:    cmd.title,
			Subtitle: cmd.subtitle,
			Shortcut: cmd.shortcut,
			actionID: cmd.id,
		})
	}
	s.mu.RUnlock()

	if query == "" {
		if len(candidates) > 24 {
			return candidates[:24]
		}
		return candidates
	}

	var matched []PaletteItem
	for _, item := range candidates {
		score := scorePaletteItem(query, item)
		if score > 0 {
			item.Score = score
			matched = append(matched, item)
		}
	}

	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].Score != matched[j].Score {
			return matched[i].Score > matched[j].Score
		}
		return matched[i].Title < matched[j].Title
	})
	if len(matched) > 24 {
		matched = matched[:24]
	}
	return matched
}

// ExecutePaletteItem performs the action associated with item and closes the command palette.
func (s *AppState) ExecutePaletteItem(item PaletteItem) {
	s.CloseCommandPalette()

	switch {
	case item.collectionID != "" && item.itemID != "":
		s.SelectCollectionItem(item.collectionID, item.itemID)
	case item.draftID != "":
		s.SelectDraft(item.draftID)
	case item.envName != "":
		s.SetActiveEnv(item.envName)
	case item.actionID != "":
		switch item.actionID {
		case "send":
			s.SendRequest()
		case "new_draft":
			s.NewScratchpadDraft()
		case "import":
			s.OpenImportModal()
		case "vault":
			s.OpenVaultModal()
		case "history":
			s.OpenHistoryModal()
		case "runner":
			s.OpenRunnerModal()
		case "graphql":
			s.OpenGraphQLExplorer("")
		case "stream":
			s.OpenStreamEndpoint(StreamProtocolWSS, "", "")
		case "rest":
			s.SetWorkspaceMode("rest")
		}
	}
}

func collectPaletteEndpoints(dst *[]PaletteItem, colID, pathPrefix string, items []CollectionItem, folders []Folder) {
	for _, it := range items {
		method := it.Request.Method
		if method == "" {
			method = "GET"
		}
		*dst = append(*dst, PaletteItem{
			ID:           "item:" + it.ID,
			Category:     "Endpoint",
			Badge:        method,
			Title:        it.Name,
			Subtitle:     pathPrefix + " • " + it.Request.URL,
			collectionID: colID,
			itemID:       it.ID,
		})
	}
	for _, f := range folders {
		collectPaletteEndpoints(dst, colID, pathPrefix+" / "+f.Name, f.Items, f.Folders)
	}
}

func scorePaletteItem(query string, item PaletteItem) int {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return 1
	}

	titleScore := fuzzyScore(q, strings.ToLower(item.Title)) * 3
	badgeScore := fuzzyScore(q, strings.ToLower(item.Badge)) * 2
	subScore := fuzzyScore(q, strings.ToLower(item.Subtitle))
	catScore := fuzzyScore(q, strings.ToLower(item.Category))

	best := titleScore
	if badgeScore > best {
		best = badgeScore
	}
	if subScore > best {
		best = subScore
	}
	if catScore > best {
		best = catScore
	}
	return best
}

// fuzzyScore rewards exact prefix, substring inclusion, and ordered subsequence matches.
func fuzzyScore(query, text string) int {
	if query == "" || text == "" {
		return 0
	}
	if text == query {
		return 120
	}
	if strings.HasPrefix(text, query) {
		return 100
	}
	if idx := strings.Index(text, query); idx >= 0 {
		return 80 - idx
	}

	// Ordered subsequence fuzzy match
	qi := 0
	consec := 0
	score := 0
	for ti := 0; ti < len(text) && qi < len(query); ti++ {
		if text[ti] == query[qi] {
			qi++
			consec++
			score += 6 + consec*3
		} else {
			consec = 0
		}
	}
	if qi != len(query) {
		return 0
	}
	if score < 10 {
		score = 10
	}
	return score
}
