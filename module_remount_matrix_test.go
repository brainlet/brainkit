package brainkit

import (
	"context"
	"net"
	"net/http"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	auditpkg "github.com/brainlet/brainkit/internal/audit"
	"github.com/brainlet/brainkit/internal/tracing"
	coretransport "github.com/brainlet/brainkit/internal/transport"
	bkmodule "github.com/brainlet/brainkit/module"
	harnesscap "github.com/brainlet/brainkit/modulecap/harness"
	transporthost "github.com/brainlet/brainkit/modulehost/transporthost"
	auditmod "github.com/brainlet/brainkit/modules/audit"
	discoverymod "github.com/brainlet/brainkit/modules/discovery"
	gatewaymod "github.com/brainlet/brainkit/modules/gateway"
	harnessmod "github.com/brainlet/brainkit/modules/harness"
	jsruntimemod "github.com/brainlet/brainkit/modules/jsruntime"
	mcpmod "github.com/brainlet/brainkit/modules/mcp"
	pluginsmod "github.com/brainlet/brainkit/modules/plugins"
	probesmod "github.com/brainlet/brainkit/modules/probes"
	schedulesmod "github.com/brainlet/brainkit/modules/schedules"
	tracingmod "github.com/brainlet/brainkit/modules/tracing"
	workflowmod "github.com/brainlet/brainkit/modules/workflow"
	"github.com/stretchr/testify/require"
)

var registerRemountEmbeddedNATS sync.Once

