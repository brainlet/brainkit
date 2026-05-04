package mcp

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/mcp/mcpmsg"
	toolsmod "github.com/brainlet/brainkit/modules/tools"
	"github.com/brainlet/brainkit/sdk"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func TestMountAutoMountsToolsDependency(t *testing.T) {
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test-mcp",
	})
	if err != nil {
		t.Fatalf("brainkit.New: %v", err)
	}
	t.Cleanup(func() { _ = k.Close() })

	if _, ok := k.Module("tools"); ok {
		t.Fatalf("tools should not be mounted before mcp dependency resolution")
	}

	if err := k.Mount(context.Background(), New(nil)); err != nil {
		t.Fatalf("Mount(mcp): %v", err)
	}

	if _, ok := k.Module("tools"); !ok {
		t.Fatalf("mcp mount should auto-mount tools dependency")
	}

	resp, err := mcpmsg.CallMcpListTools(k, context.Background(), mcpmsg.McpListToolsMsg{}, sdk.WithCallTimeout(2*time.Second))
	if err != nil {
		t.Fatalf("mcp.listTools with no servers: %v", err)
	}
	if len(resp.Tools) != 0 {
		t.Fatalf("mcp.listTools tools = %#v, want none", resp.Tools)
	}

	if _, err := mcpmsg.CallMcpCallTool(k, context.Background(), mcpmsg.McpCallToolMsg{
		Server: "missing",
		Tool:   "echo",
		Args:   map[string]any{},
	}, sdk.WithCallTimeout(2*time.Second)); err == nil {
		t.Fatal("mcp.callTool against missing server should fail")
	}
}

func TestStartupOrdersExplicitToolsDependencyBeforeMCP(t *testing.T) {
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test-mcp",
		Modules: []bkmodule.Module{
			New(nil),
			toolsmod.New(),
		},
	})
	if err != nil {
		t.Fatalf("brainkit.New: %v", err)
	}
	t.Cleanup(func() { _ = k.Close() })

	if _, ok := k.Module("tools"); !ok {
		t.Fatalf("tools should be mounted from explicit dependency")
	}
	if _, ok := k.Module("mcp"); !ok {
		t.Fatalf("mcp should be mounted after tools")
	}
}

func TestDebugSnapshotReportsClosingState(t *testing.T) {
	module := New(map[string]ServerConfig{"demo": {}})
	module.manager = NewManager()
	module.closing.Store(true)

	got := module.DebugSnapshot()
	if !got.Closing {
		t.Fatal("Closing = false, want true")
	}
	if !got.ManagerAttached {
		t.Fatal("ManagerAttached = false, want true")
	}
	if got.ConfiguredServers != 1 {
		t.Fatalf("ConfiguredServers = %d, want 1", got.ConfiguredServers)
	}
}

func TestManagerConnectReturnsInitializeAndCleanupCloseErrors(t *testing.T) {
	initErr := errors.New("initialize failed")
	closeErr := errors.New("close failed")
	client := &fakeMCPClient{initErr: initErr, closeErr: closeErr}
	manager := newManagerWithFactory(func(context.Context, string, ServerConfig) (mcpClient, error) {
		return client, nil
	})

	err := manager.Connect(context.Background(), "bad", ServerConfig{})
	if !errors.Is(err, initErr) {
		t.Fatalf("Connect error = %v, want initialize error %v", err, initErr)
	}
	if !errors.Is(err, closeErr) {
		t.Fatalf("Connect error = %v, want cleanup close error %v", err, closeErr)
	}
	if got := client.closeCalls.Load(); got != 1 {
		t.Fatalf("close calls = %d, want 1", got)
	}
	if got := manager.DebugSnapshot().ConnectedServers; got != 0 {
		t.Fatalf("connected servers = %d, want 0", got)
	}
}

func TestManagerConnectAfterCloseIsRejected(t *testing.T) {
	manager := newManagerWithFactory(func(context.Context, string, ServerConfig) (mcpClient, error) {
		t.Fatal("factory should not be called after manager close")
		return nil, nil
	})

	if err := manager.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	err := manager.Connect(context.Background(), "late", ServerConfig{})
	if !errors.Is(err, errMCPManagerClosed) {
		t.Fatalf("Connect after Close error = %v, want %v", err, errMCPManagerClosed)
	}
}

