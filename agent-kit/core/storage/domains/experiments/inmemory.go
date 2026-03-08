// Ported from: packages/core/src/storage/domains/experiments/inmemory.ts
package experiments

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	domains "github.com/brainlet/brainkit/agent-kit/core/storage/domains"
)

// Compile-time interface check.
var _ ExperimentsStorage = (*InMemoryExperimentsStorage)(nil)

// InMemoryExperimentsStorage is an in-memory implementation of ExperimentsStorage.
type InMemoryExperimentsStorage struct {
	mu                sync.RWMutex
	experiments       map[string]Experiment
	experimentResults map[string]ExperimentResult
}

// NewInMemoryExperimentsStorage creates a new InMemoryExperimentsStorage.
func NewInMemoryExperimentsStorage() *InMemoryExperimentsStorage {
	return &InMemoryExperimentsStorage{
		experiments:       make(map[string]Experiment),
		experimentResults: make(map[string]ExperimentResult),
	}
}

// Init is a no-op for the in-memory store.
func (s *InMemoryExperimentsStorage) Init(_ context.Context) error {
	return nil
}

// DangerouslyClearAll clears all experiments and results.
func (s *InMemoryExperimentsStorage) DangerouslyClearAll(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.experiments = make(map[string]Experiment)
	s.experimentResults = make(map[string]ExperimentResult)
	return nil
}

// CreateExperiment creates a new experiment.
func (s *InMemoryExperimentsStorage) CreateExperiment(_ context.Context, input CreateExperimentInput) (Experiment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New().String()
	if input.ID != nil && *input.ID != "" {
		id = *input.ID
	}
	now := time.Now()

	exp := Experiment{
		ID:             id,
		Name:           input.Name,
		Description:    input.Description,
		Metadata:       input.Metadata,
		DatasetID:      input.DatasetID,
		DatasetVersion: input.DatasetVersion,
		TargetType:     input.TargetType,
		TargetID:       input.TargetID,
		Status:         domains.ExperimentStatusPending,
		TotalItems:     input.TotalItems,
		SucceededCount: 0,
		FailedCount:    0,
		SkippedCount:   0,
		StartedAt:      nil,
		CompletedAt:    nil,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	s.experiments[id] = exp
	return exp, nil
}

// UpdateExperiment updates an existing experiment.
func (s *InMemoryExperimentsStorage) UpdateExperiment(_ context.Context, input UpdateExperimentInput) (Experiment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.experiments[input.ID]
	if !ok {
		return Experiment{}, fmt.Errorf("experiment not found: %s", input.ID)
	}

	if input.Status != nil {
		existing.Status = *input.Status
	}
	if input.TotalItems != nil {
		existing.TotalItems = *input.TotalItems
	}
	if input.SucceededCount != nil {
		existing.SucceededCount = *input.SucceededCount
	}
	if input.FailedCount != nil {
		existing.FailedCount = *input.FailedCount
	}
	if input.SkippedCount != nil {
		existing.SkippedCount = *input.SkippedCount
	}
	if input.Name != nil {
		existing.Name = input.Name
	}
	if input.Description != nil {
		existing.Description = input.Description
	}
	if input.Metadata != nil {
		existing.Metadata = input.Metadata
	}
	if input.StartedAt != nil {
		existing.StartedAt = input.StartedAt
	}
	if input.CompletedAt != nil {
		existing.CompletedAt = input.CompletedAt
	}
	existing.UpdatedAt = time.Now()

	s.experiments[input.ID] = existing
	return existing, nil
}

// GetExperimentByID retrieves an experiment by ID.
func (s *InMemoryExperimentsStorage) GetExperimentByID(_ context.Context, id string) (Experiment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	exp, ok := s.experiments[id]
	if !ok {
		return Experiment{}, nil
	}
	return exp, nil
}

// ListExperiments lists experiments with optional filtering.
func (s *InMemoryExperimentsStorage) ListExperiments(_ context.Context, args ListExperimentsInput) (ListExperimentsOutput, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var experiments []Experiment
	for _, exp := range s.experiments {
		if args.DatasetID != nil && *args.DatasetID != "" {
			// Filter by datasetID if provided.
			expDatasetID := ""
			if exp.DatasetID != nil {
				expDatasetID = *exp.DatasetID
			}
			if expDatasetID != *args.DatasetID {
				continue
			}
		}
		experiments = append(experiments, exp)
	}

	// Sort by createdAt descending (newest first).
	sort.Slice(experiments, func(i, j int) bool {
		return experiments[j].CreatedAt.Before(experiments[i].CreatedAt)
	})

	result := paginateExperiments(experiments, paginationParams{
		page:    args.Pagination.Page,
		perPage: args.Pagination.PerPage,
		noLimit: args.Pagination.PerPage <= 0,
	})

	return ListExperimentsOutput{
		Experiments: result.items,
		Pagination:  result.pagination,
	}, nil
}

// DeleteExperiment removes an experiment and its associated results.
func (s *InMemoryExperimentsStorage) DeleteExperiment(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.experiments, id)

	// Delete associated results.
	for resultID, result := range s.experimentResults {
		if result.ExperimentID == id {
			delete(s.experimentResults, resultID)
		}
	}
	return nil
}

