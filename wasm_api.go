package brainkit

// Public type aliases and constructors for WASM, shards, and persistence.
// Implementations live in internal/wasm.

import (
	_ "embed"

	"github.com/brainlet/brainkit/internal/wasm"
)

//go:embed runtime/wasm_bundle.ts
var wasmBundleSource string

// --- Modules ---

// WASMModule holds a compiled WASM module with metadata.
type WASMModule = wasm.Module

// WASMModuleInfo is the serializable metadata (no binary).
type WASMModuleInfo = wasm.ModuleInfo

// WASMService handles wasm.compile, wasm.run, and module management bus messages.
type WASMService = wasm.Service

// --- Shards ---

// ShardDescriptor describes a deployed shard's registrations.
type ShardDescriptor = wasm.ShardDescriptor

// WASMEventResult is the outcome of a shard handler invocation.
type WASMEventResult = wasm.EventResult

// validateShardDescriptor is a root-package shim for tests.
func validateShardDescriptor(desc *ShardDescriptor, exports []string) error {
	return wasm.ValidateShardDescriptor(desc, exports)
}

// --- Store ---

// KitStore provides optional persistence for WASM modules, shard descriptors, and shard state.
type KitStore = wasm.Store

// SQLiteStore implements KitStore using pure Go SQLite (modernc.org/sqlite).
type SQLiteStore = wasm.SQLiteStore

// NewSQLiteStore creates a new SQLite-backed store at the given file path.
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	return wasm.NewSQLiteStore(path)
}
