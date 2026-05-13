package agentembed

import (
	"context"
	"errors"
	"strings"
	"testing"

	quickjs "github.com/buke/quickjs-go"

	"github.com/brainlet/brainkit/internal/jsbridge"
)

func TestSandboxEvalDiagnosticIncludesCauseStackAndSnapshot(t *testing.T) {
	sandbox, err := NewSandbox(SandboxConfig{})
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}
	defer sandbox.Close()

	_, err = sandbox.Eval(context.Background(), "diagnostic-sandbox.js", `
		(async function scorerWrapper() {
			function missingSurface() {
				var shim = {};
				if (typeof shim.getVersionOverrides !== "function") {
					throw new TypeError("missing runtime surface: shim.getVersionOverrides is not a function");
				}
				return shim.getVersionOverrides();
			}
			var outer = new Error("Scorer Run Failed");
			try {
				missingSurface();
			} catch (cause) {
				outer.cause = cause;
			}
			throw outer;
		})()
	`)
	if err == nil {
		t.Fatal("Eval error = nil")
	}

	var jsErr *quickjs.Error
	if !errors.As(err, &jsErr) {
		t.Fatalf("error does not expose quickjs.Error: %T", err)
	}
	var diag *jsbridge.DiagnosticError
	if !errors.As(err, &diag) {
		t.Fatalf("error does not expose DiagnosticError: %T", err)
	}

	msg := err.Error()
	for _, want := range []string{
		"owner=agent-embed",
		"phase=eval",
		"source=diagnostic-sandbox.js",
		"Scorer Run Failed",
		"getVersionOverrides",
		"not a function",
		"Bridge snapshot:",
		"JavaScript stack:",
		"scorerWrapper",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("diagnostic missing %q:\n%s", want, msg)
		}
	}
}

func TestAgentGenerateDiagnosticIncludesProviderModelWithoutPromptBody(t *testing.T) {
	sandbox, err := NewSandbox(SandboxConfig{})
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}
	defer sandbox.Close()

	_, err = sandbox.Bridge().Eval("fake-agent-diagnostic.js", `
		globalThis.__agents["diagnostic-agent"] = {
			generate: async function() {
				var outer = new Error("Scorer Run Failed");
				outer.cause = new TypeError("runtime.getVersionOverrides is not a function");
				throw outer;
			}
		};
	`)
	if err != nil {
		t.Fatalf("install fake agent: %v", err)
	}

	agent := &Agent{
		id:       "diagnostic-agent",
		jsID:     "diagnostic_agent",
		name:     "diagnostic-agent",
		provider: "openai",
		modelID:  "gpt-diagnostic",
		sandbox:  sandbox,
	}
	_, err = agent.Generate(context.Background(), GenerateParams{Prompt: "DO_NOT_LEAK_PROMPT_BODY"})
	if err == nil {
		t.Fatal("Generate error = nil")
	}

	msg := err.Error()
	for _, want := range []string{
		"agent-embed: generate",
		"owner=agent-embed",
		"phase=agent.generate",
		"source=agent-generate.js",
		"function=Agent.generate",
		"provider=openai",
		"model=gpt-diagnostic",
		"Scorer Run Failed",
		"getVersionOverrides",
		"not a function",
		"Bridge snapshot:",
		"JavaScript stack:",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("generate diagnostic missing %q:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "DO_NOT_LEAK_PROMPT_BODY") {
		t.Fatalf("diagnostic leaked prompt body:\n%s", msg)
	}
}
