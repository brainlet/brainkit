// Ported from: packages/openai-compatible/src/completion/convert-to-openai-compatible-completion-prompt.ts
package openaicompatible

import (
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// CompletionPromptResult holds the converted completion prompt string and
// any stop sequences that should be used.
type CompletionPromptResult struct {
	Prompt        string
	StopSequences []string
}

// ConvertToCompletionPrompt converts a language model prompt (multi-message format)
// to a single-string completion prompt with user:/assistant: prefixes.
func ConvertToCompletionPrompt(prompt languagemodel.Prompt, user string, assistant string) (*CompletionPromptResult, error) {
	if user == "" {
		user = "user"
	}
	if assistant == "" {
		assistant = "assistant"
	}

	var text strings.Builder

	// Start index; if first message is system, prepend its content
	startIdx := 0
	if len(prompt) > 0 {
		if sysMsg, ok := prompt[0].(languagemodel.SystemMessage); ok {
			text.WriteString(sysMsg.Content)
			text.WriteString("\n\n")
			startIdx = 1
		}
	}

	for _, msg := range prompt[startIdx:] {
		switch m := msg.(type) {
		case languagemodel.SystemMessage:
			return nil, errors.NewInvalidPromptError(
				prompt,
				fmt.Sprintf("Unexpected system message in prompt: %s", m.Content),
				nil,
			)

		case languagemodel.UserMessage:
			var parts []string
			for _, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.TextPart:
					parts = append(parts, p.Text)
				// Other part types (e.g., FilePart) are silently ignored
				// in the completion prompt, matching the TS behavior that
				// filters on part.type === 'text'.
				}
			}
			userMessage := strings.Join(parts, "")
			text.WriteString(user)
			text.WriteString(":\n")
			text.WriteString(userMessage)
			text.WriteString("\n\n")

		case languagemodel.AssistantMessage:
			var parts []string
			for _, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.TextPart:
					parts = append(parts, p.Text)
				case languagemodel.ToolCallPart:
					return nil, errors.NewUnsupportedFunctionalityError("tool-call messages", "")
				}
			}
			assistantMessage := strings.Join(parts, "")
			text.WriteString(assistant)
			text.WriteString(":\n")
			text.WriteString(assistantMessage)
			text.WriteString("\n\n")

		case languagemodel.ToolMessage:
			return nil, errors.NewUnsupportedFunctionalityError("tool messages", "")

		default:
			return nil, fmt.Errorf("unsupported role: %T", msg)
		}
	}

	// Assistant message prefix:
	text.WriteString(assistant)
	text.WriteString(":\n")

	return &CompletionPromptResult{
		Prompt:        text.String(),
		StopSequences: []string{"\n" + user + ":"},
	}, nil
}
