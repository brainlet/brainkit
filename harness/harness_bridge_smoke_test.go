//go:build e2e

package harness

import (
	"sync"
	"testing"
	"time"
)

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
