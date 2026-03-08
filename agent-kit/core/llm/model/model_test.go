// Ported from: packages/core/src/llm/model/model.test.ts
package model

import (
	"testing"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
)

// ---------------------------------------------------------------------------
// Mock types for model tests
// ---------------------------------------------------------------------------

// mockLogger implements logger.IMastraLogger for testing.
type mockLogger struct {
	debugCalls []mockLogCall
	warnCalls  []mockLogCall
	infoCalls  []mockLogCall
	errorCalls []mockLogCall
}

type mockLogCall struct {
	Message string
	Args    []any
}

func newMockLogger() *mockLogger {
	return &mockLogger{}
}

func (l *mockLogger) Debug(message string, args ...any) {
	l.debugCalls = append(l.debugCalls, mockLogCall{Message: message, Args: args})
}
func (l *mockLogger) Info(message string, args ...any) {
	l.infoCalls = append(l.infoCalls, mockLogCall{Message: message, Args: args})
}
func (l *mockLogger) Warn(message string, args ...any) {
	l.warnCalls = append(l.warnCalls, mockLogCall{Message: message, Args: args})
}
func (l *mockLogger) Error(message string, args ...any) {
	l.errorCalls = append(l.errorCalls, mockLogCall{Message: message, Args: args})
}
func (l *mockLogger) TrackException(err *mastraerror.MastraBaseError) {}
func (l *mockLogger) GetTransports() map[string]logger.LoggerTransport {
	return nil
}
func (l *mockLogger) ListLogs(transportID string, params *logger.ListLogsParams) (logger.LogResult, error) {
	return logger.LogResult{}, nil
}
func (l *mockLogger) ListLogsByRunID(args *logger.ListLogsByRunIDFullArgs) (logger.LogResult, error) {
	return logger.LogResult{}, nil
}

// mockMastraRef implements MastraRef for testing.
type mockMastraRef struct {
	logger logger.IMastraLogger
}

func (m *mockMastraRef) GetLogger() logger.IMastraLogger {
	return m.logger
}

// mockLanguageModelV1 implements LanguageModelV1 for testing.
type mockLanguageModelV1ForTest struct {
	specVersion string
	provider    string
	modelID     string
}

