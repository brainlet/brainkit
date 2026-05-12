// Package eval owns the kit.eval bus command as a hot-mountable Kit module.
package eval

import (
	"context"
	"fmt"
	"strings"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modulecap/runtime"
	"github.com/brainlet/brainkit/modules/eval/evalmsg"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/google/uuid"
)

// Module exposes kit.eval. Construct via New and include in
// brainkit.Config.Modules when the runtime should expose JavaScript eval over the
// bus.
type Module struct {
	runtime runtimecap.EvalRuntime
}

// New creates the eval module.
func New() *Module { return &Module{} }

// ID reports the hot-mount module identifier.
func (m *Module) ID() string { return "eval" }

// Status reports maturity.
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusBeta }

// Mount registers kit.eval against the running Kit.
func (m *Module) Mount(_ context.Context, host bkmodule.Host) error {
	runtime, err := bkmodule.RequireCapability[runtimecap.EvalRuntime](host, bkmodule.CapabilityEvalRuntime)
	if err != nil {
		return fmt.Errorf("eval: %w", err)
	}
	m.runtime = runtime
	host.Scope().Defer(func(context.Context) error {
		m.runtime = nil
		return nil
	})
	if _, err := host.Commands().Handle(bkmodule.Command(m.Eval)); err != nil {
		return err
	}
	return nil
}

// Close detaches the module from Kit capabilities.
func (m *Module) Close() error {
	m.runtime = nil
	return nil
}

// Factory is the registered ModuleFactory for eval.
type Factory struct{}

// YAML is reserved for future module options.
type YAML struct{}

// Build decodes YAML and returns the eval module.
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
		Name:    "eval",
		Status:  bkmodule.StatusBeta,
		Summary: "JavaScript eval bus command (kit.eval).",
		Requires: []string{
			"jsruntime",
		},
		Commands: []bkmodule.MessageDescriptor{
			bkmodule.CommandMessage[evalmsg.KitEvalMsg, evalmsg.KitEvalResp](),
		},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.RequiredCapabilityOf[runtimecap.EvalRuntime](bkmodule.CapabilityEvalRuntime),
		},
	}
}

func init() { bkmodule.Register("eval", Factory{}) }

// Eval handles kit.eval. Mode dispatch is:
// script deploys a temporary module and reads globalThis.__module_result; js
// evaluates JavaScript in the current runtime context; module evaluates as an ES module.
func (m *Module) Eval(ctx context.Context, req evalmsg.KitEvalMsg) (*evalmsg.KitEvalResp, error) {
	if m.runtime == nil {
		return nil, fmt.Errorf("eval: runtime is not configured")
	}
	mode := req.Mode
	if mode == "" {
		if strings.HasSuffix(req.Source, ".js") {
			mode = "js"
		} else {
			mode = "script"
		}
	}
	switch mode {
	case "js":
		source := req.Source
		if source == "" {
			source = "__eval_js.js"
		}
		result, err := m.runtime.EvalJS(ctx, source, req.Code)
		if err != nil {
			return nil, err
		}
		return &evalmsg.KitEvalResp{Result: result}, nil
	case "module":
		source := req.Source
		if source == "" {
			source = "__eval_module.js"
		}
		result, err := m.runtime.EvalModule(ctx, source, req.Code)
		if err != nil {
			return nil, err
		}
		return &evalmsg.KitEvalResp{Result: result}, nil
	case "script":
		source := req.Source
		if source == "" {
			source = "__cli_eval_" + uuid.NewString() + ".js"
		}
		result, err := m.runtime.EvalScript(ctx, source, req.Code)
		if err != nil {
			return nil, err
		}
		return &evalmsg.KitEvalResp{Result: result}, nil
	default:
		return nil, &sdkerrors.ValidationError{Field: "mode", Message: "unknown eval mode: " + mode + " (want script|js|module)"}
	}
}
