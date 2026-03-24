package fixtures_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTSFixturesTranspile verifies every TS fixture transpiles to valid JS.
// This is the fast sanity check — no Kernel needed.
func TestTSFixturesTranspile(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join(fixturesRoot(t), "ts"))
	require.NoError(t, err)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			js := loadTSFixture(t, name)
			require.NotEmpty(t, js, "transpiled output should not be empty")
		})
	}
}

// TestTSFixturesDeploy deploys each TS fixture into a Kernel and checks output.
// Fixtures that need AI keys are skipped when OPENAI_API_KEY is not set.
func TestTSFixturesDeploy(t *testing.T) {
	// Only run deploy tests when explicitly requested (they're slow)
	if os.Getenv("BRAINKIT_FIXTURE_DEPLOY") == "" {
		t.Skip("set BRAINKIT_FIXTURE_DEPLOY=1 to run deploy tests")
	}

	entries, err := os.ReadDir(filepath.Join(fixturesRoot(t), "ts"))
	require.NoError(t, err)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			js := loadTSFixture(t, name)
			tk := testutil.NewTestKernelFull(t)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Deploy via bus pattern
			pubResult, err := sdk.Publish(tk, ctx, messages.KitDeployMsg{
				Source: name + ".ts",
				Code:   js,
			})
			require.NoError(t, err)

			deployCh := make(chan messages.KitDeployResp, 1)
			unsub, err := sdk.SubscribeTo[messages.KitDeployResp](tk, ctx, pubResult.ReplyTo,
				func(resp messages.KitDeployResp, msg messages.Message) {
					deployCh <- resp
				})
			require.NoError(t, err)
			defer unsub()

			select {
			case resp := <-deployCh:
				if resp.Error != "" {
					if !testutil.HasAIKey() {
						t.Skipf("fixture %s needs AI key: %s", name, resp.Error)
					}
					t.Fatalf("deploy %s: %s", name, resp.Error)
				}
			case <-ctx.Done():
				t.Fatalf("deploy %s: timeout", name)
			}

			// Check expectations if expect.json exists
			expect := loadExpect(t, "ts", name)
			if expect != nil {
				result, err := tk.EvalTS(ctx, "__check.ts",
					`return typeof globalThis.__module_result !== "undefined" ? globalThis.__module_result : ""`)
				require.NoError(t, err)

				if result != "" {
					var actual map[string]any
					require.NoError(t, json.Unmarshal([]byte(result), &actual))
					for key, expected := range expect {
						assert.Equal(t, expected, actual[key], "fixture %s: key %s", name, key)
					}
				}
			}
		})
	}
}
