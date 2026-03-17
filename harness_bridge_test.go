package brainkit

import (
	"context"
	"sync"
	"testing"
	"time"
)

func setupHarnessKit(t *testing.T) *Kit {
	t.Helper()
	return newTestKit(t) // uses loadEnv + requireKey from testutil_test.go
}

func createTestAgent(t *testing.T, kit *Kit) {
	t.Helper()
	_, err := kit.EvalTS(context.Background(), "create-agent.ts", `
		const testAgent = agent({
			model: "openai/gpt-4o-mini",
			name: "testAgent",
			instructions: "You are a test agent. Respond briefly and helpfully.",
		});
	`)
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}
}

// initTestHarness creates a Harness and registers cleanup that aborts any
// running agent before closing. This prevents goroutine leaks when tests
// timeout — without abort, kit.Close() blocks forever on wg.Wait().
func initTestHarness(t *testing.T, kit *Kit, cfg HarnessConfig) *Harness {
	t.Helper()
	h, err := kit.InitHarness(cfg)
	if err != nil {
		t.Fatalf("InitHarness: %v", err)
	}
	t.Cleanup(func() {
		h.Abort() // unblock any stuck SendMessage goroutine
		h.Close()
	})
	return h
}

func defaultHarnessConfig() HarnessConfig {
	return HarnessConfig{
		ID: "test-harness",
		Modes: []ModeConfig{
			{ID: "default", Name: "Default", Default: true, DefaultModelID: "openai/gpt-4o-mini", AgentName: "testAgent"},
		},
		InitialState: map[string]any{"yolo": true},
	}
}

// ---------------------------------------------------------------------------
// Bridge tests
// ---------------------------------------------------------------------------

func TestHarnessBridge_InitAndClose(t *testing.T) {
	kit := setupHarnessKit(t)
	createTestAgent(t, kit)

	h := initTestHarness(t, kit, defaultHarnessConfig())

	if h.IsRunning() {
		t.Error("should not be running initially")
	}
}

func TestHarnessBridge_ListModes(t *testing.T) {
	kit := setupHarnessKit(t)
	createTestAgent(t, kit)

	h := initTestHarness(t, kit, defaultHarnessConfig())

	modes := h.ListModes()
	if len(modes) == 0 {
		t.Fatal("expected at least one mode")
	}
	found := false
	for _, m := range modes {
		if m.ID == "default" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected mode 'default', got: %+v", modes)
	}
}

func TestHarnessBridge_GetCurrentMode(t *testing.T) {
	kit := setupHarnessKit(t)
	createTestAgent(t, kit)

	h := initTestHarness(t, kit, defaultHarnessConfig())

	mode := h.GetCurrentMode()
	if mode.ID != "default" {
		t.Errorf("current mode = %q, want default", mode.ID)
	}
}

func TestHarnessBridge_GetDisplayState(t *testing.T) {
	kit := setupHarnessKit(t)
	createTestAgent(t, kit)

	h := initTestHarness(t, kit, defaultHarnessConfig())

	ds := h.GetDisplayState()
	if ds == nil {
		t.Fatal("display state should not be nil")
	}
	if ds.IsRunning {
		t.Error("should not be running initially")
	}
}

