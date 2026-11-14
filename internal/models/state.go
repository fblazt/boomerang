package models

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
)

// RequestExecutor defines the interface for executing HTTP requests.
type RequestExecutor interface {
	Execute(ctx context.Context, req RequestModel) (*ResponseModel, error)
}

// EnvironmentManager defines the interface for workspace environment management and secret resolution.
type EnvironmentManager interface {
	GetEnvironments() []Environment
	GetEnvironment(nameOrID string) (*Environment, error)
	CreateEnvironment(env Environment) error
	UpdateEnvironment(env Environment) error
	DeleteEnvironment(nameOrID string) error
	DuplicateEnvironment(nameOrID string, newName string) (*Environment, error)
	SetActiveEnvironment(name string) error
	GetActiveEnvironment() *Environment
	AddVariable(envName string, v Variable, secretPlaintext string) error
	UpdateVariable(envName string, oldKey string, v Variable, secretPlaintext string) error
	DeleteVariable(envName string, key string) error
	GetResolvedVariables(envName string) (map[string]string, error)
	GetSecret(envName, key string) (string, error)
	ExportEnvironment(envName string, maskSecrets bool) (string, error)
	ImportEnvironment(name string, dotEnvContent string) (*Environment, error)
}

// FilePickerFunc opens a file selector and invokes onPicked with the selected path if confirmed.
type FilePickerFunc func(title string, extensions []string, onPicked func(path string))

// SaveFilePickerFunc opens a save-file dialog and invokes onPicked with the selected path if confirmed.
type SaveFilePickerFunc func(title, defaultName string, extensions []string, onPicked func(path string))

// StreamController defines lifecycle and frame messaging for WebSocket and SSE streams.
type StreamController interface {
	Connect(ctx context.Context, proto StreamProtocol, targetURL string, headers map[string]string, onFrame func(StreamFrame), onStatus func(StreamStatus, int64, error)) error
	Disconnect() error
	SendText(payload string) error
	SendBinary(data []byte) error
	SendPing() error
}

// GraphQLRunner defines schema introspection and GraphQL document execution.
type GraphQLRunner interface {
	Introspect(ctx context.Context, endpoint string, headers map[string]string) (*GraphQLSchema, error)
	ExecuteQuery(ctx context.Context, req GraphQLRequest) (*GraphQLResponse, error)
}

// HistoryStore defines persistence operations for executed request history logs.
type HistoryStore interface {
	RecordHistory(entry HistoryEntry) (HistoryEntry, error)
	ListHistory(filter HistoryFilter) ([]HistoryEntry, error)
	CountHistory(filter HistoryFilter) (int, error)
	ClearHistory() error
}

// DraftStore defines persistence operations for uncommitted scratchpad drafts.
type DraftStore interface {
	SaveDraft(draft ScratchpadDraft) (ScratchpadDraft, error)
	ListDrafts() ([]ScratchpadDraft, error)
	DeleteDraft(id string) error
}

// CollectionStore defines persistence operations for collections, folders, and saved requests.
type CollectionStore interface {
	Collections() []Collection
	CreateCollection(name string) (Collection, error)
	ImportCollection(col Collection) (Collection, error)
	RenameCollection(collectionID, name string) error
	DeleteCollection(collectionID string) error
	CreateFolder(collectionID, parentFolderID, name string) (Folder, error)
	RenameFolder(collectionID, folderID, newName string) error
	DeleteFolder(collectionID, folderID string) error
	SaveItem(collectionID, folderID, name string, req RequestModel) (CollectionItem, error)
	UpdateItemRequest(collectionID, itemID string, req RequestModel) (CollectionItem, error)
	RenameItem(collectionID, itemID, newName string) error
	DuplicateItem(collectionID, itemID string) (CollectionItem, error)
	ReorderItem(collectionID, itemID string, delta int) error
	MoveItem(collectionID, itemID, targetFolderID string, targetIdx int) error
	DeleteItem(collectionID, itemID string) error
}

