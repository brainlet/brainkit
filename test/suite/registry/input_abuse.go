package registry

import (
	"testing"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testInputAbuseEmptyProviderName — registering with empty name errors.
func testInputAbuseEmptyProviderName(t *testing.T, env *suite.TestEnv) {
	result, err := testutil.EvalTSErr(env.Kit, "__reg_empty_reg_adv.ts", `
		var caught = "none";
		try { kit.register("tool", "", {}); }
		catch(e) { caught = e.message || "error"; }
		return caught;
	`)
	require.NoError(t, err)
	assert.Contains(t, result, "required")
}

// testInputAbuseDuplicateRegister — registering duplicate name via kit.register.
func testInputAbuseDuplicateRegister(t *testing.T, env *suite.TestEnv) {
	result, err := testutil.EvalTSErr(env.Kit, "__reg_dup_reg_adv.ts", `
		var caught = "none";
		try {
			var t1 = createTool({id: "dup-tool-reg-adv", description: "first", execute: async () => ({v: 1})});
			kit.register("tool", "dup-tool-reg-adv", t1);
			var t2 = createTool({id: "dup-tool-reg-adv", description: "second", execute: async () => ({v: 2})});
			kit.register("tool", "dup-tool-reg-adv", t2);
			caught = "no-error";
		} catch(e) {
			caught = e.message || "error";
		}
		return caught;
	`)
	require.NoError(t, err)
	// Should either succeed (overwrite) or error — never panic
	assert.NotEqual(t, "none", result)
}

// testInputAbuseInvalidConfig — registering with invalid type errors.
func testInputAbuseInvalidConfig(t *testing.T, env *suite.TestEnv) {
	result, err := testutil.EvalTSErr(env.Kit, "__reg_invalid_reg_adv.ts", `
		var caught = "none";
		try { kit.register("banana", "test-reg-adv", {}); }
		catch(e) { caught = e.message || "error"; }
		return caught;
	`)
	require.NoError(t, err)
	assert.Contains(t, result, "invalid type")
}

// testInputAbuseMissingType — registering with empty type string errors.
func testInputAbuseMissingType(t *testing.T, env *suite.TestEnv) {
	result, err := testutil.EvalTSErr(env.Kit, "__reg_missing_type_reg_adv.ts", `
		var caught = "none";
		try { kit.register("", "test-reg-adv", {}); }
		catch(e) { caught = e.message || "error"; }
		return caught;
	`)
	require.NoError(t, err)
	// Should error on empty type — never succeed silently
	assert.NotEqual(t, "none", result)
}