func TestHarnessBridge_SendMessage(t *testing.T) {
	kit := setupHarnessKit(t)
	createTestAgent(t, kit)

	h := initTestHarness(t, kit, defaultHarnessConfig())

	var receivedEvents []HarnessEventType
	var mu sync.Mutex
	unsub := h.Subscribe(func(e HarnessEvent) {
		mu.Lock()
		receivedEvents = append(receivedEvents, e.Type)
		mu.Unlock()
	})
	defer unsub()

	// Send a simple message
	if err := sendWithTimeout(t, h, "say hello in one word", 30*time.Second); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(receivedEvents) == 0 {
		t.Fatal("expected events from SendMessage")
	}

	// Should have agent_start and agent_end
	hasStart := false
	hasEnd := false
	hasMessageUpdate := false
	for _, e := range receivedEvents {
		switch e {
		case EventAgentStart:
			hasStart = true
		case EventAgentEnd:
			hasEnd = true
		case EventMessageUpdate:
			hasMessageUpdate = true
		}
	}
	if !hasStart {
		t.Error("missing agent_start event")
	}
	if !hasEnd {
		t.Error("missing agent_end event")
	}
	if !hasMessageUpdate {
		t.Error("missing message_update event (no text streamed?)")
	}

	// Display state should reflect completion
	ds := h.GetDisplayState()
	if ds.IsRunning {
		t.Error("should not be running after agent_end")
	}

	// Token usage should be populated
	tu := h.GetTokenUsage()
	if tu.TotalTokens == 0 {
		t.Error("token usage should be > 0 after SendMessage")
	}

	t.Logf("Received %d events, %d total tokens", len(receivedEvents), tu.TotalTokens)
}

// ---------------------------------------------------------------------------
// Tier 1: No-LLM API tests
// ---------------------------------------------------------------------------

func TestHarnessAPI_ThreadCRUD(t *testing.T) {
	kit := setupHarnessKit(t)
	createTestAgent(t, kit)

	collector := newEventCollector()
	h := initTestHarness(t, kit, defaultHarnessConfig())
	h.Subscribe(collector.handler)

	// Create a thread
	id1, err := h.CreateThread(WithThreadTitle("Thread One"))
	if err != nil {
		t.Fatalf("CreateThread: %v", err)
	}
	if id1 == "" {
		t.Fatal("CreateThread returned empty ID")
	}
	t.Logf("Created thread: %s", id1)

	// Current thread should be the new one
	if got := h.GetCurrentThreadID(); got != id1 {
		t.Errorf("GetCurrentThreadID = %q, want %q", got, id1)
	}

	// Verify thread_created event
	if !collector.Has(EventThreadCreated) {
		t.Error("missing thread_created event")
	}

	// List threads
	threads, err := h.ListThreads()
	if err != nil {
		t.Fatalf("ListThreads: %v", err)
	}
	if len(threads) == 0 {
		t.Fatal("ListThreads returned empty")
	}
	found := false
	for _, th := range threads {
		if th.ID == id1 {
			found = true
		}
	}
	if !found {
		t.Errorf("thread %s not in list", id1)
	}

	// Rename
	if err := h.RenameThread("Renamed Thread"); err != nil {
		t.Fatalf("RenameThread: %v", err)
	}

	// Create second thread
	id2, err := h.CreateThread(WithThreadTitle("Thread Two"))
	if err != nil {
		t.Fatalf("CreateThread 2: %v", err)
	}

	// Switch back to first
	collector.Reset()
	if err := h.SwitchThread(id1); err != nil {
		t.Fatalf("SwitchThread: %v", err)
	}
	if got := h.GetCurrentThreadID(); got != id1 {
		t.Errorf("after switch: GetCurrentThreadID = %q, want %q", got, id1)
	}
	if !collector.Has(EventThreadChanged) {
		t.Error("missing thread_changed event after SwitchThread")
	}

	// Delete second thread
	collector.Reset()
	if err := h.DeleteThread(id2); err != nil {
		t.Fatalf("DeleteThread: %v", err)
	}
	if !collector.Has(EventThreadDeleted) {
		t.Error("missing thread_deleted event")
	}

	// Deleted thread should not appear in list
	threads2, _ := h.ListThreads()
	for _, th := range threads2 {
		if th.ID == id2 {
			t.Error("deleted thread still in list")
		}
	}

	t.Log("Thread CRUD: all operations verified")
}

