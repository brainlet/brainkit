package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func echoExecutor() ToolExecutor {
	return &GoFuncExecutor{
		Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
			return input, nil
		},
	}
}

// ============================================================
// Core tests — new format only
// ============================================================

func TestRegisterAndResolveExact(t *testing.T) {
	r := New()
	r.Register(RegisteredTool{
		Name: "brainlet/platform@1.0.0/grep", ShortName: "grep",
		Owner: "brainlet", Package: "platform", Version: "1.0.0",
		Executor: echoExecutor(),
	})
	tool, err := r.Resolve("brainlet/platform@1.0.0/grep")
	if err != nil {
		t.Fatal(err)
	}
	if tool.Name != "brainlet/platform@1.0.0/grep" {
		t.Errorf("name = %q", tool.Name)
	}
}

func TestResolveByShortName(t *testing.T) {
	r := New()
	r.Register(RegisteredTool{
		Name: "brainlet/platform@1.0.0/grep", ShortName: "grep",
		Owner: "brainlet", Package: "platform", Version: "1.0.0",
		Executor: echoExecutor(),
	})
	tool, err := r.Resolve("grep")
	if err != nil {
		t.Fatal(err)
	}
	if tool.Name != "brainlet/platform@1.0.0/grep" {
		t.Errorf("name = %q", tool.Name)
	}
}

func TestResolveNotFound(t *testing.T) {
	r := New()
	_, err := r.Resolve("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUnregister(t *testing.T) {
	r := New()
	r.Register(RegisteredTool{
		Name: "brainlet/platform@1.0.0/grep", ShortName: "grep",
		Owner: "brainlet", Package: "platform", Version: "1.0.0",
		Executor: echoExecutor(),
	})
	r.Unregister("brainlet/platform@1.0.0/grep")
	_, err := r.Resolve("grep")
	if err == nil {
		t.Fatal("expected error after unregister")
	}
}

func TestExecute(t *testing.T) {
	r := New()
	r.Register(RegisteredTool{
		Name: "brainlet/platform@1.0.0/echo", ShortName: "echo",
		Owner: "brainlet", Package: "platform", Version: "1.0.0",
		Executor: &GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return json.RawMessage(`{"echoed":true}`), nil
			},
		},
	})
	tool, _ := r.Resolve("echo")
	result, err := tool.Executor.Call(context.Background(), "user", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != `{"echoed":true}` {
		t.Errorf("result = %s", result)
	}
}

// ============================================================
// New format tests — owner/package@version/tool
// ============================================================

func TestResolve_Level1_ExactMatch(t *testing.T) {
	r := New()
	r.Register(RegisteredTool{
		Name: "brainlet/cron@1.0.0/create", Owner: "brainlet",
		Package: "cron", Version: "1.0.0", ShortName: "create", Executor: echoExecutor(),
	})
	tool, err := r.Resolve("brainlet/cron@1.0.0/create")
	if err != nil {
		t.Fatal(err)
	}
	if tool.Owner != "brainlet" || tool.Package != "cron" || tool.Version != "1.0.0" {
		t.Errorf("fields = %q/%q@%q", tool.Owner, tool.Package, tool.Version)
	}
}

func TestResolve_Level2_NoOwner(t *testing.T) {
	r := New()
	r.Register(RegisteredTool{
		Name: "brainlet/cron@1.0.0/create", Owner: "brainlet",
		Package: "cron", Version: "1.0.0", ShortName: "create", Executor: echoExecutor(),
	})
	tool, err := r.Resolve("cron@1.0.0/create")
	if err != nil {
		t.Fatal(err)
	}
	if tool.Name != "brainlet/cron@1.0.0/create" {
		t.Errorf("name = %q", tool.Name)
	}
}

func TestResolve_Level3_NoVersion(t *testing.T) {
	r := New()
	r.Register(RegisteredTool{
		Name: "brainlet/cron@1.0.0/create", Owner: "brainlet",
		Package: "cron", Version: "1.0.0", ShortName: "create", Executor: echoExecutor(),
	})
	r.Register(RegisteredTool{
		Name: "brainlet/cron@2.0.0/create", Owner: "brainlet",
		Package: "cron", Version: "2.0.0", ShortName: "create", Executor: echoExecutor(),
	})
	r.Register(RegisteredTool{
		Name: "brainlet/cron@2.1.0-beta.1/create", Owner: "brainlet",
		Package: "cron", Version: "2.1.0-beta.1", ShortName: "create", Executor: echoExecutor(),
	})
	tool, err := r.Resolve("brainlet/cron/create")
	if err != nil {
		t.Fatal(err)
	}
	if tool.Version != "2.0.0" {
		t.Errorf("version = %q, want 2.0.0", tool.Version)
	}
}

