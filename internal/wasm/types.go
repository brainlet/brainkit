package wasm

import (
	"context"
	"encoding/json"
	"time"

	"github.com/brainlet/brainkit/bus"
)

// Module holds a compiled WASM module with metadata.
type Module struct {
	Name       string    `json:"name"`
	Binary     []byte    `json:"-"`          // not serialized
	SourceHash string    `json:"sourceHash"` // SHA-256 of source
	Exports    []string  `json:"exports"`    // exported function names
	Size       int       `json:"size"`       // binary size in bytes
	CompiledAt time.Time `json:"compiledAt"`
}

// ModuleInfo is the serializable metadata (no binary).
type ModuleInfo struct {
	Name       string   `json:"name"`
	Size       int      `json:"size"`
	Exports    []string `json:"exports"`
	CompiledAt string   `json:"compiledAt"`
	SourceHash string   `json:"sourceHash"`
}

// ShardDescriptor describes a deployed shard's registrations.
type ShardDescriptor struct {
	Module     string            `json:"module"`
	Mode       string            `json:"mode"`     // "stateless" | "persistent"
	Handlers   map[string]string `json:"handlers"` // topic pattern → exported function name
	DeployedAt time.Time         `json:"deployedAt"`
}

// EventResult is the outcome of a shard handler invocation.
type EventResult struct {
	ExitCode     int    `json:"exitCode"`               // kept for wasm.run compatibility (run() returns i32)
	ReplyPayload string `json:"replyPayload,omitempty"` // captured from reply() host function (shard handlers)
	Error        string `json:"error,omitempty"`
}

// BusBridge is the narrow interface that the WASM service needs from the Kit.
// This decouples the WASM service from the full Kit struct.
type BusBridge interface {
	// Bus returns the message bus.
	Bus() *bus.Bus
	// CallerID returns the identity for bus messages.
	CallerID() string
	// WASMStore returns the optional persistence store (may be nil).
	WASMStore() Store
	// WASMBundleSource returns the embedded wasm_bundle.ts source.
	WASMBundleSource() string
}

// InjectEvent manually triggers a shard handler (for testing and SDK use).
func (s *Service) InjectEvent(ctx context.Context, shardName, topic string, payload json.RawMessage) (*EventResult, error) {
	return s.invokeShardHandler(ctx, shardName, topic, payload)
}
