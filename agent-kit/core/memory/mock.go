// Ported from: packages/core/src/memory/mock.ts
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/storage"
	memorystorage "github.com/brainlet/brainkit/agent-kit/core/storage/domains/memory"
	aktypes "github.com/brainlet/brainkit/agent-kit/core/types"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported (used only in this file)
// ---------------------------------------------------------------------------

// InMemoryStore is a stub for ../storage.InMemoryStore.
// TODO: import from storage package once the in-memory store is ported.
// For now, it wraps a MastraCompositeStore for mock purposes.
type InMemoryStore = storage.MastraCompositeStore

// ---------------------------------------------------------------------------
// MockMemory
// ---------------------------------------------------------------------------

// MockMemory is a concrete in-memory implementation of MastraMemory.
// Used primarily for testing purposes.
type MockMemory struct {
	*MastraMemoryBase
}

// MockMemoryConfig holds configuration for creating a MockMemory.
type MockMemoryConfig struct {
	// Storage is an optional in-memory store. If nil, a default is created.
	Storage *InMemoryStore
	// EnableWorkingMemory enables working memory. Defaults to false.
	EnableWorkingMemory bool
	// WorkingMemoryTemplate is an optional template for working memory.
	WorkingMemoryTemplate string
	// EnableMessageHistory enables message history. Defaults to true.
	EnableMessageHistory *bool
}

// NewMockMemory creates a new MockMemory instance.
func NewMockMemory(config *MockMemoryConfig) (*MockMemory, error) {
	if config == nil {
		config = &MockMemoryConfig{}
	}

	enableMessageHistory := true
	if config.EnableMessageHistory != nil {
		enableMessageHistory = *config.EnableMessageHistory
	}

	var workingMemory *WorkingMemory
	if config.EnableWorkingMemory {
		workingMemory = &WorkingMemory{
			Enabled:  true,
			Template: config.WorkingMemoryTemplate,
		}
	}

	var lastMessages *LastMessagesConfig
	if enableMessageHistory {
		lastMessages = &LastMessagesConfig{Count: 10}
	} else {
		lastMessages = &LastMessagesConfig{Disabled: true}
	}

	store := config.Storage
	// TODO: If store is nil, create a default InMemoryStore once that is ported.
	// For now, we allow nil storage and handle it lazily.

	base, err := NewMastraMemoryBase(MastraMemoryBaseConfig{
		Name:    "mock",
		Storage: store,
		Options: &MemoryConfig{
			WorkingMemory: workingMemory,
			LastMessages:  lastMessages,
		},
	})
	if err != nil {
		return nil, err
	}
	base.hasOwnStorage = true

	return &MockMemory{
		MastraMemoryBase: base,
	}, nil
}

// getMemoryStore retrieves the memory domain store.
func (m *MockMemory) getMemoryStore(_ context.Context) (memorystorage.MemoryStorage, error) {
	store := m.Storage().GetStore(storage.DomainMemory)
	if store == nil {
		return nil, mastraerror.NewMastraBaseError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_MEMORY_STORAGE_NOT_AVAILABLE",
			Domain:   mastraerror.ErrorDomainMastraMemory,
			Category: mastraerror.ErrorCategorySystem,
			Text:     "Memory storage is not supported by this storage adapter",
		})
	}
	memStore, ok := store.(memorystorage.MemoryStorage)
	if !ok {
		return nil, fmt.Errorf("memory storage does not implement MemoryStorage interface")
	}
	return memStore, nil
}

// GetThreadById retrieves a specific thread by its ID.
func (m *MockMemory) GetThreadById(ctx context.Context, threadId string) (*StorageThreadType, error) {
	memStore, err := m.getMemoryStore(ctx)
	if err != nil {
		return nil, err
	}
	result, err := memStore.GetThreadByID(ctx, threadId)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	thread := mapToStorageThreadType(result)
	return &thread, nil
}

// SaveThread saves or updates a thread.
func (m *MockMemory) SaveThread(ctx context.Context, thread StorageThreadType, memoryConfig *MemoryConfig) (*StorageThreadType, error) {
	memStore, err := m.getMemoryStore(ctx)
	if err != nil {
		return nil, err
	}
	result, err := memStore.SaveThread(ctx, storageThreadTypeToMap(thread))
	if err != nil {
		return nil, err
	}
	saved := mapToStorageThreadType(result)
	return &saved, nil
}

