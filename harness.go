package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Harness orchestrates agent execution with thread persistence,
// mode management, tool approval, and event streaming.
// It wraps Mastra's JS Harness via the Kit's QuickJS runtime.
type Harness struct {
	kit    *Kit
	config HarnessConfig

	subscribers []harnessSubscriber
	subMu       sync.RWMutex
	nextSubID   int

	displayState *DisplayState
	dsMu         sync.RWMutex

	tokenUsage TokenUsage
	tuMu       sync.RWMutex

	threadLock *ThreadLock

	heartbeats map[string]*time.Ticker
	hbMu       sync.Mutex

	initialized bool
	closed      bool
}

// InitHarness creates and initializes a Harness on this Kit.
// Agents referenced by modes must be created before calling this.
func (k *Kit) InitHarness(cfg HarnessConfig) (*Harness, error) {
	if err := validateHarnessConfig(cfg); err != nil {
		return nil, err
	}

	lock := cfg.ThreadLock
	if lock == nil {
		lock = defaultThreadLock()
	}

	h := &Harness{
		kit:          k,
		config:       cfg,
		displayState: NewDisplayState(),
		threadLock:   lock,
		heartbeats:   make(map[string]*time.Ticker),
	}

	// Register Go→JS bridge functions
	h.registerEventBridge()
	h.registerLockBridges()

	// Build config JSON for the JS createHarness() function
	jsConfig := h.buildJSConfig()
	configJSON, err := json.Marshal(jsConfig)
	if err != nil {
		return nil, fmt.Errorf("harness: marshal config: %w", err)
	}

	// Create JS Harness via createHarness(configJSON)
	createCode := fmt.Sprintf(`await __kit.createHarness(%s)`, quoteJSString(string(configJSON)))
	if _, err := k.EvalTS(context.Background(), "harness-create.ts", createCode); err != nil {
		return nil, fmt.Errorf("harness: create JS harness: %w", err)
	}

	// Initialize (loads storage, workspace, selects thread)
	if _, err := k.EvalTS(context.Background(), "harness-init.ts", `await __brainkit_harness.init()`); err != nil {
		return nil, fmt.Errorf("harness: init: %w", err)
	}

	// Start Go-side heartbeat timers
	h.startHeartbeats()

	h.initialized = true
	return h, nil
}

// Close stops heartbeats, releases locks, and tears down the Harness.
func (h *Harness) Close() error {
	if h.closed {
		return nil
	}
	h.closed = true
	h.stopHeartbeats()
	return nil
}

// GetDisplayState returns a thread-safe deep copy of the canonical display state.
func (h *Harness) GetDisplayState() *DisplayState {
	h.dsMu.RLock()
	defer h.dsMu.RUnlock()
	return h.displayState.clone()
}

// GetTokenUsage returns the accumulated token usage.
func (h *Harness) GetTokenUsage() TokenUsage {
	h.tuMu.RLock()
	defer h.tuMu.RUnlock()
	return h.tokenUsage
}

// IsRunning returns true if the agent is currently processing.
func (h *Harness) IsRunning() bool {
	h.dsMu.RLock()
	defer h.dsMu.RUnlock()
	return h.displayState.IsRunning
}

// ---------------------------------------------------------------------------
// Core Messaging
// ---------------------------------------------------------------------------

// SendMessage sends a user message to the current agent.
// Blocks until the agent finishes. Events stream to subscribers during execution.
func (h *Harness) SendMessage(content string, opts ...SendOption) error {
	o := &sendOptions{}
	for _, opt := range opts {
		opt(o)
	}
	args := map[string]any{"content": content}
	if o.files != nil {
		args["files"] = o.files
	}
	if o.requestContext != nil {
		args["requestContext"] = o.requestContext
	}
	b, _ := json.Marshal(args)
	return h.callJSVoid("sendMessage", string(b))
}

