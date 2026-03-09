// Ported from: packages/core/src/tool-loop-agent/__tests__/tool-loop-agent.test.ts
// (subset: isToolLoopAgentLike, getSettings, ToolLoopAgentProcessor.getAgentConfig,
// toolLoopAgentToMastraAgent — the utility/conversion tests that don't need real LLMs)
package toolloopagent

import (
	"testing"
)

func TestIsToolLoopAgentLike(t *testing.T) {
	t.Run("should return true for ToolLoopAgent", func(t *testing.T) {
		agent := &ToolLoopAgent{
			Version: "agent-v1",
			ID:      "test-agent",
			Settings: &ToolLoopAgentSettings{
				ID:    "test-agent",
				Model: "gpt-4",
			},
		}
		if !IsToolLoopAgentLike(agent) {
			t.Error("expected IsToolLoopAgentLike to return true for ToolLoopAgent")
		}
	})

	t.Run("should return false for nil", func(t *testing.T) {
		if IsToolLoopAgentLike(nil) {
			t.Error("expected IsToolLoopAgentLike to return false for nil")
		}
	})

	t.Run("should return false for non-agent object", func(t *testing.T) {
		if IsToolLoopAgentLike("not an agent") {
			t.Error("expected IsToolLoopAgentLike to return false for string")
		}
		if IsToolLoopAgentLike(42) {
			t.Error("expected IsToolLoopAgentLike to return false for int")
		}
	})
}

func TestGetSettings(t *testing.T) {
	t.Run("should extract settings from ToolLoopAgent", func(t *testing.T) {
		settings := &ToolLoopAgentSettings{
			ID:           "test-agent",
			Model:        "gpt-4",
			Instructions: "You are helpful.",
		}
		agent := &ToolLoopAgent{
			Version:  "agent-v1",
			Settings: settings,
		}

		result, err := GetSettings(agent)
		if err != nil {
			t.Fatalf("GetSettings returned error: %v", err)
		}
		if result == nil {
			t.Fatal("expected settings to be non-nil")
		}
		if result.ID != "test-agent" {
			t.Errorf("expected ID=test-agent, got %s", result.ID)
		}
		if result.Instructions != "You are helpful." {
			t.Errorf("expected Instructions='You are helpful.', got %v", result.Instructions)
		}
	})

	t.Run("should error when settings are nil", func(t *testing.T) {
		agent := &ToolLoopAgent{
			Version:  "agent-v1",
			Settings: nil,
		}

		_, err := GetSettings(agent)
		if err == nil {
			t.Fatal("expected error for nil settings")
		}
	})
}

func TestToolLoopAgentProcessor_GetAgentConfig(t *testing.T) {
	t.Run("should extract basic agent config", func(t *testing.T) {
		temp := 0.5
		maxRetries := 3
		settings := &ToolLoopAgentSettings{
			ID:           "config-agent",
			Model:        "gpt-4",
			Instructions: "Be helpful",
			Temperature:  &temp,
			MaxRetries:   &maxRetries,
			Tools:        map[string]any{"weather": "tool-def"},
		}
		agent := &ToolLoopAgent{
			Version:  "agent-v1",
			Settings: settings,
			Tools:    map[string]any{"weather": "tool-def"},
		}

		proc, err := NewToolLoopAgentProcessor(agent)
		if err != nil {
			t.Fatalf("NewToolLoopAgentProcessor returned error: %v", err)
		}

		config := proc.GetAgentConfig()
		if config.ID != "config-agent" {
			t.Errorf("expected ID=config-agent, got %s", config.ID)
		}
		if config.Instructions != "Be helpful" {
			t.Errorf("expected Instructions='Be helpful', got %v", config.Instructions)
		}
		if config.Model != "gpt-4" {
			t.Errorf("expected Model=gpt-4, got %v", config.Model)
		}
		if config.MaxRetries == nil || *config.MaxRetries != 3 {
			t.Errorf("expected MaxRetries=3, got %v", config.MaxRetries)
		}

		// Check tools from agent.GetTools()
		if config.Tools == nil || config.Tools["weather"] == nil {
			t.Error("expected tools to include weather")
		}

		// Check model settings in default options (ModelSettings is map[string]any
		// to match the real processors.ProcessInputStepResult type).
		if config.DefaultOptions == nil {
			t.Fatal("expected DefaultOptions to be set")
		}
		if config.DefaultOptions.ModelSettings == nil {
			t.Fatal("expected ModelSettings to be set")
		}
		ms, ok := config.DefaultOptions.ModelSettings.(map[string]any)
		if !ok {
			t.Fatalf("expected ModelSettings to be map[string]any, got %T", config.DefaultOptions.ModelSettings)
		}
		tempVal, ok := ms["temperature"]
		if !ok {
			t.Fatal("expected ModelSettings to contain temperature")
		}
		if tempVal != 0.5 {
			t.Errorf("expected Temperature=0.5, got %v", tempVal)
		}
	})

	t.Run("should handle empty settings gracefully", func(t *testing.T) {
		agent := &ToolLoopAgent{
			Version:  "agent-v1",
			Settings: &ToolLoopAgentSettings{},
		}

		proc, err := NewToolLoopAgentProcessor(agent)
		if err != nil {
			t.Fatalf("NewToolLoopAgentProcessor returned error: %v", err)
		}

		config := proc.GetAgentConfig()
		if config.ID != "" {
			t.Errorf("expected empty ID, got %s", config.ID)
		}
		// DefaultOptions should be nil when no options are configured
		if config.DefaultOptions != nil {
			t.Error("expected DefaultOptions to be nil when nothing is configured")
		}
	})
}

