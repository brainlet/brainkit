// Package workflow exposes brainkit's Mastra-style workflow commands
// (start, startAsync, status, resume, cancel, list, runs, restart) as a
// Kit-scoped Module. Construct via New and include in
// brainkit.Config.Modules.
package workflow

import (
	"context"
	"encoding/json"
	"fmt"

	bkmodule "github.com/brainlet/brainkit/module"
	_ "github.com/brainlet/brainkit/modules/jsruntime"
	"github.com/brainlet/brainkit/modules/workflow/workflowmsg"
)

// Module wraps the workflow bus commands.
type Module struct {
	callJS func(context.Context, string, any) (json.RawMessage, error)
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
		Name:    "workflow",
		Status:  bkmodule.StatusStable,
		Summary: "Mastra-style workflow bus commands (start, status, resume, …).",
		Requires: []string{
			"jsruntime",
		},
		Commands: []bkmodule.MessageDescriptor{
			bkmodule.CommandMessage[workflowmsg.WorkflowCancelMsg, workflowmsg.WorkflowCancelResp](),
			bkmodule.CommandMessage[workflowmsg.WorkflowListMsg, workflowmsg.WorkflowListResp](),
			bkmodule.CommandMessage[workflowmsg.WorkflowRestartMsg, workflowmsg.WorkflowRestartResp](),
			bkmodule.CommandMessage[workflowmsg.WorkflowResumeMsg, workflowmsg.WorkflowResumeResp](),
			bkmodule.CommandMessage[workflowmsg.WorkflowRunsMsg, workflowmsg.WorkflowRunsResp](),
			bkmodule.CommandMessage[workflowmsg.WorkflowStartAsyncMsg, workflowmsg.WorkflowStartAsyncResp](),
			bkmodule.CommandMessage[workflowmsg.WorkflowStartMsg, workflowmsg.WorkflowStartResp](),
			bkmodule.CommandMessage[workflowmsg.WorkflowStatusMsg, workflowmsg.WorkflowStatusResp](),
		},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.RequiredCapabilityOf[func(context.Context, string, any) (json.RawMessage, error)](bkmodule.CapabilityCallJS),
		},
	}
}

func init() { bkmodule.Register("workflow", Factory{}) }

// ID reports the hot-mount module identifier.
func (m *Module) ID() string { return "workflow" }

// Dependencies reports modules that must mount before workflow commands.
func (m *Module) Dependencies() []string { return []string{"jsruntime"} }

// Status reports maturity.
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusStable }

func (m *Module) Mount(_ context.Context, host bkmodule.Host) error {
	callJS, err := bkmodule.RequireCapability[func(context.Context, string, any) (json.RawMessage, error)](host, bkmodule.CapabilityCallJS)
	if err != nil {
		return fmt.Errorf("workflow: %w", err)
	}
	host.Scope().Defer(func(context.Context) error {
		m.callJS = nil
		return nil
	})
	m.callJS = callJS
	for _, spec := range []bkmodule.CommandSpec{
		bkmodule.Command(m.handleStart),
		bkmodule.Command(m.handleStartAsync),
		bkmodule.Command(m.handleStatus),
		bkmodule.Command(m.handleResume),
		bkmodule.Command(m.handleCancel),
		bkmodule.Command(m.handleList),
		bkmodule.Command(m.handleRuns),
		bkmodule.Command(m.handleRestart),
	} {
		if _, err := host.Commands().Handle(spec); err != nil {
			return err
		}
	}
	return nil
}

// Close is a no-op; workflow state lives in the Mastra runtime.
func (m *Module) Close() error {
	m.callJS = nil
	return nil
}

func (m *Module) call(ctx context.Context, fn string, args any) (json.RawMessage, error) {
	if m.callJS == nil {
		return nil, fmt.Errorf("workflow: module is not mounted")
	}
	return m.callJS(ctx, fn, args)
}
