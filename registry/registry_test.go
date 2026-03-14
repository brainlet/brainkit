package registry

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

func TestRegisterAndResolveExact(t *testing.T) {
	r := New()
	r.Register(RegisteredTool{
		Name: "platform.grep", ShortName: "grep",
		Namespace: "platform", Executor: echoExecutor(),
	})

	tool, err := r.Resolve("platform.grep", "user")
	if err != nil {
		t.Fatal(err)
	}
	if tool.Name != "platform.grep" {
		t.Errorf("name = %q, want platform.grep", tool.Name)
	}
}

func TestResolveByShortName(t *testing.T) {
	r := New()
	r.Register(RegisteredTool{
		Name: "platform.grep", ShortName: "grep",
		Namespace: "platform", Executor: echoExecutor(),
	})

	tool, err := r.Resolve("grep", "user")
	if err != nil {
		t.Fatal(err)
	}
	if tool.Name != "platform.grep" {
		t.Errorf("name = %q, want platform.grep", tool.Name)
	}
}

func TestResolveNamespaceOrder(t *testing.T) {
	r := New()
	r.Register(RegisteredTool{
		Name: "platform.grep", ShortName: "grep",
		Namespace: "platform", Executor: echoExecutor(),
	})
	r.Register(RegisteredTool{
		Name: "user.grep", ShortName: "grep",
		Namespace: "user", Executor: echoExecutor(),
	})

	// user.grep should win for a user caller
	tool, _ := r.Resolve("grep", "user")
	if tool.Name != "user.grep" {
		t.Errorf("name = %q, want user.grep (user namespace wins)", tool.Name)
	}
}

func TestResolvePluginNamespace(t *testing.T) {
	r := New()
	r.Register(RegisteredTool{
		Name: "plugin.postgres@1.0.0.db_query", ShortName: "db_query",
		Namespace: "plugin.postgres@1.0.0", Executor: echoExecutor(),
	})

	// Fully qualified works
	tool, err := r.Resolve("plugin.postgres@1.0.0.db_query", "user")
	if err != nil {
		t.Fatal(err)
	}
	if tool.ShortName != "db_query" {
		t.Errorf("shortName = %q, want db_query", tool.ShortName)
	}

	// Short name resolves via plugin fallback
	tool2, err := r.Resolve("db_query", "user")
	if err != nil {
		t.Fatal(err)
	}
	if tool2.Name != "plugin.postgres@1.0.0.db_query" {
		t.Errorf("name = %q", tool2.Name)
	}
}

func TestResolveNotFound(t *testing.T) {
	r := New()
	_, err := r.Resolve("nonexistent", "user")
	if err == nil {
		t.Fatal("expected error for nonexistent tool")
	}
}

func TestUnregister(t *testing.T) {
	r := New()
	r.Register(RegisteredTool{
		Name: "platform.grep", ShortName: "grep",
		Namespace: "platform", Executor: echoExecutor(),
	})
	r.Unregister("platform.grep")

	_, err := r.Resolve("grep", "user")
	if err == nil {
		t.Fatal("expected error after unregister")
	}
}

func TestListByNamespace(t *testing.T) {
	r := New()
	r.Register(RegisteredTool{Name: "platform.grep", ShortName: "grep", Namespace: "platform", Executor: echoExecutor()})
	r.Register(RegisteredTool{Name: "platform.find", ShortName: "find", Namespace: "platform", Executor: echoExecutor()})
	r.Register(RegisteredTool{Name: "user.mytool", ShortName: "mytool", Namespace: "user", Executor: echoExecutor()})

	all := r.List("")
	if len(all) != 3 {
		t.Errorf("list all = %d, want 3", len(all))
	}
	platform := r.List("platform")
	if len(platform) != 2 {
		t.Errorf("list platform = %d, want 2", len(platform))
	}
}

func TestExecute(t *testing.T) {
	r := New()
	r.Register(RegisteredTool{
		Name: "platform.echo", ShortName: "echo", Namespace: "platform",
		Executor: &GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return json.RawMessage(`{"echoed":true}`), nil
			},
		},
	})

	tool, _ := r.Resolve("echo", "user")
	result, err := tool.Executor.Call(context.Background(), "user", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != `{"echoed":true}` {
		t.Errorf("result = %s", result)
	}
}
