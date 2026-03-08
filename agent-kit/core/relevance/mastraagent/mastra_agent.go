// Ported from: packages/core/src/relevance/mastra-agent/index.ts
package mastraagent

import (
	"fmt"
	"strconv"

	"github.com/brainlet/brainkit/agent-kit/core/relevance"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// Agent is a stub for ../../agent.Agent.
// TODO: import from agent package once ported.
type Agent interface {
	// GetModel returns the underlying model.
	GetModel() (any, error)
	// Generate performs a single-shot text generation and returns the result.
	Generate(prompt string) (*GenerateResult, error)
	// GenerateLegacy performs a legacy text generation call.
	GenerateLegacy(prompt string) (*GenerateResult, error)
}

// GenerateResult is a stub representing the result of an agent generate call.
// TODO: replace with actual type from agent package once ported.
type GenerateResult struct {
	Text string
}

// MastraModelConfig is a stub for ../../llm/model/shared.types.MastraModelConfig.
// TODO: import from llm/model package once ported.
type MastraModelConfig = any

// IsSupportedLanguageModel is a stub for ../../agent.isSupportedLanguageModel.
// TODO: import from agent package once ported.
// For now, always returns true. Defined as a variable to allow test overrides.
var IsSupportedLanguageModel = func(_ any) bool {
	return true
}

// AgentConfig holds configuration for creating a new agent.
// TODO: replace with actual Agent constructor options once agent package is ported.
type AgentConfig struct {
	ID           string
	Name         string
	Instructions string
	Model        MastraModelConfig
}

// NewAgent is a stub constructor for creating an Agent.
// TODO: replace with actual agent.New() once ported.
var NewAgent func(cfg AgentConfig) Agent

// MastraAgentRelevanceScorer uses a Mastra Agent to evaluate the relevance
// of a text passage to a query by asking the model to rate similarity.
type MastraAgentRelevanceScorer struct {
	agent Agent
}

// NewMastraAgentRelevanceScorer creates a new scorer backed by a Mastra Agent
// configured with a specialized relevance-scoring system prompt.
func NewMastraAgentRelevanceScorer(name string, model MastraModelConfig) *MastraAgentRelevanceScorer {
	instructions := `You are a specialized agent for evaluating the relevance of text to queries.
Your task is to rate how well a text passage answers a given query.
Output only a number between 0 and 1, where:
1.0 = Perfectly relevant, directly answers the query
0.0 = Completely irrelevant
Consider:
- Direct relevance to the question
- Completeness of information
- Quality and specificity
Always return just the number, no explanation.`

	var agent Agent
	if NewAgent != nil {
		agent = NewAgent(AgentConfig{
			ID:           fmt.Sprintf("relevance-scorer-%s", name),
			Name:         fmt.Sprintf("Relevance Scorer %s", name),
			Instructions: instructions,
			Model:        model,
		})
	}

	return &MastraAgentRelevanceScorer{
		agent: agent,
	}
}

// GetRelevanceScore evaluates the semantic relevance between query and text
// by asking the underlying agent model to rate similarity on a 0-1 scale.
func (s *MastraAgentRelevanceScorer) GetRelevanceScore(query, text string) (float64, error) {
	if s.agent == nil {
		return 0, fmt.Errorf("agent not initialized (NewAgent constructor not set)")
	}

	prompt := relevance.CreateSimilarityPrompt(query, text)

	model, err := s.agent.GetModel()
	if err != nil {
		return 0, fmt.Errorf("failed to get model: %w", err)
	}

	var response string
	if IsSupportedLanguageModel(model) {
		result, err := s.agent.Generate(prompt)
		if err != nil {
			return 0, fmt.Errorf("generate failed: %w", err)
		}
		response = result.Text
	} else {
		result, err := s.agent.GenerateLegacy(prompt)
		if err != nil {
			return 0, fmt.Errorf("generateLegacy failed: %w", err)
		}
		response = result.Text
	}

	score, err := strconv.ParseFloat(response, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse relevance score %q: %w", response, err)
	}

	return score, nil
}

// Ensure MastraAgentRelevanceScorer implements RelevanceScoreProvider.
var _ relevance.RelevanceScoreProvider = (*MastraAgentRelevanceScorer)(nil)