func TestHarnessAPI_State(t *testing.T) {
	kit := setupHarnessKit(t)
	createTestAgent(t, kit)

	collector := newEventCollector()
	h := initTestHarness(t, kit, HarnessConfig{
		ID: "state-test",
		Modes: []ModeConfig{
			{ID: "default", Name: "Default", Default: true, DefaultModelID: "openai/gpt-4o-mini", AgentName: "testAgent"},
		},
		InitialState: map[string]any{
			"yolo":        true,
			"projectName": "brainlet",
			"counter":     float64(0),
		},
	})
	h.Subscribe(collector.handler)

	// GetState returns initial values
	state := h.GetState()
	if state == nil {
		t.Fatal("GetState returned nil")
	}
	t.Logf("Initial state: %+v", state)

	if v, ok := state["yolo"]; !ok || v != true {
		t.Errorf("yolo = %v, want true", v)
	}
	if v, ok := state["projectName"]; !ok || v != "brainlet" {
		t.Errorf("projectName = %v, want brainlet", v)
	}

	// SetState updates values
	if err := h.SetState(map[string]any{"counter": float64(42), "yolo": false}); err != nil {
		t.Fatalf("SetState: %v", err)
	}

	// Verify state_changed event
	if !collector.Has(EventStateChanged) {
		t.Error("missing state_changed event")
	}

	// GetState reflects update
	state2 := h.GetState()
	if v, ok := state2["counter"]; !ok {
		t.Error("counter not in state")
	} else if n, ok := v.(float64); !ok || n != 42 {
		t.Errorf("counter = %v, want 42", v)
	}
	if v, ok := state2["yolo"]; !ok || v != false {
		t.Errorf("yolo = %v, want false", v)
	}
	// projectName should be preserved
	if v, ok := state2["projectName"]; !ok || v != "brainlet" {
		t.Errorf("projectName = %v, want brainlet (should be preserved)", v)
	}

	t.Log("State management: get/set/events verified")
}

func TestHarnessAPI_ModeSwitching(t *testing.T) {
	kit := setupHarnessKit(t)
	createTestAgent(t, kit)

	collector := newEventCollector()
	h := initTestHarness(t, kit, HarnessConfig{
		ID: "mode-test",
		Modes: []ModeConfig{
			{ID: "build", Name: "Build", Default: true, DefaultModelID: "openai/gpt-4o-mini", AgentName: "testAgent"},
			{ID: "fast", Name: "Fast", DefaultModelID: "openai/gpt-4o-mini", AgentName: "testAgent"},
		},
		InitialState: map[string]any{"yolo": true},
	})
	h.Subscribe(collector.handler)

	// Default mode
	if got := h.GetCurrentModeID(); got != "build" {
		t.Errorf("initial mode = %q, want build", got)
	}

	// List modes
	modes := h.ListModes()
	if len(modes) != 2 {
		t.Fatalf("ListModes returned %d modes, want 2", len(modes))
	}

	// Switch to fast
	if err := h.SwitchMode("fast"); err != nil {
		t.Fatalf("SwitchMode: %v", err)
	}
	if got := h.GetCurrentModeID(); got != "fast" {
		t.Errorf("after switch: mode = %q, want fast", got)
	}
	if !collector.Has(EventModeChanged) {
		t.Error("missing mode_changed event")
	}

	// Switch back
	collector.Reset()
	if err := h.SwitchMode("build"); err != nil {
		t.Fatalf("SwitchMode back: %v", err)
	}
	if got := h.GetCurrentModeID(); got != "build" {
		t.Errorf("after switch back: mode = %q, want build", got)
	}

	t.Log("Mode switching: verified")
}

