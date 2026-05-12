package eval

import (
	"context"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/modules/eval/evalmsg"
)

type recordingEvalRuntime struct {
	jsSource     string
	moduleSource string
	scriptSource string
}

func (r *recordingEvalRuntime) EvalJS(_ context.Context, source, _ string) (string, error) {
	r.jsSource = source
	return "js", nil
}

func (r *recordingEvalRuntime) EvalModule(_ context.Context, source, _ string) (string, error) {
	r.moduleSource = source
	return "module", nil
}

func (r *recordingEvalRuntime) EvalScript(_ context.Context, source, _ string) (string, error) {
	r.scriptSource = source
	return "script", nil
}

func TestEvalDispatchesThroughNarrowEvalRuntime(t *testing.T) {
	rt := &recordingEvalRuntime{}
	m := &Module{runtime: rt}
	ctx := context.Background()

	resp, err := m.Eval(ctx, evalmsg.KitEvalMsg{Mode: "js", Source: "snippet.js", Code: `return "js";`})
	if err != nil || resp.Result != "js" {
		t.Fatalf("js eval = %#v, %v", resp, err)
	}
	if rt.jsSource != "snippet.js" {
		t.Fatalf("js source = %q, want snippet.js", rt.jsSource)
	}

	resp, err = m.Eval(ctx, evalmsg.KitEvalMsg{Mode: "module", Source: "mod.ts", Code: `export {};`})
	if err != nil || resp.Result != "module" {
		t.Fatalf("module eval = %#v, %v", resp, err)
	}
	if rt.moduleSource != "mod.ts" {
		t.Fatalf("module source = %q, want mod.ts", rt.moduleSource)
	}

	resp, err = m.Eval(ctx, evalmsg.KitEvalMsg{Mode: "script", Code: `output("script");`})
	if err != nil || resp.Result != "script" {
		t.Fatalf("script eval = %#v, %v", resp, err)
	}
	if !strings.HasPrefix(rt.scriptSource, "__cli_eval_") || !strings.HasSuffix(rt.scriptSource, ".js") {
		t.Fatalf("script source = %q, want generated __cli_eval_*.js", rt.scriptSource)
	}

	resp, err = m.Eval(ctx, evalmsg.KitEvalMsg{Mode: "script", Source: "typed-snippet.ts", Code: `output("script");`})
	if err != nil || resp.Result != "script" {
		t.Fatalf("script eval with source = %#v, %v", resp, err)
	}
	if rt.scriptSource != "typed-snippet.ts" {
		t.Fatalf("script source = %q, want explicit .ts source", rt.scriptSource)
	}
}

func TestEvalInfersScriptForExplicitTSSource(t *testing.T) {
	rt := &recordingEvalRuntime{}
	m := &Module{runtime: rt}

	resp, err := m.Eval(context.Background(), evalmsg.KitEvalMsg{Source: "typed-snippet.ts", Code: `output("typed");`})
	if err != nil || resp.Result != "script" {
		t.Fatalf("inferred script eval = %#v, %v", resp, err)
	}
	if rt.scriptSource != "typed-snippet.ts" {
		t.Fatalf("script source = %q, want explicit .ts source", rt.scriptSource)
	}
	if rt.jsSource != "" {
		t.Fatalf("js source = %q, want script path for explicit .ts source", rt.jsSource)
	}
}