// Abort cancels the current agent execution.
func (h *Harness) Abort() error {
	return h.callJSVoid("abort", "")
}

// Steer aborts the current execution and sends a new message.
func (h *Harness) Steer(content string, opts ...SendOption) error {
	o := &sendOptions{}
	for _, opt := range opts {
		opt(o)
	}
	args := map[string]any{"content": content}
	b, _ := json.Marshal(args)
	return h.callJSVoid("steer", string(b))
}

// FollowUp queues a message after the current execution finishes.
func (h *Harness) FollowUp(content string, opts ...SendOption) error {
	o := &sendOptions{}
	for _, opt := range opts {
		opt(o)
	}
	args := map[string]any{"content": content}
	b, _ := json.Marshal(args)
	return h.callJSVoid("followUp", string(b))
}

// GetCurrentRunID returns the active run ID, or empty string.
func (h *Harness) GetCurrentRunID() string {
	r, _ := h.callJSSimple("getCurrentRunId")
	var s string
	json.Unmarshal([]byte(r), &s)
	return s
}

// ---------------------------------------------------------------------------
// Thread Management
// ---------------------------------------------------------------------------

// CreateThread creates a new conversation thread.
func (h *Harness) CreateThread(opts ...ThreadOption) (string, error) {
	o := &threadOptions{}
	for _, opt := range opts {
		opt(o)
	}
	args := map[string]any{}
	if o.title != "" {
		args["title"] = o.title
	}
	b, _ := json.Marshal(args)
	r, err := h.callJS("createThread", string(b))
	if err != nil {
		return "", err
	}
	// createThread returns a HarnessThread object { id, title, ... }
	var thread HarnessThread
	if err := json.Unmarshal([]byte(r), &thread); err != nil {
		// Fallback: try as plain string
		var id string
		json.Unmarshal([]byte(r), &id)
		return id, nil
	}
	return thread.ID, nil
}

// SwitchThread switches to a different thread.
func (h *Harness) SwitchThread(threadID string) error {
	b, _ := json.Marshal(map[string]string{"threadId": threadID})
	return h.callJSVoid("switchThread", string(b))
}

// DeleteThread deletes a thread.
func (h *Harness) DeleteThread(threadID string) error {
	b, _ := json.Marshal(map[string]string{"threadId": threadID})
	return h.callJSVoid("deleteThread", string(b))
}

// ListThreads returns all threads, optionally filtered by resource.
func (h *Harness) ListThreads(opts ...ListThreadsOption) ([]HarnessThread, error) {
	o := &listThreadsOptions{}
	for _, opt := range opts {
		opt(o)
	}
	var argsJSON string
	if o.resourceID != "" {
		b, _ := json.Marshal(map[string]string{"resourceId": o.resourceID})
		argsJSON = string(b)
	}
	r, err := h.callJS("listThreads", argsJSON)
	if err != nil {
		return nil, err
	}
	var threads []HarnessThread
	json.Unmarshal([]byte(r), &threads)
	return threads, nil
}

// RenameThread renames the current thread.
func (h *Harness) RenameThread(title string) error {
	b, _ := json.Marshal(map[string]string{"title": title})
	return h.callJSVoid("renameThread", string(b))
}

// CloneThread clones a thread.
func (h *Harness) CloneThread(opts ...CloneOption) (string, error) {
	o := &cloneOptions{}
	for _, opt := range opts {
		opt(o)
	}
	args := map[string]any{}
	if o.sourceThreadID != "" {
		args["sourceThreadId"] = o.sourceThreadID
	}
	if o.title != "" {
		args["title"] = o.title
	}
	if o.resourceID != "" {
		args["resourceId"] = o.resourceID
	}
	b, _ := json.Marshal(args)
	r, err := h.callJS("cloneThread", string(b))
	if err != nil {
		return "", err
	}
	var thread HarnessThread
	if err := json.Unmarshal([]byte(r), &thread); err != nil {
		var id string
		json.Unmarshal([]byte(r), &id)
		return id, nil
	}
	return thread.ID, nil
}

