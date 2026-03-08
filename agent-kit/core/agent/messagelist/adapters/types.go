// Ported from: packages/core/src/agent/message-list/adapters/types.ts
package adapters

import (
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/state"
)

// Re-export types from state for convenience.
type (
	MastraDBMessage        = state.MastraDBMessage
	MastraMessageV1        = state.MastraMessageV1
	MessageSource          = state.MessageSource
	MemoryInfo             = state.MemoryInfo
	MastraMessageContentV2 = state.MastraMessageContentV2
	UIMessageWithMetadata  = state.UIMessageWithMetadata
	MastraMessagePart      = state.MastraMessagePart
	ProviderMetadata       = state.ProviderMetadata
	ToolInvocation         = state.ToolInvocation
)

// AdapterContext is the common adapter context passed to all adapters.
type AdapterContext struct {
	MemoryInfo      *MemoryInfo
	NewMessageID    func() string
	GenerateCreatedAt func(messageSource MessageSource, start ...any) time.Time
	// DBMessages array for looking up tool call args
	DBMessages []*MastraDBMessage
}

// AIV4AdapterContext is the context for AIV4 adapter operations.
type AIV4AdapterContext = AdapterContext

// AIV5AdapterContext is the context for AIV5 adapter operations.
type AIV5AdapterContext struct {
	MemoryInfo      *MemoryInfo
	NewMessageID    func() string
	GenerateCreatedAt func(messageSource MessageSource, start ...any) time.Time
}
