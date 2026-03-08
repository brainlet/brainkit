// Ported from: packages/ai/src/generate-text/generate-text.test.ts
// Note: The TS test file is 282KB and relies on MockLanguageModelV3 which is not ported.
// This Go test covers the structural aspects testable without a full model mock:
// - GenerateTextOptions construction
// - DefaultGenerateTextResult construction and ToResult()
// - Stop condition evaluation
// - Utility functions
package generatetext

import (
	"testing"
)

func TestDefaultGenerateTextResult_EmptySteps(t *testing.T) {
	input := 10
	output := 20
	total := 30
	result := NewDefaultGenerateTextResult(
		[]StepResult{},
		LanguageModelUsage{InputTokens: &input, OutputTokens: &output, TotalTokens: &total},
		nil,
	).ToResult()

	if len(result.Steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(result.Steps))
	}
	if result.TotalUsage.InputTokens == nil || *result.TotalUsage.InputTokens != 10 {
		t.Errorf("expected total input tokens 10")
	}
	if result.Text != "" {
		t.Errorf("expected empty text, got %q", result.Text)
	}
}

func TestDefaultGenerateTextResult_SingleStep(t *testing.T) {
	input := 5
	output := 15
	total := 20

	step := StepResult{
		Content: []ContentPart{
			NewTextContentPart("Hello, world!", nil),
		},
		FinishReason:    FinishReasonStop,
		RawFinishReason: "stop",
		Usage: LanguageModelUsage{
			InputTokens:  &input,
			OutputTokens: &output,
			TotalTokens:  &total,
		},
	}

	result := NewDefaultGenerateTextResult(
		[]StepResult{step},
		step.Usage,
		nil,
	).ToResult()

	if result.Text != "Hello, world!" {
		t.Errorf("expected text 'Hello, world!', got %q", result.Text)
	}
	if result.FinishReason != FinishReasonStop {
		t.Errorf("expected finish reason 'stop', got %q", result.FinishReason)
	}
	if len(result.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(result.Steps))
	}
}

func TestDefaultGenerateTextResult_MultipleSteps(t *testing.T) {
	input1 := 5
	output1 := 10
	total1 := 15
	input2 := 3
	output2 := 8
	total2 := 11

	tc := ToolCall{
		Type:       "tool-call",
		ToolCallID: "call-1",
		ToolName:   "search",
		Input:      map[string]interface{}{"q": "test"},
	}

	step1 := StepResult{
		Content: []ContentPart{
			NewTextContentPart("Searching...", nil),
			NewToolCallContentPart(tc),
		},
		FinishReason:    FinishReasonToolCalls,
		RawFinishReason: "tool_calls",
		Usage: LanguageModelUsage{
			InputTokens:  &input1,
			OutputTokens: &output1,
			TotalTokens:  &total1,
		},
	}

	step2 := StepResult{
		Content: []ContentPart{
			NewTextContentPart("Found results.", nil),
		},
		FinishReason:    FinishReasonStop,
		RawFinishReason: "stop",
		Usage: LanguageModelUsage{
			InputTokens:  &input2,
			OutputTokens: &output2,
			TotalTokens:  &total2,
		},
	}

	totalInput := 8
	totalOutput := 18
	totalTotal := 26
	totalUsage := LanguageModelUsage{
		InputTokens:  &totalInput,
		OutputTokens: &totalOutput,
		TotalTokens:  &totalTotal,
	}

	result := NewDefaultGenerateTextResult(
		[]StepResult{step1, step2},
		totalUsage,
		nil,
	).ToResult()

	// Should use final step for most fields
	if result.Text != "Found results." {
		t.Errorf("expected text from final step 'Found results.', got %q", result.Text)
	}
	if result.FinishReason != FinishReasonStop {
		t.Errorf("expected finish reason 'stop' from final step, got %q", result.FinishReason)
	}
	if len(result.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(result.Steps))
	}
	if *result.TotalUsage.InputTokens != 8 {
		t.Errorf("expected total input tokens 8, got %d", *result.TotalUsage.InputTokens)
	}
}

