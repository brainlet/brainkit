// Ported from: packages/core/src/processors/memory/working-memory.test.ts
package memory

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/processors"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	storagememory "github.com/brainlet/brainkit/agent-kit/core/storage/domains/memory"
)

// ---------------------------------------------------------------------------
// Mock storage for WorkingMemory tests
// ---------------------------------------------------------------------------

type mockStorageForWM struct {
	getThreadByIDResult StorageThreadType
	getThreadByIDErr    error
	getThreadByIDCalls  []string

	getResourceByIDResult StorageResourceType
	getResourceByIDErr    error
	getResourceByIDCalls  []string
}

func (m *mockStorageForWM) ListMessages(_ context.Context, _ storagememory.StorageListMessagesInput) (StorageListMessagesOutput, error) {
	return StorageListMessagesOutput{}, nil
}
func (m *mockStorageForWM) GetThreadByID(_ context.Context, threadID string) (StorageThreadType, error) {
	m.getThreadByIDCalls = append(m.getThreadByIDCalls, threadID)
	if m.getThreadByIDErr != nil {
		return nil, m.getThreadByIDErr
	}
	return m.getThreadByIDResult, nil
}
func (m *mockStorageForWM) SaveThread(_ context.Context, thread StorageThreadType) (StorageThreadType, error) {
	return thread, nil
}
func (m *mockStorageForWM) UpdateThread(_ context.Context, input UpdateThreadInput) (StorageThreadType, error) {
	return StorageThreadType{"id": input.ID}, nil
}
func (m *mockStorageForWM) SaveMessages(_ context.Context, messages []processors.MastraDBMessage) ([]processors.MastraDBMessage, error) {
	return messages, nil
}
func (m *mockStorageForWM) GetResourceByID(_ context.Context, resourceID string) (StorageResourceType, error) {
	m.getResourceByIDCalls = append(m.getResourceByIDCalls, resourceID)
	if m.getResourceByIDErr != nil {
		return nil, m.getResourceByIDErr
	}
	return m.getResourceByIDResult, nil
}

func setupWMRequestContext(threadID, resourceID string) *requestcontext.RequestContext {
	rc := requestcontext.NewRequestContext()
	rc.Set("MastraMemory", map[string]any{
		"thread": map[string]any{
			"id":         threadID,
			"resourceId": resourceID,
			"title":      "Test",
			"createdAt":  time.Now().Format(time.RFC3339),
			"updatedAt":  time.Now().Format(time.RFC3339),
		},
		"resourceId": resourceID,
	})
	return rc
}

func setupWMRequestContextWithReadOnly(threadID, resourceID string) *requestcontext.RequestContext {
	rc := requestcontext.NewRequestContext()
	rc.Set("MastraMemory", map[string]any{
		"thread": map[string]any{
			"id":         threadID,
			"resourceId": resourceID,
			"title":      "Test",
			"createdAt":  time.Now().Format(time.RFC3339),
			"updatedAt":  time.Now().Format(time.RFC3339),
		},
		"resourceId":   resourceID,
		"memoryConfig": map[string]any{"readOnly": true},
	})
	return rc
}

