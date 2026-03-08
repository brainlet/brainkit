// Ported from: packages/core/src/processors/runner.test.ts
package processors

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// mockLogger implements logger.IMastraLogger for tests.
type mockLogger struct{}

func (m *mockLogger) Debug(message string, args ...any)                               {}
func (m *mockLogger) Info(message string, args ...any)                                {}
func (m *mockLogger) Warn(message string, args ...any)                                {}
func (m *mockLogger) Error(message string, args ...any)                               {}
func (m *mockLogger) TrackException(err *mastraerror.MastraBaseError)                  {}
func (m *mockLogger) GetTransports() map[string]logger.LoggerTransport                { return nil }
func (m *mockLogger) ListLogs(_ string, _ *logger.ListLogsParams) (logger.LogResult, error) { return logger.LogResult{}, nil }
func (m *mockLogger) ListLogsByRunID(_ *logger.ListLogsByRunIDFullArgs) (logger.LogResult, error) { return logger.LogResult{}, nil }

func newMockLogger() logger.IMastraLogger { return &mockLogger{} }

func createMessage(content string, role string) MastraDBMessage {
	if role == "" {
		role = "user"
	}
	return MastraDBMessage{
		ID:   fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		Role: role,
		Content: MastraMessageContentV2{
			Format: 2,
			Parts:  []MessagePart{{Type: "text", Text: content}},
		},
		CreatedAt: time.Now(),
		ThreadID:  "test-thread",
	}
}

// ---------------------------------------------------------------------------
// testInputProcessor - a mock that implements InputProcessor
// ---------------------------------------------------------------------------

type testInputProcessor struct {
	BaseProcessor
	processInputFn     func(ProcessInputArgs) ([]MastraDBMessage, *MessageList, *ProcessInputResultWithSystemMessages, error)
	processInputStepFn func(ProcessInputStepArgs) (*ProcessInputStepResult, []MastraDBMessage, error)
}

func (p *testInputProcessor) ProcessInput(args ProcessInputArgs) ([]MastraDBMessage, *MessageList, *ProcessInputResultWithSystemMessages, error) {
	if p.processInputFn != nil {
		return p.processInputFn(args)
	}
	return nil, nil, nil, nil
}

func (p *testInputProcessor) ProcessInputStep(args ProcessInputStepArgs) (*ProcessInputStepResult, []MastraDBMessage, error) {
	if p.processInputStepFn != nil {
		return p.processInputStepFn(args)
	}
	return nil, nil, nil
}

// Satisfy OutputProcessor interface methods so it can be used as any in the runner
func (p *testInputProcessor) ProcessOutputStream(args ProcessOutputStreamArgs) (*ChunkType, error) {
	return &args.Part, nil
}
func (p *testInputProcessor) ProcessOutputResult(args ProcessOutputResultArgs) ([]MastraDBMessage, *MessageList, error) {
	return nil, nil, nil
}
func (p *testInputProcessor) ProcessOutputStep(args ProcessOutputStepArgs) ([]MastraDBMessage, *MessageList, error) {
	return nil, nil, nil
}

// ---------------------------------------------------------------------------
// testOutputProcessor - a mock that implements OutputProcessor
// ---------------------------------------------------------------------------

type testOutputProcessor struct {
	BaseProcessor
	processOutputStreamFn func(ProcessOutputStreamArgs) (*ChunkType, error)
	processOutputResultFn func(ProcessOutputResultArgs) ([]MastraDBMessage, *MessageList, error)
	processOutputStepFn   func(ProcessOutputStepArgs) ([]MastraDBMessage, *MessageList, error)
}

func (p *testOutputProcessor) ProcessOutputStream(args ProcessOutputStreamArgs) (*ChunkType, error) {
	if p.processOutputStreamFn != nil {
		return p.processOutputStreamFn(args)
	}
	return &args.Part, nil
}
func (p *testOutputProcessor) ProcessOutputResult(args ProcessOutputResultArgs) ([]MastraDBMessage, *MessageList, error) {
	if p.processOutputResultFn != nil {
		return p.processOutputResultFn(args)
	}
	return nil, nil, nil
}
func (p *testOutputProcessor) ProcessOutputStep(args ProcessOutputStepArgs) ([]MastraDBMessage, *MessageList, error) {
	if p.processOutputStepFn != nil {
		return p.processOutputStepFn(args)
	}
	return nil, nil, nil
}

