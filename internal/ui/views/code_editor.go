package views

import (
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/fblazt/boomerang/internal/models"
	"github.com/fblazt/boomerang/internal/ui/theme"
)

type syncedCodeEditor struct {
	state    *editor.State
	lastText string
	lastVer  uint64
	lastLang string
}

type stateEditorPair struct {
	mu          sync.Mutex
	body        *syncedCodeEditor
	response    *syncedCodeEditor
	gqlQuery    *syncedCodeEditor
	gqlVars     *syncedCodeEditor
	gqlResponse *syncedCodeEditor
}

var (
	editorRegMu sync.Mutex
	editorReg   = make(map[*models.AppState]*stateEditorPair)
)

func getStateEditors(state *models.AppState) *stateEditorPair {
	editorRegMu.Lock()
	defer editorRegMu.Unlock()

	pair, ok := editorReg[state]
	if !ok {
		pair = &stateEditorPair{}
		editorReg[state] = pair
	}
	return pair
}

func newSyncedCodeEditor(readOnly bool, label, placeholder string) *syncedCodeEditor {
	buf := editor.NewBuffer("")
	return &syncedCodeEditor{
		state: &editor.State{
			Buffer:          buf,
			Font:            ui.Font{Family: theme.FontMono, Size: theme.SizeBase},
			LineHeight:      1.55,
			TabSize:         2,
			SoftWrap:        true,
			ReadOnly:        readOnly,
			NoFolding:       true,
			NoScrollPastEnd: true,
			NoActiveLine:    readOnly,
			KeepScrollbars:  true,
			Placeholder:     placeholder,
			Label:           label,
		},
		lastVer: buf.Version(),
	}
}

func (e *syncedCodeEditor) applyThemeAndLang(c *ui.Context, langFilename string) {
	e.state.Theme = boomerangEditorTheme(c)
	if langFilename != e.lastLang {
		if langFilename == "" {
			e.state.Buffer.SetLanguage(nil)
		} else {
			e.state.Buffer.SetLanguage(editor.LanguageFor(langFilename))
		}
		e.lastLang = langFilename
		e.lastVer = e.state.Buffer.Version()
	}
}

func (e *syncedCodeEditor) syncEditable(externalText string, onChanged func(string)) {
	curVer := e.state.Buffer.Version()
	if curVer != e.lastVer {
		curText := e.state.Buffer.String()
		e.lastVer = curVer
		if curText != e.lastText {
			e.lastText = curText
			if onChanged != nil {
				onChanged(curText)
			}
			return
		}
	}

	if externalText != e.lastText {
		e.state.Buffer.Set(externalText)
		e.lastText = externalText
		e.lastVer = e.state.Buffer.Version()
	}
}

func (e *syncedCodeEditor) syncReadOnly(text string) {
	if text != e.lastText {
		e.state.Buffer.Set(text)
		e.lastText = text
		e.lastVer = e.state.Buffer.Version()
	}
}

func boomerangEditorTheme(c *ui.Context) *editor.Theme {
	ct := editor.DefaultTheme(c.Theme())
	ct.Background = theme.CanvasDeep
	ct.Text = theme.TextPrimary
	ct.LineNumber = theme.TextDim
	ct.ActiveLineNumber = theme.TextSecondary
	ct.ActiveLine = ui.RGBA(255, 255, 255, 0.04)
	ct.Cursor = theme.RimFocus
	ct.MatchingBracket = theme.RimFocusGlow
	ct.Tokens[editor.TokenProperty] = editor.TokenStyle{Color: theme.RimFocus}
	ct.Tokens[editor.TokenString] = editor.TokenStyle{Color: ui.RGB(0x34, 0xD3, 0x99)}
	ct.Tokens[editor.TokenNumber] = editor.TokenStyle{Color: ui.RGB(0xFB, 0xBF, 0x24)}
	ct.Tokens[editor.TokenConstant] = editor.TokenStyle{Color: ui.RGB(0xFB, 0xBF, 0x24)}
	ct.Tokens[editor.TokenKeyword] = editor.TokenStyle{Color: ui.RGB(0xC0, 0x84, 0xFC)}
	ct.Tokens[editor.TokenOperator] = editor.TokenStyle{Color: theme.TextDim}
	ct.Tokens[editor.TokenInvalid] = editor.TokenStyle{Color: theme.RimDanger}
	return ct
}