func makeUserMessage(id, text string) processors.MastraDBMessage {
	return processors.MastraDBMessage{
		MastraMessageShared: processors.MastraMessageShared{
			ID:        id,
			Role:      "user",
			CreatedAt: time.Now(),
		},
		Content: processors.MastraMessageContentV2{
			Format: 2,
			Parts:  []processors.MastraMessagePart{{Type: "text", Text: text}},
		},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestWorkingMemory(t *testing.T) {
	t.Run("Input Processing", func(t *testing.T) {
		t.Run("should use thread-scoped working memory", func(t *testing.T) {
			workingMemoryData := "# User Info\n- Name: John\n- Preference: Dark mode"

			mockStorage := &mockStorageForWM{
				getThreadByIDResult: StorageThreadType{
					"id":         "thread-123",
					"resourceId": "resource-1",
					"title":      "Test Thread",
					"metadata":   map[string]any{"workingMemory": workingMemoryData},
					"createdAt":  time.Now(),
					"updatedAt":  time.Now(),
				},
			}

			proc := NewWorkingMemory(WorkingMemoryProcessorConfig{
				Storage: mockStorage,
				Scope:   "thread",
			})

			messages := []processors.MastraDBMessage{makeUserMessage("msg-1", "Hello")}
			rc := setupWMRequestContext("thread-123", "resource-1")
			ml := &processors.MessageList{}

			_, resultML, _, err := proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:    messages,
					MessageList: ml,
					ProcessorContext: processors.ProcessorContext{
						RequestContext: rc,
					},
				},
			})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resultML != ml {
				t.Error("expected same MessageList back")
			}
			if len(mockStorage.getThreadByIDCalls) == 0 {
				t.Error("expected GetThreadByID to be called")
			}
			if mockStorage.getThreadByIDCalls[0] != "thread-123" {
				t.Errorf("expected threadID='thread-123', got %q", mockStorage.getThreadByIDCalls[0])
			}
		})

		t.Run("should use resource-scoped working memory", func(t *testing.T) {
			workingMemoryData := "# Project Context\n- Status: In Progress"

			mockStorage := &mockStorageForWM{
				getResourceByIDResult: StorageResourceType{
					"id":            "resource-456",
					"workingMemory": workingMemoryData,
				},
			}

			proc := NewWorkingMemory(WorkingMemoryProcessorConfig{
				Storage: mockStorage,
				Scope:   "resource",
			})

			messages := []processors.MastraDBMessage{makeUserMessage("msg-1", "What is the status?")}
			rc := setupWMRequestContext("thread-1", "resource-456")
			ml := &processors.MessageList{}

			_, resultML, _, err := proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:    messages,
					MessageList: ml,
					ProcessorContext: processors.ProcessorContext{
						RequestContext: rc,
					},
				},
			})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resultML != ml {
				t.Error("expected same MessageList back")
			}
			if len(mockStorage.getResourceByIDCalls) == 0 {
				t.Error("expected GetResourceByID to be called")
			}
			if mockStorage.getResourceByIDCalls[0] != "resource-456" {
				t.Errorf("expected resourceID='resource-456', got %q", mockStorage.getResourceByIDCalls[0])
			}
		})

		t.Run("should return original messages when no threadId or resourceId", func(t *testing.T) {
			mockStorage := &mockStorageForWM{}

			proc := NewWorkingMemory(WorkingMemoryProcessorConfig{
				Storage: mockStorage,
				Scope:   "thread",
			})

			messages := []processors.MastraDBMessage{makeUserMessage("msg-1", "Hello")}
			emptyRC := requestcontext.NewRequestContext()
			ml := &processors.MessageList{}

			_, resultML, _, err := proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:    messages,
					MessageList: ml,
					ProcessorContext: processors.ProcessorContext{
						RequestContext: emptyRC,
					},
				},
			})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resultML != ml {
				t.Error("expected same MessageList back")
			}
			if len(mockStorage.getThreadByIDCalls) > 0 {
				t.Error("expected GetThreadByID not to be called")
			}
			if len(mockStorage.getResourceByIDCalls) > 0 {
				t.Error("expected GetResourceByID not to be called")
			}
		})

		t.Run("should default to resource scope when scope not specified", func(t *testing.T) {
			mockStorage := &mockStorageForWM{
				getResourceByIDResult: StorageResourceType{
					"id":            "resource-1",
					"workingMemory": "Test data",
				},
			}

			proc := NewWorkingMemory(WorkingMemoryProcessorConfig{
				Storage: mockStorage,
				// scope not specified, should default to 'resource'
			})

			messages := []processors.MastraDBMessage{makeUserMessage("msg-1", "Hello")}
			rc := setupWMRequestContext("thread-123", "resource-1")
			ml := &processors.MessageList{}

			proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:    messages,
					MessageList: ml,
					ProcessorContext: processors.ProcessorContext{
						RequestContext: rc,
					},
				},
			})

			if len(mockStorage.getResourceByIDCalls) == 0 {
				t.Error("expected GetResourceByID to be called (default scope is resource)")
			}
			if len(mockStorage.getThreadByIDCalls) > 0 {
				t.Error("expected GetThreadByID not to be called when scope defaults to resource")
			}
		})
	})

	t.Run("Instruction Generation", func(t *testing.T) {
		t.Run("getWorkingMemoryToolInstruction should contain key elements", func(t *testing.T) {
			proc := NewWorkingMemory(WorkingMemoryProcessorConfig{
				Storage: &mockStorageForWM{},
			})

			template := WorkingMemoryTemplate{
				Format:  "markdown",
				Content: "# Template\n- Field:",
			}

			instruction := proc.getWorkingMemoryToolInstruction(template, "some data")

			if !strings.Contains(instruction, "WORKING_MEMORY_SYSTEM_INSTRUCTION") {
				t.Error("expected instruction to contain WORKING_MEMORY_SYSTEM_INSTRUCTION")
			}
			if !strings.Contains(instruction, "some data") {
				t.Error("expected instruction to contain working memory data")
			}
			if !strings.Contains(instruction, "updateWorkingMemory") {
				t.Error("expected instruction to contain updateWorkingMemory reference")
			}
			if !strings.Contains(instruction, "Markdown") {
				t.Error("expected instruction to contain format name 'Markdown'")
			}
		})

		t.Run("getWorkingMemoryToolInstruction JSON format should not mention markdown rules", func(t *testing.T) {
			proc := NewWorkingMemory(WorkingMemoryProcessorConfig{
				Storage: &mockStorageForWM{},
			})

			jsonContent, _ := json.Marshal(map[string]any{
				"user": map[string]any{"name": "", "email": ""},
			})

			template := WorkingMemoryTemplate{
				Format:  "json",
				Content: string(jsonContent),
			}

			instruction := proc.getWorkingMemoryToolInstruction(template, "")

			if !strings.Contains(instruction, "JSON") {
				t.Error("expected instruction to contain 'JSON'")
			}
			if !strings.Contains(instruction, "Use JSON format for all data") {
				t.Error("expected instruction to contain JSON format guidance")
			}
			if strings.Contains(instruction, "IMPORTANT: When calling updateWorkingMemory") {
				t.Error("expected JSON instruction not to contain markdown-specific rules")
			}
		})

		t.Run("getWorkingMemoryToolInstructionVNext should contain VNext-specific content", func(t *testing.T) {
			proc := NewWorkingMemory(WorkingMemoryProcessorConfig{
				Storage:  &mockStorageForWM{},
				UseVNext: true,
			})

			template := WorkingMemoryTemplate{
				Format:  "markdown",
				Content: defaultWorkingMemoryTemplate,
			}

			instruction := proc.getWorkingMemoryToolInstructionVNext(template, "Some data")

			if !strings.Contains(instruction, "If your memory has not changed") {
				t.Error("expected VNext instruction to contain 'If your memory has not changed'")
			}
			if !strings.Contains(instruction, "Information not being relevant to the current conversation") {
				t.Error("expected VNext instruction to contain relevance guidance")
			}
		})

		t.Run("getReadOnlyWorkingMemoryInstruction should contain read-only markers", func(t *testing.T) {
			proc := NewWorkingMemory(WorkingMemoryProcessorConfig{
				Storage:  &mockStorageForWM{},
				ReadOnly: true,
			})

			template := WorkingMemoryTemplate{
				Format:  "markdown",
				Content: "# Template",
			}

			instruction := proc.getReadOnlyWorkingMemoryInstruction(template, "# User Info\n- Name: John")

			if !strings.Contains(instruction, "WORKING_MEMORY_SYSTEM_INSTRUCTION (READ-ONLY)") {
				t.Error("expected read-only instruction header")
			}
			if !strings.Contains(instruction, "# User Info") {
				t.Error("expected instruction to contain working memory data")
			}
			if !strings.Contains(instruction, "read-only in the current session") {
				t.Error("expected read-only notice")
			}
			if !strings.Contains(instruction, "Act naturally") {
				t.Error("expected 'Act naturally' guidance")
			}
			if strings.Contains(instruction, "updateWorkingMemory") {
				t.Error("expected read-only instruction NOT to contain updateWorkingMemory")
			}
			if strings.Contains(instruction, "Store and update") {
				t.Error("expected read-only instruction NOT to contain 'Store and update'")
			}
		})

		t.Run("getReadOnlyWorkingMemoryInstruction should show fallback when no data", func(t *testing.T) {
			proc := NewWorkingMemory(WorkingMemoryProcessorConfig{
				Storage:  &mockStorageForWM{},
				ReadOnly: true,
			})

			template := WorkingMemoryTemplate{
				Format:  "markdown",
				Content: "# Template",
			}

			instruction := proc.getReadOnlyWorkingMemoryInstruction(template, "")

			if !strings.Contains(instruction, "No working memory data available.") {
				t.Error("expected fallback message when no data")
			}
		})
	})

	t.Run("Template Resolution", func(t *testing.T) {
		t.Run("should use default template when no template provided", func(t *testing.T) {
			proc := NewWorkingMemory(WorkingMemoryProcessorConfig{
				Storage: &mockStorageForWM{},
			})

			template := proc.resolveTemplate(nil)

			if template.Format != "markdown" {
				t.Errorf("expected format='markdown', got %q", template.Format)
			}
			if !strings.Contains(template.Content, "# User Information") {
				t.Error("expected default template to contain '# User Information'")
			}
		})

		t.Run("should use custom template when provided", func(t *testing.T) {
			customTemplate := &WorkingMemoryTemplate{
				Format:  "markdown",
				Content: "# Custom Template\n- Field 1:\n- Field 2:",
			}

			proc := NewWorkingMemory(WorkingMemoryProcessorConfig{
				Storage:  &mockStorageForWM{},
				Template: customTemplate,
			})

			template := proc.resolveTemplate(nil)

			if template.Content != customTemplate.Content {
				t.Errorf("expected custom template content, got %q", template.Content)
			}
		})

		t.Run("should use JSON format template", func(t *testing.T) {
			jsonTemplate := &WorkingMemoryTemplate{
				Format:  "json",
				Content: `{"user":{"name":"","email":""}}`,
			}

			proc := NewWorkingMemory(WorkingMemoryProcessorConfig{
				Storage:  &mockStorageForWM{},
				Template: jsonTemplate,
			})

			template := proc.resolveTemplate(nil)

			if template.Format != "json" {
				t.Errorf("expected format='json', got %q", template.Format)
			}
		})
	})

	t.Run("DefaultWorkingMemoryTemplate", func(t *testing.T) {
		t.Run("should return the default template content", func(t *testing.T) {
			content := DefaultWorkingMemoryTemplate()
			if !strings.Contains(content, "# User Information") {
				t.Error("expected default template to contain '# User Information'")
			}
			if !strings.Contains(content, "First Name") {
				t.Error("expected default template to contain 'First Name'")
			}
		})
	})

	t.Run("generateEmptyFromSchema", func(t *testing.T) {
		proc := NewWorkingMemory(WorkingMemoryProcessorConfig{
			Storage: &mockStorageForWM{},
		})

		t.Run("should generate empty values for simple schema", func(t *testing.T) {
			schema := map[string]any{
				"name":  map[string]any{"type": "string"},
				"count": map[string]any{"type": "number"},
			}

			result := proc.generateEmptyFromSchema(schema)
			if result["name"] != "" {
				t.Errorf("expected empty string for 'name', got %v", result["name"])
			}
			if result["count"] != "" {
				t.Errorf("expected empty string for 'count', got %v", result["count"])
			}
		})

		t.Run("should handle nested objects", func(t *testing.T) {
			schema := map[string]any{
				"user": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string"},
					},
				},
			}

			result := proc.generateEmptyFromSchema(schema)
			userObj, ok := result["user"].(map[string]any)
			if !ok {
				t.Fatal("expected 'user' to be a map")
			}
			if userObj["name"] != "" {
				t.Errorf("expected empty string for user.name, got %v", userObj["name"])
			}
		})

		t.Run("should handle arrays", func(t *testing.T) {
			schema := map[string]any{
				"items": map[string]any{"type": "array"},
			}

			result := proc.generateEmptyFromSchema(schema)
			items, ok := result["items"].([]any)
			if !ok {
				t.Fatal("expected 'items' to be []any")
			}
			if len(items) != 0 {
				t.Errorf("expected empty array, got %d items", len(items))
			}
		})

		t.Run("should return nil for nil schema", func(t *testing.T) {
			result := proc.generateEmptyFromSchema(nil)
			if result != nil {
				t.Errorf("expected nil for nil schema, got %v", result)
			}
		})
	})
}
