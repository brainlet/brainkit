package bus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/modules/agents/agentmsg"
	"github.com/brainlet/brainkit/modules/packages/packagemsg"
	"github.com/brainlet/brainkit/modules/registry/registrymsg"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/protocol"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// matrixPkgDeploy builds a single-file PackageDeployMsg for matrix tests.
func matrixPkgDeploy(name, entry, code string) packagemsg.PackageDeployMsg {
	manifest, _ := json.Marshal(map[string]string{"name": name, "entry": entry})
	return packagemsg.PackageDeployMsg{Manifest: manifest, Files: map[string]string{entry: code}}
}

// testTransportMatrixToolsCall — tools.call roundtrip on the env's transport.
// Ported from transport/matrix_test.go:TestBackendMatrix/tools_call.
func testTransportMatrixToolsCall(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := sdk.Call[toolmsg.ToolCallMsg, toolmsg.ToolCallResp](env.Kit, ctx, toolmsg.ToolCallMsg{
		Name:  "add",
		Input: map[string]any{"a": 10, "b": 32},
	})
	require.NoError(t, err)
	var result map[string]int
	json.Unmarshal(resp.Result, &result)
	assert.Equal(t, 42, result["sum"])
}

// testTransportMatrixToolsList — tools.list returns tools.
// Ported from transport/matrix_test.go:TestBackendMatrix/tools_list.
func testTransportMatrixToolsList(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := sdk.Call[toolmsg.ToolListMsg, toolmsg.ToolListResp](env.Kit, ctx, toolmsg.ToolListMsg{})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Tools)
}

// testTransportMatrixToolsResolve — tools.resolve finds "echo".
// Ported from transport/matrix_test.go:TestBackendMatrix/tools_resolve.
func testTransportMatrixToolsResolve(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := sdk.Call[toolmsg.ToolResolveMsg, toolmsg.ToolResolveResp](env.Kit, ctx, toolmsg.ToolResolveMsg{Name: "echo"})
	require.NoError(t, err)
	assert.Equal(t, "echo", resp.ShortName)
}

