package jsbridge

import (
	"errors"
	"strings"
	"testing"

	quickjs "github.com/buke/quickjs-go"
)

func TestWrapErrorPreservesQuickJSCauseStackAndContext(t *testing.T) {
	b, err := New(Config{}, ErrorCompat())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()

	_, err = b.EvalAsync("diagnostic-source.js", `(async function outer() {
		function inner() {
			var err = new Error("outer failure");
			err.cause = new TypeError("inner cause");
			throw err;
		}
		inner();
	})()`)
	if err == nil {
		t.Fatal("EvalAsync error = nil")
	}

	snap := b.DebugSnapshot()
	wrapped := WrapError(err, DiagnosticContext{
		RuntimeOwner:   "test-runtime",
		Phase:          "eval",
		Source:         "diagnostic-source.js",
		Function:       "__brainkit.test.call",
		Provider:       "fake-provider",
		Model:          "fake-model",
		BridgeSnapshot: &snap,
	})

	var jsErr *quickjs.Error
	if !errors.As(wrapped, &jsErr) {
		t.Fatalf("wrapped error does not expose quickjs.Error: %T", wrapped)
	}
	if jsErr.Cause == "" || !strings.Contains(jsErr.Cause, "inner cause") {
		t.Fatalf("quickjs cause = %q, want inner cause", jsErr.Cause)
	}
	var diag *DiagnosticError
	if !errors.As(wrapped, &diag) {
		t.Fatalf("wrapped error does not expose DiagnosticError: %T", wrapped)
	}

	msg := wrapped.Error()
	for _, want := range []string{
		"owner=test-runtime",
		"phase=eval",
		"source=diagnostic-source.js",
		"function=__brainkit.test.call",
		"provider=fake-provider",
		"model=fake-model",
		"outer failure",
		"inner cause",
		"Bridge snapshot:",
		"JavaScript stack:",
		"inner",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("wrapped error missing %q:\n%s", want, msg)
		}
	}
}
