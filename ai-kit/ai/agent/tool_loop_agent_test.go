// Ported from: packages/ai/src/agent/tool-loop-agent.test.ts
package agent

import (
	"testing"

	gt "github.com/brainlet/brainkit/ai-kit/ai/generatetext"
)

// mockLanguageModel is a minimal mock for testing.
type mockLanguageModel struct {
	provider string
	modelID  string
}

func (m *mockLanguageModel) Provider() string { return m.provider }
func (m *mockLanguageModel) ModelID() string  { return m.modelID }

func newMockModel() *mockLanguageModel {
	return &mockLanguageModel{
		provider: "mock-provider",
		modelID:  "mock-model-id",
	}
}

func TestToolLoopAgent_Version(t *testing.T) {
	agent := NewToolLoopAgent(ToolLoopAgentSettings{
		Model: newMockModel(),
	})

	if got := agent.Version(); got != "agent-v1" {
		t.Errorf("Version() = %q, want %q", got, "agent-v1")
	}
}

func TestToolLoopAgent_ID(t *testing.T) {
	t.Run("returns empty string when not set", func(t *testing.T) {
		agent := NewToolLoopAgent(ToolLoopAgentSettings{
			Model: newMockModel(),
		})

		if got := agent.ID(); got != "" {
			t.Errorf("ID() = %q, want empty string", got)
		}
	})

	t.Run("returns id when set", func(t *testing.T) {
		agent := NewToolLoopAgent(ToolLoopAgentSettings{
			ID:    "my-agent",
			Model: newMockModel(),
		})

		if got := agent.ID(); got != "my-agent" {
			t.Errorf("ID() = %q, want %q", got, "my-agent")
		}
	})
}

func TestToolLoopAgent_Tools(t *testing.T) {
	t.Run("returns nil when no tools set", func(t *testing.T) {
		agent := NewToolLoopAgent(ToolLoopAgentSettings{
			Model: newMockModel(),
		})

		if got := agent.Tools(); got != nil {
			t.Errorf("Tools() = %v, want nil", got)
		}
	})

	t.Run("returns tools when set", func(t *testing.T) {
		tools := gt.ToolSet{
			"myTool": gt.Tool{
				Type: "function",
			},
		}
		agent := NewToolLoopAgent(ToolLoopAgentSettings{
			Model: newMockModel(),
			Tools: tools,
		})

		got := agent.Tools()
		if got == nil {
			t.Fatal("Tools() = nil, want non-nil")
		}
		if _, ok := got["myTool"]; !ok {
			t.Error("Tools() does not contain 'myTool'")
		}
	})
}

func TestToolLoopAgent_ImplementsAgent(t *testing.T) {
	// Compile-time check that ToolLoopAgent implements Agent.
	var _ Agent = (*ToolLoopAgent)(nil)
}