// AppState holds the thread-safe global reactive state of the Boomerang application.
type AppState struct {
	mu sync.RWMutex

	// UI Redraw trigger callback (e.g. services.Invalidate or test hook)
	redrawFn       func()
	filePicker     FilePickerFunc
	saveFilePicker SaveFilePickerFunc

	// HTTP Client Executor
	executor RequestExecutor
	cancelFn context.CancelFunc

	// History, Drafts, Collections, Realtime & GraphQL persistence/engines
	historyStore    HistoryStore
	draftStore      DraftStore
	collectionStore CollectionStore
	streamCtrl      StreamController
	graphqlRunner   GraphQLRunner

	// GraphQL Explorer State
	GQLRequest           GraphQLRequest
	GQLSchema            *GraphQLSchema
	GQLSelectedType      string
	GQLSchemaSearch      string
	GQLSchemaDrawerOpen  bool
	GQLIntrospecting     bool
	GQLLoading           bool
	GQLResponse          *GraphQLResponse
	GQLResponseTab       string // "data", "extensions", "raw"
	GQLError             string
	GQLAutocompleteItems []string

	// Realtime Stream Inspector State
	WorkspaceMode         string // "rest", "stream", "graphql"
	Stream                StreamSession
	StreamFrames          []StreamFrame
	StreamFilterDirection string // "", "SENT", "RECV", "PING"

	// Scratchpad Drafts State
	Drafts        []ScratchpadDraft
	ActiveDraftID string

	// Collections Tree State
	Collections        []Collection
	ActiveCollectionID string
	ActiveItemID       string
	SelectedFolderID   string
	CollapsedNodes     map[string]bool
	CollectionFilter   string
	RenamingNodeType   string // "collection", "folder", "item"
	RenamingCollection string
	RenamingNodeID     string
	RenameDraft        string

	// Active Request & Response
	ActiveRequest  RequestModel
	ActiveResponse *ResponseModel
	IsLoading      bool
	InFlightStage  int // 1: Pre-Script, 2: Auth, 3: Dispatch, 4: Tests
	ElapsedMs      int64

	// Environment Engine / Vault Manager
	envEngine EnvironmentManager

	// Active Environment
	ActiveEnv       string
	Environments    []Environment
	EnvDropdownOpen bool
	ShowVaultModal  bool
	ActiveVaultEnv  string
	VaultModalTab   string          // "variables", "transport"
	RevealedSecrets map[string]bool // key -> isRevealed

	// Template Interpolator & Assertion hooks
	interpolateFn    func(req RequestModel, vars map[string]string) RequestModel
	interpolateStrFn func(template string, vars map[string]string) string
	testEvalFn       func(script string, req RequestModel, resp *ResponseModel) []TestAssertionResult

	// Collection Suite Runner State
	suiteRunner         SuiteRunner
	runnerCancel        context.CancelFunc
	ShowRunnerModal     bool
	RunnerCollectionID  string
	RunnerConcurrency   int
	RunnerStopOnError   bool
	RunnerRunning       bool
	RunnerProgress      SuiteRunProgress
	RunnerReport        *SuiteRunReport
	RunnerFilter        string // "all", "passed", "failed"
	RunnerStatusMessage string

	// Universal Importer Modal State
	importer             ImporterService
	ShowImportModal      bool
	ImportRawText        string
	ImportFilePath       string
	ImportTargetName     string
	ImportDetectedFormat ImportFormat
	ImportPreviewData    *ImportPreview
	ImportError          string
	ImportStatusMessage  string

	// History Timeline Modal State
	ShowHistoryModal     bool
	HistoryEntries       []HistoryEntry
	HistoryTotalCount    int
	HistoryFilter        HistoryFilter
	SelectedHistoryID    string
	HistoryStatusMessage string

	// Navigation & View Mode
	ActiveTab            string // params, headers, body, auth, settings
	SidebarOpen          bool
	CommandPalOpen       bool
	PaletteQuery         string
	PaletteSelectedIndex int
	ShowAuthSecret       bool
}

// NewAppState initializes an AppState with default values.
func NewAppState(redrawFn func()) *AppState {
	return &AppState{
		redrawFn: redrawFn,
		ActiveRequest: RequestModel{
			ID:     "req_default",
			Name:   "Health Check",
			Method: "GET",
			URL:    "https://api.internal.net/v1/health",
			Params: []KeyValuePair{},
			Headers: []KeyValuePair{
				{Key: "Accept", Value: "application/json", Enabled: true},
			},
			BodyType: "json",
			AuthType: "none",
		},
		ActiveEnv:         "PROD",
		RunnerConcurrency: 4,
		RunnerFilter:      "all",
		ActiveVaultEnv:  "PROD",
		VaultModalTab:   "variables",
		RevealedSecrets: make(map[string]bool),
		Environments:    DefaultEnvironments(),
		Collections:     DefaultCollections(),
		CollapsedNodes:  make(map[string]bool),
		ActiveTab:     "params",
		SidebarOpen:   true,
		WorkspaceMode: "rest",
		Stream: StreamSession{
			Protocol:     StreamProtocolWSS,
			URL:          "wss://stream.internal.net/orders",
			Status:       StreamDisconnected,
			ComposerType: string(OpcodeText),
			ComposerBody: "{\n  \"action\": \"subscribe\",\n  \"channel\": \"orders.v2\"\n}",
		},
		GQLSchemaDrawerOpen: true,
		GQLRequest: GraphQLRequest{
			Endpoint:      "{{base_url}}/graphql",
			Query:         "query GetUsers($limit: Int = 10) {\n  users(limit: $limit) {\n    id\n    email\n  }\n}",
			VariablesJSON: "{\n  \"limit\": 10\n}",
		},
	}
}

