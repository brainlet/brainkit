package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	auditpkg "github.com/brainlet/brainkit/internal/audit"
	toolreg "github.com/brainlet/brainkit/internal/tools"
	coretracing "github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
	plugincap "github.com/brainlet/brainkit/modulecap/plugin"
	"github.com/brainlet/brainkit/modules/plugins/pluginmsg"
	"github.com/brainlet/brainkit/sdk/pluginws"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestPluginManifestToolsUnmountWithModule(t *testing.T) {
	host := newTestPluginHost()
	module := &Module{kit: host}
	toolName := toolreg.ComposeName("acme", "demo", "1.0.0", "echo")

	resp, err := module.processPluginManifest(context.Background(), pluginmsg.PluginManifestMsg{
		Owner:   "acme",
		Name:    "demo",
		Version: "1.0.0",
		Tools: []pluginmsg.PluginToolDef{{
			Name:        "echo",
			Description: "echo",
			InputSchema: `{}`,
		}},
	})
	if err != nil {
		t.Fatalf("process plugin manifest: %v", err)
	}
	if resp == nil || !resp.Registered {
		t.Fatalf("manifest response = %#v, want registered", resp)
	}
	if _, err := host.tools.Resolve(toolName); err != nil {
		t.Fatalf("registered tool missing: %v", err)
	}

	if err := module.Close(); err != nil {
		t.Fatalf("close module: %v", err)
	}
	if _, err := host.tools.Resolve(toolName); err == nil {
		t.Fatalf("tool %q still registered after module close", toolName)
	}
}

func TestPluginWSServerCloseUnregistersConnectedTools(t *testing.T) {
	host := newTestPluginHost()
	module := &Module{kit: host}
	server, err := newPluginWSServer(module)
	if err != nil {
		t.Fatalf("new plugin ws server: %v", err)
	}
	t.Cleanup(server.Close)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, server.URL(), nil)
	if err != nil {
		t.Fatalf("dial plugin ws server: %v", err)
	}
	defer conn.CloseNow()

	manifestData, _ := json.Marshal(pluginws.Manifest{
		Owner:   "acme",
		Name:    "demo-ws",
		Version: "1.0.0",
		Tools: []pluginws.ToolDef{{
			Name:        "echo",
			Description: "echo",
			InputSchema: `{}`,
		}},
	})
	if err := wsjson.Write(ctx, conn, pluginws.Message{Type: pluginws.TypeManifest, Data: manifestData}); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	var ackMsg pluginws.Message
	if err := wsjson.Read(ctx, conn, &ackMsg); err != nil {
		t.Fatalf("read manifest ack: %v", err)
	}
	if ackMsg.Type != pluginws.TypeManifestAck {
		t.Fatalf("ack type = %q, want %q", ackMsg.Type, pluginws.TypeManifestAck)
	}
	var ack pluginws.ManifestAck
	if err := json.Unmarshal(ackMsg.Data, &ack); err != nil {
		t.Fatalf("decode manifest ack: %v", err)
	}
	if !ack.Registered || ack.Error != "" {
		t.Fatalf("manifest ack = %#v, want registered without error", ack)
	}

	toolName := toolreg.ComposeName("acme", "demo-ws", "1.0.0", "echo")
	if _, err := host.tools.Resolve(toolName); err != nil {
		t.Fatalf("registered tool missing: %v", err)
	}

	closeCtx, closeCancel := context.WithTimeout(context.Background(), time.Second)
	defer closeCancel()
	if err := server.CloseContext(closeCtx); err != nil {
		t.Fatalf("close plugin ws server: %v", err)
	}
	if _, err := host.tools.Resolve(toolName); err == nil {
		t.Fatalf("tool %q still registered after ws server close", toolName)
	}
	if got := server.debugSnapshot(); got.Listening || got.ServeRunning || got.ActiveHandlers != 0 || got.PingLoops != 0 || got.ActiveConnections != 0 {
		t.Fatalf("ws debug after close = %#v, want no listener, serve loop, handlers, pings, or connections", got)
	}
}

