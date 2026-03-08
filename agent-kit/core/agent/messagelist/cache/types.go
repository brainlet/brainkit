// Ported from: packages/core/src/agent/message-list/cache/types.ts
package cache

import (
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/state"
)

// Re-export types from state for local use.
type (
	MastraMessagePart      = state.MastraMessagePart
	UIMessageV4Part        = state.UIMessageV4Part
	MastraMessageContentV2 = state.MastraMessageContentV2
	ProviderMetadata       = state.ProviderMetadata
)
