package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateModulePackageUsesSDKAndModuleReferences(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "messages.go"), `package demo

type PingMsg struct{}

func (PingMsg) BusTopic() string { return "demo.ping" }

type PingResp struct {
	OK bool
}

type TickEvent struct{}

func (TickEvent) BusTopic() string { return "demo.tick" }
`)

	packageName, types, err := scanMessages(dir)
	if err != nil {
		t.Fatalf("scanMessages() error = %v", err)
	}
	if packageName != "demo" {
		t.Fatalf("packageName = %q, want demo", packageName)
	}
	if len(types) != 2 {
		t.Fatalf("len(types) = %d, want 2: %#v", len(types), types)
	}

	out := filepath.Join(t.TempDir(), "modules", "demo", "typed_gen.go")
	if err := generate(out, packageName, types); err != nil {
		t.Fatalf("generate() error = %v", err)
	}

	generated := readFile(t, out)
	assertContains(t, generated, "package demo")
	assertContains(t, generated, `bkmodule "github.com/brainlet/brainkit/module"`)
	assertContains(t, generated, "\"github.com/brainlet/brainkit/sdk\"")
	assertNotContains(t, generated, "func PublishPing")
	assertNotContains(t, generated, "func SubscribePingResp")
	assertContains(t, generated, "func CallPing(rt sdk.CallerRuntime, ctx context.Context, msg PingMsg, opts ...sdk.CallOption) (PingResp, error)")
	assertContains(t, generated, "return sdk.Call[PingMsg, PingResp](rt, ctx, msg, opts...)")
	assertContains(t, generated, "func CallPingWithCaller(caller bkmodule.RequestCaller, ctx context.Context, msg PingMsg, opts ...sdk.CallOption) (PingResp, error)")
	assertContains(t, generated, "return sdk.CallWithCaller[PingMsg, PingResp](caller, ctx, msg, opts...)")
	assertContains(t, generated, "func EmitTick(rt sdk.Runtime, ctx context.Context, msg TickEvent) error")
	assertContains(t, generated, "func SubscribeTick(rt sdk.Runtime, ctx context.Context, topic string, handler func(TickEvent, sdk.Message)) (func(), error)")
}

func TestGenerateNonModulePackageUsesSDKRequestCaller(t *testing.T) {
	types := []msgType{
		{Name: "PingMsg", BaseName: "Ping", RespName: "PingResp"},
	}
	out := filepath.Join(t.TempDir(), "sdk", "systemmsg", "typed_gen.go")
	if err := generate(out, "systemmsg", types); err != nil {
		t.Fatalf("generate() error = %v", err)
	}

	generated := readFile(t, out)
	assertContains(t, generated, "package systemmsg")
	assertNotContains(t, generated, "github.com/brainlet/brainkit/module")
	assertContains(t, generated, "func CallPingWithCaller(caller sdk.RequestCaller, ctx context.Context, msg PingMsg, opts ...sdk.CallOption) (PingResp, error)")
}

func TestGenerateSDKPackageUsesLocalReferences(t *testing.T) {
	types := []msgType{
		{Name: "PingMsg", BaseName: "Ping", RespName: "PingResp"},
		{Name: "TickEvent", BaseName: "Tick", IsEvent: true},
	}
	out := filepath.Join(t.TempDir(), "typed_gen.go")
	if err := generate(out, "sdk", types); err != nil {
		t.Fatalf("generate() error = %v", err)
	}

	generated := readFile(t, out)
	assertContains(t, generated, "package sdk")
	assertNotContains(t, generated, "\"github.com/brainlet/brainkit/sdk\"")
	assertNotContains(t, generated, "func PublishPing")
	assertNotContains(t, generated, "func SubscribePingResp")
	assertContains(t, generated, "func CallPing(rt CallerRuntime, ctx context.Context, msg PingMsg, opts ...CallOption) (PingResp, error)")
	assertContains(t, generated, "return Call[PingMsg, PingResp](rt, ctx, msg, opts...)")
	assertContains(t, generated, "func CallPingWithCaller(caller RequestCaller, ctx context.Context, msg PingMsg, opts ...CallOption) (PingResp, error)")
	assertContains(t, generated, "return CallWithCaller[PingMsg, PingResp](caller, ctx, msg, opts...)")
	assertContains(t, generated, "func SubscribeTick(rt Runtime, ctx context.Context, topic string, handler func(TickEvent, Message)) (func(), error)")
}

func TestGenerateEmptyPackageOmitsImports(t *testing.T) {
	out := filepath.Join(t.TempDir(), "typed_gen.go")
	if err := generate(out, "sdk", nil); err != nil {
		t.Fatalf("generate() error = %v", err)
	}

	generated := readFile(t, out)
	assertContains(t, generated, "package sdk")
	assertNotContains(t, generated, "import ")
	assertNotContains(t, generated, "context")
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("generated output does not contain %q:\n%s", needle, haystack)
	}
}

func assertNotContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Fatalf("generated output unexpectedly contains %q:\n%s", needle, haystack)
	}
}