func TestPluginWSServerCloseContextWaitsForAcceptedConnectionBeforeManifest(t *testing.T) {
	host := newTestPluginHost()
	module := &Module{kit: host}
	server, err := newPluginWSServer(module)
	if err != nil {
		t.Fatalf("new plugin ws server: %v", err)
	}
	t.Cleanup(server.Close)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, server.URL(), nil)
	if err != nil {
		t.Fatalf("dial plugin ws server: %v", err)
	}
	defer conn.CloseNow()

	waitForPluginTestCondition(t, testTimeout, func() bool {
		got := server.debugSnapshot()
		return got.ActiveHandlers == 1 && got.ActiveConnections == 0
	}, "accepted websocket handler before manifest")

	closeCtx, closeCancel := context.WithTimeout(context.Background(), time.Second)
	defer closeCancel()
	if err := server.CloseContext(closeCtx); err != nil {
		t.Fatalf("close plugin ws server: %v", err)
	}
	if got := server.debugSnapshot(); got.Listening || got.ServeRunning || got.ActiveHandlers != 0 || got.PingLoops != 0 || got.ActiveConnections != 0 {
		t.Fatalf("ws debug after pre-manifest close = %#v, want no listener, serve loop, handlers, pings, or connections", got)
	}
}

func TestPluginWSServerCloseContextWaitsForRegisteredConnectionHandlers(t *testing.T) {
	host := newTestPluginHost()
	module := &Module{kit: host}
	server, err := newPluginWSServer(module)
	if err != nil {
		t.Fatalf("new plugin ws server: %v", err)
	}
	t.Cleanup(server.Close)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	manifest := pluginws.Manifest{
		Owner:   "acme",
		Name:    "demo-wait",
		Version: "1.0.0",
		Tools: []pluginws.ToolDef{{
			Name:        "echo",
			Description: "echo",
			InputSchema: `{}`,
		}},
	}
	conn := registerPluginWS(t, ctx, server.URL(), manifest)
	defer conn.CloseNow()

	toolName := toolreg.ComposeName("acme", "demo-wait", "1.0.0", "echo")
	if _, err := host.tools.Resolve(toolName); err != nil {
		t.Fatalf("registered tool missing: %v", err)
	}
	waitForPluginTestCondition(t, testTimeout, func() bool {
		got := server.debugSnapshot()
		return got.ActiveHandlers == 1 && got.PingLoops == 1 && got.ActiveConnections == 1
	}, "registered websocket handler")

	closeCtx, closeCancel := context.WithTimeout(context.Background(), time.Second)
	defer closeCancel()
	if err := server.CloseContext(closeCtx); err != nil {
		t.Fatalf("close plugin ws server: %v", err)
	}
	if _, err := host.tools.Resolve(toolName); err == nil {
		t.Fatalf("tool %q still registered after ws server close", toolName)
	}
	if got := server.debugSnapshot(); got.Listening || got.ServeRunning || got.ActiveHandlers != 0 || got.PingLoops != 0 || got.ActiveConnections != 0 {
		t.Fatalf("ws debug after registered close = %#v, want no listener, serve loop, handlers, pings, or connections", got)
	}
}

func TestPluginWSServerReplacesDuplicateConnection(t *testing.T) {
	host := newTestPluginHost()
	module := &Module{kit: host}
	server, err := newPluginWSServer(module)
	if err != nil {
		t.Fatalf("new plugin ws server: %v", err)
	}
	t.Cleanup(server.Close)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	manifest := pluginws.Manifest{
		Owner:   "acme",
		Name:    "demo-replace",
		Version: "1.0.0",
		Tools: []pluginws.ToolDef{{
			Name:        "echo",
			Description: "echo",
			InputSchema: `{}`,
		}},
	}
	first := registerPluginWS(t, ctx, server.URL(), manifest)
	defer first.CloseNow()
	second := registerPluginWS(t, ctx, server.URL(), manifest)
	defer second.CloseNow()

	toolName := toolreg.ComposeName("acme", "demo-replace", "1.0.0", "echo")
	if _, err := host.tools.Resolve(toolName); err != nil {
		t.Fatalf("registered tool missing after connection replacement: %v", err)
	}

	server.Close()
	if _, err := host.tools.Resolve(toolName); err == nil {
		t.Fatalf("tool %q still registered after ws server close", toolName)
	}
}

func TestPluginReplayRegistrationsTimerStopsOnClose(t *testing.T) {
	host := newTestPluginHost()
	module := &Module{
		kit: host,
		registrations: map[string]pluginmsg.PluginRegisteredEvent{
			"demo": {Owner: "acme", Name: "demo", Version: "1.0.0", Tools: 1},
		},
	}
	module.replayRegistrationsAfter(10 * time.Millisecond)
	if err := module.Close(); err != nil {
		t.Fatalf("close module: %v", err)
	}
	time.Sleep(25 * time.Millisecond)
	if got := host.published.Load(); got != 0 {
		t.Fatalf("published replay events after close = %d, want 0", got)
	}
}