// SaveMessages saves messages to a thread.
func (m *MockMemory) SaveMessages(ctx context.Context, messages []MastraDBMessage, memoryConfig *MemoryConfig) (*SaveMessagesResult, error) {
	memStore, err := m.getMemoryStore(ctx)
	if err != nil {
		return nil, err
	}

	// MastraDBMessage in both packages is map[string]any, direct assignment works.
	savedMsgs, err := memStore.SaveMessages(ctx, messages)
	if err != nil {
		return nil, err
	}

	resultMessages := savedMsgs

	return &SaveMessagesResult{
		Messages: resultMessages,
	}, nil
}

// ListThreads lists threads with optional filtering.
func (m *MockMemory) ListThreads(ctx context.Context, args StorageListThreadsInput) (StorageListThreadsOutput, error) {
	memStore, err := m.getMemoryStore(ctx)
	if err != nil {
		return nil, err
	}
	storageArgs := mapToStorageListThreadsInput(args)
	result, err := memStore.ListThreads(ctx, storageArgs)
	if err != nil {
		return nil, err
	}
	return storageListThreadsOutputToMap(result), nil
}

// Recall retrieves messages for a specific thread with optional semantic recall.
func (m *MockMemory) Recall(ctx context.Context, args RecallArgs) (*RecallResult, error) {
	memStore, err := m.getMemoryStore(ctx)
	if err != nil {
		return nil, err
	}

	// Convert to storage domain type.
	listMessagesArgs := mapToStorageListMessagesInput(args.StorageListMessagesInput)

	result, err := memStore.ListMessages(ctx, listMessagesArgs)
	if err != nil {
		return nil, err
	}

	messages := result.Messages
	total := result.Total
	page := result.Page
	hasMore := result.HasMore

	return &RecallResult{
		Messages: messages,
		Total:    total,
		Page:     page,
		PerPage:  result.PerPage,
		HasMore:  hasMore,
	}, nil
}

// DeleteThread deletes a thread by ID.
func (m *MockMemory) DeleteThread(ctx context.Context, threadId string) error {
	memStore, err := m.getMemoryStore(ctx)
	if err != nil {
		return err
	}
	return memStore.DeleteThread(ctx, threadId)
}

// DeleteMessages deletes messages by their IDs.
func (m *MockMemory) DeleteMessages(ctx context.Context, messageIds MessageDeleteInput) error {
	memStore, err := m.getMemoryStore(ctx)
	if err != nil {
		return err
	}
	return memStore.DeleteMessages(ctx, messageIds)
}

// GetWorkingMemory retrieves working memory for a specific thread.
func (m *MockMemory) GetWorkingMemory(ctx context.Context, opts GetWorkingMemoryOpts) (*string, error) {
	mergedConfig := m.GetMergedThreadConfig(&MemoryConfig{})
	if opts.MemoryConfig != nil {
		mergedConfig = m.GetMergedThreadConfig(opts.MemoryConfig)
	}

	workingMemoryConfig := mergedConfig.WorkingMemory
	if workingMemoryConfig == nil || !workingMemoryConfig.Enabled {
		return nil, nil
	}

	scope := workingMemoryConfig.Scope
	if scope == "" {
		scope = WorkingMemoryScopeResource
	}

	id := opts.ResourceID
	if scope == WorkingMemoryScopeThread {
		id = opts.ThreadID
	}

	if id == "" {
		return nil, nil
	}

	memStore, err := m.getMemoryStore(ctx)
	if err != nil {
		return nil, err
	}

	resource, err := memStore.GetResourceByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if resource == nil {
		return nil, nil
	}

	// Extract workingMemory field from the resource (StorageResourceType is map[string]any)
	wm, ok := resource["workingMemory"].(string)
	if !ok || wm == "" {
		return nil, nil
	}

	return &wm, nil
}

