// Ported from: packages/core/src/processors/trailing-assistant-guard.ts
package processors

import (
	"regexp"
	"time"

	"github.com/google/uuid"
)

// claude46Pattern matches model IDs containing 4.6 or 4-6 (not preceded by a digit).
var claude46Pattern = regexp.MustCompile(`[^0-9]4[.\-]6`)

// IsMaybeClaude46 checks whether a model config could be Claude 4.6.
//
// Handles raw model configs (strings like "anthropic/claude-opus-4-6"),
// language model objects (with Provider and ModelID), dynamic functions
// (returns true as a safe default), and model fallback arrays.
func IsMaybeClaude46(model any) bool {
	if model == nil {
		return true
	}

	// Handle string model ID
	if s, ok := model.(string); ok {
		return len(s) >= 10 && s[:10] == "anthropic/" || claude46Pattern.MatchString(s)
	}

	// Handle MastraLanguageModel interface
	if m, ok := model.(MastraLanguageModel); ok {
		provider := m.Provider()
		modelID := m.ModelID()
		return len(provider) >= 10 && provider[:10] == "anthropic" && claude46Pattern.MatchString(modelID)
	}

	// Handle struct with Provider/ModelID fields
	type providerModel struct {
		Provider string
		ModelID  string
	}
	if pm, ok := model.(providerModel); ok {
		return len(pm.Provider) >= 9 && pm.Provider[:9] == "anthropic" && claude46Pattern.MatchString(pm.ModelID)
	}

	// Default: assume it could be Claude 4.6 for safety
	return true
}

// TrailingAssistantGuard guards against trailing assistant messages when using
// native structured output with Anthropic Claude 4.6.
//
// Claude 4.6 rejects requests where the last message is an assistant message when
// using output format (structured output), interpreting it as pre-filling the response.
// This processor appends a user message to prevent that error.
//
// This processor should only be added when the agent uses a Claude 4.6 model.
// Use IsMaybeClaude46 to check before adding.
//
// See: https://github.com/mastra-ai/mastra/issues/12800
type TrailingAssistantGuard struct {
	BaseProcessor
}

// NewTrailingAssistantGuard creates a new TrailingAssistantGuard processor.
func NewTrailingAssistantGuard() *TrailingAssistantGuard {
	return &TrailingAssistantGuard{
		BaseProcessor: NewBaseProcessor("trailing-assistant-guard", "Trailing Assistant Guard"),
	}
}

// ProcessInputStep checks for trailing assistant messages when structured output
// is active and appends a user message to prevent Claude 4.6 errors.
//
// Returns (*ProcessInputStepResult, nil, nil) if a guard message was appended,
// or (nil, nil, nil) if no changes are needed.
func (t *TrailingAssistantGuard) ProcessInputStep(args ProcessInputStepArgs) (*ProcessInputStepResult, []MastraDBMessage, error) {
	// Check if response format (native structured output) will be used
	willUseResponseFormat := args.StructuredOutput != nil &&
		args.StructuredOutput.Schema != nil &&
		args.StructuredOutput.Model == nil &&
		!args.StructuredOutput.JSONPromptInjection

	if !willUseResponseFormat {
		return nil, nil, nil
	}

	// Check if last message is an assistant message
	if len(args.Messages) == 0 {
		return nil, nil, nil
	}
	lastMessage := args.Messages[len(args.Messages)-1]
	if lastMessage.Role != "assistant" {
		return nil, nil, nil
	}

	// Append a user message to prevent the error
	guardMessage := MastraDBMessage{
		MastraMessageShared: MastraMessageShared{
			ID:        uuid.New().String(),
			Role:      "user",
			CreatedAt: time.Now(),
		},
		Content: MastraMessageContentV2{
			Format: 2,
			Parts: []MastraMessagePart{
				{
					Type: "text",
					Text: "Generate the structured response.",
				},
			},
		},
	}

	messages := make([]MastraDBMessage, len(args.Messages)+1)
	copy(messages, args.Messages)
	messages[len(args.Messages)] = guardMessage

	return &ProcessInputStepResult{
		Messages: messages,
	}, nil, nil
}

// ProcessInput is not implemented for this processor.
func (t *TrailingAssistantGuard) ProcessInput(args ProcessInputArgs) ([]MastraDBMessage, *MessageList, *ProcessInputResultWithSystemMessages, error) {
	return nil, nil, nil, nil
}

// ProcessOutputStream is not implemented for this processor.
func (t *TrailingAssistantGuard) ProcessOutputStream(args ProcessOutputStreamArgs) (*ChunkType, error) {
	return &args.Part, nil
}

// ProcessOutputResult is not implemented for this processor.
func (t *TrailingAssistantGuard) ProcessOutputResult(args ProcessOutputResultArgs) ([]MastraDBMessage, *MessageList, error) {
	return nil, nil, nil
}

// ProcessOutputStep is not implemented for this processor.
func (t *TrailingAssistantGuard) ProcessOutputStep(args ProcessOutputStepArgs) ([]MastraDBMessage, *MessageList, error) {
	return nil, nil, nil
}
