package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	internalstore "github.com/brainlet/brainkit/internal/store"
	bkmodule "github.com/brainlet/brainkit/module"
	controlmod "github.com/brainlet/brainkit/modules/control"
	gatewaymod "github.com/brainlet/brainkit/modules/gateway"
	jsruntimemod "github.com/brainlet/brainkit/modules/jsruntime"
	pluginsmod "github.com/brainlet/brainkit/modules/plugins"
	schedulesmod "github.com/brainlet/brainkit/modules/schedules"
	"github.com/brainlet/brainkit/modules/schedules/schedulemsg"
	tracingmod "github.com/brainlet/brainkit/modules/tracing"
	"github.com/brainlet/brainkit/sdk"
	"github.com/stretchr/testify/require"
)

type lifecycleDebugTestModule struct {
	value string
}

func (m lifecycleDebugTestModule) ID() string { return "debug-test" }

func (m lifecycleDebugTestModule) Mount(ctx context.Context, host bkmodule.Host) error {
	registry, err := bkmodule.RequireCapability[bkmodule.LifecycleDebugRegistry](host, bkmodule.CapabilityLifecycleDebugRegistry)
	if err != nil {
		return err
	}
	handle, err := registry.RegisterLifecycleDebug(ctx, "debug-test", func() any {
		return map[string]string{"value": m.value}
	})
	if err != nil {
		return err
	}
	host.Scope().Defer(handle.Close)
	return nil
}

