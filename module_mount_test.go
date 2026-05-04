package brainkit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	auditpkg "github.com/brainlet/brainkit/internal/audit"
	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
	controlmod "github.com/brainlet/brainkit/modules/control"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/stretchr/testify/require"
)

type hotEchoMsg struct {
	Text string `json:"text"`
}

func (hotEchoMsg) BusTopic() string { return "test.hot.echo" }

type hotEchoResp struct {
	Text string `json:"text"`
}

type runtimeLeaseScheduleHandler struct {
	name string
}

func (h *runtimeLeaseScheduleHandler) Schedule(context.Context, types.ScheduleConfig) (string, error) {
	return h.name, nil
}

func (h *runtimeLeaseScheduleHandler) Unschedule(context.Context, string) error { return nil }

func (h *runtimeLeaseScheduleHandler) List() []types.PersistedSchedule { return nil }

type runtimeLeaseAuditStore struct {
	name string
}

func (s *runtimeLeaseAuditStore) Record(auditpkg.Event) {}

func (s *runtimeLeaseAuditStore) Query(auditpkg.Query) ([]auditpkg.Event, error) {
	return nil, nil
}

func (s *runtimeLeaseAuditStore) Prune(time.Duration) error { return nil }

func (s *runtimeLeaseAuditStore) Count() (int64, error) { return 0, nil }

func (s *runtimeLeaseAuditStore) CountByCategory() (map[string]int64, error) {
	return nil, nil
}

func (s *runtimeLeaseAuditStore) Close() error { return nil }

type runtimeLeaseTraceStore struct {
	name string
}

func (s *runtimeLeaseTraceStore) RecordSpan(types.Span) error { return nil }

func (s *runtimeLeaseTraceStore) GetTrace(string) ([]types.Span, error) { return nil, nil }

func (s *runtimeLeaseTraceStore) ListTraces(types.TraceQuery) ([]types.TraceSummary, error) {
	return nil, nil
}

func (s *runtimeLeaseTraceStore) Close() error { return nil }

type runtimeLeasePluginHook struct {
	name string
}

func (h *runtimeLeasePluginHook) IsPluginRunning(name string) bool { return name == h.name }

func (h *runtimeLeasePluginHook) ListRunningPlugins() []types.RunningPlugin { return nil }

func (h *runtimeLeasePluginHook) RestartPlugin(context.Context, string) error { return nil }

type runtimeLeaseJSEvaluator struct {
	name string
}

func (e *runtimeLeaseJSEvaluator) EvalOnJSThread(_, _ string) (string, error) {
	return e.name, nil
}

func requireRuntimeLeaseSchedule(t *testing.T, k *Kit, want string) {
	t.Helper()
	got, ok := k.kernel.ScheduleHandler().(*runtimeLeaseScheduleHandler)
	require.True(t, ok)
	require.Equal(t, want, got.name)
}

func requireRuntimeLeaseAuditStore(t *testing.T, k *Kit, want string) {
	t.Helper()
	got, ok := k.kernel.Audit().Store().(*runtimeLeaseAuditStore)
	require.True(t, ok)
	require.Equal(t, want, got.name)
}

func requireRuntimeLeaseTraceStore(t *testing.T, k *Kit, want string) {
	t.Helper()
	got, ok := k.kernel.Tracer().Store().(*runtimeLeaseTraceStore)
	require.True(t, ok)
	require.Equal(t, want, got.name)
}

func requireRuntimeLeasePluginChecker(t *testing.T, k *Kit, want string) {
	t.Helper()
	got, ok := k.kernel.PluginChecker().(*runtimeLeasePluginHook)
	require.True(t, ok)
	require.Equal(t, want, got.name)
}

func requireRuntimeLeasePluginRestarter(t *testing.T, k *Kit, want string) {
	t.Helper()
	got, ok := k.kernel.PluginRestarter().(*runtimeLeasePluginHook)
	require.True(t, ok)
	require.Equal(t, want, got.name)
}

func requireRuntimeLeaseToolEvaluator(t *testing.T, k *Kit, want string) {
	t.Helper()
	got := k.kernel.ToolsDomain().Evaluator()
	require.NotNil(t, got)
	name, err := got.EvalOnJSThread("", "")
	require.NoError(t, err)
	require.Equal(t, want, name)
}

type hotEchoModule struct {
	id     string
	suffix string
}

func (m hotEchoModule) ID() string { return m.id }

func (m hotEchoModule) Mount(_ context.Context, host bkmodule.Host) error {
	_, err := host.Commands().Handle(bkmodule.Command(func(_ context.Context, req hotEchoMsg) (*hotEchoResp, error) {
		return &hotEchoResp{Text: req.Text + m.suffix}, nil
	}))
	return err
}

