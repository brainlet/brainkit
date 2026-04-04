package mcp

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("mcp", func(t *testing.T) {
		t.Run("list_tools", func(t *testing.T) { testListTools(t, env) })
		t.Run("call_tool", func(t *testing.T) { testCallTool(t, env) })
		t.Run("call_tool_via_registry", func(t *testing.T) { testCallToolViaRegistry(t, env) })
	})
}
