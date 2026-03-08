// Ported from: packages/core/src/storage/domains/observability/inmemory.ts
package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/storage/domains"
)

// Compile-time interface check.
var _ ObservabilityStorage = (*InMemoryObservabilityStorage)(nil)

// traceEntry is the internal structure for storing a trace with computed properties.
type traceEntry struct {
	// All spans in this trace, keyed by spanId.
	Spans map[string]SpanRecord
	// Root span for this trace (parentSpanId == nil).
	RootSpan *SpanRecord
	// Computed trace status based on root span state.
	Status TraceStatus
	// True if any span in the trace has an error.
	HasChildError bool
}

// InMemoryObservabilityStorage is an in-memory implementation of ObservabilityStorage.
type InMemoryObservabilityStorage struct {
	mu     sync.RWMutex
	traces map[string]*traceEntry // keyed by traceId
}

// NewInMemoryObservabilityStorage creates a new InMemoryObservabilityStorage.
func NewInMemoryObservabilityStorage() *InMemoryObservabilityStorage {
	return &InMemoryObservabilityStorage{
		traces: make(map[string]*traceEntry),
	}
}

// Init is a no-op for the in-memory store.
func (s *InMemoryObservabilityStorage) Init(_ context.Context) error {
	return nil
}

// DangerouslyClearAll clears all traces.
func (s *InMemoryObservabilityStorage) DangerouslyClearAll(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.traces = make(map[string]*traceEntry)
	return nil
}

// TracingStrategy returns the preferred and supported strategies.
func (s *InMemoryObservabilityStorage) TracingStrategy() TracingStrategyInfo {
	return TracingStrategyInfo{
		Preferred: TracingStrategyRealtime,
		Supported: []TracingStorageStrategy{
			TracingStrategyRealtime,
			TracingStrategyBatchWithUpdates,
			TracingStrategyInsertOnly,
		},
	}
}

// CreateSpan creates a single span.
func (s *InMemoryObservabilityStorage) CreateSpan(_ context.Context, args CreateSpanArgs) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	span := args.Span
	if span.SpanID == "" {
		return fmt.Errorf("span ID is required for creating a span")
	}
	if span.TraceID == "" {
		return fmt.Errorf("trace ID is required for creating a span")
	}

	now := time.Now()
	record := createSpanToRecord(span, now)
	s.upsertSpanToTrace(record)
	return nil
}

// BatchCreateSpans creates multiple spans.
func (s *InMemoryObservabilityStorage) BatchCreateSpans(_ context.Context, args BatchCreateSpansArgs) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for _, span := range args.Records {
		if span.SpanID == "" {
			return fmt.Errorf("span ID is required for creating a span")
		}
		if span.TraceID == "" {
			return fmt.Errorf("trace ID is required for creating a span")
		}
		record := createSpanToRecord(span, now)
		s.upsertSpanToTrace(record)
	}
	return nil
}

// GetSpan retrieves a span by trace ID and span ID.
func (s *InMemoryObservabilityStorage) GetSpan(_ context.Context, args GetSpanArgs) (*GetSpanResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	te, ok := s.traces[args.TraceID]
	if !ok {
		return nil, nil
	}

	span, ok := te.Spans[args.SpanID]
	if !ok {
		return nil, nil
	}

	return &GetSpanResponse{Span: span}, nil
}

// GetRootSpan retrieves the root span for a trace.
func (s *InMemoryObservabilityStorage) GetRootSpan(_ context.Context, args GetRootSpanArgs) (*GetRootSpanResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	te, ok := s.traces[args.TraceID]
	if !ok || te.RootSpan == nil {
		return nil, nil
	}

	return &GetRootSpanResponse{Span: *te.RootSpan}, nil
}

// GetTrace retrieves all spans for a trace.
func (s *InMemoryObservabilityStorage) GetTrace(_ context.Context, args GetTraceArgs) (*GetTraceResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	te, ok := s.traces[args.TraceID]
	if !ok {
		return nil, nil
	}

	spans := make([]SpanRecord, 0, len(te.Spans))
	for _, span := range te.Spans {
		spans = append(spans, span)
	}

	if len(spans) == 0 {
		return nil, nil
	}

	// Sort spans by startedAt.
	sort.Slice(spans, func(i, j int) bool {
		return spans[i].StartedAt.Before(spans[j].StartedAt)
	})

	return &GetTraceResponse{
		TraceID: args.TraceID,
		Spans:   spans,
	}, nil
}

