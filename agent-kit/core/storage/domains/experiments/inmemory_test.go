// Ported from: packages/core/src/storage/domains/experiments/__tests__/experiments.test.ts
package experiments

import (
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains"
	"context"
	"testing"
)

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func TestInMemoryExperimentsStorage_CreateExperiment(t *testing.T) {
	ctx := context.Background()

	t.Run("should create with pending status", func(t *testing.T) {
		storage := NewInMemoryExperimentsStorage()
		result, err := storage.CreateExperiment(ctx, CreateExperimentInput{
			Name:      strPtr("Test Experiment"),
			DatasetID: strPtr("ds-1"),
		})
		if err != nil {
			t.Fatalf("CreateExperiment returned error: %v", err)
		}
		if result.Status != domains.ExperimentStatusPending {
			t.Errorf("expected status=pending, got %s", result.Status)
		}
		if result.Name == nil || *result.Name != "Test Experiment" {
			t.Errorf("expected name='Test Experiment', got %v", result.Name)
		}
	})

	t.Run("should accept custom id", func(t *testing.T) {
		storage := NewInMemoryExperimentsStorage()
		result, err := storage.CreateExperiment(ctx, CreateExperimentInput{
			ID:        strPtr("custom-id"),
			Name:      strPtr("Custom ID"),
			DatasetID: strPtr("ds-1"),
		})
		if err != nil {
			t.Fatalf("CreateExperiment returned error: %v", err)
		}
		if result.ID != "custom-id" {
			t.Errorf("expected id=custom-id, got %s", result.ID)
		}
	})

	t.Run("should accept datasetVersion as integer", func(t *testing.T) {
		storage := NewInMemoryExperimentsStorage()
		result, err := storage.CreateExperiment(ctx, CreateExperimentInput{
			Name:           strPtr("With Version"),
			DatasetID:      strPtr("ds-1"),
			DatasetVersion: intPtr(3),
		})
		if err != nil {
			t.Fatalf("CreateExperiment returned error: %v", err)
		}
		if result.DatasetVersion == nil || *result.DatasetVersion != 3 {
			t.Errorf("expected datasetVersion=3, got %v", result.DatasetVersion)
		}
	})
}

func TestInMemoryExperimentsStorage_UpdateExperiment(t *testing.T) {
	ctx := context.Background()

	t.Run("should update status", func(t *testing.T) {
		storage := NewInMemoryExperimentsStorage()
		created, _ := storage.CreateExperiment(ctx, CreateExperimentInput{
			Name:      strPtr("Update Test"),
			DatasetID: strPtr("ds-1"),
		})
		id := created.ID

		runningStatus := domains.ExperimentStatusRunning
		result, err := storage.UpdateExperiment(ctx, UpdateExperimentInput{
			ID:     id,
			Status: &runningStatus,
		})
		if err != nil {
			t.Fatalf("UpdateExperiment returned error: %v", err)
		}
		if result.Status != domains.ExperimentStatusRunning {
			t.Errorf("expected status=running, got %s", result.Status)
		}
	})

	t.Run("should update counts", func(t *testing.T) {
		storage := NewInMemoryExperimentsStorage()
		created, _ := storage.CreateExperiment(ctx, CreateExperimentInput{
			Name:      strPtr("Counts Test"),
			DatasetID: strPtr("ds-1"),
		})
		id := created.ID

		result, err := storage.UpdateExperiment(ctx, UpdateExperimentInput{
			ID:             id,
			SucceededCount: intPtr(5),
			FailedCount:    intPtr(2),
			TotalItems:     intPtr(10),
		})
		if err != nil {
			t.Fatalf("UpdateExperiment returned error: %v", err)
		}
		if result.SucceededCount != 5 {
			t.Errorf("expected succeededCount=5, got %d", result.SucceededCount)
		}
		if result.FailedCount != 2 {
			t.Errorf("expected failedCount=2, got %d", result.FailedCount)
		}
	})

	t.Run("should error for non-existent experiment", func(t *testing.T) {
		storage := NewInMemoryExperimentsStorage()
		runningStatus := domains.ExperimentStatusRunning
		_, err := storage.UpdateExperiment(ctx, UpdateExperimentInput{
			ID:     "non-existent",
			Status: &runningStatus,
		})
		if err == nil {
			t.Fatal("expected error for non-existent experiment")
		}
	})
}