func TestRuntimeHookLeasesIgnoreStaleCloseAndRestoreActiveLease(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()

	scheduleOne, err := k.kernel.LeaseScheduleHandler(ctx, &runtimeLeaseScheduleHandler{name: "schedule-one"})
	require.NoError(t, err)
	scheduleTwo, err := k.kernel.LeaseScheduleHandler(ctx, &runtimeLeaseScheduleHandler{name: "schedule-two"})
	require.NoError(t, err)
	requireRuntimeLeaseSchedule(t, k, "schedule-two")
	require.NoError(t, scheduleOne.Close(ctx))
	requireRuntimeLeaseSchedule(t, k, "schedule-two")
	require.NoError(t, scheduleTwo.Close(ctx))
	require.Nil(t, k.kernel.ScheduleHandler())

	auditOne, err := k.kernel.LeaseAuditStore(ctx, &runtimeLeaseAuditStore{name: "audit-one"})
	require.NoError(t, err)
	auditTwo, err := k.kernel.LeaseAuditStore(ctx, &runtimeLeaseAuditStore{name: "audit-two"})
	require.NoError(t, err)
	require.NoError(t, auditOne.Close(ctx))
	requireRuntimeLeaseAuditStore(t, k, "audit-two")
	require.NoError(t, auditTwo.Close(ctx))
	require.Nil(t, k.kernel.Audit().Store())

	auditThree, err := k.kernel.LeaseAuditStore(ctx, &runtimeLeaseAuditStore{name: "audit-three"})
	require.NoError(t, err)
	auditFour, err := k.kernel.LeaseAuditStore(ctx, &runtimeLeaseAuditStore{name: "audit-four"})
	require.NoError(t, err)
	require.NoError(t, auditFour.Close(ctx))
	requireRuntimeLeaseAuditStore(t, k, "audit-three")
	require.NoError(t, auditThree.Close(ctx))
	require.Nil(t, k.kernel.Audit().Store())

	verbosityOne, err := k.kernel.LeaseAuditVerbosity(ctx, auditpkg.VerbosityVerbose)
	require.NoError(t, err)
	verbosityTwo, err := k.kernel.LeaseAuditVerbosity(ctx, auditpkg.Verbosity(42))
	require.NoError(t, err)
	require.NoError(t, verbosityOne.Close(ctx))
	require.Equal(t, auditpkg.Verbosity(42), k.kernel.Audit().Verbosity())
	require.NoError(t, verbosityTwo.Close(ctx))
	require.Equal(t, auditpkg.VerbosityNormal, k.kernel.Audit().Verbosity())

	verbosityThree, err := k.kernel.LeaseAuditVerbosity(ctx, auditpkg.VerbosityVerbose)
	require.NoError(t, err)
	verbosityFour, err := k.kernel.LeaseAuditVerbosity(ctx, auditpkg.Verbosity(42))
	require.NoError(t, err)
	require.NoError(t, verbosityFour.Close(ctx))
	require.Equal(t, auditpkg.VerbosityVerbose, k.kernel.Audit().Verbosity())
	require.NoError(t, verbosityThree.Close(ctx))
	require.Equal(t, auditpkg.VerbosityNormal, k.kernel.Audit().Verbosity())

	traceOne, err := k.kernel.LeaseTraceStore(ctx, &runtimeLeaseTraceStore{name: "trace-one"})
	require.NoError(t, err)
	traceTwo, err := k.kernel.LeaseTraceStore(ctx, &runtimeLeaseTraceStore{name: "trace-two"})
	require.NoError(t, err)
	require.NoError(t, traceOne.Close(ctx))
	requireRuntimeLeaseTraceStore(t, k, "trace-two")
	require.NoError(t, traceTwo.Close(ctx))
	require.Nil(t, k.kernel.Tracer().Store())

	traceThree, err := k.kernel.LeaseTraceStore(ctx, &runtimeLeaseTraceStore{name: "trace-three"})
	require.NoError(t, err)
	traceFour, err := k.kernel.LeaseTraceStore(ctx, &runtimeLeaseTraceStore{name: "trace-four"})
	require.NoError(t, err)
	require.NoError(t, traceFour.Close(ctx))
	requireRuntimeLeaseTraceStore(t, k, "trace-three")
	require.NoError(t, traceThree.Close(ctx))
	require.Nil(t, k.kernel.Tracer().Store())

	checkerOne, err := k.kernel.LeasePluginChecker(ctx, &runtimeLeasePluginHook{name: "checker-one"})
	require.NoError(t, err)
	checkerTwo, err := k.kernel.LeasePluginChecker(ctx, &runtimeLeasePluginHook{name: "checker-two"})
	require.NoError(t, err)
	require.NoError(t, checkerOne.Close(ctx))
	requireRuntimeLeasePluginChecker(t, k, "checker-two")
	require.NoError(t, checkerTwo.Close(ctx))
	require.Nil(t, k.kernel.PluginChecker())

	restarterOne, err := k.kernel.LeasePluginRestarter(ctx, &runtimeLeasePluginHook{name: "restarter-one"})
	require.NoError(t, err)
	restarterTwo, err := k.kernel.LeasePluginRestarter(ctx, &runtimeLeasePluginHook{name: "restarter-two"})
	require.NoError(t, err)
	require.NoError(t, restarterOne.Close(ctx))
	requireRuntimeLeasePluginRestarter(t, k, "restarter-two")
	require.NoError(t, restarterTwo.Close(ctx))
	require.Nil(t, k.kernel.PluginRestarter())

	evaluatorOne, err := k.kernel.LeaseToolEvaluator(ctx, &runtimeLeaseJSEvaluator{name: "evaluator-one"})
	require.NoError(t, err)
	evaluatorTwo, err := k.kernel.LeaseToolEvaluator(ctx, &runtimeLeaseJSEvaluator{name: "evaluator-two"})
	require.NoError(t, err)
	require.NoError(t, evaluatorOne.Close(ctx))
	requireRuntimeLeaseToolEvaluator(t, k, "evaluator-two")
	require.NoError(t, evaluatorTwo.Close(ctx))
	require.Nil(t, k.kernel.ToolsDomain().Evaluator())
}

