package fixtures_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCrossKitFixtures runs .ts fixtures that require two Kernels on a shared
// NATS transport. Fixtures live in fixtures/ts/cross-kit/.
//
// Kit A: deploys the .ts fixture
// Kit B: has Go tools registered (echo)
// Both share a NATS transport so cross-namespace messaging works.
//
// Requires: Podman (NATS container)
func TestCrossKitFixtures(t *testing.T) {
	if !testutil.PodmanAvailable() {
		t.Skip("cross-kit fixtures require Podman (NATS container)")
	}
	testutil.LoadEnv(t)
	testutil.CleanupOrphanedContainers(t)

	// Create two Kernels on shared NATS transport
	kitA, kitB := testutil.NewTestKernelPairFull(t, "nats")
	_ = kitB // Kit B provides tools via shared transport

	// Set env so .ts fixtures know Kit B's namespace
	os.Setenv("CROSS_KIT_TARGET_NS", "kit-b")

	fixtures, err := os.ReadDir(filepath.Join(fixturesRoot(t), "ts", "cross-kit"))
	if err != nil {
		t.Skip("no cross-kit fixtures found")
	}

	for _, entry := range fixtures {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			tsSource := loadTSFixtureRaw(t, "cross-kit", name)

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			_, err := kitA.Deploy(ctx, name+".ts", tsSource)
			if err != nil {
				t.Fatalf("deploy cross-kit/%s: %v", name, err)
			}

			raw, err := kitA.EvalTS(ctx, "__read_output.ts",
				`return typeof globalThis.__module_result !== "undefined" ? globalThis.__module_result : ""`)
			require.NoError(t, err)

			if raw == "" {
				t.Logf("cross-kit/%s: no output", name)
				return
			}

			var actual map[string]any
			if err := json.Unmarshal([]byte(raw), &actual); err != nil {
				t.Logf("cross-kit/%s output (raw): %s", name, raw)
				return
			}
			t.Logf("cross-kit/%s output: %s", name, truncate(raw, 500))

			expect := loadExpect(t, "cross-kit", name)
			if expect == nil {
				return
			}

			for key, expected := range expect {
				actualVal, exists := actual[key]
				if !exists {
					t.Errorf("missing key %q in output", key)
					continue
				}
				switch ev := expected.(type) {
				case bool:
					assert.Equal(t, ev, actualVal, "key %s", key)
				case string:
					if ev == "*" {
						assert.NotNil(t, actualVal)
					} else if strings.HasPrefix(ev, "~") {
						assert.Contains(t, actualVal, ev[1:])
					} else {
						assert.Equal(t, ev, actualVal)
					}
				default:
					assert.Equal(t, expected, actualVal, "key %s", key)
				}
			}
		})
	}
}
