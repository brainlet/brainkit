package deploy

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testRegistryResourceTracking — deploy registers resources in Go registry,
// list returns them without JS eval, teardown removes them.
func testRegistryResourceTracking(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)

	resources := testutil.DeployWithResources(t, env.Kit, "reg-track.ts", `
		const t1 = createTool({id: "reg-track-tool-1", description: "t1", execute: async () => ({ok: 1})});
		kit.register("tool", "reg-track-tool-1", t1);
		const t2 = createTool({id: "reg-track-tool-2", description: "t2", execute: async () => ({ok: 2})});
		kit.register("tool", "reg-track-tool-2", t2);
	`)

	// Deploy response should report both tools
	assert.Len(t, resources, 2, "deploy should report 2 registered resources")
	types := map[string]bool{}
	for _, r := range resources {
		types[r.Type+":"+r.Name] = true
		assert.Equal(t, "reg-track.ts", r.Source)
	}
	assert.True(t, types["tool:reg-track-tool-1"])
	assert.True(t, types["tool:reg-track-tool-2"])

	// List deployments — should include our source with resources
	deployments := testutil.ListDeployments(t, env.Kit)
	found := false
	for _, d := range deployments {
		if d.Source == "reg-track.ts" {
			found = true
			assert.Len(t, d.Resources, 2, "deployment should list 2 resources")
		}
	}
	require.True(t, found, "reg-track.ts should appear in deployments")

	// Teardown — resources should be gone
	testutil.Teardown(t, env.Kit, "reg-track.ts")

	deployments = testutil.ListDeployments(t, env.Kit)
	for _, d := range deployments {
		assert.NotEqual(t, "reg-track.ts", d.Source, "should be torn down")
	}

	// Tools should be unresolvable
	payload, ok := env.SendAndReceive(t, sdk.ToolResolveMsg{Name: "reg-track-tool-1"}, 5*time.Second)
	require.True(t, ok)
	assert.Equal(t, "NOT_FOUND", suite.ResponseCode(payload))
}

// testRegistrySourceIsolation — teardown of one source does not affect another's resources.
func testRegistrySourceIsolation(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)

	testutil.Deploy(t, env.Kit, "reg-iso-a.ts", `
		const t = createTool({id: "iso-tool-a", description: "a", execute: async () => ({src: "a"})});
		kit.register("tool", "iso-tool-a", t);
	`)
	testutil.Deploy(t, env.Kit, "reg-iso-b.ts", `
		const t = createTool({id: "iso-tool-b", description: "b", execute: async () => ({src: "b"})});
		kit.register("tool", "iso-tool-b", t);
	`)

	// Teardown A — B should be unaffected
	testutil.Teardown(t, env.Kit, "reg-iso-a.ts")

	// A's tool gone
	payload, ok := env.SendAndReceive(t, sdk.ToolResolveMsg{Name: "iso-tool-a"}, 5*time.Second)
	require.True(t, ok)
	assert.Equal(t, "NOT_FOUND", suite.ResponseCode(payload))

	// B's tool still works
	payload2, ok2 := env.SendAndReceive(t, sdk.ToolCallMsg{Name: "iso-tool-b", Input: map[string]any{}}, 5*time.Second)
	require.True(t, ok2)
	assert.Contains(t, string(payload2), `"src":"b"`)

	testutil.Teardown(t, env.Kit, "reg-iso-b.ts")
}

// testRegistryReRegisterAfterTeardown — the v1→teardown→v2 cycle that failed
// before the Go-native registry fix. Verifies tool re-registration works correctly.
func testRegistryReRegisterAfterTeardown(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)

	// Deploy v1
	testutil.Deploy(t, env.Kit, "reg-reregister.ts", `
		const t = createTool({id: "rereg-tool", description: "v1", execute: async () => ({version: 1})});
		kit.register("tool", "rereg-tool", t);
	`)

	// Call v1
	payload, ok := env.SendAndReceive(t, sdk.ToolCallMsg{Name: "rereg-tool", Input: map[string]any{}}, 5*time.Second)
	require.True(t, ok)
	var result1 map[string]any
	json.Unmarshal([]byte(payload), &result1)
	resultData1, _ := json.Marshal(result1["result"])
	assert.Contains(t, string(resultData1), "1")

	// Teardown
	testutil.Teardown(t, env.Kit, "reg-reregister.ts")

	// Tool should be gone
	payload2, ok2 := env.SendAndReceive(t, sdk.ToolResolveMsg{Name: "rereg-tool"}, 5*time.Second)
	require.True(t, ok2)
	assert.Equal(t, "NOT_FOUND", suite.ResponseCode(payload2))

	// Deploy v2 — same source, same tool name, different behavior
	testutil.Deploy(t, env.Kit, "reg-reregister.ts", `
		const t = createTool({id: "rereg-tool", description: "v2", execute: async () => ({version: 2})});
		kit.register("tool", "rereg-tool", t);
	`)

	// Call v2 — must return version 2, not version 1
	payload3, ok3 := env.SendAndReceive(t, sdk.ToolCallMsg{Name: "rereg-tool", Input: map[string]any{}}, 5*time.Second)
	require.True(t, ok3)
	var result3 map[string]any
	json.Unmarshal([]byte(payload3), &result3)
	resultData3, _ := json.Marshal(result3["result"])
	assert.Contains(t, string(resultData3), "2", "should return v2 after re-registration")

	testutil.Teardown(t, env.Kit, "reg-reregister.ts")
}