func TestToolLoopAgent_PrepareCall(t *testing.T) {
	t.Run("uses default StepCountIs(20) when StopWhen not set", func(t *testing.T) {
		agent := NewToolLoopAgent(ToolLoopAgentSettings{
			Model: newMockModel(),
		})

		args, err := agent.prepareCall("test prompt", nil, nil, nil)
		if err != nil {
			t.Fatalf("prepareCall() error = %v", err)
		}

		if args.StopWhen == nil {
			t.Fatal("StopWhen is nil, expected default StepCountIs(20)")
		}
		if len(args.StopWhen) != 1 {
			t.Fatalf("StopWhen has %d conditions, expected 1", len(args.StopWhen))
		}

		// Verify the default stop condition: it should stop at step count 20
		steps := make([]gt.StepResult, 20)
		met, err := args.StopWhen[0](gt.StopConditionOptions{Steps: steps})
		if err != nil {
			t.Fatalf("StopWhen[0]() error = %v", err)
		}
		if !met {
			t.Error("Default stop condition should be met at 20 steps")
		}
	})

	t.Run("uses custom StopWhen when set", func(t *testing.T) {
		customStop := gt.StepCountIs(5)
		agent := NewToolLoopAgent(ToolLoopAgentSettings{
			Model:    newMockModel(),
			StopWhen: []gt.StopCondition{customStop},
		})

		args, err := agent.prepareCall("test prompt", nil, nil, nil)
		if err != nil {
			t.Fatalf("prepareCall() error = %v", err)
		}

		if len(args.StopWhen) != 1 {
			t.Fatalf("StopWhen has %d conditions, expected 1", len(args.StopWhen))
		}

		steps := make([]gt.StepResult, 5)
		met, err := args.StopWhen[0](gt.StopConditionOptions{Steps: steps})
		if err != nil {
			t.Fatalf("StopWhen[0]() error = %v", err)
		}
		if !met {
			t.Error("Custom stop condition should be met at 5 steps")
		}
	})

	t.Run("passes settings fields through", func(t *testing.T) {
		temp := 0.7
		maxTokens := 500
		agent := NewToolLoopAgent(ToolLoopAgentSettings{
			Model:           newMockModel(),
			Instructions:    "You are a helpful assistant",
			Temperature:     &temp,
			MaxOutputTokens: &maxTokens,
			ProviderOptions: gt.ProviderOptions{
				"test": {"key": "value"},
			},
		})

		args, err := agent.prepareCall("test prompt", nil, nil, nil)
		if err != nil {
			t.Fatalf("prepareCall() error = %v", err)
		}

		if args.System != "You are a helpful assistant" {
			t.Errorf("System = %v, want %q", args.System, "You are a helpful assistant")
		}
		if args.Temperature == nil || *args.Temperature != 0.7 {
			t.Errorf("Temperature = %v, want 0.7", args.Temperature)
		}
		if args.MaxOutputTokens == nil || *args.MaxOutputTokens != 500 {
			t.Errorf("MaxOutputTokens = %v, want 500", args.MaxOutputTokens)
		}
		if args.ProviderOptions == nil {
			t.Error("ProviderOptions is nil")
		}
	})

	t.Run("uses PrepareCall when set", func(t *testing.T) {
		agent := NewToolLoopAgent(ToolLoopAgentSettings{
			Model: newMockModel(),
			PrepareCall: func(input PrepareCallInput) (*PrepareCallResult, error) {
				return &PrepareCallResult{
					Model:           input.Model,
					Instructions:    input.Instructions,
					StopWhen:        input.StopWhen,
					ProviderOptions: gt.ProviderOptions{"test": {"value": input.Options}},
				}, nil
			},
		})

		args, err := agent.prepareCall("test prompt", nil, nil, "my-option")
		if err != nil {
			t.Fatalf("prepareCall() error = %v", err)
		}

		if args.ProviderOptions == nil {
			t.Fatal("ProviderOptions is nil")
		}
		testOpts, ok := args.ProviderOptions["test"]
		if !ok {
			t.Fatal("ProviderOptions missing 'test' key")
		}
		if testOpts["value"] != "my-option" {
			t.Errorf("ProviderOptions[test][value] = %v, want %q", testOpts["value"], "my-option")
		}
	})
}

