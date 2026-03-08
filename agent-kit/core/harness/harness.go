// Ported from: packages/core/src/harness/harness.ts
package harness

import (
	"fmt"
	"sync"
	"time"
)

// Harness orchestrates multiple agent modes, shared state, memory, and storage.
// It is the core abstraction that a TUI (or other UI) controls.
type Harness struct {
	// ID is the unique identifier for this harness instance.
	ID string

	config             HarnessConfig
	state              map[string]any
	currentModeID      string
	currentThreadID    string
	resourceID         string
	defaultResourceID  string
	listeners          []HarnessEventListener
	abortRequested     bool
	currentRunID       string
	currentOperationID int
	followUpQueue      []followUpEntry
	tokenUsage         TokenUsage
	displayState       HarnessDisplayState
	workspace          *Workspace

	// Permission tracking
	sessionGrantedCategories map[string]bool
	sessionGrantedTools      map[string]bool
	permissionRules          PermissionRules

	// Pending interactive prompts
	pendingQuestions      map[string]func(string)
	pendingPlanApprovals  map[string]func(string, string)
	pendingApprovalResolve func(decision string)
	pendingApprovalToolName string

	// Heartbeat tracking
	heartbeatTimers map[string]*heartbeatEntry

	mu sync.RWMutex
}

type followUpEntry struct {
	Content        string
	RequestContext RequestContext
}

type heartbeatEntry struct {
	Timer    *time.Ticker
	Shutdown func() error
	StopCh   chan struct{}
}

// New creates a new Harness with the given configuration.
func New(config HarnessConfig) (*Harness, error) {
	if len(config.Modes) == 0 {
		return nil, fmt.Errorf("harness requires at least one agent mode")
	}

	// Find default mode
	defaultMode := config.Modes[0]
	for _, m := range config.Modes {
		if m.Default {
			defaultMode = m
			break
		}
	}

	resourceID := config.ResourceID
	if resourceID == "" {
		resourceID = config.ID
	}

	initialState := make(map[string]any)
	for k, v := range config.InitialState {
		initialState[k] = v
	}

	h := &Harness{
		ID:                       config.ID,
		config:                   config,
		state:                    initialState,
		currentModeID:            defaultMode.ID,
		resourceID:               resourceID,
		defaultResourceID:        resourceID,
		displayState:             DefaultDisplayState(),
		sessionGrantedCategories: make(map[string]bool),
		sessionGrantedTools:      make(map[string]bool),
		permissionRules: PermissionRules{
			Categories: make(map[ToolCategory]PermissionPolicy),
			Tools:      make(map[string]PermissionPolicy),
		},
		pendingQuestions:     make(map[string]func(string)),
		pendingPlanApprovals: make(map[string]func(string, string)),
		heartbeatTimers:      make(map[string]*heartbeatEntry),
		tokenUsage:           TokenUsage{},
	}

	// Seed model from mode default if not set
	if _, ok := h.state["currentModelId"]; !ok && defaultMode.DefaultModelID != "" {
		h.state["currentModelId"] = defaultMode.DefaultModelID
	}

	return h, nil
}

// =============================================================================
// Initialization
// =============================================================================

// Init initializes the harness - loads storage and workspace.
// Must be called before using the harness.
func (h *Harness) Init() error {
	// TODO: initialize internal Mastra instance for storage
	// TODO: initialize workspace if configured
	// TODO: propagate harness-level Mastra, memory, and workspace to mode agents

	h.startHeartbeats()
	return nil
}

// SelectOrCreateThread selects the most recent thread, or creates one if none exist.
func (h *Harness) SelectOrCreateThread() (*HarnessThread, error) {
	threads, err := h.ListThreads(nil)
	if err != nil {
		return nil, err
	}

	if len(threads) == 0 {
		return h.CreateThread("")
	}

	// Sort by updatedAt descending - pick most recent
	mostRecent := threads[0]
	for _, t := range threads[1:] {
		if t.UpdatedAt.After(mostRecent.UpdatedAt) {
			mostRecent = t
		}
	}

	if h.config.ThreadLock != nil {
		if err := h.config.ThreadLock.Acquire(mostRecent.ID); err != nil {
			return nil, fmt.Errorf("failed to acquire thread lock: %w", err)
		}
	}

	h.mu.Lock()
	h.currentThreadID = mostRecent.ID
	h.mu.Unlock()

	// TODO: loadThreadMetadata()

	return &mostRecent, nil
}

// =============================================================================
// State Management
// =============================================================================

