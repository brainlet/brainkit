// Ported from: packages/core/src/events/processor.ts
package events

import (
	"github.com/brainlet/brainkit/agent-kit/core/logger"
)

// MastraRef represents the top-level Mastra orchestrator.
// Defined here (not imported from core package) to break circular dependency:
// core imports events (for PubSub, Event types), so events cannot import core.
// core.Mastra struct satisfies this interface.
// The BaseEventProcessor stores this reference so concrete event processors
// (e.g. WorkflowEventProcessor) can access Mastra services like GetLogger,
// GetStorage, etc. via type assertion to their own narrow interfaces.
type MastraRef interface {
	// GetLogger returns the configured logger instance.
	GetLogger() logger.IMastraLogger
}

// IMastraLogger is a type alias to logger.IMastraLogger so that core.Mastra
// satisfies the events.MastraRef interface at compile time.
//
// Ported from: packages/core/src/events — uses mastra.getLogger()
type IMastraLogger = logger.IMastraLogger

// EventProcessor defines the interface for processing events.
// In TypeScript this is an abstract class with a protected mastra field
// and a __registerMastra method. In Go we split this into:
//   - EventProcessor interface (the abstract Process method)
//   - BaseEventProcessor struct (holds the Mastra reference and registration)
type EventProcessor interface {
	// Process handles a single event.
	Process(event Event) error
}

// BaseEventProcessor provides the shared Mastra registration logic
// that TypeScript's abstract EventProcessor class carries. Embed this
// in concrete processor implementations.
type BaseEventProcessor struct {
	Mastra MastraRef
}

// NewBaseEventProcessor creates a BaseEventProcessor with the given Mastra reference.
func NewBaseEventProcessor(mastra MastraRef) BaseEventProcessor {
	return BaseEventProcessor{Mastra: mastra}
}

// RegisterMastra sets (or replaces) the Mastra reference.
// This mirrors the TypeScript __registerMastra method.
func (b *BaseEventProcessor) RegisterMastra(mastra MastraRef) {
	b.Mastra = mastra
}
