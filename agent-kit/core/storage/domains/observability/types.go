// Ported from: packages/core/src/storage/domains/observability/types.ts
package observability

import (
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/storage/domains"
)

// ---------------------------------------------------------------------------
// Tracing Storage Strategy
// ---------------------------------------------------------------------------

// TracingStorageStrategy defines how tracing data is persisted to storage.
type TracingStorageStrategy string

const (
	// TracingStrategyRealtime persists spans in real time.
	TracingStrategyRealtime TracingStorageStrategy = "realtime"
	// TracingStrategyBatchWithUpdates batches spans and applies updates.
	TracingStrategyBatchWithUpdates TracingStorageStrategy = "batch-with-updates"
	// TracingStrategyInsertOnly only inserts spans (no updates).
	TracingStrategyInsertOnly TracingStorageStrategy = "insert-only"
)

// TracingStrategyInfo provides hints for tracing strategy selection.
type TracingStrategyInfo struct {
	Preferred TracingStorageStrategy   `json:"preferred"`
	Supported []TracingStorageStrategy `json:"supported"`
}

// ---------------------------------------------------------------------------
// Trace Status
// ---------------------------------------------------------------------------

// TraceStatus is the derived status of a trace from the root span's state.
type TraceStatus string

const (
	TraceStatusSuccess TraceStatus = "success"
	TraceStatusError   TraceStatus = "error"
	TraceStatusRunning TraceStatus = "running"
)

// ---------------------------------------------------------------------------
// Span Type
// ---------------------------------------------------------------------------

// SpanType identifies the type of span (e.g., workflow run, agent run, tool call).
// TODO: Import from the canonical observability/types package once ported.
type SpanType string

// ---------------------------------------------------------------------------
// SpanRecord — complete span record as stored in the database
// ---------------------------------------------------------------------------

