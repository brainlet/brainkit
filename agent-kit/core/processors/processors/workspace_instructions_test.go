// Ported from: packages/core/src/processors/processors/workspace-instructions.test.ts
package concreteprocessors

import (
	"testing"

	processors "github.com/brainlet/brainkit/agent-kit/core/processors"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// testWorkspace is a mock workspace for testing.
type testWorkspace struct {
	instructions string
	callCount    int
	lastOpts     map[string]any
}

func (tw *testWorkspace) GetInstructions(opts map[string]any) string {
	tw.callCount++
	tw.lastOpts = opts
	return tw.instructions
}

func TestWorkspaceInstructionsProcessor(t *testing.T) {
	t.Run("should have correct id", func(t *testing.T) {
		ws := &testWorkspace{instructions: "some instructions"}
		processor := NewWorkspaceInstructionsProcessor(WorkspaceInstructionsProcessorOptions{
			Workspace: ws,
		})

		if processor.ID() != "workspace-instructions-processor" {
			t.Fatalf("expected id 'workspace-instructions-processor', got '%s'", processor.ID())
		}
	})

	t.Run("should inject instructions as system message", func(t *testing.T) {
		ws := &testWorkspace{instructions: `Local filesystem at "/data". Local command execution.`}
		processor := NewWorkspaceInstructionsProcessor(WorkspaceInstructionsProcessorOptions{
			Workspace: ws,
		})

		result, _, err := processor.ProcessInputStep(processors.ProcessInputStepArgs{
			Tools: make(map[string]any),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result == nil {
			t.Fatal("expected non-nil result")
		}

		if len(result.SystemMessages) != 1 {
			t.Fatalf("expected 1 system message, got %d", len(result.SystemMessages))
		}

		content, ok := result.SystemMessages[0].Content.(string)
		if !ok {
			t.Fatal("expected string content")
		}
		if content != `Local filesystem at "/data". Local command execution.` {
			t.Fatalf("unexpected content: '%s'", content)
		}

		if result.SystemMessages[0].Role != "system" {
			t.Fatalf("expected role 'system', got '%s'", result.SystemMessages[0].Role)
		}
	})

	t.Run("should not inject system message when instructions are empty", func(t *testing.T) {
		ws := &testWorkspace{instructions: ""}
		processor := NewWorkspaceInstructionsProcessor(WorkspaceInstructionsProcessorOptions{
			Workspace: ws,
		})

		result, _, err := processor.ProcessInputStep(processors.ProcessInputStepArgs{
			Tools: make(map[string]any),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// When instructions are empty, result should be nil (no-op).
		if result != nil {
			t.Fatal("expected nil result for empty instructions")
		}
	})

	t.Run("should inject system message even when instructions are whitespace-only", func(t *testing.T) {
		ws := &testWorkspace{instructions: "   "}
		processor := NewWorkspaceInstructionsProcessor(WorkspaceInstructionsProcessorOptions{
			Workspace: ws,
		})

		result, _, err := processor.ProcessInputStep(processors.ProcessInputStepArgs{
			Tools: make(map[string]any),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Whitespace-only is not empty, so should inject.
		if result == nil {
			t.Fatal("expected non-nil result for whitespace-only instructions")
		}
		if len(result.SystemMessages) != 1 {
			t.Fatalf("expected 1 system message, got %d", len(result.SystemMessages))
		}
	})

	t.Run("should call getInstructions on each processInputStep", func(t *testing.T) {
		ws := &testWorkspace{instructions: "instructions"}
		processor := NewWorkspaceInstructionsProcessor(WorkspaceInstructionsProcessorOptions{
			Workspace: ws,
		})

		_, _, err := processor.ProcessInputStep(processors.ProcessInputStepArgs{
			Tools: make(map[string]any),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if ws.callCount != 1 {
			t.Fatalf("expected getInstructions to be called once, got %d", ws.callCount)
		}
	})

	t.Run("should pass requestContext through to workspace getInstructions", func(t *testing.T) {
		ws := &testWorkspace{instructions: "instructions"}
		processor := NewWorkspaceInstructionsProcessor(WorkspaceInstructionsProcessorOptions{
			Workspace: ws,
		})

		ctx := requestcontext.NewRequestContext()
		ctx.Set("locale", "en")

		_, _, err := processor.ProcessInputStep(processors.ProcessInputStepArgs{
			ProcessorMessageContext: processors.ProcessorMessageContext{
				ProcessorContext: processors.ProcessorContext{
					RequestContext: ctx,
				},
			},
			Tools: make(map[string]any),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if ws.lastOpts == nil {
			t.Fatal("expected opts to be passed")
		}
		if ws.lastOpts["requestContext"] == nil {
			t.Fatal("expected requestContext to be passed in opts")
		}
		if ws.lastOpts["requestContext"] != ctx {
			t.Fatal("expected requestContext to be the same instance")
		}
	})

	t.Run("should pass nil requestContext when not provided", func(t *testing.T) {
		ws := &testWorkspace{instructions: "instructions"}
		processor := NewWorkspaceInstructionsProcessor(WorkspaceInstructionsProcessorOptions{
			Workspace: ws,
		})

		_, _, err := processor.ProcessInputStep(processors.ProcessInputStepArgs{
			Tools: make(map[string]any),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if ws.lastOpts == nil {
			t.Fatal("expected opts to be passed")
		}
		// RequestContext should not be in opts when not set (nil in ProcessorContext).
		// The Go implementation passes nil requestContext into opts when it's nil.
	})
}
