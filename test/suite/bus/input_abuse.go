package bus

import (
	"testing"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
)

// testInputAbuseBusEmptyTopic — empty topic should error, not panic.
func testInputAbuseBusEmptyTopic(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__empty_topic_adv.ts", `
		var caught = "none";
		try { bus.publish("", {}); }
		catch(e) { caught = e.code || e.message; }
		return caught;
	`)
	assert.NotEqual(t, "none", result, "empty topic should throw an error")
}

// testInputAbuseBusLargePayload — 100KB payload should be handled cleanly.
func testInputAbuseBusLargePayload(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__big_payload_adv.ts", `
		var big = { data: "x".repeat(100000) };
		var r = bus.publish("incoming.big-adv", big);
		return r.replyTo ? "ok" : "fail";
	`)
	assert.Equal(t, "ok", result)
}

// testInputAbuseBusDeeplyNestedJSON — deeply nested object should be handled cleanly.
func testInputAbuseBusDeeplyNestedJSON(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__nested_adv.ts", `
		var obj = {};
		var curr = obj;
		for (var i = 0; i < 50; i++) {
			curr.nested = {};
			curr = curr.nested;
		}
		curr.value = "deep";
		var r = bus.publish("incoming.nested-adv", obj);
		return r.replyTo ? "ok" : "fail";
	`)
	assert.Equal(t, "ok", result)
}

// testInputAbuseBusSubscribeEmptyTopic — subscribe to empty topic should not panic.
func testInputAbuseBusSubscribeEmptyTopic(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__sub_empty_adv.ts", `
		var caught = "none";
		try { bus.subscribe("", function() {}); }
		catch(e) { caught = "error"; }
		return caught;
	`)
	// Empty subscribe should either work or error — not panic.
	_ = result
}