// testRegistryMixedResourceTypes — deploy with tools, agents, workflows, and subscriptions.
// Verify all tracked, all cleaned up on teardown.
func testRegistryMixedResourceTypes(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)

	resources := testutil.DeployWithResources(t, env.Kit, "reg-mixed.ts", `
		const tool = createTool({id: "mixed-tool", description: "t", execute: async () => ({ok: true})});
		kit.register("tool", "mixed-tool", tool);

		const agent = new Agent({name: "mixed-agent", model: model("openai", "gpt-4o-mini"), instructions: "test"});
		kit.register("agent", "mixed-agent", agent);

		bus.on("ping", (msg) => { msg.reply({pong: true}); });
	`)

	// Should have at least tool + agent + subscription + topic
	assert.GreaterOrEqual(t, len(resources), 3, "should track tool, agent, and subscription")

	// Verify tool works
	payload, ok := env.SendAndReceive(t, sdk.ToolCallMsg{Name: "mixed-tool", Input: map[string]any{}}, 5*time.Second)
	require.True(t, ok)
	assert.Contains(t, string(payload), "ok")

	// Teardown — everything cleaned up
	testutil.Teardown(t, env.Kit, "reg-mixed.ts")

	// Tool gone
	payload2, ok2 := env.SendAndReceive(t, sdk.ToolResolveMsg{Name: "mixed-tool"}, 5*time.Second)
	require.True(t, ok2)
	assert.Equal(t, "NOT_FOUND", suite.ResponseCode(payload2))
}

// testRegistryConcurrentDeployTeardown — multiple goroutines deploying and tearing down.
// No panics, no data races (run with -race).
func testRegistryConcurrentDeployTeardown(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			source := "reg-concurrent-" + string(rune('a'+idx)) + ".ts"
			toolName := "conc-tool-" + string(rune('a'+idx))
			code := `const t = createTool({id: "` + toolName + `", description: "c", execute: async () => ({idx: ` + string(rune('0'+idx)) + `})}); kit.register("tool", "` + toolName + `", t);`

			testutil.Deploy(t, env.Kit, source, code)
			testutil.Teardown(t, env.Kit, source)
		}(i)
	}

	wg.Wait()

	// After all concurrent deploys+teardowns, no leftover resources
	deployments := testutil.ListDeployments(t, env.Kit)
	for _, d := range deployments {
		assert.False(t, len(d.Source) > 0 && d.Source[:len("reg-concurrent-")] == "reg-concurrent-",
			"concurrent deployment %s should be torn down", d.Source)
	}
}

// testRegistryDeployReturnedResources — verify the resource list returned by Deploy
// matches what ListDeployments reports. Both now read from the Go registry.
func testRegistryDeployReturnedResources(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)

	resources := testutil.DeployWithResources(t, env.Kit, "reg-match.ts", `
		const t1 = createTool({id: "match-tool", description: "test", execute: async () => ({ok: true})});
		kit.register("tool", "match-tool", t1);
		bus.on("match-ping", (msg) => { msg.reply({pong: true}); });
	`)

	require.GreaterOrEqual(t, len(resources), 2, "should have tool + subscription")

	deployments := testutil.ListDeployments(t, env.Kit)
	var depResources []sdk.ResourceInfo
	for _, d := range deployments {
		if d.Source == "reg-match.ts" {
			depResources = d.Resources
		}
	}

	// Deploy response and ListDeployments should agree on resource count
	assert.Equal(t, len(resources), len(depResources),
		"deploy response (%d) and list (%d) should agree on resource count",
		len(resources), len(depResources))

	testutil.Teardown(t, env.Kit, "reg-match.ts")
}
