package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/sdk"
)

// ID reports the hot-mount module identifier.
func (gw *Gateway) ID() string { return "gateway" }

// Status reports maturity (stable).
func (gw *Gateway) Status() bkmodule.Status { return bkmodule.StatusStable }

func (gw *Gateway) Mount(_ context.Context, host bkmodule.Host) error {
	health, _ := bkmodule.Capability[bkmodule.HealthProbes](host, bkmodule.CapabilityHealthProbes)
	control, _ := bkmodule.Capability[bkmodule.RuntimeControl](host, bkmodule.CapabilityRuntimeControl)
	caller, err := bkmodule.RequireCapability[bkmodule.RequestCaller](host, bkmodule.CapabilityRequestCaller)
	if err != nil {
		return fmt.Errorf("gateway: %w", err)
	}
	gw.SetRuntime(mountedRuntime{
		messages: host.Messages(),
		health:   health,
		control:  control,
	})
	gw.caller = caller
	if err := gw.Start(); err != nil {
		return err
	}
	host.Scope().Resource(bkmodule.ResourceWithMetadata(
		bkmodule.ResourceKindHTTP,
		"gateway.listener",
		map[string]string{"listen": gw.config.Listen},
		"HTTP gateway listener.",
	))
	host.Scope().Defer(func(context.Context) error {
		gw.caller = nil
		return gw.Close()
	})
	return nil
}

// Close stops the HTTP server and unsubscribes bus route commands.
func (gw *Gateway) Close() error {
	return gw.Stop()
}

type mountedRuntime struct {
	messages bkmodule.MessageHost
	health   bkmodule.HealthProbes
	control  bkmodule.RuntimeControl
}

func (rt mountedRuntime) PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (string, error) {
	return rt.messages.PublishRaw(ctx, topic, payload)
}

func (rt mountedRuntime) SubscribeRaw(ctx context.Context, topic string, handler func(sdk.Message)) (func(), error) {
	handle, err := rt.messages.SubscribeRaw(ctx, topic, handler)
	if err != nil {
		return nil, err
	}
	return func() { _ = handle.Close(context.Background()) }, nil
}

func (rt mountedRuntime) ReplyRaw(ctx context.Context, replyTo, correlationID string, payload json.RawMessage, done bool) error {
	return rt.messages.ReplyRaw(ctx, replyTo, correlationID, payload, done)
}

func (rt mountedRuntime) Alive(ctx context.Context) bool {
	return rt.health == nil || rt.health.Alive(ctx)
}

func (rt mountedRuntime) Ready(ctx context.Context) bool {
	return rt.health == nil || rt.health.Ready(ctx)
}

func (rt mountedRuntime) IsDraining() bool {
	return rt.control != nil && rt.control.IsDraining()
}

func (rt mountedRuntime) Close() error { return nil }