func TestResourceModuleHotRemountMatrix(t *testing.T) {
	t.Run("schedules", func(t *testing.T) {
		k := newRemountTestKit(t, Config{Transport: Memory()})
		ctx := context.Background()

		mod := schedulesmod.NewModule(schedulesmod.Config{})
		require.NoError(t, k.Mount(ctx, mod))
		assertMountedModuleSurface(t, k, "schedules", []string{
			"schedules.cancel",
			"schedules.create",
			"schedules.list",
		}, []string{
			"schedules.scheduler",
			"schedule-handler",
		})
		assertLifecycleComponentMounted(t, k, "schedules")
		require.NotNil(t, k.kernel.ScheduleHandler())
		scheduleSnapshot := mod.DebugSnapshot()
		require.False(t, scheduleSnapshot.Closed)
		require.True(t, scheduleSnapshot.ScheduleHandlerLeaseAttached)

		require.NoError(t, k.Unmount(ctx, "schedules"))
		assertUnmountedModuleSurface(t, k, "schedules", []string{
			"schedules.cancel",
			"schedules.create",
			"schedules.list",
		})
		assertLifecycleComponentUnmounted(t, k, "schedules")
		require.Nil(t, k.kernel.ScheduleHandler())
		scheduleSnapshot = mod.DebugSnapshot()
		require.True(t, scheduleSnapshot.Closed)
		require.False(t, scheduleSnapshot.ScheduleHandlerLeaseAttached)
		require.Zero(t, scheduleSnapshot.ActiveTimers)

		nextMod := schedulesmod.NewModule(schedulesmod.Config{})
		require.NoError(t, k.Mount(ctx, nextMod))
		assertMountedModuleSurface(t, k, "schedules", []string{
			"schedules.cancel",
			"schedules.create",
			"schedules.list",
		}, []string{
			"schedules.scheduler",
			"schedule-handler",
		})
		assertLifecycleComponentMounted(t, k, "schedules")
		require.NotNil(t, k.kernel.ScheduleHandler())
		require.True(t, nextMod.DebugSnapshot().ScheduleHandlerLeaseAttached)
	})

	t.Run("tracing", func(t *testing.T) {
		k := newRemountTestKit(t, Config{Transport: Memory()})
		ctx := context.Background()
		store := &remountTraceStore{}

		mod := tracingmod.New(tracingmod.Config{Store: store})
		require.NoError(t, k.Mount(ctx, mod))
		assertMountedModuleSurface(t, k, "tracing", []string{
			"trace.get",
			"trace.list",
		}, []string{
			"tracing.store",
		})
		assertLifecycleComponentMounted(t, k, "tracing")
		require.Same(t, store, k.kernel.Tracer().Store())
		traceSnapshot := mod.DebugSnapshot()
		require.True(t, traceSnapshot.StoreAttached)
		require.True(t, traceSnapshot.TraceStoreLeaseAttached)

		require.NoError(t, k.Unmount(ctx, "tracing"))
		assertUnmountedModuleSurface(t, k, "tracing", []string{
			"trace.get",
			"trace.list",
		})
		assertLifecycleComponentUnmounted(t, k, "tracing")
		require.Nil(t, k.kernel.Tracer().Store())
		require.True(t, store.closed)
		traceSnapshot = mod.DebugSnapshot()
		require.False(t, traceSnapshot.StoreAttached)
		require.False(t, traceSnapshot.TraceStoreLeaseAttached)

		nextStore := &remountTraceStore{}
		nextMod := tracingmod.New(tracingmod.Config{Store: nextStore})
		require.NoError(t, k.Mount(ctx, nextMod))
		assertMountedModuleSurface(t, k, "tracing", []string{
			"trace.get",
			"trace.list",
		}, []string{
			"tracing.store",
		})
		assertLifecycleComponentMounted(t, k, "tracing")
		require.Same(t, nextStore, k.kernel.Tracer().Store())
		require.False(t, nextStore.closed)
		require.True(t, nextMod.DebugSnapshot().TraceStoreLeaseAttached)
	})

	t.Run("audit", func(t *testing.T) {
		k := newRemountTestKit(t, Config{Transport: Memory()})
		ctx := context.Background()
		store := &remountAuditStore{}

		mod := auditmod.NewModule(auditmod.Config{Store: store, Verbose: true, OwnStore: true})
		require.NoError(t, k.Mount(ctx, mod))
		assertMountedModuleSurface(t, k, "audit", []string{
			"audit.prune",
			"audit.query",
			"audit.stats",
		}, []string{
			"audit.store",
		})
		assertLifecycleComponentMounted(t, k, "audit")
		require.Same(t, store, k.kernel.Audit().Store())
		require.Equal(t, auditpkg.VerbosityVerbose, k.kernel.Audit().Verbosity())
		auditSnapshot := mod.DebugSnapshot()
		require.True(t, auditSnapshot.StoreAttached)
		require.True(t, auditSnapshot.StoreLeaseAttached)
		require.True(t, auditSnapshot.VerbosityLeaseAttached)

		require.NoError(t, k.Unmount(ctx, "audit"))
		assertUnmountedModuleSurface(t, k, "audit", []string{
			"audit.prune",
			"audit.query",
			"audit.stats",
		})
		assertLifecycleComponentUnmounted(t, k, "audit")
		require.Nil(t, k.kernel.Audit().Store())
		require.Equal(t, auditpkg.VerbosityNormal, k.kernel.Audit().Verbosity())
		require.True(t, store.closed)
		auditSnapshot = mod.DebugSnapshot()
		require.False(t, auditSnapshot.StoreAttached)
		require.False(t, auditSnapshot.StoreLeaseAttached)
		require.False(t, auditSnapshot.VerbosityLeaseAttached)

		nextStore := &remountAuditStore{}
		nextMod := auditmod.NewModule(auditmod.Config{Store: nextStore, OwnStore: true})
		require.NoError(t, k.Mount(ctx, nextMod))
		assertMountedModuleSurface(t, k, "audit", []string{
			"audit.prune",
			"audit.query",
			"audit.stats",
		}, []string{
			"audit.store",
		})
		assertLifecycleComponentMounted(t, k, "audit")
		require.Same(t, nextStore, k.kernel.Audit().Store())
		require.True(t, nextMod.DebugSnapshot().StoreLeaseAttached)
		require.False(t, nextMod.DebugSnapshot().VerbosityLeaseAttached)
	})

	t.Run("workflow keeps jsruntime dependency mounted", func(t *testing.T) {
		k := newRemountTestKit(t, Config{Transport: Memory()})
		ctx := context.Background()

		require.NoError(t, k.Mount(ctx, jsruntimemod.New()))
		require.NoError(t, k.Mount(ctx, workflowmod.New()))
		assertLifecycleComponentMounted(t, k, "jsruntime")
		assertMountedModuleSurface(t, k, "workflow", []string{
			"workflow.cancel",
			"workflow.list",
			"workflow.restart",
			"workflow.resume",
			"workflow.runs",
			"workflow.start",
			"workflow.startAsync",
			"workflow.status",
		}, nil)

		require.NoError(t, k.Unmount(ctx, "workflow"))
		assertUnmountedModuleSurface(t, k, "workflow", []string{
			"workflow.cancel",
			"workflow.list",
			"workflow.restart",
			"workflow.resume",
			"workflow.runs",
			"workflow.start",
			"workflow.startAsync",
			"workflow.status",
		})
		_, jsMounted := k.Module("jsruntime")
		require.True(t, jsMounted)
		assertLifecycleComponentMounted(t, k, "jsruntime")

		require.NoError(t, k.Mount(ctx, workflowmod.New()))
		assertMountedModuleSurface(t, k, "workflow", []string{
			"workflow.cancel",
			"workflow.list",
			"workflow.restart",
			"workflow.resume",
			"workflow.runs",
			"workflow.start",
			"workflow.startAsync",
			"workflow.status",
		}, nil)
	})

	t.Run("discovery bus", func(t *testing.T) {
		k := newRemountTestKit(t, Config{
			Transport: Memory(),
			Namespace: "discovery-remount",
			CallerID:  "discovery-remount",
		})
		ctx := context.Background()

		mod := discoverymod.NewModule(discoverymod.ModuleConfig{
			Type:      "bus",
			Name:      "discovery-remount",
			Heartbeat: time.Hour,
			TTL:       time.Hour,
		})
		require.NoError(t, k.Mount(ctx, mod))
		assertMountedModuleSurface(t, k, "discovery", nil, []string{
			"discovery.provider",
		})
		assertLifecycleComponentMounted(t, k, "discovery")
		provider, ok := mod.Provider().(*discoverymod.Bus)
		require.True(t, ok)
		discoverySnapshot := mod.DebugSnapshot()
		require.True(t, discoverySnapshot.ProviderAttached)
		require.NotNil(t, discoverySnapshot.Bus)
		require.True(t, discoverySnapshot.Bus.SubscriptionAttached)
		require.Equal(t, int64(2), discoverySnapshot.Bus.ActiveLoops)
		require.False(t, discoverySnapshot.Bus.Closed)

		require.NoError(t, k.Unmount(ctx, "discovery"))
		assertUnmountedModuleSurface(t, k, "discovery", nil)
		assertLifecycleComponentUnmounted(t, k, "discovery")
		require.Nil(t, mod.Provider())
		busSnapshot := provider.DebugSnapshot()
		require.True(t, busSnapshot.Closed)
		require.False(t, busSnapshot.SubscriptionAttached)
		require.Zero(t, busSnapshot.ActiveLoops)

		nextMod := discoverymod.NewModule(discoverymod.ModuleConfig{
			Type:      "bus",
			Name:      "discovery-remount-next",
			Heartbeat: time.Hour,
			TTL:       time.Hour,
		})
		require.NoError(t, k.Mount(ctx, nextMod))
		assertMountedModuleSurface(t, k, "discovery", nil, []string{
			"discovery.provider",
		})
		assertLifecycleComponentMounted(t, k, "discovery")
		require.True(t, nextMod.DebugSnapshot().Bus.SubscriptionAttached)
		require.Equal(t, int64(2), nextMod.DebugSnapshot().Bus.ActiveLoops)
	})

	t.Run("harness closes scoped instance", func(t *testing.T) {
		k := newRemountTestKit(t, Config{Transport: Memory()})
		ctx := context.Background()
		runtime := newRemountHarnessRuntime(t)

		require.NoError(t, k.Mount(ctx, harnessRuntimeCapabilityModule{runtime: runtime}))
		mod := harnessmod.NewModule(harnessmod.Config{Harness: remountHarnessConfig()})
		require.NoError(t, k.Mount(ctx, mod))
		assertMountedModuleSurface(t, k, "harness", nil, []string{
			"harness.instance",
		})
		assertLifecycleComponentMounted(t, k, "harness")
		require.NotNil(t, mod.Instance())
		harnessSnapshot := mod.DebugSnapshot()
		require.True(t, harnessSnapshot.InstanceAttached)
		require.True(t, harnessSnapshot.Initialized)
		require.False(t, harnessSnapshot.Closing)
		require.False(t, harnessSnapshot.Closed)

		require.NoError(t, k.Unmount(ctx, "harness"))
		assertUnmountedModuleSurface(t, k, "harness", nil)
		assertLifecycleComponentUnmounted(t, k, "harness")
		require.Nil(t, mod.Instance())
		harnessSnapshot = mod.DebugSnapshot()
		require.False(t, harnessSnapshot.InstanceAttached)
		require.Zero(t, harnessSnapshot.ActiveHeartbeatWorkers)
		_, jsMounted := k.Module("jsruntime")
		require.True(t, jsMounted)

		nextMod := harnessmod.NewModule(harnessmod.Config{Harness: remountHarnessConfig()})
		require.NoError(t, k.Mount(ctx, nextMod))
		assertMountedModuleSurface(t, k, "harness", nil, []string{
			"harness.instance",
		})
		assertLifecycleComponentMounted(t, k, "harness")
		require.NotNil(t, nextMod.Instance())
		require.True(t, nextMod.DebugSnapshot().InstanceAttached)
	})

	t.Run("mcp closes manager and keeps tools dependency mounted", func(t *testing.T) {
		k := newRemountTestKit(t, Config{Transport: Memory()})
		ctx := context.Background()

		mod := mcpmod.New(map[string]mcpmod.ServerConfig{
			"missing": {Command: "/definitely/missing/brainkit-mcp-test-server"},
		})
		require.NoError(t, k.Mount(ctx, mod))
		assertMountedModuleSurface(t, k, "mcp", []string{
			"mcp.callTool",
			"mcp.listTools",
		}, []string{
			"mcp.servers",
		})
		assertLifecycleComponentMounted(t, k, "mcp")
		mcpSnapshot := mod.DebugSnapshot()
		require.True(t, mcpSnapshot.ManagerAttached)
		require.Equal(t, 1, mcpSnapshot.ConfiguredServers)
		require.Zero(t, mcpSnapshot.ConnectedServers)
		require.Zero(t, mcpSnapshot.CachedTools)

		require.NoError(t, k.Unmount(ctx, "mcp"))
		assertUnmountedModuleSurface(t, k, "mcp", []string{
			"mcp.callTool",
			"mcp.listTools",
		})
		assertLifecycleComponentUnmounted(t, k, "mcp")
		require.False(t, mod.DebugSnapshot().ManagerAttached)
		_, toolsMounted := k.Module("tools")
		require.True(t, toolsMounted)

		nextMod := mcpmod.New(map[string]mcpmod.ServerConfig{
			"missing": {Command: "/definitely/missing/brainkit-mcp-test-server"},
		})
		require.NoError(t, k.Mount(ctx, nextMod))
		assertMountedModuleSurface(t, k, "mcp", []string{
			"mcp.callTool",
			"mcp.listTools",
		}, []string{
			"mcp.servers",
		})
		assertLifecycleComponentMounted(t, k, "mcp")
		require.True(t, nextMod.DebugSnapshot().ManagerAttached)
	})

	t.Run("gateway", func(t *testing.T) {
		k := newRemountTestKit(t, Config{Transport: Memory()})
		ctx := context.Background()

		gw := gatewaymod.New(gatewaymod.Config{Listen: "127.0.0.1:0"})
		require.NoError(t, k.Mount(ctx, gw))
		assertMountedModuleSurface(t, k, "gateway", nil, []string{
			"gateway.listener",
			"gateway.routes",
		})
		assertLifecycleComponentMounted(t, k, "gateway")
		requireHTTPStatus(t, "http://"+gw.Addr()+"/healthz", http.StatusOK)
		gatewaySnapshot := gw.DebugSnapshot()
		require.True(t, gatewaySnapshot.Listening)
		require.Equal(t, 4, gatewaySnapshot.RouteSubscriptions)
		require.Zero(t, gatewaySnapshot.StreamSessions)
		require.Zero(t, gatewaySnapshot.StreamSubscriptions)
		addr := gw.Addr()

		require.NoError(t, k.Unmount(ctx, "gateway"))
		assertUnmountedModuleSurface(t, k, "gateway", nil)
		assertLifecycleComponentUnmounted(t, k, "gateway")
		requireEventuallyPortClosed(t, addr)
		requireEventuallyGatewayStopped(t, gw)

		nextGW := gatewaymod.New(gatewaymod.Config{Listen: "127.0.0.1:0"})
		require.NoError(t, k.Mount(ctx, nextGW))
		assertMountedModuleSurface(t, k, "gateway", nil, []string{
			"gateway.listener",
			"gateway.routes",
		})
		assertLifecycleComponentMounted(t, k, "gateway")
		requireHTTPStatus(t, "http://"+nextGW.Addr()+"/healthz", http.StatusOK)
		require.True(t, nextGW.DebugSnapshot().Listening)
	})

	t.Run("plugins keeps tools dependency mounted", func(t *testing.T) {
		registerEmbeddedNATSForRemountTest()
		testID := "bkremount"
		fsRoot, err := os.MkdirTemp("", "bk-remount-*")
		require.NoError(t, err)
		t.Cleanup(func() { _ = os.RemoveAll(fsRoot) })
		k := newRemountTestKit(t, Config{
			Transport: EmbeddedNATS(WithNATSName(testID)),
			Namespace: testID,
			CallerID:  testID,
			FSRoot:    fsRoot,
		})
		ctx := context.Background()

		require.NoError(t, k.Mount(ctx, pluginsmod.NewModule(pluginsmod.Config{})))
		mod, ok := k.Module("plugins")
		require.True(t, ok)
		pluginModule, ok := mod.(*pluginsmod.Module)
		require.True(t, ok)
		assertMountedModuleSurface(t, k, "plugins", []string{
			"plugin.list",
			"plugin.manifest",
			"plugin.restart",
			"plugin.start",
			"plugin.status",
			"plugin.stop",
		}, []string{
			"plugins.manager",
			"plugins.websocket",
		})
		assertLifecycleComponentMounted(t, k, "plugins")
		require.NotNil(t, k.kernel.PluginChecker())
		require.NotNil(t, k.kernel.PluginRestarter())
		pluginsSnapshot := pluginModule.DebugSnapshot()
		require.True(t, pluginsSnapshot.PluginCheckerLeaseAttached)
		require.True(t, pluginsSnapshot.PluginRestarterLeaseAttached)
		require.Zero(t, pluginsSnapshot.ReplayTimers)
		require.False(t, pluginsSnapshot.WebSocket.Listening)

		require.NoError(t, k.Unmount(ctx, "plugins"))
		assertUnmountedModuleSurface(t, k, "plugins", []string{
			"plugin.list",
			"plugin.manifest",
			"plugin.restart",
			"plugin.start",
			"plugin.status",
			"plugin.stop",
		})
		assertLifecycleComponentUnmounted(t, k, "plugins")
		require.Nil(t, k.kernel.PluginChecker())
		require.Nil(t, k.kernel.PluginRestarter())
		pluginsSnapshot = pluginModule.DebugSnapshot()
		require.False(t, pluginsSnapshot.PluginCheckerLeaseAttached)
		require.False(t, pluginsSnapshot.PluginRestarterLeaseAttached)
		require.Zero(t, pluginsSnapshot.RunningProcesses)
		require.Zero(t, pluginsSnapshot.ReplayTimers)
		require.False(t, pluginsSnapshot.WebSocket.Listening)
		_, toolsMounted := k.Module("tools")
		require.True(t, toolsMounted)

		require.NoError(t, k.Mount(ctx, pluginsmod.NewModule(pluginsmod.Config{})))
		assertMountedModuleSurface(t, k, "plugins", []string{
			"plugin.list",
			"plugin.manifest",
			"plugin.restart",
			"plugin.start",
			"plugin.status",
			"plugin.stop",
		}, []string{
			"plugins.manager",
			"plugins.websocket",
		})
		assertLifecycleComponentMounted(t, k, "plugins")
		require.NotNil(t, k.kernel.PluginChecker())
		require.NotNil(t, k.kernel.PluginRestarter())
	})

	t.Run("probes cancels scoped loop", func(t *testing.T) {
		k := newRemountTestKit(t, Config{Transport: Memory()})
		ctx := context.Background()

		mod := probesmod.New(probesmod.Config{Interval: time.Hour})
		require.NoError(t, k.Mount(ctx, mod))
		assertMountedModuleSurface(t, k, "probes", nil, []string{
			"probes.loop",
		})
		assertLifecycleComponentMounted(t, k, "probes")
		probeSnapshot := mod.DebugSnapshot()
		require.True(t, probeSnapshot.RunnerAttached)
		require.True(t, probeSnapshot.LoopRunning)

		require.NoError(t, k.Unmount(ctx, "probes"))
		assertUnmountedModuleSurface(t, k, "probes", nil)
		assertLifecycleComponentUnmounted(t, k, "probes")
		snapshot := mod.DebugSnapshot()
		require.True(t, snapshot.Closed)
		require.False(t, snapshot.RunnerAttached)
		require.Equal(t, int64(0), snapshot.ActiveSweeps)

		nextMod := probesmod.New(probesmod.Config{Interval: time.Hour})
		require.NoError(t, k.Mount(ctx, nextMod))
		assertMountedModuleSurface(t, k, "probes", nil, []string{
			"probes.loop",
		})
		assertLifecycleComponentMounted(t, k, "probes")
		require.True(t, nextMod.DebugSnapshot().RunnerAttached)
	})
}

