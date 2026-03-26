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
	"github.com/brainlet/brainkit/kit"
	provreg "github.com/brainlet/brainkit/kit/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPluginFixtures runs .ts fixtures that require a Node with a plugin
// subprocess connected via NATS. Fixtures live in fixtures/ts/plugin/.
//
// The test plugin registers a "greet" tool. The .ts fixture calls it.
//
// Requires: Podman (NATS container), test plugin binary
func TestPluginFixtures(t *testing.T) {
	if !testutil.PodmanAvailable() {
		t.Skip("plugin fixtures require Podman (NATS container)")
	}
	testutil.LoadEnv(t)
	testutil.CleanupOrphanedContainers(t)

	// Build the test plugin binary
	pluginBinary := testutil.BuildTestPlugin(t)
	t.Logf("test plugin binary: %s", pluginBinary)

	// Create NATS transport config
	natsCfg := testutil.TransportConfigForBackend(t, "nats")
	tmpDir := t.TempDir()

	aiProviders := make(map[string]provreg.AIProviderRegistration)
	envVars := make(map[string]string)
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		aiProviders["openai"] = provreg.AIProviderRegistration{
			Type:   provreg.AIProviderOpenAI,
			Config: provreg.OpenAIProviderConfig{APIKey: key},
		}
		envVars["OPENAI_API_KEY"] = key
	}

	// Create a Node with the plugin
	node, err := kit.NewNode(kit.NodeConfig{
		Kernel: kit.KernelConfig{
			Namespace:    "test-plugin",
			CallerID:     "test-plugin-caller",
			WorkspaceDir: tmpDir,
			AIProviders:  aiProviders,
			EnvVars:      envVars,
			EmbeddedStorages: map[string]kit.EmbeddedStorageConfig{
				"default": {Path: filepath.Join(tmpDir, "brainkit.db")},
			},
		},
		Messaging: kit.MessagingConfig{
			Transport: natsCfg.Type,
			NATSURL:   natsCfg.NATSURL,
			NATSName:  "test-plugin",
		},
		Plugins: []kit.PluginConfig{
			{
				Name:   "testplugin",
				Binary: pluginBinary,
			},
		},
	})
	if err != nil {
		t.Fatalf("NewNode: %v", err)
	}
	defer node.Close()

	if err := node.Start(context.Background()); err != nil {
		t.Fatalf("Node.Start: %v", err)
	}

	// Wait for plugin to register
	time.Sleep(2 * time.Second)

	fixtures, err := os.ReadDir(filepath.Join(fixturesRoot(t), "ts", "plugin"))
	if err != nil {
		t.Skip("no plugin fixtures found")
	}

	for _, entry := range fixtures {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			tsSource := loadTSFixtureRaw(t, "plugin", name)

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			_, err := node.Kernel.Deploy(ctx, name+".ts", tsSource)
			if err != nil {
				t.Fatalf("deploy plugin/%s: %v", name, err)
			}

			raw, err := node.Kernel.EvalTS(ctx, "__read_output.ts",
				`return typeof globalThis.__module_result !== "undefined" ? globalThis.__module_result : ""`)
			require.NoError(t, err)

			if raw == "" {
				t.Logf("plugin/%s: no output", name)
				return
			}

			var actual map[string]any
			if err := json.Unmarshal([]byte(raw), &actual); err != nil {
				t.Logf("plugin/%s output (raw): %s", name, raw)
				return
			}
			t.Logf("plugin/%s output: %s", name, truncate(raw, 500))

			expect := loadExpect(t, "plugin", name)
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
