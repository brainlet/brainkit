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

		// matrix.go — secrets permission matrix (adversarial)
		t.Run("matrix_set_get_delete_list", func(t *testing.T) { testMatrixSetGetDeleteList(t, env) })
		t.Run("matrix_rotate", func(t *testing.T) { testMatrixRotate(t, env) })
		t.Run("matrix_many_secrets", func(t *testing.T) { testMatrixManySecrets(t, env) })
		t.Run("matrix_encrypted_persistence", func(t *testing.T) { testMatrixEncryptedPersistence(t, env) })
		t.Run("matrix_wrong_key_cannot_decrypt", func(t *testing.T) { testMatrixWrongKeyCannotDecrypt(t, env) })
		t.Run("matrix_audit_events", func(t *testing.T) { testMatrixAuditEvents(t, env) })
		t.Run("matrix_from_ts", func(t *testing.T) { testMatrixFromTS(t, env) })

		// input_abuse.go — secrets input abuse (adversarial)
		t.Run("input_abuse_empty_name", func(t *testing.T) { testInputAbuseEmptyName(t, env) })
		t.Run("input_abuse_large_value", func(t *testing.T) { testInputAbuseLargeValue(t, env) })
		t.Run("input_abuse_special_chars_in_name", func(t *testing.T) { testInputAbuseSpecialCharsInName(t, env) })
		t.Run("input_abuse_bulk_operations", func(t *testing.T) { testInputAbuseBulkOperations(t, env) })

		// integration.go — secrets rotation integration
		t.Run("integration_rotation", func(t *testing.T) { testSecretsRotation(t, env) })
	})
}
