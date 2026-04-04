package agents

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("agents", func(t *testing.T) {
		t.Run("list_empty", func(t *testing.T) { testListEmpty(t, env) })
		t.Run("discover_no_match", func(t *testing.T) { testDiscoverNoMatch(t, env) })
		t.Run("get_status_not_found", func(t *testing.T) { testGetStatusNotFound(t, env) })
		t.Run("set_status_not_found", func(t *testing.T) { testSetStatusNotFound(t, env) })
		t.Run("set_status_invalid", func(t *testing.T) { testSetStatusInvalid(t, env) })
	})
}
