// Package messaging owns request/reply messaging bus commands as a
// hot-mountable Kit module.
package messaging

import (
	"context"
	"fmt"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/sdk"
)

// Module exposes kit.send. Construct via New and include in
// brainkit.Config.Modules when the runtime should expose the Go-side
// request/reply bridge over the bus.
type Module struct {
	caller bkmodule.RequestCaller
}

// New creates the messaging module.
func New() *Module { return &Module{} }

// ID reports the hot-mount module identifier.
func (m *Module) ID() string { return "messaging" }

// Status reports maturity.
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusStable }

// Mount registers messaging command handlers against the running Kit.
func (m *Module) Mount(_ context.Context, host bkmodule.Host) error {
	caller, err := bkmodule.RequireCapability[bkmodule.RequestCaller](host, bkmodule.CapabilityRequestCaller)
	if err != nil {
		return fmt.Errorf("messaging: %w", err)
	}
	m.caller = caller
	host.Scope().Defer(func(context.Context) error {
		m.caller = nil
		return nil
	})
	if _, err := host.Commands().Handle(bkmodule.Command(m.Send)); err != nil {
		return err
	}
	return nil
}

// Close detaches the module from Kit capabilities. Command handles are owned by
// the module scope, so unmounting unregisters them.
func (m *Module) Close() error {
	m.caller = nil
	return nil
}

// Factory is the registered ModuleFactory for messaging.
type Factory struct{}

// YAML is reserved for future module options.
type YAML struct{}

// Build decodes YAML and returns the messaging module.
func (Factory) Build(ctx bkmodule.BuildContext) (bkmodule.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	return New(), nil
}

// Describe surfaces module metadata for module manifests.
func (Factory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    "messaging",
		Status:  bkmodule.StatusStable,
		Summary: "Request/reply messaging command (kit.send).",
		Commands: []bkmodule.MessageDescriptor{
			bkmodule.CommandMessage[KitSendMsg, KitSendResp](),
		},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.RequiredCapabilityOf[bkmodule.RequestCaller](bkmodule.CapabilityRequestCaller),
		},
	}
}

func init() { bkmodule.Register("messaging", Factory{}) }

// Send handles kit.send by issuing a nested shared-inbox call to the requested
// topic and returning the terminal reply payload.
func (m *Module) Send(ctx context.Context, req KitSendMsg) (*KitSendResp, error) {
	if m.caller == nil {
		return nil, fmt.Errorf("messaging: caller is not configured")
	}
	payload, err := m.caller.Call(ctx, req.Topic, req.Payload, sdk.CallerConfig{})
	if err != nil {
		return nil, err
	}
	return &KitSendResp{Payload: payload}, nil
}
