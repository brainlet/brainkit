// Ported from: packages/google/src/convert-to-google-generative-ai-messages.ts
package google

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// ConvertToGoogleMessagesOptions configures message conversion.
type ConvertToGoogleMessagesOptions struct {
	IsGemmaModel        bool
	ProviderOptionsName string
}

// ConvertToGoogleMessages converts a LanguageModel prompt to the Google
// Generative AI message format.
func ConvertToGoogleMessages(
	prompt languagemodel.Prompt,
	opts *ConvertToGoogleMessagesOptions,
) (*GooglePrompt, error) {
	var systemInstructionParts []GoogleTextPart
	var contents []GoogleContent
	systemMessagesAllowed := true

	isGemmaModel := false
	providerOptionsName := "google"
	if opts != nil {
		isGemmaModel = opts.IsGemmaModel
		if opts.ProviderOptionsName != "" {
			providerOptionsName = opts.ProviderOptionsName
		}
	}

	for _, msg := range prompt {
		switch m := msg.(type) {
		case languagemodel.SystemMessage:
			if !systemMessagesAllowed {
				return nil, errors.NewUnsupportedFunctionalityError(
					"system messages are only supported at the beginning of the conversation",
					"",
				)
			}
			systemInstructionParts = append(systemInstructionParts, GoogleTextPart{Text: m.Content})

		case languagemodel.UserMessage:
			systemMessagesAllowed = false
			var parts []GoogleContentPart
			for _, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.TextPart:
					text := p.Text
					parts = append(parts, GoogleContentPart{Text: &text})
				case languagemodel.FilePart:
					mediaType := p.MediaType
					if mediaType == "image/*" {
						mediaType = "image/jpeg"
					}
					switch d := p.Data.(type) {
					case languagemodel.DataContentString:
						// Check if it's a URL
						if u, err := url.Parse(d.Value); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
							parts = append(parts, GoogleContentPart{
								FileData: &GoogleFileData{
									MimeType: mediaType,
									FileURI:  d.Value,
								},
							})
						} else {
							// Base64 string
							parts = append(parts, GoogleContentPart{
								InlineData: &GoogleInlineData{
									MimeType: mediaType,
									Data:     d.Value,
								},
							})
						}
					case languagemodel.DataContentBytes:
						parts = append(parts, GoogleContentPart{
							InlineData: &GoogleInlineData{
								MimeType: mediaType,
								Data:     providerutils.ConvertBytesToBase64(d.Data),
							},
						})
					}
				}
			}
			contents = append(contents, GoogleContent{Role: "user", Parts: parts})

		case languagemodel.AssistantMessage:
			systemMessagesAllowed = false
			var parts []GoogleContentPart
			for _, part := range m.Content {
				providerOpts := getProviderOpts(part, providerOptionsName)
				thoughtSignature := extractThoughtSignature(providerOpts)

				switch p := part.(type) {
				case languagemodel.TextPart:
					if len(p.Text) == 0 {
						continue
					}
					text := p.Text
					cp := GoogleContentPart{
						Text:             &text,
						ThoughtSignature: thoughtSignature,
					}
					parts = append(parts, cp)

				case languagemodel.ReasoningPart:
					if len(p.Text) == 0 {
						continue
					}
					text := p.Text
					thought := true
					cp := GoogleContentPart{
						Text:             &text,
						Thought:          &thought,
						ThoughtSignature: thoughtSignature,
					}
					parts = append(parts, cp)

				case languagemodel.FilePart:
					switch d := p.Data.(type) {
					case languagemodel.DataContentString:
						if u, err := url.Parse(d.Value); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
							return nil, errors.NewUnsupportedFunctionalityError(
								"File data URLs in assistant messages are not supported",
								"",
							)
						}
						parts = append(parts, GoogleContentPart{
							InlineData: &GoogleInlineData{
								MimeType: p.MediaType,
								Data:     d.Value,
							},
							ThoughtSignature: thoughtSignature,
						})
					case languagemodel.DataContentBytes:
						parts = append(parts, GoogleContentPart{
							InlineData: &GoogleInlineData{
								MimeType: p.MediaType,
								Data:     providerutils.ConvertBytesToBase64(d.Data),
							},
							ThoughtSignature: thoughtSignature,
						})
					}

				case languagemodel.ToolCallPart:
					parts = append(parts, GoogleContentPart{
						FunctionCall: &GoogleFunctionCall{
							Name: p.ToolName,
							Args: p.Input,
						},
						ThoughtSignature: thoughtSignature,
					})
				}
			}
			contents = append(contents, GoogleContent{Role: "model", Parts: parts})

		case languagemodel.ToolMessage:
			systemMessagesAllowed = false
			var parts []GoogleContentPart
			for _, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.ToolApprovalResponsePart:
					continue
				case languagemodel.ToolResultPart:
					output := p.Output
					switch o := output.(type) {
					case languagemodel.ToolResultOutputContent:
						for _, contentPart := range o.Value {
							switch cp := contentPart.(type) {
							case languagemodel.ToolResultContentText:
								parts = append(parts, GoogleContentPart{
									FunctionResponse: &GoogleFunctionResponse{
										Name: p.ToolName,
										Response: map[string]any{
											"name":    p.ToolName,
											"content": cp.Text,
										},
									},
								})
							case languagemodel.ToolResultContentImageData:
								parts = append(parts, GoogleContentPart{
									InlineData: &GoogleInlineData{
										MimeType: cp.MediaType,
										Data:     cp.Data,
									},
								})
								text := "Tool executed successfully and returned this image as a response"
								parts = append(parts, GoogleContentPart{Text: &text})
							default:
								jsonBytes, _ := json.Marshal(contentPart)
								text := string(jsonBytes)
								parts = append(parts, GoogleContentPart{Text: &text})
							}
						}
					case languagemodel.ToolResultOutputExecutionDenied:
						reason := "Tool execution denied."
						if o.Reason != nil {
							reason = *o.Reason
						}
						parts = append(parts, GoogleContentPart{
							FunctionResponse: &GoogleFunctionResponse{
								Name: p.ToolName,
								Response: map[string]any{
									"name":    p.ToolName,
									"content": reason,
								},
							},
						})
					case languagemodel.ToolResultOutputText:
						parts = append(parts, GoogleContentPart{
							FunctionResponse: &GoogleFunctionResponse{
								Name: p.ToolName,
								Response: map[string]any{
									"name":    p.ToolName,
									"content": o.Value,
								},
							},
						})
					case languagemodel.ToolResultOutputJSON:
						jsonBytes, _ := json.Marshal(o.Value)
						parts = append(parts, GoogleContentPart{
							FunctionResponse: &GoogleFunctionResponse{
								Name: p.ToolName,
								Response: map[string]any{
									"name":    p.ToolName,
									"content": string(jsonBytes),
								},
							},
						})
					default:
						// For any other output types, serialize the value
						var content string
						jsonBytes, err := json.Marshal(output)
						if err != nil {
							content = fmt.Sprintf("%v", output)
						} else {
							content = string(jsonBytes)
						}
						parts = append(parts, GoogleContentPart{
							FunctionResponse: &GoogleFunctionResponse{
								Name: p.ToolName,
								Response: map[string]any{
									"name":    p.ToolName,
									"content": content,
								},
							},
						})
					}
				}
			}
			contents = append(contents, GoogleContent{Role: "user", Parts: parts})
		}
	}

	// For Gemma models, prepend system instructions to the first user message.
	if isGemmaModel && len(systemInstructionParts) > 0 && len(contents) > 0 && contents[0].Role == "user" {
		var texts []string
		for _, part := range systemInstructionParts {
			texts = append(texts, part.Text)
		}
		systemText := strings.Join(texts, "\n\n") + "\n\n"
		prependPart := GoogleContentPart{Text: &systemText}
		contents[0].Parts = append([]GoogleContentPart{prependPart}, contents[0].Parts...)
	}

	result := &GooglePrompt{
		Contents: contents,
	}

	if len(systemInstructionParts) > 0 && !isGemmaModel {
		result.SystemInstruction = &GoogleSystemInstruction{
			Parts: systemInstructionParts,
		}
	}

	return result, nil
}

func getProviderOpts(part languagemodel.AssistantMessagePart, providerOptionsName string) map[string]any {
	var providerOptions map[string]map[string]any

	switch p := part.(type) {
	case languagemodel.TextPart:
		providerOptions = p.ProviderOptions
	case languagemodel.ReasoningPart:
		providerOptions = p.ProviderOptions
	case languagemodel.FilePart:
		providerOptions = p.ProviderOptions
	case languagemodel.ToolCallPart:
		providerOptions = p.ProviderOptions
	default:
		return nil
	}

	if providerOptions == nil {
		return nil
	}

	if opts, ok := providerOptions[providerOptionsName]; ok {
		return opts
	}

	// Fallback: if using vertex, try google; if using google, try vertex
	if providerOptionsName != "google" {
		if opts, ok := providerOptions["google"]; ok {
			return opts
		}
	} else {
		if opts, ok := providerOptions["vertex"]; ok {
			return opts
		}
	}

	return nil
}

func extractThoughtSignature(providerOpts map[string]any) *string {
	if providerOpts == nil {
		return nil
	}
	if ts, ok := providerOpts["thoughtSignature"]; ok && ts != nil {
		s := fmt.Sprintf("%v", ts)
		return &s
	}
	return nil
}