func TestRuntimeTraceStoreLeaseRestoresConfiguredBase(t *testing.T) {
	base := &runtimeLeaseTraceStore{name: "base"}
	k, err := New(Config{Transport: Memory(), TraceStore: base})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	lease, err := k.kernel.LeaseTraceStore(ctx, &runtimeLeaseTraceStore{name: "mounted"})
	require.NoError(t, err)
	requireRuntimeLeaseTraceStore(t, k, "mounted")
	require.NoError(t, lease.Close(ctx))
	requireRuntimeLeaseTraceStore(t, k, "base")
}

func TestRuntimeToolEvaluatorLeaseRestoresConfiguredBase(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	base := &runtimeLeaseJSEvaluator{name: "base"}
	k.kernel.ToolsDomain().SetEvaluator(base)

	ctx := context.Background()
	lease, err := k.kernel.LeaseToolEvaluator(ctx, &runtimeLeaseJSEvaluator{name: "mounted"})
	require.NoError(t, err)
	requireRuntimeLeaseToolEvaluator(t, k, "mounted")
	require.NoError(t, lease.Close(ctx))
	requireRuntimeLeaseToolEvaluator(t, k, "base")
	k.kernel.ToolsDomain().SetEvaluator(nil)
}

func TestKitMountCommandAfterStartUnmountAndRemount(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	require.NoError(t, k.Mount(ctx, hotEchoModule{id: "hot-echo", suffix: "-one"}))
	require.True(t, k.hasCommand("test.hot.echo"))

	resp, err := Call[hotEchoMsg, hotEchoResp](k, ctx, hotEchoMsg{Text: "hello"}, WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	require.Equal(t, "hello-one", resp.Text)

	require.NoError(t, k.Unmount(ctx, "hot-echo"))
	require.False(t, k.hasCommand("test.hot.echo"))

	require.NoError(t, k.Mount(ctx, hotEchoModule{id: "hot-echo", suffix: "-two"}))
	resp, err = Call[hotEchoMsg, hotEchoResp](k, ctx, hotEchoMsg{Text: "hello"}, WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	require.Equal(t, "hello-two", resp.Text)
}

func TestMountedModulesIncludesRuntimeCommandManifest(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	require.NoError(t, k.Mount(context.Background(), hotEchoModule{id: "hot-echo-manifest", suffix: "-one"}))

	descs := k.MountedModules()
	require.Len(t, descs, 1)
	require.Equal(t, "hot-echo-manifest", descs[0].Name)
	require.Len(t, descs[0].Commands, 1)
	require.Equal(t, "test.hot.echo", descs[0].Commands[0].Topic)
	require.Equal(t, bkmodule.MessageKindCommand, descs[0].Commands[0].Kind)
	require.Contains(t, descs[0].Commands[0].Request, "hotEchoMsg")
	require.Contains(t, descs[0].Commands[0].Response, "hotEchoResp")

	require.NoError(t, k.Unmount(context.Background(), "hot-echo-manifest"))
	require.Empty(t, k.MountedModules())
}

type cleanupModule struct {
	id     string
	closed *bool
}

func (m cleanupModule) ID() string { return m.id }

func (m cleanupModule) Mount(_ context.Context, host bkmodule.Host) error {
	host.Scope().Defer(func(context.Context) error {
		*m.closed = true
		return nil
	})
	return nil
}

func TestKitCloseClosesMountedScopes(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)

	closed := false
	require.NoError(t, k.Mount(context.Background(), cleanupModule{id: "cleanup", closed: &closed}))

	require.NoError(t, k.Close())
	require.True(t, closed)
}

type toolModule struct {
	id string
}

func (m toolModule) ID() string { return m.id }

func (m toolModule) Mount(ctx context.Context, host bkmodule.Host) error {
	_, err := host.Tools().Register(ctx, bkmodule.ToolSpec{
		Name: "test.tool",
		Executor: bkmodule.ToolExecutorFunc(func(context.Context, string, json.RawMessage) (json.RawMessage, error) {
			return json.RawMessage(`{"ok":true}`), nil
		}),
	})
	return err
}

func TestKitUnmountUnregistersScopedTools(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	require.NoError(t, k.Mount(ctx, toolModule{id: "tool"}))
	_, err = k.kernel.Tools.Resolve("test.tool")
	require.NoError(t, err)

	require.NoError(t, k.Unmount(ctx, "tool"))
	_, err = k.kernel.Tools.Resolve("test.tool")
	require.ErrorAs(t, err, new(*sdkerrors.NotFoundError))
}

func TestMountedModulesIncludesRuntimeToolResources(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	require.NoError(t, k.Mount(context.Background(), toolModule{id: "tool-manifest"}))

	descs := k.MountedModules()
	require.Len(t, descs, 1)
	require.Equal(t, "tool-manifest", descs[0].Name)
	require.Len(t, descs[0].Resources, 1)
	require.Equal(t, bkmodule.ResourceKindTool, descs[0].Resources[0].Kind)
	require.Equal(t, "test.tool", descs[0].Resources[0].Name)
}

type eventModule struct {
	id   string
	seen chan string
}

func (m eventModule) ID() string { return m.id }

func (m eventModule) Mount(ctx context.Context, host bkmodule.Host) error {
	_, err := host.Messages().SubscribeRaw(ctx, "test.event", func(msg sdk.Message) {
		select {
		case m.seen <- string(msg.Payload):
		default:
		}
	})
	return err
}

type blockingSubscriptionModule struct {
	id      string
	entered chan struct{}
	release chan struct{}
}

func (m blockingSubscriptionModule) ID() string { return m.id }

func (m blockingSubscriptionModule) Mount(ctx context.Context, host bkmodule.Host) error {
	_, err := host.Messages().SubscribeRaw(ctx, "test.blocking.event", func(sdk.Message) {
		select {
		case m.entered <- struct{}{}:
		default:
		}
		<-m.release
	})
	return err
}

func TestKitUnmountCancelsScopedSubscriptions(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	seen := make(chan string, 1)
	require.NoError(t, k.Mount(ctx, eventModule{id: "events", seen: seen}))

	_, err = k.PublishRaw(ctx, "test.event", json.RawMessage(`"one"`))
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		select {
		case got := <-seen:
			return got == `"one"`
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)

	require.NoError(t, k.Unmount(ctx, "events"))
	_, err = k.PublishRaw(ctx, "test.event", json.RawMessage(`"two"`))
	require.NoError(t, err)

	select {
	case got := <-seen:
		t.Fatalf("received event after unmount: %s", got)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestKitUnmountRetainsBlockingScopedSubscriptionForRetry(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	baseline := k.kernel.TransportDebugSnapshot()
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})

	mod := blockingSubscriptionModule{id: "blocking-subscription", entered: entered, release: release}
	require.NoError(t, k.Mount(ctx, mod))
	require.Equal(t, baseline.ActiveSubscriptions+1, k.kernel.TransportDebugSnapshot().ActiveSubscriptions)

	_, err = k.PublishRaw(ctx, "test.blocking.event", json.RawMessage(`"one"`))
	require.NoError(t, err)
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("blocking subscription did not receive event")
	}

	stopCtx, cancelStop := context.WithTimeout(ctx, 20*time.Millisecond)
	err = k.Unmount(stopCtx, mod.id)
	cancelStop()
	require.ErrorIs(t, err, context.DeadlineExceeded)
	_, mounted := k.Module(mod.id)
	require.True(t, mounted)
	require.Equal(t, baseline.ActiveSubscriptions+1, k.kernel.TransportDebugSnapshot().ActiveSubscriptions)

	close(release)
	retryCtx, cancelRetry := context.WithTimeout(ctx, time.Second)
	defer cancelRetry()
	require.NoError(t, k.Unmount(retryCtx, mod.id))
	_, mounted = k.Module(mod.id)
	require.False(t, mounted)
	require.Eventually(t, func() bool {
		return k.kernel.TransportDebugSnapshot().ActiveSubscriptions == baseline.ActiveSubscriptions
	}, time.Second, 10*time.Millisecond)
}