func TestModuleHotRemountDoesNotLeakTransportState(t *testing.T) {
	k := newRemountTestKit(t, Config{Transport: Memory()})
	ctx := context.Background()
	baseline := k.kernel.TransportDebugSnapshot()

	for range 5 {
		require.NoError(t, k.Mount(ctx, schedulesmod.NewModule(schedulesmod.Config{})))
		mounted := k.kernel.TransportDebugSnapshot()
		require.GreaterOrEqual(t, mounted.Router.Handlers, baseline.Router.Handlers+3)
		require.GreaterOrEqual(t, mounted.Router.StartedHandlers, baseline.Router.StartedHandlers+3)
		require.NoError(t, k.Unmount(ctx, "schedules"))
		requireTransportDebugSnapshotEventually(t, k, baseline)
	}

	for range 3 {
		gw := gatewaymod.New(gatewaymod.Config{Listen: "127.0.0.1:0"})
		require.NoError(t, k.Mount(ctx, gw))
		mounted := k.kernel.TransportDebugSnapshot()
		require.GreaterOrEqual(t, mounted.ActiveSubscriptions, baseline.ActiveSubscriptions+4)
		addr := gw.Addr()
		require.NoError(t, k.Unmount(ctx, "gateway"))
		requireEventuallyPortClosed(t, addr)
		requireTransportDebugSnapshotEventually(t, k, baseline)
	}
}

