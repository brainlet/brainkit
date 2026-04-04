package secrets

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("secrets", func(t *testing.T) {
		t.Run("set_and_get", func(t *testing.T) { testSetAndGet(t, env) })
		t.Run("delete", func(t *testing.T) { testDelete(t, env) })
		t.Run("list", func(t *testing.T) { testList(t, env) })
		t.Run("rotate", func(t *testing.T) { testRotate(t, env) })
		t.Run("js_bridge", func(t *testing.T) { testJSBridge(t, env) })
		t.Run("audit_events", func(t *testing.T) { testAuditEvents(t, env) })
		t.Run("concurrent_access", func(t *testing.T) { testConcurrentAccess(t, env) })
		t.Run("dev_mode_no_encryption", func(t *testing.T) { testDevModeNoEncryption(t, env) })
		t.Run("list_never_leaks_values", func(t *testing.T) { testListNeverLeaksValues(t, env) })
	})
}
