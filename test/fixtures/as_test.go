package fixtures_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/require"
)

// TestASFixturesCompile verifies every AS fixture compiles to WASM.
// This is the fast sanity check — compiles each fixture, no deploy.
func TestASFixturesCompile(t *testing.T) {
	// AS compilation is expensive — only run when requested
	if os.Getenv("BRAINKIT_FIXTURE_AS") == "" {
		t.Skip("set BRAINKIT_FIXTURE_AS=1 to run AS fixture tests")
	}

	entries, err := os.ReadDir(filepath.Join(fixturesRoot(t), "as"))
	require.NoError(t, err)

	tk := testutil.NewTestKernelFull(t)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			source := loadASFixture(t, name)

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			// Compile AS → WASM
			pubResult, err := sdk.Publish(tk, ctx, messages.WasmCompileMsg{
				Source:  source,
				Options: &messages.WasmCompileOpts{Name: name},
			})
			require.NoError(t, err)

			compileCh := make(chan messages.WasmCompileResp, 1)
			unsub, err := sdk.SubscribeTo[messages.WasmCompileResp](tk, ctx, pubResult.ReplyTo,
				func(resp messages.WasmCompileResp, msg messages.Message) {
					compileCh <- resp
				})
			require.NoError(t, err)
			defer unsub()

			select {
			case resp := <-compileCh:
				if resp.Error != "" {
					t.Fatalf("compile %s: %s", name, resp.Error)
				}
				require.NotZero(t, resp.Size, "compiled binary should not be empty")
				t.Logf("compiled %s: %d bytes, %d exports", name, resp.Size, len(resp.Exports))
			case <-ctx.Done():
				t.Fatalf("compile timeout for %s", name)
			}

			if strings.HasPrefix(name, "shard-") {
				// Deploy shard and verify handlers registered
				deployResult, err := sdk.Publish(tk, ctx, messages.WasmDeployMsg{Name: name})
				require.NoError(t, err)

				deployCh := make(chan messages.WasmDeployResp, 1)
				unsub2, err := sdk.SubscribeTo[messages.WasmDeployResp](tk, ctx, deployResult.ReplyTo,
					func(resp messages.WasmDeployResp, msg messages.Message) {
						deployCh <- resp
					})
				require.NoError(t, err)
				defer unsub2()

				select {
				case resp := <-deployCh:
					if resp.Error != "" {
						t.Fatalf("deploy shard %s: %s", name, resp.Error)
					}
					require.NotEmpty(t, resp.Handlers, "shard %s should have handlers", name)
					t.Logf("deployed shard %s: mode=%s, %d handlers", name, resp.Mode, len(resp.Handlers))
				case <-ctx.Done():
					t.Fatalf("deploy timeout for shard %s", name)
				}
			} else {
				// Run the module
				runResult, err := sdk.Publish(tk, ctx, messages.WasmRunMsg{ModuleID: name})
				require.NoError(t, err)

				runCh := make(chan messages.WasmRunResp, 1)
				unsub2, err := sdk.SubscribeTo[messages.WasmRunResp](tk, ctx, runResult.ReplyTo,
					func(resp messages.WasmRunResp, msg messages.Message) {
						runCh <- resp
					})
				require.NoError(t, err)
				defer unsub2()

				select {
				case resp := <-runCh:
					if resp.Error != "" {
						t.Fatalf("run %s: %s", name, resp.Error)
					}
					// Check expect.json
					expect := loadExpect(t, "as", name)
					if expect != nil {
						if ec, ok := expect["exitCode"]; ok {
							require.Equal(t, int(ec.(float64)), resp.ExitCode, "exit code for %s", name)
						}
					}
					t.Logf("ran %s: exitCode=%d", name, resp.ExitCode)
				case <-ctx.Done():
					t.Fatalf("run timeout for %s", name)
				}
			}
		})
	}
}
