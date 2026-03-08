// Ported from: packages/core/src/loop/loop.test.ts
package loop

import (
	"testing"
)

// The TS loop.test.ts imports test suites from test-utils/ (textStreamTests,
// fullStreamTests, resultObjectTests, optionsTests, generateTextTestsV5,
// toolsTests, streamObjectTests) which each define many sub-tests exercising
// the loop function with mock language models and streams.
//
// These test suites rely on extensive AI SDK mock infrastructure that is not
// yet ported to Go (see testutils/stubs.go). Each top-level describe/it block
// is preserved here as t.Run with t.Skip until the test utilities and
// dependent packages (agent, stream, llm) are ported.

func TestLoop(t *testing.T) {
	t.Run("AISDK v5", func(t *testing.T) {
		t.Run("textStreamTests", func(t *testing.T) {
			t.Skip("not yet implemented: test-utils/textStream not ported")
		})

		t.Run("fullStreamTests v2", func(t *testing.T) {
			t.Skip("not yet implemented: test-utils/fullStream not ported")
		})

		t.Run("resultObjectTests v2", func(t *testing.T) {
			t.Skip("not yet implemented: test-utils/resultObject not ported")
		})

		t.Run("optionsTests", func(t *testing.T) {
			t.Skip("not yet implemented: test-utils/options not ported")
		})

		t.Run("generateTextTestsV5", func(t *testing.T) {
			t.Skip("not yet implemented: test-utils/generateText not ported")
		})

		t.Run("toolsTests", func(t *testing.T) {
			t.Skip("not yet implemented: test-utils/tools not ported")
		})

		t.Run("streamObjectTests", func(t *testing.T) {
			t.Skip("not yet implemented: test-utils/streamObject not ported")
		})
	})

	t.Run("AISDK v6 V3 models", func(t *testing.T) {
		t.Run("fullStreamTests v3", func(t *testing.T) {
			t.Skip("not yet implemented: test-utils/fullStream not ported")
		})

		t.Run("resultObjectTests v3", func(t *testing.T) {
			t.Skip("not yet implemented: test-utils/resultObject not ported")
		})
	})
}

// TestLoopValidatesModels tests that Loop returns an error when no models
// are provided. This is one behaviour we can verify without the full
// test-utils infrastructure.
func TestLoopValidatesModels(t *testing.T) {
	_, err := Loop(LoopOptions{
		// No models provided
	})
	if err == nil {
		t.Fatal("expected error when models slice is empty, got nil")
	}

	mastraErr, ok := err.(*MastraError)
	if !ok {
		t.Fatalf("expected *MastraError, got %T", err)
	}
	if mastraErr.ID != "LOOP_MODELS_EMPTY" {
		t.Errorf("expected error ID LOOP_MODELS_EMPTY, got %s", mastraErr.ID)
	}
}

// TestLoopReturnsOutputWithModels verifies that Loop returns a non-nil
// DestructurableOutput when at least one model config is provided.
func TestLoopReturnsOutputWithModels(t *testing.T) {
	result, err := Loop(LoopOptions{
		Models: []ModelManagerModelConfig{
			{},
		},
		AgentID: "test-agent",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ModelOutput == nil {
		t.Fatal("expected non-nil ModelOutput")
	}
}

// TestLoopUsesCustomRunID verifies that a provided RunID is used.
func TestLoopUsesCustomRunID(t *testing.T) {
	// The Loop function doesn't expose runID directly on the return value,
	// but we can verify it doesn't error with a custom runID.
	result, err := Loop(LoopOptions{
		Models: []ModelManagerModelConfig{
			{},
		},
		RunID:   "custom-run-id",
		AgentID: "test-agent",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// TestLoopUsesIDGenerator verifies that a custom IDGenerator is called
// when no RunID is provided.
func TestLoopUsesIDGenerator(t *testing.T) {
	called := false
	result, err := Loop(LoopOptions{
		Models: []ModelManagerModelConfig{
			{},
		},
		AgentID: "test-agent",
		IDGenerator: func(ctx *IdGeneratorContext) string {
			called = true
			if ctx.IdType != "run" {
				t.Errorf("expected IdType 'run', got %s", ctx.IdType)
			}
			if ctx.Source == nil || *ctx.Source != "agent" {
				t.Errorf("expected Source 'agent', got %v", ctx.Source)
			}
			if ctx.EntityId == nil || *ctx.EntityId != "test-agent" {
				t.Errorf("expected EntityId 'test-agent', got %v", ctx.EntityId)
			}
			return "generated-run-id"
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !called {
		t.Error("expected IDGenerator to be called")
	}
}
