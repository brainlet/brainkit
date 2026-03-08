// Ported from: packages/perplexity/src/convert-to-perplexity-messages.ts
package perplexity

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// ConvertToPerplexityMessages converts a standard prompt to the
// Perplexity message format.
func ConvertToPerplexityMessages(prompt languagemodel.Prompt) (PerplexityPrompt, error) {
	messages := PerplexityPrompt{}

	for _, msg := range prompt {
		switch m := msg.(type) {
		case languagemodel.SystemMessage:
			messages = append(messages, PerplexityMessage{
				Role:    "system",
				Content: m.Content,
			})

		case languagemodel.UserMessage:
			converted, err := convertUserOrAssistantMessage("user", m.Content)
			if err != nil {
				return nil, err
			}
			messages = append(messages, converted)

		case languagemodel.AssistantMessage:
			converted, err := convertAssistantMessage(m.Content)
			if err != nil {
				return nil, err
			}
			messages = append(messages, converted)

		case languagemodel.ToolMessage:
			return nil, errors.NewUnsupportedFunctionalityError("Tool messages", "")

		default:
			return nil, fmt.Errorf("unsupported message role: %T", msg)
		}
	}

	return messages, nil
}

// convertUserOrAssistantMessage converts user message parts to a PerplexityMessage.
func convertUserOrAssistantMessage(role string, content []languagemodel.UserMessagePart) (PerplexityMessage, error) {
	hasMultipartContent := false
	for _, part := range content {
		if fp, ok := part.(languagemodel.FilePart); ok {
			if strings.HasPrefix(fp.MediaType, "image/") || fp.MediaType == "application/pdf" {
				hasMultipartContent = true
				break
			}
		}
	}

	var messageContent []any
	for i, part := range content {
		switch p := part.(type) {
		case languagemodel.TextPart:
			messageContent = append(messageContent, PerplexityTextContent{
				Type: "text",
				Text: p.Text,
			})
		case languagemodel.FilePart:
			converted := convertFilePart(p, i)
			if converted != nil {
				messageContent = append(messageContent, converted)
			}
		}
	}

	if hasMultipartContent {
		return PerplexityMessage{
			Role:    role,
			Content: messageContent,
		}, nil
	}

	// Text-only: join as a single string
	var textParts []string
	for _, c := range messageContent {
		if tc, ok := c.(PerplexityTextContent); ok {
			textParts = append(textParts, tc.Text)
		}
	}
	return PerplexityMessage{
		Role:    role,
		Content: strings.Join(textParts, ""),
	}, nil
}

// convertAssistantMessage converts assistant message parts to a PerplexityMessage.
func convertAssistantMessage(content []languagemodel.AssistantMessagePart) (PerplexityMessage, error) {
	hasMultipartContent := false
	for _, part := range content {
		if fp, ok := part.(languagemodel.FilePart); ok {
			if strings.HasPrefix(fp.MediaType, "image/") || fp.MediaType == "application/pdf" {
				hasMultipartContent = true
				break
			}
		}
	}

	var messageContent []any
	for i, part := range content {
		switch p := part.(type) {
		case languagemodel.TextPart:
			messageContent = append(messageContent, PerplexityTextContent{
				Type: "text",
				Text: p.Text,
			})
		case languagemodel.FilePart:
			converted := convertFilePart(p, i)
			if converted != nil {
				messageContent = append(messageContent, converted)
			}
		}
	}

	if hasMultipartContent {
		return PerplexityMessage{
			Role:    "assistant",
			Content: messageContent,
		}, nil
	}

	// Text-only: join as a single string
	var textParts []string
	for _, c := range messageContent {
		if tc, ok := c.(PerplexityTextContent); ok {
			textParts = append(textParts, tc.Text)
		}
	}
	return PerplexityMessage{
		Role:    "assistant",
		Content: strings.Join(textParts, ""),
	}, nil
}

// convertFilePart converts a FilePart to a Perplexity message content part.
func convertFilePart(part languagemodel.FilePart, index int) any {
	if part.MediaType == "application/pdf" {
		dataStr := resolveDataContentToString(part.Data)
		if isURL(dataStr) {
			fileName := part.Filename
			return PerplexityFileURLContent{
				Type:     "file_url",
				FileURL:  PerplexityFileURLReference{URL: dataStr},
				FileName: fileName,
			}
		}
		defaultName := fmt.Sprintf("document-%d.pdf", index)
		fileName := &defaultName
		if part.Filename != nil {
			fileName = part.Filename
		}
		return PerplexityFileURLContent{
			Type:     "file_url",
			FileURL:  PerplexityFileURLReference{URL: dataStr},
			FileName: fileName,
		}
	}

	if strings.HasPrefix(part.MediaType, "image/") {
		dataStr := resolveDataContentToString(part.Data)
		if isURL(dataStr) {
			return PerplexityImageURLContent{
				Type:     "image_url",
				ImageURL: PerplexityImageURLReference{URL: dataStr},
			}
		}
		mediaType := part.MediaType
		if mediaType == "" {
			mediaType = "image/jpeg"
		}
		dataURL := fmt.Sprintf("data:%s;base64,%s", mediaType, dataStr)
		return PerplexityImageURLContent{
			Type:     "image_url",
			ImageURL: PerplexityImageURLReference{URL: dataURL},
		}
	}

	return nil
}

// resolveDataContentToString converts DataContent to a string representation.
func resolveDataContentToString(data languagemodel.DataContent) string {
	switch d := data.(type) {
	case languagemodel.DataContentString:
		return d.Value
	case languagemodel.DataContentBytes:
		return base64.StdEncoding.EncodeToString(d.Data)
	default:
		return ""
	}
}

// isURL checks if a string looks like a URL.
func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
