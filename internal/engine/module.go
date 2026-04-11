package engine

// Module is an optional subsystem that registers commands and manages its own lifecycle.
// Modules receive *Kernel at Init time and use push registration (k.RegisterCommand)
// to add their commands to the catalog.
type Module interface {
	// Name returns the module identifier (e.g., "mcp", "secrets").
	Name() string

	// Init is called after core catalog construction but before the transport router starts.
	// Modules register their commands here via k.RegisterCommand(spec).
	Init(k *Kernel) error

	// Close is called during Kernel shutdown.
	Close() error
}
