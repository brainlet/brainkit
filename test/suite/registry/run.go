package registry

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("registry", func(t *testing.T) {
		// providers.go — Go-side + JS bridge registry ops
		t.Run("go_side_register_and_list", func(t *testing.T) { testGoSideRegisterAndList(t, env) })
		t.Run("go_side_runtime_register_unregister", func(t *testing.T) { testGoSideRuntimeRegisterUnregister(t, env) })
		t.Run("js_bridge_has", func(t *testing.T) { testJSBridgeHas(t, env) })
		t.Run("js_bridge_list", func(t *testing.T) { testJSBridgeList(t, env) })
		t.Run("js_bridge_resolve", func(t *testing.T) { testJSBridgeResolve(t, env) })
		t.Run("with_deployed_ts", func(t *testing.T) { testWithDeployedTS(t, env) })

		// packages_client.go — registry client search/fetch tests
		t.Run("search_by_name", func(t *testing.T) { testSearchByName(t, env) })
		t.Run("search_by_capability", func(t *testing.T) { testSearchByCapability(t, env) })
		t.Run("search_multiple_capabilities", func(t *testing.T) { testSearchMultipleCapabilities(t, env) })
		t.Run("search_no_results", func(t *testing.T) { testSearchNoResults(t, env) })
		t.Run("search_all_plugins", func(t *testing.T) { testSearchAllPlugins(t, env) })
		t.Run("fetch_manifest", func(t *testing.T) { testFetchManifest(t, env) })
		t.Run("fetch_manifest_specific_version", func(t *testing.T) { testFetchManifestSpecificVersion(t, env) })
		t.Run("fetch_manifest_wrong_version", func(t *testing.T) { testFetchManifestWrongVersion(t, env) })
		t.Run("fetch_manifest_not_found", func(t *testing.T) { testFetchManifestNotFound(t, env) })
		t.Run("multiple_registries", func(t *testing.T) { testMultipleRegistries(t, env) })
		t.Run("registry_with_auth", func(t *testing.T) { testRegistryWithAuth(t, env) })
	})
}