// GetCurrentThreadID returns the current thread ID.
func (h *Harness) GetCurrentThreadID() string {
	r, _ := h.callJSSimple("getCurrentThreadId")
	var s string
	json.Unmarshal([]byte(r), &s)
	return s
}

// ListMessages returns messages for the current or specified thread.
func (h *Harness) ListMessages(opts ...ListMessagesOption) ([]HarnessMessage, error) {
	o := &listMessagesOptions{}
	for _, opt := range opts {
		opt(o)
	}
	args := map[string]any{}
	if o.threadID != "" {
		args["threadId"] = o.threadID
	}
	if o.limit > 0 {
		args["limit"] = o.limit
	}
	b, _ := json.Marshal(args)
	r, err := h.callJS("listMessages", string(b))
	if err != nil {
		return nil, err
	}
	var msgs []HarnessMessage
	json.Unmarshal([]byte(r), &msgs)
	return msgs, nil
}

// ---------------------------------------------------------------------------
// Mode Management
// ---------------------------------------------------------------------------

// SwitchMode switches to a different mode.
func (h *Harness) SwitchMode(modeID string) error {
	b, _ := json.Marshal(map[string]string{"modeId": modeID})
	return h.callJSVoid("switchMode", string(b))
}

// ListModes returns all configured modes.
func (h *Harness) ListModes() []Mode {
	r, _ := h.callJSSimple("listModes")
	var modes []Mode
	json.Unmarshal([]byte(r), &modes)
	return modes
}

// GetCurrentMode returns the active mode.
func (h *Harness) GetCurrentMode() Mode {
	r, _ := h.callJSSimple("getCurrentMode")
	var mode Mode
	json.Unmarshal([]byte(r), &mode)
	return mode
}

// GetCurrentModeID returns the active mode ID.
func (h *Harness) GetCurrentModeID() string {
	r, _ := h.callJSSimple("getCurrentModeId")
	var s string
	json.Unmarshal([]byte(r), &s)
	return s
}

// ---------------------------------------------------------------------------
// Model Management
// ---------------------------------------------------------------------------

// SwitchModel changes the active model.
func (h *Harness) SwitchModel(modelID string, opts ...ModelOption) error {
	o := &modelOptions{}
	for _, opt := range opts {
		opt(o)
	}
	args := map[string]any{"modelId": modelID}
	if o.scope != "" {
		args["scope"] = o.scope
	}
	if o.modeID != "" {
		args["modeId"] = o.modeID
	}
	b, _ := json.Marshal(args)
	return h.callJSVoid("switchModel", string(b))
}

// ListAvailableModels returns all available models with auth status.
func (h *Harness) ListAvailableModels() ([]AvailableModel, error) {
	r, err := h.callJSSimple("listAvailableModels")
	if err != nil {
		return nil, err
	}
	var models []AvailableModel
	json.Unmarshal([]byte(r), &models)
	return models, nil
}

// GetCurrentModelID returns the current model ID.
func (h *Harness) GetCurrentModelID() string {
	r, _ := h.callJSSimple("getCurrentModelId")
	var s string
	json.Unmarshal([]byte(r), &s)
	return s
}

// HasModelSelected returns true if a model is selected.
func (h *Harness) HasModelSelected() bool {
	r, _ := h.callJSSimple("hasModelSelected")
	var b bool
	json.Unmarshal([]byte(r), &b)
	return b
}

// ---------------------------------------------------------------------------
// Permissions & Tool Approval
// ---------------------------------------------------------------------------