func TestMountedModulesIncludesRuntimeSubscriptionManifest(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	seen := make(chan string, 1)
	require.NoError(t, k.Mount(context.Background(), eventModule{id: "events-manifest", seen: seen}))

	descs := k.MountedModules()
	require.Len(t, descs, 1)
	require.Equal(t, "events-manifest", descs[0].Name)
	require.Len(t, descs[0].Subscriptions, 1)
	require.Equal(t, "test.event", descs[0].Subscriptions[0].Topic)
	require.Equal(t, bkmodule.MessageKindSubscription, descs[0].Subscriptions[0].Kind)
}

type capabilityModule struct {
	id string
}

func (m capabilityModule) ID() string { return m.id }

func (m capabilityModule) Mount(ctx context.Context, host bkmodule.Host) error {
	_, err := host.Capabilities().Provide(ctx, "test.capability", "value")
	return err
}

func TestKitUnmountRemovesScopedCapabilities(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	require.NoError(t, k.Mount(ctx, capabilityModule{id: "capability"}))
	value, ok := k.caps.Get("test.capability")
	require.True(t, ok)
	require.Equal(t, "value", value)

	require.NoError(t, k.Unmount(ctx, "capability"))
	_, ok = k.caps.Get("test.capability")
	require.False(t, ok)
}

func TestMountedModulesIncludesRuntimeProvidedCapabilities(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	require.NoError(t, k.Mount(context.Background(), capabilityModule{id: "capability-manifest"}))

	descs := k.MountedModules()
	require.Len(t, descs, 1)
	require.Equal(t, "capability-manifest", descs[0].Name)
	require.Len(t, descs[0].Capabilities, 1)
	require.Equal(t, bkmodule.CapabilityProvided, descs[0].Capabilities[0].Direction)
	require.Equal(t, "test.capability", descs[0].Capabilities[0].Name)
	require.Contains(t, descs[0].Capabilities[0].Type, "string")
	require.NotNil(t, descs[0].CapabilityGroups)
	require.Len(t, descs[0].CapabilityGroups.Provided, 1)
	require.Equal(t, "test.capability", descs[0].CapabilityGroups.Provided[0].Name)
}

