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

		// input_abuse.go — tools input abuse (adversarial)
		t.Run("input_abuse_call_nonexistent", func(t *testing.T) { testInputAbuseCallNonexistent(t, env) })
		t.Run("input_abuse_wrong_input_type", func(t *testing.T) { testInputAbuseWrongInputType(t, env) })
		t.Run("input_abuse_empty_tool_name", func(t *testing.T) { testInputAbuseEmptyToolName(t, env) })
		t.Run("input_abuse_oversized_input", func(t *testing.T) { testInputAbuseOversizedInput(t, env) })

		// e2e.go — tool pipeline end-to-end
		t.Run("e2e_tool_pipeline", func(t *testing.T) { testToolPipeline(t, env) })
	})
}
