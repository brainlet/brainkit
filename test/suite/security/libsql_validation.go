package security

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testLibSQLFileURLBlocked — LibSQLStore rejects file: URLs (sandbox escape prevention).
func testLibSQLFileURLBlocked(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	ctx := context.Background()

	_, err := freshEnv.Kernel.Deploy(ctx, "file-url-test-sec.ts", `
		try {
			var store = new LibSQLStore({ url: "file:./sneaky.db" });
			output({ blocked: false });
		} catch(e) {
			output({ blocked: true, code: e.code || "unknown", message: e.message || String(e) });
		}
	`)
	require.NoError(t, err)

	result, err := freshEnv.Kernel.EvalTS(ctx, "__get_output_sec.ts", `return globalThis.__module_result || "null";`)
	require.NoError(t, err)

	var parsed struct {
		Blocked bool   `json:"blocked"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal([]byte(result), &parsed), "result was: %s", result)
	assert.True(t, parsed.Blocked, "file: URL should be blocked")
	assert.Equal(t, "VALIDATION_ERROR", parsed.Code)
	assert.Contains(t, parsed.Message, "file:")
}

// testLibSQLHttpURLNotBlocked — http: URLs should not trigger file: URL validation.
func testLibSQLHttpURLNotBlocked(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	ctx := context.Background()

	_, err := freshEnv.Kernel.Deploy(ctx, "http-url-test-sec.ts", `
		try {
			var store = new LibSQLStore({ url: "http://127.0.0.1:9999" });
			output({ code: "none" });
		} catch(e) {
			output({ code: e.code || "unknown", message: e.message || String(e) });
		}
	`)
	require.NoError(t, err)

	result, err := freshEnv.Kernel.EvalTS(ctx, "__get_output_http_sec.ts", `return globalThis.__module_result || "null";`)
	require.NoError(t, err)

	var parsed struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal([]byte(result), &parsed), "result was: %s", result)
	assert.NotEqual(t, "VALIDATION_ERROR", parsed.Code, "http: URL should not trigger file: URL validation")
}
