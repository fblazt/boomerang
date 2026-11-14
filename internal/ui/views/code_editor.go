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
	mu       sync.Mutex
	body     *syncedCodeEditor
	response *syncedCodeEditor
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

	ui.Column(c).
		Height(180).
		Background(theme.CanvasDeep).
		Radius(6).
		Padding(4).
		Border(1, theme.BorderSubtle).
		Children(func() {
			editor.View(c, edState).Fill()
		})
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

	ui.Column(c).
		Grow(1).
		MinHeight(120).
		Background(theme.CanvasDeep).
		Radius(6).
		Padding(4).
		Border(1, theme.BorderSubtle).
		Children(func() {
			editor.View(c, edState).Fill()
		})
}
