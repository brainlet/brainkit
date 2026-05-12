package engine

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/embed/typescript"
	runtimejs "github.com/brainlet/brainkit/internal/jsruntime"
	"github.com/brainlet/brainkit/internal/types"
)

func TestKernelRawTSDeployTranspilesBeforeEvaluation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	k, err := NewKernel(types.KernelConfig{
		Namespace:        "raw-ts-test",
		CallerID:         "raw-ts-test",
		DeferRouterStart: true,
	})
	if err != nil {
		t.Fatalf("NewKernel: %v", err)
	}
	t.Cleanup(func() { _ = k.Close() })

	if err := runtimejs.Enable(ctx, k, runtimejs.WithSourcePreparer(func(source, code string) (string, error) {
		return typescript.TranspileTS(code, source)
	})); err != nil {
		t.Fatalf("Enable JS runtime: %v", err)
	}

	_, err = k.Deploy(ctx, "raw-typescript.ts", `import { output } from "kit";

interface Config {
  value: string;
}

type Result = { value: string };

const cfg: Config = { value: "ok" };
const result: Result = { value: cfg.value };
output(result);`)
	if err != nil {
		t.Fatalf("Deploy raw TypeScript: %v", err)
	}

	got, err := k.EvalJS(ctx, "__read_raw_ts_result.js", `return globalThis.__module_result;`)
	if err != nil {
		t.Fatalf("EvalJS read result: %v", err)
	}
	if got != `{"value":"ok"}` {
		t.Fatalf("raw TypeScript deployment result = %q, want %q", got, `{"value":"ok"}`)
	}
}
