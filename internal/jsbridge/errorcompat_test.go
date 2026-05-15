package jsbridge

import (
	"encoding/json"
	"testing"
)

func TestErrorPrepareStackTraceCallsites(t *testing.T) {
	b := newTestBridge(t, ErrorCompat())

	result := evalString(t, b, `
		const previous = Error.prepareStackTrace;
		Error.prepareStackTrace = function(_, stack) { return stack; };
		function capture() { return new Error("boom").stack; }
		const stack = capture();
		Error.prepareStackTrace = previous;
		const first = stack[0];
		JSON.stringify({
			isArray: Array.isArray(stack),
			hasGetFileName: first && typeof first.getFileName === "function",
			hasFunctionName: first && typeof first.getFunctionName === "function",
			hasToString: first && typeof first.toString === "function",
			stackTextType: typeof new Error("plain").stack,
			instanceofError: new Error("plain") instanceof Error
		});
	`)

	var parsed struct {
		IsArray         bool   `json:"isArray"`
		HasGetFileName  bool   `json:"hasGetFileName"`
		HasFunctionName bool   `json:"hasFunctionName"`
		HasToString     bool   `json:"hasToString"`
		StackTextType   string `json:"stackTextType"`
		InstanceofError bool   `json:"instanceofError"`
	}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, result)
	}
	if !parsed.IsArray || !parsed.HasGetFileName || !parsed.HasFunctionName || !parsed.HasToString {
		t.Fatalf("prepareStackTrace callsites are incomplete: %+v", parsed)
	}
	if parsed.StackTextType != "string" || !parsed.InstanceofError {
		t.Fatalf("plain Error stack/instanceof contract changed: %+v", parsed)
	}
}
