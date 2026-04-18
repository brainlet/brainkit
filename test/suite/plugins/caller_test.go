package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	pluginsmod "github.com/brainlet/brainkit/modules/plugins"
	"github.com/brainlet/brainkit/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPluginCallerExposesRouter verifies that wsClient exposes a
// non-nil Caller once the plugin is connected. The plugin's "whoami"
// tool returns the caller's inbox topic — the test compares it to the
// expected `_brainkit.plugin-inbox.<owner>.<name>` shape. This is the
// session-09 bundle A contract: Client.Caller() is live inside tool
// handlers, and the inbox name follows the documented scheme.
func TestPluginCallerExposesRouter(t *testing.T) {
	if testing.Short() {
		t.Skip("plugin e2e test")
	}

	dir := t.TempDir()

	pluginCode := `package main

import (
	"context"

	bkplugin "github.com/brainlet/brainkit/sdk/plugin"
)

type EmptyIn struct{}
type InboxOut struct {
	Inbox string ` + "`json:\"inbox\"`" + `
}

func main() {
	p := bkplugin.New("test", "caller-test", "0.1.0")

	bkplugin.Tool(p, "whoami", "report caller inbox", func(_ context.Context, c bkplugin.Client, _ EmptyIn) (InboxOut, error) {
		cl := c.Caller()
		if cl == nil {
			return InboxOut{Inbox: "<nil>"}, nil
		}
		return InboxOut{Inbox: cl.Inbox()}, nil
	})

	p.OnStart(func(_ bkplugin.Client) error { return nil })

	if err := p.Run(); err != nil {
		panic(err)
	}
}
`

	pluginDir := filepath.Join(dir, "caller-plugin")
	require.NoError(t, os.MkdirAll(pluginDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "main.go"), []byte(pluginCode), 0644))

	projectRoot, _ := filepath.Abs("../../..")
	goModContent := fmt.Sprintf(`module test-caller-plugin
go 1.26.0

require (
	github.com/brainlet/brainkit v0.0.0-00010101000000-000000000000
	github.com/brainlet/brainkit/sdk v0.0.0-00010101000000-000000000000
)

replace (
	github.com/brainlet/brainkit => %s
	github.com/brainlet/brainkit/sdk => %s/sdk
	github.com/brainlet/brainkit/vendor_typescript => %s/vendor_typescript
	github.com/brainlet/brainkit/vendor_quickjs => %s/vendor_quickjs
)
`, projectRoot, projectRoot, projectRoot, projectRoot)
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "go.mod"), []byte(goModContent), 0644))

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = pluginDir
	tidy.Stdout = os.Stdout
	tidy.Stderr = os.Stderr
	require.NoError(t, tidy.Run(), "go mod tidy")

	binaryPath := filepath.Join(dir, "caller-plugin-bin")
	build := exec.Command("go", "build", "-o", binaryPath, ".")
	build.Dir = pluginDir
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	require.NoError(t, build.Run(), "plugin build")

	tmpDir := t.TempDir()
	kit, err := brainkit.New(brainkit.Config{
		Namespace: "test-caller-plugin",
		Transport: brainkit.EmbeddedNATS(),
		FSRoot:    tmpDir,
		Modules: []brainkit.Module{
			pluginsmod.NewModule(pluginsmod.Config{
				Plugins: []brainkit.PluginConfig{{
					Name: "caller-test", Binary: binaryPath, AutoRestart: false,
				}},
			}),
		},
	})
	require.NoError(t, err)
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	regCh := make(chan struct{}, 1)
	unsub, _ := sdk.SubscribeTo[sdk.PluginRegisteredEvent](kit, ctx, "plugin.registered",
		func(evt sdk.PluginRegisteredEvent, _ sdk.Message) {
			if evt.Name == "caller-test" {
				select {
				case regCh <- struct{}{}:
				default:
				}
			}
		})
	select {
	case <-regCh:
		unsub()
	case <-ctx.Done():
		unsub()
		t.Fatal("plugin did not register")
	}

	// Call the plugin's whoami tool and verify the reported inbox.
	resp, err := brainkit.Call[sdk.ToolCallMsg, sdk.ToolCallResp](kit, ctx, sdk.ToolCallMsg{
		Name:  "whoami",
		Input: map[string]any{},
	}, brainkit.WithCallTimeout(15*time.Second))
	require.NoError(t, err)

	var out struct {
		Inbox string `json:"inbox"`
	}
	require.NoError(t, json.Unmarshal(resp.Result, &out))
	assert.Equal(t, "_brainkit.plugin-inbox.test.caller-test", out.Inbox,
		"plugin's wsClient.Caller() must expose the documented inbox topic")

	// Pre-existing testutil reference kept so the import never drifts
	// into unused even when this test is the only consumer.
	_ = testutil.Deploy
}
