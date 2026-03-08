// Ported from: packages/core/src/storage/domains/datasets/inmemory.ts
package datasets

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	domains "github.com/brainlet/brainkit/agent-kit/core/storage/domains"
)

// Compile-time interface check.
var _ DatasetsStorage = (*InMemoryDatasetsStorage)(nil)

// InMemoryDatasetsStorage is an in-memory implementation of DatasetsStorage.
type InMemoryDatasetsStorage struct {
	mu              sync.RWMutex
	datasets        map[string]DatasetRecord
	datasetItems    map[string][]DatasetItemRow // keyed by item ID, value is array of SCD-2 rows
	datasetVersions map[string]DatasetVersion
}

// NewInMemoryDatasetsStorage creates a new InMemoryDatasetsStorage.
func NewInMemoryDatasetsStorage() *InMemoryDatasetsStorage {
	return &InMemoryDatasetsStorage{
		datasets:        make(map[string]DatasetRecord),
		datasetItems:    make(map[string][]DatasetItemRow),
		datasetVersions: make(map[string]DatasetVersion),
	}
}

// Init is a no-op for the in-memory store.
func (s *InMemoryDatasetsStorage) Init(_ context.Context) error {
	return nil
}

// DangerouslyClearAll clears all data.
func (s *InMemoryDatasetsStorage) DangerouslyClearAll(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.datasets = make(map[string]DatasetRecord)
	s.datasetItems = make(map[string][]DatasetItemRow)
	s.datasetVersions = make(map[string]DatasetVersion)
	return nil
}

// --- Dataset CRUD ---

