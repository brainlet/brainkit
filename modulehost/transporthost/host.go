package transporthost

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk"
)

// Host owns the runtime transport, router, command host, remote client, caller,
// and bus metrics for one Kit runtime.
type Host struct {
	transport     *transport.Transport
	router        *message.Router
	remote        *transport.RemoteClient
	commandHost   *transport.Host
	caller        *sdk.Caller
	busMetrics    *transport.Metrics
	ownsTransport bool
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

	router, err := message.NewRouter(message.RouterConfig{}, watermill.NopLogger{})
	if err != nil {
		if ownsTransport {
			_ = transportSet.Close()
		}
		return nil, fmt.Errorf("brainkit: router: %w", err)
	}

	busMetrics := transport.NewMetrics()
	router.AddMiddleware(
		transport.DepthMiddleware,
		transport.CallerIDMiddleware(cfg.CallerID),
		transport.MetricsMiddleware(busMetrics),
	)
	if cfg.MaxConcurrency > 0 {
		router.AddMiddleware(transport.MaxConcurrencyMiddleware(cfg.MaxConcurrency))
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
		runtimeID = watermill.NewUUID()
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

// Running returns a channel closed by Watermill once handlers are subscribed.
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

// CloseRouter stops the Watermill router.
func (h *Host) CloseRouter() error {
	if h == nil || h.router == nil {
		return nil
	}
	return h.router.Close()
}

// CloseCaller closes the shared reply router.
func (h *Host) CloseCaller() error {
	if h == nil || h.caller == nil {
		return nil
	}
	return h.caller.Close()
}

// CloseOwnedTransport closes the transport only when this host created it.
func (h *Host) CloseOwnedTransport() error {
	if h == nil || !h.ownsTransport || h.transport == nil {
		return nil
	}
	return h.transport.Close()
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
	wmsg := message.NewMessage(watermill.NewUUID(), []byte(payload))
	wmsg.SetContext(ctx)
	wmsg.Metadata.Set("correlationId", correlationID)
	if done {
		wmsg.Metadata.Set("done", "true")
	}
	if envelope {
		wmsg.Metadata.Set("envelope", "true")
	}
	return h.transport.Publisher.Publish(replyTo, wmsg)
}