func TestResolve_Level4_Bare(t *testing.T) {
	r := New()
	r.Register(RegisteredTool{
		Name: "brainlet/cron@1.0.0/create", Owner: "brainlet",
		Package: "cron", Version: "1.0.0", ShortName: "create", Executor: echoExecutor(),
	})
	tool, err := r.Resolve("cron/create")
	if err != nil {
		t.Fatal(err)
	}
	if tool.Owner != "brainlet" || tool.Package != "cron" {
		t.Errorf("got %q/%q", tool.Owner, tool.Package)
	}
}

func TestResolve_Level4_Bare_ThirdParty(t *testing.T) {
	r := New()
	r.Register(RegisteredTool{
		Name: "acme/cron@1.0.0/create", Owner: "acme",
		Package: "cron", Version: "1.0.0", ShortName: "create", Executor: echoExecutor(),
	})
	tool, err := r.Resolve("cron/create")
	if err != nil {
		t.Fatal(err)
	}
	if tool.Owner != "acme" {
		t.Errorf("owner = %q, want acme", tool.Owner)
	}
}

func TestResolve_Level5_ShortName(t *testing.T) {
	r := New()
	r.Register(RegisteredTool{
		Name: "brainlet/cron@1.0.0/create", Owner: "brainlet",
		Package: "cron", Version: "1.0.0", ShortName: "create", Executor: echoExecutor(),
	})
	tool, err := r.Resolve("create")
	if err != nil {
		t.Fatal(err)
	}
	if tool.Name != "brainlet/cron@1.0.0/create" {
		t.Errorf("name = %q", tool.Name)
	}
}

func TestResolve_Level2_ThirdPartyRequiresOwner(t *testing.T) {
	r := New()
	r.Register(RegisteredTool{
		Name: "acme-corp/cron@1.0.0/create", Owner: "acme-corp",
		Package: "cron", Version: "1.0.0", ShortName: "create", Executor: echoExecutor(),
	})
	_, err := r.Resolve("cron@1.0.0/create")
	if err == nil {
		t.Fatal("expected error: no-owner defaults to brainlet")
	}
	tool, err := r.Resolve("acme-corp/cron@1.0.0/create")
	if err != nil {
		t.Fatal(err)
	}
	if tool.Owner != "acme-corp" {
		t.Errorf("owner = %q", tool.Owner)
	}
}

func TestRegister_AutoPopulatesNewFormatFields(t *testing.T) {
	r := New()
	r.Register(RegisteredTool{Name: "brainlet/postgres@2.1.0/query", Executor: echoExecutor()})
	tool, err := r.Resolve("brainlet/postgres@2.1.0/query")
	if err != nil {
		t.Fatal(err)
	}
	if tool.Owner != "brainlet" || tool.Package != "postgres" || tool.Version != "2.1.0" || tool.ShortName != "query" {
		t.Errorf("auto = %q/%q@%q/%q", tool.Owner, tool.Package, tool.Version, tool.ShortName)
	}
}

func TestList_ByOwner(t *testing.T) {
	r := New()
	r.Register(RegisteredTool{Name: "brainlet/cron@1.0.0/create", Owner: "brainlet", Package: "cron", Version: "1.0.0", ShortName: "create", Executor: echoExecutor()})
	r.Register(RegisteredTool{Name: "brainlet/cron@1.0.0/list", Owner: "brainlet", Package: "cron", Version: "1.0.0", ShortName: "list", Executor: echoExecutor()})
	r.Register(RegisteredTool{Name: "acme/cron@1.0.0/create", Owner: "acme", Package: "cron", Version: "1.0.0", ShortName: "create", Executor: echoExecutor()})
	r.Register(RegisteredTool{Name: "brainlet/platform@1.0.0/echo", Owner: "brainlet", Package: "platform", Version: "1.0.0", ShortName: "echo", Executor: echoExecutor()})

	if n := len(r.List("brainlet")); n != 3 {
		t.Errorf("brainlet = %d, want 3", n)
	}
	if n := len(r.List("brainlet/cron")); n != 2 {
		t.Errorf("brainlet/cron = %d, want 2", n)
	}
	if n := len(r.List("platform")); n != 1 {
		t.Errorf("platform (package) = %d, want 1", n)
	}
	if n := len(r.List("")); n != 4 {
		t.Errorf("all = %d, want 4", n)
	}
}