type remountTraceStore struct {
	closed bool
}

func (s *remountTraceStore) RecordSpan(tracing.Span) error { return nil }

func (s *remountTraceStore) GetTrace(string) ([]tracing.Span, error) { return nil, nil }

func (s *remountTraceStore) ListTraces(tracing.TraceQuery) ([]tracing.TraceSummary, error) {
	return nil, nil
}

func (s *remountTraceStore) Close() error {
	s.closed = true
	return nil
}

type remountAuditStore struct {
	closed bool
}

func (s *remountAuditStore) Record(auditpkg.Event) {}

func (s *remountAuditStore) Query(auditpkg.Query) ([]auditpkg.Event, error) { return nil, nil }

func (s *remountAuditStore) Prune(time.Duration) error { return nil }

func (s *remountAuditStore) Count() (int64, error) { return 0, nil }

func (s *remountAuditStore) CountByCategory() (map[string]int64, error) { return nil, nil }

func (s *remountAuditStore) Close() error {
	s.closed = true
	return nil
}

type harnessRuntimeCapabilityModule struct {
	runtime *remountHarnessRuntime
}

func (m harnessRuntimeCapabilityModule) ID() string { return "jsruntime" }

func (m harnessRuntimeCapabilityModule) Mount(ctx context.Context, host bkmodule.Host) error {
	_, err := host.Capabilities().Provide(ctx, bkmodule.CapabilityHarnessRuntime, harnesscap.Runtime(m.runtime))
	return err
}

