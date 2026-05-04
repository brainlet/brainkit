package eval

import (
	"context"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/modules/eval/evalmsg"
)

type recordingEvalRuntime struct {
	tsSource     string
	moduleSource string
	scriptSource string
}

func (r *recordingEvalRuntime) EvalTS(_ context.Context, source, _ string) (string, error) {
	r.tsSource = source
	return "ts", nil
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

	resp, err := m.Eval(ctx, evalmsg.KitEvalMsg{Mode: "ts", Source: "snippet.ts", Code: `return "ts";`})
	if err != nil || resp.Result != "ts" {
		t.Fatalf("ts eval = %#v, %v", resp, err)
	}
	if rt.tsSource != "snippet.ts" {
		t.Fatalf("ts source = %q, want snippet.ts", rt.tsSource)
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
	if !strings.HasPrefix(rt.scriptSource, "__cli_eval_") || !strings.HasSuffix(rt.scriptSource, ".ts") {
		t.Fatalf("script source = %q, want generated __cli_eval_*.ts", rt.scriptSource)
	}
}
