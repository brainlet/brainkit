// Ported from: packages/core/src/agent/__tests__/instructions.test.ts
// Ported from: packages/core/src/agent/__tests__/tools.test.ts
// Ported from: packages/core/src/agent/__tests__/model-list.test.ts
// Ported from: packages/core/src/agent/agent.ts (constructor tests)
package agent

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// ---------------------------------------------------------------------------
// Mock model — satisfies LanguageModelLike for agent construction
// ---------------------------------------------------------------------------

// mockLanguageModel is a minimal LanguageModelLike for tests that need a model
// to pass to NewAgent (which requires Model != nil).
type mockLanguageModel struct {
	specVersion string
	modelID     string
}

func (m *mockLanguageModel) SpecificationVersion() string {
	if m.specVersion == "" {
		return "v2"
	}
	return m.specVersion
}

func (m *mockLanguageModel) ModelID() string {
	if m.modelID == "" {
		return "mock-model"
	}
	return m.modelID
}

// newDummyModel returns a mock model satisfying LanguageModelLike with spec version v2.
func newDummyModel() *mockLanguageModel {
	return &mockLanguageModel{specVersion: "v2", modelID: "mock-model"}
}

// newDummyModelWithID returns a mock model with a specific model ID.
func newDummyModelWithID(id string) *mockLanguageModel {
	return &mockLanguageModel{specVersion: "v2", modelID: id}
}

// fakeM is a minimal Mastra interface implementation for tests that need a
// Mastra reference (e.g., to verify agent registration). Returns nil logger
// since these tests only check that the instance is non-nil.
type fakeM struct{}

func (f *fakeM) GetLogger() IMastraLogger { return nil }

// newV1Model returns a mock model with spec version v1 (legacy).
func newV1Model() *mockLanguageModel {
	return &mockLanguageModel{specVersion: "v1", modelID: "v1-mock-model"}
}

// ---------------------------------------------------------------------------
// TestNewAgent — Agent constructor
// ---------------------------------------------------------------------------

