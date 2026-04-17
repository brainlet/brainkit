package bus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// matrixPkgDeploy builds a single-file PackageDeployMsg for matrix tests.
func matrixPkgDeploy(name, entry, code string) sdk.PackageDeployMsg {
	manifest, _ := json.Marshal(map[string]string{"name": name, "entry": entry})
	return sdk.PackageDeployMsg{Manifest: manifest, Files: map[string]string{entry: code}}
}

// testTransportMatrixToolsCall — tools.call roundtrip on the env's transport.
// Ported from transport/matrix_test.go:TestBackendMatrix/tools_call.
func testTransportMatrixToolsCall(t *testing.T, env *suite.TestEnv) {
	rt := sdk.Runtime(env.Kit)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pr, err := sdk.Publish(rt, ctx, sdk.ToolCallMsg{
		Name:  "add",
		Input: map[string]any{"a": 10, "b": 32},
	})
	require.NoError(t, err)
	ch := make(chan sdk.ToolCallResp, 1)
	unsub, err := sdk.SubscribeTo[sdk.ToolCallResp](rt, ctx, pr.ReplyTo, func(r sdk.ToolCallResp, m sdk.Message) { ch <- r })
	require.NoError(t, err)
	defer unsub()

	var resp sdk.ToolCallResp
	select {
	case resp = <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	var result map[string]int
	json.Unmarshal(resp.Result, &result)
	assert.Equal(t, 42, result["sum"])
}

// testTransportMatrixToolsList — tools.list returns tools.
// Ported from transport/matrix_test.go:TestBackendMatrix/tools_list.
func testTransportMatrixToolsList(t *testing.T, env *suite.TestEnv) {
	rt := sdk.Runtime(env.Kit)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pr, err := sdk.Publish(rt, ctx, sdk.ToolListMsg{})
	require.NoError(t, err)
	ch := make(chan sdk.ToolListResp, 1)
	unsub, err := sdk.SubscribeTo[sdk.ToolListResp](rt, ctx, pr.ReplyTo, func(r sdk.ToolListResp, m sdk.Message) { ch <- r })
	require.NoError(t, err)
	defer unsub()

	var resp sdk.ToolListResp
	select {
	case resp = <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.NotEmpty(t, resp.Tools)
}

// testTransportMatrixToolsResolve — tools.resolve finds "echo".
// Ported from transport/matrix_test.go:TestBackendMatrix/tools_resolve.
func testTransportMatrixToolsResolve(t *testing.T, env *suite.TestEnv) {
	rt := sdk.Runtime(env.Kit)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pr, err := sdk.Publish(rt, ctx, sdk.ToolResolveMsg{Name: "echo"})
	require.NoError(t, err)
	ch := make(chan sdk.ToolResolveResp, 1)
	unsub, err := sdk.SubscribeTo[sdk.ToolResolveResp](rt, ctx, pr.ReplyTo, func(r sdk.ToolResolveResp, m sdk.Message) { ch <- r })
	require.NoError(t, err)
	defer unsub()

	var resp sdk.ToolResolveResp
	select {
	case resp = <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
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
	rt := sdk.Runtime(env.Kit)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pr, err := sdk.Publish(rt, ctx, sdk.AgentListMsg{})
	require.NoError(t, err)
	ch := make(chan sdk.AgentListResp, 1)
	unsub, err := sdk.SubscribeTo[sdk.AgentListResp](rt, ctx, pr.ReplyTo, func(r sdk.AgentListResp, m sdk.Message) { ch <- r })
	require.NoError(t, err)
	defer unsub()

	var resp sdk.AgentListResp
	select {
	case resp = <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.NotNil(t, resp.Agents)
}

// testTransportMatrixKitDeployTeardown — kit.deploy + call + teardown.
// Ported from transport/matrix_test.go:TestBackendMatrix/kit_deploy_teardown.
func testTransportMatrixKitDeployTeardown(t *testing.T, env *suite.TestEnv) {
	rt := sdk.Runtime(env.Kit)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pr, err := sdk.Publish(rt, ctx, matrixPkgDeploy("matrix-deploy-suite", "matrix-deploy-suite.ts", `
			const matrixTool = createTool({
				id: "matrix-tool-suite",
				description: "matrix test tool",
				execute: async () => ({ backend: "works" })
			});
			kit.register("tool", "matrix-tool-suite", matrixTool);
		`))
	require.NoError(t, err)
	deployCh := make(chan sdk.PackageDeployResp, 1)
	deployUnsub, err := sdk.SubscribeTo[sdk.PackageDeployResp](rt, ctx, pr.ReplyTo, func(r sdk.PackageDeployResp, m sdk.Message) { deployCh <- r })
	require.NoError(t, err)
	defer deployUnsub()

	var deployResp sdk.PackageDeployResp
	select {
	case deployResp = <-deployCh:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.True(t, deployResp.Deployed)

	// Verify tool is callable
	callPR, err := sdk.Publish(rt, ctx, sdk.ToolCallMsg{
		Name: "matrix-tool-suite", Input: map[string]any{},
	})
	require.NoError(t, err)
	callCh := make(chan sdk.ToolCallResp, 1)
	callUnsub, err := sdk.SubscribeTo[sdk.ToolCallResp](rt, ctx, callPR.ReplyTo, func(r sdk.ToolCallResp, m sdk.Message) { callCh <- r })
	require.NoError(t, err)
	defer callUnsub()

	var callResp sdk.ToolCallResp
	select {
	case callResp = <-callCh:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	var result map[string]string
	json.Unmarshal(callResp.Result, &result)
	assert.Equal(t, "works", result["backend"])

	// Teardown
	tdPR, err := sdk.Publish(rt, ctx, sdk.PackageTeardownMsg{Name: "matrix-deploy-suite"})
	require.NoError(t, err)
	tdCh := make(chan sdk.PackageTeardownResp, 1)
	tdUnsub, err := sdk.SubscribeTo[sdk.PackageTeardownResp](rt, ctx, tdPR.ReplyTo, func(r sdk.PackageTeardownResp, m sdk.Message) { tdCh <- r })
	require.NoError(t, err)
	defer tdUnsub()

	select {
	case <-tdCh:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// testTransportMatrixKitRedeploy — deploy then redeploy.
// Ported from transport/matrix_test.go:TestBackendMatrix/kit_redeploy.
func testTransportMatrixKitRedeploy(t *testing.T, env *suite.TestEnv) {
	rt := sdk.Runtime(env.Kit)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sdk.Publish(rt, ctx, matrixPkgDeploy("matrix-redeploy-suite", "matrix-redeploy-suite.ts", `var v = 1;`))

	// Re-deploying the same package name is hot-replace via DeploymentManager.
	pr, err := sdk.Publish(rt, ctx, matrixPkgDeploy("matrix-redeploy-suite", "matrix-redeploy-suite.ts", `var v = 2;`))
	require.NoError(t, err)
	ch := make(chan sdk.PackageDeployResp, 1)
	unsub, err := sdk.SubscribeTo[sdk.PackageDeployResp](rt, ctx, pr.ReplyTo, func(r sdk.PackageDeployResp, m sdk.Message) { ch <- r })
	require.NoError(t, err)
	defer unsub()

	var resp sdk.PackageDeployResp
	select {
	case resp = <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.True(t, resp.Deployed)

	tdPR, _ := sdk.Publish(rt, ctx, sdk.PackageTeardownMsg{Name: "matrix-redeploy-suite"})
	tdCh := make(chan sdk.PackageTeardownResp, 1)
	tdUnsub, _ := sdk.SubscribeTo[sdk.PackageTeardownResp](rt, ctx, tdPR.ReplyTo, func(r sdk.PackageTeardownResp, m sdk.Message) { tdCh <- r })
	defer tdUnsub()
	select {
	case <-tdCh:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// testTransportMatrixRegistryHasList — registry.has + registry.list.
// Ported from transport/matrix_test.go:TestBackendMatrix/registry_has_list.
func testTransportMatrixRegistryHasList(t *testing.T, env *suite.TestEnv) {
	rt := sdk.Runtime(env.Kit)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pr, err := sdk.Publish(rt, ctx, sdk.RegistryHasMsg{
		Category: "provider", Name: "nonexistent",
	})
	require.NoError(t, err)
	ch := make(chan sdk.RegistryHasResp, 1)
	unsub, err := sdk.SubscribeTo[sdk.RegistryHasResp](rt, ctx, pr.ReplyTo, func(r sdk.RegistryHasResp, m sdk.Message) { ch <- r })
	require.NoError(t, err)
	defer unsub()

	var resp sdk.RegistryHasResp
	select {
	case resp = <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.False(t, resp.Found)

	listPR, err := sdk.Publish(rt, ctx, sdk.RegistryListMsg{Category: "provider"})
	require.NoError(t, err)
	listCh := make(chan sdk.RegistryListResp, 1)
	listUnsub, err := sdk.SubscribeTo[sdk.RegistryListResp](rt, ctx, listPR.ReplyTo, func(r sdk.RegistryListResp, m sdk.Message) { listCh <- r })
	require.NoError(t, err)
	defer listUnsub()

	var listResp sdk.RegistryListResp
	select {
	case listResp = <-listCh:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.NotNil(t, listResp.Items)
}

// testTransportMatrixAsyncCorrelation — publish returns a correlation.
// Ported from transport/matrix_test.go:TestBackendMatrix/async_correlation.
func testTransportMatrixAsyncCorrelation(t *testing.T, env *suite.TestEnv) {
	rt := sdk.Runtime(env.Kit)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	corrID, err := sdk.Publish(rt, ctx, sdk.ToolListMsg{})
	require.NoError(t, err)
	assert.NotEmpty(t, corrID)
}