func TestMergeCallbacks(t *testing.T) {
	t.Run("mergeOnStartCallbacks both set", func(t *testing.T) {
		var calls []string
		settings := OnStartCallback(func(event gt.OnStartEvent) {
			calls = append(calls, "settings")
		})
		method := OnStartCallback(func(event gt.OnStartEvent) {
			calls = append(calls, "method")
		})

		merged := mergeOnStartCallbacks(settings, method)
		if merged == nil {
			t.Fatal("merged is nil")
		}
		merged(gt.OnStartEvent{})

		if len(calls) != 2 {
			t.Fatalf("expected 2 calls, got %d", len(calls))
		}
		if calls[0] != "settings" {
			t.Errorf("calls[0] = %q, want %q", calls[0], "settings")
		}
		if calls[1] != "method" {
			t.Errorf("calls[1] = %q, want %q", calls[1], "method")
		}
	})

	t.Run("mergeOnStartCallbacks only settings", func(t *testing.T) {
		var called bool
		settings := OnStartCallback(func(event gt.OnStartEvent) {
			called = true
		})

		merged := mergeOnStartCallbacks(settings, nil)
		if merged == nil {
			t.Fatal("merged is nil")
		}
		merged(gt.OnStartEvent{})
		if !called {
			t.Error("settings callback was not called")
		}
	})

	t.Run("mergeOnStartCallbacks only method", func(t *testing.T) {
		var called bool
		method := OnStartCallback(func(event gt.OnStartEvent) {
			called = true
		})

		merged := mergeOnStartCallbacks(nil, method)
		if merged == nil {
			t.Fatal("merged is nil")
		}
		merged(gt.OnStartEvent{})
		if !called {
			t.Error("method callback was not called")
		}
	})

	t.Run("mergeOnStartCallbacks both nil", func(t *testing.T) {
		merged := mergeOnStartCallbacks(nil, nil)
		if merged != nil {
			t.Error("merged should be nil when both are nil")
		}
	})

	t.Run("mergeOnStepStartCallbacks both set", func(t *testing.T) {
		var calls []string
		settings := OnStepStartCallback(func(event gt.OnStepStartEvent) {
			calls = append(calls, "settings")
		})
		method := OnStepStartCallback(func(event gt.OnStepStartEvent) {
			calls = append(calls, "method")
		})

		merged := mergeOnStepStartCallbacks(settings, method)
		if merged == nil {
			t.Fatal("merged is nil")
		}
		merged(gt.OnStepStartEvent{})

		if len(calls) != 2 || calls[0] != "settings" || calls[1] != "method" {
			t.Errorf("calls = %v, want [settings, method]", calls)
		}
	})

	t.Run("mergeOnStepFinishCallbacks both set", func(t *testing.T) {
		var calls []string
		settings := OnStepFinishCallback(func(event gt.OnStepFinishEvent) {
			calls = append(calls, "settings")
		})
		method := OnStepFinishCallback(func(event gt.OnStepFinishEvent) {
			calls = append(calls, "method")
		})

		merged := mergeOnStepFinishCallbacks(settings, method)
		if merged == nil {
			t.Fatal("merged is nil")
		}
		merged(gt.OnStepFinishEvent{})

		if len(calls) != 2 || calls[0] != "settings" || calls[1] != "method" {
			t.Errorf("calls = %v, want [settings, method]", calls)
		}
	})

	t.Run("mergeOnFinishCallbacks both set", func(t *testing.T) {
		var calls []string
		settings := OnFinishCallback(func(event gt.OnFinishEvent) {
			calls = append(calls, "settings")
		})
		method := OnFinishCallback(func(event gt.OnFinishEvent) {
			calls = append(calls, "method")
		})

		merged := mergeOnFinishCallbacks(settings, method)
		if merged == nil {
			t.Fatal("merged is nil")
		}
		merged(gt.OnFinishEvent{})

		if len(calls) != 2 || calls[0] != "settings" || calls[1] != "method" {
			t.Errorf("calls = %v, want [settings, method]", calls)
		}
	})

	t.Run("mergeOnToolCallStartCallbacks both set", func(t *testing.T) {
		var calls []string
		settings := OnToolCallStartCallback(func(event gt.OnToolCallStartEvent) {
			calls = append(calls, "settings")
		})
		method := OnToolCallStartCallback(func(event gt.OnToolCallStartEvent) {
			calls = append(calls, "method")
		})

		merged := mergeOnToolCallStartCallbacks(settings, method)
		if merged == nil {
			t.Fatal("merged is nil")
		}
		merged(gt.OnToolCallStartEvent{})

		if len(calls) != 2 || calls[0] != "settings" || calls[1] != "method" {
			t.Errorf("calls = %v, want [settings, method]", calls)
		}
	})

	t.Run("mergeOnToolCallFinishCallbacks both set", func(t *testing.T) {
		var calls []string
		settings := OnToolCallFinishCallback(func(event gt.OnToolCallFinishEvent) {
			calls = append(calls, "settings")
		})
		method := OnToolCallFinishCallback(func(event gt.OnToolCallFinishEvent) {
			calls = append(calls, "method")
		})

		merged := mergeOnToolCallFinishCallbacks(settings, method)
		if merged == nil {
			t.Fatal("merged is nil")
		}
		merged(gt.OnToolCallFinishEvent{})

		if len(calls) != 2 || calls[0] != "settings" || calls[1] != "method" {
			t.Errorf("calls = %v, want [settings, method]", calls)
		}
	})
}

