package cli

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("cli", func(t *testing.T) {
		// cobra.go — scaffolder + version (no running server needed).
		t.Run("version", func(t *testing.T) { testVersion(t, env) })
		t.Run("version_json", func(t *testing.T) { testVersionJSON(t, env) })
		t.Run("new_package", func(t *testing.T) { testNewPackage(t, env) })
		t.Run("new_plugin", func(t *testing.T) { testNewPlugin(t, env) })
		t.Run("new_server", func(t *testing.T) { testNewServer(t, env) })

		// cobra.go — 5-verb CLI round-trips against a running
		// gateway (starts its own server on a random port).
		t.Run("inspect/health", func(t *testing.T) { testInspectHealth(t, env) })
		t.Run("inspect/health_json", func(t *testing.T) { testInspectHealthJSON(t, env) })
		t.Run("call/health_round_trip", func(t *testing.T) { testCallVerb(t, env) })
		t.Run("deploy/round_trip", func(t *testing.T) { testDeployVerb(t, env) })
		t.Run("deploy/full_workflow", func(t *testing.T) { testDeployFullWorkflow(t, env) })

		// commands.go — bus command tests (kit.eval, kit.health, kit.send)
		t.Run("kit_eval", func(t *testing.T) { testKitEval(t, env) })
		t.Run("kit_health", func(t *testing.T) { testKitHealth(t, env) })
		t.Run("kit_send_request_reply", func(t *testing.T) { testKitSendRequestReply(t, env) })
		t.Run("kit_send_with_await", func(t *testing.T) { testKitSendWithAwait(t, env) })
	})
}
