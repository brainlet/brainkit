package plugins

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetricsPluginE2E builds the metrics plugin, starts a Kit with it,
// and verifies all 5 tools work end-to-end.
func TestMetricsPluginE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("plugin e2e test")
	}

	// Build the metrics plugin binary
	pluginDir := filepath.Join("..", "..", "..", "plugins", "brainkit-plugin-metrics")
	if _, err := os.Stat(pluginDir); err != nil {
		// Try from project root
		projectRoot, _ := filepath.Abs("../../..")
		pluginDir = filepath.Join(projectRoot, "..", "plugins", "brainkit-plugin-metrics")
	}
	binaryPath := filepath.Join(t.TempDir(), "metrics-plugin")
	build := exec.Command("go", "build", "-o", binaryPath, ".")
	build.Dir = pluginDir
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	require.NoError(t, build.Run(), "metrics plugin must compile")

	// Start Kit with metrics plugin
	tmpDir := t.TempDir()
	kit, err := brainkit.New(brainkit.Config{
		Namespace: "test-metrics-plugin",
		Transport: brainkit.EmbeddedNATS(),
		FSRoot:    tmpDir,
		Plugins: []brainkit.PluginConfig{{
			Name: "metrics", Binary: binaryPath, AutoRestart: false,
		}},
	})
	require.NoError(t, err)
	defer kit.Close()

	// Wait for plugin registration
	testutil.WaitForPlugin(t, kit, "metrics", 15*time.Second)

	// Deploy something to generate audit events
	testutil.Deploy(t, kit, "metrics-e2e-test.ts", `
		bus.on("ping", function(msg) { msg.reply({pong: true}); });
	`)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Test 1: metrics.snapshot via plugin tool
	t.Run("snapshot", func(t *testing.T) {
		payload := callPluginTool(t, kit, ctx, "snapshot", map[string]any{})
		var snap struct {
			ActiveDeployments int    `json:"activeDeployments"`
			Uptime            string `json:"uptime"`
		}
		require.NoError(t, json.Unmarshal(payload, &snap))
		assert.GreaterOrEqual(t, snap.ActiveDeployments, 1)
		assert.NotEmpty(t, snap.Uptime)
		t.Logf("snapshot: deployments=%d uptime=%s", snap.ActiveDeployments, snap.Uptime)
	})

	// Test 2: audit-query via plugin tool
	t.Run("audit_query", func(t *testing.T) {
		payload := callPluginTool(t, kit, ctx, "audit-query", map[string]any{"category": "deploy", "limit": 10})
		var result struct {
			Events []json.RawMessage `json:"events"`
			Total  int64             `json:"total"`
		}
		require.NoError(t, json.Unmarshal(payload, &result))
		assert.GreaterOrEqual(t, len(result.Events), 1, "should have audit events")
		t.Logf("audit-query: %d events (total: %d)", len(result.Events), result.Total)
	})

	// Test 3: audit-stats via plugin tool
	t.Run("audit_stats", func(t *testing.T) {
		payload := callPluginTool(t, kit, ctx, "audit-stats", map[string]any{})
		var result struct {
			TotalEvents      int64            `json:"totalEvents"`
			EventsByCategory map[string]int64 `json:"eventsByCategory"`
		}
		require.NoError(t, json.Unmarshal(payload, &result))
		assert.Greater(t, result.TotalEvents, int64(0))
		t.Logf("audit-stats: total=%d categories=%v", result.TotalEvents, result.EventsByCategory)
	})

	// Test 4: health via plugin tool
	t.Run("health", func(t *testing.T) {
		payload := callPluginTool(t, kit, ctx, "health", map[string]any{})
		var result struct {
			Healthy bool   `json:"healthy"`
			Status  string `json:"status"`
		}
		require.NoError(t, json.Unmarshal(payload, &result))
		assert.True(t, result.Healthy || result.Status != "")
		t.Logf("health: healthy=%v status=%s", result.Healthy, result.Status)
	})

	// Test 5: audit-prune via plugin tool
	t.Run("audit_prune", func(t *testing.T) {
		payload := callPluginTool(t, kit, ctx, "audit-prune", map[string]any{"olderThanHours": 1})
		var result struct {
			Pruned bool `json:"pruned"`
		}
		require.NoError(t, json.Unmarshal(payload, &result))
		assert.True(t, result.Pruned)
		t.Logf("audit-prune: pruned=%v", result.Pruned)
	})
}

// callPluginTool calls a tool via the bus and returns the result payload.
func callPluginTool(t *testing.T, kit *brainkit.Kit, ctx context.Context, toolName string, input any) json.RawMessage {
	t.Helper()

	replyTo := "tools.call.reply.metrics-" + toolName
	ch := make(chan json.RawMessage, 1)
	unsub, err := kit.SubscribeRaw(ctx, replyTo, func(m sdk.Message) {
		ch <- json.RawMessage(m.Payload)
	})
	require.NoError(t, err)
	defer unsub()

	sdk.Publish(kit, ctx, sdk.ToolCallMsg{
		Name:  toolName,
		Input: input,
	}, sdk.WithReplyTo(replyTo))

	select {
	case payload := <-ch:
		var resp sdk.ToolCallResp
		require.NoError(t, json.Unmarshal(payload, &resp))
		if resp.Error != "" {
			t.Fatalf("tool %s error: %s", toolName, resp.Error)
		}
		return resp.Result
	case <-ctx.Done():
		t.Fatalf("timeout calling %s", toolName)
		return nil
	}
}