func detectBodyFilename(bodyType, body string) string {
	switch bodyType {
	case "json":
		return "payload.json"
	case "text":
		return ""
	}
	trimmed := strings.TrimSpace(body)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return "response.json"
	}
	if strings.HasPrefix(trimmed, "<") {
		return "response.html"
	}
	return ""
}

func renderSyncedCodeBox(c *ui.Context, edState *editor.State, height float32, minHeight float32) {
	col := ui.Column(c).
		Background(theme.CanvasDeep).
		Radius(6).
		Padding(4).
		Border(1, theme.BorderSubtle)
	if height > 0 {
		col.Height(height)
	} else {
		col.Grow(1)
		if minHeight > 0 {
			col.MinHeight(minHeight)
		}
	}
	col.Children(func() {
		editor.View(c, edState).Fill()
	})
}

func renderRequestBodyEditor(c *ui.Context, state *models.AppState, bodyType, bodyText string) {
	pair := getStateEditors(state)
	pair.mu.Lock()
	if pair.body == nil {
		pair.body = newSyncedCodeEditor(false, "Request Body Editor", "{\n  \"key\": \"value\"\n}")
	}
	ed := pair.body
	ed.applyThemeAndLang(c, detectBodyFilename(bodyType, bodyText))
	ed.syncEditable(bodyText, func(updated string) {
		state.Mutate(func(s *models.AppState) {
			s.ActiveRequest.Body = updated
		})
	})
	edState := ed.state
	pair.mu.Unlock()

	renderSyncedCodeBox(c, edState, 180, 0)
}

func renderResponseBodyEditor(c *ui.Context, state *models.AppState, body string) {
	if body == "" {
		body = "{}"
	}

	pair := getStateEditors(state)
	pair.mu.Lock()
	if pair.response == nil {
		pair.response = newSyncedCodeEditor(true, "Response Body Viewer", "")
	}
	ed := pair.response
	ed.applyThemeAndLang(c, detectBodyFilename("", body))
	ed.syncReadOnly(body)
	edState := ed.state
	pair.mu.Unlock()

	renderSyncedCodeBox(c, edState, 0, 120)
}

func renderGraphQLQueryEditor(c *ui.Context, state *models.AppState, queryText string) {
	pair := getStateEditors(state)
	pair.mu.Lock()
	if pair.gqlQuery == nil {
		pair.gqlQuery = newSyncedCodeEditor(false, "GraphQL Query Editor", "query {\n  \n}")
	}
	ed := pair.gqlQuery
	ed.applyThemeAndLang(c, "query.graphql")
	ed.syncEditable(queryText, func(updated string) {
		state.SetGraphQLQuery(updated)
	})
	edState := ed.state
	pair.mu.Unlock()

	renderSyncedCodeBox(c, edState, 0, 140)
}

func renderGraphQLVariablesEditor(c *ui.Context, state *models.AppState, varsText string) {
	pair := getStateEditors(state)
	pair.mu.Lock()
	if pair.gqlVars == nil {
		pair.gqlVars = newSyncedCodeEditor(false, "GraphQL Variables Editor", "{\n  \"id\": \"1\"\n}")
	}
	ed := pair.gqlVars
	ed.applyThemeAndLang(c, "variables.json")
	ed.syncEditable(varsText, func(updated string) {
		state.SetGraphQLVariables(updated)
	})
	edState := ed.state
	pair.mu.Unlock()

	renderSyncedCodeBox(c, edState, 130, 0)
}

func renderGraphQLResponseEditor(c *ui.Context, state *models.AppState, body string) {
	if strings.TrimSpace(body) == "" {
		body = "{}"
	}

	pair := getStateEditors(state)
	pair.mu.Lock()
	if pair.gqlResponse == nil {
		pair.gqlResponse = newSyncedCodeEditor(true, "GraphQL Response Viewer", "")
	}
	ed := pair.gqlResponse
	ed.applyThemeAndLang(c, "response.json")
	ed.syncReadOnly(body)
	edState := ed.state
	pair.mu.Unlock()

	renderSyncedCodeBox(c, edState, 0, 140)
}
