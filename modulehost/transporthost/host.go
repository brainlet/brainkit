package transporthost

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/sdk"
	"github.com/google/uuid"
)

// Host owns the runtime transport, router, command host, remote client, caller,
// and bus metrics for one Kit runtime.
type Host struct {
	transport     *transport.Transport
	router        *transport.Router
	remote        *transport.RemoteClient
	commandHost   *transport.Host
	caller        *sdk.Caller
	busMetrics    *transport.Metrics
	ownsTransport bool

	closingRouter    atomic.Bool
	closingCaller    atomic.Bool
	closingTransport atomic.Bool
	closedRouter     atomic.Bool
	closedCaller     atomic.Bool
	closedTransport  atomic.Bool
}

// DebugSnapshot is a test/debug view of transport-host lifecycle bookkeeping.
type DebugSnapshot struct {
	Router              transport.RouterDebugSnapshot
	Caller              sdk.CallerDebugSnapshot
	OwnsTransport       bool
	ActiveSubscriptions int64
	ClosingRouter       bool
	ClosingCaller       bool
	ClosingTransport    bool
	ClosedRouter        bool
	ClosedCaller        bool
	ClosedTransport     bool
}

// New builds the transport host from kernel configuration. Concrete network
// backends are still injected through cfg.Transport; otherwise a memory
// transport is created.
func New(cfg types.KernelConfig, logger *slog.Logger) (*Host, error) {
	transportSet, ownsTransport, err := resolveTransport(cfg)
	if err != nil {
		return nil, err
	}

	remote := transport.NewRemoteClientWithTransport(cfg.Namespace, cfg.CallerID, transportSet)
	remote.SetIdentity(cfg.ClusterID, cfg.RuntimeID)

	busMetrics := transport.NewMetrics()
	remote.SetMetrics(busMetrics)
	router, err := transport.NewRouter(cfg.CallerID, busMetrics, cfg.MaxConcurrency)
	if err != nil {
		if ownsTransport {
			_ = transportSet.Close()
		}
		return nil, fmt.Errorf("brainkit: router: %w", err)
	}

	h := &Host{
		transport:     transportSet,
		router:        router,
		remote:        remote,
		commandHost:   transport.NewHostWithTransport(cfg.Namespace, router, transportSet),
		busMetrics:    busMetrics,
		ownsTransport: ownsTransport,
	}

	h.commandHost.RegisterCommands([]transport.RawCommandBinding{{
		Name:  "_brainkit.router.keepalive",
		Topic: "_brainkit.router.keepalive",
		Handle: func(context.Context, json.RawMessage) (json.RawMessage, error) {
			return nil, nil
		},
	}})

	runtimeID := cfg.RuntimeID
	if runtimeID == "" {
		runtimeID = uuid.NewString()
	}
	caller, err := sdk.NewCaller(h, runtimeID, logger)
	if err != nil {
		if ownsTransport {
			_ = transportSet.Close()
		}
		return nil, fmt.Errorf("brainkit: caller: %w", err)
	}
	h.caller = caller
	return h, nil
}

func resolveTransport(cfg types.KernelConfig) (*transport.Transport, bool, error) {
	if cfg.Transport != nil {
		transportSet, ok := cfg.Transport.(*transport.Transport)
		if !ok {
			return nil, false, fmt.Errorf("brainkit: transport must be *transport.Transport, got %T", cfg.Transport)
		}
		return transportSet, false, nil
	}
	transportSet, err := transport.NewTransportSet(transport.TransportConfig{Type: "memory", Namespace: cfg.Namespace})
	if err != nil {
		return nil, false, fmt.Errorf("brainkit: internal transport: %w", err)
	}
	return transportSet, true, nil
}

// Caller returns the shared-inbox reply router.
func (h *Host) Caller() *sdk.Caller {
	if h == nil {
		return nil
	}
	return h.caller
}

// Remote returns the transport-level remote client.
func (h *Host) Remote() *transport.RemoteClient {
	if h == nil {
		return nil
	}
	return h.remote
}

// TransportKind returns the normalized transport kind.
func (h *Host) TransportKind() string {
	if h == nil || h.transport == nil {
		return ""
	}
	return h.transport.Kind
}

// Metrics returns bus message counters.
func (h *Host) Metrics() *transport.Metrics {
	if h == nil {
		return nil
	}
	return h.busMetrics
}

// DebugSnapshot returns lifecycle bookkeeping counts for teardown tests.
func (h *Host) DebugSnapshot() DebugSnapshot {
	if h == nil {
		return DebugSnapshot{}
	}
	var active int64
	if h.remote != nil {
		active = h.remote.ActiveSubscriptions()
	}
	var router transport.RouterDebugSnapshot
	if h.router != nil {
		router = h.router.DebugSnapshot()
	}
	var caller sdk.CallerDebugSnapshot
	if h.caller != nil {
		caller = h.caller.DebugSnapshot()
	}
	return DebugSnapshot{
		Router:              router,
		Caller:              caller,
		OwnsTransport:       h.ownsTransport,
		ActiveSubscriptions: active,
		ClosingRouter:       h.closingRouter.Load(),
		ClosingCaller:       h.closingCaller.Load(),
		ClosingTransport:    h.closingTransport.Load(),
		ClosedRouter:        h.closedRouter.Load(),
		ClosedCaller:        h.closedCaller.Load(),
		ClosedTransport:     h.closedTransport.Load(),
	}
}

