// Ported from: packages/xai/src/responses/convert-to-xai-responses-input.ts
package xai

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// convertToXaiResponsesInputResult is the result of converting prompt to responses input.
type convertToXaiResponsesInputResult struct {
	Input         XaiResponsesInput
	InputWarnings []shared.Warning
}

// convertToXaiResponsesInput converts AI SDK prompt messages to xAI responses API input format.
func convertToXaiResponsesInput(prompt []languagemodel.Message) (convertToXaiResponsesInputResult, error) {
	var input XaiResponsesInput
	var inputWarnings []shared.Warning

	for _, message := range prompt {
		switch msg := message.(type) {
		case languagemodel.SystemMessage:
			input = append(input, XaiResponsesSystemMessage{
				Role:    "system",
				Content: msg.Content,
			})

		case languagemodel.UserMessage:
			var contentParts []XaiResponsesUserMessageContentPart

			for _, block := range msg.Content {
				switch b := block.(type) {
				case languagemodel.TextPart:
					contentParts = append(contentParts, XaiResponsesUserMessageContentPart{
						Type: "input_text",
						Text: b.Text,
					})

				case languagemodel.FilePart:
					if strings.HasPrefix(b.MediaType, "image/") {
						mediaType := b.MediaType
						if mediaType == "image/*" {
							mediaType = "image/jpeg"
						}

						var imageURL string
						switch d := b.Data.(type) {
						case languagemodel.DataContentString:
							// Check if it's a URL
							if _, err := url.ParseRequestURI(d.Value); err == nil && (strings.HasPrefix(d.Value, "http://") || strings.HasPrefix(d.Value, "https://")) {
								imageURL = d.Value
							} else {
								imageURL = fmt.Sprintf("data:%s;base64,%s", mediaType, d.Value)
							}
						case languagemodel.DataContentBytes:
							base64Data := providerutils.ConvertToBase64Bytes(d.Data)
							imageURL = fmt.Sprintf("data:%s;base64,%s", mediaType, base64Data)
						}

						contentParts = append(contentParts, XaiResponsesUserMessageContentPart{
							Type:     "input_image",
							ImageURL: imageURL,
						})
					} else {
						return convertToXaiResponsesInputResult{}, errors.NewUnsupportedFunctionalityError(
							fmt.Sprintf("file part media type %s", b.MediaType), "",
						)
					}

				default:
					inputWarnings = append(inputWarnings, shared.OtherWarning{
						Message: "xAI Responses API does not support this content type in user messages",
					})
				}
			}

			input = append(input, XaiResponsesUserMessage{
				Role:    "user",
				Content: contentParts,
			})

		case languagemodel.AssistantMessage:
			for _, part := range msg.Content {
				switch p := part.(type) {
				case languagemodel.TextPart:
					var id *string
					if p.ProviderOptions != nil {
						if xaiOpts, ok := p.ProviderOptions["xai"]; ok {
							if itemId, ok := xaiOpts["itemId"].(string); ok {
								id = &itemId
							}
						}
					}

					input = append(input, XaiResponsesAssistantMessage{
						Role:    "assistant",
						Content: p.Text,
						ID:      id,
					})

				case languagemodel.ToolCallPart:
					if p.ProviderExecuted != nil && *p.ProviderExecuted {
						continue
					}

					var id *string
					if p.ProviderOptions != nil {
						if xaiOpts, ok := p.ProviderOptions["xai"]; ok {
							if itemId, ok := xaiOpts["itemId"].(string); ok {
								id = &itemId
							}
						}
					}

					callID := p.ToolCallID
					toolID := callID
					if id != nil {
						toolID = *id
					}

					argsJSON, _ := json.Marshal(p.Input)

					input = append(input, XaiResponsesToolCallInput{
						Type:      "function_call",
						ID:        toolID,
						CallID:    &callID,
						Name:      &p.ToolName,
						Arguments: strPtr(string(argsJSON)),
						Status:    "completed",
					})

				case languagemodel.ToolResultPart:
					// Skip tool results in assistant messages
					continue

				case languagemodel.ReasoningPart:
					inputWarnings = append(inputWarnings, shared.OtherWarning{
						Message: "xAI Responses API does not support reasoning in assistant messages",
					})

				case languagemodel.FilePart:
					inputWarnings = append(inputWarnings, shared.OtherWarning{
						Message: "xAI Responses API does not support file in assistant messages",
					})

				default:
					inputWarnings = append(inputWarnings, shared.OtherWarning{
						Message: "xAI Responses API does not support this content type in assistant messages",
					})
				}
			}

		case languagemodel.ToolMessage:
			for _, part := range msg.Content {
				switch p := part.(type) {
				case languagemodel.ToolApprovalResponsePart:
					continue

				case languagemodel.ToolResultPart:
					output := p.Output

					var outputValue string
					switch o := output.(type) {
					case languagemodel.ToolResultOutputText:
						outputValue = o.Value
					case languagemodel.ToolResultOutputErrorText:
						outputValue = o.Value
					case languagemodel.ToolResultOutputExecutionDenied:
						if o.Reason != nil {
							outputValue = *o.Reason
						} else {
							outputValue = "tool execution denied"
						}
					case languagemodel.ToolResultOutputJSON:
						jsonBytes, _ := json.Marshal(o.Value)
						outputValue = string(jsonBytes)
					case languagemodel.ToolResultOutputErrorJSON:
						jsonBytes, _ := json.Marshal(o.Value)
						outputValue = string(jsonBytes)
					case languagemodel.ToolResultOutputContent:
						var parts []string
						for _, item := range o.Value {
							if textItem, ok := item.(languagemodel.ToolResultContentText); ok {
								parts = append(parts, textItem.Text)
							}
						}
						outputValue = strings.Join(parts, "")
					default:
						outputValue = ""
					}

					input = append(input, XaiResponsesFunctionCallOutput{
						Type:   "function_call_output",
						CallID: p.ToolCallID,
						Output: outputValue,
					})
				}
			}

		default:
			inputWarnings = append(inputWarnings, shared.OtherWarning{
				Message: "unsupported message role",
			})
		}
	}

	return convertToXaiResponsesInputResult{
		Input:         input,
		InputWarnings: inputWarnings,
	}, nil
}

// strPtr returns a pointer to a string.
func strPtr(s string) *string {
	return &s
}
