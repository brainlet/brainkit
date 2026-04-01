package adversarial_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeploy_TSImportsStripped(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "imports.ts", `
		import { bus, kit, tools, output } from "kit";
		import { generateText, z } from "ai";
		import { Agent, createTool } from "agent";

		output({ imported: true, busExists: typeof bus === "object" });
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__imp.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || {});
	`)
	assert.Contains(t, result, `"imported":true`)
	assert.Contains(t, result, `"busExists":true`)
}

func TestDeploy_MultipleDeploymentsCoexist(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		src := fmt.Sprintf("coexist-%d.ts", i)
		_, err := tk.Deploy(ctx, src, fmt.Sprintf(`
			bus.on("ping", function(msg) { msg.reply({id: %d}); });
		`, i))
		require.NoError(t, err)
	}

	deps := tk.ListDeployments()
	assert.Equal(t, 10, len(deps))

	for i := 0; i < 10; i++ {
		tk.Teardown(ctx, fmt.Sprintf("coexist-%d.ts", i))
	}

	deps2 := tk.ListDeployments()
	assert.Equal(t, 0, len(deps2))
}

func TestDeploy_RedeployPreservesOtherDeployments(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "stable.ts", `output("stable");`)
	require.NoError(t, err)
	_, err = tk.Deploy(ctx, "changing.ts", `output("v1");`)
	require.NoError(t, err)

	_, err = tk.Redeploy(ctx, "changing.ts", `output("v2");`)
	require.NoError(t, err)

	deps := tk.ListDeployments()
	sources := make([]string, len(deps))
	for i, d := range deps {
		sources[i] = d.Source
	}
	assert.Contains(t, sources, "stable.ts")
	assert.Contains(t, sources, "changing.ts")
}

func TestDeploy_LongSourceName(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	longName := strings.Repeat("a", 200) + ".ts"
	_, err := tk.Deploy(ctx, longName, `output("long");`)
	if err != nil {
		return
	}
	tk.Teardown(ctx, longName)
}

func TestDeploy_UnicodeSourceName(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "日本語.ts", `output("unicode");`)
	if err != nil {
		return
	}
	deps := tk.ListDeployments()
	found := false
	for _, d := range deps {
		if d.Source == "日本語.ts" {
			found = true
		}
	}
	assert.True(t, found)
	tk.Teardown(ctx, "日本語.ts")
}

func TestDeploy_JSNotTS(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "plain.js", `output("js works");`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__js.ts", `return String(globalThis.__module_result || "");`)
	assert.Equal(t, "js works", result)
}

func TestDeploy_EmptyCode(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "empty.ts", "")
	if err != nil {
		return
	}
	tk.Teardown(ctx, "empty.ts")
}

func TestDeploy_CodeWithOnlyComments(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "comments.ts", `
		// This file does nothing
		/* Just comments */
	`)
	if err != nil {
		return
	}
	tk.Teardown(ctx, "comments.ts")
}

func TestDeploy_AsyncInit(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "async-init.ts", `
		const result = await Promise.resolve(42);
		output({ asyncResult: result });
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__async.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || {});
	`)
	assert.Contains(t, result, "42")
}

func TestDeploy_ToolWithComplexSchema(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "complex-tool.ts", `
		import { createTool, z } from "agent";
		import { kit, output } from "kit";

		const tool = createTool({
			id: "complex",
			description: "complex input",
			inputSchema: z.object({
				name: z.string(),
				age: z.number().optional(),
				tags: z.array(z.string()).optional(),
				nested: z.object({ key: z.string() }).optional(),
			}),
			execute: async ({ name, age, tags, nested }) => ({
				processed: true,
				name,
				age: age || 0,
				tagCount: (tags || []).length,
				hasNested: !!nested,
			}),
		});
		kit.register("tool", "complex", tool);
		output({ registered: true });
	`)
	require.NoError(t, err)

	payload, ok := sendAndReceive(t, tk, messages.ToolCallMsg{
		Name: "complex",
		Input: map[string]any{
			"name": "test",
			"age":  25,
			"tags": []string{"a", "b"},
			"nested": map[string]string{"key": "val"},
		},
	}, 5*time.Second)
	require.True(t, ok)
	assert.Contains(t, string(payload), "processed")
}

func TestDeploy_MultipleToolsOneDeployment(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "multi-tools.ts", `
		import { createTool, z } from "agent";
		import { kit, output } from "kit";

		for (let i = 0; i < 5; i++) {
			const t = createTool({
				id: "batch-" + i,
				description: "batch tool " + i,
				inputSchema: z.object({}),
				execute: async () => ({ toolIndex: i }),
			});
			kit.register("tool", "batch-" + i, t);
		}
		output({ toolsRegistered: 5 });
	`)
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		payload, ok := sendAndReceive(t, tk, messages.ToolResolveMsg{Name: fmt.Sprintf("batch-%d", i)}, 5*time.Second)
		require.True(t, ok, "batch-%d should be resolvable", i)
		assert.NotContains(t, string(payload), "not found")
	}
}

func TestDeploy_AgentRegistration(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "agent-reg.ts", `
		import { kit, output } from "kit";
		kit.register("agent", "test-bot", {});
		output({ registered: true });
	`)
	require.NoError(t, err)

	pr, _ := sdk.Publish(tk, ctx, messages.AgentListMsg{})
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "test-bot")
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}

func TestDeploy_WorkflowRegistration(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "wf-reg.ts", `
		import { kit, output } from "kit";
		kit.register("workflow", "test-workflow", {});
		output({ registered: true });
	`)
	require.NoError(t, err)

	resources, _ := tk.ListResources("workflow")
	found := false
	for _, r := range resources {
		if r.Name == "test-workflow" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestDeploy_MemoryRegistration(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "mem-reg.ts", `
		import { kit, output } from "kit";
		kit.register("memory", "test-memory", {});
		output({ registered: true });
	`)
	require.NoError(t, err)

	resources, _ := tk.ListResources("memory")
	found := false
	for _, r := range resources {
		if r.Name == "test-memory" {
			found = true
		}
	}
	assert.True(t, found)
}