func TestPluginWSConnClosesLateSubscriptionsAfterCleanup(t *testing.T) {
	pc := &pluginWSConn{
		name:    "demo",
		pending: map[string]chan pluginws.ToolResult{},
	}
	var stopped atomic.Int32

	pc.trackSubscription(pluginTestSubscriptionHandle{stop: func() { stopped.Add(1) }})
	subs := pc.cleanup(nil, "test cleanup")
	if got := stopped.Load(); got != 1 {
		t.Fatalf("stopped tracked subscriptions = %d, want 1", got)
	}
	if got := len(subs); got != 1 {
		t.Fatalf("cleanup returned subscriptions = %d, want 1", got)
	}

	pc.trackSubscription(pluginTestSubscriptionHandle{stop: func() { stopped.Add(1) }})
	if got := stopped.Load(); got != 2 {
		t.Fatalf("stopped late subscription = %d, want 2", got)
	}

	if got := len(pc.cleanup(nil, "second cleanup")); got != 0 {
		t.Fatalf("second cleanup returned subscriptions = %d, want 0", got)
	}
	if got := stopped.Load(); got != 2 {
		t.Fatalf("subscription stopped more than once = %d, want 2", got)
	}
}

func TestPluginWSServerCloseContextRetainsEventSubscriptionsWhenCloseTimesOut(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var closeStarted sync.Once
	var stopped atomic.Int32
	sub := pluginTestSubscriptionHandle{
		stop: func() { stopped.Add(1) },
		close: func(ctx context.Context) error {
			closeStarted.Do(func() { close(started) })
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})

	server := &pluginWSServer{
		conns: map[string]*pluginWSConn{
			"demo": {
				name:    "demo",
				pending: map[string]chan pluginws.ToolResult{},
				subs:    []pluginSubscriptionHandle{sub},
			},
		},
		sockets: map[*websocket.Conn]struct{}{},
	}

	closeCtx, cancelClose := context.WithTimeout(context.Background(), 20*time.Millisecond)
	err := server.CloseContext(closeCtx)
	cancelClose()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CloseContext error = %v, want context deadline exceeded", err)
	}
	select {
	case <-started:
	default:
		t.Fatal("subscription close did not start")
	}
	if got := stopped.Load(); got != 1 {
		t.Fatalf("subscription stop calls = %d, want 1", got)
	}
	if got := server.debugSnapshot().Subscriptions; got != 1 {
		t.Fatalf("debug subscriptions after timeout = %d, want retained subscription", got)
	}

	close(release)
	retryCtx, cancelRetry := context.WithTimeout(context.Background(), time.Second)
	defer cancelRetry()
	if err := server.CloseContext(retryCtx); err != nil {
		t.Fatalf("retry CloseContext: %v", err)
	}
	if got := server.debugSnapshot().Subscriptions; got != 0 {
		t.Fatalf("debug subscriptions after retry = %d, want 0", got)
	}
}

func TestPluginWSServerCloseContextRetriesWithSinglePendingHandlerAndPingWait(t *testing.T) {
	serveDone := make(chan struct{})
	close(serveDone)
	server := &pluginWSServer{
		conns:     map[string]*pluginWSConn{},
		sockets:   map[*websocket.Conn]struct{}{},
		serveDone: serveDone,
	}
	server.connWG.Add(1)
	server.activeHandlers.Store(1)
	server.pingWG.Add(1)
	server.activePingLoops.Store(1)

	for i := 0; i < 2; i++ {
		closeCtx, cancelClose := context.WithTimeout(context.Background(), 10*time.Millisecond)
		err := server.CloseContext(closeCtx)
		cancelClose()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("CloseContext attempt %d error = %v, want context deadline exceeded", i+1, err)
		}
		got := server.debugSnapshot()
		if got.ActiveHandlers != 1 || got.PingLoops != 1 {
			t.Fatalf("debug after close attempt %d = %#v, want one active handler and ping loop", i+1, got)
		}
	}

	server.activeHandlers.Add(-1)
	server.connWG.Done()
	server.activePingLoops.Add(-1)
	server.pingWG.Done()
	retryCtx, cancelRetry := context.WithTimeout(context.Background(), time.Second)
	defer cancelRetry()
	if err := server.CloseContext(retryCtx); err != nil {
		t.Fatalf("retry CloseContext: %v", err)
	}
	got := server.debugSnapshot()
	if got.ActiveHandlers != 0 || got.PingLoops != 0 {
		t.Fatalf("debug after retry = %#v, want no active handlers or ping loops", got)
	}
}