// SetRedrawFunc attaches a redraw callback to the state.
func (s *AppState) SetRedrawFunc(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.redrawFn = fn
}

// SetFilePicker attaches a file selection handler (e.g., native OS dialog or test stub).
func (s *AppState) SetFilePicker(fn FilePickerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.filePicker = fn
}

// PickFile triggers the configured file picker if available.
func (s *AppState) PickFile(title string, extensions []string, onPicked func(path string)) {
	s.mu.RLock()
	fp := s.filePicker
	s.mu.RUnlock()
	if fp != nil && onPicked != nil {
		fp(title, extensions, onPicked)
	}
}

// SetSaveFilePicker attaches a save-file dialog handler (native OS dialog or test stub).
func (s *AppState) SetSaveFilePicker(fn SaveFilePickerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saveFilePicker = fn
}

// PickSaveFile triggers the configured save-file dialog if available.
func (s *AppState) PickSaveFile(title, defaultName string, extensions []string, onPicked func(path string)) {
	s.mu.RLock()
	sp := s.saveFilePicker
	s.mu.RUnlock()
	if sp != nil && onPicked != nil {
		sp(title, defaultName, extensions, onPicked)
	}
}

// SetExecutor configures the HTTP request execution engine.
func (s *AppState) SetExecutor(exec RequestExecutor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.executor = exec
}

// SetTestEvaluator attaches the JavaScript-style response assertion evaluator.
func (s *AppState) SetTestEvaluator(fn func(script string, req RequestModel, resp *ResponseModel) []TestAssertionResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.testEvalFn = fn
}

// SetHistoryStore attaches a persistent history store for logging completed request executions.
func (s *AppState) SetHistoryStore(store HistoryStore) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.historyStore = store
}

// HistoryStore returns the configured HistoryStore if any.
func (s *AppState) HistoryStore() HistoryStore {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.historyStore
}

// SetInterpolator registers the template interpolation functions for request execution and string resolution.
func (s *AppState) SetInterpolator(
	reqFn func(req RequestModel, vars map[string]string) RequestModel,
	strFn func(template string, vars map[string]string) string,
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.interpolateFn = reqFn
	s.interpolateStrFn = strFn
}

// SetEnvEngine attaches an EnvironmentManager to AppState and syncs environments.
func (s *AppState) SetEnvEngine(eng EnvironmentManager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.envEngine = eng
	if eng != nil {
		s.Environments = eng.GetEnvironments()
		if active := eng.GetActiveEnvironment(); active != nil {
			s.ActiveEnv = active.Name
			s.ActiveVaultEnv = active.Name
		}
	}
}

// EnvEngine returns the configured EnvironmentManager instance.
func (s *AppState) EnvEngine() EnvironmentManager {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.envEngine
}

// GetActiveVariables returns a map of all enabled variables in the active environment.
func (s *AppState) GetActiveVariables() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getActiveVariablesLocked()
}

func (s *AppState) getActiveVariablesLocked() map[string]string {
	if s.envEngine != nil {
		if resolved, err := s.envEngine.GetResolvedVariables(s.ActiveEnv); err == nil {
			return resolved
		}
	}
	vars := make(map[string]string)
	for _, env := range s.Environments {
		if env.Name == s.ActiveEnv {
			for _, v := range env.Variables {
				if v.Enabled {
					val := v.CurrentValue
					if val == "" {
						val = v.InitialValue
					}
					vars[v.Key] = val
				}
			}
			break
		}
	}
	return vars
}

// GetActiveEnvironment returns a copy of the currently active Environment.
func (s *AppState) GetActiveEnvironment() *Environment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, env := range s.Environments {
		if env.Name == s.ActiveEnv {
			envCopy := env
			return &envCopy
		}
	}
	return nil
}

// InterpolateActiveString resolves template variables in a string against active environment variables.
func (s *AppState) InterpolateActiveString(template string) string {
	s.mu.RLock()
	strFn := s.interpolateStrFn
	vars := s.getActiveVariablesLocked()
	s.mu.RUnlock()

	if strFn != nil {
		return strFn(template, vars)
	}
	return template
}

