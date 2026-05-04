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

func (gw *Gateway) Mount(ctx context.Context, host bkmodule.Host) error {
	health, _ := bkmodule.Capability[bkmodule.HealthProbes](host, bkmodule.CapabilityHealthProbes)
	control, _ := bkmodule.Capability[bkmodule.RuntimeControl](host, bkmodule.CapabilityRuntimeControl)
	lifecycleDebug, _ := bkmodule.Capability[bkmodule.LifecycleDebugRegistry](host, bkmodule.CapabilityLifecycleDebugRegistry)
	caller, err := bkmodule.RequireCapability[bkmodule.RequestCaller](host, bkmodule.CapabilityRequestCaller)
	if err != nil {
		return fmt.Errorf("gateway: %w", err)
	}
	gw.SetRuntime(mountedRuntime{
		messages: host.Messages(),
		health:   health,
		control:  control,
	})
	gw.setCaller(caller)
	if err := gw.Start(); err != nil {
		return err
	}
	if lifecycleDebug != nil {
		handle, err := lifecycleDebug.RegisterLifecycleDebug(ctx, "gateway", func() any {
			return gw.DebugSnapshot()
		})
		if err != nil {
			_ = gw.Close()
			return fmt.Errorf("gateway: lifecycle debug: %w", err)
		}
		host.Scope().Defer(handle.Close)
	}
	host.Scope().Resource(bkmodule.ResourceWithMetadata(
		bkmodule.ResourceKindHTTP,
		"gateway.listener",
		map[string]string{"listen": gw.config.Listen},
		"HTTP gateway listener.",
	))
	host.Scope().Resource(bkmodule.Resource(bkmodule.ResourceKindHTTP, "gateway.routes", "Runtime HTTP route table."))
	host.Scope().Defer(func(closeCtx context.Context) error {
		if err := gw.StopContext(closeCtx); err != nil {
			return err
		}
		gw.setCaller(nil)
		return nil
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
	handle, err := rt.SubscribeRawHandle(ctx, topic, handler)
	if err != nil {
		return nil, err
	}
	return func() { _ = handle.Close(context.Background()) }, nil
}

func (rt mountedRuntime) SubscribeRawHandle(ctx context.Context, topic string, handler func(sdk.Message)) (bkmodule.Handle, error) {
	return rt.messages.SubscribeRaw(ctx, topic, handler)
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
