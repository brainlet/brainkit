// Ported from: packages/ai/src/prompt/standardize-prompt.ts
package prompt

import "fmt"

// StandardizedPrompt is the result of standardizing a prompt.
type StandardizedPrompt struct {
	// System is the system message.
	// Can be a string, SystemModelMessage, or []SystemModelMessage.
	System interface{}
	// Messages is the list of messages.
	Messages []ModelMessage
}

// InvalidPromptError is returned when the prompt is invalid.
type InvalidPromptError struct {
	Name    string
	Message string
	Prompt  Prompt
	Cause   error
}

func (e *InvalidPromptError) Error() string {
	return e.Message
}

func (e *InvalidPromptError) Unwrap() error {
	return e.Cause
}

// IsInvalidPromptError checks whether the given error is an InvalidPromptError.
func IsInvalidPromptError(err error) bool {
	_, ok := err.(*InvalidPromptError)
	return ok
}

// StandardizePrompt validates and standardizes the prompt.
func StandardizePrompt(p Prompt) (*StandardizedPrompt, error) {
	if p.PromptValue == nil && p.Messages == nil {
		return nil, &InvalidPromptError{
			Name:    "AI_InvalidPromptError",
			Message: "prompt or messages must be defined",
			Prompt:  p,
		}
	}

	if p.PromptValue != nil && p.Messages != nil {
		return nil, &InvalidPromptError{
			Name:    "AI_InvalidPromptError",
			Message: "prompt and messages cannot be defined at the same time",
			Prompt:  p,
		}
	}

	// validate system
	if p.System != nil {
		if err := validateSystem(p.System, p); err != nil {
			return nil, err
		}
	}

	var messages []ModelMessage

	if p.PromptValue != nil {
		switch v := p.PromptValue.(type) {
		case string:
			messages = []ModelMessage{
				UserModelMessage{Role: "user", Content: v},
			}
		case []ModelMessage:
			messages = v
		default:
			return nil, &InvalidPromptError{
				Name:    "AI_InvalidPromptError",
				Message: "prompt must be a string or []ModelMessage",
				Prompt:  p,
			}
		}
	} else if p.Messages != nil {
		messages = p.Messages
	} else {
		return nil, &InvalidPromptError{
			Name:    "AI_InvalidPromptError",
			Message: "prompt or messages must be defined",
			Prompt:  p,
		}
	}

	if len(messages) == 0 {
		return nil, &InvalidPromptError{
			Name:    "AI_InvalidPromptError",
			Message: "messages must not be empty",
			Prompt:  p,
		}
	}

	return &StandardizedPrompt{
		Messages: messages,
		System:   p.System,
	}, nil
}

func validateSystem(system interface{}, p Prompt) error {
	switch s := system.(type) {
	case string:
		return nil
	case SystemModelMessage:
		if s.Role != "system" {
			return &InvalidPromptError{
				Name:    "AI_InvalidPromptError",
				Message: "system must be a string, SystemModelMessage, or array of SystemModelMessage",
				Prompt:  p,
			}
		}
		return nil
	case *SystemModelMessage:
		if s.Role != "system" {
			return &InvalidPromptError{
				Name:    "AI_InvalidPromptError",
				Message: "system must be a string, SystemModelMessage, or array of SystemModelMessage",
				Prompt:  p,
			}
		}
		return nil
	case []SystemModelMessage:
		for _, msg := range s {
			if msg.Role != "system" {
				return &InvalidPromptError{
					Name:    "AI_InvalidPromptError",
					Message: "system must be a string, SystemModelMessage, or array of SystemModelMessage",
					Prompt:  p,
				}
			}
		}
		return nil
	default:
		return &InvalidPromptError{
			Name:    "AI_InvalidPromptError",
			Message: fmt.Sprintf("system must be a string, SystemModelMessage, or array of SystemModelMessage, got %T", system),
			Prompt:  p,
		}
	}
}