// RequestRedraw safely triggers a frame re-render on the UI thread.
func (s *AppState) RequestRedraw() {
	s.mu.RLock()
	fn := s.redrawFn
	s.mu.RUnlock()

	if fn != nil {
		fn()
	}
}

// Mutate performs thread-safe state mutation, auto-persisting uncommitted scratchpad edits, and triggers a redraw.
func (s *AppState) Mutate(fn func(s *AppState)) {
	var draftSnapshot *ScratchpadDraft
	var store DraftStore

	s.mu.Lock()
	var trackDraft bool
	var before RequestModel
	if s.draftStore != nil && s.ActiveItemID == "" {
		trackDraft = true
		before = CloneRequestModel(s.ActiveRequest)
	}
	fn(s)
	if trackDraft && s.ActiveItemID == "" && !reflect.DeepEqual(before, s.ActiveRequest) {
		snap := s.prepareDraftSnapshotLocked()
		draftSnapshot = &snap
		store = s.draftStore
	}
	s.mu.Unlock()

	if draftSnapshot != nil && store != nil {
		if saved, err := store.SaveDraft(*draftSnapshot); err == nil {
			s.upsertDraftSnapshot(saved)
		}
	}

	s.RequestRedraw()
}

// View executes a read-only closure with read-lock held.
func (s *AppState) View(fn func(s *AppState)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fn(s)
}

// SetMethod updates the HTTP verb and requests redraw.
func (s *AppState) SetMethod(method string) {
	s.Mutate(func(state *AppState) {
		state.ActiveRequest.Method = method
	})
}

// SetURL updates the URL string and syncs extracted parameters, preserving custom descriptions and disabled rows.
func (s *AppState) SetURL(rawURL string) {
	s.Mutate(func(state *AppState) {
		state.ActiveRequest.URL = rawURL
		state.ActiveRequest.Params = MergeURLParams(state.ActiveRequest.Params, rawURL)
	})
}

// SetParams updates parameters and syncs back to the URL.
func (s *AppState) SetParams(params []KeyValuePair) {
	s.Mutate(func(state *AppState) {
		state.ActiveRequest.Params = params
		state.ActiveRequest.URL = SyncParamsToURL(state.ActiveRequest.URL, params)
	})
}

// AddParam adds a new parameter and updates the URL.
func (s *AppState) AddParam(key, value, desc string) {
	s.Mutate(func(state *AppState) {
		state.ActiveRequest.Params = append(state.ActiveRequest.Params, KeyValuePair{
			Key:         key,
			Value:       value,
			Description: desc,
			Enabled:     true,
		})
		state.ActiveRequest.URL = SyncParamsToURL(state.ActiveRequest.URL, state.ActiveRequest.Params)
	})
}

// RemoveParam removes a parameter by index and updates the URL.
func (s *AppState) RemoveParam(index int) {
	s.Mutate(func(state *AppState) {
		if index >= 0 && index < len(state.ActiveRequest.Params) {
			state.ActiveRequest.Params = append(state.ActiveRequest.Params[:index], state.ActiveRequest.Params[index+1:]...)
			state.ActiveRequest.URL = SyncParamsToURL(state.ActiveRequest.URL, state.ActiveRequest.Params)
		}
	})
}

// ToggleParam flips the enabled status of a parameter and updates the URL.
func (s *AppState) ToggleParam(index int) {
	s.Mutate(func(state *AppState) {
		if index >= 0 && index < len(state.ActiveRequest.Params) {
			state.ActiveRequest.Params[index].Enabled = !state.ActiveRequest.Params[index].Enabled
			state.ActiveRequest.URL = SyncParamsToURL(state.ActiveRequest.URL, state.ActiveRequest.Params)
		}
	})
}

// SetHeaders updates request headers.
func (s *AppState) SetHeaders(headers []KeyValuePair) {
	s.Mutate(func(state *AppState) {
		state.ActiveRequest.Headers = headers
	})
}

// AddHeader appends a new header row.
func (s *AppState) AddHeader(key, value, desc string) {
	s.Mutate(func(state *AppState) {
		state.ActiveRequest.Headers = append(state.ActiveRequest.Headers, KeyValuePair{
			Key:         key,
			Value:       value,
			Description: desc,
			Enabled:     true,
		})
	})
}