// ListTraces lists traces with filtering, pagination, and sorting.
func (s *InMemoryObservabilityStorage) ListTraces(_ context.Context, args ListTracesArgs) (*ListTracesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Apply defaults.
	pagination := args.Pagination
	if pagination == nil {
		pagination = &domains.PaginationArgs{Page: 0, PerPage: 10}
	}
	orderBy := args.OrderBy
	if orderBy == nil {
		orderBy = &TracesOrderBy{
			Field:     TracesOrderByStartedAt,
			Direction: domains.SortDESC,
		}
	}

	// Collect matching root spans.
	var matchingRootSpans []SpanRecord
	for _, te := range s.traces {
		if te.RootSpan == nil {
			continue
		}
		if s.traceMatchesFilters(te, args.Filters) {
			matchingRootSpans = append(matchingRootSpans, *te.RootSpan)
		}
	}

	// Sort by orderBy field.
	sortField := orderBy.Field
	sortDir := orderBy.Direction

	sort.Slice(matchingRootSpans, func(i, j int) bool {
		if sortField == TracesOrderByEndedAt {
			aEnd := matchingRootSpans[i].EndedAt
			bEnd := matchingRootSpans[j].EndedAt

			// Handle nil (running spans).
			if aEnd == nil && bEnd == nil {
				return false
			}
			if aEnd == nil {
				return sortDir == domains.SortDESC
			}
			if bEnd == nil {
				return sortDir == domains.SortASC
			}

			if sortDir == domains.SortDESC {
				return bEnd.Before(*aEnd)
			}
			return aEnd.Before(*bEnd)
		}

		// Default: startedAt (never nil).
		if sortDir == domains.SortDESC {
			return matchingRootSpans[j].StartedAt.Before(matchingRootSpans[i].StartedAt)
		}
		return matchingRootSpans[i].StartedAt.Before(matchingRootSpans[j].StartedAt)
	})

	// Apply pagination.
	total := len(matchingRootSpans)
	page := pagination.Page
	perPage := pagination.PerPage
	if perPage <= 0 {
		perPage = 10
	}
	start := page * perPage
	end := start + perPage
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	paged := matchingRootSpans[start:end]

	return &ListTracesResponse{
		Spans: ToTraceSpans(paged),
		Pagination: domains.PaginationInfo{
			Total:   total,
			Page:    page,
			PerPage: perPage,
			HasMore: end < total,
		},
	}, nil
}

// UpdateSpan updates a span.
func (s *InMemoryObservabilityStorage) UpdateSpan(_ context.Context, args UpdateSpanArgs) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	te, ok := s.traces[args.TraceID]
	if !ok {
		return fmt.Errorf("trace not found for span update")
	}

	span, ok := te.Spans[args.SpanID]
	if !ok {
		return fmt.Errorf("span not found for update")
	}

	// Apply partial updates.
	updated := applySpanUpdates(span, args.Updates)
	updated.UpdatedAt = time.Now()

	te.Spans[args.SpanID] = updated

	// Update root span reference if this is the root span.
	if updated.ParentSpanID == nil {
		te.RootSpan = &updated
	}

	s.recomputeTraceProperties(te)
	return nil
}

// BatchUpdateSpans updates multiple spans.
func (s *InMemoryObservabilityStorage) BatchUpdateSpans(_ context.Context, args BatchUpdateSpansArgs) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, record := range args.Records {
		te, ok := s.traces[record.TraceID]
		if !ok {
			return fmt.Errorf("trace not found for span update")
		}

		span, ok := te.Spans[record.SpanID]
		if !ok {
			return fmt.Errorf("span not found for update")
		}

		updated := applySpanUpdates(span, record.Updates)
		updated.UpdatedAt = time.Now()

		te.Spans[record.SpanID] = updated

		if updated.ParentSpanID == nil {
			te.RootSpan = &updated
		}

		s.recomputeTraceProperties(te)
	}
	return nil
}