// Satisfy InputProcessor interface methods
func (p *testOutputProcessor) ProcessInput(args ProcessInputArgs) ([]MastraDBMessage, *MessageList, *ProcessInputResultWithSystemMessages, error) {
	return nil, nil, nil, nil
}
func (p *testOutputProcessor) ProcessInputStep(args ProcessInputStepArgs) (*ProcessInputStepResult, []MastraDBMessage, error) {
	return nil, nil, nil
}

// ---------------------------------------------------------------------------
// Tests: ProcessorRunner > Input Processors
// ---------------------------------------------------------------------------

func TestProcessorRunner_InputProcessors(t *testing.T) {
	t.Run("should run input processors in order", func(t *testing.T) {
		executionOrder := []string{}
		p1 := &testInputProcessor{
			BaseProcessor: NewBaseProcessor("processor1", "Processor 1"),
			processInputFn: func(args ProcessInputArgs) ([]MastraDBMessage, *MessageList, *ProcessInputResultWithSystemMessages, error) {
				executionOrder = append(executionOrder, "processor1")
				return nil, nil, nil, nil
			},
		}
		p2 := &testInputProcessor{
			BaseProcessor: NewBaseProcessor("processor2", "Processor 2"),
			processInputFn: func(args ProcessInputArgs) ([]MastraDBMessage, *MessageList, *ProcessInputResultWithSystemMessages, error) {
				executionOrder = append(executionOrder, "processor2")
				return nil, nil, nil, nil
			},
		}

		runner := NewProcessorRunner(ProcessorRunnerConfig{
			InputProcessors:  []any{p1, p2},
			OutputProcessors: []any{},
			Logger:           newMockLogger(),
			AgentName:        "test-agent",
		})

		messageList := &MessageList{}
		_, err := runner.RunInputProcessors(messageList, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(executionOrder, []string{"processor1", "processor2"}) {
			t.Errorf("expected execution order [processor1, processor2], got %v", executionOrder)
		}
	})

	t.Run("should run input processors sequentially in order", func(t *testing.T) {
		executionOrder := []string{}
		p1 := &testInputProcessor{
			BaseProcessor: NewBaseProcessor("processor1", "Processor 1"),
			processInputFn: func(args ProcessInputArgs) ([]MastraDBMessage, *MessageList, *ProcessInputResultWithSystemMessages, error) {
				executionOrder = append(executionOrder, "processor1-start")
				time.Sleep(10 * time.Millisecond)
				executionOrder = append(executionOrder, "processor1-end")
				return nil, nil, nil, nil
			},
		}
		p2 := &testInputProcessor{
			BaseProcessor: NewBaseProcessor("processor2", "Processor 2"),
			processInputFn: func(args ProcessInputArgs) ([]MastraDBMessage, *MessageList, *ProcessInputResultWithSystemMessages, error) {
				executionOrder = append(executionOrder, "processor2-start")
				time.Sleep(10 * time.Millisecond)
				executionOrder = append(executionOrder, "processor2-end")
				return nil, nil, nil, nil
			},
		}

		runner := NewProcessorRunner(ProcessorRunnerConfig{
			InputProcessors:  []any{p1, p2},
			OutputProcessors: []any{},
			Logger:           newMockLogger(),
			AgentName:        "test-agent",
		})

		messageList := &MessageList{}
		_, err := runner.RunInputProcessors(messageList, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := []string{"processor1-start", "processor1-end", "processor2-start", "processor2-end"}
		if !reflect.DeepEqual(executionOrder, expected) {
			t.Errorf("expected %v, got %v", expected, executionOrder)
		}
	})

	t.Run("should abort if tripwire is triggered in input processor", func(t *testing.T) {
		p1 := &testInputProcessor{
			BaseProcessor: NewBaseProcessor("processor1", "Processor 1"),
			processInputFn: func(args ProcessInputArgs) ([]MastraDBMessage, *MessageList, *ProcessInputResultWithSystemMessages, error) {
				return nil, nil, nil, args.Abort("Test abort reason", nil)
			},
		}
		p2 := &testInputProcessor{
			BaseProcessor: NewBaseProcessor("processor2", "Processor 2"),
			processInputFn: func(args ProcessInputArgs) ([]MastraDBMessage, *MessageList, *ProcessInputResultWithSystemMessages, error) {
				return nil, nil, nil, nil
			},
		}

		runner := NewProcessorRunner(ProcessorRunnerConfig{
			InputProcessors:  []any{p1, p2},
			OutputProcessors: []any{},
			Logger:           newMockLogger(),
			AgentName:        "test-agent",
		})

		messageList := &MessageList{}
		_, err := runner.RunInputProcessors(messageList, nil, 0)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var tw *TripWire
		if !errors.As(err, &tw) {
			t.Fatalf("expected TripWire error, got %T: %v", err, err)
		}
		if !strings.Contains(tw.Message, "Test abort reason") {
			t.Errorf("expected message containing 'Test abort reason', got %q", tw.Message)
		}
	})

	t.Run("should abort with default message when no reason provided", func(t *testing.T) {
		p1 := &testInputProcessor{
			BaseProcessor: NewBaseProcessor("processor1", "Processor 1"),
			processInputFn: func(args ProcessInputArgs) ([]MastraDBMessage, *MessageList, *ProcessInputResultWithSystemMessages, error) {
				return nil, nil, nil, args.Abort("", nil)
			},
		}

		runner := NewProcessorRunner(ProcessorRunnerConfig{
			InputProcessors:  []any{p1},
			OutputProcessors: []any{},
			Logger:           newMockLogger(),
			AgentName:        "test-agent",
		})

		messageList := &MessageList{}
		_, err := runner.RunInputProcessors(messageList, nil, 0)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var tw *TripWire
		if !errors.As(err, &tw) {
			t.Fatalf("expected TripWire error, got %T: %v", err, err)
		}
		if !strings.Contains(tw.Message, "Tripwire triggered by processor1") {
			t.Errorf("expected default tripwire message, got %q", tw.Message)
		}
	})

	t.Run("should not execute subsequent processors after tripwire", func(t *testing.T) {
		executionOrder := []string{}
		p1 := &testInputProcessor{
			BaseProcessor: NewBaseProcessor("processor1", "Processor 1"),
			processInputFn: func(args ProcessInputArgs) ([]MastraDBMessage, *MessageList, *ProcessInputResultWithSystemMessages, error) {
				executionOrder = append(executionOrder, "processor1")
				return nil, nil, nil, args.Abort("Abort after processor1", nil)
			},
		}
		p2 := &testInputProcessor{
			BaseProcessor: NewBaseProcessor("processor2", "Processor 2"),
			processInputFn: func(args ProcessInputArgs) ([]MastraDBMessage, *MessageList, *ProcessInputResultWithSystemMessages, error) {
				executionOrder = append(executionOrder, "processor2")
				return nil, nil, nil, nil
			},
		}

		runner := NewProcessorRunner(ProcessorRunnerConfig{
			InputProcessors:  []any{p1, p2},
			OutputProcessors: []any{},
			Logger:           newMockLogger(),
			AgentName:        "test-agent",
		})

		messageList := &MessageList{}
		_, _ = runner.RunInputProcessors(messageList, nil, 0)
		if !reflect.DeepEqual(executionOrder, []string{"processor1"}) {
			t.Errorf("expected only processor1 to run, got %v", executionOrder)
		}
	})

	t.Run("should skip processors that do not implement processInput (InputProcessor interface)", func(t *testing.T) {
		executionOrder := []string{}
		p1 := &testInputProcessor{
			BaseProcessor: NewBaseProcessor("processor1", "Processor 1"),
			processInputFn: func(args ProcessInputArgs) ([]MastraDBMessage, *MessageList, *ProcessInputResultWithSystemMessages, error) {
				executionOrder = append(executionOrder, "processor1")
				return nil, nil, nil, nil
			},
		}

		// A processor that is NOT an InputProcessor (just a string in the []any slice)
		notAProcessor := "not-a-processor"

		p3 := &testInputProcessor{
			BaseProcessor: NewBaseProcessor("processor3", "Processor 3"),
			processInputFn: func(args ProcessInputArgs) ([]MastraDBMessage, *MessageList, *ProcessInputResultWithSystemMessages, error) {
				executionOrder = append(executionOrder, "processor3")
				return nil, nil, nil, nil
			},
		}

		runner := NewProcessorRunner(ProcessorRunnerConfig{
			InputProcessors:  []any{p1, notAProcessor, p3},
			OutputProcessors: []any{},
			Logger:           newMockLogger(),
			AgentName:        "test-agent",
		})

		messageList := &MessageList{}
		_, err := runner.RunInputProcessors(messageList, nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(executionOrder, []string{"processor1", "processor3"}) {
			t.Errorf("expected [processor1, processor3], got %v", executionOrder)
		}
	})

	t.Run("Issue #9969: should provide systemMessages parameter to processInput", func(t *testing.T) {
		t.Skip("not yet implemented: requires MessageList.addSystem and getAllSystemMessages which are not ported")
		// TS test adds system messages to the message list and verifies that
		// processInput receives them via args.SystemMessages.
	})

	t.Run("Issue #9969: should allow InputProcessor to modify system messages via return value", func(t *testing.T) {
		t.Skip("not yet implemented: requires MessageList.addSystem, full system message handling in runner")
	})

	t.Run("Issue #9969: should continue to allow adding new system messages via return array", func(t *testing.T) {
		t.Skip("not yet implemented: requires MessageList system message handling")
	})
}

// ---------------------------------------------------------------------------
// Tests: ProcessorRunner > Output Processors
// ---------------------------------------------------------------------------

func TestProcessorRunner_OutputProcessors(t *testing.T) {
	t.Run("should run output processors in order", func(t *testing.T) {
		t.Skip("not yet implemented: requires MessageList.get.all.prompt() which is not ported")
		// TS test creates two output processors that push messages to messages array,
		// then verifies the order.
	})

	t.Run("should abort if tripwire is triggered in output processor", func(t *testing.T) {
		p1 := &testOutputProcessor{
			BaseProcessor: NewBaseProcessor("processor1", "Processor 1"),
			processOutputResultFn: func(args ProcessOutputResultArgs) ([]MastraDBMessage, *MessageList, error) {
				return nil, nil, args.Abort("Output processor abort", nil)
			},
		}

		runner := NewProcessorRunner(ProcessorRunnerConfig{
			InputProcessors:  []any{},
			OutputProcessors: []any{p1},
			Logger:           newMockLogger(),
			AgentName:        "test-agent",
		})

		messageList := &MessageList{}
		_, err := runner.RunOutputProcessors(messageList, nil, 0, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var tw *TripWire
		if !errors.As(err, &tw) {
			t.Fatalf("expected TripWire error, got %T: %v", err, err)
		}
		if !strings.Contains(tw.Message, "Output processor abort") {
			t.Errorf("expected message 'Output processor abort', got %q", tw.Message)
		}
	})

	t.Run("should skip processors that do not implement processOutputResult (OutputProcessor interface)", func(t *testing.T) {
		t.Skip("not yet implemented: requires MessageList.get.all.prompt() which is not ported")
	})
}

// ---------------------------------------------------------------------------
// Tests: ProcessorRunner > Stream Processing
// ---------------------------------------------------------------------------

func TestProcessorRunner_StreamProcessing(t *testing.T) {
	t.Run("should process text chunks through output processors", func(t *testing.T) {
		p1 := &testOutputProcessor{
			BaseProcessor: NewBaseProcessor("processor1", "Processor 1"),
			processOutputStreamFn: func(args ProcessOutputStreamArgs) (*ChunkType, error) {
				if args.Part.Type == "text-delta" {
					if payload, ok := args.Part.Payload.(map[string]any); ok {
						if text, ok := payload["text"].(string); ok {
							return &ChunkType{
								Type:    "text-delta",
								Payload: map[string]any{"text": strings.ToUpper(text)},
							}, nil
						}
					}
				}
				return &args.Part, nil
			},
		}

		runner := NewProcessorRunner(ProcessorRunnerConfig{
			InputProcessors:  []any{},
			OutputProcessors: []any{p1},
			Logger:           newMockLogger(),
			AgentName:        "test-agent",
		})

		processorStates := &sync.Map{}
		result := runner.ProcessPart(
			ChunkType{Type: "text-delta", Payload: map[string]any{"text": "hello world", "id": "text-1"}},
			processorStates, nil, nil, 0, nil,
		)
		if result.Blocked {
			t.Error("expected not blocked")
		}
	})

	t.Run("should abort stream when processor calls abort", func(t *testing.T) {
		p1 := &testOutputProcessor{
			BaseProcessor: NewBaseProcessor("processor1", "Processor 1"),
			processOutputStreamFn: func(args ProcessOutputStreamArgs) (*ChunkType, error) {
				if args.Part.Type == "text-delta" {
					if payload, ok := args.Part.Payload.(map[string]any); ok {
						if text, ok := payload["text"].(string); ok {
							if strings.Contains(text, "blocked") {
								return nil, args.Abort("Content blocked", nil)
							}
						}
					}
				}
				return &args.Part, nil
			},
		}

		runner := NewProcessorRunner(ProcessorRunnerConfig{
			InputProcessors:  []any{},
			OutputProcessors: []any{p1},
			Logger:           newMockLogger(),
			AgentName:        "test-agent",
		})

		processorStates := &sync.Map{}
		result := runner.ProcessPart(
			ChunkType{Type: "text-delta", Payload: map[string]any{"text": "blocked content", "id": "text-1"}},
			processorStates, nil, nil, 0, nil,
		)
		if result.Part != nil {
			t.Error("expected nil part when aborted")
		}
		if !result.Blocked {
			t.Error("expected blocked=true")
		}
		if result.Reason != "Content blocked" {
			t.Errorf("expected reason 'Content blocked', got %q", result.Reason)
		}
	})

	t.Run("should handle processor errors gracefully", func(t *testing.T) {
		p1 := &testOutputProcessor{
			BaseProcessor: NewBaseProcessor("processor1", "Processor 1"),
			processOutputStreamFn: func(args ProcessOutputStreamArgs) (*ChunkType, error) {
				return nil, errors.New("Processor error")
			},
		}

		runner := NewProcessorRunner(ProcessorRunnerConfig{
			InputProcessors:  []any{},
			OutputProcessors: []any{p1},
			Logger:           newMockLogger(),
			AgentName:        "test-agent",
		})

		processorStates := &sync.Map{}
		result := runner.ProcessPart(
			ChunkType{Type: "text-delta", Payload: map[string]any{"text": "test content", "id": "text-1"}},
			processorStates, nil, nil, 0, nil,
		)
		// On error, the processor is skipped and the original part continues
		if result.Blocked {
			t.Error("expected not blocked on processor error")
		}
	})

	t.Run("should skip processors that do not implement processOutputStream", func(t *testing.T) {
		p1 := &testOutputProcessor{
			BaseProcessor: NewBaseProcessor("processor1", "Processor 1"),
			processOutputStreamFn: func(args ProcessOutputStreamArgs) (*ChunkType, error) {
				if args.Part.Type == "text-delta" {
					if payload, ok := args.Part.Payload.(map[string]any); ok {
						if text, ok := payload["text"].(string); ok {
							return &ChunkType{
								Type:    "text-delta",
								Payload: map[string]any{"text": strings.ToUpper(text)},
							}, nil
						}
					}
				}
				return &args.Part, nil
			},
		}

		// p2 is NOT an OutputProcessor (just a string)
		notAProcessor := "not-a-processor"

		p3 := &testOutputProcessor{
			BaseProcessor: NewBaseProcessor("processor3", "Processor 3"),
			processOutputStreamFn: func(args ProcessOutputStreamArgs) (*ChunkType, error) {
				if args.Part.Type == "text-delta" {
					if payload, ok := args.Part.Payload.(map[string]any); ok {
						if text, ok := payload["text"].(string); ok {
							return &ChunkType{
								Type:    "text-delta",
								Payload: map[string]any{"text": text + "!"},
							}, nil
						}
					}
				}
				return &args.Part, nil
			},
		}

		runner := NewProcessorRunner(ProcessorRunnerConfig{
			InputProcessors:  []any{},
			OutputProcessors: []any{p1, notAProcessor, p3},
			Logger:           newMockLogger(),
			AgentName:        "test-agent",
		})

		processorStates := &sync.Map{}
		result := runner.ProcessPart(
			ChunkType{Type: "text-delta", Payload: map[string]any{"text": "hello", "id": "text-1"}},
			processorStates, nil, nil, 0, nil,
		)
		if result.Part == nil {
			t.Fatal("expected non-nil part")
		}
		if payload, ok := result.Part.Payload.(map[string]any); ok {
			if text, ok := payload["text"].(string); ok {
				if text != "HELLO!" {
					t.Errorf("expected 'HELLO!', got %q", text)
				}
			} else {
				t.Error("expected text payload")
			}
		} else {
			t.Error("expected map payload")
		}
		if result.Blocked {
			t.Error("expected not blocked")
		}
	})

	t.Run("should return original text when no output processors are configured", func(t *testing.T) {
		runner := NewProcessorRunner(ProcessorRunnerConfig{
			InputProcessors:  []any{},
			OutputProcessors: []any{},
			Logger:           newMockLogger(),
			AgentName:        "test-agent",
		})

		processorStates := &sync.Map{}
		result := runner.ProcessPart(
			ChunkType{Type: "text-delta", Payload: map[string]any{"text": "original text", "id": "text-1"}},
			processorStates, nil, nil, 0, nil,
		)
		if result.Part == nil {
			t.Fatal("expected non-nil part")
		}
		if payload, ok := result.Part.Payload.(map[string]any); ok {
			if text, ok := payload["text"].(string); ok {
				if text != "original text" {
					t.Errorf("expected 'original text', got %q", text)
				}
			}
		}
		if result.Blocked {
			t.Error("expected not blocked")
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: ProcessorRunner > Stateful Stream Processing
// ---------------------------------------------------------------------------

func TestProcessorRunner_StatefulStreamProcessing(t *testing.T) {
	t.Run("should process chunks with state management", func(t *testing.T) {
		p := &testOutputProcessor{
			BaseProcessor: NewBaseProcessor("statefulProcessor", "Stateful Processor"),
			processOutputStreamFn: func(args ProcessOutputStreamArgs) (*ChunkType, error) {
				// Only emit when we see a period
				shouldEmit := args.Part.Type == "text-delta"
				if shouldEmit {
					if payload, ok := args.Part.Payload.(map[string]any); ok {
						if text, ok := payload["text"].(string); ok {
							shouldEmit = strings.Contains(text, ".")
						} else {
							shouldEmit = false
						}
					} else {
						shouldEmit = false
					}
				}
				if shouldEmit {
					// Accumulate text from all prior parts
					var accText string
					for _, sp := range args.StreamParts {
						if sp.Type == "text-delta" {
							if p, ok := sp.Payload.(map[string]any); ok {
								if t, ok := p["text"].(string); ok {
									accText += t
								}
							}
						}
					}
					return &ChunkType{
						Type:    "text-delta",
						Payload: map[string]any{"text": accText},
					}, nil
				}
				return nil, nil
			},
		}

		runner := NewProcessorRunner(ProcessorRunnerConfig{
			InputProcessors:  []any{},
			OutputProcessors: []any{p},
			Logger:           newMockLogger(),
			AgentName:        "test-agent",
		})

		processorStates := &sync.Map{}

		result1 := runner.ProcessPart(
			ChunkType{Type: "text-delta", Payload: map[string]any{"text": "Hello world", "id": "text-1"}},
			processorStates, nil, nil, 0, nil,
		)
		if result1.Part != nil {
			t.Error("expected nil part (no period yet)")
		}

		result2 := runner.ProcessPart(
			ChunkType{Type: "text-delta", Payload: map[string]any{"text": ".", "id": "text-2"}},
			processorStates, nil, nil, 0, nil,
		)
		if result2.Part == nil {
			t.Fatal("expected non-nil part on period")
		}
		if payload, ok := result2.Part.Payload.(map[string]any); ok {
			if text, ok := payload["text"].(string); ok {
				if text != "Hello world." {
					t.Errorf("expected 'Hello world.', got %q", text)
				}
			}
		}
	})

	t.Run("should accumulate chunks for moderation decisions", func(t *testing.T) {
		p := &testOutputProcessor{
			BaseProcessor: NewBaseProcessor("moderationProcessor", "Moderation Processor"),
			processOutputStreamFn: func(args ProcessOutputStreamArgs) (*ChunkType, error) {
				// Check accumulated text for violence
				var accText string
				for _, sp := range args.StreamParts {
					if sp.Type == "text-delta" {
						if p, ok := sp.Payload.(map[string]any); ok {
							if t, ok := p["text"].(string); ok {
								accText += t
							}
						}
					}
				}
				if strings.Contains(accText, "punch") && strings.Contains(accText, "face") {
					return nil, args.Abort("Violent content detected", nil)
				}
				return &args.Part, nil
			},
		}

		runner := NewProcessorRunner(ProcessorRunnerConfig{
			InputProcessors:  []any{},
			OutputProcessors: []any{p},
			Logger:           newMockLogger(),
			AgentName:        "test-agent",
		})

		processorStates := &sync.Map{}

		result1 := runner.ProcessPart(
			ChunkType{Type: "text-delta", Payload: map[string]any{"text": "i want to ", "id": "text-1"}},
			processorStates, nil, nil, 0, nil,
		)
		if result1.Part == nil {
			t.Error("expected part for harmless text")
		}

		result2 := runner.ProcessPart(
			ChunkType{Type: "text-delta", Payload: map[string]any{"text": "punch", "id": "text-2"}},
			processorStates, nil, nil, 0, nil,
		)
		if result2.Part == nil {
			t.Error("expected part for partial trigger")
		}

		result3 := runner.ProcessPart(
			ChunkType{Type: "text-delta", Payload: map[string]any{"text": " you in the face", "id": "text-3"}},
			processorStates, nil, nil, 0, nil,
		)
		if result3.Part != nil {
			t.Error("expected nil part when violence detected")
		}
		if !result3.Blocked {
			t.Error("expected blocked=true")
		}
		if result3.Reason != "Violent content detected" {
			t.Errorf("expected reason 'Violent content detected', got %q", result3.Reason)
		}
	})

	t.Run("should handle custom state management", func(t *testing.T) {
		p := &testOutputProcessor{
			BaseProcessor: NewBaseProcessor("customStateProcessor", "Custom State Processor"),
			processOutputStreamFn: func(args ProcessOutputStreamArgs) (*ChunkType, error) {
				wordCount := 0
				if v, ok := args.State["wordCount"]; ok {
					wordCount = v.(int)
				}
				if args.Part.Type == "text-delta" {
					if payload, ok := args.Part.Payload.(map[string]any); ok {
						if text, ok := payload["text"].(string); ok {
							words := strings.Fields(text)
							wordCount += len(words)
							args.State["wordCount"] = wordCount
						}
					}
				}

				shouldEmit := wordCount%3 == 0
				if shouldEmit {
					if args.Part.Type == "text-delta" {
						if payload, ok := args.Part.Payload.(map[string]any); ok {
							if text, ok := payload["text"].(string); ok {
								return &ChunkType{
									Type:    "text-delta",
									Payload: map[string]any{"text": strings.ToUpper(text)},
								}, nil
							}
						}
					}
					return &ChunkType{
						Type:    "text-delta",
						Payload: map[string]any{"text": ""},
					}, nil
				}
				return nil, nil
			},
		}

		runner := NewProcessorRunner(ProcessorRunnerConfig{
			InputProcessors:  []any{},
			OutputProcessors: []any{p},
			Logger:           newMockLogger(),
			AgentName:        "test-agent",
		})

		processorStates := &sync.Map{}

		result1 := runner.ProcessPart(
			ChunkType{Type: "text-delta", Payload: map[string]any{"text": "hello world", "id": "text-1"}},
			processorStates, nil, nil, 0, nil,
		)
		if result1.Part != nil {
			t.Error("expected nil part (2 words, not divisible by 3)")
		}

		result2 := runner.ProcessPart(
			ChunkType{Type: "text-delta", Payload: map[string]any{"text": " goodbye", "id": "text-2"}},
			processorStates, nil, nil, 0, nil,
		)
		if result2.Part == nil {
			t.Fatal("expected non-nil part (3 words, divisible by 3)")
		}
		if payload, ok := result2.Part.Payload.(map[string]any); ok {
			if text, ok := payload["text"].(string); ok {
				if text != " GOODBYE" {
					t.Errorf("expected ' GOODBYE', got %q", text)
				}
			}
		}
	})

	t.Run("should handle stream end detection", func(t *testing.T) {
		p := &testOutputProcessor{
			BaseProcessor: NewBaseProcessor("streamEndProcessor", "Stream End Processor"),
			processOutputStreamFn: func(args ProcessOutputStreamArgs) (*ChunkType, error) {
				if args.Part.Type == "text-delta" {
					if payload, ok := args.Part.Payload.(map[string]any); ok {
						if text, ok := payload["text"].(string); ok {
							if text == "" {
								// Emit accumulated text at stream end
								var accText string
								for _, sp := range args.StreamParts {
									if sp.Type == "text-delta" {
										if p, ok := sp.Payload.(map[string]any); ok {
											if t, ok := p["text"].(string); ok {
												accText += t
											}
										}
									}
								}
								return &ChunkType{
									Type:    "text-delta",
									Payload: map[string]any{"text": strings.ToUpper(accText)},
								}, nil
							}
						}
					}
				}
				return nil, nil
			},
		}

		runner := NewProcessorRunner(ProcessorRunnerConfig{
			InputProcessors:  []any{},
			OutputProcessors: []any{p},
			Logger:           newMockLogger(),
			AgentName:        "test-agent",
		})

		processorStates := &sync.Map{}

		runner.ProcessPart(
			ChunkType{Type: "text-delta", Payload: map[string]any{"text": "hello", "id": "text-1"}},
			processorStates, nil, nil, 0, nil,
		)
		runner.ProcessPart(
			ChunkType{Type: "text-delta", Payload: map[string]any{"text": " world", "id": "text-2"}},
			processorStates, nil, nil, 0, nil,
		)

		result := runner.ProcessPart(
			ChunkType{Type: "text-delta", Payload: map[string]any{"text": "", "id": "text-3"}},
			processorStates, nil, nil, 0, nil,
		)
		if result.Part == nil {
			t.Fatal("expected non-nil part on stream end")
		}
		if payload, ok := result.Part.Payload.(map[string]any); ok {
			if text, ok := payload["text"].(string); ok {
				if text != "HELLO WORLD" {
					t.Errorf("expected 'HELLO WORLD', got %q", text)
				}
			}
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: ProcessorRunner > Stream Processing Integration
// ---------------------------------------------------------------------------

func TestProcessorRunner_StreamProcessingIntegration(t *testing.T) {
	t.Run("should create a readable stream that processes text chunks", func(t *testing.T) {
		t.Skip("not yet implemented: requires runOutputProcessorsForStream which is not ported to Go")
	})

	t.Run("should emit tripwire when processor aborts stream", func(t *testing.T) {
		t.Skip("not yet implemented: requires runOutputProcessorsForStream which is not ported to Go")
	})
}