func TestManagerConnectDuplicateServerIsRejected(t *testing.T) {
	var calls atomic.Int32
	client := &fakeMCPClient{}
	manager := newManagerWithFactory(func(context.Context, string, ServerConfig) (mcpClient, error) {
		calls.Add(1)
		return client, nil
	})

	if err := manager.Connect(context.Background(), "dupe", ServerConfig{}); err != nil {
		t.Fatalf("first Connect: %v", err)
	}
	err := manager.Connect(context.Background(), "dupe", ServerConfig{})
	if !errors.Is(err, errMCPServerAlreadyConnected) {
		t.Fatalf("duplicate Connect error = %v, want %v", err, errMCPServerAlreadyConnected)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("factory calls = %d, want 1", got)
	}
	snapshot := manager.DebugSnapshot()
	if snapshot.ConnectedServers != 1 || len(snapshot.Servers) != 1 || snapshot.Servers[0] != "dupe" {
		t.Fatalf("snapshot after duplicate connect = %#v, want original client retained", snapshot)
	}
}

func TestManagerConnectClosedDuringInitializeCleansUpClient(t *testing.T) {
	initStarted := make(chan struct{})
	initRelease := make(chan struct{})
	client := &fakeMCPClient{
		initStarted: initStarted,
		initRelease: initRelease,
	}
	manager := newManagerWithFactory(func(context.Context, string, ServerConfig) (mcpClient, error) {
		return client, nil
	})

	done := make(chan error, 1)
	go func() {
		done <- manager.Connect(context.Background(), "late", ServerConfig{})
	}()
	select {
	case <-initStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for initialize to start")
	}
	if err := manager.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	close(initRelease)
	err := <-done
	if !errors.Is(err, errMCPManagerClosed) {
		t.Fatalf("Connect error = %v, want %v", err, errMCPManagerClosed)
	}
	if got := client.closeCalls.Load(); got != 1 {
		t.Fatalf("close calls = %d, want 1", got)
	}
	snapshot := manager.DebugSnapshot()
	if snapshot.ConnectedServers != 0 || snapshot.CachedTools != 0 {
		t.Fatalf("snapshot after rejected connect = %#v, want no clients", snapshot)
	}
}

func TestManagerCloseReturnsErrorsAndKeepsFailedClients(t *testing.T) {
	want := errors.New("close failed")
	manager := NewManager()
	okClient := &fakeMCPClient{}
	failedClient := &fakeMCPClient{closeErr: want}
	manager.clients["ok"] = &mcpClientEntry{client: okClient}
	manager.clients["fail"] = &mcpClientEntry{client: failedClient}
	manager.tools["ok"] = []ToolInfo{{Name: "ok"}}
	manager.tools["fail"] = []ToolInfo{{Name: "fail"}}

	err := manager.Close()
	if !errors.Is(err, want) {
		t.Fatalf("Close error = %v, want %v", err, want)
	}
	if got := okClient.closeCalls.Load(); got != 1 {
		t.Fatalf("ok close calls = %d, want 1", got)
	}
	if got := failedClient.closeCalls.Load(); got != 1 {
		t.Fatalf("failed close calls = %d, want 1", got)
	}
	snapshot := manager.DebugSnapshot()
	if snapshot.Closing {
		t.Fatal("Closing = true after Close returned")
	}
	if snapshot.ConnectedServers != 1 || snapshot.CachedTools != 1 || len(snapshot.Servers) != 1 || snapshot.Servers[0] != "fail" {
		t.Fatalf("snapshot after failed close = %#v, want only failed client retained", snapshot)
	}

	failedClient.closeErr = nil
	if err := manager.Close(); err != nil {
		t.Fatalf("retry Close: %v", err)
	}
	if got := failedClient.closeCalls.Load(); got != 2 {
		t.Fatalf("failed close calls after retry = %d, want 2", got)
	}
	snapshot = manager.DebugSnapshot()
	if snapshot.ConnectedServers != 0 || snapshot.CachedTools != 0 {
		t.Fatalf("snapshot after retry = %#v, want no clients", snapshot)
	}
}