// SpanRecord represents a complete span record as stored in the database.
type SpanRecord struct {
	// Required identifiers.
	TraceID string `json:"traceId"`
	SpanID  string `json:"spanId"`
	Name    string `json:"name"`
	SpanTyp SpanType `json:"spanType"`
	IsEvent bool   `json:"isEvent"`
	StartedAt time.Time `json:"startedAt"`

	// Parent span reference (empty string = root span).
	ParentSpanID *string `json:"parentSpanId,omitempty"`

	// Entity identification — first-class fields for filtering.
	EntityType *string `json:"entityType,omitempty"`
	EntityID   *string `json:"entityId,omitempty"`
	EntityName *string `json:"entityName,omitempty"`

	// Identity & tenancy — for multi-tenant applications.
	UserID         *string `json:"userId,omitempty"`
	OrganizationID *string `json:"organizationId,omitempty"`
	ResourceID     *string `json:"resourceId,omitempty"`

	// Correlation IDs — for linking related operations.
	RunID     *string `json:"runId,omitempty"`
	SessionID *string `json:"sessionId,omitempty"`
	ThreadID  *string `json:"threadId,omitempty"`
	RequestID *string `json:"requestId,omitempty"`

	// Deployment context — these fields only exist on the root span.
	Environment *string        `json:"environment,omitempty"`
	Source      *string        `json:"source,omitempty"`
	ServiceName *string        `json:"serviceName,omitempty"`
	Scope       map[string]any `json:"scope,omitempty"`

	// Filterable data — user-defined metadata and tags (tags only on root span).
	Metadata map[string]any `json:"metadata,omitempty"`
	Tags     []string       `json:"tags,omitempty"`

	// Additional span-specific fields.
	Attributes map[string]any `json:"attributes,omitempty"`
	Links      []any          `json:"links,omitempty"`
	Input      any            `json:"input,omitempty"`
	Output     any            `json:"output,omitempty"`
	Error      any            `json:"error,omitempty"`
	EndedAt    *time.Time     `json:"endedAt,omitempty"`

	// Database timestamps.
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ComputeTraceStatus computes the trace status from a root span's error
// and endedAt fields.
//   - ERROR: if error is present (regardless of endedAt)
//   - RUNNING: if endedAt is nil and no error
//   - SUCCESS: if endedAt is present and no error
func ComputeTraceStatus(span *SpanRecord) TraceStatus {
	if span.Error != nil {
		return TraceStatusError
	}
	if span.EndedAt == nil {
		return TraceStatusRunning
	}
	return TraceStatusSuccess
}

// ---------------------------------------------------------------------------
// TraceSpan — root span with computed status
// ---------------------------------------------------------------------------

// TraceSpan is a SpanRecord extended with a computed Status field.
// Used for root spans returned from ListTraces.
type TraceSpan struct {
	SpanRecord
	Status TraceStatus `json:"status"`
}

// ToTraceSpan converts a SpanRecord to a TraceSpan by adding computed status.
func ToTraceSpan(span *SpanRecord) TraceSpan {
	return TraceSpan{
		SpanRecord: *span,
		Status:     ComputeTraceStatus(span),
	}
}

// ToTraceSpans converts a slice of SpanRecords to TraceSpans.
func ToTraceSpans(spans []SpanRecord) []TraceSpan {
	result := make([]TraceSpan, len(spans))
	for i := range spans {
		result[i] = ToTraceSpan(&spans[i])
	}
	return result
}

// ---------------------------------------------------------------------------
// CreateSpanRecord — span record without db timestamps (for creation)
// ---------------------------------------------------------------------------

// CreateSpanRecord is a SpanRecord without database timestamps, used for creation.
type CreateSpanRecord struct {
	TraceID string   `json:"traceId"`
	SpanID  string   `json:"spanId"`
	Name    string   `json:"name"`
	SpanTyp SpanType `json:"spanType"`
	IsEvent bool     `json:"isEvent"`
	StartedAt time.Time `json:"startedAt"`

	ParentSpanID *string `json:"parentSpanId,omitempty"`

	EntityType *string `json:"entityType,omitempty"`
	EntityID   *string `json:"entityId,omitempty"`
	EntityName *string `json:"entityName,omitempty"`

	UserID         *string `json:"userId,omitempty"`
	OrganizationID *string `json:"organizationId,omitempty"`
	ResourceID     *string `json:"resourceId,omitempty"`

	RunID     *string `json:"runId,omitempty"`
	SessionID *string `json:"sessionId,omitempty"`
	ThreadID  *string `json:"threadId,omitempty"`
	RequestID *string `json:"requestId,omitempty"`

	Environment *string        `json:"environment,omitempty"`
	Source      *string        `json:"source,omitempty"`
	ServiceName *string        `json:"serviceName,omitempty"`
	Scope       map[string]any `json:"scope,omitempty"`

	Metadata map[string]any `json:"metadata,omitempty"`
	Tags     []string       `json:"tags,omitempty"`

	Attributes map[string]any `json:"attributes,omitempty"`
	Links      []any          `json:"links,omitempty"`
	Input      any            `json:"input,omitempty"`
	Output     any            `json:"output,omitempty"`
	Error      any            `json:"error,omitempty"`
	EndedAt    *time.Time     `json:"endedAt,omitempty"`
}

// ---------------------------------------------------------------------------
// UpdateSpanRecord — partial span data for updates (no timestamps, no IDs)
// ---------------------------------------------------------------------------

// UpdateSpanRecord holds partial span data for updates.
// All fields are optional (pointer types).
type UpdateSpanRecord struct {
	Name    *string   `json:"name,omitempty"`
	SpanTyp *SpanType `json:"spanType,omitempty"`
	IsEvent *bool     `json:"isEvent,omitempty"`
	StartedAt *time.Time `json:"startedAt,omitempty"`

	ParentSpanID *string `json:"parentSpanId,omitempty"`

	EntityType *string `json:"entityType,omitempty"`
	EntityID   *string `json:"entityId,omitempty"`
	EntityName *string `json:"entityName,omitempty"`

	UserID         *string `json:"userId,omitempty"`
	OrganizationID *string `json:"organizationId,omitempty"`
	ResourceID     *string `json:"resourceId,omitempty"`

	RunID     *string `json:"runId,omitempty"`
	SessionID *string `json:"sessionId,omitempty"`
	ThreadID  *string `json:"threadId,omitempty"`
	RequestID *string `json:"requestId,omitempty"`

	Environment *string        `json:"environment,omitempty"`
	Source      *string        `json:"source,omitempty"`
	ServiceName *string        `json:"serviceName,omitempty"`
	Scope       map[string]any `json:"scope,omitempty"`

	Metadata map[string]any `json:"metadata,omitempty"`
	Tags     []string       `json:"tags,omitempty"`

	Attributes map[string]any `json:"attributes,omitempty"`
	Links      []any          `json:"links,omitempty"`
	Input      any            `json:"input,omitempty"`
	Output     any            `json:"output,omitempty"`
	Error      any            `json:"error,omitempty"`
	EndedAt    *time.Time     `json:"endedAt,omitempty"`
}

// ---------------------------------------------------------------------------
// Storage Operation Args & Responses
// ---------------------------------------------------------------------------

// CreateSpanArgs holds the arguments for creating a single span.
type CreateSpanArgs struct {
	Span CreateSpanRecord `json:"span"`
}

// BatchCreateSpansArgs holds the arguments for batch creating spans.
type BatchCreateSpansArgs struct {
	Records []CreateSpanRecord `json:"records"`
}

// GetSpanArgs holds the arguments for getting a single span.
type GetSpanArgs struct {
	TraceID string `json:"traceId"`
	SpanID  string `json:"spanId"`
}

// GetSpanResponse is the response containing a single span.
type GetSpanResponse struct {
	Span SpanRecord `json:"span"`
}

// GetRootSpanArgs holds the arguments for getting a root span.
type GetRootSpanArgs struct {
	TraceID string `json:"traceId"`
}

// GetRootSpanResponse is the response containing a single root span.
type GetRootSpanResponse struct {
	Span SpanRecord `json:"span"`
}

// GetTraceArgs holds the arguments for getting a single trace.
type GetTraceArgs struct {
	TraceID string `json:"traceId"`
}

// GetTraceResponse is the response containing a trace with all its spans.
type GetTraceResponse struct {
	TraceID string       `json:"traceId"`
	Spans   []SpanRecord `json:"spans"`
}

// TraceRecord is an alias for GetTraceResponse.
type TraceRecord = GetTraceResponse

// ---------------------------------------------------------------------------
// Trace Filter & List Types
// ---------------------------------------------------------------------------

// TracesFilter holds filters for querying traces.
type TracesFilter struct {
	// Date range filters.
	StartedAt *domains.DateRange `json:"startedAt,omitempty"`
	EndedAt   *domains.DateRange `json:"endedAt,omitempty"`

	// Span type filter.
	SpanTyp *SpanType `json:"spanType,omitempty"`

	// Shared fields (matched against root span).
	EntityType *string `json:"entityType,omitempty"`
	EntityID   *string `json:"entityId,omitempty"`
	EntityName *string `json:"entityName,omitempty"`

	UserID         *string `json:"userId,omitempty"`
	OrganizationID *string `json:"organizationId,omitempty"`
	ResourceID     *string `json:"resourceId,omitempty"`

	RunID     *string `json:"runId,omitempty"`
	SessionID *string `json:"sessionId,omitempty"`
	ThreadID  *string `json:"threadId,omitempty"`
	RequestID *string `json:"requestId,omitempty"`

	Environment *string        `json:"environment,omitempty"`
	Source      *string        `json:"source,omitempty"`
	ServiceName *string        `json:"serviceName,omitempty"`
	Scope       map[string]any `json:"scope,omitempty"`

	Metadata map[string]any `json:"metadata,omitempty"`
	Tags     []string       `json:"tags,omitempty"`

	// Filter-specific derived status fields.
	Status        *TraceStatus `json:"status,omitempty"`
	HasChildError *bool        `json:"hasChildError,omitempty"`
}

// TracesOrderByField defines the fields available for ordering trace results.
type TracesOrderByField string

const (
	TracesOrderByStartedAt TracesOrderByField = "startedAt"
	TracesOrderByEndedAt   TracesOrderByField = "endedAt"
)

// TracesOrderBy is the order-by configuration for trace queries.
type TracesOrderBy struct {
	Field     TracesOrderByField    `json:"field"`
	Direction domains.SortDirection `json:"direction"`
}

// ListTracesArgs holds the arguments for listing traces.
type ListTracesArgs struct {
	Filters    *TracesFilter        `json:"filters,omitempty"`
	Pagination *domains.PaginationArgs `json:"pagination,omitempty"`
	OrderBy    *TracesOrderBy       `json:"orderBy,omitempty"`
}

// ListTracesResponse contains paginated root spans with computed status.
type ListTracesResponse struct {
	Pagination domains.PaginationInfo `json:"pagination"`
	Spans      []TraceSpan            `json:"spans"`
}

// UpdateSpanArgs holds the arguments for updating a single span.
type UpdateSpanArgs struct {
	SpanID  string           `json:"spanId"`
	TraceID string           `json:"traceId"`
	Updates UpdateSpanRecord `json:"updates"`
}

// BatchUpdateSpansArgs holds the arguments for batch updating spans.
type BatchUpdateSpansArgs struct {
	Records []struct {
		TraceID string           `json:"traceId"`
		SpanID  string           `json:"spanId"`
		Updates UpdateSpanRecord `json:"updates"`
	} `json:"records"`
}

// BatchDeleteTracesArgs holds the arguments for batch deleting traces.
type BatchDeleteTracesArgs struct {
	TraceIDs []string `json:"traceIds"`
}