func TestToolLoopAgent_ToGenerateTextOptions(t *testing.T) {
	t.Run("string prompt sets Prompt field", func(t *testing.T) {
		args := &preparedCallArgs{
			Model:  newMockModel(),
			Prompt: "Hello, world!",
		}
		opts := args.toGenerateTextOptions(nil, nil)
		if opts.Prompt != "Hello, world!" {
			t.Errorf("Prompt = %q, want %q", opts.Prompt, "Hello, world!")
		}
	})

	t.Run("prompt messages set Messages field", func(t *testing.T) {
		msgs := []gt.ModelMessage{{Role: "user", Content: "test"}}
		args := &preparedCallArgs{
			Model:          newMockModel(),
			PromptMessages: msgs,
		}
		opts := args.toGenerateTextOptions(nil, nil)
		if len(opts.Messages) != 1 {
			t.Errorf("Messages length = %d, want 1", len(opts.Messages))
		}
	})

	t.Run("messages set Messages field", func(t *testing.T) {
		msgs := []gt.ModelMessage{{Role: "user", Content: "test"}}
		args := &preparedCallArgs{
			Model:    newMockModel(),
			Messages: msgs,
		}
		opts := args.toGenerateTextOptions(nil, nil)
		if len(opts.Messages) != 1 {
			t.Errorf("Messages length = %d, want 1", len(opts.Messages))
		}
	})

	t.Run("passes settings through", func(t *testing.T) {
		temp := 0.7
		maxTokens := 500
		system := "Be helpful"
		args := &preparedCallArgs{
			Model:           newMockModel(),
			Prompt:          "Hello",
			System:          system,
			Temperature:     &temp,
			MaxOutputTokens: &maxTokens,
			ProviderOptions: gt.ProviderOptions{"test": {"key": "val"}},
			ActiveTools:     []string{"tool1"},
		}
		opts := args.toGenerateTextOptions(nil, nil)

		if opts.System != system {
			t.Errorf("System = %v, want %q", opts.System, system)
		}
		if opts.Temperature == nil || *opts.Temperature != 0.7 {
			t.Errorf("Temperature = %v, want 0.7", opts.Temperature)
		}
		if opts.MaxOutputTokens == nil || *opts.MaxOutputTokens != 500 {
			t.Errorf("MaxOutputTokens = %v, want 500", opts.MaxOutputTokens)
		}
		if len(opts.ActiveTools) != 1 || opts.ActiveTools[0] != "tool1" {
			t.Errorf("ActiveTools = %v, want [tool1]", opts.ActiveTools)
		}
	})
}

func TestToolLoopAgent_ToStreamTextOptions(t *testing.T) {
	t.Run("string prompt sets Prompt field", func(t *testing.T) {
		args := &preparedCallArgs{
			Model:  newMockModel(),
			Prompt: "Hello, world!",
		}
		opts := args.toStreamTextOptions(nil, nil, nil)
		if opts.Prompt != "Hello, world!" {
			t.Errorf("Prompt = %q, want %q", opts.Prompt, "Hello, world!")
		}
	})

	t.Run("applies transform", func(t *testing.T) {
		args := &preparedCallArgs{
			Model:  newMockModel(),
			Prompt: "test",
		}
		transform := gt.StreamTextTransform(func(opts gt.StreamTextTransformOptions) func(input <-chan gt.TextStreamPart, output chan<- gt.TextStreamPart) {
			return func(input <-chan gt.TextStreamPart, output chan<- gt.TextStreamPart) {}
		})
		opts := args.toStreamTextOptions(nil, nil, []gt.StreamTextTransform{transform})
		if opts.Transform == nil {
			t.Error("Transform is nil, expected non-nil")
		}
	})
}