// ListTools returns tools available to the agent from this memory instance.
//
// Ported from: packages/core/src/memory/mock.ts listTools (lines 149-212)
//
// In TS, this creates a tool via createTool with a zod input schema and an execute
// function that handles thread creation, resource validation, and working memory updates.
// Since the tools package is not fully ported, we return a map[string]any tool definition
// that includes the input schema and a Go execute function matching the TS behavior.
func (m *MockMemory) ListTools(config *MemoryConfig) map[string]ToolAction {
	mergedConfig := m.GetMergedThreadConfig(config)
	if mergedConfig.WorkingMemory == nil || !mergedConfig.WorkingMemory.Enabled {
		return map[string]ToolAction{}
	}

	// Build input schema equivalent to z.object({ memory: z.string() })
	inputSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"memory": map[string]any{
				"type": "string",
			},
		},
		"required": []string{"memory"},
	}

	return map[string]ToolAction{
		"updateWorkingMemory": map[string]any{
			"id":          "update-working-memory",
			"description": "Update the working memory with new information. Any data not included will be overwritten.",
			"inputSchema": inputSchema,
			"execute": func(inputData map[string]any, toolContext map[string]any) (map[string]any, error) {
				// Extract context values
				var threadId, resourceId string
				if agent, ok := toolContext["agent"].(map[string]any); ok {
					threadId, _ = agent["threadId"].(string)
					resourceId, _ = agent["resourceId"].(string)
				}

				// Determine scope
				scope := WorkingMemoryScopeResource
				if mergedConfig.WorkingMemory.Scope != "" {
					scope = mergedConfig.WorkingMemory.Scope
				}

				if scope == WorkingMemoryScopeThread && threadId == "" {
					return nil, fmt.Errorf("Thread ID is required for thread-scoped working memory updates")
				}
				if scope == WorkingMemoryScopeResource && resourceId == "" {
					return nil, fmt.Errorf("Resource ID is required for resource-scoped working memory updates")
				}

				// Ensure thread exists
				if threadId != "" {
					thread, err := m.GetThreadById(context.Background(), threadId)
					if err != nil {
						return nil, err
					}

					if thread == nil {
						_, err = m.CreateThread(context.Background(), CreateThreadOpts{
							ThreadID:     threadId,
							ResourceID:   resourceId,
							MemoryConfig: config,
						})
						if err != nil {
							return nil, err
						}
					} else if thread.ResourceID != "" && resourceId != "" && thread.ResourceID != resourceId {
						return nil, fmt.Errorf(
							"Thread with id %s resourceId does not match the current resourceId %s",
							threadId, resourceId,
						)
					}
				}

				// Normalize working memory to string
				memoryValue, _ := inputData["memory"].(string)
				if memoryValue == "" {
					// Try JSON marshaling if it's not a string
					if rawMemory, ok := inputData["memory"]; ok {
						bytes, err := json.Marshal(rawMemory)
						if err == nil {
							memoryValue = string(bytes)
						}
					}
				}

				// Update working memory
				err := m.UpdateWorkingMemory(context.Background(), UpdateWorkingMemoryOpts{
					ThreadID:      threadId,
					ResourceID:    resourceId,
					WorkingMemory: memoryValue,
					MemoryConfig:  config,
				})
				if err != nil {
					return nil, err
				}

				return map[string]any{"success": true}, nil
			},
		},
	}
}

// GetWorkingMemoryTemplate gets the working memory template.
func (m *MockMemory) GetWorkingMemoryTemplate(ctx context.Context, memoryConfig *MemoryConfig) (*WorkingMemoryTemplate, error) {
	mergedConfig := m.GetMergedThreadConfig(memoryConfig)
	workingMemoryConfig := mergedConfig.WorkingMemory

	if workingMemoryConfig == nil || !workingMemoryConfig.Enabled {
		return nil, nil
	}

	if workingMemoryConfig.Template != "" {
		return &WorkingMemoryTemplate{
			Format:  WorkingMemoryFormatMarkdown,
			Content: workingMemoryConfig.Template,
		}, nil
	}

	if workingMemoryConfig.Schema != nil {
		// In TS: converts ZodObject or JSONSchema7 to JSON string.
		// In Go: attempt JSON serialization of the schema.
		schemaBytes, err := json.Marshal(workingMemoryConfig.Schema)
		if err != nil {
			return nil, fmt.Errorf("error converting schema: %w", err)
		}
		return &WorkingMemoryTemplate{
			Format:  WorkingMemoryFormatJSON,
			Content: string(schemaBytes),
		}, nil
	}

	return nil, nil
}

