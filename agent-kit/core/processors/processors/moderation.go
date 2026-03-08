// Ported from: packages/core/src/processors/processors/moderation.ts
package concreteprocessors

import (
	"fmt"
	"log"
	"math"
	"strings"

	processors "github.com/brainlet/brainkit/agent-kit/core/processors"
)

// ---------------------------------------------------------------------------
// ModerationCategoryScore
// ---------------------------------------------------------------------------

// ModerationCategoryScore holds a single moderation category score.
type ModerationCategoryScore struct {
	Category string  `json:"category"`
	Score    float64 `json:"score"`
}

// ModerationResult holds the result of a moderation evaluation.
type ModerationResult struct {
	CategoryScores []ModerationCategoryScore `json:"category_scores"`
	Reason         *string                   `json:"reason"`
}

// ---------------------------------------------------------------------------
// ModerationOptions
// ---------------------------------------------------------------------------

// ModerationStructuredOutputOptions holds structured output options for the moderation agent.
type ModerationStructuredOutputOptions struct {
	JSONPromptInjection bool `json:"jsonPromptInjection,omitempty"`
}

// ModerationOptions configures the ModerationProcessor.
type ModerationOptions struct {
	// Model configuration for the moderation agent.
	Model MastraModelConfig

	// Categories to check for moderation. Default: OpenAI categories.
	Categories []string

	// Threshold for flagging (0-1). Default: 0.5.
	Threshold float64

	// Strategy when content is flagged: "block", "warn", "filter". Default: "block".
	Strategy string

	// Instructions is custom moderation instructions for the agent.
	Instructions string

	// IncludeScores includes confidence scores in logs. Default: false.
	IncludeScores bool

	// ChunkWindow is the number of previous chunks to include for context. Default: 0.
	ChunkWindow int

	// StructuredOutputOptions for the moderation agent.
	StructuredOutputOptions *ModerationStructuredOutputOptions

	// ProviderOptions are provider-specific options.
	ProviderOptions ProviderOptions
}

// ---------------------------------------------------------------------------
// ModerationProcessor
// ---------------------------------------------------------------------------

// defaultModerationCategories are the default OpenAI moderation categories.
var defaultModerationCategories = []string{
	"hate",
	"hate/threatening",
	"harassment",
	"harassment/threatening",
	"self-harm",
	"self-harm/intent",
	"self-harm/instructions",
	"sexual",
	"sexual/minors",
	"violence",
	"violence/graphic",
}

// ModerationProcessor uses an internal agent to evaluate content against
// configurable moderation categories for content safety.
type ModerationProcessor struct {
	processors.BaseProcessor
	moderationAgent         Agent
	categories              []string
	threshold               float64
	strategy                string
	includeScores           bool
	chunkWindow             int
	structuredOutputOptions *ModerationStructuredOutputOptions
	providerOptions         ProviderOptions
}

// NewModerationProcessor creates a new ModerationProcessor.
func NewModerationProcessor(opts ModerationOptions) *ModerationProcessor {
	categories := opts.Categories
	if len(categories) == 0 {
		categories = defaultModerationCategories
	}

	threshold := opts.Threshold
	if threshold == 0 {
		threshold = 0.5
	}

	strategy := opts.Strategy
	if strategy == "" {
		strategy = "block"
	}

	// TODO: Create actual moderation agent once Agent is ported.

	return &ModerationProcessor{
		BaseProcessor:           processors.NewBaseProcessor("moderation", "Moderation"),
		moderationAgent:         nil,
		categories:              categories,
		threshold:               threshold,
		strategy:                strategy,
		includeScores:           opts.IncludeScores,
		chunkWindow:             opts.ChunkWindow,
		structuredOutputOptions: opts.StructuredOutputOptions,
		providerOptions:         opts.ProviderOptions,
	}
}

// ProcessInput evaluates each message for moderation violations.
func (mp *ModerationProcessor) ProcessInput(args processors.ProcessInputArgs) (
	[]processors.MastraDBMessage, *processors.MessageList, *processors.ProcessInputResultWithSystemMessages, error,
) {
	messages := args.Messages

	if len(messages) == 0 {
		return messages, nil, nil, nil
	}

	var passedMessages []processors.MastraDBMessage

	for _, message := range messages {
		textContent := extractTextContentFromMessage(message)
		if strings.TrimSpace(textContent) == "" {
			passedMessages = append(passedMessages, message)
			continue
		}

		moderationResult, err := mp.moderateContent(textContent, false)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("moderation failed: %w", err)
		}

		if mp.isModerationFlagged(moderationResult) {
			err := mp.handleFlaggedContent(moderationResult, mp.strategy, args.Abort)
			if err != nil {
				return nil, nil, nil, err
			}

			// If we reach here, strategy is "warn" or "filter".
			if mp.strategy == "filter" {
				continue // Skip this message
			}
		}

		passedMessages = append(passedMessages, message)
	}

	return passedMessages, nil, nil, nil
}