type remountHarnessRuntime struct{}

func newRemountHarnessRuntime(t *testing.T) *remountHarnessRuntime {
	t.Helper()
	return &remountHarnessRuntime{}
}

func (r *remountHarnessRuntime) EvalTS(context.Context, string, string) (string, error) {
	return "", nil
}

func (r *remountHarnessRuntime) RuntimeContext() context.Context { return context.Background() }

func (r *remountHarnessRuntime) RegisterEventBridge(func(string)) error { return nil }

func (r *remountHarnessRuntime) RegisterLockBridge(func(string) error, func(string) error) error {
	return nil
}

func (r *remountHarnessRuntime) EvalControl(context.Context, string, string) (string, error) {
	return "", nil
}

func remountHarnessConfig() harnessmod.HarnessConfig {
	return harnessmod.HarnessConfig{
		ID: "remount-harness",
		Modes: []harnessmod.ModeConfig{{
			ID:        "default",
			Name:      "Default",
			Default:   true,
			AgentName: "agent",
		}},
	}
}

func newRemountTestKit(t *testing.T, cfg Config) *Kit {
	t.Helper()
	k, err := New(cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, k.Close())
	})
	return k
}

func registerEmbeddedNATSForRemountTest() {
	registerRemountEmbeddedNATS.Do(func() {
		RegisterTransportBuilder("embedded", func(ctx TransportBuildContext) (any, error) {
			transportSet, err := coretransport.NewTransportSet(coretransport.TransportConfig{
				Type:      "memory",
				Namespace: ctx.Namespace,
			})
			if err != nil {
				return nil, err
			}
			transportSet.Kind = ctx.Config.Kind()
			return transportSet, nil
		})
	})
}