func TestLifecycleDebugRegistryIsScopedToModuleMount(t *testing.T) {
	k, err := New(Config{
		Transport: Memory(),
		Modules:   []bkmodule.Module{controlmod.New()},
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	require.NoError(t, k.Mount(ctx, lifecycleDebugTestModule{value: "mounted"}))

	resp, err := controlmod.CallKitLifecycle(k, ctx, controlmod.KitLifecycleMsg{}, sdk.WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	component := requireLifecycleComponent(t, resp.Lifecycle, "debug-test")
	var payload map[string]string
	require.NoError(t, json.Unmarshal(component.Data, &payload))
	require.Equal(t, "mounted", payload["value"])
	require.Equal(t, 2, resp.Lifecycle.Runtime.MountedModules)
	require.GreaterOrEqual(t, resp.Lifecycle.Runtime.Transport.ActiveSubscriptions, int64(2))

	require.NoError(t, k.Unmount(ctx, "debug-test"))
	resp, err = controlmod.CallKitLifecycle(k, ctx, controlmod.KitLifecycleMsg{}, sdk.WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	requireNoLifecycleComponent(t, resp.Lifecycle, "debug-test")
}

func TestLifecycleDebugSnapshotReportsTransportOwnershipAndClosedState(t *testing.T) {
	k, err := New(Config{
		Transport: Memory(),
		Modules:   []bkmodule.Module{controlmod.New()},
	})
	require.NoError(t, err)
	closed := false
	defer func() {
		if !closed {
			require.NoError(t, k.Close())
		}
	}()

	snapshot := k.kernel.LifecycleDebugSnapshot().Transport
	require.True(t, snapshot.OwnsTransport)
	require.False(t, snapshot.CallerClosed)
	require.Zero(t, snapshot.CallerPendingCalls)
	require.Zero(t, snapshot.CallerStreamDrains)
	require.False(t, snapshot.ClosedRouter)
	require.False(t, snapshot.ClosedCaller)
	require.False(t, snapshot.ClosedTransport)

	require.NoError(t, k.Close())
	closed = true

	snapshot = k.kernel.LifecycleDebugSnapshot().Transport
	require.True(t, snapshot.OwnsTransport)
	require.True(t, snapshot.ClosedRouter)
	require.True(t, snapshot.ClosedCaller)
	require.True(t, snapshot.ClosedTransport)
	require.True(t, snapshot.CallerClosed)
	require.Zero(t, snapshot.CallerPendingCalls)
	require.Zero(t, snapshot.CallerStreamDrains)
	require.False(t, snapshot.ClosingRouter)
	require.False(t, snapshot.ClosingCaller)
	require.False(t, snapshot.ClosingTransport)
}

func TestGatewayRegistersLifecycleDebugSnapshot(t *testing.T) {
	gw := gatewaymod.New(gatewaymod.Config{Listen: "127.0.0.1:0"})
	k, err := New(Config{
		Transport: Memory(),
		Modules:   []bkmodule.Module{controlmod.New(), gw},
	})
	require.NoError(t, err)
	defer k.Close()

	resp, err := controlmod.CallKitLifecycle(k, context.Background(), controlmod.KitLifecycleMsg{}, sdk.WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	component := requireLifecycleComponent(t, resp.Lifecycle, "gateway")
	var snapshot struct {
		Listening           bool   `json:"listening"`
		ServerAttached      bool   `json:"serverAttached"`
		Address             string `json:"address"`
		RouteCount          int    `json:"routeCount"`
		RouteSubscriptions  int    `json:"routeSubscriptions"`
		ActiveConnections   int64  `json:"activeConnections"`
		StreamSessions      int    `json:"streamSessions"`
		TerminalSessions    int    `json:"terminalStreamSessions"`
		StreamSubscriptions int    `json:"streamSubscriptions"`
	}
	require.NoError(t, json.Unmarshal(component.Data, &snapshot))
	require.True(t, snapshot.Listening)
	require.True(t, snapshot.ServerAttached)
	require.Equal(t, gw.Addr(), snapshot.Address)
	require.Equal(t, 4, snapshot.RouteSubscriptions)
	require.Zero(t, snapshot.StreamSessions)
	require.Zero(t, snapshot.TerminalSessions)
	require.Zero(t, snapshot.StreamSubscriptions)
}

func TestJSRuntimeRegistersLifecycleDebugSnapshot(t *testing.T) {
	store, err := internalstore.NewSQLiteKitStore(filepath.Join(t.TempDir(), "kit.db"))
	require.NoError(t, err)
	k, err := New(Config{
		Transport: Memory(),
		Store:     store,
		Modules:   []bkmodule.Module{controlmod.New(), jsruntimemod.New()},
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	resp, err := controlmod.CallKitLifecycle(k, ctx, controlmod.KitLifecycleMsg{}, sdk.WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	component := requireLifecycleComponent(t, resp.Lifecycle, "jsruntime")
	var snapshot struct {
		Phase                               string         `json:"phase"`
		Closed                              bool           `json:"closed"`
		ActiveDeployments                   int            `json:"activeDeployments"`
		ResourceCount                       int            `json:"resourceCount"`
		ResourcesByType                     map[string]int `json:"resourcesByType"`
		BridgeSubscriptions                 int            `json:"bridgeSubscriptions"`
		RuntimeHostPropagationSubscriptions int            `json:"runtimeHostPropagationSubscriptions"`
	}
	require.NoError(t, json.Unmarshal(component.Data, &snapshot))
	require.Equal(t, "active", snapshot.Phase)
	require.False(t, snapshot.Closed)
	require.Zero(t, snapshot.ActiveDeployments)
	require.Zero(t, snapshot.ResourceCount)
	require.Zero(t, snapshot.BridgeSubscriptions)
	require.Equal(t, 2, snapshot.RuntimeHostPropagationSubscriptions)

	resources, err := k.kernel.Deploy(ctx, "lifecycle-debug.ts", `bus.subscribe("lifecycle.debug.event", async () => {});`)
	require.NoError(t, err)
	require.NotEmpty(t, resources)

	resp, err = controlmod.CallKitLifecycle(k, ctx, controlmod.KitLifecycleMsg{}, sdk.WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	component = requireLifecycleComponent(t, resp.Lifecycle, "jsruntime")
	require.NoError(t, json.Unmarshal(component.Data, &snapshot))
	require.Equal(t, "active", snapshot.Phase)
	require.False(t, snapshot.Closed)
	require.Equal(t, 1, snapshot.ActiveDeployments)
	require.GreaterOrEqual(t, snapshot.ResourceCount, 1)
	require.GreaterOrEqual(t, snapshot.ResourcesByType["subscription"], 1)
	require.GreaterOrEqual(t, snapshot.BridgeSubscriptions, 1)
	require.Equal(t, 2, snapshot.RuntimeHostPropagationSubscriptions)

	require.NoError(t, k.Unmount(ctx, "jsruntime"))
	resp, err = controlmod.CallKitLifecycle(k, ctx, controlmod.KitLifecycleMsg{}, sdk.WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	requireNoLifecycleComponent(t, resp.Lifecycle, "jsruntime")
}

func TestPluginsRegistersLifecycleDebugSnapshot(t *testing.T) {
	registerEmbeddedNATSForRemountTest()
	testID := fmt.Sprintf("plugins-lifecycle-debug-%d", time.Now().UnixNano())
	k, err := New(Config{
		Transport: EmbeddedNATS(WithNATSName(testID)),
		Namespace: testID,
		CallerID:  testID,
		Modules: []bkmodule.Module{
			controlmod.New(),
			pluginsmod.NewModule(pluginsmod.Config{}),
		},
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	resp, err := controlmod.CallKitLifecycle(k, ctx, controlmod.KitLifecycleMsg{}, sdk.WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	component := requireLifecycleComponent(t, resp.Lifecycle, "plugins")
	var snapshot struct {
		ConfiguredPlugins            int  `json:"configuredPlugins"`
		RunningProcesses             int  `json:"runningProcesses"`
		StoppingProcesses            int  `json:"stoppingProcesses"`
		RestartCount                 int  `json:"restartCount"`
		RegisteredPlugins            int  `json:"registeredPlugins"`
		RegisteredTools              int  `json:"registeredTools"`
		PluginToolOwners             int  `json:"pluginToolOwners"`
		PluginToolReferences         int  `json:"pluginToolReferences"`
		ReplayTimers                 int  `json:"replayTimers"`
		PluginCheckerLeaseAttached   bool `json:"pluginCheckerLeaseAttached"`
		PluginRestarterLeaseAttached bool `json:"pluginRestarterLeaseAttached"`
		WebSocket                    struct {
			Listening         bool `json:"listening"`
			ActiveConnections int  `json:"activeConnections"`
			PendingToolCalls  int  `json:"pendingToolCalls"`
			Subscriptions     int  `json:"subscriptions"`
			RegisteredTools   int  `json:"registeredTools"`
		} `json:"websocket"`
	}
	require.NoError(t, json.Unmarshal(component.Data, &snapshot))
	require.Zero(t, snapshot.ConfiguredPlugins)
	require.Zero(t, snapshot.RunningProcesses)
	require.Zero(t, snapshot.StoppingProcesses)
	require.Zero(t, snapshot.RestartCount)
	require.Zero(t, snapshot.RegisteredPlugins)
	require.Zero(t, snapshot.RegisteredTools)
	require.Zero(t, snapshot.PluginToolOwners)
	require.Zero(t, snapshot.PluginToolReferences)
	require.Zero(t, snapshot.ReplayTimers)
	require.True(t, snapshot.PluginCheckerLeaseAttached)
	require.True(t, snapshot.PluginRestarterLeaseAttached)
	require.False(t, snapshot.WebSocket.Listening)
	require.Zero(t, snapshot.WebSocket.ActiveConnections)
	require.Zero(t, snapshot.WebSocket.PendingToolCalls)
	require.Zero(t, snapshot.WebSocket.Subscriptions)
	require.Zero(t, snapshot.WebSocket.RegisteredTools)

	require.NoError(t, k.Unmount(ctx, "plugins"))
	resp, err = controlmod.CallKitLifecycle(k, ctx, controlmod.KitLifecycleMsg{}, sdk.WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	requireNoLifecycleComponent(t, resp.Lifecycle, "plugins")
}

func TestSchedulesRegistersLifecycleDebugSnapshot(t *testing.T) {
	k, err := New(Config{
		Transport: Memory(),
		Modules: []bkmodule.Module{
			controlmod.New(),
			schedulesmod.NewModule(schedulesmod.Config{}),
		},
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	resp, err := controlmod.CallKitLifecycle(k, ctx, controlmod.KitLifecycleMsg{}, sdk.WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	component := requireLifecycleComponent(t, resp.Lifecycle, "schedules")
	var snapshot struct {
		ConfiguredSchedules          int  `json:"configuredSchedules"`
		ActiveTimers                 int  `json:"activeTimers"`
		OneTimeSchedules             int  `json:"oneTimeSchedules"`
		RepeatingSchedules           int  `json:"repeatingSchedules"`
		Closed                       bool `json:"closed"`
		StoreConfigured              bool `json:"storeConfigured"`
		ScheduleHandlerLeaseAttached bool `json:"scheduleHandlerLeaseAttached"`
	}
	require.NoError(t, json.Unmarshal(component.Data, &snapshot))
	require.Zero(t, snapshot.ConfiguredSchedules)
	require.Zero(t, snapshot.ActiveTimers)
	require.False(t, snapshot.Closed)
	require.False(t, snapshot.StoreConfigured)
	require.True(t, snapshot.ScheduleHandlerLeaseAttached)

	created, err := schedulemsg.CallScheduleCreate(k, ctx, schedulemsg.ScheduleCreateMsg{
		Expression: "in 1h",
		Topic:      "lifecycle.debug.scheduled",
		Payload:    json.RawMessage(`{}`),
	}, sdk.WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	resp, err = controlmod.CallKitLifecycle(k, ctx, controlmod.KitLifecycleMsg{}, sdk.WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	component = requireLifecycleComponent(t, resp.Lifecycle, "schedules")
	require.NoError(t, json.Unmarshal(component.Data, &snapshot))
	require.Equal(t, 1, snapshot.ConfiguredSchedules)
	require.Equal(t, 1, snapshot.ActiveTimers)
	require.Equal(t, 1, snapshot.OneTimeSchedules)
	require.Zero(t, snapshot.RepeatingSchedules)
	require.False(t, snapshot.Closed)
	require.True(t, snapshot.ScheduleHandlerLeaseAttached)

	require.NoError(t, k.Unmount(ctx, "schedules"))
	resp, err = controlmod.CallKitLifecycle(k, ctx, controlmod.KitLifecycleMsg{}, sdk.WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	requireNoLifecycleComponent(t, resp.Lifecycle, "schedules")
}

func TestTracingRegistersLifecycleDebugSnapshot(t *testing.T) {
	store := &remountTraceStore{}
	k, err := New(Config{
		Transport: Memory(),
		Modules: []bkmodule.Module{
			controlmod.New(),
			tracingmod.New(tracingmod.Config{Store: store}),
		},
	})
	require.NoError(t, err)
	defer k.Close()
	require.Same(t, store, k.kernel.Tracer().Store())

	ctx := context.Background()
	resp, err := controlmod.CallKitLifecycle(k, ctx, controlmod.KitLifecycleMsg{}, sdk.WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	component := requireLifecycleComponent(t, resp.Lifecycle, "tracing")
	var snapshot struct {
		StoreConfigured         bool   `json:"storeConfigured"`
		StoreAttached           bool   `json:"storeAttached"`
		StoreType               string `json:"storeType"`
		StoreCloseable          bool   `json:"storeCloseable"`
		TraceStoreLeaseAttached bool   `json:"traceStoreLeaseAttached"`
	}
	require.NoError(t, json.Unmarshal(component.Data, &snapshot))
	require.True(t, snapshot.StoreConfigured)
	require.True(t, snapshot.StoreAttached)
	require.True(t, snapshot.StoreCloseable)
	require.True(t, snapshot.TraceStoreLeaseAttached)
	require.Contains(t, snapshot.StoreType, "remountTraceStore")

	require.NoError(t, k.Unmount(ctx, "tracing"))
	require.Nil(t, k.kernel.Tracer().Store())
	require.True(t, store.closed)
	resp, err = controlmod.CallKitLifecycle(k, ctx, controlmod.KitLifecycleMsg{}, sdk.WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	requireNoLifecycleComponent(t, resp.Lifecycle, "tracing")
}

func requireLifecycleComponent(t *testing.T, snapshot bkmodule.LifecycleDebugSnapshot, name string) bkmodule.LifecycleDebugComponent {
	t.Helper()
	for _, component := range snapshot.Components {
		if component.Name == name {
			require.Empty(t, component.Error)
			require.NotEmpty(t, component.Data)
			return component
		}
	}
	t.Fatalf("lifecycle component %q not found in %#v", name, snapshot.Components)
	return bkmodule.LifecycleDebugComponent{}
}

func requireNoLifecycleComponent(t *testing.T, snapshot bkmodule.LifecycleDebugSnapshot, name string) {
	t.Helper()
	for _, component := range snapshot.Components {
		require.NotEqual(t, name, component.Name)
	}
}
