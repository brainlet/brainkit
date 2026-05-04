package packages

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/modules/packages/packagemsg"
	"github.com/brainlet/brainkit/modules/secrets/secretmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writePackageFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	os.MkdirAll(filepath.Dir(path), 0755)
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
}

func testMultiFileProject(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t, suite.WithPersistence(), suite.WithSecretKey("test-key"))
	ctx := context.Background()

	dir := t.TempDir()
	writePackageFile(t, dir, "manifest.json", `{
		"name": "test-pkg",
		"version": "1.0.0",
		"entry": "index.ts"
	}`)
	writePackageFile(t, dir, "config.ts", `export const PREFIX = "Hello";`)
	writePackageFile(t, dir, "index.ts", `
		import { PREFIX } from "./config";
		bus.on("greet", (msg) => {
			msg.reply({ text: PREFIX + " " + msg.payload.name });
		});
	`)

	deployResp, err := packagemsg.CallPackageDeploy(env.Kit, ctx, packagemsg.PackageDeployMsg{Path: dir}, sdk.WithCallTimeout(10*time.Second))
	require.NoError(t, err)
	require.True(t, deployResp.Deployed)
	assert.Equal(t, "test-pkg", deployResp.Name)
	assert.Equal(t, "test-pkg.ts", deployResp.Source)

	time.Sleep(200 * time.Millisecond)

	serviceResp := sendToServiceAndWait(t, env.Kit, "test-pkg", "greet", map[string]string{"name": "World"})
	assert.Equal(t, "Hello World", serviceResp["text"])
}

func testListAndTeardown(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t, suite.WithPersistence(), suite.WithSecretKey("test-key"))
	ctx := context.Background()

	dir := t.TempDir()
	writePackageFile(t, dir, "manifest.json", `{
		"name": "list-test",
		"version": "2.0.0",
		"entry": "index.ts"
	}`)
	writePackageFile(t, dir, "index.ts", `bus.on("ping", (msg) => { msg.reply({pong: true}); });`)

	deployResp, err := packagemsg.CallPackageDeploy(env.Kit, ctx, packagemsg.PackageDeployMsg{Path: dir}, sdk.WithCallTimeout(10*time.Second))
	require.NoError(t, err)
	require.True(t, deployResp.Deployed)

	listResp, err := packagemsg.CallPackageListDeployed(env.Kit, ctx, packagemsg.PackageListDeployedMsg{}, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)
	require.Len(t, listResp.Packages, 1)
	assert.Equal(t, "list-test", listResp.Packages[0].Name)
	assert.Equal(t, "list-test.ts", listResp.Packages[0].Source)

	tearResp, err := packagemsg.CallPackageTeardown(env.Kit, ctx, packagemsg.PackageTeardownMsg{Name: "list-test"}, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)
	assert.True(t, tearResp.Removed)

	listResp2, err := packagemsg.CallPackageListDeployed(env.Kit, ctx, packagemsg.PackageListDeployedMsg{}, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)
	assert.Len(t, listResp2.Packages, 0)
}

func testSecretDependencyCheck(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t, suite.WithPersistence(), suite.WithSecretKey("test-key"))
	ctx := context.Background()

	dir := t.TempDir()
	writePackageFile(t, dir, "manifest.json", `{
		"name": "needs-secret",
		"version": "1.0.0",
		"entry": "index.ts",
		"requires": { "secrets": ["MY_REQUIRED_SECRET"] }
	}`)
	writePackageFile(t, dir, "index.ts", `bus.on("x", (msg) => { msg.reply({}); });`)

	_, err := packagemsg.CallPackageDeploy(env.Kit, ctx, packagemsg.PackageDeployMsg{Path: dir}, sdk.WithCallTimeout(5*time.Second))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MY_REQUIRED_SECRET")
	assert.Contains(t, err.Error(), "not set")

	setResp, err := secretmsg.CallSecretsSet(env.Kit, ctx, secretmsg.SecretsSetMsg{Name: "MY_REQUIRED_SECRET", Value: "secret-value"}, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)
	assert.True(t, setResp.Stored)

	deployResp, err := packagemsg.CallPackageDeploy(env.Kit, ctx, packagemsg.PackageDeployMsg{Path: dir}, sdk.WithCallTimeout(10*time.Second))
	require.NoError(t, err)
	assert.True(t, deployResp.Deployed)
}

func testInlineFilesRedeployPicksUpNewCode(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t, suite.WithPersistence())
	ctx := context.Background()

	manifest := `{
		"name": "evolve-pkg",
		"version": "1.0.0",
		"entry": "index.ts"
	}`

	v1Code := `bus.on("check", (msg) => { msg.reply({ version: "v1" }); });`

	resp, err := packagemsg.CallPackageDeploy(env.Kit, ctx, packagemsg.PackageDeployMsg{
		Manifest: json.RawMessage(manifest),
		Files:    map[string]string{"index.ts": v1Code},
	}, sdk.WithCallTimeout(10*time.Second))
	require.NoError(t, err)
	require.True(t, resp.Deployed)
	t.Logf("v1 deployed: %s", resp.Source)

	time.Sleep(200 * time.Millisecond)

	v1Resp := sendToServiceAndWait(t, env.Kit, "evolve-pkg", "check", nil)
	require.Equal(t, "v1", v1Resp["version"], "v1 should return version=v1")

	v2Code := `bus.on("check", (msg) => { msg.reply({ version: "v2" }); });`

	resp2, err := packagemsg.CallPackageDeploy(env.Kit, ctx, packagemsg.PackageDeployMsg{
		Manifest: json.RawMessage(manifest),
		Files:    map[string]string{"index.ts": v2Code},
	}, sdk.WithCallTimeout(10*time.Second))
	require.NoError(t, err)
	require.True(t, resp2.Deployed)

	time.Sleep(200 * time.Millisecond)

	v2Resp := sendToServiceAndWait(t, env.Kit, "evolve-pkg", "check", nil)
	require.Equal(t, "v2", v2Resp["version"], "REDEPLOY BUG: v2 should return version=v2 but got %v", v2Resp["version"])
}

func testTopicCollision(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t, suite.WithPersistence())
	ctx := context.Background()

	dir := t.TempDir()
	writePackageFile(t, dir, "manifest.json", `{
		"name": "collision-test",
		"version": "1.0.0",
		"entry": "index.ts"
	}`)
	writePackageFile(t, dir, "index.ts", `
		bus.on("greet", (msg) => { msg.reply({ from: "first" }); });
		bus.on("greet", (msg) => { msg.reply({ from: "second" }); });
	`)

	_, err := packagemsg.CallPackageDeploy(env.Kit, ctx, packagemsg.PackageDeployMsg{Path: dir}, sdk.WithCallTimeout(10*time.Second))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already subscribed")
}

func sendToServiceAndWait(t *testing.T, k interface {
	sdk.Runtime
	sdk.CallerRuntime
}, service, topic string, payload any) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	data, err := json.Marshal(payload)
	require.NoError(t, err)
	resp, err := sdk.Call[sdk.CustomMsg, map[string]any](k, ctx, sdk.CustomMsg{
		Topic:   packageServiceTopic(service, topic),
		Payload: data,
	})
	require.NoError(t, err)
	return resp
}

func packageServiceTopic(service, topic string) string {
	name := strings.TrimSuffix(service, ".ts")
	name = strings.ReplaceAll(name, "/", ".")
	return "ts." + name + "." + topic
}
