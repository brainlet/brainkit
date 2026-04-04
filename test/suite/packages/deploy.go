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
		"services": {
			"greeter": { "entry": "greeter.ts" }
		}
	}`)
	writePackageFile(t, dir, "config.ts", `export const PREFIX = "Hello";`)
	writePackageFile(t, dir, "greeter.ts", `
		import { PREFIX } from "./config";
		bus.on("greet", (msg) => {
			msg.reply({ text: PREFIX + " " + msg.payload.name });
		});
	`)

	pub, err := sdk.Publish(env.Kernel, ctx, messages.PackageDeployMsg{Path: dir})
	require.NoError(t, err)

	deployCh := make(chan messages.PackageDeployResp, 1)
	cancel, _ := sdk.SubscribeTo[messages.PackageDeployResp](env.Kernel, ctx, pub.ReplyTo, func(resp messages.PackageDeployResp, _ messages.Message) { deployCh <- resp })
	defer cancel()

	select {
	case resp := <-deployCh:
		require.True(t, resp.Deployed)
		assert.Equal(t, "test-pkg", resp.Name)
		assert.Len(t, resp.Services, 1)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for package deploy")
	}

	time.Sleep(200 * time.Millisecond)

	sendPR, _ := sdk.SendToService(env.Kernel, ctx, "test-pkg/greeter.ts", "greet", map[string]string{"name": "World"})
	replyCh := make(chan map[string]any, 1)
	replyCancel, _ := env.Kernel.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) {
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
		"services": { "svc": { "entry": "svc.ts" } }
	}`)
	writePackageFile(t, dir, "svc.ts", `bus.on("ping", (msg) => { msg.reply({pong: true}); });`)

	pub, _ := sdk.Publish(env.Kernel, ctx, messages.PackageDeployMsg{Path: dir})
	ch := make(chan messages.PackageDeployResp, 1)
	cancel, _ := sdk.SubscribeTo[messages.PackageDeployResp](env.Kernel, ctx, pub.ReplyTo, func(resp messages.PackageDeployResp, _ messages.Message) { ch <- resp })
	<-ch
	cancel()

	pub2, _ := sdk.Publish(env.Kernel, ctx, messages.PackageListDeployedMsg{})
	listCh := make(chan messages.PackageListDeployedResp, 1)
	cancel2, _ := sdk.SubscribeTo[messages.PackageListDeployedResp](env.Kernel, ctx, pub2.ReplyTo, func(resp messages.PackageListDeployedResp, _ messages.Message) { listCh <- resp })

	select {
	case resp := <-listCh:
		cancel2()
		require.Len(t, resp.Packages, 1)
		assert.Equal(t, "list-test", resp.Packages[0].Name)
	case <-time.After(5 * time.Second):
		cancel2()
		t.Fatal("timeout listing packages")
	}

	pub3, _ := sdk.Publish(env.Kernel, ctx, messages.PackageTeardownMsg{Name: "list-test"})
	tearCh := make(chan messages.PackageTeardownResp, 1)
	cancel3, _ := sdk.SubscribeTo[messages.PackageTeardownResp](env.Kernel, ctx, pub3.ReplyTo, func(resp messages.PackageTeardownResp, _ messages.Message) { tearCh <- resp })
	select {
	case resp := <-tearCh:
		cancel3()
		assert.True(t, resp.Removed)
	case <-time.After(5 * time.Second):
		cancel3()
		t.Fatal("timeout")
	}

	pub4, _ := sdk.Publish(env.Kernel, ctx, messages.PackageListDeployedMsg{})
	listCh2 := make(chan messages.PackageListDeployedResp, 1)
	cancel4, _ := sdk.SubscribeTo[messages.PackageListDeployedResp](env.Kernel, ctx, pub4.ReplyTo, func(resp messages.PackageListDeployedResp, _ messages.Message) { listCh2 <- resp })
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
		"services": { "svc": { "entry": "svc.ts" } },
		"requires": { "secrets": ["MY_REQUIRED_SECRET"] }
	}`)
	writePackageFile(t, dir, "svc.ts", `bus.on("x", (msg) => { msg.reply({}); });`)

	pub, _ := sdk.Publish(env.Kernel, ctx, messages.PackageDeployMsg{Path: dir})
	errCh := make(chan string, 1)
	cancel, _ := env.Kernel.SubscribeRaw(ctx, pub.ReplyTo, func(msg messages.Message) {
		var resp map[string]any
		json.Unmarshal(msg.Payload, &resp)
		if e, ok := resp["error"].(string); ok { errCh <- e }
	})
	defer cancel()

	select {
	case errMsg := <-errCh:
		assert.Contains(t, errMsg, "MY_REQUIRED_SECRET")
		assert.Contains(t, errMsg, "not set")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}

	setPub, _ := sdk.Publish(env.Kernel, ctx, messages.SecretsSetMsg{Name: "MY_REQUIRED_SECRET", Value: "secret-value"})
	setCh := make(chan messages.SecretsSetResp, 1)
	setCancel, _ := sdk.SubscribeTo[messages.SecretsSetResp](env.Kernel, ctx, setPub.ReplyTo, func(resp messages.SecretsSetResp, _ messages.Message) { setCh <- resp })
	<-setCh
	setCancel()

	pub2, _ := sdk.Publish(env.Kernel, ctx, messages.PackageDeployMsg{Path: dir})
	deployCh := make(chan messages.PackageDeployResp, 1)
	cancel2, _ := sdk.SubscribeTo[messages.PackageDeployResp](env.Kernel, ctx, pub2.ReplyTo, func(resp messages.PackageDeployResp, _ messages.Message) { deployCh <- resp })
	defer cancel2()

	select {
	case resp := <-deployCh:
		assert.True(t, resp.Deployed)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout")
	}
}
