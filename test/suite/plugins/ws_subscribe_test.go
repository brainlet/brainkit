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
	pluginsmod "github.com/brainlet/brainkit/modules/plugins"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPluginWSSubscribe verifies that a plugin can subscribe to bus events via WS.
// The plugin subscribes to a topic, we publish to that topic, and verify the plugin
// received the event by calling a tool that reports what it received.
func TestPluginWSSubscribe(t *testing.T) {
	if testing.Short() {
		t.Skip("plugin e2e test")
	}

	dir := t.TempDir()

	// Write a minimal plugin that subscribes to "test.events" and tracks received events
	pluginCode := `package main

import (
	"context"
	"encoding/json"
	"sync"

	bkplugin "github.com/brainlet/brainkit/sdk/plugin"
)

var (
	received []json.RawMessage
	mu       sync.Mutex
)

type StatusInput struct{}
type StatusOutput struct {
	Count    int               ` + "`json:\"count\"`" + `
	Messages []json.RawMessage ` + "`json:\"messages\"`" + `
}

func main() {
	p := bkplugin.New("test", "sub-test", "0.1.0")

	bkplugin.Tool(p, "status", "report received events", func(_ context.Context, _ bkplugin.Client, _ StatusInput) (StatusOutput, error) {
		mu.Lock()
		defer mu.Unlock()
		return StatusOutput{Count: len(received), Messages: received}, nil
	})

	bkplugin.On[json.RawMessage](p, "test.events", func(_ context.Context, payload json.RawMessage, _ bkplugin.Client) {
		mu.Lock()
		received = append(received, payload)
		mu.Unlock()
	})

	p.OnStart(func(_ bkplugin.Client) error {
		return nil
	})

	if err := p.Run(); err != nil {
		panic(err)
	}
}
`

	// Write plugin source
	pluginDir := filepath.Join(dir, "sub-plugin")
	os.MkdirAll(pluginDir, 0755)
	os.WriteFile(filepath.Join(pluginDir, "main.go"), []byte(pluginCode), 0644)
	// Find project root (directory containing go.mod)
	projectRoot, _ := filepath.Abs("../../..")
	os.WriteFile(filepath.Join(pluginDir, "go.mod"), []byte(fmt.Sprintf(`module test-sub-plugin
go 1.26.0
require github.com/brainlet/brainkit/sdk v0.0.0-00010101000000-000000000000
replace github.com/brainlet/brainkit/sdk => %s/sdk
`, projectRoot)), 0644)

	// Tidy + build
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = pluginDir
	tidy.Stdout = os.Stdout
	tidy.Stderr = os.Stderr
	require.NoError(t, tidy.Run(), "go mod tidy must succeed")

	binaryPath := filepath.Join(dir, "sub-plugin-bin")
	build := exec.Command("go", "build", "-o", binaryPath, ".")
	build.Dir = pluginDir
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	require.NoError(t, build.Run(), "plugin must compile")

	// Start Kit with plugin
	kit, err := brainkit.New(brainkit.Config{
		Namespace: "test-ws-sub",
		Transport: brainkit.EmbeddedNATS(),
		Modules: []brainkit.Module{
			pluginsmod.NewModule(pluginsmod.Config{
				Plugins: []brainkit.PluginConfig{{
					Name: "sub-test", Binary: binaryPath, AutoRestart: false,
				}},
			}),
		},
	})
	require.NoError(t, err)
	defer kit.Close()

	// Wait for plugin registration
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	regCh := make(chan struct{}, 1)
	unsub, _ := sdk.SubscribeTo[sdk.PluginRegisteredEvent](kit, ctx, "plugin.registered",
		func(evt sdk.PluginRegisteredEvent, _ sdk.Message) {
			if evt.Name == "sub-test" {
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

	// Wait for the plugin's WS subscription to be registered on the embedded NATS server.
	time.Sleep(1 * time.Second)

	// Publish 3 events to "test.events"
	for i := 0; i < 3; i++ {
		payload, _ := json.Marshal(map[string]int{"seq": i})
		kit.PublishRaw(ctx, "test.events", payload)
		time.Sleep(200 * time.Millisecond)
	}

	// Wait for events to reach plugin over WS
	time.Sleep(1 * time.Second)

	// Call the "status" tool to check what the plugin received
	replyTo := "tools.call.reply.ws-sub-test"
	replyCh := make(chan sdk.Message, 1)
	unsubReply, _ := kit.SubscribeRaw(ctx, replyTo, func(m sdk.Message) {
		replyCh <- m
	})
	defer unsubReply()

	sdk.Publish(kit, ctx, sdk.ToolCallMsg{
		Name:  "status",
		Input: map[string]any{},
	}, sdk.WithReplyTo(replyTo))

	select {
	case msg := <-replyCh:
		require.Empty(t, suite.ResponseErrorMessage(msg.Payload))
		data := suite.ResponseDataFromMsg(msg)
		var resp sdk.ToolCallResp
		require.NoError(t, json.Unmarshal(data, &resp))

		var status struct {
			Count    int               `json:"count"`
			Messages []json.RawMessage `json:"messages"`
		}
		require.NoError(t, json.Unmarshal(resp.Result, &status))
		assert.GreaterOrEqual(t, status.Count, 1, "plugin should have received at least 1 event")
		t.Logf("plugin received %d events", status.Count)

	case <-ctx.Done():
		t.Fatal("timeout calling status tool")
	}
}
