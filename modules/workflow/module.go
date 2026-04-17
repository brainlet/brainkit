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

// New creates a workflow module. It has no configuration today; the
// constructor is kept for forward-compatibility.
func New() *Module { return &Module{} }

// Name reports the module identifier.
func (m *Module) Name() string { return "workflow" }

// Status reports maturity.
func (m *Module) Status() brainkit.ModuleStatus { return brainkit.ModuleStatusBeta }

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
