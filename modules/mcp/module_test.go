package mcp

import (
	"context"
	"testing"

	"github.com/brainlet/brainkit"
	toolsmod "github.com/brainlet/brainkit/modules/tools"
)

func TestMountAutoMountsToolsDependency(t *testing.T) {
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test-mcp",
	})
	if err != nil {
		t.Fatalf("brainkit.New: %v", err)
	}
	t.Cleanup(func() { _ = k.Close() })

	if _, ok := k.Module("tools"); ok {
		t.Fatalf("tools should not be mounted before mcp dependency resolution")
	}

	if err := k.Mount(context.Background(), New(nil)); err != nil {
		t.Fatalf("Mount(mcp): %v", err)
	}

	if _, ok := k.Module("tools"); !ok {
		t.Fatalf("mcp mount should auto-mount tools dependency")
	}
}

func TestStartupOrdersExplicitToolsDependencyBeforeMCP(t *testing.T) {
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test-mcp",
		Modules: []brainkit.Module{
			New(nil),
			toolsmod.New(),
		},
	})
	if err != nil {
		t.Fatalf("brainkit.New: %v", err)
	}
	t.Cleanup(func() { _ = k.Close() })

	if _, ok := k.Module("tools"); !ok {
		t.Fatalf("tools should be mounted from explicit dependency")
	}
	if _, ok := k.Module("mcp"); !ok {
		t.Fatalf("mcp should be mounted after tools")
	}
}
