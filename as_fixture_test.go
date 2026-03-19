package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/registry"
)

// TestASFixtures runs every .ts file in testdata/as/ as an AS→WASM module.
// Each fixture must export `run(): i32` returning 0 on success.
// A single Kit is shared across all fixtures that don't require an API key,
// so we pay the Kit init cost once (~3s) instead of per-fixture.
func TestASFixtures(t *testing.T) {
	fixtures, err := filepath.Glob("testdata/as/*.ts")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no AS fixtures found in testdata/as/")
	}

	// Shared Kit for fixtures that don't need an API key
	sharedKit := newTestKitNoKey(t)
	ctx := context.Background()

	// Register echo tool once — available to all tool-calling fixtures
	sharedKit.Tools.Register(registry.RegisteredTool{
		Name: "platform.echo", ShortName: "echo", Namespace: "platform",
		Description: "Returns its args as result",
		InputSchema: json.RawMessage(`{"type":"object"}`),
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return input, nil
			},
		},
	})

	for _, f := range fixtures {
		name := strings.TrimSuffix(filepath.Base(f), ".ts")
		t.Run(name, func(t *testing.T) {
			source, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}

			// Fixtures requiring OpenAI get their own Kit
			needsKey := strings.HasPrefix(name, "host-call-agent") ||
				strings.HasPrefix(name, "pattern-agent")

			kit := sharedKit
			if needsKey {
				key := os.Getenv("OPENAI_API_KEY")
				if key == "" {
					t.Skip("skipping: requires OPENAI_API_KEY")
				}
				kit = newTestKit(t)
				kit.Tools.Register(registry.RegisteredTool{
					Name: "platform.echo", ShortName: "echo", Namespace: "platform",
					Description: "Returns its args as result",
					InputSchema: json.RawMessage(`{"type":"object"}`),
					Executor: &registry.GoFuncExecutor{
						Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
							return input, nil
						},
					},
				})
				kit.EvalTS(ctx, "setup-agent.ts", `
					agent({
						name: "test-helper",
						model: "openai/gpt-4o-mini",
						instructions: "Reply with exactly: AGENT_RESPONSE_OK",
					});
				`)
			}

			// Compile with unique module name per fixture
			compileJS := fmt.Sprintf(
				"await wasm.compile(%s, { name: %q, runtime: \"incremental\" });",
				backtickQuote(string(source)), name,
			)
			_, err = kit.EvalTS(ctx, "compile-"+name+".ts", compileJS)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}

			// Run
			runJS := fmt.Sprintf(
				"var r = await wasm.run(%q); return JSON.stringify(r);", name,
			)
			result, err := kit.EvalTS(ctx, "run-"+name+".ts", runJS)
			if err != nil {
				t.Fatalf("run: %v", err)
			}

			var rr struct{ ExitCode int `json:"exitCode"` }
			json.Unmarshal([]byte(result), &rr)
			if rr.ExitCode != 0 {
				t.Fatalf("exitCode=%d (subtest %d failed)", rr.ExitCode, rr.ExitCode)
			}
		})
	}
}

// backtickQuote wraps AS source in JS backticks, escaping internal backticks and ${.
func backtickQuote(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "`", "\\`")
	s = strings.ReplaceAll(s, "${", "\\${")
	return "`" + s + "`"
}
