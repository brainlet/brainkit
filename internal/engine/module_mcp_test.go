package engine

import (
	"testing"

	"github.com/brainlet/brainkit/internal/types"
)

func TestMCPModuleName(t *testing.T) {
	m := NewMCPModule(nil)
	if m.Name() != "mcp" {
		t.Fatalf("expected name 'mcp', got %q", m.Name())
	}
}

func TestMCPModuleInitNoServers(t *testing.T) {
	m := NewMCPModule(nil)
	cat := buildCommandCatalog()
	k := &Kernel{catalog: cat}

	initialCount := len(cat.ordered)

	if err := m.Init(k); err != nil {
		t.Fatalf("Init with no servers should not error: %v", err)
	}

	// No commands registered when no servers configured
	if len(cat.ordered) != initialCount {
		t.Fatalf("expected %d commands (no MCP), got %d", initialCount, len(cat.ordered))
	}
}

func TestMCPModuleCloseNilManager(t *testing.T) {
	m := NewMCPModule(nil)
	if err := m.Close(); err != nil {
		t.Fatalf("Close with nil manager should not error: %v", err)
	}
}

func TestMCPModuleInitRegistersCommands(t *testing.T) {
	servers := map[string]types.MCPServerConfig{
		// Bogus server — connect will fail, but commands still register.
		"test": {URL: "http://localhost:0/nonexistent"},
	}
	m := NewMCPModule(servers)
	cat := buildCommandCatalog()
	initialCount := len(cat.ordered)

	k := &Kernel{
		catalog: cat,
		config:  types.KernelConfig{},
	}

	_ = m.Init(k) // connect fails, but commands still register

	if len(cat.ordered) != initialCount+2 {
		t.Fatalf("expected %d commands (initial + 2 MCP), got %d", initialCount+2, len(cat.ordered))
	}
	if !cat.HasCommand("mcp.listTools") {
		t.Fatal("mcp.listTools not registered")
	}
	if !cat.HasCommand("mcp.callTool") {
		t.Fatal("mcp.callTool not registered")
	}
}