func assertMountedModuleSurface(t *testing.T, k *Kit, id string, commands []string, resources []string) {
	t.Helper()
	_, mounted := k.Module(id)
	require.Truef(t, mounted, "expected module %q to be mounted", id)
	for _, topic := range commands {
		require.Truef(t, k.hasCommand(topic), "expected command %q to be registered", topic)
	}
	desc, ok := mountedDescriptor(k, id)
	require.Truef(t, ok, "expected mounted descriptor for %q", id)
	resourceNames := resourceNames(desc.Resources)
	for _, name := range resources {
		require.Contains(t, resourceNames, name)
	}
}

func assertUnmountedModuleSurface(t *testing.T, k *Kit, id string, commands []string) {
	t.Helper()
	_, mounted := k.Module(id)
	require.Falsef(t, mounted, "expected module %q to be unmounted", id)
	for _, topic := range commands {
		require.Falsef(t, k.hasCommand(topic), "expected command %q to be unregistered", topic)
	}
	_, ok := mountedDescriptor(k, id)
	require.Falsef(t, ok, "expected mounted descriptor for %q to be removed", id)
	requireLifecycleNoTransientWork(t, k)
}

func assertLifecycleComponentMounted(t *testing.T, k *Kit, name string) {
	t.Helper()
	requireLifecycleComponent(t, k.lifecycleDebugSnapshot(), name)
}