func TestPluginDebugSnapshotReportsOwnedCounters(t *testing.T) {
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()

	module := &Module{
		cfg: Config{
			Plugins: []PluginConfig{{Name: "static"}},
			Store:   testPluginStore{},
		},
		registrations: map[string]pluginmsg.PluginRegisteredEvent{
			"demo": {Owner: "acme", Name: "demo", Version: "1.0.0", Tools: 2},
		},
		toolRefs: map[string]int{
			"acme/demo@1.0.0/echo": 2,
			"acme/side@1.0.0/ping": 1,
		},
		pluginToolRefs: map[string]map[string]int{
			"demo": {
				"acme/demo@1.0.0/echo": 2,
			},
			"side": {
				"acme/side@1.0.0/ping": 1,
			},
		},
		replayTimers: []*time.Timer{timer},
	}
	module.pluginCheckerLeaseActive.Store(true)
	module.pluginRestarterLeaseActive.Store(true)

	manager := newPluginManager(module)
	manager.stopping = true
	manager.plugins["demo"] = &pluginConn{config: PluginConfig{Name: "demo"}, restarts: 2, stopping: true}
	manager.plugins["side"] = &pluginConn{config: PluginConfig{Name: "side"}, restarts: 1}
	serveDone := make(chan struct{})
	defer close(serveDone)
	ws := &pluginWSServer{
		closing:   true,
		serveDone: serveDone,
		conns: map[string]*pluginWSConn{
			"demo": {
				name: "demo",
				pending: map[string]chan pluginws.ToolResult{
					"call-1": make(chan pluginws.ToolResult, 1),
				},
				subs:  []pluginSubscriptionHandle{pluginTestSubscriptionHandle{}},
				tools: []string{"acme/demo@1.0.0/echo", "acme/demo@1.0.0/ping"},
			},
		},
	}
	ws.activeHandlers.Store(2)
	ws.activePingLoops.Store(1)
	manager.wsServer = ws
	module.manager = manager

	got := module.DebugSnapshot()
	if !got.ManagerStopping {
		t.Fatalf("ManagerStopping = false, want true")
	}
	if got.ConfiguredPlugins != 1 {
		t.Fatalf("ConfiguredPlugins = %d, want 1", got.ConfiguredPlugins)
	}
	if got.RunningProcesses != 2 {
		t.Fatalf("RunningProcesses = %d, want 2", got.RunningProcesses)
	}
	if got.StoppingProcesses != 1 {
		t.Fatalf("StoppingProcesses = %d, want 1", got.StoppingProcesses)
	}
	if got.RestartCount != 3 {
		t.Fatalf("RestartCount = %d, want 3", got.RestartCount)
	}
	if got.RegisteredPlugins != 1 || got.RegisteredTools != 2 {
		t.Fatalf("registration counters = plugins %d tools %d, want 1 and 2", got.RegisteredPlugins, got.RegisteredTools)
	}
	if got.PluginToolOwners != 2 || got.PluginToolReferences != 3 {
		t.Fatalf("tool ownership counters = owners %d refs %d, want 2 and 3", got.PluginToolOwners, got.PluginToolReferences)
	}
	if got.ReplayTimers != 1 || !got.StoreConfigured {
		t.Fatalf("resource counters = replay timers %d store %v, want 1 and true", got.ReplayTimers, got.StoreConfigured)
	}
	if !got.PluginCheckerLeaseAttached || !got.PluginRestarterLeaseAttached {
		t.Fatalf("lease flags = checker %v restarter %v, want both true", got.PluginCheckerLeaseAttached, got.PluginRestarterLeaseAttached)
	}
	if got.WebSocket.ActiveConnections != 1 || got.WebSocket.PendingToolCalls != 1 || got.WebSocket.Subscriptions != 1 || got.WebSocket.RegisteredTools != 2 {
		t.Fatalf("websocket counters = %#v, want active=1 pending=1 subscriptions=1 tools=2", got.WebSocket)
	}
	if !got.WebSocket.Closing || !got.WebSocket.ServeRunning || got.WebSocket.ActiveHandlers != 2 || got.WebSocket.PingLoops != 1 {
		t.Fatalf("websocket lifecycle counters = %#v, want closing=true serveRunning=true activeHandlers=2 pingLoops=1", got.WebSocket)
	}
}

