// Package workflow exposes brainkit's Mastra-style workflow commands
// (start, startAsync, status, resume, cancel, list, runs, restart) as a
// Kit-scoped Module. Construct via New and include in
// brainkit.Config.Modules.
package workflow

import (
	"github.com/brainlet/brainkit"
)

// Module wraps the workflow bus commands.
type Module struct {
	kit *brainkit.Kit
}

// New creates a workflow module. It has no configuration today.
func New() *Module { return &Module{} }

// Factory is the registered ModuleFactory for workflow.
type Factory struct{}

// YAML is the config shape decoded by the registry factory. The
// module has no options today, but a named type means future fields
// can land without breaking existing configs.
type YAML struct{}

// Build returns a fresh workflow module. A non-nil decode error
// propagates so typos like `workflow: true` (scalar instead of map)
// surface at startup instead of being swallowed.
func (Factory) Build(ctx brainkit.ModuleContext) (brainkit.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	return New(), nil
}

// Describe surfaces module metadata for `brainkit modules list`.
func (Factory) Describe() brainkit.ModuleDescriptor {
	return brainkit.ModuleDescriptor{
		Name:    "workflow",
		Status:  brainkit.ModuleStatusStable,
		Summary: "Mastra-style workflow bus commands (start, status, resume, …).",
	}
}

func init() { brainkit.RegisterModule("workflow", Factory{}) }

// Name reports the module identifier.
func (m *Module) Name() string { return "workflow" }

// Status reports maturity.
func (m *Module) Status() brainkit.ModuleStatus { return brainkit.ModuleStatusStable }

// Init registers the workflow bus commands.
func (m *Module) Init(k *brainkit.Kit) error {
	m.kit = k
	k.RegisterCommand(brainkit.Command(m.handleStart))
	k.RegisterCommand(brainkit.Command(m.handleStartAsync))
	k.RegisterCommand(brainkit.Command(m.handleStatus))
	k.RegisterCommand(brainkit.Command(m.handleResume))
	k.RegisterCommand(brainkit.Command(m.handleCancel))
	k.RegisterCommand(brainkit.Command(m.handleList))
	k.RegisterCommand(brainkit.Command(m.handleRuns))
	k.RegisterCommand(brainkit.Command(m.handleRestart))
	return nil
}

// Close is a no-op; workflow state lives in the Mastra runtime.
func (m *Module) Close() error { return nil }