func (m *mockLanguageModelV1ForTest) SpecificationVersion() string { return m.specVersion }
func (m *mockLanguageModelV1ForTest) Provider() string             { return m.provider }
func (m *mockLanguageModelV1ForTest) ModelID() string              { return m.modelID }

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestMastraLLMV1(t *testing.T) {
	ml := newMockLogger()
	mastra := &mockMastraRef{logger: ml}

	mockModel := &mockLanguageModelV1ForTest{
		specVersion: "v1",
		provider:    "test-provider",
		modelID:     "test-model",
	}

	t.Run("constructor", func(t *testing.T) {
		t.Run("should initialize with model only", func(t *testing.T) {
			llm := NewMastraLLMV1(MastraLLMV1Config{
				Model: mockModel,
			})
			if llm == nil {
				t.Fatal("expected non-nil MastraLLMV1")
			}
		})

		t.Run("should initialize with both model and mastra", func(t *testing.T) {
			llm := NewMastraLLMV1(MastraLLMV1Config{
				Model:  mockModel,
				Mastra: mastra,
			})
			if llm == nil {
				t.Fatal("expected non-nil MastraLLMV1")
			}
		})
	})

	t.Run("generate", func(t *testing.T) {
		llm := NewMastraLLMV1(MastraLLMV1Config{
			Model:  mockModel,
			Mastra: mastra,
		})

		t.Run("should generate text output by default", func(t *testing.T) {
			t.Skip("not yet implemented - generateText requires AI SDK integration")
			messages := []CoreMessage{{Role: "user", Content: "test message"}}
			temp := 0.7
			_, err := llm.Generate(messages, &GenerateOptions{
				Temperature: &temp,
				MaxSteps:    5,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})

		t.Run("should generate structured output when output is provided", func(t *testing.T) {
			t.Skip("not yet implemented - generateObject requires AI SDK integration")
			messages := []CoreMessage{{Role: "user", Content: "test message"}}
			temp := 0.7
			schema := map[string]any{"type": "object", "properties": map[string]any{"content": map[string]any{"type": "string"}}}
			_, err := llm.Generate(messages, &GenerateOptions{
				Temperature: &temp,
				Output:      schema,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})

		t.Run("should convert string message to CoreMessage format", func(t *testing.T) {
			msgs := llm.ConvertToMessages("test message")
			if len(msgs) != 1 {
				t.Fatalf("expected 1 message, got %d", len(msgs))
			}
			if msgs[0].Role != "user" {
				t.Errorf("role = %q, want %q", msgs[0].Role, "user")
			}
			if msgs[0].Content != "test message" {
				t.Errorf("content = %v, want %q", msgs[0].Content, "test message")
			}
		})

		t.Run("should convert string array to CoreMessage format", func(t *testing.T) {
			msgs := llm.ConvertToMessages([]string{"message 1", "message 2"})
			if len(msgs) != 2 {
				t.Fatalf("expected 2 messages, got %d", len(msgs))
			}
			if msgs[0].Content != "message 1" {
				t.Errorf("content[0] = %v, want %q", msgs[0].Content, "message 1")
			}
			if msgs[1].Content != "message 2" {
				t.Errorf("content[1] = %v, want %q", msgs[1].Content, "message 2")
			}
		})

		t.Run("should pass through tool conversion", func(t *testing.T) {
			t.Skip("not yet implemented - generateText requires AI SDK integration")
		})

		t.Run("should handle onStepFinish callback", func(t *testing.T) {
			t.Skip("not yet implemented - generateText requires AI SDK integration")
		})
	})

	t.Run("stream", func(t *testing.T) {
		llm := NewMastraLLMV1(MastraLLMV1Config{
			Model:  mockModel,
			Mastra: mastra,
		})

		t.Run("should stream text by default", func(t *testing.T) {
			t.Skip("not yet implemented - streamText requires AI SDK integration")
			messages := []CoreMessage{{Role: "user", Content: "test message"}}
			temp := 0.7
			_, err := llm.Stream(messages, &StreamOptions{
				Temperature: &temp,
				MaxSteps:    5,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})

		t.Run("should handle string messages", func(t *testing.T) {
			t.Skip("not yet implemented - streamText requires AI SDK integration")
		})

		t.Run("should handle array of string messages", func(t *testing.T) {
			t.Skip("not yet implemented - streamText requires AI SDK integration")
		})

		t.Run("should stream structured output with schema", func(t *testing.T) {
			t.Skip("not yet implemented - streamObject requires AI SDK integration")
		})

		t.Run("should stream structured output with JSON schema", func(t *testing.T) {
			t.Skip("not yet implemented - streamObject requires AI SDK integration")
		})

		t.Run("should handle callbacks for text streaming", func(t *testing.T) {
			t.Skip("not yet implemented - streamText requires AI SDK integration")
		})

		t.Run("should handle callbacks for structured output streaming", func(t *testing.T) {
			t.Skip("not yet implemented - streamObject requires AI SDK integration")
		})
	})

	t.Run("__text", func(t *testing.T) {
		t.Run("should generate text with correct parameters", func(t *testing.T) {
			t.Skip("not yet implemented - generateText requires AI SDK integration")
		})

		t.Run("should handle tool conversion", func(t *testing.T) {
			t.Skip("not yet implemented - generateText requires AI SDK integration")
		})

		t.Run("should handle pre-converted tools", func(t *testing.T) {
			t.Skip("not yet implemented - generateText requires AI SDK integration")
		})

		t.Run("should handle onStepFinish callback", func(t *testing.T) {
			t.Skip("not yet implemented - generateText requires AI SDK integration")
		})

		t.Run("should handle rate limiting", func(t *testing.T) {
			t.Skip("not yet implemented - generateText requires AI SDK integration")
		})

		t.Run("should log debug messages", func(t *testing.T) {
			t.Skip("not yet implemented - generateText requires AI SDK integration")
		})

		t.Run("should handle step change logging", func(t *testing.T) {
			t.Skip("not yet implemented - generateText requires AI SDK integration")
		})
	})

	t.Run("__stream", func(t *testing.T) {
		t.Run("should stream text with correct parameters", func(t *testing.T) {
			t.Skip("not yet implemented - streamText requires AI SDK integration")
		})

		t.Run("should handle tool conversion", func(t *testing.T) {
			t.Skip("not yet implemented - streamText requires AI SDK integration")
		})

		t.Run("should handle pre-converted tools", func(t *testing.T) {
			t.Skip("not yet implemented - streamText requires AI SDK integration")
		})

		t.Run("should handle callbacks", func(t *testing.T) {
			t.Skip("not yet implemented - streamText requires AI SDK integration")
		})

		t.Run("should log debug messages", func(t *testing.T) {
			llmLocal := NewMastraLLMV1(MastraLLMV1Config{
				Model:  mockModel,
				Mastra: mastra,
			})
			messages := []CoreMessage{{Role: "user", Content: "test message"}}
			temp := 0.7
			// Calling Stream will call Debug internally even though streamText is not implemented
			_, _ = llmLocal.Stream(messages, &StreamOptions{
				RunID:       "test-run",
				Temperature: &temp,
				MaxSteps:    5,
			})
			// Verify debug was called (the Stream method logs before attempting streamText)
			found := false
			for _, call := range ml.debugCalls {
				if call.Message == "[LLM] - Streaming text" {
					found = true
					break
				}
			}
			if !found {
				t.Error("expected debug log '[LLM] - Streaming text' to be called")
			}
		})

		t.Run("should handle step change logging", func(t *testing.T) {
			t.Skip("not yet implemented - streamText requires AI SDK integration")
		})
	})

	t.Run("__textObject", func(t *testing.T) {
		t.Run("should generate structured output with schema", func(t *testing.T) {
			t.Skip("not yet implemented - generateObject requires AI SDK integration")
		})

		t.Run("should handle array type schemas", func(t *testing.T) {
			t.Skip("not yet implemented - generateObject requires AI SDK integration")
		})

		t.Run("should handle JSON schema input", func(t *testing.T) {
			t.Skip("not yet implemented - generateObject requires AI SDK integration")
		})

		t.Run("should integrate tools correctly", func(t *testing.T) {
			t.Skip("not yet implemented - generateObject requires AI SDK integration")
		})
	})

	t.Run("__streamObject", func(t *testing.T) {
		t.Run("should stream object with schema", func(t *testing.T) {
			t.Skip("not yet implemented - streamObject requires AI SDK integration")
		})

		t.Run("should handle array type schemas", func(t *testing.T) {
			t.Skip("not yet implemented - streamObject requires AI SDK integration")
		})

		t.Run("should handle JSON schema input", func(t *testing.T) {
			t.Skip("not yet implemented - streamObject requires AI SDK integration")
		})

		t.Run("should handle callbacks", func(t *testing.T) {
			t.Skip("not yet implemented - streamObject requires AI SDK integration")
		})

		t.Run("should log debug messages", func(t *testing.T) {
			t.Skip("not yet implemented - streamObject requires AI SDK integration")
		})

		t.Run("should handle pre-converted tools", func(t *testing.T) {
			t.Skip("not yet implemented - streamObject requires AI SDK integration")
		})

		t.Run("should handle rate limiting", func(t *testing.T) {
			t.Skip("not yet implemented - streamObject requires AI SDK integration")
		})
	})

	t.Run("error logging via Mastra logger (issue #12184)", func(t *testing.T) {
		t.Run("should log error through Mastra logger when __text (generateText) fails", func(t *testing.T) {
			t.Skip("not yet implemented - generateText requires AI SDK integration")
		})

		t.Run("should log error through Mastra logger when __textObject (generateObject) fails", func(t *testing.T) {
			t.Skip("not yet implemented - generateObject requires AI SDK integration")
		})

		t.Run("should log streaming error through Mastra logger when __stream (streamText) fails", func(t *testing.T) {
			t.Skip("not yet implemented - streamText requires AI SDK integration")
		})

		t.Run("should log streaming error through Mastra logger when __streamObject (streamObject) fails", func(t *testing.T) {
			t.Skip("not yet implemented - streamObject requires AI SDK integration")
		})
	})
}

func TestMastraLLMV1_GetProvider(t *testing.T) {
	mockModel := &mockLanguageModelV1ForTest{
		specVersion: "v1",
		provider:    "test-provider",
		modelID:     "test-model",
	}
	llm := NewMastraLLMV1(MastraLLMV1Config{Model: mockModel})
	if got := llm.GetProvider(); got != "test-provider" {
		t.Errorf("GetProvider() = %q, want %q", got, "test-provider")
	}
}

func TestMastraLLMV1_GetModelID(t *testing.T) {
	mockModel := &mockLanguageModelV1ForTest{
		specVersion: "v1",
		provider:    "test-provider",
		modelID:     "test-model",
	}
	llm := NewMastraLLMV1(MastraLLMV1Config{Model: mockModel})
	if got := llm.GetModelID(); got != "test-model" {
		t.Errorf("GetModelID() = %q, want %q", got, "test-model")
	}
}

func TestMastraLLMV1_RegisterPrimitives(t *testing.T) {
	mockModel := &mockLanguageModelV1ForTest{
		specVersion: "v1",
		provider:    "test-provider",
		modelID:     "test-model",
	}
	ml := newMockLogger()
	llm := NewMastraLLMV1(MastraLLMV1Config{Model: mockModel})
	llm.RegisterPrimitives(MastraPrimitives{Logger: ml})
	// After registering primitives, the logger should be the mock logger.
	// Compare as interface since Logger() returns logger.IMastraLogger.
	if llm.Logger() != logger.IMastraLogger(ml) {
		t.Error("expected logger to be updated after RegisterPrimitives")
	}
}