func TestInMemoryExperimentsStorage_GetExperimentByID(t *testing.T) {
	ctx := context.Background()

	t.Run("should return experiment by id", func(t *testing.T) {
		storage := NewInMemoryExperimentsStorage()
		created, _ := storage.CreateExperiment(ctx, CreateExperimentInput{
			Name:      strPtr("Get Test"),
			DatasetID: strPtr("ds-1"),
		})
		id := created.ID

		result, err := storage.GetExperimentByID(ctx, id)
		if err != nil {
			t.Fatalf("GetExperimentByID returned error: %v", err)
		}
		if result.ID == "" {
			t.Fatal("expected experiment to exist")
		}
		if result.Name == nil || *result.Name != "Get Test" {
			t.Errorf("expected name='Get Test', got %v", result.Name)
		}
	})

	t.Run("should return zero-value for non-existent", func(t *testing.T) {
		storage := NewInMemoryExperimentsStorage()
		result, err := storage.GetExperimentByID(ctx, "non-existent")
		if err != nil {
			t.Fatalf("GetExperimentByID returned error: %v", err)
		}
		if result.ID != "" {
			t.Error("expected zero-value experiment for non-existent")
		}
	})
}

func TestInMemoryExperimentsStorage_ListExperiments(t *testing.T) {
	ctx := context.Background()

	t.Run("should list all experiments", func(t *testing.T) {
		storage := NewInMemoryExperimentsStorage()
		_, _ = storage.CreateExperiment(ctx, CreateExperimentInput{
			Name: strPtr("Exp 1"), DatasetID: strPtr("ds-1"),
		})
		_, _ = storage.CreateExperiment(ctx, CreateExperimentInput{
			Name: strPtr("Exp 2"), DatasetID: strPtr("ds-1"),
		})

		result, err := storage.ListExperiments(ctx, ListExperimentsInput{})
		if err != nil {
			t.Fatalf("ListExperiments returned error: %v", err)
		}
		if result.Pagination.Total != 2 {
			t.Errorf("expected total=2, got %d", result.Pagination.Total)
		}
	})

	t.Run("should filter by datasetId", func(t *testing.T) {
		storage := NewInMemoryExperimentsStorage()
		_, _ = storage.CreateExperiment(ctx, CreateExperimentInput{
			Name: strPtr("A"), DatasetID: strPtr("ds-A"),
		})
		_, _ = storage.CreateExperiment(ctx, CreateExperimentInput{
			Name: strPtr("B"), DatasetID: strPtr("ds-B"),
		})
		_, _ = storage.CreateExperiment(ctx, CreateExperimentInput{
			Name: strPtr("C"), DatasetID: strPtr("ds-A"),
		})

		result, _ := storage.ListExperiments(ctx, ListExperimentsInput{
			DatasetID: strPtr("ds-A"),
		})
		if result.Pagination.Total != 2 {
			t.Errorf("expected total=2 for ds-A, got %d", result.Pagination.Total)
		}
	})

	t.Run("should paginate", func(t *testing.T) {
		storage := NewInMemoryExperimentsStorage()
		for i := 0; i < 5; i++ {
			_, _ = storage.CreateExperiment(ctx, CreateExperimentInput{
				Name: strPtr("P"), DatasetID: strPtr("ds-1"),
			})
		}

		result, _ := storage.ListExperiments(ctx, ListExperimentsInput{
			Pagination: domains.StoragePagination{Page: 0, PerPage: 2},
		})
		if len(result.Experiments) != 2 {
			t.Errorf("expected 2 experiments, got %d", len(result.Experiments))
		}
		if !result.Pagination.HasMore {
			t.Error("expected hasMore=true")
		}
	})
}

