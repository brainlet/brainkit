package secrets

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/secrets/secretmsg"
	"github.com/brainlet/brainkit/test/suite"
)

// testNoModuleCommandsAbsent verifies that secrets.* bus commands are only
// available when modules/secrets is wired.
func testNoModuleCommandsAbsent(t *testing.T, _ *suite.TestEnv) {
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test-secrets-no-module",
		CallerID:  "test",
		FSRoot:    t.TempDir(),
	})
	if err != nil {
		t.Fatalf("brainkit.New: %v", err)
	}
	t.Cleanup(func() { k.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := secretmsg.CallSecretsList(k, ctx, secretmsg.SecretsListMsg{}); err == nil {
		t.Fatal("expected secrets.list to fail without secrets module")
	}
}
