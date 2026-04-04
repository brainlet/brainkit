package tools

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("tools", func(t *testing.T) {
		t.Run("list", func(t *testing.T) { testToolsList(t, env) })
		t.Run("resolve_echo", func(t *testing.T) { testToolsResolveEcho(t, env) })
		t.Run("resolve_not_found", func(t *testing.T) { testToolsResolveNotFound(t, env) })
		t.Run("call_echo", func(t *testing.T) { testToolsCallEcho(t, env) })
		t.Run("call_add", func(t *testing.T) { testToolsCallAdd(t, env) })
		t.Run("call_not_found", func(t *testing.T) { testToolsCallNotFound(t, env) })
	})
}
