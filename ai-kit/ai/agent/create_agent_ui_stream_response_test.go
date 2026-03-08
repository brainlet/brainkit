// Ported from: packages/ai/src/agent/create-agent-ui-stream-response.test.ts
package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"

	gt "github.com/brainlet/brainkit/ai-kit/ai/generatetext"
)

// mockAgent implements the Agent interface for testing.
type mockAgent struct {
	id         string
	tools      gt.ToolSet
	generateFn func(params AgentCallParameters) (*gt.GenerateTextResult, error)
	streamFn   func(params AgentStreamParameters) (*gt.StreamTextResult, error)
}

func (m *mockAgent) Version() string { return "agent-v1" }
func (m *mockAgent) ID() string      { return m.id }
func (m *mockAgent) Tools() gt.ToolSet { return m.tools }

func (m *mockAgent) Generate(params AgentCallParameters) (*gt.GenerateTextResult, error) {
	if m.generateFn != nil {
		return m.generateFn(params)
	}
	return &gt.GenerateTextResult{Text: "mock response"}, nil
}

func (m *mockAgent) Stream(params AgentStreamParameters) (*gt.StreamTextResult, error) {
	if m.streamFn != nil {
		return m.streamFn(params)
	}
	return &gt.StreamTextResult{Text: "mock stream response"}, nil
}