func TestPluginCloseContextReturnsLeaseErrors(t *testing.T) {
	checkerErr := errors.New("detach plugin checker")
	restarterErr := errors.New("detach plugin restarter")
	ctx := context.WithValue(context.Background(), pluginCloseContextKey{}, "close")
	checkerHandle := &pluginCloseHandle{err: checkerErr}
	restarterHandle := &pluginCloseHandle{err: restarterErr}
	module := &Module{
		pluginCheckerLease:   checkerHandle,
		pluginRestarterLease: restarterHandle,
	}
	module.pluginCheckerLeaseActive.Store(true)
	module.pluginRestarterLeaseActive.Store(true)

	err := module.CloseContext(ctx)
	if !errors.Is(err, checkerErr) {
		t.Fatalf("CloseContext error = %v, want checker error %v", err, checkerErr)
	}
	if !errors.Is(err, restarterErr) {
		t.Fatalf("CloseContext error = %v, want restarter error %v", err, restarterErr)
	}
	if checkerHandle.ctx != ctx {
		t.Fatalf("checker lease closed with context %v, want %v", checkerHandle.ctx, ctx)
	}
	if restarterHandle.ctx != ctx {
		t.Fatalf("restarter lease closed with context %v, want %v", restarterHandle.ctx, ctx)
	}
	if module.pluginCheckerLease == nil || module.pluginRestarterLease == nil {
		t.Fatal("plugin leases were cleared after close failure")
	}
	if !module.pluginCheckerLeaseActive.Load() || !module.pluginRestarterLeaseActive.Load() {
		t.Fatal("plugin leases no longer marked active after close failure")
	}

	checkerHandle.setErr(nil)
	restarterHandle.setErr(nil)
	if err := module.CloseContext(ctx); err != nil {
		t.Fatalf("retry CloseContext: %v", err)
	}
	if checkerHandle.closeCount() != 2 || restarterHandle.closeCount() != 2 {
		t.Fatalf("lease close counts = checker %d restarter %d, want 2 and 2", checkerHandle.closeCount(), restarterHandle.closeCount())
	}
	if module.pluginCheckerLease != nil || module.pluginRestarterLease != nil {
		t.Fatal("plugin leases were not cleared after successful retry")
	}
	if module.pluginCheckerLeaseActive.Load() || module.pluginRestarterLeaseActive.Load() {
		t.Fatal("plugin leases still marked active after successful retry")
	}
}

func TestPluginCloseContextClosesAndClearsLazyWSServer(t *testing.T) {
	host := newTestPluginHost()
	module := &Module{kit: host}
	manager := newPluginManager(module)
	ws, err := newPluginWSServer(module)
	if err != nil {
		t.Fatalf("new plugin ws server: %v", err)
	}
	if !ws.debugSnapshot().Listening {
		t.Fatal("plugin ws server should be listening before close")
	}
	manager.wsServer = ws
	module.manager = manager

	if err := module.CloseContext(context.Background()); err != nil {
		t.Fatalf("CloseContext: %v", err)
	}
	if module.manager.wsServer != nil {
		t.Fatal("plugin manager wsServer was not cleared")
	}
	if got := ws.debugSnapshot(); got.Listening || got.ServeRunning || got.ActiveHandlers != 0 || got.PingLoops != 0 || got.ActiveConnections != 0 || got.Subscriptions != 0 {
		t.Fatalf("ws debug after close = %#v, want fully detached", got)
	}
}

func TestPluginCloseContextKeepsLazyWSServerWhenCloseTimesOut(t *testing.T) {
	host := newTestPluginHost()
	module := &Module{kit: host}
	manager := newPluginManager(module)
	serveDone := make(chan struct{})
	ws := &pluginWSServer{
		mod:       module,
		conns:     map[string]*pluginWSConn{},
		sockets:   map[*websocket.Conn]struct{}{},
		serveDone: serveDone,
	}
	manager.wsServer = ws
	module.manager = manager

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := module.CloseContext(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CloseContext error = %v, want context deadline exceeded", err)
	}
	if manager.wsServer != ws {
		t.Fatal("plugin manager wsServer was cleared after close timeout")
	}
	got := module.DebugSnapshot()
	if !got.WebSocket.Closing || !got.WebSocket.ServeRunning {
		t.Fatalf("snapshot after close timeout = %#v, want retained closing websocket", got)
	}

	close(serveDone)
	if err := module.CloseContext(context.Background()); err != nil {
		t.Fatalf("retry CloseContext: %v", err)
	}
	if manager.wsServer != nil {
		t.Fatal("plugin manager wsServer was not cleared after successful retry")
	}
}