// RemoveHeader deletes a header row by index.
func (s *AppState) RemoveHeader(index int) {
	s.Mutate(func(state *AppState) {
		if index >= 0 && index < len(state.ActiveRequest.Headers) {
			state.ActiveRequest.Headers = append(state.ActiveRequest.Headers[:index], state.ActiveRequest.Headers[index+1:]...)
		}
	})
}

// ToggleHeader flips the enabled status of a header row.
func (s *AppState) ToggleHeader(index int) {
	s.Mutate(func(state *AppState) {
		if index >= 0 && index < len(state.ActiveRequest.Headers) {
			state.ActiveRequest.Headers[index].Enabled = !state.ActiveRequest.Headers[index].Enabled
		}
	})
}

// SetActiveTab updates the active configuration tab.
func (s *AppState) SetActiveTab(tab string) {
	s.Mutate(func(state *AppState) {
		state.ActiveTab = tab
	})
}

// SetLoading updates loading state and elapsed duration.
func (s *AppState) SetLoading(loading bool, elapsedMs int64) {
	s.Mutate(func(state *AppState) {
		state.IsLoading = loading
		state.ElapsedMs = elapsedMs
	})
}

// SetResponse sets the execution response and requests redraw.
func (s *AppState) SetResponse(resp *ResponseModel) {
	s.Mutate(func(state *AppState) {
		state.ActiveResponse = resp
		state.IsLoading = false
	})
}

// SetActiveEnv switches environment (PROD, STG, DEV, etc.).
func (s *AppState) SetActiveEnv(env string) {
	s.Mutate(func(state *AppState) {
		state.ActiveEnv = env
		state.ActiveVaultEnv = env
		if state.envEngine != nil {
			_ = state.envEngine.SetActiveEnvironment(env)
		}
	})
}

// ToggleEnvDropdown toggles visibility of the environment switcher dropdown.
func (s *AppState) ToggleEnvDropdown() {
	s.Mutate(func(state *AppState) {
		state.EnvDropdownOpen = !state.EnvDropdownOpen
	})
}

// OpenEnvDropdown opens the environment switcher dropdown.
func (s *AppState) OpenEnvDropdown() {
	s.Mutate(func(state *AppState) {
		state.EnvDropdownOpen = true
	})
}

// CloseEnvDropdown closes the environment switcher dropdown.
func (s *AppState) CloseEnvDropdown() {
	s.Mutate(func(state *AppState) {
		state.EnvDropdownOpen = false
	})
}

// SwitchEnvByIndex switches the active environment to the environment at the given index.
func (s *AppState) SwitchEnvByIndex(idx int) {
	s.Mutate(func(state *AppState) {
		if idx >= 0 && idx < len(state.Environments) {
			name := state.Environments[idx].Name
			state.ActiveEnv = name
			state.ActiveVaultEnv = name
			if state.envEngine != nil {
				_ = state.envEngine.SetActiveEnvironment(name)
			}
		}
	})
}

// ToggleVaultModal toggles the Environment & Secret Vault Manager modal.
func (s *AppState) ToggleVaultModal() {
	s.Mutate(func(state *AppState) {
		state.ShowVaultModal = !state.ShowVaultModal
		if state.ShowVaultModal {
			state.EnvDropdownOpen = false
		}
	})
}

// OpenVaultModal opens the Environment & Secret Vault Manager modal.
func (s *AppState) OpenVaultModal() {
	s.Mutate(func(state *AppState) {
		state.ShowVaultModal = true
		state.EnvDropdownOpen = false
	})
}

// CloseVaultModal closes the Environment & Secret Vault Manager modal.
func (s *AppState) CloseVaultModal() {
	s.Mutate(func(state *AppState) {
		state.ShowVaultModal = false
	})
}

// SetActiveVaultEnv sets which environment is being edited inside the vault modal.
func (s *AppState) SetActiveVaultEnv(env string) {
	s.Mutate(func(state *AppState) {
		state.ActiveVaultEnv = env
	})
}

// SetVaultModalTab updates active tab in the vault modal ("variables" or "transport").
func (s *AppState) SetVaultModalTab(tab string) {
	s.Mutate(func(state *AppState) {
		state.VaultModalTab = tab
	})
}

// ToggleSecretReveal toggles visibility of a specific secret variable.
func (s *AppState) ToggleSecretReveal(key string) {
	s.Mutate(func(state *AppState) {
		if state.RevealedSecrets == nil {
			state.RevealedSecrets = make(map[string]bool)
		}
		composite := key
		if !strings.Contains(key, ":") && state.ActiveVaultEnv != "" {
			composite = state.ActiveVaultEnv + ":" + key
		}
		state.RevealedSecrets[composite] = !state.RevealedSecrets[composite]
		state.RevealedSecrets[key] = state.RevealedSecrets[composite]
	})
}