// AddExperimentResult adds a per-item experiment result.
func (s *InMemoryExperimentsStorage) AddExperimentResult(_ context.Context, input AddExperimentResultInput) (ExperimentResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New().String()
	if input.ID != nil && *input.ID != "" {
		id = *input.ID
	}
	now := time.Now()

	result := ExperimentResult{
		ID:                 id,
		ExperimentID:       input.ExperimentID,
		ItemID:             input.ItemID,
		ItemDatasetVersion: input.ItemDatasetVersion,
		Input:              input.Input,
		Output:             input.Output,
		GroundTruth:        input.GroundTruth,
		Error:              input.Error,
		StartedAt:          input.StartedAt,
		CompletedAt:        input.CompletedAt,
		RetryCount:         input.RetryCount,
		TraceID:            input.TraceID,
		CreatedAt:          now,
	}

	s.experimentResults[result.ID] = result
	return result, nil
}

// GetExperimentResultByID retrieves an experiment result by ID.
func (s *InMemoryExperimentsStorage) GetExperimentResultByID(_ context.Context, id string) (ExperimentResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result, ok := s.experimentResults[id]
	if !ok {
		return ExperimentResult{}, nil
	}
	return result, nil
}

// ListExperimentResults lists results for an experiment.
func (s *InMemoryExperimentsStorage) ListExperimentResults(_ context.Context, args ListExperimentResultsInput) (ListExperimentResultsOutput, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []ExperimentResult
	for _, r := range s.experimentResults {
		if r.ExperimentID == args.ExperimentID {
			results = append(results, r)
		}
	}

	// Sort by startedAt ascending (execution order).
	sort.Slice(results, func(i, j int) bool {
		return results[i].StartedAt.Before(results[j].StartedAt)
	})

	paginated := paginateResults(results, paginationParams{
		page:    args.Pagination.Page,
		perPage: args.Pagination.PerPage,
		noLimit: args.Pagination.PerPage <= 0,
	})

	return ListExperimentResultsOutput{
		Results:    paginated.items,
		Pagination: paginated.pagination,
	}, nil
}

// DeleteExperimentResults removes all results for an experiment.
func (s *InMemoryExperimentsStorage) DeleteExperimentResults(_ context.Context, experimentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for resultID, result := range s.experimentResults {
		if result.ExperimentID == experimentID {
			delete(s.experimentResults, resultID)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Pagination helpers
// ---------------------------------------------------------------------------

type paginationParams struct {
	page    int
	perPage int
	noLimit bool
}

type paginatedResult[T any] struct {
	items      []T
	pagination domains.PaginationInfo
}

func paginateExperiments(items []Experiment, p paginationParams) paginatedResult[Experiment] {
	if items == nil {
		items = []Experiment{}
	}
	effectivePerPage := p.perPage
	if p.noLimit {
		effectivePerPage = math.MaxInt
	}
	total := len(items)
	start := p.page * effectivePerPage
	end := start + effectivePerPage
	if end > total {
		end = total
	}
	if start > total {
		start = total
	}
	responsePerPage := p.perPage
	if p.noLimit {
		responsePerPage = domains.PerPageDisabled
	}
	hasMore := !p.noLimit && total > end
	return paginatedResult[Experiment]{
		items: items[start:end],
		pagination: domains.PaginationInfo{
			Total:   total,
			Page:    p.page,
			PerPage: responsePerPage,
			HasMore: hasMore,
		},
	}
}

func paginateResults(items []ExperimentResult, p paginationParams) paginatedResult[ExperimentResult] {
	if items == nil {
		items = []ExperimentResult{}
	}
	effectivePerPage := p.perPage
	if p.noLimit {
		effectivePerPage = math.MaxInt
	}
	total := len(items)
	start := p.page * effectivePerPage
	end := start + effectivePerPage
	if end > total {
		end = total
	}
	if start > total {
		start = total
	}
	responsePerPage := p.perPage
	if p.noLimit {
		responsePerPage = domains.PerPageDisabled
	}
	hasMore := !p.noLimit && total > end
	return paginatedResult[ExperimentResult]{
		items: items[start:end],
		pagination: domains.PaginationInfo{
			Total:   total,
			Page:    p.page,
			PerPage: responsePerPage,
			HasMore: hasMore,
		},
	}
}