func (s *InMemoryDatasetsStorage) CreateDataset(_ context.Context, input CreateDatasetInput) (DatasetRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New().String()
	now := time.Now()

	ds := DatasetRecord{
		ID:                id,
		Name:              input.Name,
		Description:       input.Description,
		Metadata:          input.Metadata,
		InputSchema:       input.InputSchema,
		GroundTruthSchema: input.GroundTruthSchema,
		Version:           0,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	s.datasets[id] = ds
	return ds, nil
}

func (s *InMemoryDatasetsStorage) GetDatasetByID(_ context.Context, id string) (DatasetRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ds, ok := s.datasets[id]
	if !ok {
		return DatasetRecord{}, nil
	}
	return ds, nil
}

func (s *InMemoryDatasetsStorage) UpdateDataset(_ context.Context, args UpdateDatasetInput) (DatasetRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.datasets[args.ID]
	if !ok {
		return DatasetRecord{}, fmt.Errorf("dataset not found: %s", args.ID)
	}

	if args.Name != nil {
		existing.Name = *args.Name
	}
	if args.Description != nil {
		existing.Description = args.Description
	}
	if args.Metadata != nil {
		existing.Metadata = args.Metadata
	}
	if args.InputSchema != nil {
		existing.InputSchema = args.InputSchema
	}
	if args.GroundTruthSchema != nil {
		existing.GroundTruthSchema = args.GroundTruthSchema
	}
	existing.UpdatedAt = time.Now()

	s.datasets[args.ID] = existing
	return existing, nil
}

func (s *InMemoryDatasetsStorage) DeleteDataset(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for itemID, rows := range s.datasetItems {
		if len(rows) > 0 && rows[0].DatasetID == id {
			delete(s.datasetItems, itemID)
		}
	}
	for vID, v := range s.datasetVersions {
		if v.DatasetID == id {
			delete(s.datasetVersions, vID)
		}
	}

	delete(s.datasets, id)
	return nil
}

func (s *InMemoryDatasetsStorage) ListDatasets(_ context.Context, args ListDatasetsInput) (ListDatasetsOutput, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var dsets []DatasetRecord
	for _, ds := range s.datasets {
		dsets = append(dsets, ds)
	}

	sort.Slice(dsets, func(i, j int) bool {
		return dsets[j].CreatedAt.Before(dsets[i].CreatedAt)
	})

	p := parsePagFromStruct(args.Pagination, 100)
	result := paginate(dsets, p)

	return ListDatasetsOutput{
		Datasets:   result.items,
		Pagination: result.pag,
	}, nil
}

// --- Item CRUD (SCD-2 internally) ---

func (s *InMemoryDatasetsStorage) AddItem(_ context.Context, args AddDatasetItemInput) (DatasetItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ds, ok := s.datasets[args.DatasetID]
	if !ok {
		return DatasetItem{}, fmt.Errorf("dataset not found: %s", args.DatasetID)
	}

	newVersion := ds.Version + 1
	ds.Version = newVersion
	s.datasets[args.DatasetID] = ds

	now := time.Now()
	id := uuid.New().String()
	row := DatasetItemRow{
		ID:             id,
		DatasetID:      args.DatasetID,
		DatasetVersion: newVersion,
		ValidTo:        nil,
		IsDeleted:      false,
		Input:          args.Input,
		GroundTruth:    args.GroundTruth,
		Metadata:       args.Metadata,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	s.datasetItems[id] = []DatasetItemRow{row}
	s.createDatasetVersionInternal(args.DatasetID, newVersion)

	return toDatasetItem(row), nil
}

func (s *InMemoryDatasetsStorage) UpdateItem(_ context.Context, args UpdateDatasetItemInput) (DatasetItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, ok := s.datasetItems[args.ID]
	if !ok || len(rows) == 0 {
		return DatasetItem{}, fmt.Errorf("item not found: %s", args.ID)
	}

	currentRow := findCurrentRow(rows)
	if currentRow == nil {
		return DatasetItem{}, fmt.Errorf("item not found: %s", args.ID)
	}
	if currentRow.DatasetID != args.DatasetID {
		return DatasetItem{}, fmt.Errorf("item %s does not belong to dataset %s", args.ID, args.DatasetID)
	}

	ds, ok := s.datasets[args.DatasetID]
	if !ok {
		return DatasetItem{}, fmt.Errorf("dataset not found: %s", args.DatasetID)
	}

	newVersion := ds.Version + 1
	ds.Version = newVersion
	s.datasets[args.DatasetID] = ds

	currentRow.ValidTo = &newVersion

	now := time.Now()
	newInput := currentRow.Input
	if args.Input != nil {
		newInput = args.Input
	}
	newGroundTruth := currentRow.GroundTruth
	if args.GroundTruth != nil {
		newGroundTruth = args.GroundTruth
	}
	newMetadata := currentRow.Metadata
	if args.Metadata != nil {
		newMetadata = args.Metadata
	}
	newRow := DatasetItemRow{
		ID:             args.ID,
		DatasetID:      args.DatasetID,
		DatasetVersion: newVersion,
		ValidTo:        nil,
		IsDeleted:      false,
		Input:          newInput,
		GroundTruth:    newGroundTruth,
		Metadata:       newMetadata,
		CreatedAt:      currentRow.CreatedAt,
		UpdatedAt:      now,
	}
	rows = append(rows, newRow)
	s.datasetItems[args.ID] = rows
	s.createDatasetVersionInternal(args.DatasetID, newVersion)

	return toDatasetItem(newRow), nil
}

func (s *InMemoryDatasetsStorage) DeleteItem(_ context.Context, args DeleteItemArgs) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, ok := s.datasetItems[args.ID]
	if !ok || len(rows) == 0 {
		return nil
	}

	currentRow := findCurrentRow(rows)
	if currentRow == nil {
		return nil
	}
	if currentRow.DatasetID != args.DatasetID {
		return fmt.Errorf("item %s does not belong to dataset %s", args.ID, args.DatasetID)
	}

	ds, ok := s.datasets[args.DatasetID]
	if !ok {
		return fmt.Errorf("dataset not found: %s", args.DatasetID)
	}

	newVersion := ds.Version + 1
	ds.Version = newVersion
	s.datasets[args.DatasetID] = ds

	currentRow.ValidTo = &newVersion

	now := time.Now()
	rows = append(rows, DatasetItemRow{
		ID:             args.ID,
		DatasetID:      args.DatasetID,
		DatasetVersion: newVersion,
		ValidTo:        nil,
		IsDeleted:      true,
		Input:          currentRow.Input,
		GroundTruth:    currentRow.GroundTruth,
		Metadata:       currentRow.Metadata,
		CreatedAt:      currentRow.CreatedAt,
		UpdatedAt:      now,
	})
	s.datasetItems[args.ID] = rows
	s.createDatasetVersionInternal(args.DatasetID, newVersion)
	return nil
}

func (s *InMemoryDatasetsStorage) GetItemByID(_ context.Context, args GetItemByIDArgs) (DatasetItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, ok := s.datasetItems[args.ID]
	if !ok || len(rows) == 0 {
		return DatasetItem{}, nil
	}

	if args.DatasetVersion != nil {
		for i := range rows {
			if rows[i].DatasetVersion == *args.DatasetVersion && !rows[i].IsDeleted {
				return toDatasetItem(rows[i]), nil
			}
		}
		return DatasetItem{}, nil
	}

	cr := findCurrentRowConst(rows)
	if cr == nil {
		return DatasetItem{}, nil
	}
	return toDatasetItem(*cr), nil
}

func (s *InMemoryDatasetsStorage) GetItemsByVersion(_ context.Context, args GetItemsByVersionArgs) ([]DatasetItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var items []DatasetItem
	for _, rows := range s.datasetItems {
		if len(rows) == 0 || rows[0].DatasetID != args.DatasetID {
			continue
		}
		for _, r := range rows {
			if r.DatasetVersion <= args.Version &&
				(r.ValidTo == nil || *r.ValidTo > args.Version) &&
				!r.IsDeleted {
				items = append(items, toDatasetItem(r))
				break
			}
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if !items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[j].CreatedAt.Before(items[i].CreatedAt)
		}
		return items[j].ID < items[i].ID
	})

	return items, nil
}