// ToggleSecretRevealForEnv toggles secret visibility scoped to a specific environment.
func (s *AppState) ToggleSecretRevealForEnv(envName, key string) {
	s.Mutate(func(state *AppState) {
		if state.RevealedSecrets == nil {
			state.RevealedSecrets = make(map[string]bool)
		}
		composite := envName + ":" + key
		state.RevealedSecrets[composite] = !state.RevealedSecrets[composite]
		state.RevealedSecrets[key] = state.RevealedSecrets[composite]
	})
}

// IsSecretRevealed checks if a secret variable is unmasked.
func (s *AppState) IsSecretRevealed(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.RevealedSecrets == nil {
		return false
	}
	if s.ActiveVaultEnv != "" {
		if val, ok := s.RevealedSecrets[s.ActiveVaultEnv+":"+key]; ok {
			return val
		}
	}
	return s.RevealedSecrets[key]
}

// IsSecretRevealedForEnv checks if a secret variable is unmasked for the given environment.
func (s *AppState) IsSecretRevealedForEnv(envName, key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.RevealedSecrets == nil {
		return false
	}
	composite := envName + ":" + key
	if val, ok := s.RevealedSecrets[composite]; ok {
		return val
	}
	return s.RevealedSecrets[key]
}

// GetSecretPlaintext retrieves the decrypted secret from the vault engine or active variables.
func (s *AppState) GetSecretPlaintext(envName, key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.envEngine != nil {
		if val, err := s.envEngine.GetSecret(envName, key); err == nil && val != "" {
			return val, true
		}
	}
	for _, env := range s.Environments {
		if strings.EqualFold(env.Name, envName) {
			for _, v := range env.Variables {
				if v.Key == key && v.CurrentValue != "" {
					return v.CurrentValue, true
				}
			}
		}
	}
	return "", false
}

// AddVariableToEnv adds a new variable to the given environment.
func (s *AppState) AddVariableToEnv(envName string, v Variable) {
	s.AddVariableToEnvWithSecret(envName, v, "")
}

// AddVariableToEnvWithSecret adds a new variable with optional plaintext secret for vault encryption.
func (s *AppState) AddVariableToEnvWithSecret(envName string, v Variable, secretPlaintext string) {
	s.Mutate(func(state *AppState) {
		if state.envEngine != nil {
			_ = state.envEngine.AddVariable(envName, v, secretPlaintext)
			state.Environments = state.envEngine.GetEnvironments()
			return
		}
		for i := range state.Environments {
			if strings.EqualFold(state.Environments[i].Name, envName) {
				state.Environments[i].Variables = append(state.Environments[i].Variables, v)
				break
			}
		}
	})
}

// UpdateVariableInEnv updates a variable in the given environment.
func (s *AppState) UpdateVariableInEnv(envName string, key string, v Variable) {
	s.UpdateVariableInEnvWithSecret(envName, key, v, "")
}

// UpdateVariableInEnvWithSecret updates a variable in the given environment with optional secret plaintext.
func (s *AppState) UpdateVariableInEnvWithSecret(envName string, key string, v Variable, secretPlaintext string) {
	s.Mutate(func(state *AppState) {
		if state.envEngine != nil {
			_ = state.envEngine.UpdateVariable(envName, key, v, secretPlaintext)
			state.Environments = state.envEngine.GetEnvironments()
			return
		}
		for i := range state.Environments {
			if strings.EqualFold(state.Environments[i].Name, envName) {
				for j := range state.Environments[i].Variables {
					if state.Environments[i].Variables[j].Key == key {
						state.Environments[i].Variables[j] = v
						return
					}
				}
			}
		}
	})
}

// DeleteVariableFromEnv deletes a variable from the given environment.
func (s *AppState) DeleteVariableFromEnv(envName string, key string) {
	s.Mutate(func(state *AppState) {
		if state.envEngine != nil {
			_ = state.envEngine.DeleteVariable(envName, key)
			state.Environments = state.envEngine.GetEnvironments()
			return
		}
		for i := range state.Environments {
			if strings.EqualFold(state.Environments[i].Name, envName) {
				vars := state.Environments[i].Variables
				for j := range vars {
					if vars[j].Key == key {
						state.Environments[i].Variables = append(vars[:j], vars[j+1:]...)
						return
					}
				}
			}
		}
	})
}