// GetState returns the current harness state (read-only snapshot).
func (h *Harness) GetState() map[string]any {
	h.mu.RLock()
	defer h.mu.RUnlock()

	snapshot := make(map[string]any, len(h.state))
	for k, v := range h.state {
		snapshot[k] = v
	}
	return snapshot
}

// SetState updates harness state. Emits state_changed event.
func (h *Harness) SetState(updates map[string]any) error {
	h.mu.Lock()
	changedKeys := make([]string, 0, len(updates))
	for k, v := range updates {
		h.state[k] = v
		changedKeys = append(changedKeys, k)
	}
	stateCopy := make(map[string]any, len(h.state))
	for k, v := range h.state {
		stateCopy[k] = v
	}
	h.mu.Unlock()

	h.emit(HarnessEvent{
		Type:        "state_changed",
		State:       stateCopy,
		ChangedKeys: changedKeys,
	})
	return nil
}

// =============================================================================
// Mode Management
// =============================================================================

// ListModes returns all configured modes.
func (h *Harness) ListModes() []HarnessMode {
	return h.config.Modes
}

// GetCurrentModeID returns the current mode ID.
func (h *Harness) GetCurrentModeID() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.currentModeID
}

// GetCurrentMode returns the current mode configuration.
func (h *Harness) GetCurrentMode() (*HarnessMode, error) {
	h.mu.RLock()
	modeID := h.currentModeID
	h.mu.RUnlock()

	for i := range h.config.Modes {
		if h.config.Modes[i].ID == modeID {
			return &h.config.Modes[i], nil
		}
	}
	return nil, fmt.Errorf("mode not found: %s", modeID)
}

// SwitchMode switches to a different mode. Aborts any in-progress generation.
func (h *Harness) SwitchMode(modeID string) error {
	var found bool
	for _, m := range h.config.Modes {
		if m.ID == modeID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("mode not found: %s", modeID)
	}

	h.Abort()

	h.mu.Lock()
	previousModeID := h.currentModeID
	h.currentModeID = modeID
	h.mu.Unlock()

	// TODO: save current model to outgoing mode, load incoming mode's model

	h.emit(HarnessEvent{
		Type:           "mode_changed",
		ModeID:         modeID,
		PreviousModeID: previousModeID,
	})
	return nil
}

// GetModelName returns a short display name from the current model ID.
func (h *Harness) GetModelName() string {
	modelID := h.GetCurrentModelID()
	if modelID == "" || modelID == "unknown" {
		if modelID == "" {
			return "unknown"
		}
		return modelID
	}
	// Return the last segment after "/"
	for i := len(modelID) - 1; i >= 0; i-- {
		if modelID[i] == '/' {
			return modelID[i+1:]
		}
	}
	return modelID
}

// GetFullModelID returns the full model ID (e.g., "anthropic/claude-sonnet-4").
func (h *Harness) GetFullModelID() string {
	return h.GetCurrentModelID()
}

// SwitchModel switches to a different model at runtime.
func (h *Harness) SwitchModel(modelID, scope, modeID string) error {
	if scope == "" {
		scope = "thread"
	}
	if modeID == "" {
		modeID = h.GetCurrentModeID()
	}

	if modeID == h.GetCurrentModeID() {
		_ = h.SetState(map[string]any{"currentModelId": modelID})
	}

	// TODO: persist to thread metadata if scope == "thread"

	if h.config.ModelUseCountTracker != nil {
		h.config.ModelUseCountTracker(modelID)
	}

	h.emit(HarnessEvent{
		Type:    "model_changed",
		ModelID: modelID,
		Scope:   scope,
		ModeID:  modeID,
	})
	return nil
}

// GetCurrentModelID returns the current model ID from state.
func (h *Harness) GetCurrentModelID() string {
	state := h.GetState()
	if v, ok := state["currentModelId"].(string); ok {
		return v
	}
	return ""
}

// HasModelSelected returns whether a model has been selected.
func (h *Harness) HasModelSelected() bool {
	return h.GetCurrentModelID() != ""
}

// =============================================================================
// Thread Management
// =============================================================================

// GetCurrentThreadID returns the current thread ID, or empty string if none.
func (h *Harness) GetCurrentThreadID() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.currentThreadID
}

// GetResourceID returns the current resource ID.
func (h *Harness) GetResourceID() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.resourceID
}

// SetResourceID sets the resource ID and clears the current thread.
func (h *Harness) SetResourceID(resourceID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.resourceID = resourceID
	h.currentThreadID = ""
}