// BatchDeleteTraces deletes multiple traces.
func (s *InMemoryObservabilityStorage) BatchDeleteTraces(_ context.Context, args BatchDeleteTracesArgs) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, traceID := range args.TraceIDs {
		delete(s.traces, traceID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// upsertSpanToTrace inserts or updates a span and recomputes trace properties.
// Must be called with write lock held.
func (s *InMemoryObservabilityStorage) upsertSpanToTrace(span SpanRecord) {
	te, ok := s.traces[span.TraceID]
	if !ok {
		te = &traceEntry{
			Spans:         make(map[string]SpanRecord),
			RootSpan:      nil,
			Status:        TraceStatusRunning,
			HasChildError: false,
		}
		s.traces[span.TraceID] = te
	}

	te.Spans[span.SpanID] = span

	// Update root span if this is a root span (nil parentSpanId).
	if span.ParentSpanID == nil {
		te.RootSpan = &span
	}

	s.recomputeTraceProperties(te)
}

// recomputeTraceProperties recomputes derived trace properties.
func (s *InMemoryObservabilityStorage) recomputeTraceProperties(te *traceEntry) {
	if len(te.Spans) == 0 {
		return
	}

	// Compute hasChildError.
	te.HasChildError = false
	for _, span := range te.Spans {
		if span.Error != nil {
			te.HasChildError = true
			break
		}
	}

	// Compute status from root span.
	if te.RootSpan != nil {
		te.Status = ComputeTraceStatus(te.RootSpan)
	} else {
		te.Status = TraceStatusRunning
	}
}

// traceMatchesFilters checks if a trace matches all provided filters.
func (s *InMemoryObservabilityStorage) traceMatchesFilters(te *traceEntry, filters *TracesFilter) bool {
	if filters == nil {
		return true
	}

	root := te.RootSpan
	if root == nil {
		return false
	}

	// Date range filters on startedAt.
	if filters.StartedAt != nil {
		if filters.StartedAt.Start != nil && root.StartedAt.Before(*filters.StartedAt.Start) {
			return false
		}
		if filters.StartedAt.End != nil && root.StartedAt.After(*filters.StartedAt.End) {
			return false
		}
	}

	// Date range filters on endedAt.
	if filters.EndedAt != nil {
		if root.EndedAt == nil {
			return false
		}
		if filters.EndedAt.Start != nil && root.EndedAt.Before(*filters.EndedAt.Start) {
			return false
		}
		if filters.EndedAt.End != nil && root.EndedAt.After(*filters.EndedAt.End) {
			return false
		}
	}

	// Span type filter.
	if filters.SpanTyp != nil && root.SpanTyp != *filters.SpanTyp {
		return false
	}

	// Entity filters.
	if filters.EntityType != nil && ptrStr(root.EntityType) != *filters.EntityType {
		return false
	}
	if filters.EntityID != nil && ptrStr(root.EntityID) != *filters.EntityID {
		return false
	}
	if filters.EntityName != nil && ptrStr(root.EntityName) != *filters.EntityName {
		return false
	}

	// Identity & Tenancy filters.
	if filters.UserID != nil && ptrStr(root.UserID) != *filters.UserID {
		return false
	}
	if filters.OrganizationID != nil && ptrStr(root.OrganizationID) != *filters.OrganizationID {
		return false
	}
	if filters.ResourceID != nil && ptrStr(root.ResourceID) != *filters.ResourceID {
		return false
	}

	// Correlation ID filters.
	if filters.RunID != nil && ptrStr(root.RunID) != *filters.RunID {
		return false
	}
	if filters.SessionID != nil && ptrStr(root.SessionID) != *filters.SessionID {
		return false
	}
	if filters.ThreadID != nil && ptrStr(root.ThreadID) != *filters.ThreadID {
		return false
	}
	if filters.RequestID != nil && ptrStr(root.RequestID) != *filters.RequestID {
		return false
	}

	// Deployment context filters.
	if filters.Environment != nil && ptrStr(root.Environment) != *filters.Environment {
		return false
	}
	if filters.Source != nil && ptrStr(root.Source) != *filters.Source {
		return false
	}
	if filters.ServiceName != nil && ptrStr(root.ServiceName) != *filters.ServiceName {
		return false
	}

	// Scope filter (partial match).
	if filters.Scope != nil {
		if root.Scope == nil {
			return false
		}
		for k, v := range filters.Scope {
			if !jsonValueEquals(root.Scope[k], v) {
				return false
			}
		}
	}

	// Metadata filter (partial match).
	if filters.Metadata != nil {
		if root.Metadata == nil {
			return false
		}
		for k, v := range filters.Metadata {
			if !jsonValueEquals(root.Metadata[k], v) {
				return false
			}
		}
	}

	// Tags filter (all provided tags must be present).
	if len(filters.Tags) > 0 {
		if root.Tags == nil {
			return false
		}
		tagSet := make(map[string]bool, len(root.Tags))
		for _, t := range root.Tags {
			tagSet[t] = true
		}
		for _, t := range filters.Tags {
			if !tagSet[t] {
				return false
			}
		}
	}

	// Derived status filter.
	if filters.Status != nil && te.Status != *filters.Status {
		return false
	}

	// Has child error filter.
	if filters.HasChildError != nil && te.HasChildError != *filters.HasChildError {
		return false
	}

	return true
}

// createSpanToRecord converts a CreateSpanRecord to a SpanRecord.
func createSpanToRecord(cs CreateSpanRecord, now time.Time) SpanRecord {
	return SpanRecord{
		TraceID:        cs.TraceID,
		SpanID:         cs.SpanID,
		Name:           cs.Name,
		SpanTyp:        cs.SpanTyp,
		IsEvent:        cs.IsEvent,
		StartedAt:      cs.StartedAt,
		ParentSpanID:   cs.ParentSpanID,
		EntityType:     cs.EntityType,
		EntityID:       cs.EntityID,
		EntityName:     cs.EntityName,
		UserID:         cs.UserID,
		OrganizationID: cs.OrganizationID,
		ResourceID:     cs.ResourceID,
		RunID:          cs.RunID,
		SessionID:      cs.SessionID,
		ThreadID:       cs.ThreadID,
		RequestID:      cs.RequestID,
		Environment:    cs.Environment,
		Source:         cs.Source,
		ServiceName:    cs.ServiceName,
		Scope:          cs.Scope,
		Metadata:       cs.Metadata,
		Tags:           cs.Tags,
		Attributes:     cs.Attributes,
		Links:          cs.Links,
		Input:          cs.Input,
		Output:         cs.Output,
		Error:          cs.Error,
		EndedAt:        cs.EndedAt,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// applySpanUpdates applies partial updates to a span record.
func applySpanUpdates(span SpanRecord, updates UpdateSpanRecord) SpanRecord {
	if updates.Name != nil {
		span.Name = *updates.Name
	}
	if updates.SpanTyp != nil {
		span.SpanTyp = *updates.SpanTyp
	}
	if updates.IsEvent != nil {
		span.IsEvent = *updates.IsEvent
	}
	if updates.StartedAt != nil {
		span.StartedAt = *updates.StartedAt
	}
	if updates.ParentSpanID != nil {
		span.ParentSpanID = updates.ParentSpanID
	}
	if updates.EntityType != nil {
		span.EntityType = updates.EntityType
	}
	if updates.EntityID != nil {
		span.EntityID = updates.EntityID
	}
	if updates.EntityName != nil {
		span.EntityName = updates.EntityName
	}
	if updates.UserID != nil {
		span.UserID = updates.UserID
	}
	if updates.OrganizationID != nil {
		span.OrganizationID = updates.OrganizationID
	}
	if updates.ResourceID != nil {
		span.ResourceID = updates.ResourceID
	}
	if updates.RunID != nil {
		span.RunID = updates.RunID
	}
	if updates.SessionID != nil {
		span.SessionID = updates.SessionID
	}
	if updates.ThreadID != nil {
		span.ThreadID = updates.ThreadID
	}
	if updates.RequestID != nil {
		span.RequestID = updates.RequestID
	}
	if updates.Environment != nil {
		span.Environment = updates.Environment
	}
	if updates.Source != nil {
		span.Source = updates.Source
	}
	if updates.ServiceName != nil {
		span.ServiceName = updates.ServiceName
	}
	if updates.Scope != nil {
		span.Scope = updates.Scope
	}
	if updates.Metadata != nil {
		span.Metadata = updates.Metadata
	}
	if updates.Tags != nil {
		span.Tags = updates.Tags
	}
	if updates.Attributes != nil {
		span.Attributes = updates.Attributes
	}
	if updates.Links != nil {
		span.Links = updates.Links
	}
	if updates.Input != nil {
		span.Input = updates.Input
	}
	if updates.Output != nil {
		span.Output = updates.Output
	}
	if updates.Error != nil {
		span.Error = updates.Error
	}
	if updates.EndedAt != nil {
		span.EndedAt = updates.EndedAt
	}
	return span
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// jsonValueEquals compares two values using JSON-based deep equality.
func jsonValueEquals(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Try direct comparison first for simple types.
	if reflect.DeepEqual(a, b) {
		return true
	}

	// Fall back to JSON comparison for complex types.
	aJSON, errA := json.Marshal(a)
	bJSON, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return false
	}
	return string(aJSON) == string(bJSON)
}
