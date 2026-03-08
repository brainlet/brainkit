// Ported from: packages/openai/src/completion/convert-to-openai-completion-prompt.ts
package openai

import (
	"fmt"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// ConvertToOpenAICompletionPromptResult holds the converted completion prompt.
type ConvertToOpenAICompletionPromptResult struct {
	Prompt        string
	StopSequences []string
}

// ConvertToOpenAICompletionPrompt converts a standard prompt to an OpenAI completion prompt string.
func ConvertToOpenAICompletionPrompt(
	prompt languagemodel.Prompt,
	user string,
	assistant string,
) ConvertToOpenAICompletionPromptResult {
	if user == "" {
		user = "user"
	}
	if assistant == "" {
		assistant = "assistant"
	}

	text := ""

	// If first message is a system message, add it to the text
	remaining := prompt
	if len(remaining) > 0 {
		if sys, ok := remaining[0].(languagemodel.SystemMessage); ok {
			text += sys.Content + "\n\n"
			remaining = remaining[1:]
		}
	}

	for _, msg := range remaining {
		switch m := msg.(type) {
		case languagemodel.SystemMessage:
			panic(errors.NewInvalidPromptError(
				prompt,
				fmt.Sprintf("Unexpected system message in prompt: %s", m.Content),
				nil,
			))

		case languagemodel.UserMessage:
			userMessage := ""
			for _, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.TextPart:
					userMessage += p.Text
				}
			}
			text += fmt.Sprintf("%s:\n%s\n\n", user, userMessage)

		case languagemodel.AssistantMessage:
			assistantMessage := ""
			for _, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.TextPart:
					assistantMessage += p.Text
				case languagemodel.ToolCallPart:
					panic(errors.NewUnsupportedFunctionalityError("tool-call messages", ""))
				}
			}
			text += fmt.Sprintf("%s:\n%s\n\n", assistant, assistantMessage)

		case languagemodel.ToolMessage:
			panic(errors.NewUnsupportedFunctionalityError("tool messages", ""))

		default:
			panic(fmt.Sprintf("Unsupported role: %T", m))
		}
	}

	// Assistant message prefix
	text += fmt.Sprintf("%s:\n", assistant)

	return ConvertToOpenAICompletionPromptResult{
		Prompt:        text,
		StopSequences: []string{fmt.Sprintf("\n%s:", user)},
	}
}
