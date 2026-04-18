package audit

import (
	auditpkg "github.com/brainlet/brainkit/internal/audit"
)

// Store re-exports the core auditpkg.Store interface so callers of
// audit.NewModule can name the argument type without importing
// internal/audit directly.
type Store = auditpkg.Store

// Event is re-exported for module-facing store implementations.
type Event = auditpkg.Event

// Query is re-exported for module-facing store implementations.
type Query = auditpkg.Query

// Verbosity is re-exported for ergonomic Config construction.
type Verbosity = auditpkg.Verbosity

const (
	VerbosityNormal  = auditpkg.VerbosityNormal
	VerbosityVerbose = auditpkg.VerbosityVerbose
)

// Config configures the audit module.
type Config struct {
	// Store is the backing audit event store. Without a store the Recorder
	// in core is a no-op; the bus handlers return empty results.
	Store Store

	// Verbose flips the Recorder's verbosity: when true, bus.command.completed
	// and periodic metrics snapshots are also recorded (high volume).
	Verbose bool

	// OwnStore, when true, asks the module to Close the store on its own
	// Close. Set false if the caller shares the store with other subsystems
	// and owns its lifecycle.
	OwnStore bool
}