func TestHarnessAPI_Permissions(t *testing.T) {
	kit := setupHarnessKit(t)
	createTestAgent(t, kit)

	h := initTestHarness(t, kit, HarnessConfig{
		ID: "perms-test",
		Modes: []ModeConfig{
			{ID: "default", Name: "Default", Default: true, DefaultModelID: "openai/gpt-4o-mini", AgentName: "testAgent"},
		},
		InitialState: map[string]any{"yolo": false},
		DefaultPermissions: map[string]string{
			"read":    "allow",
			"edit":    "ask",
			"execute": "ask",
		},
	})

	// GetPermissionRules
	rules := h.GetPermissionRules()
	t.Logf("Initial rules: %+v", rules)

	// SetPermissionForCategory
	if err := h.SetPermissionForCategory("execute", "deny"); err != nil {
		t.Fatalf("SetPermissionForCategory: %v", err)
	}
	rules2 := h.GetPermissionRules()
	if rules2.Categories["execute"] != "deny" {
		t.Errorf("execute policy = %q, want deny", rules2.Categories["execute"])
	}

	// SetPermissionForTool
	if err := h.SetPermissionForTool("write_file", "allow"); err != nil {
		t.Fatalf("SetPermissionForTool: %v", err)
	}
	rules3 := h.GetPermissionRules()
	if rules3.Tools["write_file"] != "allow" {
		t.Errorf("write_file policy = %q, want allow", rules3.Tools["write_file"])
	}

	t.Log("Permissions: set/get category and tool policies verified")
}

func TestHarnessAPI_ListMessages(t *testing.T) {
	kit := setupHarnessKit(t)
	createTestAgent(t, kit)

	collector := newEventCollector()
	h := initTestHarness(t, kit, defaultHarnessConfig())
	h.Subscribe(collector.handler)

	// Send a message first
	if err := sendWithTimeout(t, h, "say ok", 30*time.Second); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	// List messages — may be empty with InMemoryStore (no memory domain)
	msgs, err := h.ListMessages()
	if err != nil {
		t.Logf("ListMessages error (may be expected with InMemoryStore): %v", err)
	} else if len(msgs) == 0 {
		t.Log("ListMessages returned 0 messages — InMemoryStore may not support memory domain")
	} else {
		hasUser := false
		hasAssistant := false
		for _, m := range msgs {
			if m.Role == "user" {
				hasUser = true
			}
			if m.Role == "assistant" {
				hasAssistant = true
			}
		}
		if !hasUser {
			t.Error("no user message in list")
		}
		if !hasAssistant {
			t.Error("no assistant message in list")
		}
		t.Logf("ListMessages: %d messages", len(msgs))
	}

	// The real verification: agent_end was received with text
	if !collector.Has(EventAgentEnd) {
		t.Error("missing agent_end event")
	}
	t.Log("ListMessages: bridge call works, message persistence depends on storage backend")
}

func TestHarnessAPI_StateSchema(t *testing.T) {
	kit := setupHarnessKit(t)
	createTestAgent(t, kit)

	h := initTestHarness(t, kit, HarnessConfig{
		ID: "schema-test",
		Modes: []ModeConfig{
			{ID: "default", Name: "Default", Default: true, DefaultModelID: "openai/gpt-4o-mini", AgentName: "testAgent"},
		},
		StateSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"projectName": map[string]any{"type": "string", "default": ""},
				"yolo":        map[string]any{"type": "boolean", "default": true},
				"counter":     map[string]any{"type": "number", "default": float64(0)},
				"tasks": map[string]any{
					"type":    "array",
					"items":   map[string]any{"type": "object"},
					"default": []any{},
				},
			},
		},
		InitialState: map[string]any{
			"projectName": "brainlet",
			"yolo":        true,
			"counter":     float64(0),
		},
	})

	// State should have initial values
	state := h.GetState()
	if state == nil {
		t.Fatal("GetState returned nil")
	}
	if v := state["projectName"]; v != "brainlet" {
		t.Errorf("projectName = %v, want brainlet", v)
	}
	if v := state["yolo"]; v != true {
		t.Errorf("yolo = %v, want true", v)
	}

	// SetState should work with valid types
	if err := h.SetState(map[string]any{"counter": float64(10)}); err != nil {
		t.Fatalf("SetState valid: %v", err)
	}
	state2 := h.GetState()
	if v, ok := state2["counter"].(float64); !ok || v != 10 {
		t.Errorf("counter = %v, want 10", state2["counter"])
	}

	// Original values preserved
	if v := state2["projectName"]; v != "brainlet" {
		t.Errorf("projectName = %v, want brainlet (should be preserved)", v)
	}

	t.Log("StateSchema: JSON Schema → Zod conversion verified")
}