// CreateEnvironment adds a new workspace environment.
func (s *AppState) CreateEnvironment(name, color string) {
	s.Mutate(func(state *AppState) {
		if color == "" {
			color = "#38BDF8"
		}
		newEnv := Environment{
			ID:    "env_" + strings.ToLower(name),
			Name:  name,
			Color: color,
			Variables: []Variable{
				{Key: "base_url", InitialValue: "http://localhost:8080", CurrentValue: "http://localhost:8080", Type: VariableTypeText, Enabled: true},
			},
			TransportPolicy: TransportPolicy{
				Protocol:  "HTTP/2",
				StrictSSL: true,
			},
		}
		if state.envEngine != nil {
			_ = state.envEngine.CreateEnvironment(newEnv)
			state.Environments = state.envEngine.GetEnvironments()
		} else {
			state.Environments = append(state.Environments, newEnv)
		}
		state.ActiveVaultEnv = name
	})
}

// DeleteEnvironment removes an environment by name.
func (s *AppState) DeleteEnvironment(name string) {
	s.Mutate(func(state *AppState) {
		if len(state.Environments) <= 1 {
			return
		}
		if state.envEngine != nil {
			_ = state.envEngine.DeleteEnvironment(name)
			state.Environments = state.envEngine.GetEnvironments()
		} else {
			index := -1
			for i, env := range state.Environments {
				if env.Name == name {
					index = i
					break
				}
			}
			if index != -1 {
				state.Environments = append(state.Environments[:index], state.Environments[index+1:]...)
			}
		}
		if state.ActiveEnv == name && len(state.Environments) > 0 {
			state.ActiveEnv = state.Environments[0].Name
		}
		if state.ActiveVaultEnv == name && len(state.Environments) > 0 {
			state.ActiveVaultEnv = state.Environments[0].Name
		}
	})
}

// DuplicateEnvironment duplicates an environment by name.
func (s *AppState) DuplicateEnvironment(name, newName string) {
	s.Mutate(func(state *AppState) {
		if state.envEngine != nil {
			if _, err := state.envEngine.DuplicateEnvironment(name, newName); err == nil {
				state.Environments = state.envEngine.GetEnvironments()
				state.ActiveVaultEnv = newName
				return
			}
		}
		var src *Environment
		for _, env := range state.Environments {
			if env.Name == name {
				cpy := env
				src = &cpy
				break
			}
		}
		if src != nil {
			dup := *src
			dup.ID = "env_" + strings.ToLower(newName)
			dup.Name = newName
			dup.IsDefault = false
			dupVars := make([]Variable, len(src.Variables))
			copy(dupVars, src.Variables)
			dup.Variables = dupVars
			state.Environments = append(state.Environments, dup)
			state.ActiveVaultEnv = newName
		}
	})
}

// UpdateTransportPolicy updates the transport policy settings for an environment.
func (s *AppState) UpdateTransportPolicy(envName string, p TransportPolicy) {
	s.Mutate(func(state *AppState) {
		for i := range state.Environments {
			if state.Environments[i].Name == envName {
				state.Environments[i].TransportPolicy = p
				if state.envEngine != nil {
					_ = state.envEngine.UpdateEnvironment(state.Environments[i])
				}
				break
			}
		}
	})
}

// ExportDotEnv exports an environment to .env format.
func (s *AppState) ExportDotEnv(envName string, maskSecrets bool) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.envEngine != nil {
		return s.envEngine.ExportEnvironment(envName, maskSecrets)
	}
	return "", fmt.Errorf("vault engine not configured")
}

// ImportDotEnv imports a .env format string into the named environment.
func (s *AppState) ImportDotEnv(envName string, content string) error {
	s.Mutate(func(state *AppState) {
		if state.envEngine != nil {
			env, err := state.envEngine.ImportEnvironment(envName, content)
			if err == nil {
				state.Environments = state.envEngine.GetEnvironments()
				state.ActiveVaultEnv = env.Name
			}
		}
	})
	return nil
}

// ToggleAuthSecret flips visibility of masked passwords and tokens.
func (s *AppState) ToggleAuthSecret() {
	s.Mutate(func(state *AppState) {
		state.ShowAuthSecret = !state.ShowAuthSecret
	})
}