// testTransportMatrixFSWriteRead — fs write+read roundtrip.
// Ported from transport/matrix_test.go:TestBackendMatrix/fs_write_read.
func testTransportMatrixFSWriteRead(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__test_matrix_fs.ts", `
		fs.writeFileSync("matrix-test-suite.txt", "backend:memory");
		return fs.readFileSync("matrix-test-suite.txt", "utf8");
	`)
	assert.Equal(t, "backend:memory", result)
}

// testTransportMatrixFSMkdirListStatDelete — fs mkdir, list, stat, delete.
// Ported from transport/matrix_test.go:TestBackendMatrix/fs_mkdir_list_stat_delete.
func testTransportMatrixFSMkdirListStatDelete(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__test_matrix_fsdir.ts", `
		fs.mkdirSync("matrix-dir-suite", {recursive: true});
		fs.writeFileSync("matrix-dir-suite/a.txt", "a");
		var files = fs.readdirSync("matrix-dir-suite");
		var s = fs.statSync("matrix-dir-suite/a.txt");
		fs.unlinkSync("matrix-dir-suite/a.txt");
		return JSON.stringify({fileCount: files.length, isDir: s.isDirectory()});
	`)
	var resp struct {
		FileCount int  `json:"fileCount"`
		IsDir     bool `json:"isDir"`
	}
	json.Unmarshal([]byte(result), &resp)
	assert.Equal(t, 1, resp.FileCount)
	assert.False(t, resp.IsDir)
}

// testTransportMatrixAgentsListEmpty — agents.list returns non-nil.
// Ported from transport/matrix_test.go:TestBackendMatrix/agents_list_empty.
func testTransportMatrixAgentsListEmpty(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := sdk.Call[agentmsg.AgentListMsg, agentmsg.AgentListResp](env.Kit, ctx, agentmsg.AgentListMsg{})
	require.NoError(t, err)
	assert.NotNil(t, resp.Agents)
}

// testTransportMatrixKitDeployTeardown — kit.deploy + call + teardown.
// Ported from transport/matrix_test.go:TestBackendMatrix/kit_deploy_teardown.
func testTransportMatrixKitDeployTeardown(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	deployResp, err := sdk.Call[packagemsg.PackageDeployMsg, packagemsg.PackageDeployResp](env.Kit, ctx, matrixPkgDeploy("matrix-deploy-suite", "matrix-deploy-suite.ts", `
			const matrixTool = createTool({
				id: "matrix-tool-suite",
				description: "matrix test tool",
				execute: async () => ({ backend: "works" })
			});
			kit.register("tool", "matrix-tool-suite", matrixTool);
		`))
	require.NoError(t, err)
	assert.True(t, deployResp.Deployed)

	// Verify tool is callable
	callResp, err := sdk.Call[toolmsg.ToolCallMsg, toolmsg.ToolCallResp](env.Kit, ctx, toolmsg.ToolCallMsg{
		Name: "matrix-tool-suite", Input: map[string]any{},
	})
	require.NoError(t, err)
	var result map[string]string
	json.Unmarshal(callResp.Result, &result)
	assert.Equal(t, "works", result["backend"])

	// Teardown
	_, err = sdk.Call[packagemsg.PackageTeardownMsg, packagemsg.PackageTeardownResp](env.Kit, ctx, packagemsg.PackageTeardownMsg{Name: "matrix-deploy-suite"})
	require.NoError(t, err)
}

// testTransportMatrixKitRedeploy — deploy then redeploy.
// Ported from transport/matrix_test.go:TestBackendMatrix/kit_redeploy.
func testTransportMatrixKitRedeploy(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_, err := sdk.Call[packagemsg.PackageDeployMsg, packagemsg.PackageDeployResp](env.Kit, ctx,
		matrixPkgDeploy("matrix-redeploy-suite", "matrix-redeploy-suite.ts", `bus.on("v", (msg) => msg.reply({ version: 1 }));`))
	require.NoError(t, err)

	// Re-deploying the same package name is hot-replace via DeploymentManager.
	resp, err := sdk.Call[packagemsg.PackageDeployMsg, packagemsg.PackageDeployResp](env.Kit, ctx,
		matrixPkgDeploy("matrix-redeploy-suite", "matrix-redeploy-suite.ts", `bus.on("v", (msg) => msg.reply({ version: 2 }));`))
	require.NoError(t, err)
	assert.True(t, resp.Deployed)

	_, err = sdk.Call[packagemsg.PackageTeardownMsg, packagemsg.PackageTeardownResp](env.Kit, ctx, packagemsg.PackageTeardownMsg{Name: "matrix-redeploy-suite"})
	require.NoError(t, err)
}

// testTransportMatrixRegistryHasList — registry.has + registry.list.
// Ported from transport/matrix_test.go:TestBackendMatrix/registry_has_list.
func testTransportMatrixRegistryHasList(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := sdk.Call[registrymsg.RegistryHasMsg, registrymsg.RegistryHasResp](env.Kit, ctx, registrymsg.RegistryHasMsg{
		Category: "provider", Name: "nonexistent",
	})
	require.NoError(t, err)
	assert.False(t, resp.Found)

	listResp, err := sdk.Call[registrymsg.RegistryListMsg, registrymsg.RegistryListResp](env.Kit, ctx, registrymsg.RegistryListMsg{Category: "provider"})
	require.NoError(t, err)
	assert.NotNil(t, listResp.Items)
}

// testTransportMatrixAsyncCorrelation — publish returns a correlation.
// Ported from transport/matrix_test.go:TestBackendMatrix/async_correlation.
func testTransportMatrixAsyncCorrelation(t *testing.T, env *suite.TestEnv) {
	rt := sdk.Runtime(env.Kit)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	corrID, err := protocol.Publish(rt, ctx, toolmsg.ToolListMsg{})
	require.NoError(t, err)
	assert.NotEmpty(t, corrID)
}