// GetDefaultResourceID returns the default resource ID.
func (h *Harness) GetDefaultResourceID() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.defaultResourceID
}

// CreateThread creates a new thread with an optional title.
func (h *Harness) CreateThread(title string) (*HarnessThread, error) {
	now := time.Now()
	threadID := h.generateID()

	thread := HarnessThread{
		ID:         threadID,
		ResourceID: h.GetResourceID(),
		Title:      title,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// TODO: acquire thread lock, persist to storage

	h.mu.Lock()
	h.currentThreadID = thread.ID
	h.tokenUsage = TokenUsage{}
	h.mu.Unlock()

	h.emit(HarnessEvent{
		Type:   "thread_created",
		Thread: &thread,
	})
	return &thread, nil
}

// SwitchThread switches to a different thread.
func (h *Harness) SwitchThread(threadID string) error {
	h.Abort()

	// TODO: acquire/release thread locks, verify thread exists in storage

	h.mu.Lock()
	previousThreadID := h.currentThreadID
	h.currentThreadID = threadID
	h.mu.Unlock()

	// TODO: loadThreadMetadata()

	h.emit(HarnessEvent{
		Type:             "thread_changed",
		ThreadID:         threadID,
		PreviousThreadID: previousThreadID,
	})
	return nil
}

// ListThreads returns all threads for the current resource (or all resources if allResources is true).
func (h *Harness) ListThreads(options *ListThreadsOptions) ([]HarnessThread, error) {
	// TODO: query storage for threads
	return nil, nil
}

// ListThreadsOptions holds options for ListThreads.
type ListThreadsOptions struct {
	AllResources bool
}

// RenameThread renames the current thread.
func (h *Harness) RenameThread(title string) error {
	// TODO: persist to storage
	return nil
}

// DeleteThread deletes a thread by ID.
func (h *Harness) DeleteThread(threadID string) error {
	// TODO: delete from storage, release lock if current thread

	h.mu.Lock()
	isDeletingCurrent := h.currentThreadID == threadID
	if isDeletingCurrent {
		h.currentThreadID = ""
		h.tokenUsage = TokenUsage{}
	}
	h.mu.Unlock()

	h.emit(HarnessEvent{
		Type:     "thread_deleted",
		ThreadID: threadID,
	})
	return nil
}

// =============================================================================
// Observational Memory
// =============================================================================

// GetObserverModelID returns the observer model ID from state.
func (h *Harness) GetObserverModelID() string {
	state := h.GetState()
	if v, ok := state["observerModelId"].(string); ok {
		return v
	}
	if h.config.OMConfig != nil {
		return h.config.OMConfig.DefaultObserverModelID
	}
	return ""
}

// GetReflectorModelID returns the reflector model ID from state.
func (h *Harness) GetReflectorModelID() string {
	state := h.GetState()
	if v, ok := state["reflectorModelId"].(string); ok {
		return v
	}
	if h.config.OMConfig != nil {
		return h.config.OMConfig.DefaultReflectorModelID
	}
	return ""
}

// SwitchObserverModel changes the observer model ID.
func (h *Harness) SwitchObserverModel(modelID string) error {
	_ = h.SetState(map[string]any{"observerModelId": modelID})
	// TODO: persist to thread metadata
	h.emit(HarnessEvent{
		Type:    "om_model_changed",
		ModelID: modelID,
		// Role is "observer" - stored in a generic field
	})
	return nil
}

// SwitchReflectorModel changes the reflector model ID.
func (h *Harness) SwitchReflectorModel(modelID string) error {
	_ = h.SetState(map[string]any{"reflectorModelId": modelID})
	// TODO: persist to thread metadata
	h.emit(HarnessEvent{
		Type:    "om_model_changed",
		ModelID: modelID,
		// Role is "reflector"
	})
	return nil
}

// =============================================================================
// Subagent Model Management
// =============================================================================

// GetSubagentModelID returns the configured subagent model ID.
func (h *Harness) GetSubagentModelID(agentType string) string {
	state := h.GetState()
	if agentType != "" {
		key := fmt.Sprintf("subagentModelId_%s", agentType)
		if v, ok := state[key].(string); ok {
			return v
		}
	}
	if v, ok := state["subagentModelId"].(string); ok {
		return v
	}
	return ""
}

// SetSubagentModelID sets the subagent model ID.
func (h *Harness) SetSubagentModelID(modelID, agentType string) error {
	key := "subagentModelId"
	if agentType != "" {
		key = fmt.Sprintf("subagentModelId_%s", agentType)
	}
	_ = h.SetState(map[string]any{key: modelID})
	// TODO: persist to thread metadata

	scope := "global"
	if h.currentThreadID != "" {
		scope = "thread"
	}
	h.emit(HarnessEvent{
		Type:      "subagent_model_changed",
		ModelID:   modelID,
		Scope:     scope,
		AgentType: agentType,
	})
	return nil
}

// =============================================================================
// Permissions
// =============================================================================

// GrantSessionCategory grants permission for a tool category for this session.
func (h *Harness) GrantSessionCategory(category ToolCategory) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sessionGrantedCategories[string(category)] = true
}

