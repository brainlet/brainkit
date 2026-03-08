// Ported from: packages/core/src/relevance/mastra-agent/index.test.ts
package mastraagent

import (
	"fmt"
	"math"
	"testing"

	"github.com/brainlet/brainkit/agent-kit/core/relevance"
)

// mockAgent implements the Agent interface for testing.
type mockAgent struct {
	model          any
	modelErr       error
	generateResult *GenerateResult
	generateErr    error
	legacyResult   *GenerateResult
	legacyErr      error
}

func (m *mockAgent) GetModel() (any, error) {
	return m.model, m.modelErr
}

func (m *mockAgent) Generate(prompt string) (*GenerateResult, error) {
	return m.generateResult, m.generateErr
}

func (m *mockAgent) GenerateLegacy(prompt string) (*GenerateResult, error) {
	return m.legacyResult, m.legacyErr
}

func TestNewMastraAgentRelevanceScorer(t *testing.T) {
	// Save and restore the global NewAgent constructor.
	origNewAgent := NewAgent
	defer func() { NewAgent = origNewAgent }()

	t.Run("creates scorer with agent when NewAgent is set", func(t *testing.T) {
		var capturedCfg AgentConfig
		NewAgent = func(cfg AgentConfig) Agent {
			capturedCfg = cfg
			return &mockAgent{model: "test-model"}
		}

		scorer := NewMastraAgentRelevanceScorer("test", "gpt-4")

		if scorer == nil {
			t.Fatal("scorer should not be nil")
		}
		if capturedCfg.ID != "relevance-scorer-test" {
			t.Errorf("agent ID = %q, want %q", capturedCfg.ID, "relevance-scorer-test")
		}
		if capturedCfg.Name != "Relevance Scorer test" {
			t.Errorf("agent Name = %q, want %q", capturedCfg.Name, "Relevance Scorer test")
		}
		if capturedCfg.Instructions == "" {
			t.Error("agent Instructions should not be empty")
		}
	})

	t.Run("creates scorer with nil agent when NewAgent is nil", func(t *testing.T) {
		NewAgent = nil
		scorer := NewMastraAgentRelevanceScorer("test", "model")
		if scorer == nil {
			t.Fatal("scorer should not be nil")
		}
		// Agent is nil, so GetRelevanceScore should return error.
		_, err := scorer.GetRelevanceScore("q", "t")
		if err == nil {
			t.Fatal("expected error when agent is nil")
		}
	})
}

func TestMastraAgentRelevanceScorerGetRelevanceScore(t *testing.T) {
	origNewAgent := NewAgent
	origIsSupportedLM := IsSupportedLanguageModel
	defer func() {
		NewAgent = origNewAgent
		IsSupportedLanguageModel = origIsSupportedLM
	}()

	t.Run("returns parsed score from agent generate", func(t *testing.T) {
		agent := &mockAgent{
			model:          "supported-model",
			generateResult: &GenerateResult{Text: "0.85"},
		}
		NewAgent = func(_ AgentConfig) Agent { return agent }
		IsSupportedLanguageModel = func(_ any) bool { return true }

		scorer := NewMastraAgentRelevanceScorer("test", "model")
		score, err := scorer.GetRelevanceScore("what is Go?", "Go is a language.")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if math.Abs(score-0.85) > 0.001 {
			t.Errorf("score = %f, want 0.85", score)
		}
	})

	t.Run("uses legacy generate for unsupported models", func(t *testing.T) {
		agent := &mockAgent{
			model:        "unsupported-model",
			legacyResult: &GenerateResult{Text: "0.72"},
		}
		NewAgent = func(_ AgentConfig) Agent { return agent }
		IsSupportedLanguageModel = func(_ any) bool { return false }

		scorer := NewMastraAgentRelevanceScorer("test", "model")
		score, err := scorer.GetRelevanceScore("query", "text")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if math.Abs(score-0.72) > 0.001 {
			t.Errorf("score = %f, want 0.72", score)
		}
	})

	t.Run("returns error for non-numeric response", func(t *testing.T) {
		agent := &mockAgent{
			model:          "model",
			generateResult: &GenerateResult{Text: "not a number"},
		}
		NewAgent = func(_ AgentConfig) Agent { return agent }
		IsSupportedLanguageModel = func(_ any) bool { return true }

		scorer := NewMastraAgentRelevanceScorer("test", "model")
		_, err := scorer.GetRelevanceScore("q", "t")
		if err == nil {
			t.Fatal("expected error for non-numeric response")
		}
	})

	t.Run("returns error when GetModel fails", func(t *testing.T) {
		agent := &mockAgent{
			modelErr: fmt.Errorf("model unavailable"),
		}
		NewAgent = func(_ AgentConfig) Agent { return agent }

		scorer := NewMastraAgentRelevanceScorer("test", "model")
		_, err := scorer.GetRelevanceScore("q", "t")
		if err == nil {
			t.Fatal("expected error when GetModel fails")
		}
	})

	t.Run("returns error when Generate fails", func(t *testing.T) {
		agent := &mockAgent{
			model:       "model",
			generateErr: fmt.Errorf("generation failed"),
		}
		NewAgent = func(_ AgentConfig) Agent { return agent }
		IsSupportedLanguageModel = func(_ any) bool { return true }

		scorer := NewMastraAgentRelevanceScorer("test", "model")
		_, err := scorer.GetRelevanceScore("q", "t")
		if err == nil {
			t.Fatal("expected error when Generate fails")
		}
	})

	t.Run("returns error when agent is nil", func(t *testing.T) {
		NewAgent = nil
		scorer := NewMastraAgentRelevanceScorer("test", "model")
		_, err := scorer.GetRelevanceScore("q", "t")
		if err == nil {
			t.Fatal("expected error when agent is nil")
		}
	})
}

func TestMastraAgentRelevanceScorerImplementsInterface(t *testing.T) {
	t.Run("satisfies RelevanceScoreProvider interface", func(t *testing.T) {
		origNewAgent := NewAgent
		defer func() { NewAgent = origNewAgent }()

		NewAgent = func(_ AgentConfig) Agent {
			return &mockAgent{model: "m", generateResult: &GenerateResult{Text: "0.5"}}
		}

		scorer := NewMastraAgentRelevanceScorer("test", "model")
		var _ relevance.RelevanceScoreProvider = scorer
	})
}
