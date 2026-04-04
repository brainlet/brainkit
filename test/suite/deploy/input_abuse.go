package deploy

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDeployEmptySource — empty source name is rejected.
func testDeployEmptySource(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()
	_, err := env.Kernel.Deploy(ctx, "", `output("hello");`)
	assert.Error(t, err, "empty source should be rejected")
	_, err2 := env.Kernel.Deploy(ctx, "   ", `output("hello");`)
	assert.Error(t, err2, "whitespace-only source should be rejected")
}

// testDeployEmptyCode — deploy with completely empty code.
func testDeployEmptyCode(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()
	_, err := env.Kernel.Deploy(ctx, "empty-deploy-adv.ts", "")
	// Should succeed or fail cleanly — never panic
	if err != nil {
		return
	}
	env.Kernel.Teardown(ctx, "empty-deploy-adv.ts")
}

// testDeployHugeCode — deploy 1MB of code (mostly comments).
func testDeployHugeCode(t *testing.T, env *suite.TestEnv) {
	big := "// " + strings.Repeat("x", 1024*1024) + "\noutput('big');"
	_, err := env.Kernel.Deploy(context.Background(), "huge-deploy-adv.ts", big)
	// Should succeed or fail cleanly — never hang
	if err != nil {
		return
	}
	env.Kernel.Teardown(context.Background(), "huge-deploy-adv.ts")
}

// testDeploySourcePathTraversal — source with path traversal characters.
func testDeploySourcePathTraversal(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()
	cases := []string{
		"../escape-deploy-adv.ts",
		"path/with\x00null-deploy-adv.ts",
		`path"with"quotes-deploy-adv.ts`,
		"path`with`backticks-deploy-adv.ts",
		"path with spaces-deploy-adv.ts",
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			_, err := env.Kernel.Deploy(ctx, source, `output("hi");`)
			// Should either succeed (source is just an identifier) or error cleanly — never panic
			_ = err
			if err == nil {
				env.Kernel.Teardown(ctx, source)
			}
		})
	}
}

// testDeployThenImmediateTeardown — deploy and immediately teardown.
func testDeployThenImmediateTeardown(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()
	_, err := env.Kernel.Deploy(ctx, "instant-teardown-deploy-adv.ts", `
		const t = createTool({ id: "instant-td-tool", description: "td", execute: async () => ({}) });
		kit.register("tool", "instant-td-tool", t);
	`)
	require.NoError(t, err)

	// Immediately teardown without waiting
	removed, err := env.Kernel.Teardown(ctx, "instant-teardown-deploy-adv.ts")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, removed, 0)

	// Verify deployment is gone
	deps := env.Kernel.ListDeployments()
	for _, d := range deps {
		assert.NotEqual(t, "instant-teardown-deploy-adv.ts", d.Source, "torn down deployment should not be listed")
	}
}

// testDeployDuplicateSource — deploy same source twice without teardown.
func testDeployDuplicateSource(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()
	_, err := env.Kernel.Deploy(ctx, "dup-deploy-adv.ts", `output("v1");`)
	require.NoError(t, err)

	_, err2 := env.Kernel.Deploy(ctx, "dup-deploy-adv.ts", `output("v2");`)
	assert.Error(t, err2, "duplicate deploy should be rejected")
	assert.Contains(t, err2.Error(), "already exists")

	env.Kernel.Teardown(ctx, "dup-deploy-adv.ts")
}

// testDeployInvalidTSSyntax — deploy code with invalid TypeScript syntax.
func testDeployInvalidTSSyntax(t *testing.T, env *suite.TestEnv) {
	_, err := env.Kernel.Deploy(context.Background(), "invalid-syntax-deploy-adv.ts", "const x: = {{{;;;")
	assert.Error(t, err)
	// Error could come from transpiler OR QuickJS eval — both are valid
	assert.True(t, strings.Contains(err.Error(), "transpile") || strings.Contains(err.Error(), "eval"),
		"expected transpile or eval error, got: %s", err.Error())
}

// testDeployNullBytesInSourceName — source name containing null bytes.
func testDeployNullBytesInSourceName(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()
	_, err := env.Kernel.Deploy(ctx, "null\x00byte-deploy-adv.ts", `output("null");`)
	// Should either succeed (source is just an identifier) or error cleanly — never panic
	if err != nil {
		return
	}
	env.Kernel.Teardown(ctx, "null\x00byte-deploy-adv.ts")
}

// testDeployThrowsDuringInit — deploy code that throws during initialization.
func testDeployThrowsDuringInit(t *testing.T, env *suite.TestEnv) {
	_, err := env.Kernel.Deploy(context.Background(), "throw-init-deploy-adv.ts", `
		throw new Error("init explosion");
	`)
	assert.Error(t, err)

	// Deployment should be cleaned up — not left in a half-state
	deps := env.Kernel.ListDeployments()
	for _, d := range deps {
		assert.NotEqual(t, "throw-init-deploy-adv.ts", d.Source, "failed deployment should not be listed")
	}
}

// testDeployPartialCleanup — deploy that registers a tool then throws: tool should be cleaned up.
func testDeployPartialCleanup(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	_, err := env.Kernel.Deploy(ctx, "partial-deploy-adv.ts", `
		const t = createTool({ id: "partial-adv-tool", description: "partial", execute: async () => ({}) });
		kit.register("tool", "partial-adv-tool", t);
		throw new Error("after registration");
	`)
	assert.Error(t, err)

	// The tool should have been cleaned up by teardown
	pr, pubErr := sdk.Publish(env.Kernel, ctx, messages.ToolResolveMsg{Name: "partial-adv-tool"})
	require.NoError(t, pubErr)

	ch := make(chan []byte, 1)
	unsub, _ := env.Kernel.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case payload := <-ch:
		assert.Contains(t, string(payload), "not found")
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}

// testDeployRedeployDifferentTools — redeploy replaces tools: old tool should be gone.
func testDeployRedeployDifferentTools(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	// Deploy v1 with tool A
	_, err := env.Kernel.Deploy(ctx, "evolving-deploy-adv.ts", `
		const a = createTool({ id: "tool-a-adv", description: "v1", execute: async () => ({ v: 1 }) });
		kit.register("tool", "tool-a-adv", a);
	`)
	require.NoError(t, err)

	// Redeploy with tool B (no tool A)
	_, err = env.Kernel.Redeploy(ctx, "evolving-deploy-adv.ts", `
		const b = createTool({ id: "tool-b-adv", description: "v2", execute: async () => ({ v: 2 }) });
		kit.register("tool", "tool-b-adv", b);
	`)
	require.NoError(t, err)

	// tool-a should not exist
	pr, _ := sdk.Publish(env.Kernel, ctx, messages.ToolResolveMsg{Name: "tool-a-adv"})
	ch := make(chan []byte, 1)
	unsub, _ := env.Kernel.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()
	select {
	case payload := <-ch:
		assert.Contains(t, string(payload), "not found")
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}

	env.Kernel.Teardown(ctx, "evolving-deploy-adv.ts")
}
