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

	h, err := kit.InitHarness(defaultHarnessConfig())
	if err != nil {
		t.Fatalf("InitHarness: %v", err)
	}
	defer h.Close()

	if h.IsRunning() {
		t.Error("should not be running initially")
	}
}

func TestHarnessBridge_ListModes(t *testing.T) {
	kit := setupHarnessKit(t)
	createTestAgent(t, kit)

	h, err := kit.InitHarness(defaultHarnessConfig())
	if err != nil {
		t.Fatalf("InitHarness: %v", err)
	}
	defer h.Close()

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

	h, err := kit.InitHarness(defaultHarnessConfig())
	if err != nil {
		t.Fatalf("InitHarness: %v", err)
	}
	defer h.Close()

	mode := h.GetCurrentMode()
	if mode.ID != "default" {
		t.Errorf("current mode = %q, want default", mode.ID)
	}
}

func TestHarnessBridge_GetDisplayState(t *testing.T) {
	kit := setupHarnessKit(t)
	createTestAgent(t, kit)

	h, err := kit.InitHarness(defaultHarnessConfig())
	if err != nil {
		t.Fatalf("InitHarness: %v", err)
	}
	defer h.Close()

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

	h, err := kit.InitHarness(defaultHarnessConfig())
	if err != nil {
		t.Fatalf("InitHarness: %v", err)
	}
	defer h.Close()

	var receivedEvents []HarnessEventType
	var mu sync.Mutex
	unsub := h.Subscribe(func(e HarnessEvent) {
		mu.Lock()
		receivedEvents = append(receivedEvents, e.Type)
		mu.Unlock()
	})
	defer unsub()

	// Send a simple message
	done := make(chan error, 1)
	go func() {
		done <- h.SendMessage("say hello in one word")
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("SendMessage: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("SendMessage timed out")
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
