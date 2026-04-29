package brainkit

import (
	"context"
	"testing"
	"time"

	evalmod "github.com/brainlet/brainkit/modules/eval"
	"github.com/brainlet/brainkit/modules/eval/evalmsg"
	jsruntimemod "github.com/brainlet/brainkit/modules/jsruntime"
	"github.com/brainlet/brainkit/modules/standard"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/stretchr/testify/require"
)

func TestZeroConfigStartsLightCoreWithoutJSRuntime(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	require.False(t, k.kernel.HasJSRuntime())
	require.True(t, k.Alive(context.Background()))

	_, err = k.kernel.EvalTS(context.Background(), "__disabled.ts", `return "ok"`)
	require.ErrorAs(t, err, new(*sdkerrors.NotConfiguredError))
	require.False(t, k.HasCommand("kit.eval"))
}

func TestExplicitEmptyProvidersDisableEnvAutoDetect(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test-auto-detect-disabled")

	k, err := New(Config{
		Transport: Memory(),
		Providers: []ProviderConfig{},
	})
	require.NoError(t, err)
	defer k.Close()

	require.Empty(t, k.Providers().List())
}

func TestNilProvidersAutoDetectFromEnv(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test-auto-detect-enabled")

	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	require.NotEmpty(t, k.Providers().List())
}

func TestJSRuntimeExplicitlyEnablesEvalRuntime(t *testing.T) {
	k, err := New(Config{Transport: Memory(), JSRuntime: true})
	require.NoError(t, err)
	defer k.Close()

	require.True(t, k.kernel.HasJSRuntime())
	result, err := k.kernel.EvalTS(context.Background(), "__enabled.ts", `return "ok"`)
	require.NoError(t, err)
	require.Equal(t, "ok", result)
}

func TestStandardCommandSetAutoEnablesJSRuntime(t *testing.T) {
	k, err := New(Config{Transport: Memory(), Modules: standard.CommandSet()})
	require.NoError(t, err)
	defer k.Close()

	require.True(t, k.kernel.HasJSRuntime())
	require.True(t, k.HasCommand("kit.eval"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := k.kernel.EvalTS(ctx, "__standard.ts", `return "ok"`)
	require.NoError(t, err)
	require.Equal(t, "ok", result)
}

func TestJSRuntimeCanHotMountAfterKitStart(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()
	require.False(t, k.kernel.HasJSRuntime())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, k.Mount(ctx, jsruntimemod.New()))
	require.True(t, k.kernel.HasJSRuntime())

	result, err := k.kernel.EvalTS(ctx, "__hot_runtime.ts", `return "hot"`)
	require.NoError(t, err)
	require.Equal(t, "hot", result)

	require.NoError(t, k.Mount(ctx, evalmod.New()))
	resp, err := Call[evalmsg.KitEvalMsg, evalmsg.KitEvalResp](k, ctx, evalmsg.KitEvalMsg{
		Mode:   "ts",
		Source: "__hot_eval.ts",
		Code:   `return "eval"`,
	})
	require.NoError(t, err)
	require.Equal(t, "eval", resp.Result)
}

func TestJSRuntimeModuleIsMountedBeforeDependents(t *testing.T) {
	k, err := New(Config{
		Transport: Memory(),
		Modules: []Module{
			evalmod.New(),
			jsruntimemod.New(),
		},
	})
	require.NoError(t, err)
	defer k.Close()

	require.True(t, k.kernel.HasJSRuntime())
	require.True(t, k.HasCommand("kit.eval"))
}

func TestJSDependentHotMountAutoMountsJSRuntime(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()
	require.False(t, k.kernel.HasJSRuntime())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, k.Mount(ctx, evalmod.New()))
	require.True(t, k.kernel.HasJSRuntime())
	require.True(t, k.HasCommand("kit.eval"))
}
