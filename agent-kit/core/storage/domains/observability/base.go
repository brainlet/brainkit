// Ported from: packages/core/src/storage/domains/observability/base.ts
package observability

import "context"

// ---------------------------------------------------------------------------
// ObservabilityStorage Interface
// ---------------------------------------------------------------------------

// ObservabilityStorage is the storage interface for the observability domain.
// Unlike most storage domains, this is NOT abstract in TS — it provides default
// implementations that throw errors. Adapters override only the methods they support.
//
// In Go, we model this as an interface. Adapters that do not support a method
// should return an appropriate error.
type ObservabilityStorage interface {
	// Init initializes the storage domain (creates tables, etc).
	Init(ctx context.Context) error

	// DangerouslyClearAll clears all data. Primarily used for testing.
	DangerouslyClearAll(ctx context.Context) error

	// TracingStrategy provides hints for tracing strategy selection by the
	// DefaultExporter. Returns the preferred strategy and all supported strategies.
	TracingStrategy() TracingStrategyInfo

	// --- Span Operations ---

	// CreateSpan creates a single Span record in the storage provider.
	CreateSpan(ctx context.Context, args CreateSpanArgs) error

	// UpdateSpan updates a single Span with partial data.
	// Primarily used for realtime trace creation.
	UpdateSpan(ctx context.Context, args UpdateSpanArgs) error

	// GetSpan retrieves a single span.
	GetSpan(ctx context.Context, args GetSpanArgs) (*GetSpanResponse, error)

	// GetRootSpan retrieves a single root span.
	GetRootSpan(ctx context.Context, args GetRootSpanArgs) (*GetRootSpanResponse, error)

	// --- Trace Operations ---

	// GetTrace retrieves a single trace with all its associated spans.
	GetTrace(ctx context.Context, args GetTraceArgs) (*GetTraceResponse, error)

	// ListTraces retrieves a list of traces with optional filtering.
	ListTraces(ctx context.Context, args ListTracesArgs) (*ListTracesResponse, error)

	// --- Batch Operations ---

	// BatchCreateSpans creates multiple Spans in a single batch.
	BatchCreateSpans(ctx context.Context, args BatchCreateSpansArgs) error

	// BatchUpdateSpans updates multiple Spans in a single batch.
	BatchUpdateSpans(ctx context.Context, args BatchUpdateSpansArgs) error

	// BatchDeleteTraces deletes multiple traces and all their associated spans.
	BatchDeleteTraces(ctx context.Context, args BatchDeleteTracesArgs) error
}