func (s *InMemoryDatasetsStorage) GetItemHistory(_ context.Context, itemID string) ([]DatasetItemRow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, ok := s.datasetItems[itemID]
	if !ok {
		return []DatasetItemRow{}, nil
	}

	sorted := make([]DatasetItemRow, len(rows))
	copy(sorted, rows)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[j].DatasetVersion < sorted[i].DatasetVersion
	})

	return sorted, nil
}

func (s *InMemoryDatasetsStorage) ListItems(_ context.Context, args ListDatasetItemsInput) (ListDatasetItemsOutput, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var items []DatasetItem

	if args.Version != nil {
		version := *args.Version
		for _, rows := range s.datasetItems {
			if len(rows) == 0 || rows[0].DatasetID != args.DatasetID {
				continue
			}
			for _, r := range rows {
				if r.DatasetVersion <= version &&
					(r.ValidTo == nil || *r.ValidTo > version) &&
					!r.IsDeleted {
					items = append(items, toDatasetItem(r))
					break
				}
			}
		}
	} else {
		for _, rows := range s.datasetItems {
			if len(rows) == 0 || rows[0].DatasetID != args.DatasetID {
				continue
			}
			cr := findCurrentRowConst(rows)
			if cr != nil {
				items = append(items, toDatasetItem(*cr))
			}
		}
	}

	if args.Search != nil && *args.Search != "" {
		searchLower := strings.ToLower(*args.Search)
		var filtered []DatasetItem
		for _, item := range items {
			inputStr := jsonStr(item.Input)
			groundTruthStr := jsonStr(item.GroundTruth)
			if strings.Contains(strings.ToLower(inputStr), searchLower) ||
				strings.Contains(strings.ToLower(groundTruthStr), searchLower) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	sort.Slice(items, func(i, j int) bool {
		if !items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[j].CreatedAt.Before(items[i].CreatedAt)
		}
		return items[j].ID < items[i].ID
	})

	p := parsePagFromStruct(args.Pagination, 100)
	result := paginate(items, p)

	return ListDatasetItemsOutput{
		Items:      result.items,
		Pagination: result.pag,
	}, nil
}

// --- Dataset Version Methods ---

func (s *InMemoryDatasetsStorage) CreateDatasetVersion(_ context.Context, datasetID string, version int) (DatasetVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.createDatasetVersionInternal(datasetID, version), nil
}

func (s *InMemoryDatasetsStorage) ListDatasetVersions(_ context.Context, input ListDatasetVersionsInput) (ListDatasetVersionsOutput, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var versions []DatasetVersion
	for _, v := range s.datasetVersions {
		if v.DatasetID == input.DatasetID {
			versions = append(versions, v)
		}
	}
	sort.Slice(versions, func(i, j int) bool {
		return versions[j].Version < versions[i].Version
	})

	p := parsePagFromStruct(input.Pagination, 100)
	result := paginate(versions, p)

	return ListDatasetVersionsOutput{
		Versions:   result.items,
		Pagination: result.pag,
	}, nil
}

// --- Batch Operations ---

func (s *InMemoryDatasetsStorage) BatchInsertItems(_ context.Context, input BatchInsertItemsInput) ([]DatasetItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ds, ok := s.datasets[input.DatasetID]
	if !ok {
		return nil, fmt.Errorf("dataset not found: %s", input.DatasetID)
	}

	newVersion := ds.Version + 1
	ds.Version = newVersion
	s.datasets[input.DatasetID] = ds

	now := time.Now()
	var items []DatasetItem

	for _, item := range input.Items {
		id := uuid.New().String()
		row := DatasetItemRow{
			ID:             id,
			DatasetID:      input.DatasetID,
			DatasetVersion: newVersion,
			ValidTo:        nil,
			IsDeleted:      false,
			Input:          item.Input,
			GroundTruth:    item.GroundTruth,
			Metadata:       item.Metadata,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		s.datasetItems[id] = []DatasetItemRow{row}
		items = append(items, toDatasetItem(row))
	}

	s.createDatasetVersionInternal(input.DatasetID, newVersion)
	return items, nil
}

func (s *InMemoryDatasetsStorage) BatchDeleteItems(_ context.Context, input BatchDeleteItemsInput) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ds, ok := s.datasets[input.DatasetID]
	if !ok {
		return fmt.Errorf("dataset not found: %s", input.DatasetID)
	}

	newVersion := ds.Version + 1
	ds.Version = newVersion
	s.datasets[input.DatasetID] = ds

	now := time.Now()

	for _, itemID := range input.ItemIDs {
		rows, ok := s.datasetItems[itemID]
		if !ok {
			continue
		}
		currentRow := findCurrentRow(rows)
		if currentRow == nil || currentRow.DatasetID != input.DatasetID {
			continue
		}

		currentRow.ValidTo = &newVersion

		rows = append(rows, DatasetItemRow{
			ID:             itemID,
			DatasetID:      input.DatasetID,
			DatasetVersion: newVersion,
			ValidTo:        nil,
			IsDeleted:      true,
			Input:          currentRow.Input,
			GroundTruth:    currentRow.GroundTruth,
			Metadata:       currentRow.Metadata,
			CreatedAt:      currentRow.CreatedAt,
			UpdatedAt:      now,
		})
		s.datasetItems[itemID] = rows
	}

	s.createDatasetVersionInternal(input.DatasetID, newVersion)
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func toDatasetItem(row DatasetItemRow) DatasetItem {
	return DatasetItem{
		ID:             row.ID,
		DatasetID:      row.DatasetID,
		DatasetVersion: row.DatasetVersion,
		Input:          row.Input,
		GroundTruth:    row.GroundTruth,
		Metadata:       row.Metadata,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func (s *InMemoryDatasetsStorage) createDatasetVersionInternal(datasetID string, version int) DatasetVersion {
	id := uuid.New().String()
	dv := DatasetVersion{
		ID:        id,
		DatasetID: datasetID,
		Version:   version,
		CreatedAt: time.Now(),
	}
	s.datasetVersions[id] = dv
	return dv
}

func findCurrentRow(rows []DatasetItemRow) *DatasetItemRow {
	for i := range rows {
		if rows[i].ValidTo == nil && !rows[i].IsDeleted {
			return &rows[i]
		}
	}
	return nil
}

func findCurrentRowConst(rows []DatasetItemRow) *DatasetItemRow {
	for i := range rows {
		if rows[i].ValidTo == nil && !rows[i].IsDeleted {
			return &rows[i]
		}
	}
	return nil
}

type pagParams struct {
	page    int
	perPage int
	noLimit bool
}

func parsePagFromStruct(pag domains.StoragePagination, defaultPerPage int) pagParams {
	page := pag.Page
	perPage := pag.PerPage
	if perPage <= 0 {
		if perPage == 0 {
			perPage = defaultPerPage
		} else {
			perPage = math.MaxInt
			return pagParams{page: page, perPage: perPage, noLimit: true}
		}
	}
	return pagParams{page: page, perPage: perPage, noLimit: false}
}

type pagResult[T any] struct {
	items []T
	pag   domains.PaginationInfo
}

func paginate[T any](items []T, p pagParams) pagResult[T] {
	if items == nil {
		items = []T{}
	}
	total := len(items)
	start := p.page * p.perPage
	end := start + p.perPage
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	responsePerPage := p.perPage
	if p.noLimit {
		responsePerPage = domains.PerPageDisabled
	}
	hasMore := !p.noLimit && total > end
	return pagResult[T]{
		items: items[start:end],
		pag: domains.PaginationInfo{
			Total:   total,
			Page:    p.page,
			PerPage: responsePerPage,
			HasMore: hasMore,
		},
	}
}

func jsonStr(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}
