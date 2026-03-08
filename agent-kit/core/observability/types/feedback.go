// Ported from: packages/core/src/observability/types/feedback.ts
package types

import "time"

// ============================================================================
// FeedbackInput (User Input)
// ============================================================================

// FeedbackInput holds user-provided feedback data for human evaluation of span/trace quality.
type FeedbackInput struct {
	// Source of the feedback (e.g., "user", "admin", "qa").
	Source string `json:"source"`
	// FeedbackType is the type of feedback (e.g., "thumbs", "rating", "correction").
	FeedbackType string `json:"feedbackType"`
	// Value is the feedback value. Use ValueNumber or ValueString to set.
	Value any `json:"value"`
	// Comment is an optional comment explaining the feedback.
	Comment string `json:"comment,omitempty"`
	// UserID is the user who provided the feedback.
	UserID string `json:"userId,omitempty"`
	// ExperimentID is the experiment identifier for A/B testing or evaluation runs.
	ExperimentID string `json:"experimentId,omitempty"`
	// Metadata is additional metadata specific to this feedback.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ============================================================================
// ExportedFeedback (Event Bus Transport)
// ============================================================================

// ExportedFeedback is feedback data transported via the event bus.
// Must be JSON-serializable.
type ExportedFeedback struct {
	// Timestamp is when the feedback was recorded.
	Timestamp time.Time `json:"timestamp"`
	// TraceID is the trace receiving feedback.
	TraceID string `json:"traceId"`
	// SpanID is the specific span receiving feedback (empty = trace-level feedback).
	SpanID string `json:"spanId,omitempty"`
	// Source of the feedback.
	Source string `json:"source"`
	// FeedbackType is the type of feedback.
	FeedbackType string `json:"feedbackType"`
	// Value is the feedback value.
	Value any `json:"value"`
	// Comment is an optional comment.
	Comment string `json:"comment,omitempty"`
	// ExperimentID is the experiment identifier for A/B testing.
	ExperimentID string `json:"experimentId,omitempty"`
	// Metadata is user-defined metadata.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ============================================================================
// FeedbackEvent (Event Bus Event)
// ============================================================================

// FeedbackEvent is a feedback event emitted to the ObservabilityBus.
type FeedbackEvent struct {
	Type     string           `json:"type"` // always "feedback"
	Feedback ExportedFeedback `json:"feedback"`
}

// NewFeedbackEvent creates a new FeedbackEvent with the type set to "feedback".
func NewFeedbackEvent(feedback ExportedFeedback) FeedbackEvent {
	return FeedbackEvent{
		Type:     "feedback",
		Feedback: feedback,
	}
}