func TestToolLoopAgentToMastraAgent(t *testing.T) {
	t.Run("should convert ToolLoopAgent to Mastra Agent", func(t *testing.T) {
		agent := &ToolLoopAgent{
			Version: "agent-v1",
			ID:      "weather-agent",
			Settings: &ToolLoopAgentSettings{
				ID:           "weather-agent",
				Model:        "gpt-4",
				Instructions: "You are a helpful weather assistant.",
				Tools:        map[string]any{"weather": "weather-tool"},
			},
			Tools: map[string]any{"weather": "weather-tool"},
		}

		mastraAgent, err := ToolLoopAgentToMastraAgent(agent, nil)
		if err != nil {
			t.Fatalf("ToolLoopAgentToMastraAgent returned error: %v", err)
		}

		if mastraAgent.ID != "weather-agent" {
			t.Errorf("expected ID=weather-agent, got %s", mastraAgent.ID)
		}
		if mastraAgent.Name != "weather-agent" {
			t.Errorf("expected Name=weather-agent, got %s", mastraAgent.Name)
		}
		if mastraAgent.Instructions != "You are a helpful weather assistant." {
			t.Errorf("expected Instructions match, got %v", mastraAgent.Instructions)
		}
		if len(mastraAgent.InputProcessors) != 1 {
			t.Errorf("expected 1 input processor, got %d", len(mastraAgent.InputProcessors))
		}
	})

	t.Run("should use fallback name when no ID", func(t *testing.T) {
		agent := &ToolLoopAgent{
			Version: "agent-v1",
			Settings: &ToolLoopAgentSettings{
				Model: "gpt-4",
			},
		}

		mastraAgent, err := ToolLoopAgentToMastraAgent(agent, &ToolLoopAgentToMastraAgentOptions{
			FallbackName: "my-fallback",
		})
		if err != nil {
			t.Fatalf("ToolLoopAgentToMastraAgent returned error: %v", err)
		}
		if mastraAgent.ID != "my-fallback" {
			t.Errorf("expected ID=my-fallback, got %s", mastraAgent.ID)
		}
	})

	t.Run("should generate ID when no ID and no fallback", func(t *testing.T) {
		agent := &ToolLoopAgent{
			Version: "agent-v1",
			Settings: &ToolLoopAgentSettings{
				Model: "gpt-4",
			},
		}

		mastraAgent, err := ToolLoopAgentToMastraAgent(agent, nil)
		if err != nil {
			t.Fatalf("ToolLoopAgentToMastraAgent returned error: %v", err)
		}
		if len(mastraAgent.ID) < 16 {
			t.Errorf("expected generated ID with prefix, got %s", mastraAgent.ID)
		}
		// Should start with "tool-loop-agent-"
		if mastraAgent.ID[:16] != "tool-loop-agent-" {
			t.Errorf("expected ID prefix 'tool-loop-agent-', got %s", mastraAgent.ID[:16])
		}
	})
}

func TestToolLoopAgentProcessor_ProcessInputStep(t *testing.T) {
	t.Run("should return result without prepare hooks", func(t *testing.T) {
		agent := &ToolLoopAgent{
			Version: "agent-v1",
			Settings: &ToolLoopAgentSettings{
				Model: "gpt-4",
			},
		}

		proc, err := NewToolLoopAgentProcessor(agent)
		if err != nil {
			t.Fatalf("NewToolLoopAgentProcessor returned error: %v", err)
		}

		result, err := proc.ProcessInputStep(&ProcessInputStepArgs{
			StepNumber: 0,
		})
		if err != nil {
			t.Fatalf("ProcessInputStep returned error: %v", err)
		}
		if result == nil {
			t.Fatal("expected result to be non-nil")
		}
	})

	t.Run("should call prepareCall on step 0", func(t *testing.T) {
		prepareCallCalled := false
		agent := &ToolLoopAgent{
			Version: "agent-v1",
			Settings: &ToolLoopAgentSettings{
				Model: "gpt-4",
				PrepareCall: func(input any) (map[string]any, error) {
					prepareCallCalled = true
					return map[string]any{
						"temperature": 0.9,
					}, nil
				},
			},
		}

		proc, _ := NewToolLoopAgentProcessor(agent)
		result, err := proc.ProcessInputStep(&ProcessInputStepArgs{
			StepNumber: 0,
			Model:      "gpt-4",
		})
		if err != nil {
			t.Fatalf("ProcessInputStep returned error: %v", err)
		}
		if !prepareCallCalled {
			t.Error("expected prepareCall to be called on step 0")
		}
		if result.ModelSettings == nil {
			t.Fatal("expected ModelSettings to be set from prepareCall result")
		}
		if result.ModelSettings["temperature"] != 0.9 {
			t.Errorf("expected temperature=0.9 from prepareCall result, got %v", result.ModelSettings["temperature"])
		}
	})

	t.Run("should not call prepareCall on step > 0", func(t *testing.T) {
		prepareCallCalled := false
		agent := &ToolLoopAgent{
			Version: "agent-v1",
			Settings: &ToolLoopAgentSettings{
				Model: "gpt-4",
				PrepareCall: func(input any) (map[string]any, error) {
					prepareCallCalled = true
					return nil, nil
				},
			},
		}

		proc, _ := NewToolLoopAgentProcessor(agent)
		_, _ = proc.ProcessInputStep(&ProcessInputStepArgs{
			StepNumber: 1,
		})
		if prepareCallCalled {
			t.Error("expected prepareCall NOT to be called on step > 0")
		}
	})
}
