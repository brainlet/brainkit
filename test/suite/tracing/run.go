package tracing

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("tracing", func(t *testing.T) {
		t.Run("command_request_creates_span", func(t *testing.T) { testCommandRequestCreatesSpan(t, env) })
		t.Run("handler_creates_span", func(t *testing.T) { testHandlerCreatesSpan(t, env) })
		t.Run("query_via_bus", func(t *testing.T) { testQueryViaBus(t, env) })
		t.Run("no_store_no_op", func(t *testing.T) { testNoStoreNoOp(t, env) })
		t.Run("tool_call_creates_span", func(t *testing.T) { testToolCallCreatesSpan(t, env) })
		t.Run("deploy_creates_span", func(t *testing.T) { testDeployCreatesSpan(t, env) })
		t.Run("query_by_source", func(t *testing.T) { testQueryBySource(t, env) })
		t.Run("empty_store", func(t *testing.T) { testEmptyStore(t, env) })
	})
}