// UpdateWorkingMemory updates working memory for a thread.
func (m *MockMemory) UpdateWorkingMemory(ctx context.Context, opts UpdateWorkingMemoryOpts) error {
	mergedConfig := m.GetMergedThreadConfig(opts.MemoryConfig)
	workingMemoryConfig := mergedConfig.WorkingMemory

	if workingMemoryConfig == nil || !workingMemoryConfig.Enabled {
		return nil
	}

	scope := workingMemoryConfig.Scope
	if scope == "" {
		scope = WorkingMemoryScopeResource
	}

	id := opts.ResourceID
	if scope == WorkingMemoryScopeThread {
		id = opts.ThreadID
	}

	if id == "" {
		return fmt.Errorf("cannot update working memory: %s ID is required", scope)
	}

	memStore, err := m.getMemoryStore(ctx)
	if err != nil {
		return err
	}

	_, err = memStore.UpdateResource(ctx, memorystorage.UpdateResourceInput{
		ResourceID:    id,
		WorkingMemory: &opts.WorkingMemory,
	})
	return err
}

// ExperimentalUpdateWorkingMemoryVNext is the experimental v-next working memory update.
func (m *MockMemory) ExperimentalUpdateWorkingMemoryVNext(ctx context.Context, opts ExperimentalUpdateWorkingMemoryVNextOpts) (*ExperimentalUpdateResult, error) {
	err := m.UpdateWorkingMemory(ctx, UpdateWorkingMemoryOpts{
		ThreadID:      opts.ThreadID,
		ResourceID:    opts.ResourceID,
		WorkingMemory: opts.WorkingMemory,
		MemoryConfig:  opts.MemoryConfig,
	})
	if err != nil {
		return &ExperimentalUpdateResult{
			Success: false,
			Reason:  err.Error(),
		}, nil
	}
	return &ExperimentalUpdateResult{
		Success: true,
		Reason:  "Working memory updated successfully",
	}, nil
}

// CloneThread clones a thread with all its messages.
func (m *MockMemory) CloneThread(ctx context.Context, args StorageCloneThreadInput) (StorageCloneThreadOutput, error) {
	memStore, err := m.getMemoryStore(ctx)
	if err != nil {
		return nil, err
	}
	storageArgs := mapToStorageCloneThreadInput(args)
	result, err := memStore.CloneThread(ctx, storageArgs)
	if err != nil {
		return nil, err
	}
	return storageCloneThreadOutputToMap(result), nil
}

// CreateThread creates a new thread (overrides base to use mock's SaveThread).
func (m *MockMemory) CreateThread(ctx context.Context, opts CreateThreadOpts) (*StorageThreadType, error) {
	// Use base implementation but override saveThread behavior
	return m.MastraMemoryBase.createThreadWithSaver(ctx, opts, func(ctx context.Context, thread StorageThreadType, config *MemoryConfig) (*StorageThreadType, error) {
		return m.SaveThread(ctx, thread, config)
	})
}

// createThreadWithSaver is a helper on the base that allows concrete types to inject their SaveThread.
func (m *MastraMemoryBase) createThreadWithSaver(ctx context.Context, opts CreateThreadOpts, saver func(context.Context, StorageThreadType, *MemoryConfig) (*StorageThreadType, error)) (*StorageThreadType, error) {
	threadID := opts.ThreadID
	if threadID == "" {
		source := aktypes.IdGeneratorSourceMemory
		threadID = m.GenerateId(&IdGeneratorContext{
			IdType:     aktypes.IdTypeThread,
			Source:     &source,
			ResourceId: &opts.ResourceID,
		})
	}

	now := time.Now()
	thread := StorageThreadType{
		ID:         threadID,
		Title:      opts.Title,
		ResourceID: opts.ResourceID,
		CreatedAt:  now,
		UpdatedAt:  now,
		Metadata:   opts.Metadata,
	}

	shouldSave := true
	if opts.SaveThread != nil {
		shouldSave = *opts.SaveThread
	}

	if shouldSave {
		return saver(ctx, thread, opts.MemoryConfig)
	}
	return &thread, nil
}