type missingPreflightCapabilityModule struct {
	id     string
	called *bool
}

func (m missingPreflightCapabilityModule) ID() string { return m.id }

func (m missingPreflightCapabilityModule) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name: m.id,
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.RequiredCapabilityOf[string]("test.preflight.missing"),
			bkmodule.OptionalCapabilityOf[string]("test.preflight.optional"),
		},
	}
}

func (m missingPreflightCapabilityModule) Mount(context.Context, bkmodule.Host) error {
	*m.called = true
	return nil
}

func TestKitMountPreflightsRequiredCapabilitiesBeforeMount(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	called := false
	err = k.Mount(context.Background(), missingPreflightCapabilityModule{
		id:     "preflight-missing",
		called: &called,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), `module "preflight-missing" missing required capabilities`)
	require.Contains(t, err.Error(), "test.preflight.missing")
	require.False(t, called)

	_, mounted := k.Module("preflight-missing")
	require.False(t, mounted)
	require.Empty(t, k.MountedModules())
}

type preflightCapabilityProviderModule struct {
	id   string
	name string
}

func (m preflightCapabilityProviderModule) ID() string { return m.id }

func (m preflightCapabilityProviderModule) Mount(ctx context.Context, host bkmodule.Host) error {
	_, err := host.Capabilities().Provide(ctx, m.name, "provided")
	return err
}

type preflightCapabilityProviderFactory struct {
	id   string
	name string
}

func (f preflightCapabilityProviderFactory) Build(bkmodule.BuildContext) (bkmodule.Module, error) {
	return preflightCapabilityProviderModule{id: f.id, name: f.name}, nil
}

func (f preflightCapabilityProviderFactory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name: f.id,
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.ProvidedCapabilityOf[string](f.name),
		},
	}
}

type preflightCapabilityConsumerModule struct {
	id   string
	name string
}

func (m preflightCapabilityConsumerModule) ID() string { return m.id }

func (m preflightCapabilityConsumerModule) Mount(_ context.Context, host bkmodule.Host) error {
	value, err := bkmodule.RequireCapability[string](host, m.name)
	if err != nil {
		return err
	}
	if value != "provided" {
		return fmt.Errorf("capability %q: got %q", m.name, value)
	}
	return nil
}

type preflightCapabilityConsumerFactory struct {
	id      string
	dep     string
	capName string
}

func (f preflightCapabilityConsumerFactory) Build(bkmodule.BuildContext) (bkmodule.Module, error) {
	return preflightCapabilityConsumerModule{id: f.id, name: f.capName}, nil
}

func (f preflightCapabilityConsumerFactory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:     f.id,
		Requires: []string{f.dep},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.RequiredCapabilityOf[string](f.capName),
		},
	}
}

func TestKitMountPreflightRunsAfterDescriptorDependencies(t *testing.T) {
	const depID = "test-preflight-capability-provider"
	const modID = "test-preflight-capability-consumer"
	const capName = "test.preflight.provided"

	bkmodule.Register(depID, preflightCapabilityProviderFactory{id: depID, name: capName})
	bkmodule.Register(modID, preflightCapabilityConsumerFactory{id: modID, dep: depID, capName: capName})

	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	require.NoError(t, k.Mount(context.Background(), preflightCapabilityConsumerModule{id: modID, name: capName}))
	_, depMounted := k.Module(depID)
	require.True(t, depMounted)
	_, modMounted := k.Module(modID)
	require.True(t, modMounted)

	descs := k.MountedModules()
	require.Len(t, descs, 2)
	var consumer bkmodule.Descriptor
	for _, desc := range descs {
		if desc.Name == modID {
			consumer = desc
		}
	}
	require.Equal(t, modID, consumer.Name)
	require.NotNil(t, consumer.CapabilityGroups)
	require.Len(t, consumer.CapabilityGroups.Required, 1)
	require.Equal(t, capName, consumer.CapabilityGroups.Required[0].Name)
}

func TestModulePreflightReportsCapabilityAvailabilitySources(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	mountedProviderID := "test-preflight-source-mounted-provider-" + suffix
	plannedProviderID := "test-preflight-source-planned-provider-" + suffix
	mountedCap := "test.preflight.source.mounted." + suffix
	plannedCap := "test.preflight.source.planned." + suffix
	missingCap := "test.preflight.source.missing." + suffix

	bkmodule.Register(plannedProviderID, preflightCapabilityProviderFactory{id: plannedProviderID, name: plannedCap})

	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	require.NoError(t, k.Mount(context.Background(), preflightCapabilityProviderModule{
		id:   mountedProviderID,
		name: mountedCap,
	}))

	preflight := k.preflightModuleDescriptor(bkmodule.Descriptor{
		Name:     "test-preflight-source-consumer-" + suffix,
		Requires: []string{plannedProviderID},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.RequiredCapabilityOf[string](bkmodule.CapabilityRuntimeID),
			bkmodule.RequiredCapabilityOf[string](mountedCap),
			bkmodule.RequiredCapabilityOf[string](plannedCap),
			bkmodule.OptionalCapabilityOf[string](missingCap),
		},
	})
	require.True(t, preflight.Ready)

	requireCapabilityAvailability(t, preflight.RequiredCapabilities, bkmodule.CapabilityRuntimeID, true, "core", "brainkit.core")
	requireCapabilityAvailability(t, preflight.RequiredCapabilities, mountedCap, true, "mounted", mountedProviderID)
	requireCapabilityAvailability(t, preflight.RequiredCapabilities, plannedCap, true, "planned", plannedProviderID)
	requireCapabilityAvailability(t, preflight.OptionalCapabilities, missingCap, false, "", "")
}