// RespondToToolApproval responds to a tool approval request.
// Uses direct bridge eval to avoid nested async issues during agent stream.
func (h *Harness) RespondToToolApproval(decision ToolApprovalDecision) error {
	b, _ := json.Marshal(map[string]string{"decision": string(decision)})
	code := fmt.Sprintf(`__brainkit_harness.respondToToolApproval(JSON.parse(%s))`, quoteJSString(string(b)))
	if h.kit.bridge.IsEvalBusy() {
		_, err := h.kit.bridge.EvalOnJSThread("harness-respond-approval.js", code)
		return err
	}
	_, err := h.kit.bridge.Eval("harness-respond-approval.js", code)
	if err != nil {
		return fmt.Errorf("respondToToolApproval: %w", err)
	}
	return nil
}

// SetPermissionForCategory sets the default policy for a tool category.
func (h *Harness) SetPermissionForCategory(category, policy string) error {
	b, _ := json.Marshal(map[string]string{"category": category, "policy": policy})
	return h.callJSVoid("setPermissionForCategory", string(b))
}

// SetPermissionForTool overrides the category policy for a specific tool.
func (h *Harness) SetPermissionForTool(toolName, policy string) error {
	b, _ := json.Marshal(map[string]string{"toolName": toolName, "policy": policy})
	return h.callJSVoid("setPermissionForTool", string(b))
}

// GetPermissionRules returns the current permission rules.
func (h *Harness) GetPermissionRules() PermissionRules {
	r, _ := h.callJSSimple("getPermissionRules")
	var rules PermissionRules
	json.Unmarshal([]byte(r), &rules)
	return rules
}

// GrantSessionCategory auto-approves all tools in a category for this session.
func (h *Harness) GrantSessionCategory(category string) error {
	b, _ := json.Marshal(map[string]string{"category": category})
	return h.callJSVoid("grantSessionCategory", string(b))
}

// GrantSessionTool auto-approves a specific tool for this session.
func (h *Harness) GrantSessionTool(toolName string) error {
	b, _ := json.Marshal(map[string]string{"toolName": toolName})
	return h.callJSVoid("grantSessionTool", string(b))
}

// ---------------------------------------------------------------------------
// Interactive Tools
// ---------------------------------------------------------------------------

// RespondToQuestion answers an ask_user tool invocation.
// Uses direct bridge eval (not EvalTS) to avoid nested async wrapper issues
// when called while SendMessage is awaiting the agent stream.
func (h *Harness) RespondToQuestion(questionID, answer string) error {
	b, _ := json.Marshal(map[string]string{"questionId": questionID, "answer": answer})
	code := fmt.Sprintf(`__brainkit_harness.respondToQuestion(JSON.parse(%s))`, quoteJSString(string(b)))
	if h.kit.bridge.IsEvalBusy() {
		_, err := h.kit.bridge.EvalOnJSThread("harness-respond-question.js", code)
		return err
	}
	_, err := h.kit.bridge.Eval("harness-respond-question.js", code)
	if err != nil {
		return fmt.Errorf("respondToQuestion: %w", err)
	}
	return nil
}

