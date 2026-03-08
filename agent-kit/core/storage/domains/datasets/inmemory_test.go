// Ported from: packages/core/src/storage/domains/datasets/__tests__/datasets.test.ts
package datasets

import (
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains"
	"context"
	"testing"
)

func TestInMemoryDatasetsStorage_DatasetCRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("should create a dataset with version=0", func(t *testing.T) {
		storage := NewInMemoryDatasetsStorage()
		result, err := storage.CreateDataset(ctx, CreateDatasetInput{
			Name:     "Test Dataset",
			Metadata: map[string]any{"env": "test"},
		})
		if err != nil {
			t.Fatalf("CreateDataset returned error: %v", err)
		}
		if result.Name != "Test Dataset" {
			t.Errorf("expected name='Test Dataset', got %s", result.Name)
		}
		// version should be 0 initially
		if result.Version != 0 {
			t.Errorf("expected version=0, got %d", result.Version)
		}
	})

	t.Run("should get dataset by id", func(t *testing.T) {
		storage := NewInMemoryDatasetsStorage()
		created, _ := storage.CreateDataset(ctx, CreateDatasetInput{
			Name: "Get Test",
		})
		id := created.ID

		result, err := storage.GetDatasetByID(ctx, id)
		if err != nil {
			t.Fatalf("GetDatasetByID returned error: %v", err)
		}
		if result.ID == "" {
			t.Fatal("expected dataset to exist")
		}
		if result.Name != "Get Test" {
			t.Errorf("expected name='Get Test', got %s", result.Name)
		}
	})

	t.Run("should update dataset", func(t *testing.T) {
		storage := NewInMemoryDatasetsStorage()
		created, _ := storage.CreateDataset(ctx, CreateDatasetInput{
			Name: "Before Update",
		})
		id := created.ID

		updatedName := "After Update"
		updatedDesc := "Updated description"
		result, err := storage.UpdateDataset(ctx, UpdateDatasetInput{
			ID:          id,
			Name:        &updatedName,
			Description: &updatedDesc,
		})
		if err != nil {
			t.Fatalf("UpdateDataset returned error: %v", err)
		}
		if result.Name != "After Update" {
			t.Errorf("expected name='After Update', got %s", result.Name)
		}
	})

	t.Run("should delete dataset and cascade items", func(t *testing.T) {
		storage := NewInMemoryDatasetsStorage()
		created, _ := storage.CreateDataset(ctx, CreateDatasetInput{
			Name: "Delete Test",
		})
		id := created.ID

		// Add an item first
		_, _ = storage.AddItem(ctx, AddDatasetItemInput{
			DatasetID: id,
			Input:     map[string]any{"text": "hello"},
		})

		err := storage.DeleteDataset(ctx, id)
		if err != nil {
			t.Fatalf("DeleteDataset returned error: %v", err)
		}

		result, _ := storage.GetDatasetByID(ctx, id)
		if result.ID != "" {
			t.Error("expected dataset to be deleted")
		}
	})
}

