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

		// storage_runtime.go — runtime storage register/unregister (adversarial)
		t.Run("storage_runtime_add_remove", func(t *testing.T) { testStorageRuntimeAddRemove(t, env) })
		t.Run("storage_runtime_add_duplicate", func(t *testing.T) { testStorageRuntimeAddDuplicate(t, env) })
		t.Run("storage_runtime_remove_nonexistent", func(t *testing.T) { testStorageRuntimeRemoveNonexistent(t, env) })
		t.Run("storage_runtime_url_for_nonexistent", func(t *testing.T) { testStorageRuntimeURLForNonexistent(t, env) })
		t.Run("storage_runtime_sqlite_add", func(t *testing.T) { testStorageRuntimeSQLiteAdd(t, env) })
		t.Run("storage_runtime_list_resources", func(t *testing.T) { testStorageRuntimeListResources(t, env) })
		t.Run("storage_runtime_resources_from_source", func(t *testing.T) { testStorageRuntimeResourcesFromSource(t, env) })
		t.Run("storage_runtime_scaling_pool", func(t *testing.T) { testStorageRuntimeScalingPool(t, env) })
		t.Run("storage_runtime_kernel_multiple_storages", func(t *testing.T) { testStorageRuntimeKernelMultipleStorages(t, env) })

		// input_abuse.go — registry input abuse (adversarial)
		t.Run("input_abuse_empty_provider_name", func(t *testing.T) { testInputAbuseEmptyProviderName(t, env) })
		t.Run("input_abuse_duplicate_register", func(t *testing.T) { testInputAbuseDuplicateRegister(t, env) })
		t.Run("input_abuse_invalid_config", func(t *testing.T) { testInputAbuseInvalidConfig(t, env) })
		t.Run("input_abuse_missing_type", func(t *testing.T) { testInputAbuseMissingType(t, env) })

		// provider_management.go — provider add/remove via typed bus commands
		t.Run("bus/provider_add", func(t *testing.T) { testProviderAddViaBus(t, env) })
		t.Run("bus/provider_add_invalid", func(t *testing.T) { testProviderAddInvalidName(t, env) })
		t.Run("bus/provider_remove", func(t *testing.T) { testProviderRemoveViaBus(t, env) })
		t.Run("bus/provider_add_then_resolve_ts", func(t *testing.T) { testProviderAddThenResolveFromTS(t, env) })

		// storage_management.go — storage add/remove via typed bus commands
		t.Run("bus/storage_add", func(t *testing.T) { testStorageAddViaBus(t, env) })
		t.Run("bus/storage_remove", func(t *testing.T) { testStorageRemoveViaBus(t, env) })
		t.Run("bus/storage_remove_nonexistent", func(t *testing.T) { testStorageRemoveNonexistent(t, env) })
		t.Run("bus/storage_add_sqlite_deploy", func(t *testing.T) { testStorageAddSQLiteThenDeployUses(t, env) })
		t.Run("bus/storage_add_memory_deploy", func(t *testing.T) { testStorageAddMemoryThenDeployUses(t, env) })

		// vector_management.go — vector add/remove via typed bus commands
		t.Run("bus/vector_add", func(t *testing.T) { testVectorAddViaBus(t, env) })
		t.Run("bus/vector_remove", func(t *testing.T) { testVectorRemoveViaBus(t, env) })
		t.Run("bus/vector_add_then_resolve_ts", func(t *testing.T) { testVectorAddThenResolveFromTS(t, env) })
	})
}
