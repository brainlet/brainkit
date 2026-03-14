package brainkit

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	quickjs "github.com/buke/quickjs-go"

	"github.com/brainlet/brainkit/jsbridge"
)

func TestModuleImportBasic(t *testing.T) {
	b, err := jsbridge.New(jsbridge.Config{})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	ctx := b.Context()

	val := ctx.LoadModule(`
		export const greeting = "hello from module";
		export function add(a, b) { return a + b; }
	`, "mylib", quickjs.EvalLoadOnly(true))
	if val.IsException() {
		t.Fatalf("LoadModule: %v", ctx.Exception())
	}
	val.Free()

	val2 := ctx.LoadModule(`
		import { greeting, add } from "mylib";
		globalThis.__test_greeting = greeting;
		globalThis.__test_add = add(3, 4);
	`, "test.js")
	if val2.IsException() {
		t.Fatalf("import: %v", ctx.Exception())
	}
	val2.Free()

	result, err := b.Eval("check.js", `JSON.stringify({
		greeting: globalThis.__test_greeting,
		add: globalThis.__test_add,
	})`)
	if err != nil {
		t.Fatal(err)
	}
	defer result.Free()

	expected := `{"greeting":"hello from module","add":7}`
	if result.String() != expected {
		t.Errorf("got %s, want %s", result.String(), expected)
	}
}

func TestContract_ImportFromBrainlet(t *testing.T) {
	kit := newTestKit(t)

	result, err := kit.EvalModule(context.Background(), "test-import.js", `
		import { agent, sandbox, output } from "brainlet";

		const a = agent({
			model: "openai/gpt-4o-mini",
			instructions: "Reply with exactly: IMPORT_WORKS",
		});
		const r = await a.generate("Say it");
		output({ text: r.text, sandboxNs: sandbox.namespace });
	`)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out struct {
		Text      string `json:"text"`
		SandboxNs string `json:"sandboxNs"`
	}
	json.Unmarshal([]byte(result), &out)

	if !strings.Contains(strings.ToUpper(out.Text), "IMPORT_WORKS") {
		t.Errorf("unexpected text: %q", out.Text)
	}
	if out.SandboxNs != "test" {
		t.Errorf("sandbox.namespace = %q, want test", out.SandboxNs)
	}
	t.Logf("Contract import { agent, sandbox } from 'brainlet': text=%q ns=%q", out.Text, out.SandboxNs)
}

func TestContract_ImportAIFromBrainlet(t *testing.T) {
	kit := newTestKit(t)

	result, err := kit.EvalModule(context.Background(), "test-ai-import.js", `
		import { ai, output } from "brainlet";

		const r = await ai.generate({
			model: "openai/gpt-4o-mini",
			prompt: "Reply with exactly: IMPORTED_AI",
		});
		output({ text: r.text });
	`)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out struct{ Text string }
	json.Unmarshal([]byte(result), &out)
	if !strings.Contains(strings.ToUpper(out.Text), "IMPORTED_AI") {
		t.Errorf("unexpected: %q", out.Text)
	}
	t.Logf("Contract import { ai } from 'brainlet': %q", out.Text)
}
