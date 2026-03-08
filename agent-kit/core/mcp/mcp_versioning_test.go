// Ported from: packages/core/src/mcp/mcp-versioning.test.ts
//
// NOTE: The original TS tests rely on the Mastra class and its getMCPServerById
// method which is not yet ported to Go. Tests that require the full Mastra class
// are skipped. Tests for MCPServerBase functionality that is independently
// testable are included.
package mcp

import (
	"testing"
)

func TestMCPServerBase_NewMCPServerBase(t *testing.T) {
	t.Run("should set name and version", func(t *testing.T) {
		server := NewMCPServerBase(MCPServerConfig{
			Name:    "test-server",
			Version: "1.0.0",
		})
		if server.Name != "test-server" {
			t.Errorf("expected Name=test-server, got %s", server.Name)
		}
		if server.Version != "1.0.0" {
			t.Errorf("expected Version=1.0.0, got %s", server.Version)
		}
	})

	t.Run("should generate UUID when no ID provided", func(t *testing.T) {
		server := NewMCPServerBase(MCPServerConfig{
			Name:    "no-id",
			Version: "1.0.0",
		})
		if server.ID() == "" {
			t.Error("expected auto-generated ID")
		}
	})

	t.Run("should slugify provided ID", func(t *testing.T) {
		server := NewMCPServerBase(MCPServerConfig{
			Name:    "slug-test",
			Version: "1.0.0",
			ID:      "My Server Name",
		})
		if server.ID() != "my-server-name" {
			t.Errorf("expected slugified ID=my-server-name, got %s", server.ID())
		}
	})

	t.Run("should default isLatest to true", func(t *testing.T) {
		server := NewMCPServerBase(MCPServerConfig{
			Name:    "latest-test",
			Version: "1.0.0",
		})
		if !server.IsLatest {
			t.Error("expected IsLatest=true by default")
		}
	})

	t.Run("should respect isLatest=false", func(t *testing.T) {
		f := false
		server := NewMCPServerBase(MCPServerConfig{
			Name:     "not-latest",
			Version:  "1.0.0",
			IsLatest: &f,
		})
		if server.IsLatest {
			t.Error("expected IsLatest=false")
		}
	})

	t.Run("should set release date from config", func(t *testing.T) {
		server := NewMCPServerBase(MCPServerConfig{
			Name:        "date-test",
			Version:     "1.0.0",
			ReleaseDate: "2024-01-15T00:00:00Z",
		})
		if server.ReleaseDate != "2024-01-15T00:00:00Z" {
			t.Errorf("expected ReleaseDate='2024-01-15T00:00:00Z', got %s", server.ReleaseDate)
		}
	})

	t.Run("should auto-generate release date when not provided", func(t *testing.T) {
		server := NewMCPServerBase(MCPServerConfig{
			Name:    "auto-date",
			Version: "1.0.0",
		})
		if server.ReleaseDate == "" {
			t.Error("expected auto-generated ReleaseDate")
		}
	})

	t.Run("should store description and instructions", func(t *testing.T) {
		server := NewMCPServerBase(MCPServerConfig{
			Name:         "desc-test",
			Version:      "1.0.0",
			Description:  "A test server",
			Instructions: "Use this server for testing",
		})
		if server.Description != "A test server" {
			t.Errorf("expected Description='A test server', got %s", server.Description)
		}
		if server.Instructions != "Use this server for testing" {
			t.Errorf("expected Instructions match, got %s", server.Instructions)
		}
	})
}

func TestMCPServerBase_SetID(t *testing.T) {
	t.Run("should set ID when not already set", func(t *testing.T) {
		server := NewMCPServerBase(MCPServerConfig{
			Name:    "setid-test",
			Version: "1.0.0",
		})
		originalID := server.ID()

		server.SetID("new-id")
		if server.ID() != "new-id" {
			t.Errorf("expected ID=new-id, got %s", server.ID())
		}
		_ = originalID
	})

	t.Run("should not override when ID was explicitly set", func(t *testing.T) {
		server := NewMCPServerBase(MCPServerConfig{
			Name:    "setid-locked",
			Version: "1.0.0",
			ID:      "explicit-id",
		})

		server.SetID("should-not-change")
		if server.ID() != "explicit-id" {
			t.Errorf("expected ID=explicit-id (unchanged), got %s", server.ID())
		}
	})
}

func TestMCPServerBase_Tools(t *testing.T) {
	t.Run("should return empty map initially", func(t *testing.T) {
		server := NewMCPServerBase(MCPServerConfig{
			Name:    "tools-test",
			Version: "1.0.0",
		})
		tools := server.Tools()
		if tools == nil {
			t.Fatal("expected non-nil tools map")
		}
		if len(tools) != 0 {
			t.Errorf("expected empty tools map, got %d entries", len(tools))
		}
	})

	t.Run("should return copy (mutation safety)", func(t *testing.T) {
		server := NewMCPServerBase(MCPServerConfig{
			Name:    "tools-copy",
			Version: "1.0.0",
		})
		tools := server.Tools()
		tools["hacked"] = "should not persist"

		toolsAgain := server.Tools()
		if _, ok := toolsAgain["hacked"]; ok {
			t.Error("expected Tools() to return a copy, not a mutable reference")
		}
	})
}

func TestSlugify(t *testing.T) {
	t.Run("should lowercase and replace spaces", func(t *testing.T) {
		if slugify("Hello World") != "hello-world" {
			t.Errorf("expected hello-world, got %s", slugify("Hello World"))
		}
	})

	t.Run("should replace underscores", func(t *testing.T) {
		if slugify("hello_world") != "hello-world" {
			t.Errorf("expected hello-world, got %s", slugify("hello_world"))
		}
	})

	t.Run("should remove special characters", func(t *testing.T) {
		if slugify("Hello@World!") != "helloworld" {
			t.Errorf("expected helloworld, got %s", slugify("Hello@World!"))
		}
	})

	t.Run("should collapse multiple hyphens", func(t *testing.T) {
		if slugify("hello---world") != "hello-world" {
			t.Errorf("expected hello-world, got %s", slugify("hello---world"))
		}
	})

	t.Run("should trim leading/trailing hyphens", func(t *testing.T) {
		if slugify("-hello-") != "hello" {
			t.Errorf("expected hello, got %s", slugify("-hello-"))
		}
	})
}

// The following tests from the TS source require the Mastra class which
// is not yet ported to Go. They are left as skipped stubs for documentation.

func TestMCPVersioning_MastraGetMCPServer(t *testing.T) {
	t.Skip("requires Mastra class not yet ported to Go")

	// In the TS source, these tests verify:
	// - getMCPServer returns undefined when no servers registered
	// - getMCPServer returns undefined for unknown logical ID
	// - getMCPServer fetches specific version by logical ID + version string
	// - getMCPServer returns undefined for missing version
	// - getMCPServer fetches latest by releaseDate when no version specified
	// - getMCPServer handles invalid dates (falls back to first found)
	// - getMCPServer warns when all dates are invalid
}