func requireCapabilityAvailability(t *testing.T, caps []bkmodule.CapabilityAvailability, name string, available bool, source, provider string) {
	t.Helper()
	for _, cap := range caps {
		if cap.Name != name {
			continue
		}
		require.Equal(t, available, cap.Available)
		require.Equal(t, source, cap.Source)
		require.Equal(t, provider, cap.Provider)
		return
	}
	t.Fatalf("capability %q not found in %#v", name, caps)
}

type preflightSideEffectModule struct {
	id      string
	mounted *bool
}

func (m preflightSideEffectModule) ID() string { return m.id }

func (m preflightSideEffectModule) Mount(context.Context, bkmodule.Host) error {
	*m.mounted = true
	return nil
}

type preflightSideEffectFactory struct {
	id      string
	mounted *bool
}

func (f preflightSideEffectFactory) Build(bkmodule.BuildContext) (bkmodule.Module, error) {
	return preflightSideEffectModule{id: f.id, mounted: f.mounted}, nil
}

func (f preflightSideEffectFactory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{Name: f.id}
}

func TestKitMountPreflightsDependencyGraphBeforeAutoMount(t *testing.T) {
	const depID = "test-preflight-available-dependency"
	const missingDepID = "test-preflight-missing-dependency"
	const modID = "test-preflight-dependent-missing-dependency"

	depMounted := false
	bkmodule.Register(depID, preflightSideEffectFactory{id: depID, mounted: &depMounted})
	bkmodule.Register(modID, descriptorDependencyFactory{id: modID, requires: []string{depID, missingDepID}})

	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	err = k.Mount(context.Background(), descriptorDependencyModule{id: modID})
	require.Error(t, err)
	require.Contains(t, err.Error(), `module "test-preflight-dependent-missing-dependency" preflight failed`)
	require.Contains(t, err.Error(), "missing modules: "+missingDepID)
	require.False(t, depMounted)

	_, depMountedLive := k.Module(depID)
	require.False(t, depMountedLive)
	_, modMounted := k.Module(modID)
	require.False(t, modMounted)
}

type resourceModule struct {
	id string
}

func (m resourceModule) ID() string { return m.id }

func (m resourceModule) Mount(_ context.Context, host bkmodule.Host) error {
	host.Scope().Resource(bkmodule.Resource(bkmodule.ResourceKindHook, "test.hook", "test hook"))
	child := host.Scope().Child("child")
	child.Resource(bkmodule.Resource(bkmodule.ResourceKindProcess, "test.child", "child resource"))
	return nil
}

func TestMountedModulesIncludesScopeResources(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	require.NoError(t, k.Mount(context.Background(), resourceModule{id: "resources"}))

	descs := k.MountedModules()
	require.Len(t, descs, 1)
	require.Equal(t, "resources", descs[0].Name)
	require.Equal(t, []bkmodule.ResourceDescriptor{
		bkmodule.Resource(bkmodule.ResourceKindHook, "test.hook", "test hook"),
		bkmodule.Resource(bkmodule.ResourceKindProcess, "test.child", "child resource"),
	}, descs[0].Resources)
}

type descriptorDependencyModule struct {
	id string
}

func (m descriptorDependencyModule) ID() string { return m.id }

func (m descriptorDependencyModule) Mount(context.Context, bkmodule.Host) error { return nil }

type descriptorDependencyFactory struct {
	id       string
	requires []string
}

func (f descriptorDependencyFactory) Build(bkmodule.BuildContext) (bkmodule.Module, error) {
	return descriptorDependencyModule{id: f.id}, nil
}

func (f descriptorDependencyFactory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{Name: f.id, Requires: f.requires}
}

func TestDescriptorRequiresDriveHotMountDependencies(t *testing.T) {
	const depID = "test-descriptor-dependency"
	const modID = "test-descriptor-dependent"

	bkmodule.Register(depID, descriptorDependencyFactory{id: depID})
	bkmodule.Register(modID, descriptorDependencyFactory{id: modID, requires: []string{depID}})

	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	require.NoError(t, k.Mount(context.Background(), descriptorDependencyModule{id: modID}))
	_, depMounted := k.Module(depID)
	require.True(t, depMounted)
	_, modMounted := k.Module(modID)
	require.True(t, modMounted)

	descs := k.MountedModules()
	require.Len(t, descs, 2)
	require.Equal(t, depID, descs[0].Name)
	require.Equal(t, modID, descs[1].Name)
	require.Equal(t, []string{depID}, descs[1].Requires)
}