func TestManagerDisconnectReturnsErrorsAndKeepsFailedClient(t *testing.T) {
	want := errors.New("close failed")
	manager := NewManager()
	client := &fakeMCPClient{closeErr: want}
	manager.clients["fail"] = &mcpClientEntry{client: client}
	manager.tools["fail"] = []ToolInfo{{Name: "fail"}}

	err := manager.Disconnect("fail")
	if !errors.Is(err, want) {
		t.Fatalf("Disconnect error = %v, want %v", err, want)
	}
	snapshot := manager.DebugSnapshot()
	if snapshot.ConnectedServers != 1 || snapshot.CachedTools != 1 {
		t.Fatalf("snapshot after failed disconnect = %#v, want client retained", snapshot)
	}

	client.closeErr = nil
	if err := manager.Disconnect("fail"); err != nil {
		t.Fatalf("retry Disconnect: %v", err)
	}
	snapshot = manager.DebugSnapshot()
	if snapshot.ConnectedServers != 0 || snapshot.CachedTools != 0 {
		t.Fatalf("snapshot after retry = %#v, want no clients", snapshot)
	}
}

func TestManagerDebugSnapshotReportsClosingDuringSlowClose(t *testing.T) {
	manager := NewManager()
	started := make(chan struct{})
	release := make(chan struct{})
	manager.clients["slow"] = &mcpClientEntry{client: &fakeMCPClient{
		closeStarted: started,
		closeRelease: release,
	}}
	manager.tools["slow"] = []ToolInfo{{Name: "slow"}}

	done := make(chan error, 1)
	go func() {
		done <- manager.Close()
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for close to start")
	}

	snapshotDone := make(chan DebugSnapshot, 1)
	go func() {
		snapshotDone <- manager.DebugSnapshot()
	}()
	select {
	case snapshot := <-snapshotDone:
		if !snapshot.Closing {
			t.Fatalf("snapshot during close = %#v, want Closing=true", snapshot)
		}
		if snapshot.ClosingClients != 1 {
			t.Fatalf("snapshot during close = %#v, want one closing client", snapshot)
		}
		if snapshot.ConnectedServers != 1 || snapshot.CachedTools != 1 || len(snapshot.Servers) != 1 || snapshot.Servers[0] != "slow" {
			t.Fatalf("snapshot during close = %#v, want slow client still tracked", snapshot)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("DebugSnapshot blocked while client Close was in progress")
	}

	close(release)
	if err := <-done; err != nil {
		t.Fatalf("Close: %v", err)
	}
	snapshot := manager.DebugSnapshot()
	if snapshot.Closing || snapshot.ConnectedServers != 0 || snapshot.CachedTools != 0 {
		t.Fatalf("snapshot after close = %#v, want closed manager", snapshot)
	}
}

func TestManagerCloseContextReturnsDeadlineForStuckClientClose(t *testing.T) {
	manager := NewManager()
	started := make(chan struct{})
	release := make(chan struct{})
	manager.clients["slow"] = &mcpClientEntry{client: &fakeMCPClient{
		closeStarted: started,
		closeRelease: release,
	}}
	manager.tools["slow"] = []ToolInfo{{Name: "slow"}}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := manager.CloseContext(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CloseContext error = %v, want context deadline exceeded", err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("client close did not start")
	}
	snapshot := manager.DebugSnapshot()
	if snapshot.Closing || snapshot.ClosingClients != 1 || snapshot.ConnectedServers != 1 || snapshot.CachedTools != 1 {
		t.Fatalf("snapshot after close timeout = %#v, want retained closing client", snapshot)
	}

	close(release)
	requireEventuallyMCPDebug(t, func(snapshot DebugSnapshot) bool {
		return snapshot.ClosingClients == 0 && snapshot.ConnectedServers == 0 && snapshot.CachedTools == 0
	}, manager.DebugSnapshot, "stuck client cleanup")
}

func TestModuleDebugSnapshotReportsClosingDuringSlowClose(t *testing.T) {
	module := New(map[string]ServerConfig{"slow": {}})
	started := make(chan struct{})
	release := make(chan struct{})
	manager := NewManager()
	manager.clients["slow"] = &mcpClientEntry{client: &fakeMCPClient{
		closeStarted: started,
		closeRelease: release,
	}}
	manager.tools["slow"] = []ToolInfo{{Name: "slow"}}
	module.manager = manager

	done := make(chan error, 1)
	go func() {
		done <- module.Close()
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for close to start")
	}

	snapshotDone := make(chan DebugSnapshot, 1)
	go func() {
		snapshotDone <- module.DebugSnapshot()
	}()
	select {
	case snapshot := <-snapshotDone:
		if !snapshot.Closing || !snapshot.ManagerAttached {
			t.Fatalf("snapshot during module close = %#v, want closing attached manager", snapshot)
		}
		if snapshot.ClosingClients != 1 {
			t.Fatalf("snapshot during module close = %#v, want one closing client", snapshot)
		}
		if snapshot.ConnectedServers != 1 || snapshot.CachedTools != 1 || len(snapshot.Servers) != 1 || snapshot.Servers[0] != "slow" {
			t.Fatalf("snapshot during module close = %#v, want slow client still tracked", snapshot)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("DebugSnapshot blocked while module Close was in progress")
	}

	close(release)
	if err := <-done; err != nil {
		t.Fatalf("Close: %v", err)
	}
	snapshot := module.DebugSnapshot()
	if snapshot.Closing || snapshot.ManagerAttached {
		t.Fatalf("snapshot after close = %#v, want detached manager", snapshot)
	}
}

func TestModuleCloseContextKeepsManagerWhileClientCloseTimesOut(t *testing.T) {
	module := New(map[string]ServerConfig{"slow": {}})
	started := make(chan struct{})
	release := make(chan struct{})
	manager := NewManager()
	manager.clients["slow"] = &mcpClientEntry{client: &fakeMCPClient{
		closeStarted: started,
		closeRelease: release,
	}}
	manager.tools["slow"] = []ToolInfo{{Name: "slow"}}
	module.manager = manager

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := module.CloseContext(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CloseContext error = %v, want context deadline exceeded", err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("client close did not start")
	}
	snapshot := module.DebugSnapshot()
	if !snapshot.ManagerAttached || snapshot.Closing || snapshot.ClosingClients != 1 || snapshot.ConnectedServers != 1 {
		t.Fatalf("snapshot after module close timeout = %#v, want retained closing manager", snapshot)
	}

	close(release)
	requireEventuallyMCPDebug(t, func(snapshot DebugSnapshot) bool {
		return snapshot.ManagerAttached && snapshot.ClosingClients == 0 && snapshot.ConnectedServers == 0
	}, module.DebugSnapshot, "module manager client cleanup")
	if err := module.CloseContext(context.Background()); err != nil {
		t.Fatalf("retry CloseContext: %v", err)
	}
	if snapshot := module.DebugSnapshot(); snapshot.ManagerAttached {
		t.Fatalf("snapshot after retry = %#v, want detached manager", snapshot)
	}
}

func requireEventuallyMCPDebug(t *testing.T, ok func(DebugSnapshot) bool, snapshot func() DebugSnapshot, label string) {
	t.Helper()
	deadline := time.After(time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %s; last snapshot = %#v", label, snapshot())
		case <-ticker.C:
			if ok(snapshot()) {
				return
			}
		}
	}
}

type fakeMCPClient struct {
	initErr      error
	listToolsErr error
	tools        []mcplib.Tool
	callResult   *mcplib.CallToolResult
	callErr      error
	closeErr     error

	initStarted  chan struct{}
	initRelease  chan struct{}
	initOnce     sync.Once
	closeStarted chan struct{}
	closeRelease chan struct{}
	closeOnce    sync.Once
	closeCalls   atomic.Int32
}

func (f *fakeMCPClient) Initialize(context.Context, mcplib.InitializeRequest) (*mcplib.InitializeResult, error) {
	if f.initStarted != nil {
		f.initOnce.Do(func() { close(f.initStarted) })
	}
	if f.initRelease != nil {
		<-f.initRelease
	}
	if f.initErr != nil {
		return nil, f.initErr
	}
	return &mcplib.InitializeResult{}, nil
}

func (f *fakeMCPClient) ListTools(context.Context, mcplib.ListToolsRequest) (*mcplib.ListToolsResult, error) {
	if f.listToolsErr != nil {
		return nil, f.listToolsErr
	}
	return &mcplib.ListToolsResult{Tools: append([]mcplib.Tool(nil), f.tools...)}, nil
}

func (f *fakeMCPClient) CallTool(context.Context, mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	if f.callErr != nil {
		return nil, f.callErr
	}
	if f.callResult != nil {
		return f.callResult, nil
	}
	return &mcplib.CallToolResult{Content: []mcplib.Content{mcplib.TextContent{Type: "text", Text: `"ok"`}}}, nil
}

func (f *fakeMCPClient) Close() error {
	f.closeCalls.Add(1)
	if f.closeStarted != nil {
		f.closeOnce.Do(func() { close(f.closeStarted) })
	}
	if f.closeRelease != nil {
		<-f.closeRelease
	}
	return f.closeErr
}
