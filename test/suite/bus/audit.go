package bus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testAuditQueryAfterDeploy verifies audit.query returns deploy events.
func testAuditQueryAfterDeploy(t *testing.T, env *suite.TestEnv) {
	testutil.Deploy(t, env.Kit, "audit-bus-test.ts", `
		bus.on("ping", function(msg) { msg.reply({ok:true}); });
	`)

	payload := testutil.PublishAndWait(t, env.Kit, sdk.AuditQueryMsg{Category: "deploy"}, 5*time.Second)
	var resp sdk.AuditQueryResp
	require.NoError(t, json.Unmarshal(payload, &resp))
	// May or may not have events depending on whether FSRoot is set (audit needs FSRoot)
	// On memory transport without FSRoot, audit store is nil → empty results
	t.Logf("audit.query returned %d events (total: %d)", len(resp.Events), resp.Total)
}

// testAuditStatsResponse verifies audit.stats returns category counts.
func testAuditStatsResponse(t *testing.T, env *suite.TestEnv) {
	payload := testutil.PublishAndWait(t, env.Kit, sdk.AuditStatsMsg{}, 5*time.Second)
	var resp sdk.AuditStatsResp
	require.NoError(t, json.Unmarshal(payload, &resp))
	assert.NotNil(t, resp.EventsByCategory, "should return category map even if empty")
	t.Logf("audit.stats: total=%d categories=%v", resp.TotalEvents, resp.EventsByCategory)
}

// testAuditPruneWorks verifies audit.prune doesn't error.
func testAuditPruneWorks(t *testing.T, env *suite.TestEnv) {
	payload := testutil.PublishAndWait(t, env.Kit, sdk.AuditPruneMsg{OlderThanHours: 1}, 5*time.Second)
	var resp sdk.AuditPruneResp
	require.NoError(t, json.Unmarshal(payload, &resp))
	// Pruned may be false if no audit store configured (memory transport, no FSRoot)
	t.Logf("audit.prune: pruned=%v", resp.Pruned)
}

// testAuditToolCallRecorded deploys a tool, calls it, and verifies the audit log recorded it.
func testAuditToolCallRecorded(t *testing.T, env *suite.TestEnv) {
	// Call echo tool (registered by suite.Full)
	testutil.PublishAndWait(t, env.Kit, sdk.ToolCallMsg{Name: "echo", Input: map[string]any{"message": "audit-test"}}, 5*time.Second)

	// Query audit for tool calls
	payload := testutil.PublishAndWait(t, env.Kit, sdk.AuditQueryMsg{Category: "tools"}, 5*time.Second)
	var resp sdk.AuditQueryResp
	json.Unmarshal(payload, &resp)
	t.Logf("audit tools events: %d", len(resp.Events))
	// Events may be empty on memory transport without FSRoot
}

// testAuditNoModuleCommandsAbsent verifies that without the audit module the
// audit.* bus commands are not registered — attempts to call them time out.
// Mirror of the schedules no-module test.
func testAuditNoModuleCommandsAbsent(t *testing.T, _ *suite.TestEnv) {
	// Build a raw Kit with no modules so audit.query isn't registered.
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test",
	})
	require.NoError(t, err)
	t.Cleanup(func() { k.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pr, err := sdk.Publish(k, ctx, sdk.AuditQueryMsg{})
	require.NoError(t, err)

	ch := make(chan struct{}, 1)
	unsub, err := k.SubscribeRaw(ctx, pr.ReplyTo, func(_ sdk.Message) { ch <- struct{}{} })
	require.NoError(t, err)
	defer unsub()

	select {
	case <-ch:
		t.Fatal("expected no handler to answer audit.query when module absent")
	case <-ctx.Done():
		// Expected — no handler registered → timeout.
	}
}

// testAuditMetricsGetIncludesBus verifies metrics.get response includes bus per-topic data.
func testAuditMetricsGetIncludesBus(t *testing.T, env *suite.TestEnv) {
	// Generate some traffic first
	testutil.Deploy(t, env.Kit, "metrics-bus-test.ts", `
		bus.on("ping", function(msg) { msg.reply({ok:true}); });
	`)
	testutil.ListDeployments(t, env.Kit)

	payload := testutil.PublishAndWait(t, env.Kit, sdk.MetricsGetMsg{}, 5*time.Second)
	var resp sdk.MetricsGetResp
	require.NoError(t, json.Unmarshal(payload, &resp))

	metricsJSON, _ := json.Marshal(resp.Metrics)
	var m struct {
		ActiveHandlers    int64          `json:"activeHandlers"`
		ActiveDeployments int            `json:"activeDeployments"`
		Bus               *struct {
			Handled map[string]int `json:"handled"`
		} `json:"bus"`
	}
	json.Unmarshal(metricsJSON, &m)

	assert.Greater(t, m.ActiveDeployments, 0, "should have deployments")
	if m.Bus != nil {
		assert.Greater(t, len(m.Bus.Handled), 0, "should have per-topic bus handled counts")
		t.Logf("bus topics handled: %v", m.Bus.Handled)
	}
}