func TestKitUnmountRefusesMountedDependents(t *testing.T) {
	const depID = "test-direct-lifecycle-dependency"
	const modID = "test-direct-lifecycle-dependent"

	bkmodule.Register(depID, descriptorDependencyFactory{id: depID})
	bkmodule.Register(modID, descriptorDependencyFactory{id: modID, requires: []string{depID}})

	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	require.NoError(t, k.Mount(context.Background(), descriptorDependencyModule{id: modID}))
	err = k.Unmount(context.Background(), depID)
	require.Error(t, err)
	require.Contains(t, err.Error(), modID)

	_, depMounted := k.Module(depID)
	require.True(t, depMounted)
	_, modMounted := k.Module(modID)
	require.True(t, modMounted)
}

type orderedCleanupModule struct {
	id    string
	order *[]string
}

func (m orderedCleanupModule) ID() string { return m.id }

func (m orderedCleanupModule) Mount(_ context.Context, host bkmodule.Host) error {
	host.Scope().Defer(func(context.Context) error {
		*m.order = append(*m.order, m.id)
		return nil
	})
	return nil
}

func TestKitCloseClosesMountedScopesInReverseMountOrder(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)

	var order []string
	require.NoError(t, k.Mount(context.Background(), orderedCleanupModule{id: "first", order: &order}))
	require.NoError(t, k.Mount(context.Background(), orderedCleanupModule{id: "second", order: &order}))

	require.NoError(t, k.Close())
	require.Equal(t, []string{"second", "first"}, order)
}

func TestControlModuleExposesMountedModuleManifest(t *testing.T) {
	k, err := New(Config{Transport: Memory(), Modules: []bkmodule.Module{controlmod.New()}})
	require.NoError(t, err)
	defer k.Close()

	resp, err := controlmod.CallKitModules(k, context.Background(), controlmod.KitModulesMsg{}, sdk.WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	require.Len(t, resp.Modules, 1)
	require.Equal(t, "control", resp.Modules[0].Name)
	require.Contains(t, commandTopics(resp.Modules[0].Commands), "kit.modules")
}

type lifecycleEchoFactory struct {
	id string
}

func (f lifecycleEchoFactory) Build(ctx bkmodule.BuildContext) (bkmodule.Module, error) {
	var cfg struct {
		Suffix string `yaml:"suffix"`
	}
	if err := ctx.Decode(&cfg); err != nil {
		return nil, err
	}
	return hotEchoModule{id: f.id, suffix: cfg.Suffix}, nil
}

func (f lifecycleEchoFactory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    f.id,
		Status:  bkmodule.StatusStable,
		Summary: "Test lifecycle echo module.",
		Commands: []bkmodule.MessageDescriptor{
			bkmodule.CommandMessage[hotEchoMsg, hotEchoResp](),
		},
	}
}

