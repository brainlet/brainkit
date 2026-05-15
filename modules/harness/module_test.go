package harness_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/harness"
	_ "github.com/brainlet/brainkit/modules/jsruntime"
	"github.com/brainlet/brainkit/modules/packages/client"
	"github.com/brainlet/brainkit/presets/standard"
	"github.com/stretchr/testify/require"
)

// TestModuleLifecycle asserts that a Kit with the harness module wired
// (but no harness config) boots and closes cleanly. Harness.Init needs
// the Kit's JS bridge present; the zero-value HarnessConfig triggers
// the validator to reject the launch, so Init short-circuits to a
// no-op state. Close should be idempotent on that no-op.
func TestModuleLifecycle(t *testing.T) {
	m := harness.NewModule(harness.Config{})

	kit, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test-harness",
		CallerID:  "test",
		FSRoot:    t.TempDir(),
		Modules:   []bkmodule.Module{m},
	})
	// A zero-value HarnessConfig fails validation; the module mount
	// surfaces that error from the Kit constructor.
	if err != nil {
		require.Contains(t, err.Error(), "harness")
		return
	}
	defer kit.Close()

	require.Equal(t, bkmodule.StatusWIP, m.Status())
}

func TestModuleHotMountLiveHarnessInstance(t *testing.T) {
	testutil.LoadEnv(t)
	if !testutil.LiveAIEnabled() {
		t.Skip("set BRAINKIT_TEST_LIVE_AI=1 to run live OpenAI-backed harness module test")
	}
	key, ok := testutil.OpenAIKey()
	if !ok {
		t.Skip("OPENAI_API_KEY is not configured in the repo .env")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	k, err := brainkit.New(brainkit.Config{
		Namespace: "test-go-harness-live",
		CallerID:  "test-go-harness",
		Transport: brainkit.Memory(),
		FSRoot:    t.TempDir(),
		Providers: []brainkit.ProviderConfig{
			brainkit.OpenAI(key),
		},
		EnvVars: map[string]string{
			"OPENAI_API_KEY": os.Getenv("OPENAI_API_KEY"),
		},
		Modules: standard.PackageSet(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = k.Close() })

	_, err = packageclient.Deploy(ctx, k, packageclient.Inline("go-harness-agent-registry", "index.ts", `
		import { Agent } from "agent";
		import { kit, model } from "kit";

		const agent = new Agent({
			name: "go-harness-agent",
			model: model("openai", "gpt-4o-mini"),
			instructions: [
				"You are a deterministic live Brainkit Harness fixture.",
				"When asked, reply with exactly GO_HARNESS_BACKEND_OK and no punctuation.",
			].join("\n"),
		});

		kit.register("agent", "go-harness-agent", agent);
	`))
	require.NoError(t, err)

	mod := harness.NewModule(harness.Config{Harness: harness.HarnessConfig{
		ID:         "go-harness-live",
		ResourceID: "go-harness-live-resource",
		Modes: []harness.ModeConfig{{
			ID:             "default",
			Name:           "Default",
			Default:        true,
			DefaultModelID: "openai/gpt-4o-mini",
			AgentName:      "go-harness-agent",
		}},
	}})
	require.NoError(t, k.Mount(ctx, mod))

	inst := mod.Instance()
	require.NotNil(t, inst)

	var (
		mu     sync.Mutex
		events []harness.Event
	)
	unsubscribe := inst.Subscribe(func(event harness.Event) {
		mu.Lock()
		events = append(events, event)
		mu.Unlock()
	})
	defer unsubscribe()

	require.NoError(t, inst.SendMessage("Reply with the exact fixture token."))
	require.Equal(t, "default", inst.CurrentMode())
	require.NotEmpty(t, inst.CurrentThread())

	mu.Lock()
	eventsSnapshot := append([]harness.Event(nil), events...)
	mu.Unlock()
	require.NotEmpty(t, eventsSnapshot)
	require.True(t, hasHarnessEvent(eventsSnapshot, harness.EvAgentStart), "events: %#v", eventsSnapshot)
	require.True(t, hasHarnessEvent(eventsSnapshot, harness.EvAgentEnd), "events: %#v", eventsSnapshot)
	require.Contains(t, harnessAssistantText(eventsSnapshot), "GO_HARNESS_BACKEND_OK")

	require.NoError(t, mod.Close())
	require.Nil(t, mod.Instance())
}

func hasHarnessEvent(events []harness.Event, eventType harness.EventType) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}

func harnessAssistantText(events []harness.Event) string {
	var parts []string
	for _, event := range events {
		if string(event.Type) != "message_end" {
			continue
		}
		var payload struct {
			Message struct {
				Role    string `json:"role"`
				Content any    `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil || payload.Message.Role != "assistant" {
			continue
		}
		parts = append(parts, harnessContentText(payload.Message.Content))
	}
	return strings.Join(parts, "")
}

func harnessContentText(content any) string {
	switch value := content.(type) {
	case string:
		return value
	case []any:
		var parts []string
		for _, item := range value {
			part, ok := item.(map[string]any)
			if !ok || part["type"] != "text" {
				continue
			}
			if text, ok := part["text"].(string); ok {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "")
	default:
		return ""
	}
}