func TestInMemoryDatasetsStorage_ItemCRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("should add an item", func(t *testing.T) {
		storage := NewInMemoryDatasetsStorage()
		created, _ := storage.CreateDataset(ctx, CreateDatasetInput{Name: "Items Test"})
		dsID := created.ID

		result, err := storage.AddItem(ctx, AddDatasetItemInput{
			DatasetID:   dsID,
			Input:       map[string]any{"text": "hello"},
			GroundTruth: map[string]any{"label": "greeting"},
		})
		if err != nil {
			t.Fatalf("AddItem returned error: %v", err)
		}
		if result.DatasetID != dsID {
			t.Errorf("expected datasetId=%s, got %s", dsID, result.DatasetID)
		}
	})

	t.Run("should get item by id", func(t *testing.T) {
		storage := NewInMemoryDatasetsStorage()
		created, _ := storage.CreateDataset(ctx, CreateDatasetInput{Name: "Get Item"})
		dsID := created.ID

		added, _ := storage.AddItem(ctx, AddDatasetItemInput{
			DatasetID: dsID,
			Input:     map[string]any{"text": "test"},
		})
		itemID := added.ID

		result, err := storage.GetItemByID(ctx, GetItemByIDArgs{
			ID: itemID,
		})
		if err != nil {
			t.Fatalf("GetItemByID returned error: %v", err)
		}
		if result.ID == "" {
			t.Fatal("expected item to exist")
		}
	})

	t.Run("should update item", func(t *testing.T) {
		storage := NewInMemoryDatasetsStorage()
		created, _ := storage.CreateDataset(ctx, CreateDatasetInput{Name: "Update Item"})
		dsID := created.ID

		added, _ := storage.AddItem(ctx, AddDatasetItemInput{
			DatasetID: dsID,
			Input:     map[string]any{"text": "before"},
		})
		itemID := added.ID

		result, err := storage.UpdateItem(ctx, UpdateDatasetItemInput{
			ID:        itemID,
			DatasetID: dsID,
			Input:     map[string]any{"text": "after"},
		})
		if err != nil {
			t.Fatalf("UpdateItem returned error: %v", err)
		}
		input, ok := result.Input.(map[string]any)
		if !ok {
			t.Fatalf("expected input to be map[string]any, got %T", result.Input)
		}
		if input["text"] != "after" {
			t.Errorf("expected input.text='after', got %v", input["text"])
		}
	})

	t.Run("should delete item", func(t *testing.T) {
		storage := NewInMemoryDatasetsStorage()
		created, _ := storage.CreateDataset(ctx, CreateDatasetInput{Name: "Delete Item"})
		dsID := created.ID

		added, _ := storage.AddItem(ctx, AddDatasetItemInput{
			DatasetID: dsID,
			Input:     map[string]any{"text": "delete-me"},
		})
		itemID := added.ID

		err := storage.DeleteItem(ctx, DeleteItemArgs{
			ID:        itemID,
			DatasetID: dsID,
		})
		if err != nil {
			t.Fatalf("DeleteItem returned error: %v", err)
		}
	})
}

func TestInMemoryDatasetsStorage_SCD2Versioning(t *testing.T) {
	ctx := context.Background()

	t.Run("should increment dataset version when items change", func(t *testing.T) {
		storage := NewInMemoryDatasetsStorage()
		created, _ := storage.CreateDataset(ctx, CreateDatasetInput{Name: "SCD2"})
		dsID := created.ID

		// Add first item — bumps version to 1
		_, _ = storage.AddItem(ctx, AddDatasetItemInput{
			DatasetID: dsID,
			Input:     map[string]any{"text": "v1"},
		})

		ds, _ := storage.GetDatasetByID(ctx, dsID)
		if ds.Version != 1 {
			t.Errorf("expected version=1 after first item, got %d", ds.Version)
		}

		// Add second item — bumps version to 2
		_, _ = storage.AddItem(ctx, AddDatasetItemInput{
			DatasetID: dsID,
			Input:     map[string]any{"text": "v2"},
		})

		ds, _ = storage.GetDatasetByID(ctx, dsID)
		if ds.Version != 2 {
			t.Errorf("expected version=2 after second item, got %d", ds.Version)
		}
	})
}

func TestInMemoryDatasetsStorage_BatchOperations(t *testing.T) {
	ctx := context.Background()

	t.Run("batchInsertItems should use single version increment", func(t *testing.T) {
		storage := NewInMemoryDatasetsStorage()
		created, _ := storage.CreateDataset(ctx, CreateDatasetInput{Name: "Batch"})
		dsID := created.ID

		_, err := storage.BatchInsertItems(ctx, BatchInsertItemsInput{
			DatasetID: dsID,
			Items: []domains.BatchInsertItemInput{
				{Input: map[string]any{"text": "item1"}},
				{Input: map[string]any{"text": "item2"}},
				{Input: map[string]any{"text": "item3"}},
			},
		})
		if err != nil {
			t.Fatalf("BatchInsertItems returned error: %v", err)
		}

		ds, _ := storage.GetDatasetByID(ctx, dsID)
		// Batch insert should only increment version once
		if ds.Version != 1 {
			t.Errorf("expected version=1 after batch insert, got %d", ds.Version)
		}
	})
}

func TestInMemoryDatasetsStorage_DangerouslyClearAll(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryDatasetsStorage()
	_, _ = storage.CreateDataset(ctx, CreateDatasetInput{Name: "Clear 1"})
	_, _ = storage.CreateDataset(ctx, CreateDatasetInput{Name: "Clear 2"})

	err := storage.DangerouslyClearAll(ctx)
	if err != nil {
		t.Fatalf("DangerouslyClearAll returned error: %v", err)
	}

	result, _ := storage.ListDatasets(ctx, ListDatasetsInput{})
	if result.Pagination.Total != 0 {
		t.Errorf("expected total=0, got %d", result.Pagination.Total)
	}
}
