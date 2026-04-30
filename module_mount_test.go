package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

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

func TestControlModuleExposesMountedModuleManifest(t *testing.T) {
	k, err := New(Config{Transport: Memory(), Modules: []Module{controlmod.New()}})
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

	k, err := New(Config{Transport: Memory(), Modules: []Module{controlmod.New()}})
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

	k, err := New(Config{Transport: Memory(), Modules: []Module{controlmod.New()}})
	require.NoError(t, err)
	defer k.Close()

	_, err = controlmod.CallKitModuleMount(k, context.Background(), controlmod.KitModuleMountMsg{ID: id}, sdk.WithCallTimeout(2*time.Second))
	require.Error(t, err)
	require.False(t, k.hasCommand("test.hot.echo"))
	_, mounted := k.Module(id)
	require.False(t, mounted)
}

func TestControlModuleMountHonorsDescriptorDependencies(t *testing.T) {
	const depID = "test-control-lifecycle-dependency"
	const modID = "test-control-lifecycle-dependent"

	bkmodule.Register(depID, descriptorDependencyFactory{id: depID})
	bkmodule.Register(modID, descriptorDependencyFactory{id: modID, requires: []string{depID}})

	k, err := New(Config{Transport: Memory(), Modules: []Module{controlmod.New()}})
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

func commandTopics(commands []bkmodule.MessageDescriptor) []string {
	out := make([]string, 0, len(commands))
	for _, command := range commands {
		out = append(out, command.Topic)
	}
	return out
}