func TestPluginDebugSnapshotReportsClosingDuringSlowLeaseClose(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	handle := &pluginCloseHandle{closeStarted: started, closeRelease: release}
	module := &Module{pluginRestarterLease: handle}
	module.pluginRestarterLeaseActive.Store(true)

	done := make(chan error, 1)
	go func() {
		done <- module.CloseContext(context.Background())
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for plugin restarter lease close to start")
	}

	snapshotDone := make(chan DebugSnapshot, 1)
	go func() {
		snapshotDone <- module.DebugSnapshot()
	}()
	select {
	case snapshot := <-snapshotDone:
		if !snapshot.Closing || !snapshot.PluginRestarterLeaseAttached {
			t.Fatalf("snapshot during close = %#v, want closing attached restarter lease", snapshot)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("DebugSnapshot blocked while plugin restarter lease Close was in progress")
	}

	close(release)
	if err := <-done; err != nil {
		t.Fatalf("CloseContext: %v", err)
	}
	snapshot := module.DebugSnapshot()
	if snapshot.Closing || snapshot.PluginRestarterLeaseAttached {
		t.Fatalf("snapshot after close = %#v, want detached restarter lease", snapshot)
	}
}

func TestPluginManagerStopAllBlocksLateStarts(t *testing.T) {
	host := newTestPluginHost()
	module := &Module{kit: host}
	manager := newPluginManager(module)
	done := make(chan struct{})
	close(done)
	manager.plugins["existing"] = &pluginConn{
		config: PluginConfig{Name: "existing"},
		done:   done,
	}

	if err := manager.stopAll(context.Background()); err != nil {
		t.Fatalf("stopAll: %v", err)
	}

	err := manager.startPlugin(PluginConfig{Name: "late-start"}, 0)
	if !errors.Is(err, errPluginManagerStopping) {
		t.Fatalf("startPlugin after stopAll error = %v, want %v", err, errPluginManagerStopping)
	}
	if got := manager.listPlugins(); len(got) != 0 {
		t.Fatalf("running plugins after stopAll = %#v, want none", got)
	}
}

func TestPluginManagerStaleStopDoesNotDeleteReplacement(t *testing.T) {
	host := newTestPluginHost()
	module := &Module{kit: host}
	manager := newPluginManager(module)
	oldDone := make(chan struct{})
	close(oldDone)
	oldConn := &pluginConn{
		config: PluginConfig{Name: "demo"},
		done:   oldDone,
	}
	newConn := &pluginConn{
		config: PluginConfig{Name: "demo"},
		done:   make(chan struct{}),
	}
	manager.plugins["demo"] = newConn

	if err := manager.stopPlugin(context.Background(), "demo", oldConn); err != nil {
		t.Fatalf("stopPlugin: %v", err)
	}

	manager.mu.Lock()
	got := manager.plugins["demo"]
	manager.mu.Unlock()
	if got != newConn {
		t.Fatalf("stale stop deleted replacement plugin connection")
	}
}

func TestPluginStopKeepsUndrainedProcessForRetry(t *testing.T) {
	oldKillWait := pluginKillWaitTimeout
	pluginKillWaitTimeout = 10 * time.Millisecond
	t.Cleanup(func() { pluginKillWaitTimeout = oldKillWait })

	host := newTestPluginHost()
	module := &Module{kit: host}
	manager := newPluginManager(module)
	module.manager = manager
	pc := &pluginConn{
		config: PluginConfig{Name: "stuck"},
		done:   make(chan struct{}),
		stopCh: make(chan struct{}),
	}
	manager.plugins["stuck"] = pc

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := manager.stopPlugin(ctx, "stuck", pc)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("stopPlugin error = %v, want context canceled", err)
	}
	manager.mu.Lock()
	got := manager.plugins["stuck"]
	manager.mu.Unlock()
	if got != pc {
		t.Fatal("undrained plugin was removed after close deadline")
	}
	snapshot := module.DebugSnapshot()
	if snapshot.RunningProcesses != 1 || snapshot.StoppingProcesses != 1 {
		t.Fatalf("snapshot after timed-out stop = %#v, want one running/stopping process", snapshot)
	}

	close(pc.done)
	if err := manager.stopPlugin(context.Background(), "stuck", pc); err != nil {
		t.Fatalf("retry stopPlugin: %v", err)
	}
	manager.mu.Lock()
	got = manager.plugins["stuck"]
	manager.mu.Unlock()
	if got != nil {
		t.Fatal("drained plugin was not removed after retry")
	}
	snapshot = module.DebugSnapshot()
	if snapshot.RunningProcesses != 0 || snapshot.StoppingProcesses != 0 {
		t.Fatalf("snapshot after retry = %#v, want no running/stopping processes", snapshot)
	}
}

