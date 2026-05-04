package brainkit

import (
	"context"
	"testing"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"
	evalmod "github.com/brainlet/brainkit/modules/eval"
	"github.com/brainlet/brainkit/modules/eval/evalmsg"
	jsruntimemod "github.com/brainlet/brainkit/modules/jsruntime"
	"github.com/brainlet/brainkit/presets/standard"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/brainlet/brainkit/stores"
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
	require.False(t, k.hasCommand("kit.eval"))
}

func TestExplicitEmptyProvidersDisableEnvAutoDetect(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test-auto-detect-disabled")

	k, err := New(Config{
		Transport: Memory(),
		Providers: []ProviderConfig{},
	})
	require.NoError(t, err)
	defer k.Close()

	require.Empty(t, k.kernel.ProviderRegistry().ListAIProviders())
}

func TestNilProvidersAutoDetectFromEnv(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test-auto-detect-enabled")

	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	require.NotEmpty(t, k.kernel.ProviderRegistry().ListAIProviders())
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

func TestStorageAndVectorConfigDoNotAutoEnableJSRuntime(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := New(Config{
		Transport: Memory(),
		FSRoot:    tmpDir,
		Storages: map[string]StorageConfig{
			"default": SQLiteStorage(tmpDir + "/storage.db"),
		},
		Vectors: map[string]VectorConfig{
			"docs": SQLiteVector(tmpDir + "/vectors.db"),
		},
	})
	require.NoError(t, err)
	defer k.Close()

	require.False(t, k.kernel.HasJSRuntime())
	require.Empty(t, k.kernel.StorageURL("default"))
	require.NotEmpty(t, k.kernel.ProviderRegistry().ListStorages())
	require.NotEmpty(t, k.kernel.ProviderRegistry().ListVectorStores())
}

func TestStandardCommandSetAutoEnablesJSRuntime(t *testing.T) {
	k, err := New(Config{Transport: Memory(), Modules: standard.CommandSet()})
	require.NoError(t, err)
	defer k.Close()

	require.True(t, k.kernel.HasJSRuntime())
	require.True(t, k.hasCommand("kit.eval"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := k.kernel.EvalTS(ctx, "__standard.ts", `return "ok"`)
	require.NoError(t, err)
	require.Equal(t, "ok", result)
}

func TestStandardCommandModulesUnmountCleanly(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mods := standard.CommandSet()
	for _, mod := range mods {
		require.NoError(t, k.Mount(ctx, mod), mod.ID())
	}
	require.True(t, k.kernel.HasJSRuntime())
	require.NotEmpty(t, k.MountedModules())

	for i := len(mods) - 1; i >= 0; i-- {
		id := mods[i].ID()
		require.NoError(t, k.Unmount(ctx, id), id)
		_, mounted := k.Module(id)
		require.False(t, mounted, id)
	}
	require.False(t, k.kernel.HasJSRuntime())
	require.Empty(t, k.MountedModules())
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

func TestJSRuntimeCanHotUnmountAndRemount(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, k.Mount(ctx, jsruntimemod.New()))
	require.True(t, k.kernel.HasJSRuntime())
	result, err := k.kernel.EvalTS(ctx, "__hot_unmount_before.ts", `return "before"`)
	require.NoError(t, err)
	require.Equal(t, "before", result)

	resources, err := k.kernel.Deploy(ctx, "__hot_unmount_deploy.ts", `bus.subscribe("test.hot.unmount.event", async () => {});`)
	require.NoError(t, err)
	require.NotEmpty(t, resources)
	require.NotEmpty(t, k.kernel.ListDeployments())

	require.NoError(t, k.Unmount(ctx, "jsruntime"))
	require.False(t, k.kernel.HasJSRuntime())
	_, err = k.kernel.EvalTS(ctx, "__hot_unmount_after.ts", `return "after"`)
	require.ErrorAs(t, err, new(*sdkerrors.NotConfiguredError))

	require.NoError(t, k.Mount(ctx, jsruntimemod.New()))
	require.True(t, k.kernel.HasJSRuntime())
	require.Empty(t, k.kernel.ListDeployments())
	result, err = k.kernel.EvalTS(ctx, "__hot_unmount_remount.ts", `return "remount"`)
	require.NoError(t, err)
	require.Equal(t, "remount", result)
}

func TestJSRuntimeHotUnmountPreservesPersistedDeployments(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := stores.NewSQLite(tmpDir + "/kit.db")
	require.NoError(t, err)

	k, err := New(Config{
		Transport: Memory(),
		FSRoot:    tmpDir,
		Store:     store,
		Modules:   []bkmodule.Module{jsruntimemod.New()},
	})
	require.NoError(t, err)
	defer k.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = k.kernel.Deploy(ctx, "persist-hot-unmount.ts", `bus.on("ping", function(msg) { msg.reply({ok: true}); });`)
	require.NoError(t, err)
	persisted, err := store.LoadDeployments()
	require.NoError(t, err)
	require.Len(t, persisted, 1)

	require.NoError(t, k.Unmount(ctx, "jsruntime"))
	persisted, err = store.LoadDeployments()
	require.NoError(t, err)
	require.Len(t, persisted, 1)
	require.False(t, k.kernel.HasJSRuntime())

	require.NoError(t, k.Mount(ctx, jsruntimemod.New()))
	require.True(t, k.kernel.HasJSRuntime())
	require.Len(t, k.kernel.ListDeployments(), 1)
}

func TestJSRuntimeModuleIsMountedBeforeDependents(t *testing.T) {
	k, err := New(Config{
		Transport: Memory(),
		Modules: []bkmodule.Module{
			evalmod.New(),
			jsruntimemod.New(),
		},
	})
	require.NoError(t, err)
	defer k.Close()

	require.True(t, k.kernel.HasJSRuntime())
	require.True(t, k.hasCommand("kit.eval"))
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
	require.True(t, k.hasCommand("kit.eval"))
}

func TestJSRuntimeUnmountRefusesMountedDependents(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()
	require.False(t, k.kernel.HasJSRuntime())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, k.Mount(ctx, evalmod.New()))
	require.True(t, k.kernel.HasJSRuntime())

	err = k.Unmount(ctx, "jsruntime")
	require.Error(t, err)
	require.Contains(t, err.Error(), "eval")
	require.True(t, k.kernel.HasJSRuntime())

	require.NoError(t, k.Unmount(ctx, "eval"))
	require.NoError(t, k.Unmount(ctx, "jsruntime"))
	require.False(t, k.kernel.HasJSRuntime())
}