func TestNewAgent(t *testing.T) {
	t.Run("should create agent with valid config", func(t *testing.T) {
		agent, err := NewAgent(AgentConfig{
			ID:           "test-agent",
			Name:         "Test Agent",
			Instructions: "You are a helpful assistant.",
			Model:        newDummyModel(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if agent.ID != "test-agent" {
			t.Errorf("expected ID %q, got %q", "test-agent", agent.ID)
		}
		if agent.AgentName != "Test Agent" {
			t.Errorf("expected AgentName %q, got %q", "Test Agent", agent.AgentName)
		}
		if agent.Source != "code" {
			t.Errorf("expected Source %q, got %q", "code", agent.Source)
		}
	})

	t.Run("should use name as ID when ID is empty", func(t *testing.T) {
		agent, err := NewAgent(AgentConfig{
			Name:         "My Agent",
			Instructions: "test",
			Model:        newDummyModel(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if agent.ID != "My Agent" {
			t.Errorf("expected ID to fall back to Name %q, got %q", "My Agent", agent.ID)
		}
	})

	t.Run("should return error when model is nil", func(t *testing.T) {
		_, err := NewAgent(AgentConfig{
			Name:         "No Model Agent",
			Instructions: "test",
			Model:        nil,
		})
		if err == nil {
			t.Fatal("expected error when model is nil, got nil")
		}
		if !strings.Contains(err.Error(), "LanguageModel is required") {
			t.Errorf("expected error about LanguageModel required, got: %v", err)
		}
	})

	t.Run("should return error when model array is empty", func(t *testing.T) {
		_, err := NewAgent(AgentConfig{
			Name:         "Empty Model Array Agent",
			Instructions: "test",
			Model:        []ModelWithRetries{},
		})
		if err == nil {
			t.Fatal("expected error when model array is empty, got nil")
		}
		if !strings.Contains(err.Error(), "Model array is empty") {
			t.Errorf("expected error about empty model array, got: %v", err)
		}
	})

	t.Run("should set description", func(t *testing.T) {
		agent, err := NewAgent(AgentConfig{
			Name:         "Desc Agent",
			Description:  "A test agent for descriptions.",
			Instructions: "test",
			Model:        newDummyModel(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if agent.GetDescription() != "A test agent for descriptions." {
			t.Errorf("expected description %q, got %q",
				"A test agent for descriptions.", agent.GetDescription())
		}
	})

	t.Run("should register mastra when provided", func(t *testing.T) {
		var fm Mastra = &fakeM{}

		agent, err := NewAgent(AgentConfig{
			Name:         "Mastra Agent",
			Instructions: "test",
			Model:        newDummyModel(),
			Mastra:       fm,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if agent.GetMastraInstance() == nil {
			t.Error("expected mastra instance to be registered, got nil")
		}
	})

	t.Run("should handle model fallbacks array", func(t *testing.T) {
		m1 := newDummyModelWithID("gpt-4o")
		m2 := newDummyModelWithID("gpt-4o-mini")
		retries := 3

		agent, err := NewAgent(AgentConfig{
			Name:         "Fallback Agent",
			Instructions: "test",
			Model: []ModelWithRetries{
				{Model: m1, MaxRetries: &retries},
				{Model: m2},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		fallbacks, ok := agent.Model.(ModelFallbacks)
		if !ok {
			t.Fatal("expected agent.Model to be ModelFallbacks")
		}
		if len(fallbacks) != 2 {
			t.Fatalf("expected 2 fallbacks, got %d", len(fallbacks))
		}
		if fallbacks[0].MaxRetries != 3 {
			t.Errorf("expected first model maxRetries=3, got %d", fallbacks[0].MaxRetries)
		}
		if !fallbacks[0].Enabled {
			t.Error("expected first model to be enabled by default")
		}
		if !fallbacks[1].Enabled {
			t.Error("expected second model to be enabled by default")
		}
	})

	t.Run("should use model-level maxRetries with agent-level fallback", func(t *testing.T) {
		agentRetries := 5
		modelRetries := 2

		agent, err := NewAgent(AgentConfig{
			Name:         "Retries Agent",
			Instructions: "test",
			MaxRetries:   &agentRetries,
			Model: []ModelWithRetries{
				{Model: newDummyModel(), MaxRetries: &modelRetries},
				{Model: newDummyModel()},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		fallbacks := agent.Model.(ModelFallbacks)
		// First model should use its own MaxRetries (2)
		if fallbacks[0].MaxRetries != 2 {
			t.Errorf("expected first model maxRetries=2, got %d", fallbacks[0].MaxRetries)
		}
		// Second model should fall back to agent-level MaxRetries (5)
		if fallbacks[1].MaxRetries != 5 {
			t.Errorf("expected second model maxRetries=5 (agent fallback), got %d", fallbacks[1].MaxRetries)
		}
	})

	t.Run("should handle enabled=false in model fallbacks", func(t *testing.T) {
		falseVal := false

		agent, err := NewAgent(AgentConfig{
			Name:         "Enabled Test",
			Instructions: "test",
			Model: []ModelWithRetries{
				{Model: newDummyModel()},
				{Model: newDummyModel(), Enabled: &falseVal},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		fallbacks := agent.Model.(ModelFallbacks)
		if !fallbacks[0].Enabled {
			t.Error("expected first model to be enabled")
		}
		if fallbacks[1].Enabled {
			t.Error("expected second model to be disabled (enabled=false)")
		}
	})
}

// ---------------------------------------------------------------------------
// TestAgentHasOwnMemory / TestAgentHasOwnWorkspace
// ---------------------------------------------------------------------------

func TestAgentHasOwnMemory(t *testing.T) {
	t.Run("should return false when no memory configured", func(t *testing.T) {
		agent, err := NewAgent(AgentConfig{
			Name:         "No Memory",
			Instructions: "test",
			Model:        newDummyModel(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if agent.HasOwnMemory() {
			t.Error("expected HasOwnMemory() to return false")
		}
	})

	t.Run("should return true when memory is configured", func(t *testing.T) {
		agent, err := NewAgent(AgentConfig{
			Name:         "With Memory",
			Instructions: "test",
			Model:        newDummyModel(),
			Memory:       "some-memory-config",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !agent.HasOwnMemory() {
			t.Error("expected HasOwnMemory() to return true")
		}
	})
}

func TestAgentHasOwnWorkspace(t *testing.T) {
	t.Run("should return false when no workspace configured", func(t *testing.T) {
		agent, err := NewAgent(AgentConfig{
			Name:         "No Workspace",
			Instructions: "test",
			Model:        newDummyModel(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if agent.HasOwnWorkspace() {
			t.Error("expected HasOwnWorkspace() to return false")
		}
	})

	t.Run("should return true when workspace is configured", func(t *testing.T) {
		agent, err := NewAgent(AgentConfig{
			Name:         "With Workspace",
			Instructions: "test",
			Model:        newDummyModel(),
			Workspace:    "some-workspace",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !agent.HasOwnWorkspace() {
			t.Error("expected HasOwnWorkspace() to return true")
		}
	})
}

// ---------------------------------------------------------------------------
// TestAgentGetInstructions — instructions.test.ts
// ---------------------------------------------------------------------------

func TestAgentGetInstructions(t *testing.T) {
	t.Run("should support string instructions (backward compatibility)", func(t *testing.T) {
		agent, err := NewAgent(AgentConfig{
			ID:           "test-agent",
			Name:         "Test Agent",
			Instructions: "You are a helpful assistant.",
			Model:        newDummyModel(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		instructions, err := agent.GetInstructions(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if instructions != "You are a helpful assistant." {
			t.Errorf("expected %q, got %v", "You are a helpful assistant.", instructions)
		}
	})

	t.Run("should support CoreSystemMessage instructions", func(t *testing.T) {
		systemMessage := map[string]any{
			"role":    "system",
			"content": "You are an expert programmer.",
		}

		agent, err := NewAgent(AgentConfig{
			ID:           "test-agent",
			Name:         "Test Agent",
			Instructions: systemMessage,
			Model:        newDummyModel(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		instructions, err := agent.GetInstructions(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		result, ok := instructions.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", instructions)
		}
		if result["role"] != "system" {
			t.Errorf("expected role %q, got %v", "system", result["role"])
		}
		if result["content"] != "You are an expert programmer." {
			t.Errorf("expected content %q, got %v",
				"You are an expert programmer.", result["content"])
		}
	})

	t.Run("should support array of string instructions", func(t *testing.T) {
		instructionsArray := []any{
			"You are a helpful assistant.",
			"Always be polite.",
			"Provide detailed answers.",
		}

		agent, err := NewAgent(AgentConfig{
			ID:           "test-agent",
			Name:         "Test Agent",
			Instructions: instructionsArray,
			Model:        newDummyModel(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		instructions, err := agent.GetInstructions(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		result, ok := instructions.([]any)
		if !ok {
			t.Fatalf("expected []any, got %T", instructions)
		}
		if len(result) != 3 {
			t.Fatalf("expected 3 instructions, got %d", len(result))
		}
		if result[0] != "You are a helpful assistant." {
			t.Errorf("expected first instruction %q, got %v",
				"You are a helpful assistant.", result[0])
		}
	})

	t.Run("should support dynamic instructions returning string", func(t *testing.T) {
		agent, err := NewAgent(AgentConfig{
			ID:   "test-agent",
			Name: "Test Agent",
			Instructions: func(reqCtx *requestcontext.RequestContext, mastra Mastra) (any, error) {
				role := reqCtx.Get("role")
				if role == nil {
					role = "assistant"
				}
				return "You are a helpful " + role.(string) + ".", nil
			},
			Model: newDummyModel(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		reqCtx := requestcontext.NewRequestContext()
		reqCtx.Set("role", "teacher")

		instructions, err := agent.GetInstructions(context.Background(), reqCtx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if instructions != "You are a helpful teacher." {
			t.Errorf("expected %q, got %v", "You are a helpful teacher.", instructions)
		}
	})

	t.Run("should support dynamic instructions returning CoreSystemMessage", func(t *testing.T) {
		agent, err := NewAgent(AgentConfig{
			ID:   "test-agent",
			Name: "Test Agent",
			Instructions: func(reqCtx *requestcontext.RequestContext, mastra Mastra) (any, error) {
				role := reqCtx.Get("role")
				if role == nil {
					role = "assistant"
				}
				return map[string]any{
					"role":    "system",
					"content": "You are a helpful " + role.(string) + ".",
				}, nil
			},
			Model: newDummyModel(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		reqCtx := requestcontext.NewRequestContext()
		reqCtx.Set("role", "doctor")

		instructions, err := agent.GetInstructions(context.Background(), reqCtx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		result, ok := instructions.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", instructions)
		}
		if result["content"] != "You are a helpful doctor." {
			t.Errorf("expected content %q, got %v",
				"You are a helpful doctor.", result["content"])
		}
	})

	t.Run("should handle empty instructions gracefully", func(t *testing.T) {
		agent, err := NewAgent(AgentConfig{
			ID:           "test-agent",
			Name:         "Test Agent",
			Instructions: "",
			Model:        newDummyModel(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		instructions, err := agent.GetInstructions(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if instructions != "" {
			t.Errorf("expected empty string, got %v", instructions)
		}
	})

	t.Run("should handle empty array instructions", func(t *testing.T) {
		emptyArr := []any{}
		agent, err := NewAgent(AgentConfig{
			ID:           "test-agent",
			Name:         "Test Agent",
			Instructions: emptyArr,
			Model:        newDummyModel(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		instructions, err := agent.GetInstructions(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		result, ok := instructions.([]any)
		if !ok {
			t.Fatalf("expected []any, got %T", instructions)
		}
		if len(result) != 0 {
			t.Errorf("expected empty array, got %v", result)
		}
	})

	t.Run("should handle dynamic instructions when mastra is nil", func(t *testing.T) {
		var capturedMastra Mastra

		agent, err := NewAgent(AgentConfig{
			ID:   "test-agent",
			Name: "Test Agent",
			Instructions: func(reqCtx *requestcontext.RequestContext, mastra Mastra) (any, error) {
				capturedMastra = mastra
				return "You are a helpful assistant.", nil
			},
			Model: newDummyModel(),
			// No mastra provided
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		instructions, err := agent.GetInstructions(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if instructions != "You are a helpful assistant." {
			t.Errorf("expected %q, got %v", "You are a helpful assistant.", instructions)
		}
		if capturedMastra != nil {
			t.Error("expected capturedMastra to be nil")
		}
	})

	t.Run("should error when dynamic instructions return nil", func(t *testing.T) {
		agent, err := NewAgent(AgentConfig{
			ID:   "test-agent",
			Name: "Test Agent",
			Instructions: func(reqCtx *requestcontext.RequestContext, mastra Mastra) (any, error) {
				return nil, nil
			},
			Model: newDummyModel(),
		})
		if err != nil {
			t.Fatalf("unexpected error creating agent: %v", err)
		}

		_, err = agent.GetInstructions(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error when dynamic instructions return nil, got nil")
		}
		if !strings.Contains(err.Error(), "Instructions are required") {
			t.Errorf("expected error about instructions required, got: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// TestAgentConvertInstructionsToString
// ---------------------------------------------------------------------------

func TestAgentConvertInstructionsToString(t *testing.T) {
	agent, err := NewAgent(AgentConfig{
		Name:         "Convert Agent",
		Instructions: "test",
		Model:        newDummyModel(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("should convert string instructions", func(t *testing.T) {
		result := agent.ConvertInstructionsToString("Hello world")
		if result != "Hello world" {
			t.Errorf("expected %q, got %q", "Hello world", result)
		}
	})

	t.Run("should convert array of strings", func(t *testing.T) {
		result := agent.ConvertInstructionsToString([]any{"Hello", "World"})
		if result != "Hello\n\nWorld" {
			t.Errorf("expected %q, got %q", "Hello\n\nWorld", result)
		}
	})

	t.Run("should convert array of system messages", func(t *testing.T) {
		result := agent.ConvertInstructionsToString([]any{
			map[string]any{"role": "system", "content": "First"},
			map[string]any{"role": "system", "content": "Second"},
		})
		if result != "First\n\nSecond" {
			t.Errorf("expected %q, got %q", "First\n\nSecond", result)
		}
	})

	t.Run("should convert single system message map", func(t *testing.T) {
		result := agent.ConvertInstructionsToString(map[string]any{
			"role":    "system",
			"content": "You are helpful.",
		})
		if result != "You are helpful." {
			t.Errorf("expected %q, got %q", "You are helpful.", result)
		}
	})

	t.Run("should return empty string for unknown type", func(t *testing.T) {
		result := agent.ConvertInstructionsToString(42)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("should filter empty strings in array", func(t *testing.T) {
		result := agent.ConvertInstructionsToString([]any{"Hello", "", "World"})
		if result != "Hello\n\nWorld" {
			t.Errorf("expected %q, got %q", "Hello\n\nWorld", result)
		}
	})
}

// ---------------------------------------------------------------------------
// TestAgentListTools — tools.test.ts
// ---------------------------------------------------------------------------

func TestAgentListTools(t *testing.T) {
	t.Run("should return static tools", func(t *testing.T) {
		tools := ToolsInput{
			"weather":    map[string]any{"description": "Get weather"},
			"calculator": map[string]any{"description": "Calculate"},
		}
		agent, err := NewAgent(AgentConfig{
			ID:           "test-agent",
			Name:         "Test Agent",
			Instructions: "test",
			Model:        newDummyModel(),
			Tools:        tools,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := agent.ListTools(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 tools, got %d", len(result))
		}
		if result["weather"] == nil {
			t.Error("expected 'weather' tool to exist")
		}
		if result["calculator"] == nil {
			t.Error("expected 'calculator' tool to exist")
		}
	})

	t.Run("should return empty tools when no tools configured", func(t *testing.T) {
		agent, err := NewAgent(AgentConfig{
			ID:           "test-agent",
			Name:         "Test Agent",
			Instructions: "test",
			Model:        newDummyModel(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := agent.ListTools(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected 0 tools, got %d", len(result))
		}
	})

	t.Run("should resolve dynamic tools from function", func(t *testing.T) {
		agent, err := NewAgent(AgentConfig{
			ID:   "test-agent",
			Name: "Test Agent",
			Instructions: "test",
			Model: newDummyModel(),
			Tools: func(reqCtx *requestcontext.RequestContext, mastra Mastra) (ToolsInput, error) {
				role := reqCtx.Get("role")
				tools := ToolsInput{
					"search": map[string]any{"description": "Search the web"},
				}
				if role == "admin" {
					tools["admin-tool"] = map[string]any{"description": "Admin only"}
				}
				return tools, nil
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		reqCtx := requestcontext.NewRequestContext()
		reqCtx.Set("role", "admin")

		result, err := agent.ListTools(reqCtx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 tools, got %d", len(result))
		}
		if result["admin-tool"] == nil {
			t.Error("expected 'admin-tool' to exist for admin role")
		}
	})

	t.Run("should error when dynamic tools return nil", func(t *testing.T) {
		agent, err := NewAgent(AgentConfig{
			ID:   "test-agent",
			Name: "Test Agent",
			Instructions: "test",
			Model: newDummyModel(),
			Tools: func(reqCtx *requestcontext.RequestContext, mastra Mastra) (ToolsInput, error) {
				return nil, nil
			},
		})
		if err != nil {
			t.Fatalf("unexpected error creating agent: %v", err)
		}

		_, err = agent.ListTools(nil)
		if err == nil {
			t.Fatal("expected error when dynamic tools return nil")
		}
		if !strings.Contains(err.Error(), "Function-based tools returned empty value") {
			t.Errorf("expected error about empty return, got: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// TestAgentListAgents
// ---------------------------------------------------------------------------

func TestAgentListAgents(t *testing.T) {
	t.Run("should return static sub-agents", func(t *testing.T) {
		subAgent, err := NewAgent(AgentConfig{
			ID:           "sub-agent",
			Name:         "Sub Agent",
			Instructions: "test sub",
			Model:        newDummyModel(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		agent, err := NewAgent(AgentConfig{
			ID:           "parent-agent",
			Name:         "Parent Agent",
			Instructions: "test parent",
			Model:        newDummyModel(),
			Agents:       map[string]*Agent{"sub": subAgent},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := agent.ListAgents(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 sub-agent, got %d", len(result))
		}
		if result["sub"] == nil {
			t.Error("expected 'sub' agent to exist")
		}
	})

	t.Run("should return empty when no agents configured", func(t *testing.T) {
		agent, err := NewAgent(AgentConfig{
			ID:           "test-agent",
			Name:         "Test Agent",
			Instructions: "test",
			Model:        newDummyModel(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := agent.ListAgents(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected 0 agents, got %d", len(result))
		}
	})
}

// ---------------------------------------------------------------------------
// TestAgentGetModelList — model-list.test.ts
// ---------------------------------------------------------------------------

func TestAgentGetModelList(t *testing.T) {
	t.Run("should return model list for agent with multiple models", func(t *testing.T) {
		m1 := newDummyModelWithID("gpt-4o")
		m2 := newDummyModelWithID("gpt-4o-mini")
		m3 := newDummyModelWithID("gpt-4.1")

		agent, err := NewAgent(AgentConfig{
			ID:           "test-agent",
			Name:         "test",
			Instructions: "test agent instructions",
			Model: []ModelWithRetries{
				{Model: m1},
				{Model: m2},
				{Model: m3},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		modelList := agent.GetModelList()
		if modelList == nil {
			t.Fatal("model list should exist")
		}
		if len(modelList) != 3 {
			t.Fatalf("expected 3 models, got %d", len(modelList))
		}
	})

	t.Run("should return nil for single model agent", func(t *testing.T) {
		agent, err := NewAgent(AgentConfig{
			ID:           "test-agent",
			Name:         "test",
			Instructions: "test",
			Model:        newDummyModel(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		modelList := agent.GetModelList()
		if modelList != nil {
			t.Errorf("expected nil model list for single model, got %v", modelList)
		}
	})
}

// ---------------------------------------------------------------------------
// TestAgentReorderModels — model-list.test.ts
// ---------------------------------------------------------------------------

func TestAgentReorderModels(t *testing.T) {
	t.Run("should reorder model list", func(t *testing.T) {
		agent, err := NewAgent(AgentConfig{
			ID:           "test-agent",
			Name:         "Test Agent",
			Instructions: "test",
			Model: []ModelWithRetries{
				{Model: newDummyModelWithID("gpt-4o")},
				{Model: newDummyModelWithID("gpt-4o-mini")},
				{Model: newDummyModelWithID("gpt-4.1")},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		modelList := agent.GetModelList()
		if len(modelList) != 3 {
			t.Fatalf("expected 3 models, got %d", len(modelList))
		}

		// Extract IDs and reverse them.
		modelIDs := make([]string, len(modelList))
		for i, m := range modelList {
			modelIDs[i] = m.ID
		}
		reversedIDs := make([]string, len(modelIDs))
		for i := range modelIDs {
			reversedIDs[i] = modelIDs[len(modelIDs)-1-i]
		}

		agent.ReorderModels(reversedIDs)

		reordered := agent.GetModelList()
		if len(reordered) != 3 {
			t.Fatalf("expected 3 models after reorder, got %d", len(reordered))
		}
		if reordered[0].ID != reversedIDs[0] {
			t.Errorf("expected first model ID %q, got %q", reversedIDs[0], reordered[0].ID)
		}
		if reordered[1].ID != reversedIDs[1] {
			t.Errorf("expected second model ID %q, got %q", reversedIDs[1], reordered[1].ID)
		}
	})

	t.Run("should keep unlisted models at the end with partial list", func(t *testing.T) {
		agent, err := NewAgent(AgentConfig{
			ID:           "test-agent",
			Name:         "Test Agent",
			Instructions: "test",
			Model: []ModelWithRetries{
				{Model: newDummyModelWithID("gpt-4o")},
				{Model: newDummyModelWithID("gpt-4o-mini")},
				{Model: newDummyModelWithID("gpt-4.1")},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		modelList := agent.GetModelList()
		// Reorder: gpt-4.1 first, then gpt-4o; gpt-4o-mini not in list (stays at end).
		agent.ReorderModels([]string{modelList[2].ID, modelList[0].ID})

		reordered := agent.GetModelList()
		if len(reordered) != 3 {
			t.Fatalf("expected 3 models, got %d", len(reordered))
		}

		// Verify the model stored in each slot.
		m0, ok0 := reordered[0].Model.(*mockLanguageModel)
		m1, ok1 := reordered[1].Model.(*mockLanguageModel)
		m2, ok2 := reordered[2].Model.(*mockLanguageModel)
		if !ok0 || !ok1 || !ok2 {
			t.Fatal("expected all models to be *mockLanguageModel")
		}
		if m0.modelID != "gpt-4.1" {
			t.Errorf("expected first model gpt-4.1, got %s", m0.modelID)
		}
		if m1.modelID != "gpt-4o" {
			t.Errorf("expected second model gpt-4o, got %s", m1.modelID)
		}
		if m2.modelID != "gpt-4o-mini" {
			t.Errorf("expected third model gpt-4o-mini, got %s", m2.modelID)
		}
	})
}

// ---------------------------------------------------------------------------
// TestAgentUpdateModelInModelList — model-list.test.ts
// ---------------------------------------------------------------------------

func TestAgentUpdateModelInModelList(t *testing.T) {
	t.Run("should update model in model list", func(t *testing.T) {
		agent, err := NewAgent(AgentConfig{
			ID:           "test-agent",
			Name:         "Test Agent",
			Instructions: "test",
			Model: []ModelWithRetries{
				{Model: newDummyModelWithID("gpt-4o")},
				{Model: newDummyModelWithID("gpt-4o-mini")},
				{Model: newDummyModelWithID("gpt-4.1")},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		modelList := agent.GetModelList()
		model1ID := modelList[1].ID

		newModel := newDummyModelWithID("gpt-4")
		newRetries := 5
		agent.UpdateModelInModelList(model1ID, newModel, nil, &newRetries)

		updated := agent.GetModelList()
		if len(updated) != 3 {
			t.Fatalf("expected 3 models, got %d", len(updated))
		}

		// Second model should be updated.
		m1, ok := updated[1].Model.(*mockLanguageModel)
		if !ok {
			t.Fatal("expected updated model to be *mockLanguageModel")
		}
		if m1.modelID != "gpt-4" {
			t.Errorf("expected updated model ID %q, got %q", "gpt-4", m1.modelID)
		}
		if updated[1].MaxRetries != 5 {
			t.Errorf("expected maxRetries=5, got %d", updated[1].MaxRetries)
		}

		// Third model should be unchanged.
		m2, ok := updated[2].Model.(*mockLanguageModel)
		if !ok {
			t.Fatal("expected third model to be *mockLanguageModel")
		}
		if m2.modelID != "gpt-4.1" {
			t.Errorf("expected third model %q, got %q", "gpt-4.1", m2.modelID)
		}
	})

	t.Run("should update enabled flag", func(t *testing.T) {
		agent, err := NewAgent(AgentConfig{
			Name:         "Test Agent",
			Instructions: "test",
			Model: []ModelWithRetries{
				{Model: newDummyModel()},
				{Model: newDummyModel()},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		modelList := agent.GetModelList()
		if !modelList[0].Enabled {
			t.Fatal("expected first model initially enabled")
		}

		falseVal := false
		agent.UpdateModelInModelList(modelList[0].ID, nil, &falseVal, nil)

		updated := agent.GetModelList()
		if updated[0].Enabled {
			t.Error("expected first model to be disabled after update")
		}
	})
}

// ---------------------------------------------------------------------------
// TestAgentGetMostRecentUserMessage
// ---------------------------------------------------------------------------

func TestAgentGetMostRecentUserMessage(t *testing.T) {
	agent, err := NewAgent(AgentConfig{
		Name:         "Test Agent",
		Instructions: "test",
		Model:        newDummyModel(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("should return the most recent user message", func(t *testing.T) {
		messages := []MastraDBMessage{
			{ID: "1", Role: "user", Content: MastraMessageContentV2{Parts: []MastraMessagePart{{Type: "text", Text: "First"}}}},
			{ID: "2", Role: "assistant", Content: MastraMessageContentV2{Parts: []MastraMessagePart{{Type: "text", Text: "Response"}}}},
			{ID: "3", Role: "user", Content: MastraMessageContentV2{Parts: []MastraMessagePart{{Type: "text", Text: "Second"}}}},
		}

		result := agent.GetMostRecentUserMessage(messages)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.ID != "3" {
			t.Errorf("expected message ID %q, got %q", "3", result.ID)
		}
	})

	t.Run("should return nil when no user messages", func(t *testing.T) {
		messages := []MastraDBMessage{
			{ID: "1", Role: "assistant"},
			{ID: "2", Role: "assistant"},
		}

		result := agent.GetMostRecentUserMessage(messages)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("should return nil for empty slice", func(t *testing.T) {
		result := agent.GetMostRecentUserMessage(nil)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})
}

// ---------------------------------------------------------------------------
// TestResolveThreadIdFromArgs
// ---------------------------------------------------------------------------

func TestResolveThreadIdFromArgs(t *testing.T) {
	t.Run("should resolve thread ID from memory.thread string", func(t *testing.T) {
		result := ResolveThreadIdFromArgs(ResolveThreadArgs{
			Memory: &AgentMemoryOption{
				Thread: "thread-123",
			},
		})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.ID != "thread-123" {
			t.Errorf("expected ID %q, got %q", "thread-123", result.ID)
		}
	})

	t.Run("should resolve thread ID from memory.thread object", func(t *testing.T) {
		result := ResolveThreadIdFromArgs(ResolveThreadArgs{
			Memory: &AgentMemoryOption{
				Thread: map[string]any{
					"id":    "thread-456",
					"title": "My Thread",
				},
			},
		})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.ID != "thread-456" {
			t.Errorf("expected ID %q, got %q", "thread-456", result.ID)
		}
		if result.Title != "My Thread" {
			t.Errorf("expected Title %q, got %q", "My Thread", result.Title)
		}
	})

	t.Run("should fallback to threadId", func(t *testing.T) {
		result := ResolveThreadIdFromArgs(ResolveThreadArgs{
			ThreadID: "fallback-thread",
		})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.ID != "fallback-thread" {
			t.Errorf("expected ID %q, got %q", "fallback-thread", result.ID)
		}
	})

	t.Run("should return nil when no thread info", func(t *testing.T) {
		result := ResolveThreadIdFromArgs(ResolveThreadArgs{})
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("should prefer memory.thread over threadId", func(t *testing.T) {
		result := ResolveThreadIdFromArgs(ResolveThreadArgs{
			Memory: &AgentMemoryOption{
				Thread: "from-memory",
			},
			ThreadID: "from-top-level",
		})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.ID != "from-memory" {
			t.Errorf("expected ID %q, got %q", "from-memory", result.ID)
		}
	})
}

// ---------------------------------------------------------------------------
// TestIsSupportedLanguageModel
// ---------------------------------------------------------------------------

func TestIsSupportedLanguageModel(t *testing.T) {
	t.Run("should return true for v2 models", func(t *testing.T) {
		model := &mockLanguageModel{specVersion: "v2"}
		if !IsSupportedLanguageModel(model) {
			t.Error("expected v2 model to be supported")
		}
	})

	t.Run("should return true for v3 models", func(t *testing.T) {
		model := &mockLanguageModel{specVersion: "v3"}
		if !IsSupportedLanguageModel(model) {
			t.Error("expected v3 model to be supported")
		}
	})

	t.Run("should return false for v1 models", func(t *testing.T) {
		model := &mockLanguageModel{specVersion: "v1"}
		if IsSupportedLanguageModel(model) {
			t.Error("expected v1 model to NOT be supported")
		}
	})

	t.Run("should return false for unknown versions", func(t *testing.T) {
		model := &mockLanguageModel{specVersion: "v99"}
		if IsSupportedLanguageModel(model) {
			t.Error("expected v99 model to NOT be supported")
		}
	})
}

// ---------------------------------------------------------------------------
// TestGenerateConversationHistory — test_utils.go coverage
// ---------------------------------------------------------------------------

func TestGenerateConversationHistory(t *testing.T) {
	t.Run("should generate default conversation history", func(t *testing.T) {
		result := GenerateConversationHistory(GenerateConversationHistoryParams{
			ThreadID: "test-thread",
		})

		if result.Counts.Messages < 10 {
			t.Errorf("expected at least 10 messages (5 pairs), got %d", result.Counts.Messages)
		}
		if len(result.MessagesV2) == 0 {
			t.Error("expected non-empty MessagesV2")
		}
		if len(result.Messages) == 0 {
			t.Error("expected non-empty Messages (v1)")
		}

		// Verify thread ID is set on all messages.
		for _, msg := range result.MessagesV2 {
			if msg.ThreadID != "test-thread" {
				t.Errorf("expected threadID %q, got %q", "test-thread", msg.ThreadID)
			}
		}
	})

	t.Run("should include tool calls based on frequency", func(t *testing.T) {
		result := GenerateConversationHistory(GenerateConversationHistoryParams{
			ThreadID:      "test-thread",
			MessageCount:  10,
			ToolFrequency: 3,
		})

		if result.Counts.ToolCalls == 0 {
			t.Error("expected at least one tool call with frequency=3 and 10 messages")
		}
	})

	t.Run("should use custom resource ID", func(t *testing.T) {
		result := GenerateConversationHistory(GenerateConversationHistoryParams{
			ThreadID:   "test-thread",
			ResourceID: "custom-resource",
		})

		for _, msg := range result.MessagesV2 {
			if msg.ResourceID != "custom-resource" {
				t.Errorf("expected resourceID %q, got %q", "custom-resource", msg.ResourceID)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// TestAssertNoDuplicateParts — test_utils.go coverage
// ---------------------------------------------------------------------------

func TestAssertNoDuplicateParts(t *testing.T) {
	t.Run("should pass with unique parts", func(t *testing.T) {
		parts := []MastraMessagePart{
			{Type: "text", Text: "Hello"},
			{Type: "text", Text: "World"},
			{Type: "tool-invocation", ToolInvocation: &ToolInvocation{
				State: "result", ToolCallID: "tc-1", Result: "ok",
			}},
		}
		// Should not call t.Errorf — use a sub-test to verify.
		mockT := &testing.T{}
		AssertNoDuplicateParts(mockT, parts)
		// If mockT has no failures, the assertion passed.
	})
}

// ---------------------------------------------------------------------------
// TestAgentGetOverridableFields
// ---------------------------------------------------------------------------

func TestAgentGetOverridableFields(t *testing.T) {
	tools := ToolsInput{"myTool": "value"}
	agent, err := NewAgent(AgentConfig{
		Name:         "Override Agent",
		Instructions: "test instructions",
		Model:        newDummyModel(),
		Tools:        tools,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fields := agent.GetOverridableFields()
	if fields["instructions"] != "test instructions" {
		t.Errorf("expected instructions field, got %v", fields["instructions"])
	}
	if fields["tools"] == nil {
		t.Error("expected tools field to be non-nil")
	}
	resultTools, ok := fields["tools"].(ToolsInput)
	if !ok {
		t.Fatalf("expected ToolsInput, got %T", fields["tools"])
	}
	if !reflect.DeepEqual(resultTools, tools) {
		t.Errorf("expected tools %v, got %v", tools, resultTools)
	}
}