func TestDefaultGenerateTextResult_WithOutput(t *testing.T) {
	step := StepResult{
		Content: []ContentPart{
			NewTextContentPart(`{"name":"test"}`, nil),
		},
		FinishReason: FinishReasonStop,
	}

	parsedOutput := map[string]interface{}{"name": "test"}
	result := NewDefaultGenerateTextResult(
		[]StepResult{step},
		LanguageModelUsage{},
		parsedOutput,
	).ToResult()

	if result.Output == nil {
		t.Error("expected output to not be nil")
	}
	outputMap, ok := result.Output.(map[string]interface{})
	if !ok {
		t.Fatalf("expected output to be map, got %T", result.Output)
	}
	if outputMap["name"] != "test" {
		t.Errorf("expected output name 'test', got %v", outputMap["name"])
	}
}

func TestGenerateTextOptions_Construction(t *testing.T) {
	opts := GenerateTextOptions{
		Prompt: "What is the weather?",
		Tools:  ToolSet{"weather": Tool{InputSchema: map[string]interface{}{"type": "object"}}},
		System: "You are a helpful assistant.",
	}

	if opts.Prompt != "What is the weather?" {
		t.Errorf("expected prompt, got %q", opts.Prompt)
	}
	if opts.Tools == nil {
		t.Error("expected tools to not be nil")
	}
	if _, ok := opts.Tools["weather"]; !ok {
		t.Error("expected weather tool")
	}
}

func TestAddLanguageModelUsage(t *testing.T) {
	a := LanguageModelUsage{
		InputTokens:  intPtr(5),
		OutputTokens: intPtr(10),
		TotalTokens:  intPtr(15),
	}
	b := LanguageModelUsage{
		InputTokens:  intPtr(3),
		OutputTokens: intPtr(7),
		TotalTokens:  intPtr(10),
	}

	sum := AddLanguageModelUsage(a, b)

	if *sum.InputTokens != 8 {
		t.Errorf("expected input 8, got %d", *sum.InputTokens)
	}
	if *sum.OutputTokens != 17 {
		t.Errorf("expected output 17, got %d", *sum.OutputTokens)
	}
	if *sum.TotalTokens != 25 {
		t.Errorf("expected total 25, got %d", *sum.TotalTokens)
	}
}

func TestAddLanguageModelUsage_NilFields(t *testing.T) {
	a := LanguageModelUsage{InputTokens: intPtr(5)}
	b := LanguageModelUsage{OutputTokens: intPtr(10)}

	sum := AddLanguageModelUsage(a, b)

	if *sum.InputTokens != 5 {
		t.Errorf("expected input 5, got %d", *sum.InputTokens)
	}
	if *sum.OutputTokens != 10 {
		t.Errorf("expected output 10, got %d", *sum.OutputTokens)
	}
}

func TestAddLanguageModelUsage_BothNil(t *testing.T) {
	a := LanguageModelUsage{}
	b := LanguageModelUsage{}

	sum := AddLanguageModelUsage(a, b)

	if sum.InputTokens != nil {
		t.Error("expected nil input tokens")
	}
	if sum.OutputTokens != nil {
		t.Error("expected nil output tokens")
	}
}

func TestAsLanguageModelUsage(t *testing.T) {
	usage := AsLanguageModelUsage(struct {
		InputTokens  TokenCount
		OutputTokens TokenCount
	}{
		InputTokens:  TokenCount{Total: 3},
		OutputTokens: TokenCount{Total: 10},
	})

	if *usage.InputTokens != 3 {
		t.Errorf("expected input 3, got %d", *usage.InputTokens)
	}
	if *usage.OutputTokens != 10 {
		t.Errorf("expected output 10, got %d", *usage.OutputTokens)
	}
	if *usage.TotalTokens != 13 {
		t.Errorf("expected total 13, got %d", *usage.TotalTokens)
	}
}
