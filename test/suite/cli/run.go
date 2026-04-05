package cli

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("cli", func(t *testing.T) {
		// cobra.go — CLI command tests (no running instance needed)
		t.Run("version", func(t *testing.T) { testVersion(t, env) })
		t.Run("version_json", func(t *testing.T) { testVersionJSON(t, env) })
		t.Run("init", func(t *testing.T) { testInit(t, env) })
		t.Run("new_package", func(t *testing.T) { testNewPackage(t, env) })
		t.Run("new_plugin", func(t *testing.T) { testNewPlugin(t, env) })

		// cobra.go — Full E2E (needs running brainkit start)
		t.Run("full_workflow", func(t *testing.T) { testFullWorkflow(t, env) })
		t.Run("send_with_async_handler", func(t *testing.T) { testSendWithAsyncHandler(t, env) })
		t.Run("redeploy_picks_up_new_code", func(t *testing.T) { testRedeployPicksUpNewCode(t, env) })

		// commands.go — bus command tests (kit.eval, kit.health, kit.send)
		t.Run("kit_eval", func(t *testing.T) { testKitEval(t, env) })
		t.Run("kit_health", func(t *testing.T) { testKitHealth(t, env) })
		t.Run("kit_send_request_reply", func(t *testing.T) { testKitSendRequestReply(t, env) })
		t.Run("kit_send_with_await", func(t *testing.T) { testKitSendWithAwait(t, env) })
	})
}