// RespondToPlanApproval responds to a plan approval request.
// Uses direct bridge eval to avoid nested async issues during agent stream.
func (h *Harness) RespondToPlanApproval(planID string, resp PlanResponse) error {
	b, _ := json.Marshal(map[string]any{"planId": planID, "response": resp})
	code := fmt.Sprintf(`__brainkit_harness.respondToPlanApproval(JSON.parse(%s))`, quoteJSString(string(b)))
	if h.kit.bridge.IsEvalBusy() {
		_, err := h.kit.bridge.EvalOnJSThread("harness-respond-plan.js", code)
		return err
	}
	_, err := h.kit.bridge.Eval("harness-respond-plan.js", code)
	if err != nil {
		return fmt.Errorf("respondToPlanApproval: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// State Management
// ---------------------------------------------------------------------------

// GetState returns the current Harness state.
func (h *Harness) GetState() map[string]any {
	r, _ := h.callJSSimple("getState")
	var state map[string]any
	json.Unmarshal([]byte(r), &state)
	return state
}

// SetState updates Harness state. Validated by Zod in JS.
func (h *Harness) SetState(updates map[string]any) error {
	b, _ := json.Marshal(updates)
	return h.callJSVoid("setState", string(b))
}

// ---------------------------------------------------------------------------
// Observational Memory
// ---------------------------------------------------------------------------

// SwitchObserverModel changes the OM observer model.
func (h *Harness) SwitchObserverModel(modelID string) error {
	b, _ := json.Marshal(map[string]string{"modelId": modelID})
	return h.callJSVoid("switchObserverModel", string(b))
}

// SwitchReflectorModel changes the OM reflector model.
func (h *Harness) SwitchReflectorModel(modelID string) error {
	b, _ := json.Marshal(map[string]string{"modelId": modelID})
	return h.callJSVoid("switchReflectorModel", string(b))
}

// GetObserverModelID returns the current observer model ID.
func (h *Harness) GetObserverModelID() string {
	r, _ := h.callJSSimple("getObserverModelId")
	var s string
	json.Unmarshal([]byte(r), &s)
	return s
}

// GetReflectorModelID returns the current reflector model ID.
func (h *Harness) GetReflectorModelID() string {
	r, _ := h.callJSSimple("getReflectorModelId")
	var s string
	json.Unmarshal([]byte(r), &s)
	return s
}

// ---------------------------------------------------------------------------
// Subagents
// ---------------------------------------------------------------------------

// SetSubagentModelID sets the model for a subagent type.
func (h *Harness) SetSubagentModelID(modelID, agentType string) error {
	b, _ := json.Marshal(map[string]string{"modelId": modelID, "agentType": agentType})
	return h.callJSVoid("setSubagentModelId", string(b))
}

// GetSubagentModelID returns the model for a subagent type.
func (h *Harness) GetSubagentModelID(agentType string) string {
	b, _ := json.Marshal(map[string]string{"agentType": agentType})
	r, _ := h.callJS("getSubagentModelId", string(b))
	var s string
	json.Unmarshal([]byte(r), &s)
	return s
}

// ---------------------------------------------------------------------------
// Workspace
// ---------------------------------------------------------------------------

// HasWorkspace returns true if a workspace is configured.
func (h *Harness) HasWorkspace() bool {
	r, _ := h.callJSSimple("hasWorkspace")
	var b bool
	json.Unmarshal([]byte(r), &b)
	return b
}

// IsWorkspaceReady returns true if the workspace is initialized and ready.
func (h *Harness) IsWorkspaceReady() bool {
	r, _ := h.callJSSimple("isWorkspaceReady")
	var b bool
	json.Unmarshal([]byte(r), &b)
	return b
}

// DestroyWorkspace destroys the current workspace.
func (h *Harness) DestroyWorkspace() error {
	return h.callJSVoid("destroyWorkspace", "")
}

// ---------------------------------------------------------------------------
// Session & Resource
// ---------------------------------------------------------------------------

// GetSession returns the current session info.
func (h *Harness) GetSession() HarnessSession {
	r, _ := h.callJSSimple("getSession")
	var sess HarnessSession
	json.Unmarshal([]byte(r), &sess)
	return sess
}

// SetResourceID scopes threads to a specific resource.
func (h *Harness) SetResourceID(resourceID string) error {
	b, _ := json.Marshal(map[string]string{"resourceId": resourceID})
	return h.callJSVoid("setResourceId", string(b))
}

// GetResourceID returns the current resource ID.
func (h *Harness) GetResourceID() string {
	r, _ := h.callJSSimple("getResourceId")
	var s string
	json.Unmarshal([]byte(r), &s)
	return s
}

// GetKnownResourceIDs returns all known resource IDs.
func (h *Harness) GetKnownResourceIDs() ([]string, error) {
	r, err := h.callJSSimple("getKnownResourceIds")
	if err != nil {
		return nil, err
	}
	var ids []string
	json.Unmarshal([]byte(r), &ids)
	return ids, nil
}

// ---------------------------------------------------------------------------
// Internal
// ---------------------------------------------------------------------------

// buildJSConfig translates Go config to the JSON shape expected by createHarness().
func (h *Harness) buildJSConfig() map[string]any {
	modes := make([]map[string]any, len(h.config.Modes))
	for i, m := range h.config.Modes {
		modes[i] = map[string]any{
			"id":             m.ID,
			"name":           m.Name,
			"default":        m.Default,
			"defaultModelId": m.DefaultModelID,
			"color":          m.Color,
			"agentName":      m.AgentName,
		}
	}

	cfg := map[string]any{
		"id":    h.config.ID,
		"modes": modes,
	}

	if h.config.ResourceID != "" {
		cfg["resourceId"] = h.config.ResourceID
	}
	if h.config.StateSchema != nil {
		cfg["stateSchema"] = h.config.StateSchema
	}
	if h.config.InitialState != nil {
		cfg["initialState"] = h.config.InitialState
	}
	if len(h.config.Subagents) > 0 {
		subs := make([]map[string]any, len(h.config.Subagents))
		for i, s := range h.config.Subagents {
			subs[i] = map[string]any{
				"id":             s.ID,
				"allowedTools":   s.AllowedTools,
				"defaultModelId": s.DefaultModelID,
				"instructions":   s.Instructions,
			}
		}
		cfg["subagents"] = subs
	}
	if h.config.OMConfig != nil {
		cfg["omConfig"] = map[string]any{
			"defaultObserverModel":  h.config.OMConfig.DefaultObserverModel,
			"defaultReflectorModel": h.config.OMConfig.DefaultReflectorModel,
			"observationThreshold":  h.config.OMConfig.ObservationThreshold,
			"reflectionThreshold":   h.config.OMConfig.ReflectionThreshold,
		}
	}
	if len(h.config.Permissions) > 0 {
		// Convert typed map to string map for JS
		perms := make(map[string]string, len(h.config.Permissions))
		for cat, pol := range h.config.Permissions {
			perms[string(cat)] = string(pol)
		}
		cfg["defaultPermissions"] = perms
	}
	if len(h.config.ToolCategories) > 0 {
		// Convert typed map to string map for JS
		cats := make(map[string]string, len(h.config.ToolCategories))
		for tool, cat := range h.config.ToolCategories {
			cats[tool] = string(cat)
		}
		cfg["toolCategories"] = cats
	}

	return cfg
}

// startHeartbeats starts Go-side heartbeat timers.
func (h *Harness) startHeartbeats() {
	for _, hb := range h.config.HeartbeatHandlers {
		interval := time.Duration(hb.IntervalMs) * time.Millisecond
		if interval <= 0 {
			continue
		}
		ticker := time.NewTicker(interval)

		h.hbMu.Lock()
		h.heartbeats[hb.ID] = ticker
		h.hbMu.Unlock()

		handler := hb.Handler
		if hb.Immediate {
			go func() {
				defer func() { recover() }()
				handler()
			}()
		}

		go func(t *time.Ticker, fn func() error) {
			for range t.C {
				if h.closed {
					return
				}
				func() {
					defer func() { recover() }()
					fn()
				}()
			}
		}(ticker, handler)
	}
}

// stopHeartbeats stops all heartbeat timers and calls shutdown functions.
func (h *Harness) stopHeartbeats() {
	h.hbMu.Lock()
	defer h.hbMu.Unlock()

	for _, ticker := range h.heartbeats {
		ticker.Stop()
	}

	for _, hb := range h.config.HeartbeatHandlers {
		if hb.Shutdown != nil {
			func() {
				defer func() { recover() }()
				hb.Shutdown()
			}()
		}
	}

	h.heartbeats = make(map[string]*time.Ticker)
}
