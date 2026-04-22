package deploy

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Endowment gap tests — verify that declared APIs are actually
// reachable inside a deployed .ts Compartment. Each test deploys
// a probe that typeof-checks specific symbols and writes the
// results to output(), which the test reads via EvalTS.

// testEndowmentToolsAvailable checks that `tools.list`, `tools.call`,
// `tools.resolve`, and `tool()` are defined inside a deployment — both
// via bare globals (endowments) AND via the `import { tools } from "kit"`
// form (esbuild external → stripESImports → bare reference).
func testEndowmentToolsAvailable(t *testing.T, env *suite.TestEnv) {
	const source = "endow-tools-deploy-adv.ts"
	code := `
		import { tools, kit } from "kit";

		var checks = {
			toolsType: typeof tools,
			toolsCallType: typeof tools?.call,
			toolsListType: typeof tools?.list,
			toolsResolveType: typeof tools?.resolve,
			toolFnType: typeof tool,
		};

		// If tools.list works, call it
		if (typeof tools === "object" && typeof tools.list === "function") {
			try { checks.listResult = tools.list(); }
			catch (e) { checks.listError = String(e); }
		}

		// If tool() works, try resolving the "echo" tool (registered by suite.Full)
		if (typeof tool === "function") {
			try { checks.echoResolved = !!tool("echo"); }
			catch (e) { checks.echoError = String(e); }
		}

		output(checks);
	`
	testutil.Deploy(t, env.Kit, source, code)
	t.Cleanup(func() { testutil.Teardown(t, env.Kit, source) })

	raw := testutil.EvalTS(t, env.Kit, "__read_endow_tools.ts", `return globalThis.__module_result || "null"`)
	var checks map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &checks))

	t.Logf("tools endowment checks: %v", checks)

	assert.Equal(t, "object", checks["toolsType"], "tools must be an object in deployment")
	assert.Equal(t, "function", checks["toolsCallType"], "tools.call must be a function")
	assert.Equal(t, "function", checks["toolsListType"], "tools.list must be a function")
	assert.Equal(t, "function", checks["toolsResolveType"], "tools.resolve must be a function")
	assert.Equal(t, "function", checks["toolFnType"], "tool() must be a function")

	if checks["listResult"] != nil {
		list, ok := checks["listResult"].([]any)
		assert.True(t, ok, "tools.list() must return an array")
		if ok && len(list) > 0 {
			t.Logf("tools.list() returned %d tools", len(list))
		}
	}
}

// testEndowmentBusCancelSurface checks that `bus.onCancel` and
// `bus.withCancelController` are available inside a deployment.
func testEndowmentBusCancelSurface(t *testing.T, env *suite.TestEnv) {
	const source = "endow-cancel-deploy-adv.ts"
	code := `
		var checks = {
			busOnCancelType: typeof bus.onCancel,
			busWithCancelControllerType: typeof bus.withCancelController,
		};
		output(checks);
	`
	testutil.Deploy(t, env.Kit, source, code)
	t.Cleanup(func() { testutil.Teardown(t, env.Kit, source) })

	raw := testutil.EvalTS(t, env.Kit, "__read_endow_cancel.ts", `return globalThis.__module_result || "null"`)
	var checks map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &checks))

	t.Logf("cancel endowment checks: %v", checks)

	assert.Equal(t, "function", checks["busOnCancelType"], "bus.onCancel must be a function")
	assert.Equal(t, "function", checks["busWithCancelControllerType"], "bus.withCancelController must be a function")
}

// testEndowmentWebSocketAndSetImmediate checks that WebSocket and
// setImmediate are available as globals inside a deployment.
func testEndowmentWebSocketAndSetImmediate(t *testing.T, env *suite.TestEnv) {
	const source = "endow-websocket-deploy-adv.ts"
	code := `
		var checks = {
			webSocketType: typeof WebSocket,
			setImmediateType: typeof setImmediate,
		};
		output(checks);
	`
	testutil.Deploy(t, env.Kit, source, code)
	t.Cleanup(func() { testutil.Teardown(t, env.Kit, source) })

	raw := testutil.EvalTS(t, env.Kit, "__read_endow_ws.ts", `return globalThis.__module_result || "null"`)
	var checks map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &checks))

	t.Logf("WebSocket/setImmediate endowment checks: %v", checks)

	assert.Equal(t, "function", checks["webSocketType"], "WebSocket must be a function (constructor)")
	assert.Equal(t, "function", checks["setImmediateType"], "setImmediate must be a function")
}

// testEndowmentMsgOnCancel checks that `msg.onCancel` is available
// on bus messages received inside a deployment handler.
func testEndowmentMsgOnCancel(t *testing.T, env *suite.TestEnv) {
	const source = "endow-msg-cancel-deploy-adv.ts"
	code := `
		bus.on("probe", (msg) => {
			msg.reply({
				hasOnCancel: typeof msg.onCancel === "function",
			});
		});
	`
	testutil.Deploy(t, env.Kit, source, code)
	t.Cleanup(func() { testutil.Teardown(t, env.Kit, source) })

	payload, err := env.PublishAndWait(t,
		sdk.CustomMsg{Topic: "ts.endow-msg-cancel-deploy-adv.probe", Payload: json.RawMessage(`{}`)},
		10*time.Second)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(suite.ResponseData(payload), &result))

	t.Logf("msg.onCancel check: %v", result)
	assert.Equal(t, true, result["hasOnCancel"], "msg.onCancel must be available on bus messages")
}