func assertLifecycleComponentUnmounted(t *testing.T, k *Kit, name string) {
	t.Helper()
	requireNoLifecycleComponent(t, k.lifecycleDebugSnapshot(), name)
}

func mountedDescriptor(k *Kit, id string) (bkmodule.Descriptor, bool) {
	for _, desc := range k.MountedModules() {
		if desc.Name == id {
			return desc, true
		}
	}
	return bkmodule.Descriptor{}, false
}

func resourceNames(resources []bkmodule.ResourceDescriptor) []string {
	out := make([]string, 0, len(resources))
	for _, res := range resources {
		out = append(out, res.Name)
	}
	return out
}

func requireEventuallyPortClosed(t *testing.T, addr string) {
	t.Helper()
	require.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err != nil {
			return true
		}
		_ = conn.Close()
		return false
	}, 2*time.Second, 20*time.Millisecond)
}

func requireEventuallyGatewayStopped(t *testing.T, gw *gatewaymod.Gateway) {
	t.Helper()
	require.Eventually(t, func() bool {
		snapshot := gw.DebugSnapshot()
		return !snapshot.Listening &&
			snapshot.RouteSubscriptions == 0 &&
			snapshot.StreamSessions == 0 &&
			snapshot.StreamSubscriptions == 0
	}, 2*time.Second, 20*time.Millisecond)
}