func TestInMemoryExperimentsStorage_DeleteExperiment(t *testing.T) {
	ctx := context.Background()

	t.Run("should delete experiment and cascade results", func(t *testing.T) {
		storage := NewInMemoryExperimentsStorage()
		created, _ := storage.CreateExperiment(ctx, CreateExperimentInput{
			Name: strPtr("Delete Test"), DatasetID: strPtr("ds-1"),
		})
		expID := created.ID

		// Add a result
		_, _ = storage.AddExperimentResult(ctx, AddExperimentResultInput{
			ExperimentID: expID,
			ItemID:       "item-1",
			Input:        map[string]any{"text": "hello"},
			Output:       map[string]any{"text": "world"},
		})

		err := storage.DeleteExperiment(ctx, expID)
		if err != nil {
			t.Fatalf("DeleteExperiment returned error: %v", err)
		}

		result, _ := storage.GetExperimentByID(ctx, expID)
		if result.ID != "" {
			t.Error("expected experiment to be deleted")
		}

		// Results should also be gone
		results, _ := storage.ListExperimentResults(ctx, ListExperimentResultsInput{
			ExperimentID: expID,
		})
		if results.Pagination.Total != 0 {
			t.Errorf("expected 0 results after cascade delete, got %d", results.Pagination.Total)
		}
	})
}

func TestInMemoryExperimentsStorage_ExperimentResults(t *testing.T) {
	ctx := context.Background()

	t.Run("should add an experiment result", func(t *testing.T) {
		storage := NewInMemoryExperimentsStorage()
		created, _ := storage.CreateExperiment(ctx, CreateExperimentInput{
			Name: strPtr("Results Test"), DatasetID: strPtr("ds-1"),
		})
		expID := created.ID

		result, err := storage.AddExperimentResult(ctx, AddExperimentResultInput{
			ExperimentID: expID,
			ItemID:       "item-1",
			Input:        map[string]any{"text": "hello"},
			Output:       map[string]any{"text": "world"},
		})
		if err != nil {
			t.Fatalf("AddExperimentResult returned error: %v", err)
		}
		if result.ExperimentID != expID {
			t.Errorf("expected experimentId=%s, got %s", expID, result.ExperimentID)
		}
	})

	t.Run("should list experiment results with pagination", func(t *testing.T) {
		storage := NewInMemoryExperimentsStorage()
		created, _ := storage.CreateExperiment(ctx, CreateExperimentInput{
			Name: strPtr("List Results"), DatasetID: strPtr("ds-1"),
		})
		expID := created.ID

		for i := 0; i < 5; i++ {
			_, _ = storage.AddExperimentResult(ctx, AddExperimentResultInput{
				ExperimentID: expID,
				ItemID:       "item-1",
				Input:        map[string]any{"idx": i},
				Output:       map[string]any{"idx": i},
			})
		}

		result, _ := storage.ListExperimentResults(ctx, ListExperimentResultsInput{
			ExperimentID: expID,
			Pagination:   domains.StoragePagination{Page: 0, PerPage: 2},
		})
		if len(result.Results) != 2 {
			t.Errorf("expected 2 results, got %d", len(result.Results))
		}
		if result.Pagination.Total != 5 {
			t.Errorf("expected total=5, got %d", result.Pagination.Total)
		}
	})

	t.Run("should store error in result", func(t *testing.T) {
		storage := NewInMemoryExperimentsStorage()
		created, _ := storage.CreateExperiment(ctx, CreateExperimentInput{
			Name: strPtr("Error Results"), DatasetID: strPtr("ds-1"),
		})
		expID := created.ID

		result, err := storage.AddExperimentResult(ctx, AddExperimentResultInput{
			ExperimentID: expID,
			ItemID:       "item-1",
			Input:        map[string]any{"text": "hello"},
			Error:        &domains.ExperimentResultError{Message: "something went wrong"},
		})
		if err != nil {
			t.Fatalf("AddExperimentResult returned error: %v", err)
		}
		if result.Error == nil || result.Error.Message != "something went wrong" {
			t.Errorf("expected error message='something went wrong', got %v", result.Error)
		}
	})
}

func TestInMemoryExperimentsStorage_DangerouslyClearAll(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryExperimentsStorage()
	_, _ = storage.CreateExperiment(ctx, CreateExperimentInput{
		Name: strPtr("Clear 1"), DatasetID: strPtr("ds-1"),
	})

	err := storage.DangerouslyClearAll(ctx)
	if err != nil {
		t.Fatalf("DangerouslyClearAll returned error: %v", err)
	}

	result, _ := storage.ListExperiments(ctx, ListExperimentsInput{})
	if result.Pagination.Total != 0 {
		t.Errorf("expected total=0, got %d", result.Pagination.Total)
	}
}
