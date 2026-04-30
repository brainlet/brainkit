// Package eval owns the kit.eval bus command as a hot-mountable Kit module.
package eval

import (
	"context"
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/eval/evalmsg"
	_ "github.com/brainlet/brainkit/modules/jsruntime"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/google/uuid"
)

// Module exposes kit.eval. Construct via New and include in
// brainkit.Config.Modules when the runtime should expose JS/TS eval over the
// bus.
type Module struct {
	runtime runtime
}

type runtime interface {
	Deploy(context.Context, string, string, ...types.DeployOption) ([]types.ResourceInfo, error)
	Teardown(context.Context, string) (int, error)
	EvalTS(context.Context, string, string) (string, error)
	EvalModule(context.Context, string, string) (string, error)
}

// New creates the eval module.
func New() *Module { return &Module{} }

// ID reports the hot-mount module identifier.
func (m *Module) ID() string { return "eval" }

// Dependencies reports modules that must mount before eval.
func (m *Module) Dependencies() []string { return []string{"jsruntime"} }

// Status reports maturity.
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusBeta }

// Mount registers kit.eval against the running Kit.
func (m *Module) Mount(_ context.Context, host bkmodule.Host) error {
	runtime, err := bkmodule.RequireCapability[runtime](host, bkmodule.CapabilityEvalRuntime)
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

// Describe surfaces module metadata for `brainkit modules list`.
func (Factory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    "eval",
		Status:  bkmodule.StatusBeta,
		Summary: "JS/TS eval bus command (kit.eval).",
		Requires: []string{
			"jsruntime",
		},
		Commands: []bkmodule.MessageDescriptor{
			bkmodule.CommandMessage[evalmsg.KitEvalMsg, evalmsg.KitEvalResp](),
		},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.RequiredCapabilityOf[runtime](bkmodule.CapabilityEvalRuntime),
		},
	}
}

func init() { bkmodule.Register("eval", Factory{}) }

// Eval handles kit.eval. Mode dispatch preserves the legacy behavior:
// script deploys a temporary module and reads globalThis.__module_result; ts
// evaluates in the current runtime context; module evaluates as an ES module.
func (m *Module) Eval(ctx context.Context, req evalmsg.KitEvalMsg) (*evalmsg.KitEvalResp, error) {
	if m.runtime == nil {
		return nil, fmt.Errorf("eval: runtime is not configured")
	}
	mode := req.Mode
	if mode == "" {
		if strings.HasSuffix(req.Source, ".ts") {
			mode = "ts"
		} else {
			mode = "script"
		}
	}
	switch mode {
	case "ts":
		source := req.Source
		if source == "" {
			source = "__eval_ts.ts"
		}
		result, err := m.runtime.EvalTS(ctx, source, req.Code)
		if err != nil {
			return nil, err
		}
		return &evalmsg.KitEvalResp{Result: result}, nil
	case "module":
		source := req.Source
		if source == "" {
			source = "__eval_module.ts"
		}
		result, err := m.runtime.EvalModule(ctx, source, req.Code)
		if err != nil {
			return nil, err
		}
		return &evalmsg.KitEvalResp{Result: result}, nil
	case "script":
		source := "__cli_eval_" + uuid.NewString() + ".ts"
		if _, err := m.runtime.Deploy(ctx, source, req.Code); err != nil {
			return nil, err
		}
		defer m.runtime.Teardown(ctx, source)
		result, _ := m.runtime.EvalTS(ctx, "__read_eval.ts", `return globalThis.__module_result || "null";`)
		return &evalmsg.KitEvalResp{Result: result}, nil
	default:
		return nil, &sdkerrors.ValidationError{Field: "mode", Message: "unknown eval mode: " + mode + " (want script|ts|module)"}
	}
}
