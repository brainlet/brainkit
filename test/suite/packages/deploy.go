package packages

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
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

	pub, err := sdk.Publish(env.Kit, ctx, messages.PackageDeployMsg{Path: dir})
	require.NoError(t, err)

	deployCh := make(chan messages.PackageDeployResp, 1)
	cancel, _ := sdk.SubscribeTo[messages.PackageDeployResp](env.Kit, ctx, pub.ReplyTo, func(resp messages.PackageDeployResp, _ messages.Message) { deployCh <- resp })
	defer cancel()

	select {
	case resp := <-deployCh:
		require.True(t, resp.Deployed)
		assert.Equal(t, "test-pkg", resp.Name)
		assert.Equal(t, "test-pkg.ts", resp.Source)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for package deploy")
	}

	time.Sleep(200 * time.Millisecond)

	sendPR, _ := sdk.SendToService(env.Kit, ctx, "test-pkg", "greet", map[string]string{"name": "World"})
	replyCh := make(chan map[string]any, 1)
	replyCancel, _ := env.Kit.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) {
		var resp map[string]any
		json.Unmarshal(msg.Payload, &resp)
		replyCh <- resp
	})
	defer replyCancel()

	select {
	case resp := <-replyCh:
		assert.Equal(t, "Hello World", resp["text"])
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for greeter response")
	}
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

	pub, _ := sdk.Publish(env.Kit, ctx, messages.PackageDeployMsg{Path: dir})
	ch := make(chan messages.PackageDeployResp, 1)
	cancel, _ := sdk.SubscribeTo[messages.PackageDeployResp](env.Kit, ctx, pub.ReplyTo, func(resp messages.PackageDeployResp, _ messages.Message) { ch <- resp })
	<-ch
	cancel()

	pub2, _ := sdk.Publish(env.Kit, ctx, messages.PackageListDeployedMsg{})
	listCh := make(chan messages.PackageListDeployedResp, 1)
	cancel2, _ := sdk.SubscribeTo[messages.PackageListDeployedResp](env.Kit, ctx, pub2.ReplyTo, func(resp messages.PackageListDeployedResp, _ messages.Message) { listCh <- resp })

	select {
	case resp := <-listCh:
		cancel2()
		require.Len(t, resp.Packages, 1)
		assert.Equal(t, "list-test", resp.Packages[0].Name)
		assert.Equal(t, "list-test.ts", resp.Packages[0].Source)
	case <-time.After(5 * time.Second):
		cancel2()
		t.Fatal("timeout listing packages")
	}

	pub3, _ := sdk.Publish(env.Kit, ctx, messages.PackageTeardownMsg{Name: "list-test"})
	tearCh := make(chan messages.PackageTeardownResp, 1)
	cancel3, _ := sdk.SubscribeTo[messages.PackageTeardownResp](env.Kit, ctx, pub3.ReplyTo, func(resp messages.PackageTeardownResp, _ messages.Message) { tearCh <- resp })
	select {
	case resp := <-tearCh:
		cancel3()
		assert.True(t, resp.Removed)
	case <-time.After(5 * time.Second):
		cancel3()
		t.Fatal("timeout")
	}

	pub4, _ := sdk.Publish(env.Kit, ctx, messages.PackageListDeployedMsg{})
	listCh2 := make(chan messages.PackageListDeployedResp, 1)
	cancel4, _ := sdk.SubscribeTo[messages.PackageListDeployedResp](env.Kit, ctx, pub4.ReplyTo, func(resp messages.PackageListDeployedResp, _ messages.Message) { listCh2 <- resp })
	defer cancel4()
	select {
	case resp := <-listCh2:
		assert.Len(t, resp.Packages, 0)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
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

	pub, _ := sdk.Publish(env.Kit, ctx, messages.PackageDeployMsg{Path: dir})
	errCh := make(chan string, 1)
	cancel, _ := env.Kit.SubscribeRaw(ctx, pub.ReplyTo, func(msg messages.Message) {
		var resp map[string]any
		json.Unmarshal(msg.Payload, &resp)
		if e, ok := resp["error"].(string); ok {
			errCh <- e
		}
	})
	defer cancel()

	select {
	case errMsg := <-errCh:
		assert.Contains(t, errMsg, "MY_REQUIRED_SECRET")
		assert.Contains(t, errMsg, "not set")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}

	setPub, _ := sdk.Publish(env.Kit, ctx, messages.SecretsSetMsg{Name: "MY_REQUIRED_SECRET", Value: "secret-value"})
	setCh := make(chan messages.SecretsSetResp, 1)
	setCancel, _ := sdk.SubscribeTo[messages.SecretsSetResp](env.Kit, ctx, setPub.ReplyTo, func(resp messages.SecretsSetResp, _ messages.Message) { setCh <- resp })
	<-setCh
	setCancel()

	pub2, _ := sdk.Publish(env.Kit, ctx, messages.PackageDeployMsg{Path: dir})
	deployCh := make(chan messages.PackageDeployResp, 1)
	cancel2, _ := sdk.SubscribeTo[messages.PackageDeployResp](env.Kit, ctx, pub2.ReplyTo, func(resp messages.PackageDeployResp, _ messages.Message) { deployCh <- resp })
	defer cancel2()

	select {
	case resp := <-deployCh:
		assert.True(t, resp.Deployed)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout")
	}
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

	pub, err := sdk.Publish(env.Kit, ctx, messages.PackageDeployMsg{
		Manifest: json.RawMessage(manifest),
		Files:    map[string]string{"index.ts": v1Code},
	})
	require.NoError(t, err)

	deployCh := make(chan messages.PackageDeployResp, 1)
	cancel, _ := sdk.SubscribeTo[messages.PackageDeployResp](env.Kit, ctx, pub.ReplyTo,
		func(resp messages.PackageDeployResp, _ messages.Message) { deployCh <- resp })
	select {
	case resp := <-deployCh:
		cancel()
		require.True(t, resp.Deployed)
		t.Logf("v1 deployed: %s", resp.Source)
	case <-time.After(10 * time.Second):
		cancel()
		t.Fatal("timeout deploying v1")
	}

	time.Sleep(200 * time.Millisecond)

	v1Resp := sendToServiceAndWait(t, env.Kit, "evolve-pkg", "check", nil)
	require.Equal(t, "v1", v1Resp["version"], "v1 should return version=v1")

	v2Code := `bus.on("check", (msg) => { msg.reply({ version: "v2" }); });`

	pub2, err := sdk.Publish(env.Kit, ctx, messages.PackageDeployMsg{
		Manifest: json.RawMessage(manifest),
		Files:    map[string]string{"index.ts": v2Code},
	})
	require.NoError(t, err)

	// Listen for raw response to capture errors
	v2ReplyCh := make(chan messages.Message, 1)
	cancel2, _ := env.Kit.SubscribeRaw(ctx, pub2.ReplyTo, func(msg messages.Message) {
		select {
		case v2ReplyCh <- msg:
		default:
		}
	})
	select {
	case msg := <-v2ReplyCh:
		cancel2()
		var resp messages.PackageDeployResp
		json.Unmarshal(msg.Payload, &resp)
		if resp.Error != "" {
			t.Fatalf("v2 deploy error: %s", resp.Error)
		}
		require.True(t, resp.Deployed)
	case <-time.After(10 * time.Second):
		cancel2()
		t.Fatal("timeout deploying v2")
	}

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

	pub, _ := sdk.Publish(env.Kit, ctx, messages.PackageDeployMsg{Path: dir})
	errCh := make(chan string, 1)
	cancel, _ := env.Kit.SubscribeRaw(ctx, pub.ReplyTo, func(msg messages.Message) {
		var resp map[string]any
		json.Unmarshal(msg.Payload, &resp)
		if e, ok := resp["error"].(string); ok {
			errCh <- e
		}
	})
	defer cancel()

	select {
	case errMsg := <-errCh:
		assert.Contains(t, errMsg, "already subscribed")
	case <-time.After(10 * time.Second):
		t.Fatal("expected topic collision error")
	}
}

func sendToServiceAndWait(t *testing.T, k interface {
	sdk.Runtime
	SubscribeRaw(context.Context, string, func(messages.Message)) (func(), error)
}, service, topic string, payload any) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, err := sdk.SendToService(k.(sdk.Runtime), ctx, service, topic, payload)
	require.NoError(t, err)

	replyCh := make(chan map[string]any, 1)
	unsub, err := k.SubscribeRaw(ctx, pr.ReplyTo, func(msg messages.Message) {
		var resp map[string]any
		json.Unmarshal(msg.Payload, &resp)
		select {
		case replyCh <- resp:
		default:
		}
	})
	require.NoError(t, err)
	defer unsub()

	select {
	case resp := <-replyCh:
		return resp
	case <-ctx.Done():
		t.Fatalf("timeout waiting for response from %s/%s", service, topic)
		return nil
	}
}