func TestPluginStopWakesRestartBackoff(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not available")
	}
	host := newTestPluginHost()
	module := &Module{kit: host}
	manager := newPluginManager(module)

	err = manager.startPlugin(PluginConfig{
		Name:            "quick-exit",
		Binary:          sh,
		Args:            []string{"-c", "echo READY:quick-exit"},
		AutoRestart:     true,
		MaxRestarts:     5,
		StartTimeout:    time.Second,
		ShutdownTimeout: time.Second,
	}, 0)
	if err != nil {
		t.Fatalf("startPlugin: %v", err)
	}
	manager.mu.Lock()
	pc := manager.plugins["quick-exit"]
	manager.mu.Unlock()
	if pc == nil {
		t.Fatal("quick-exit plugin was not tracked")
	}

	select {
	case <-pc.exited:
	case <-time.After(time.Second):
		t.Fatal("quick-exit process did not exit")
	}

	start := time.Now()
	if err := manager.stopPlugin(context.Background(), "quick-exit", pc); err != nil {
		t.Fatalf("stopPlugin: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("stopPlugin waited for restart backoff: %s", elapsed)
	}
}

func TestPluginStopKillsProcessAfterShutdownTimeout(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not available")
	}
	host := newTestPluginHost()
	module := &Module{kit: host}
	manager := newPluginManager(module)

	err = manager.startPlugin(PluginConfig{
		Name:            "stubborn",
		Binary:          sh,
		Args:            []string{"-c", "trap '' TERM; echo READY:stubborn; while true; do sleep 1; done"},
		StartTimeout:    time.Second,
		ShutdownTimeout: 50 * time.Millisecond,
	}, 0)
	if err != nil {
		t.Fatalf("startPlugin: %v", err)
	}
	manager.mu.Lock()
	pc := manager.plugins["stubborn"]
	manager.mu.Unlock()
	if pc == nil {
		t.Fatal("stubborn plugin was not tracked")
	}

	start := time.Now()
	err = manager.stopPlugin(context.Background(), "stubborn", pc)
	if err == nil {
		t.Fatal("stopPlugin: expected shutdown timeout error")
	}
	if !strings.Contains(err.Error(), "shutdown timeout") {
		t.Fatalf("stopPlugin error = %v, want shutdown timeout", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("stopPlugin waited %s after shutdown timeout", elapsed)
	}
	if got := manager.listPlugins(); len(got) != 0 {
		t.Fatalf("running plugins after shutdown timeout = %#v, want none", got)
	}
}

func TestPluginStartFailureBeforeReadyWaitsAndClosesLazyWSServer(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not available")
	}
	host := newTestPluginHost()
	module := &Module{kit: host}
	manager := newPluginManager(module)

	start := time.Now()
	err = manager.startPlugin(PluginConfig{
		Name:         "exits-before-ready",
		Binary:       sh,
		Args:         []string{"-c", "exit 7"},
		StartTimeout: 5 * time.Second,
	}, 0)
	if err == nil {
		t.Fatal("startPlugin: expected error")
	}
	if !strings.Contains(err.Error(), "exited before READY") {
		t.Fatalf("startPlugin error = %v, want exited before READY", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("startPlugin waited %s for an already-exited plugin", elapsed)
	}
	if manager.wsServer != nil {
		t.Fatal("lazy plugin websocket server still open after startup failure")
	}
	if got := manager.listPlugins(); len(got) != 0 {
		t.Fatalf("running plugins after startup failure = %#v, want none", got)
	}
}

func TestPluginReadyTimeoutWaitsAndClosesLazyWSServer(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not available")
	}
	host := newTestPluginHost()
	module := &Module{kit: host}
	manager := newPluginManager(module)

	start := time.Now()
	err = manager.startPlugin(PluginConfig{
		Name:         "never-ready",
		Binary:       sh,
		Args:         []string{"-c", "sleep 10"},
		StartTimeout: 50 * time.Millisecond,
	}, 0)
	if err == nil {
		t.Fatal("startPlugin: expected timeout")
	}
	if !strings.Contains(err.Error(), "plugin READY") {
		t.Fatalf("startPlugin error = %v, want plugin READY timeout", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("startPlugin waited %s after READY timeout", elapsed)
	}
	if manager.wsServer != nil {
		t.Fatal("lazy plugin websocket server still open after READY timeout")
	}
	if got := manager.listPlugins(); len(got) != 0 {
		t.Fatalf("running plugins after READY timeout = %#v, want none", got)
	}
}