func TestControlModuleMountDescribeUnmountRegisteredModule(t *testing.T) {
	const id = "test-control-lifecycle-echo"
	bkmodule.Register(id, lifecycleEchoFactory{id: id})

	k, err := New(Config{Transport: Memory(), Modules: []bkmodule.Module{controlmod.New()}})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	descResp, err := controlmod.CallKitModuleDescribe(k, ctx, controlmod.KitModuleDescribeMsg{ID: id}, sdk.WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	require.False(t, descResp.Mounted)
	require.Equal(t, id, descResp.Module.Name)

	mountResp, err := controlmod.CallKitModuleMount(k, ctx, controlmod.KitModuleMountMsg{
		ID:         id,
		ConfigYAML: "suffix: -mounted\n",
	}, sdk.WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	require.Equal(t, id, mountResp.Module.Name)
	require.True(t, k.hasCommand("test.hot.echo"))

	echoResp, err := Call[hotEchoMsg, hotEchoResp](k, ctx, hotEchoMsg{Text: "hello"}, WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	require.Equal(t, "hello-mounted", echoResp.Text)

	descResp, err = controlmod.CallKitModuleDescribe(k, ctx, controlmod.KitModuleDescribeMsg{ID: id}, sdk.WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	require.True(t, descResp.Mounted)

	unmountResp, err := controlmod.CallKitModuleUnmount(k, ctx, controlmod.KitModuleUnmountMsg{ID: id}, sdk.WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	require.Equal(t, id, unmountResp.Module.Name)
	require.False(t, k.hasCommand("test.hot.echo"))
}

type failingLifecycleFactory struct {
	id string
}

func (f failingLifecycleFactory) Build(bkmodule.BuildContext) (bkmodule.Module, error) {
	return failingLifecycleModule{id: f.id}, nil
}

func (f failingLifecycleFactory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{Name: f.id}
}

type failingLifecycleModule struct {
	id string
}

func (m failingLifecycleModule) ID() string { return m.id }

func (m failingLifecycleModule) Mount(_ context.Context, host bkmodule.Host) error {
	if _, err := host.Commands().Handle(bkmodule.Command(func(_ context.Context, req hotEchoMsg) (*hotEchoResp, error) {
		return &hotEchoResp{Text: req.Text}, nil
	})); err != nil {
		return err
	}
	return fmt.Errorf("boom")
}

func TestControlModuleFailedMountRollsBackPartialScope(t *testing.T) {
	const id = "test-control-lifecycle-failing"
	bkmodule.Register(id, failingLifecycleFactory{id: id})

	k, err := New(Config{Transport: Memory(), Modules: []bkmodule.Module{controlmod.New()}})
	require.NoError(t, err)
	defer k.Close()

	_, err = controlmod.CallKitModuleMount(k, context.Background(), controlmod.KitModuleMountMsg{ID: id}, sdk.WithCallTimeout(2*time.Second))
	require.Error(t, err)
	require.False(t, k.hasCommand("test.hot.echo"))
	_, mounted := k.Module(id)
	require.False(t, mounted)
}

type missingCapabilityLifecycleFactory struct {
	id      string
	capName string
}

func (f missingCapabilityLifecycleFactory) Build(bkmodule.BuildContext) (bkmodule.Module, error) {
	return preflightCapabilityConsumerModule{id: f.id, name: f.capName}, nil
}

func (f missingCapabilityLifecycleFactory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name: f.id,
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.RequiredCapabilityOf[string](f.capName),
		},
	}
}

func TestControlModuleDescribeIncludesCapabilityPreflight(t *testing.T) {
	const id = "test-control-lifecycle-preflight-missing"
	const capName = "test.control.preflight.missing"
	bkmodule.Register(id, missingCapabilityLifecycleFactory{id: id, capName: capName})

	k, err := New(Config{Transport: Memory(), Modules: []bkmodule.Module{controlmod.New()}})
	require.NoError(t, err)
	defer k.Close()

	resp, err := controlmod.CallKitModuleDescribe(k, context.Background(), controlmod.KitModuleDescribeMsg{ID: id}, sdk.WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	require.False(t, resp.Mounted)
	require.False(t, resp.Preflight.Ready)
	require.Len(t, resp.Preflight.RequiredCapabilities, 1)
	require.Equal(t, capName, resp.Preflight.RequiredCapabilities[0].Name)
	require.False(t, resp.Preflight.RequiredCapabilities[0].Available)
	require.Len(t, resp.Preflight.MissingRequiredCapabilities, 1)
	require.Equal(t, capName, resp.Preflight.MissingRequiredCapabilities[0].Name)

	_, err = controlmod.CallKitModuleMount(k, context.Background(), controlmod.KitModuleMountMsg{ID: id}, sdk.WithCallTimeout(2*time.Second))
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required capabilities")
	_, mounted := k.Module(id)
	require.False(t, mounted)
}

func TestControlModuleMountHonorsDescriptorDependencies(t *testing.T) {
	const depID = "test-control-lifecycle-dependency"
	const modID = "test-control-lifecycle-dependent"

	bkmodule.Register(depID, descriptorDependencyFactory{id: depID})
	bkmodule.Register(modID, descriptorDependencyFactory{id: modID, requires: []string{depID}})

	k, err := New(Config{Transport: Memory(), Modules: []bkmodule.Module{controlmod.New()}})
	require.NoError(t, err)
	defer k.Close()

	_, err = controlmod.CallKitModuleMount(k, context.Background(), controlmod.KitModuleMountMsg{ID: modID}, sdk.WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	_, depMounted := k.Module(depID)
	require.True(t, depMounted)
	_, modMounted := k.Module(modID)
	require.True(t, modMounted)

	_, err = controlmod.CallKitModuleUnmount(k, context.Background(), controlmod.KitModuleUnmountMsg{ID: depID}, sdk.WithCallTimeout(2*time.Second))
	require.Error(t, err)
	_, depMounted = k.Module(depID)
	require.True(t, depMounted)
}

type unmountRetryModule struct {
	id    string
	fail  bool
	calls int
	err   error
}

func (m *unmountRetryModule) ID() string { return m.id }

func (m *unmountRetryModule) Mount(_ context.Context, host bkmodule.Host) error {
	host.Scope().Defer(func(context.Context) error {
		m.calls++
		if m.fail {
			return m.err
		}
		return nil
	})
	return nil
}

func TestUnmountRetainsMountedModuleWhenScopeCloseFails(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	want := errors.New("cleanup failed")
	mod := &unmountRetryModule{id: "unmount-retry", fail: true, err: want}
	require.NoError(t, k.Mount(context.Background(), mod))

	err = k.Unmount(context.Background(), mod.id)
	require.ErrorIs(t, err, want)
	require.Equal(t, 1, mod.calls)
	_, mounted := k.Module(mod.id)
	require.True(t, mounted)

	mod.fail = false
	require.NoError(t, k.Unmount(context.Background(), mod.id))
	require.Equal(t, 2, mod.calls)
	_, mounted = k.Module(mod.id)
	require.False(t, mounted)
}

func TestCloseMountedRetainsFailedScopesForRetry(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	want := errors.New("cleanup failed")
	mod := &unmountRetryModule{id: "close-mounted-retry", fail: true, err: want}
	require.NoError(t, k.Mount(context.Background(), mod))

	err = k.closeMounted(context.Background())
	require.ErrorIs(t, err, want)
	require.Equal(t, 1, mod.calls)
	_, mounted := k.Module(mod.id)
	require.True(t, mounted)

	mod.fail = false
	require.NoError(t, k.closeMounted(context.Background()))
	require.Equal(t, 2, mod.calls)
	_, mounted = k.Module(mod.id)
	require.False(t, mounted)
}

func commandTopics(commands []bkmodule.MessageDescriptor) []string {
	out := make([]string, 0, len(commands))
	for _, command := range commands {
		out = append(out, command.Topic)
	}
	return out
}