// ProcessInputStep is not implemented for this processor.
func (mp *ModerationProcessor) ProcessInputStep(args processors.ProcessInputStepArgs) (*processors.ProcessInputStepResult, []processors.MastraDBMessage, error) {
	return nil, nil, nil
}

// ProcessOutputResult processes output messages for moderation (same as input).
func (mp *ModerationProcessor) ProcessOutputResult(args processors.ProcessOutputResultArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	// Reuse ProcessInput logic.
	result, _, _, err := mp.ProcessInput(processors.ProcessInputArgs{
		ProcessorMessageContext: args.ProcessorMessageContext,
		State:                   args.State,
	})
	if err != nil {
		return nil, nil, err
	}
	return result, nil, nil
}

// ProcessOutputStream moderates streaming text-delta chunks.
func (mp *ModerationProcessor) ProcessOutputStream(args processors.ProcessOutputStreamArgs) (*processors.ChunkType, error) {
	part := args.Part

	if part.Type != "text-delta" {
		return &part, nil
	}

	contentToModerate := mp.buildContextFromChunks(args.StreamParts)

	moderationResult, err := mp.moderateContent(contentToModerate, true)
	if err != nil {
		log.Printf("[ModerationProcessor] Stream moderation failed: %v", err)
		return &part, nil
	}

	if mp.isModerationFlagged(moderationResult) {
		abortErr := mp.handleFlaggedContent(moderationResult, mp.strategy, args.Abort)
		if abortErr != nil {
			return nil, abortErr
		}

		if mp.strategy == "filter" {
			return nil, nil
		}
	}

	return &part, nil
}

// ProcessOutputStep is not implemented for this processor.
func (mp *ModerationProcessor) ProcessOutputStep(args processors.ProcessOutputStepArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------

// moderateContent evaluates content using the moderation agent.
func (mp *ModerationProcessor) moderateContent(content string, isStream bool) (*ModerationResult, error) {
	// TODO: Once Agent is ported, use the moderation agent with structured output.
	log.Println("[ModerationProcessor] Moderation agent not yet ported, allowing content")
	return &ModerationResult{
		CategoryScores: nil,
		Reason:         nil,
	}, nil
}

// isModerationFlagged checks if any category score exceeds the threshold.
func (mp *ModerationProcessor) isModerationFlagged(result *ModerationResult) bool {
	if len(result.CategoryScores) == 0 {
		return false
	}

	maxScore := 0.0
	for _, cat := range result.CategoryScores {
		maxScore = math.Max(maxScore, cat.Score)
	}

	return maxScore >= mp.threshold
}

// handleFlaggedContent handles flagged content based on strategy.
func (mp *ModerationProcessor) handleFlaggedContent(
	result *ModerationResult,
	strategy string,
	abort func(string, *processors.TripWireOptions) error,
) error {
	var flaggedCategories []string
	for _, cat := range result.CategoryScores {
		if cat.Score >= mp.threshold {
			flaggedCategories = append(flaggedCategories, cat.Category)
		}
	}

	message := fmt.Sprintf("Content flagged for moderation. Categories: %s", strings.Join(flaggedCategories, ", "))
	if result.Reason != nil {
		message += fmt.Sprintf(". Reason: %s", *result.Reason)
	}
	if mp.includeScores {
		var scores []string
		for _, cat := range result.CategoryScores {
			scores = append(scores, fmt.Sprintf("%s: %.2f", cat.Category, cat.Score))
		}
		message += fmt.Sprintf(". Scores: %s", strings.Join(scores, ", "))
	}

	switch strategy {
	case "block":
		if abort != nil {
			return abort(message, nil)
		}
		return fmt.Errorf("%s", message)
	case "warn":
		log.Printf("[ModerationProcessor] %s", message)
	case "filter":
		log.Printf("[ModerationProcessor] Filtered message: %s", message)
	}

	return nil
}

// buildContextFromChunks builds context string from chunks based on chunkWindow.
func (mp *ModerationProcessor) buildContextFromChunks(streamParts []processors.ChunkType) string {
	if mp.chunkWindow == 0 {
		if len(streamParts) == 0 {
			return ""
		}
		currentChunk := streamParts[len(streamParts)-1]
		if currentChunk.Type == "text-delta" {
			if payload, ok := currentChunk.Payload.(map[string]any); ok {
				if text, ok := payload["text"].(string); ok {
					return text
				}
			}
		}
		return ""
	}

	start := len(streamParts) - mp.chunkWindow
	if start < 0 {
		start = 0
	}
	contextChunks := streamParts[start:]

	var textContent []string
	for _, part := range contextChunks {
		if part.Type == "text-delta" {
			if payload, ok := part.Payload.(map[string]any); ok {
				if text, ok := payload["text"].(string); ok {
					textContent = append(textContent, text)
				}
			}
		}
	}

	return strings.Join(textContent, "")
}