// RegisterCommand live-mounts one command handler.
func (h *Host) RegisterCommand(ctx context.Context, binding transport.RawCommandBinding) (*transport.CommandHandle, error) {
	return h.commandHost.RegisterCommand(ctx, binding)
}

// RegisterCommands installs command handlers before router startup.
func (h *Host) RegisterCommands(bindings []transport.RawCommandBinding) {
	h.commandHost.RegisterCommands(bindings)
}

// IsRunning reports whether the router is already running.
func (h *Host) IsRunning() bool {
	if h == nil || h.router == nil {
		return false
	}
	select {
	case <-h.router.Running():
		return true
	default:
		return false
	}
}

// Run starts the router in the background.
func (h *Host) Run(ctx context.Context) {
	go func() {
		_ = h.router.Run(ctx)
	}()
}

// Running returns a channel closed once handlers are subscribed.
func (h *Host) Running() <-chan struct{} {
	return h.router.Running()
}

// Start starts the router and waits until it is running or ctx expires.
func (h *Host) Start(ctx context.Context) error {
	if h.IsRunning() {
		return nil
	}
	h.Run(context.Background())
	select {
	case <-h.Running():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// CloseRouter stops the message router.
func (h *Host) CloseRouter() error {
	if h == nil || h.router == nil {
		return nil
	}
	h.closingRouter.Store(true)
	defer h.closingRouter.Store(false)
	if err := h.router.Close(); err != nil {
		return err
	}
	h.closedRouter.Store(true)
	return nil
}

// CloseCaller closes the shared reply router.
func (h *Host) CloseCaller() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return h.CloseCallerContext(ctx)
}

// CloseCallerContext closes the shared reply router under caller-owned
// lifecycle cancellation.
func (h *Host) CloseCallerContext(ctx context.Context) error {
	if h == nil || h.caller == nil {
		return nil
	}
	h.closingCaller.Store(true)
	defer h.closingCaller.Store(false)
	if err := h.caller.CloseContext(ctx); err != nil {
		return err
	}
	h.closedCaller.Store(true)
	return nil
}

// CloseOwnedTransport closes the transport only when this host created it.
func (h *Host) CloseOwnedTransport() error {
	if h == nil || !h.ownsTransport || h.transport == nil {
		return nil
	}
	h.closingTransport.Store(true)
	defer h.closingTransport.Store(false)
	if err := h.transport.Close(); err != nil {
		return err
	}
	h.closedTransport.Store(true)
	return nil
}

// Close releases all transport-host resources. Kernel normally calls the
// narrower close methods to preserve shutdown ordering around JS/storage state.
func (h *Host) Close() error {
	var firstErr error
	collect := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	collect(h.CloseRouter())
	collect(h.CloseCaller())
	collect(h.CloseOwnedTransport())
	return firstErr
}

// PublishRaw sends a message to a topic.
func (h *Host) PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (string, error) {
	return h.remote.PublishRaw(ctx, topic, payload)
}

// SubscribeRaw subscribes to a topic.
func (h *Host) SubscribeRaw(ctx context.Context, topic string, handler func(sdk.Message)) (func(), error) {
	return h.remote.SubscribeRaw(ctx, topic, handler)
}

// SubscribeRawHandle subscribes to a topic with context-aware close semantics.
func (h *Host) SubscribeRawHandle(ctx context.Context, topic string, handler func(sdk.Message)) (bkmodule.Handle, error) {
	handle, err := h.remote.SubscribeRawHandle(ctx, topic, handler)
	if err != nil {
		return nil, err
	}
	return bkmodule.HandleFunc(handle.CloseContext), nil
}

// PublishRawToNamespace publishes to a target namespace.
func (h *Host) PublishRawToNamespace(ctx context.Context, targetNamespace, topic string, payload json.RawMessage) (string, error) {
	return h.remote.PublishRawToNamespace(ctx, targetNamespace, topic, payload)
}

// SubscribeRawToNamespace subscribes in a target namespace.
func (h *Host) SubscribeRawToNamespace(ctx context.Context, targetNamespace, topic string, handler func(sdk.Message)) (func(), error) {
	return h.remote.SubscribeRawToNamespace(ctx, targetNamespace, topic, handler)
}

// PublishRawWithMeta sends a message with extra metadata.
func (h *Host) PublishRawWithMeta(ctx context.Context, topic string, payload json.RawMessage, extra map[string]string) (string, error) {
	return h.remote.PublishRawWithMeta(ctx, topic, payload, extra)
}

// SubscribeRawFanOut subscribes using the fan-out subscriber.
func (h *Host) SubscribeRawFanOut(ctx context.Context, topic string, handler func(sdk.Message)) (func(), error) {
	return h.remote.SubscribeRawFanOut(ctx, topic, handler)
}

// PublishReply publishes directly to an already resolved reply topic.
func (h *Host) PublishReply(ctx context.Context, replyTo, correlationID string, payload json.RawMessage, done bool, envelope bool) error {
	if replyTo == "" {
		return nil
	}
	return h.transport.PublishResolved(ctx, replyTo, correlationID, payload, done, envelope)
}