// GrantSessionTool grants permission for a specific tool for this session.
func (h *Harness) GrantSessionTool(toolName string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sessionGrantedTools[toolName] = true
}

// GetSessionGrants returns the current session grants.
func (h *Harness) GetSessionGrants() (categories []ToolCategory, tools []string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for k := range h.sessionGrantedCategories {
		categories = append(categories, ToolCategory(k))
	}
	for k := range h.sessionGrantedTools {
		tools = append(tools, k)
	}
	return
}

// GetToolCategory returns the tool category for a given tool name.
func (h *Harness) GetToolCategory(toolName string) *ToolCategory {
	if h.config.ToolCategoryResolver != nil {
		return h.config.ToolCategoryResolver(toolName)
	}
	return nil
}

// SetPermissionForCategory sets the permission policy for a tool category.
func (h *Harness) SetPermissionForCategory(category ToolCategory, policy PermissionPolicy) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.permissionRules.Categories[category] = policy
}

// SetPermissionForTool sets the permission policy for a specific tool.
func (h *Harness) SetPermissionForTool(toolName string, policy PermissionPolicy) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.permissionRules.Tools[toolName] = policy
}

// GetPermissionRules returns the current permission rules.
func (h *Harness) GetPermissionRules() PermissionRules {
	h.mu.RLock()
	defer h.mu.RUnlock()
	// Return a copy
	cats := make(map[ToolCategory]PermissionPolicy)
	for k, v := range h.permissionRules.Categories {
		cats[k] = v
	}
	tools := make(map[string]PermissionPolicy)
	for k, v := range h.permissionRules.Tools {
		tools[k] = v
	}
	return PermissionRules{Categories: cats, Tools: tools}
}

// =============================================================================
// Messaging
// =============================================================================

// SendMessage sends a message to the current agent mode.
// TODO: implement full message flow (agent stream, tool execution, memory persistence).
func (h *Harness) SendMessage(content string) error {
	if h.currentThreadID == "" {
		_, err := h.CreateThread("")
		if err != nil {
			return fmt.Errorf("failed to create thread: %w", err)
		}
	}

	h.mu.Lock()
	h.abortRequested = false
	h.currentOperationID++
	h.mu.Unlock()

	h.emit(HarnessEvent{Type: "agent_start"})

	// TODO: resolve agent, build request context, stream agent response,
	// handle tool approvals, persist messages, emit events

	h.emit(HarnessEvent{Type: "agent_end", Reason: "complete"})
	return nil
}

// =============================================================================
// Control
// =============================================================================

// Abort aborts the current agent operation.
func (h *Harness) Abort() {
	h.mu.Lock()
	h.abortRequested = true
	h.mu.Unlock()
	// TODO: abort via context cancellation
}

// Steer interrupts the current generation to steer the agent's behavior.
func (h *Harness) Steer(content string) error {
	h.Abort()
	return h.SendMessage(content)
}

// FollowUp queues a follow-up message to be sent after the current generation completes.
func (h *Harness) FollowUp(content string) error {
	h.mu.Lock()
	h.followUpQueue = append(h.followUpQueue, followUpEntry{Content: content})
	count := len(h.followUpQueue)
	h.mu.Unlock()

	h.emit(HarnessEvent{Type: "follow_up_queued", Count: count})
	return nil
}

// GetFollowUpCount returns the number of pending follow-up messages.
func (h *Harness) GetFollowUpCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.followUpQueue)
}

// IsRunning returns whether an agent operation is currently in progress.
func (h *Harness) IsRunning() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.displayState.IsRunning
}

// GetCurrentRunID returns the current run ID or empty string.
func (h *Harness) GetCurrentRunID() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.currentRunID
}

// GetDisplayState returns the current display state (read-only snapshot).
func (h *Harness) GetDisplayState() HarnessDisplayState {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.displayState
}

// =============================================================================
// Tool Approval
// =============================================================================