func TestCreateAgentUIStreamResponse(t *testing.T) {
	t.Run("requires agent", func(t *testing.T) {
		w := httptest.NewRecorder()
		err := CreateAgentUIStreamResponse(w, CreateAgentUIStreamResponseOptions{
			Agent: nil,
		})
		if err == nil {
			t.Error("expected error when agent is nil")
		}
	})

	t.Run("calls agent stream with messages", func(t *testing.T) {
		var streamCalled bool
		agent := &mockAgent{
			tools: gt.ToolSet{
				"example": gt.Tool{
					Type: "function",
				},
			},
			streamFn: func(params AgentStreamParameters) (*gt.StreamTextResult, error) {
				streamCalled = true
				return &gt.StreamTextResult{Text: "Hello, world!"}, nil
			},
		}

		w := httptest.NewRecorder()
		err := CreateAgentUIStreamResponse(w, CreateAgentUIStreamResponseOptions{
			Agent: agent,
			UIMessages: []interface{}{
				map[string]interface{}{
					"role": "user",
					"id":   "msg-1",
					"parts": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": "Hello, world!",
						},
					},
				},
			},
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !streamCalled {
			t.Error("agent.Stream was not called")
		}
	})

	t.Run("sets custom headers", func(t *testing.T) {
		agent := &mockAgent{
			streamFn: func(params AgentStreamParameters) (*gt.StreamTextResult, error) {
				return &gt.StreamTextResult{Text: "test"}, nil
			},
		}

		w := httptest.NewRecorder()
		err := CreateAgentUIStreamResponse(w, CreateAgentUIStreamResponseOptions{
			Agent:      agent,
			UIMessages: []interface{}{},
			Headers: map[string]string{
				"X-Custom": "test-value",
			},
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := w.Header().Get("X-Custom"); got != "test-value" {
			t.Errorf("X-Custom header = %q, want %q", got, "test-value")
		}
	})

	t.Run("sets custom status code", func(t *testing.T) {
		agent := &mockAgent{
			streamFn: func(params AgentStreamParameters) (*gt.StreamTextResult, error) {
				return &gt.StreamTextResult{Text: "test"}, nil
			},
		}

		w := httptest.NewRecorder()
		err := CreateAgentUIStreamResponse(w, CreateAgentUIStreamResponseOptions{
			Agent:      agent,
			UIMessages: []interface{}{},
			Status:     http.StatusCreated,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := w.Code; got != http.StatusCreated {
			t.Errorf("status code = %d, want %d", got, http.StatusCreated)
		}
	})

	t.Run("defaults to 200 OK", func(t *testing.T) {
		agent := &mockAgent{
			streamFn: func(params AgentStreamParameters) (*gt.StreamTextResult, error) {
				return &gt.StreamTextResult{Text: "test"}, nil
			},
		}

		w := httptest.NewRecorder()
		err := CreateAgentUIStreamResponse(w, CreateAgentUIStreamResponseOptions{
			Agent:      agent,
			UIMessages: []interface{}{},
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := w.Code; got != http.StatusOK {
			t.Errorf("status code = %d, want %d", got, http.StatusOK)
		}
	})

	t.Run("passes onStepFinish through", func(t *testing.T) {
		var receivedOnStepFinish bool
		agent := &mockAgent{
			streamFn: func(params AgentStreamParameters) (*gt.StreamTextResult, error) {
				if params.OnStepFinish != nil {
					receivedOnStepFinish = true
				}
				return &gt.StreamTextResult{Text: "test"}, nil
			},
		}

		w := httptest.NewRecorder()
		err := CreateAgentUIStreamResponse(w, CreateAgentUIStreamResponseOptions{
			Agent:      agent,
			UIMessages: []interface{}{},
			OnStepFinish: func(event gt.OnStepFinishEvent) {
				// no-op callback for test
			},
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !receivedOnStepFinish {
			t.Error("onStepFinish was not passed to the agent")
		}
	})
}

func TestCreateAgentUIStream(t *testing.T) {
	t.Run("requires agent", func(t *testing.T) {
		_, err := CreateAgentUIStream(CreateAgentUIStreamOptions{
			Agent: nil,
		})
		if err == nil {
			t.Error("expected error when agent is nil")
		}
	})

	t.Run("calls agent stream", func(t *testing.T) {
		var streamCalled bool
		agent := &mockAgent{
			streamFn: func(params AgentStreamParameters) (*gt.StreamTextResult, error) {
				streamCalled = true
				return &gt.StreamTextResult{Text: "streamed"}, nil
			},
		}

		result, err := CreateAgentUIStream(CreateAgentUIStreamOptions{
			Agent:      agent,
			UIMessages: []interface{}{},
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !streamCalled {
			t.Error("agent.Stream was not called")
		}

		if result.Text != "streamed" {
			t.Errorf("result.Text = %q, want %q", result.Text, "streamed")
		}
	})
}

func TestPipeAgentUIStreamToResponse(t *testing.T) {
	t.Run("requires agent", func(t *testing.T) {
		w := httptest.NewRecorder()
		err := PipeAgentUIStreamToResponse(PipeAgentUIStreamToResponseOptions{
			Response: w,
			Agent:    nil,
		})
		if err == nil {
			t.Error("expected error when agent is nil")
		}
	})

	t.Run("requires response", func(t *testing.T) {
		agent := &mockAgent{
			streamFn: func(params AgentStreamParameters) (*gt.StreamTextResult, error) {
				return &gt.StreamTextResult{Text: "test"}, nil
			},
		}
		err := PipeAgentUIStreamToResponse(PipeAgentUIStreamToResponseOptions{
			Response: nil,
			Agent:    agent,
		})
		if err == nil {
			t.Error("expected error when response is nil")
		}
	})

	t.Run("pipes stream to response", func(t *testing.T) {
		var streamCalled bool
		agent := &mockAgent{
			streamFn: func(params AgentStreamParameters) (*gt.StreamTextResult, error) {
				streamCalled = true
				return &gt.StreamTextResult{Text: "piped"}, nil
			},
		}

		w := httptest.NewRecorder()
		err := PipeAgentUIStreamToResponse(PipeAgentUIStreamToResponseOptions{
			Response:   w,
			Agent:      agent,
			UIMessages: []interface{}{},
			Headers: map[string]string{
				"Content-Type": "text/event-stream",
			},
			Status: http.StatusOK,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !streamCalled {
			t.Error("agent.Stream was not called")
		}

		if got := w.Header().Get("Content-Type"); got != "text/event-stream" {
			t.Errorf("Content-Type = %q, want %q", got, "text/event-stream")
		}
	})
}
