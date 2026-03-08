// Ported from: packages/ai/src/generate-text/parse-tool-call.test.ts
package generatetext

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestParseToolCall_ValidToolCall(t *testing.T) {
	result, err := ParseToolCall(ParseToolCallOptions{
		ToolCall: LanguageModelV4ToolCall{
			Type:       "tool-call",
			ToolName:   "testTool",
			ToolCallID: "123",
			Input:      `{"param1": "test", "param2": 42}`,
		},
		Tools: ToolSet{
			"testTool": Tool{
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"param1": map[string]interface{}{"type": "string"},
						"param2": map[string]interface{}{"type": "number"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Type != "tool-call" {
		t.Errorf("expected type 'tool-call', got %q", result.Type)
	}
	if result.ToolCallID != "123" {
		t.Errorf("expected toolCallId '123', got %q", result.ToolCallID)
	}
	if result.ToolName != "testTool" {
		t.Errorf("expected toolName 'testTool', got %q", result.ToolName)
	}

	inputMap, ok := result.Input.(map[string]interface{})
	if !ok {
		t.Fatalf("expected input to be a map, got %T", result.Input)
	}
	if inputMap["param1"] != "test" {
		t.Errorf("expected param1 'test', got %v", inputMap["param1"])
	}
	if inputMap["param2"] != float64(42) {
		t.Errorf("expected param2 42, got %v", inputMap["param2"])
	}
}

func TestParseToolCall_ValidProviderExecutedDynamicToolCall(t *testing.T) {
	result, err := ParseToolCall(ParseToolCallOptions{
		ToolCall: LanguageModelV4ToolCall{
			Type:             "tool-call",
			ToolName:         "testTool",
			ToolCallID:       "123",
			Input:            `{"param1": "test", "param2": 42}`,
			ProviderExecuted: true,
			Dynamic:          true,
			ProviderMetadata: ProviderMetadata{
				"testProvider": {"signature": "sig"},
			},
		},
		Tools: ToolSet{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Type != "tool-call" {
		t.Errorf("expected type 'tool-call', got %q", result.Type)
	}
	if !result.Dynamic {
		t.Error("expected dynamic to be true")
	}
	if !result.ProviderExecuted {
		t.Error("expected providerExecuted to be true")
	}
	if result.ProviderMetadata == nil {
		t.Error("expected providerMetadata to not be nil")
	}
	if result.ProviderMetadata["testProvider"]["signature"] != "sig" {
		t.Errorf("expected testProvider.signature 'sig', got %v", result.ProviderMetadata["testProvider"]["signature"])
	}
}

func TestParseToolCall_ValidToolCallWithProviderMetadata(t *testing.T) {
	result, err := ParseToolCall(ParseToolCallOptions{
		ToolCall: LanguageModelV4ToolCall{
			Type:       "tool-call",
			ToolName:   "testTool",
			ToolCallID: "123",
			Input:      `{"param1": "test", "param2": 42}`,
			ProviderMetadata: ProviderMetadata{
				"testProvider": {"signature": "sig"},
			},
		},
		Tools: ToolSet{
			"testTool": Tool{
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"param1": map[string]interface{}{"type": "string"},
						"param2": map[string]interface{}{"type": "number"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProviderMetadata == nil {
		t.Fatal("expected providerMetadata to not be nil")
	}
	if result.ProviderMetadata["testProvider"]["signature"] != "sig" {
		t.Errorf("expected testProvider.signature 'sig', got %v", result.ProviderMetadata["testProvider"]["signature"])
	}
}

func TestParseToolCall_EmptyInputNoSchema(t *testing.T) {
	result, err := ParseToolCall(ParseToolCallOptions{
		ToolCall: LanguageModelV4ToolCall{
			Type:       "tool-call",
			ToolName:   "testTool",
			ToolCallID: "123",
			Input:      "",
		},
		Tools: ToolSet{
			"testTool": Tool{
				InputSchema: map[string]interface{}{"type": "object"},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inputMap, ok := result.Input.(map[string]interface{})
	if !ok {
		t.Fatalf("expected input to be a map, got %T", result.Input)
	}
	if len(inputMap) != 0 {
		t.Errorf("expected empty map, got %v", inputMap)
	}
}

func TestParseToolCall_EmptyObjectInputNoSchema(t *testing.T) {
	result, err := ParseToolCall(ParseToolCallOptions{
		ToolCall: LanguageModelV4ToolCall{
			Type:       "tool-call",
			ToolName:   "testTool",
			ToolCallID: "123",
			Input:      "{}",
		},
		Tools: ToolSet{
			"testTool": Tool{
				InputSchema: map[string]interface{}{"type": "object"},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inputMap, ok := result.Input.(map[string]interface{})
	if !ok {
		t.Fatalf("expected input to be a map, got %T", result.Input)
	}
	if len(inputMap) != 0 {
		t.Errorf("expected empty map, got %v", inputMap)
	}
}

func TestParseToolCall_NoSuchToolWhenToolsNil(t *testing.T) {
	result, err := ParseToolCall(ParseToolCallOptions{
		ToolCall: LanguageModelV4ToolCall{
			Type:       "tool-call",
			ToolName:   "testTool",
			ToolCallID: "123",
			Input:      "{}",
		},
		Tools: nil,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Invalid {
		t.Error("expected invalid to be true")
	}
	if !result.Dynamic {
		t.Error("expected dynamic to be true")
	}
	if result.Error == nil {
		t.Error("expected error to be set")
	}
	if _, ok := result.Error.(*NoSuchToolError); !ok {
		t.Errorf("expected NoSuchToolError, got %T", result.Error)
	}
}

func TestParseToolCall_NoSuchToolWhenToolNotFound(t *testing.T) {
	result, err := ParseToolCall(ParseToolCallOptions{
		ToolCall: LanguageModelV4ToolCall{
			Type:       "tool-call",
			ToolName:   "nonExistentTool",
			ToolCallID: "123",
			Input:      "{}",
		},
		Tools: ToolSet{
			"testTool": Tool{
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"param1": map[string]interface{}{"type": "string"},
						"param2": map[string]interface{}{"type": "number"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Invalid {
		t.Error("expected invalid to be true")
	}
	if !result.Dynamic {
		t.Error("expected dynamic to be true")
	}
	nste, ok := result.Error.(*NoSuchToolError)
	if !ok {
		t.Fatalf("expected NoSuchToolError, got %T", result.Error)
	}
	if nste.ToolName != "nonExistentTool" {
		t.Errorf("expected tool name 'nonExistentTool', got %q", nste.ToolName)
	}
}

func TestParseToolCall_InvalidToolInput(t *testing.T) {
	result, err := ParseToolCall(ParseToolCallOptions{
		ToolCall: LanguageModelV4ToolCall{
			Type:       "tool-call",
			ToolName:   "testTool",
			ToolCallID: "123",
			Input:      "invalid json",
		},
		Tools: ToolSet{
			"testTool": Tool{
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"param1": map[string]interface{}{"type": "string"},
						"param2": map[string]interface{}{"type": "number"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Invalid {
		t.Error("expected invalid to be true")
	}
	if result.Error == nil {
		t.Error("expected error to be set")
	}
}

func TestParseToolCall_RepairToolCall_Success(t *testing.T) {
	repairCalled := false
	repairToolCall := func(opts ToolCallRepairOptions) (*LanguageModelV4ToolCall, error) {
		repairCalled = true
		return &LanguageModelV4ToolCall{
			Type:       "tool-call",
			ToolName:   "testTool",
			ToolCallID: "123",
			Input:      `{"param1": "test", "param2": 42}`,
		}, nil
	}

	result, err := ParseToolCall(ParseToolCallOptions{
		ToolCall: LanguageModelV4ToolCall{
			Type:       "tool-call",
			ToolName:   "testTool",
			ToolCallID: "123",
			Input:      "invalid json",
		},
		Tools: ToolSet{
			"testTool": Tool{
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"param1": map[string]interface{}{"type": "string"},
						"param2": map[string]interface{}{"type": "number"},
					},
				},
			},
		},
		RepairToolCall: repairToolCall,
		Messages:       []ModelMessage{{Role: "user", Content: "test message"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !repairCalled {
		t.Error("expected repair function to be called")
	}
	if result.Invalid {
		t.Error("expected invalid to be false after repair")
	}

	inputMap, ok := result.Input.(map[string]interface{})
	if !ok {
		t.Fatalf("expected input to be a map, got %T", result.Input)
	}
	if inputMap["param1"] != "test" {
		t.Errorf("expected param1 'test', got %v", inputMap["param1"])
	}
}

func TestParseToolCall_RepairToolCall_ReturnsNull(t *testing.T) {
	repairToolCall := func(opts ToolCallRepairOptions) (*LanguageModelV4ToolCall, error) {
		return nil, nil
	}

	result, err := ParseToolCall(ParseToolCallOptions{
		ToolCall: LanguageModelV4ToolCall{
			Type:       "tool-call",
			ToolName:   "testTool",
			ToolCallID: "123",
			Input:      "invalid json",
		},
		Tools: ToolSet{
			"testTool": Tool{
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"param1": map[string]interface{}{"type": "string"},
						"param2": map[string]interface{}{"type": "number"},
					},
				},
			},
		},
		RepairToolCall: repairToolCall,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Invalid {
		t.Error("expected invalid to be true when repair returns nil")
	}
}

func TestParseToolCall_RepairToolCall_Throws(t *testing.T) {
	repairToolCall := func(opts ToolCallRepairOptions) (*LanguageModelV4ToolCall, error) {
		return nil, errors.New("test error")
	}

	result, err := ParseToolCall(ParseToolCallOptions{
		ToolCall: LanguageModelV4ToolCall{
			Type:       "tool-call",
			ToolName:   "testTool",
			ToolCallID: "123",
			Input:      "invalid json",
		},
		Tools: ToolSet{
			"testTool": Tool{
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"param1": map[string]interface{}{"type": "string"},
						"param2": map[string]interface{}{"type": "number"},
					},
				},
			},
		},
		RepairToolCall: repairToolCall,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Invalid {
		t.Error("expected invalid to be true when repair throws")
	}
	if _, ok := result.Error.(*ToolCallRepairError); !ok {
		t.Errorf("expected ToolCallRepairError, got %T", result.Error)
	}
}

func TestParseToolCall_DynamicTool(t *testing.T) {
	result, err := ParseToolCall(ParseToolCallOptions{
		ToolCall: LanguageModelV4ToolCall{
			Type:       "tool-call",
			ToolName:   "testTool",
			ToolCallID: "123",
			Input:      `{"param1": "test", "param2": 42}`,
		},
		Tools: ToolSet{
			"testTool": Tool{
				Type: "dynamic",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"param1": map[string]interface{}{"type": "string"},
						"param2": map[string]interface{}{"type": "number"},
					},
				},
				Execute: func(input interface{}, opts ToolExecuteOptions) (interface{}, error) {
					return "result", nil
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Dynamic {
		t.Error("expected dynamic to be true for dynamic tool")
	}
}

func TestParseToolCall_ToolTitle_Dynamic(t *testing.T) {
	result, err := ParseToolCall(ParseToolCallOptions{
		ToolCall: LanguageModelV4ToolCall{
			Type:       "tool-call",
			ToolCallID: "call-1",
			ToolName:   "weather",
			Input:      `{"location":"Paris"}`,
		},
		Tools: ToolSet{
			"weather": Tool{
				Type:  "dynamic",
				Title: "Weather Information",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{"type": "string"},
					},
				},
				Execute: func(input interface{}, opts ToolExecuteOptions) (interface{}, error) {
					return "sunny", nil
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Title != "Weather Information" {
		t.Errorf("expected title 'Weather Information', got %q", result.Title)
	}
	if !result.Dynamic {
		t.Error("expected dynamic to be true")
	}
}

func TestParseToolCall_ToolTitle_Static(t *testing.T) {
	result, err := ParseToolCall(ParseToolCallOptions{
		ToolCall: LanguageModelV4ToolCall{
			Type:       "tool-call",
			ToolCallID: "call-2",
			ToolName:   "calculator",
			Input:      `{"a":5,"b":3}`,
		},
		Tools: ToolSet{
			"calculator": Tool{
				Title: "Calculator",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"a": map[string]interface{}{"type": "number"},
						"b": map[string]interface{}{"type": "number"},
					},
				},
				Execute: func(input interface{}, opts ToolExecuteOptions) (interface{}, error) {
					return 8, nil
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Title != "Calculator" {
		t.Errorf("expected title 'Calculator', got %q", result.Title)
	}
	if result.Dynamic {
		t.Error("expected dynamic to be false for static tool")
	}
}

func TestParseToolCall_ToolTitle_InvalidToolCall(t *testing.T) {
	result, err := ParseToolCall(ParseToolCallOptions{
		ToolCall: LanguageModelV4ToolCall{
			Type:       "tool-call",
			ToolCallID: "call-4",
			ToolName:   "invalidTool",
			Input:      "invalid json",
		},
		Tools: ToolSet{
			"invalidTool": Tool{
				Title: "Invalid Tool",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"required": map[string]interface{}{"type": "string"},
					},
				},
				Execute: func(input interface{}, opts ToolExecuteOptions) (interface{}, error) {
					return "result", nil
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Invalid {
		t.Error("expected invalid to be true")
	}
	if result.Title != "Invalid Tool" {
		t.Errorf("expected title 'Invalid Tool', got %q", result.Title)
	}
}

// tryParseJSON helper test
func TestTryParseJSON(t *testing.T) {
	// Valid JSON
	result := tryParseJSON(`{"key": "value"}`)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if m["key"] != "value" {
		t.Errorf("expected 'value', got %v", m["key"])
	}

	// Invalid JSON returns the raw string
	result = tryParseJSON("invalid json")
	s, ok := result.(string)
	if !ok {
		t.Fatalf("expected string, got %T", result)
	}
	if s != "invalid json" {
		t.Errorf("expected 'invalid json', got %q", s)
	}
}

// Ensure JSON serialization roundtrips work
func TestParseToolCall_JSONRoundtrip(t *testing.T) {
	result, err := ParseToolCall(ParseToolCallOptions{
		ToolCall: LanguageModelV4ToolCall{
			Type:       "tool-call",
			ToolName:   "testTool",
			ToolCallID: "123",
			Input:      `{"param1": "test", "param2": 42}`,
		},
		Tools: ToolSet{
			"testTool": Tool{
				InputSchema: map[string]interface{}{
					"type": "object",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Ensure the result can be marshalled/unmarshalled
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
}