func requireHTTPStatus(t *testing.T, url string, status int) {
	t.Helper()
	client := http.Client{Timeout: time.Second}
	resp, err := client.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, status, resp.StatusCode)
}

func requireTransportDebugSnapshotEventually(t *testing.T, k *Kit, want transporthost.DebugSnapshot) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if transportDebugSnapshotEqual(k.kernel.TransportDebugSnapshot(), want) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.Equal(t, want, k.kernel.TransportDebugSnapshot())
}

func requireLifecycleNoTransientWork(t *testing.T, k *Kit) {
	t.Helper()
	require.Eventually(t, func() bool {
		return lifecycleHasNoTransientWork(k.lifecycleDebugSnapshot().Runtime)
	}, 2*time.Second, 20*time.Millisecond, "runtime lifecycle still has transient work: %#v", k.lifecycleDebugSnapshot().Runtime)
}

func lifecycleHasNoTransientWork(runtime bkmodule.LifecycleRuntimeDebug) bool {
	return runtime.ActiveHandlers == 0 &&
		runtime.Provider.ActiveProbes == 0 &&
		runtime.Provider.ActiveOperations == 0 &&
		runtime.Storage.ActiveCloses == 0 &&
		runtime.Transport.ActiveStreamHeartbeats == 0 &&
		runtime.Transport.CallerPendingCalls == 0 &&
		runtime.Transport.CallerStreamDrains == 0
}

func transportDebugSnapshotEqual(got, want transporthost.DebugSnapshot) bool {
	return got.ActiveSubscriptions == want.ActiveSubscriptions &&
		got.OwnsTransport == want.OwnsTransport &&
		got.ClosingRouter == want.ClosingRouter &&
		got.ClosingCaller == want.ClosingCaller &&
		got.ClosingTransport == want.ClosingTransport &&
		got.ClosedRouter == want.ClosedRouter &&
		got.ClosedCaller == want.ClosedCaller &&
		got.ClosedTransport == want.ClosedTransport &&
		got.Router.Handlers == want.Router.Handlers &&
		got.Router.StartedHandlers == want.Router.StartedHandlers &&
		got.Router.StoppedHandlers == want.Router.StoppedHandlers &&
		reflect.DeepEqual(got.Router.Topics, want.Router.Topics)
}