func registerPluginWS(t *testing.T, ctx context.Context, url string, manifest pluginws.Manifest) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial plugin ws server: %v", err)
	}
	manifestData, _ := json.Marshal(manifest)
	if err := wsjson.Write(ctx, conn, pluginws.Message{Type: pluginws.TypeManifest, Data: manifestData}); err != nil {
		conn.CloseNow()
		t.Fatalf("write manifest: %v", err)
	}
	var ackMsg pluginws.Message
	if err := wsjson.Read(ctx, conn, &ackMsg); err != nil {
		conn.CloseNow()
		t.Fatalf("read manifest ack: %v", err)
	}
	if ackMsg.Type != pluginws.TypeManifestAck {
		conn.CloseNow()
		t.Fatalf("ack type = %q, want %q", ackMsg.Type, pluginws.TypeManifestAck)
	}
	var ack pluginws.ManifestAck
	if err := json.Unmarshal(ackMsg.Data, &ack); err != nil {
		conn.CloseNow()
		t.Fatalf("decode manifest ack: %v", err)
	}
	if !ack.Registered || ack.Error != "" {
		conn.CloseNow()
		t.Fatalf("manifest ack = %#v, want registered without error", ack)
	}
	return conn
}

func waitForPluginTestCondition(t *testing.T, timeout time.Duration, fn func() bool, label string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if fn() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", label)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

const testTimeout = 5 * time.Second

type testPluginHost struct {
	tools     *toolreg.ToolRegistry
	audit     *auditpkg.Recorder
	tracer    *coretracing.Tracer
	logger    *slog.Logger
	shutdown  chan struct{}
	published atomic.Int32
}

func newTestPluginHost() *testPluginHost {
	return &testPluginHost{
		tools:    toolreg.New(),
		audit:    auditpkg.NewRecorder(nil, "test", "test"),
		tracer:   coretracing.NewTracer(nil, 1),
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		shutdown: make(chan struct{}),
	}
}

func (h *testPluginHost) PublishRaw(context.Context, string, json.RawMessage) (string, error) {
	h.published.Add(1)
	return "", nil
}
func (h *testPluginHost) TransportKind() string { return "nats" }
func (h *testPluginHost) Store() Store          { return nil }
func (h *testPluginHost) Logger() *slog.Logger  { return h.logger }
func (h *testPluginHost) ReportError(error, types.ErrorContext) {
}
func (h *testPluginHost) LeasePluginChecker(context.Context, bkmodule.PluginChecker) (bkmodule.Handle, error) {
	return bkmodule.HandleFunc(func(context.Context) error { return nil }), nil
}
func (h *testPluginHost) LeasePluginRestarter(context.Context, plugincap.Restarter) (bkmodule.Handle, error) {
	return bkmodule.HandleFunc(func(context.Context) error { return nil }), nil
}
func (h *testPluginHost) Audit() *auditpkg.Recorder       { return h.audit }
func (h *testPluginHost) Tools() *toolreg.ToolRegistry    { return h.tools }
func (h *testPluginHost) Tracer() *coretracing.Tracer     { return h.tracer }
func (h *testPluginHost) Remote() *transport.RemoteClient { return nil }
func (h *testPluginHost) Namespace() string               { return "test" }
func (h *testPluginHost) CallerID() string                { return "test" }
func (h *testPluginHost) SecretStore() types.SecretStore  { return nil }
func (h *testPluginHost) ShutdownSignal() <-chan struct{} { return h.shutdown }

type testPluginStore struct{}

func (testPluginStore) LoadRunningPlugins() ([]types.RunningPluginRecord, error) { return nil, nil }
func (testPluginStore) SaveRunningPlugin(types.RunningPluginRecord) error        { return nil }
func (testPluginStore) DeleteRunningPlugin(string) error                         { return nil }
func (testPluginStore) LoadInstalledPlugins() ([]types.InstalledPlugin, error)   { return nil, nil }

type pluginCloseContextKey struct{}

type pluginTestSubscriptionHandle struct {
	stop  func()
	close func(context.Context) error
}

func (h pluginTestSubscriptionHandle) Stop() {
	if h.stop != nil {
		h.stop()
	}
}

func (h pluginTestSubscriptionHandle) CloseContext(ctx context.Context) error {
	if h.close != nil {
		return h.close(ctx)
	}
	return nil
}

type pluginCloseHandle struct {
	mu           sync.Mutex
	ctx          context.Context
	err          error
	closed       int
	closeStarted chan struct{}
	closeRelease chan struct{}
	closeOnce    sync.Once
}

func (h *pluginCloseHandle) Close(ctx context.Context) error {
	h.mu.Lock()
	h.ctx = ctx
	h.closed++
	err := h.err
	started := h.closeStarted
	release := h.closeRelease
	h.mu.Unlock()

	if started != nil {
		h.closeOnce.Do(func() { close(started) })
	}
	if release != nil {
		<-release
	}
	return err
}

func (h *pluginCloseHandle) setErr(err error) {
	h.mu.Lock()
	h.err = err
	h.mu.Unlock()
}

func (h *pluginCloseHandle) closeCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.closed
}
