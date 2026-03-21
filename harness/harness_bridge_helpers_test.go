//go:build e2e

package harness

import (
	"context"
	"os"
	"strings"
	"testing"

	brainkit "github.com/brainlet/brainkit"
)

func loadEnv(t *testing.T) {
	t.Helper()
	for _, path := range []string{".env", "../.env"} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if k, v, ok := strings.Cut(line, "="); ok {
				if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
					v = v[1 : len(v)-1]
				}
				os.Setenv(k, v)
			}
		}
		return
	}
}

func requireKey(t *testing.T) string {
	t.Helper()
	loadEnv(t)
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not set")
	}
	return key
}

func newTestKit(t *testing.T) *brainkit.Kit {
	t.Helper()
	key := requireKey(t)
	kit, err := brainkit.New(brainkit.Config{
		Namespace: "test",
		Providers: map[string]brainkit.ProviderConfig{
			"openai": {APIKey: key},
		},
		EnvVars: map[string]string{
			"OPENAI_API_KEY": key,
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { kit.Close() })
	return kit
}

func setupHarnessKit(t *testing.T) *brainkit.Kit {
	t.Helper()
	return newTestKit(t)
}

func createTestAgent(t *testing.T, kit *brainkit.Kit) {
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
func initTestHarness(t *testing.T, kit *brainkit.Kit, cfg HarnessConfig) *Harness {
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