// RespondToToolApproval responds to a pending tool approval request.
func (h *Harness) RespondToToolApproval(decision string) {
	h.mu.Lock()
	resolve := h.pendingApprovalResolve
	h.pendingApprovalResolve = nil
	h.pendingApprovalToolName = ""
	h.mu.Unlock()

	if resolve != nil {
		resolve(decision)
	}
}

// =============================================================================
// Question/Plan Approval
// =============================================================================

// RegisterQuestion registers a pending question resolver.
func (h *Harness) RegisterQuestion(questionID string, resolve func(string)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pendingQuestions[questionID] = resolve
}

// RespondToQuestion provides an answer to a pending question.
func (h *Harness) RespondToQuestion(questionID, answer string) {
	h.mu.Lock()
	resolve, ok := h.pendingQuestions[questionID]
	if ok {
		delete(h.pendingQuestions, questionID)
	}
	h.mu.Unlock()

	if ok && resolve != nil {
		resolve(answer)
	}
}

// RegisterPlanApproval registers a pending plan approval resolver.
func (h *Harness) RegisterPlanApproval(planID string, resolve func(action, feedback string)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pendingPlanApprovals[planID] = resolve
}

// RespondToPlanApproval responds to a pending plan approval request.
func (h *Harness) RespondToPlanApproval(planID, action, feedback string) {
	h.mu.Lock()
	resolve, ok := h.pendingPlanApprovals[planID]
	if ok {
		delete(h.pendingPlanApprovals, planID)
	}
	h.mu.Unlock()

	if ok && resolve != nil {
		resolve(action, feedback)
	}

	if action == "approved" {
		h.emit(HarnessEvent{Type: "plan_approved"})
		// Switch to default mode on approval
		defaultMode := h.config.Modes[0]
		for _, m := range h.config.Modes {
			if m.Default {
				defaultMode = m
				break
			}
		}
		_ = h.SwitchMode(defaultMode.ID)
	}
}

// =============================================================================
// Events
// =============================================================================

// Subscribe registers a listener for harness events. Returns an unsubscribe function.
func (h *Harness) Subscribe(listener HarnessEventListener) func() {
	h.mu.Lock()
	h.listeners = append(h.listeners, listener)
	index := len(h.listeners) - 1
	h.mu.Unlock()

	return func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if index < len(h.listeners) {
			h.listeners = append(h.listeners[:index], h.listeners[index+1:]...)
		}
	}
}

func (h *Harness) emit(event HarnessEvent) {
	h.mu.RLock()
	listeners := make([]HarnessEventListener, len(h.listeners))
	copy(listeners, h.listeners)
	h.mu.RUnlock()

	for _, listener := range listeners {
		listener(event)
	}
}

// =============================================================================
// Heartbeats
// =============================================================================

func (h *Harness) startHeartbeats() {
	for _, handler := range h.config.HeartbeatHandlers {
		if _, exists := h.heartbeatTimers[handler.ID]; exists {
			continue
		}

		intervalMs := handler.IntervalMs
		if intervalMs <= 0 {
			intervalMs = 1000
		}

		ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
		stopCh := make(chan struct{})

		entry := &heartbeatEntry{
			Timer:    ticker,
			Shutdown: handler.Shutdown,
			StopCh:   stopCh,
		}
		h.heartbeatTimers[handler.ID] = entry

		immediate := handler.Immediate == nil || *handler.Immediate
		handlerFn := handler.Handler

		go func() {
			if immediate && handlerFn != nil {
				_ = handlerFn()
			}
			for {
				select {
				case <-ticker.C:
					if handlerFn != nil {
						_ = handlerFn()
					}
				case <-stopCh:
					return
				}
			}
		}()
	}
}

// StopHeartbeats stops all heartbeat handlers.
func (h *Harness) StopHeartbeats() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for id, entry := range h.heartbeatTimers {
		entry.Timer.Stop()
		close(entry.StopCh)
		if entry.Shutdown != nil {
			_ = entry.Shutdown()
		}
		delete(h.heartbeatTimers, id)
	}
}

// =============================================================================
// Token Usage
// =============================================================================

// GetTokenUsage returns the cumulative token usage for the current thread.
func (h *Harness) GetTokenUsage() TokenUsage {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.tokenUsage
}

// =============================================================================
// Helpers
// =============================================================================

func (h *Harness) generateID() string {
	if h.config.IDGenerator != nil {
		return h.config.IDGenerator()
	}
	return fmt.Sprintf("%d-%d", time.Now().UnixMilli(), time.Now().UnixNano()%10000)
}