// SendRequest triggers dispatching ActiveRequest asynchronously with live timer updates.
func (s *AppState) SendRequest() {
	s.mu.Lock()
	if s.IsLoading {
		s.mu.Unlock()
		return
	}
	s.IsLoading = true
	s.InFlightStage = 1
	s.ElapsedMs = 0
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFn = cancel
	exec := s.executor
	reqCopy := s.ActiveRequest
	reqCopy.Params = append([]KeyValuePair(nil), s.ActiveRequest.Params...)
	reqCopy.Headers = append([]KeyValuePair(nil), s.ActiveRequest.Headers...)
	reqCopy.FormData = append([]KeyValuePair(nil), s.ActiveRequest.FormData...)
	reqCopy.URLEncoded = append([]KeyValuePair(nil), s.ActiveRequest.URLEncoded...)

	if s.interpolateFn != nil {
		activeVars := s.getActiveVariablesLocked()
		reqCopy = s.interpolateFn(reqCopy, activeVars)
	}
	if reqCopy.AuthType != "" && reqCopy.AuthType != "none" {
		s.InFlightStage = 2
	}

	if reqCopy.TransportPolicy == nil {
		for _, env := range s.Environments {
			if env.Name == s.ActiveEnv {
				policyCopy := env.TransportPolicy
				reqCopy.TransportPolicy = &policyCopy
				break
			}
		}
	}
	activeEnv := s.ActiveEnv
	histStore := s.historyStore
	testEval := s.testEvalFn
	s.InFlightStage = 3
	s.mu.Unlock()

	s.RequestRedraw()

	go func() {
		startTime := time.Now()
		done := make(chan struct{})

		// Background ticker to update elapsed duration in real time
		go func() {
			ticker := time.NewTicker(30 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					s.mu.Lock()
					if !s.IsLoading {
						s.mu.Unlock()
						return
					}
					s.ElapsedMs = time.Since(startTime).Milliseconds()
					s.mu.Unlock()
					s.RequestRedraw()
				}
			}
		}()

		var resp *ResponseModel
		var err error
		if exec != nil {
			resp, err = exec.Execute(ctx, reqCopy)
		} else {
			// Built-in simulated response for testing without mock server
			time.Sleep(30 * time.Millisecond)
			resp = &ResponseModel{
				StatusCode: 200,
				StatusText: "200 OK",
				Duration:   30 * time.Millisecond,
				SizeBytes:  1024,
				Proto:      "HTTP/2.0",
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       "{\n  \"status\": \"success\"\n}",
				Telemetry: Telemetry{
					DNS:      5,
					TCP:      10,
					TLS:      15,
					TTFB:     20,
					Download: 5,
					Total:    55,
				},
			}
		}
		close(done)

		if err != nil && resp == nil {
			resp = &ResponseModel{
				Error: err.Error(),
			}
		}

		if testEval != nil && strings.TrimSpace(reqCopy.TestScript) != "" {
			s.mu.Lock()
			s.InFlightStage = 4
			s.mu.Unlock()
			resp.Assertions = testEval(reqCopy.TestScript, reqCopy, resp)
		}

		elapsedMs := time.Since(startTime).Milliseconds()
		if histStore != nil {
			_, _ = histStore.RecordHistory(buildHistoryEntry(reqCopy, resp, activeEnv, elapsedMs))
		}

		s.mu.Lock()
		s.ActiveResponse = resp
		s.IsLoading = false
		s.InFlightStage = 0
		s.cancelFn = nil
		s.ElapsedMs = elapsedMs
		s.mu.Unlock()

		s.RequestRedraw()
	}()
}

// CancelRequest cancels the currently active request if any.
func (s *AppState) CancelRequest() {
	s.mu.Lock()
	if s.cancelFn != nil {
		s.cancelFn()
		s.cancelFn = nil
	}
	s.IsLoading = false
	s.InFlightStage = 0
	s.mu.Unlock()

	s.RequestRedraw()
}

// BindServices wires mygo Context Services to trigger Invalidate on state mutations.
func (s *AppState) BindServices(services ui.Services) {
	s.SetRedrawFunc(func() {
		services.Invalidate()
	})
}

func buildHistoryEntry(req RequestModel, resp *ResponseModel, activeEnv string, fallbackMs int64) HistoryEntry {
	entry := HistoryEntry{
		RequestID:       req.ID,
		Method:          req.Method,
		URL:             req.URL,
		DurationMs:      fallbackMs,
		EnvironmentName: activeEnv,
		ExecutedAt:      time.Now().UTC(),
		Request:         CloneRequestModel(req),
	}
	if resp != nil {
		respCopy := *resp
		entry.Response = &respCopy
		entry.StatusCode = resp.StatusCode
		entry.StatusText = resp.StatusText
		entry.SizeBytes = resp.SizeBytes
		entry.Protocol = resp.Proto
		entry.Error = resp.Error
		entry.Telemetry = resp.Telemetry
		if d := resp.Duration.Milliseconds(); d > 0 {
			entry.DurationMs = d
		}
	}
	return entry
}
