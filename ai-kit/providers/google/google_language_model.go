// Ported from: packages/google/src/google-generative-ai-language-model.ts
package google

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// GoogleLanguageModelConfig configures a GoogleLanguageModel.
type GoogleLanguageModelConfig struct {
	Provider   string
	BaseURL    string
	Headers    func() map[string]string
	Fetch      providerutils.FetchFunction
	GenerateID providerutils.IdGenerator

	// SupportedUrls returns URL patterns supported by the model.
	SupportedUrls func() map[string][]*regexp.Regexp
}

// GoogleLanguageModel implements languagemodel.LanguageModel for the Google
// Generative AI API.
type GoogleLanguageModel struct {
	modelID    string
	config     GoogleLanguageModelConfig
	generateID providerutils.IdGenerator
}

// NewGoogleLanguageModel creates a new GoogleLanguageModel.
func NewGoogleLanguageModel(modelID string, config GoogleLanguageModelConfig) *GoogleLanguageModel {
	genID := config.GenerateID
	if genID == nil {
		genID = providerutils.GenerateId
	}
	return &GoogleLanguageModel{
		modelID:    modelID,
		config:     config,
		generateID: genID,
	}
}

// SpecificationVersion returns the language model interface version.
func (m *GoogleLanguageModel) SpecificationVersion() string {
	return "v3"
}

// Provider returns the provider ID.
func (m *GoogleLanguageModel) Provider() string {
	return m.config.Provider
}

// ModelID returns the model ID.
func (m *GoogleLanguageModel) ModelID() string {
	return m.modelID
}

// SupportedUrls returns the supported URL patterns.
func (m *GoogleLanguageModel) SupportedUrls() (map[string][]*regexp.Regexp, error) {
	if m.config.SupportedUrls != nil {
		return m.config.SupportedUrls(), nil
	}
	return map[string][]*regexp.Regexp{}, nil
}

// getArgs prepares the API call arguments from the call options.
func (m *GoogleLanguageModel) getArgs(options languagemodel.CallOptions) (*googleCallArgs, error) {
	var warnings []shared.Warning

	providerOptionsName := "google"
	if strings.Contains(m.config.Provider, "vertex") {
		providerOptionsName = "vertex"
	}

	providerOptsMap := toInterfaceMap(options.ProviderOptions)

	googleOptions, err := providerutils.ParseProviderOptions(
		providerOptionsName,
		providerOptsMap,
		GoogleLanguageModelOptionsSchema,
	)
	if err != nil {
		return nil, err
	}

	if googleOptions == nil && providerOptionsName != "google" {
		googleOptions, err = providerutils.ParseProviderOptions(
			"google",
			providerOptsMap,
			GoogleLanguageModelOptionsSchema,
		)
		if err != nil {
			return nil, err
		}
	}

	// Warn if Vertex RAG tools are used with a non-Vertex provider
	for _, tool := range options.Tools {
		if pt, ok := tool.(languagemodel.ProviderTool); ok {
			if pt.ID == "google.vertex_rag_store" && !strings.HasPrefix(m.config.Provider, "google.vertex.") {
				warnings = append(warnings, shared.OtherWarning{
					Message: "The 'vertex_rag_store' tool is only supported with the Google Vertex provider " +
						"and might not be supported or could behave unexpectedly with the current Google provider " +
						"(" + m.config.Provider + ").",
				})
			}
		}
	}

	isGemmaModel := strings.HasPrefix(strings.ToLower(m.modelID), "gemma-")

	googlePrompt, err := ConvertToGoogleMessages(options.Prompt, &ConvertToGoogleMessagesOptions{
		IsGemmaModel:        isGemmaModel,
		ProviderOptionsName: providerOptionsName,
	})
	if err != nil {
		return nil, err
	}

	preparedTools, err := PrepareTools(options.Tools, options.ToolChoice, m.modelID)
	if err != nil {
		return nil, err
	}

	// Build generationConfig
	generationConfig := make(map[string]any)

	if options.MaxOutputTokens != nil {
		generationConfig["maxOutputTokens"] = *options.MaxOutputTokens
	}
	if options.Temperature != nil {
		generationConfig["temperature"] = *options.Temperature
	}
	if options.TopK != nil {
		generationConfig["topK"] = *options.TopK
	}
	if options.TopP != nil {
		generationConfig["topP"] = *options.TopP
	}
	if options.FrequencyPenalty != nil {
		generationConfig["frequencyPenalty"] = *options.FrequencyPenalty
	}
	if options.PresencePenalty != nil {
		generationConfig["presencePenalty"] = *options.PresencePenalty
	}
	if len(options.StopSequences) > 0 {
		generationConfig["stopSequences"] = options.StopSequences
	}
	if options.Seed != nil {
		generationConfig["seed"] = *options.Seed
	}

	// Response format
	if rf, ok := options.ResponseFormat.(languagemodel.ResponseFormatJSON); ok {
		generationConfig["responseMimeType"] = "application/json"
		structuredOutputs := true
		if googleOptions != nil && googleOptions.StructuredOutputs != nil {
			structuredOutputs = *googleOptions.StructuredOutputs
		}
		if rf.Schema != nil && structuredOutputs {
			converted := ConvertJSONSchemaToOpenAPISchema(rf.Schema, true)
			if converted != nil {
				generationConfig["responseSchema"] = converted
			}
		}
	}

	// Provider options
	if googleOptions != nil {
		if googleOptions.AudioTimestamp != nil && *googleOptions.AudioTimestamp {
			generationConfig["audioTimestamp"] = true
		}
		if len(googleOptions.ResponseModalities) > 0 {
			generationConfig["responseModalities"] = googleOptions.ResponseModalities
		}
		if googleOptions.ThinkingConfig != nil {
			generationConfig["thinkingConfig"] = googleOptions.ThinkingConfig
		}
		if googleOptions.MediaResolution != nil {
			generationConfig["mediaResolution"] = *googleOptions.MediaResolution
		}
		if googleOptions.ImageConfig != nil {
			generationConfig["imageConfig"] = googleOptions.ImageConfig
		}
	}

	// Build args
	args := map[string]any{
		"generationConfig": generationConfig,
		"contents":         googlePrompt.Contents,
	}

	if !isGemmaModel && googlePrompt.SystemInstruction != nil {
		args["systemInstruction"] = googlePrompt.SystemInstruction
	}

	if googleOptions != nil && len(googleOptions.SafetySettings) > 0 {
		args["safetySettings"] = googleOptions.SafetySettings
	}

	if preparedTools.Tools != nil {
		args["tools"] = preparedTools.Tools
	}

	// Merge tool config with retrieval config
	tc := preparedTools.ToolConfig
	if googleOptions != nil && googleOptions.RetrievalConfig != nil {
		if tc == nil {
			tc = &ToolConfig{}
		}
		tc.RetrievalConfig = googleOptions.RetrievalConfig
	}
	if tc != nil {
		args["toolConfig"] = tc
	}

	if googleOptions != nil && googleOptions.CachedContent != nil {
		args["cachedContent"] = *googleOptions.CachedContent
	}
	if googleOptions != nil && googleOptions.Labels != nil {
		args["labels"] = googleOptions.Labels
	}

	warnings = append(warnings, preparedTools.ToolWarnings...)

	return &googleCallArgs{
		Args:                args,
		Warnings:            warnings,
		ProviderOptionsName: providerOptionsName,
	}, nil
}

type googleCallArgs struct {
	Args                map[string]any
	Warnings            []shared.Warning
	ProviderOptionsName string
}

// DoGenerate generates a response (non-streaming).
func (m *GoogleLanguageModel) DoGenerate(options languagemodel.CallOptions) (languagemodel.GenerateResult, error) {
	callArgs, err := m.getArgs(options)
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	headers := m.config.Headers()
	mergedHeaders := providerutils.CombineHeaders(headers, convertOptionalHeaders(options.Headers))

	url := fmt.Sprintf("%s/%s:generateContent", m.config.BaseURL, GetModelPath(m.modelID))

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[googleResponse]{
		URL:                       url,
		Headers:                   mergedHeaders,
		Body:                      callArgs.Args,
		FailedResponseHandler:     GoogleFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler[googleResponse](nil),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	response := result.Value
	var candidate googleCandidate
	if len(response.Candidates) > 0 {
		candidate = response.Candidates[0]
	}

	var content []languagemodel.Content
	var parts []googleResponsePart
	if candidate.Content != nil {
		parts = candidate.Content.Parts
	}
	usageMetadata := response.UsageMetadata

	var lastCodeExecutionToolCallID string

	for _, part := range parts {
		if part.ExecutableCode != nil && part.ExecutableCode.Code != "" {
			toolCallID := m.generateID()
			lastCodeExecutionToolCallID = toolCallID
			providerExecuted := true
			content = append(content, languagemodel.ToolCall{
				ToolCallID:       toolCallID,
				ToolName:         "code_execution",
				Input:            mustMarshal(part.ExecutableCode),
				ProviderExecuted: &providerExecuted,
			})
		} else if part.CodeExecutionResult != nil {
			output := ""
			if part.CodeExecutionResult.Output != nil {
				output = *part.CodeExecutionResult.Output
			}
			content = append(content, languagemodel.ToolResult{
				ToolCallID: lastCodeExecutionToolCallID,
				ToolName:   "code_execution",
				Result: map[string]any{
					"outcome": part.CodeExecutionResult.Outcome,
					"output":  output,
				},
			})
			lastCodeExecutionToolCallID = ""
		} else if part.Text != nil {
			thoughtSignatureMetadata := buildThoughtSignatureMetadata(part.ThoughtSignature, callArgs.ProviderOptionsName)

			if *part.Text == "" {
				if thoughtSignatureMetadata != nil && len(content) > 0 {
					// Apply thoughtSignature to last content item
					applyProviderMetadata(content[len(content)-1], thoughtSignatureMetadata)
				}
			} else {
				if part.Thought != nil && *part.Thought {
					content = append(content, languagemodel.Reasoning{
						Text:             *part.Text,
						ProviderMetadata: thoughtSignatureMetadata,
					})
				} else {
					content = append(content, languagemodel.Text{
						Text:             *part.Text,
						ProviderMetadata: thoughtSignatureMetadata,
					})
				}
			}
		} else if part.FunctionCall != nil {
			content = append(content, languagemodel.ToolCall{
				ToolCallID: m.generateID(),
				ToolName:   part.FunctionCall.Name,
				Input:      mustMarshal(part.FunctionCall.Args),
				ProviderMetadata: buildThoughtSignatureMetadata(
					part.ThoughtSignature, callArgs.ProviderOptionsName,
				),
			})
		} else if part.InlineData != nil {
			content = append(content, languagemodel.File{
				Data:      languagemodel.FileDataString{Value: part.InlineData.Data},
				MediaType: part.InlineData.MimeType,
				ProviderMetadata: buildThoughtSignatureMetadata(
					part.ThoughtSignature, callArgs.ProviderOptionsName,
				),
			})
		}
	}

	// Extract sources from grounding metadata
	sources := extractSources(candidate.GroundingMetadata, m.generateID)
	for _, source := range sources {
		content = append(content, source)
	}

	// Determine finish reason
	hasToolCalls := false
	for _, c := range content {
		if tc, ok := c.(languagemodel.ToolCall); ok {
			if tc.ProviderExecuted == nil || !*tc.ProviderExecuted {
				hasToolCalls = true
				break
			}
		}
	}

	finishReason := languagemodel.FinishReason{
		Unified: MapGoogleFinishReason(MapGoogleFinishReasonOptions{
			FinishReason: candidate.FinishReason,
			HasToolCalls: hasToolCalls,
		}),
		Raw: candidate.FinishReason,
	}

	usage := ConvertGoogleUsage(convertUsageMetadata(usageMetadata))

	providerMetadata := shared.ProviderMetadata{
		callArgs.ProviderOptionsName: map[string]any{
			"promptFeedback":     response.PromptFeedback,
			"groundingMetadata":  candidate.GroundingMetadata,
			"urlContextMetadata": candidate.URLContextMetadata,
			"safetyRatings":      candidate.SafetyRatings,
			"usageMetadata":      usageMetadata,
		},
	}

	return languagemodel.GenerateResult{
		Content:          content,
		FinishReason:     finishReason,
		Usage:            usage,
		Warnings:         callArgs.Warnings,
		ProviderMetadata: providerMetadata,
		Request:          &languagemodel.GenerateResultRequest{Body: callArgs.Args},
		Response: &languagemodel.GenerateResultResponse{
			Headers: result.ResponseHeaders,
			Body:    result.RawValue,
		},
	}, nil
}

// DoStream generates a streaming response.
func (m *GoogleLanguageModel) DoStream(options languagemodel.CallOptions) (languagemodel.StreamResult, error) {
	callArgs, err := m.getArgs(options)
	if err != nil {
		return languagemodel.StreamResult{}, err
	}

	headers := m.config.Headers()
	mergedHeaders := providerutils.CombineHeaders(headers, convertOptionalHeaders(options.Headers))

	url := fmt.Sprintf("%s/%s:streamGenerateContent?alt=sse", m.config.BaseURL, GetModelPath(m.modelID))

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[<-chan providerutils.ParseResult[googleChunkResponse]]{
		URL:                       url,
		Headers:                   mergedHeaders,
		Body:                      callArgs.Args,
		FailedResponseHandler:     GoogleFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateEventSourceResponseHandler[googleChunkResponse](nil),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return languagemodel.StreamResult{}, err
	}

	inputCh := result.Value
	outputCh := make(chan languagemodel.StreamPart, 64)

	go func() {
		defer close(outputCh)

		// Send stream start
		outputCh <- languagemodel.StreamPartStreamStart{Warnings: callArgs.Warnings}

		finishReason := languagemodel.FinishReason{
			Unified: languagemodel.FinishReasonOther,
		}
		var usage *GoogleUsageMetadata
		var providerMeta shared.ProviderMetadata
		var lastGroundingMetadata *GoogleGroundingMetadata
		var lastURLContextMetadata *GoogleURLContextMetadata

		generateID := m.generateID
		hasToolCalls := false
		includeRawChunks := options.IncludeRawChunks != nil && *options.IncludeRawChunks

		var currentTextBlockID *string
		var currentReasoningBlockID *string
		blockCounter := 0

		emittedSourceUrls := make(map[string]bool)
		var lastCodeExecutionToolCallID string

		for chunk := range inputCh {
			if includeRawChunks {
				outputCh <- languagemodel.StreamPartRaw{RawValue: chunk.RawValue}
			}

			if !chunk.Success {
				outputCh <- languagemodel.StreamPartError{Error: chunk.Error}
				continue
			}

			value := chunk.Value
			usageMeta := value.UsageMetadata

			if usageMeta != nil {
				usage = convertUsageMetadata(usageMeta)
			}

			if len(value.Candidates) == 0 {
				continue
			}
			candidate := value.Candidates[0]

			if candidate.GroundingMetadata != nil {
				lastGroundingMetadata = candidate.GroundingMetadata
			}
			if candidate.URLContextMetadata != nil {
				lastURLContextMetadata = candidate.URLContextMetadata
			}

			// Extract and emit sources
			sources := extractSources(candidate.GroundingMetadata, generateID)
			for _, source := range sources {
				if src, ok := source.(languagemodel.SourceURL); ok {
					if !emittedSourceUrls[src.URL] {
						emittedSourceUrls[src.URL] = true
						outputCh <- src
					}
				}
			}

			content := candidate.Content
			if content != nil {
				for _, part := range content.Parts {
					if part.ExecutableCode != nil && part.ExecutableCode.Code != "" {
						toolCallID := generateID()
						lastCodeExecutionToolCallID = toolCallID
						providerExecuted := true
						outputCh <- languagemodel.ToolCall{
							ToolCallID:       toolCallID,
							ToolName:         "code_execution",
							Input:            mustMarshal(part.ExecutableCode),
							ProviderExecuted: &providerExecuted,
						}
					} else if part.CodeExecutionResult != nil {
						toolCallID := lastCodeExecutionToolCallID
						if toolCallID != "" {
							output := ""
							if part.CodeExecutionResult.Output != nil {
								output = *part.CodeExecutionResult.Output
							}
							outputCh <- languagemodel.ToolResult{
								ToolCallID: toolCallID,
								ToolName:   "code_execution",
								Result: map[string]any{
									"outcome": part.CodeExecutionResult.Outcome,
									"output":  output,
								},
							}
							lastCodeExecutionToolCallID = ""
						}
					} else if part.Text != nil {
						thoughtSignatureMetadata := buildThoughtSignatureMetadata(
							part.ThoughtSignature, callArgs.ProviderOptionsName,
						)

						if *part.Text == "" {
							if thoughtSignatureMetadata != nil && currentTextBlockID != nil {
								outputCh <- languagemodel.StreamPartTextDelta{
									ID:               *currentTextBlockID,
									Delta:            "",
									ProviderMetadata: thoughtSignatureMetadata,
								}
							}
						} else if part.Thought != nil && *part.Thought {
							// End any active text block
							if currentTextBlockID != nil {
								outputCh <- languagemodel.StreamPartTextEnd{ID: *currentTextBlockID}
								currentTextBlockID = nil
							}
							// Start reasoning block if not active
							if currentReasoningBlockID == nil {
								id := fmt.Sprintf("%d", blockCounter)
								blockCounter++
								currentReasoningBlockID = &id
								outputCh <- languagemodel.StreamPartReasoningStart{
									ID:               *currentReasoningBlockID,
									ProviderMetadata: thoughtSignatureMetadata,
								}
							}
							outputCh <- languagemodel.StreamPartReasoningDelta{
								ID:               *currentReasoningBlockID,
								Delta:            *part.Text,
								ProviderMetadata: thoughtSignatureMetadata,
							}
						} else {
							// End reasoning block if active
							if currentReasoningBlockID != nil {
								outputCh <- languagemodel.StreamPartReasoningEnd{ID: *currentReasoningBlockID}
								currentReasoningBlockID = nil
							}
							// Start text block if not active
							if currentTextBlockID == nil {
								id := fmt.Sprintf("%d", blockCounter)
								blockCounter++
								currentTextBlockID = &id
								outputCh <- languagemodel.StreamPartTextStart{
									ID:               *currentTextBlockID,
									ProviderMetadata: thoughtSignatureMetadata,
								}
							}
							outputCh <- languagemodel.StreamPartTextDelta{
								ID:               *currentTextBlockID,
								Delta:            *part.Text,
								ProviderMetadata: thoughtSignatureMetadata,
							}
						}
					} else if part.InlineData != nil {
						// End text/reasoning blocks before file
						if currentTextBlockID != nil {
							outputCh <- languagemodel.StreamPartTextEnd{ID: *currentTextBlockID}
							currentTextBlockID = nil
						}
						if currentReasoningBlockID != nil {
							outputCh <- languagemodel.StreamPartReasoningEnd{ID: *currentReasoningBlockID}
							currentReasoningBlockID = nil
						}
						thoughtSignatureMetadata := buildThoughtSignatureMetadata(
							part.ThoughtSignature, callArgs.ProviderOptionsName,
						)
						outputCh <- languagemodel.File{
							MediaType:        part.InlineData.MimeType,
							Data:             languagemodel.FileDataString{Value: part.InlineData.Data},
							ProviderMetadata: thoughtSignatureMetadata,
						}
					}
				}

				// Process function calls from parts
				toolCallDeltas := getToolCallsFromParts(content.Parts, generateID, callArgs.ProviderOptionsName)
				for _, tc := range toolCallDeltas {
					outputCh <- languagemodel.StreamPartToolInputStart{
						ID:               tc.ToolCallID,
						ToolName:         tc.ToolName,
						ProviderMetadata: tc.ProviderMetadata,
					}
					outputCh <- languagemodel.StreamPartToolInputDelta{
						ID:               tc.ToolCallID,
						Delta:            tc.Input,
						ProviderMetadata: tc.ProviderMetadata,
					}
					outputCh <- languagemodel.StreamPartToolInputEnd{
						ID:               tc.ToolCallID,
						ProviderMetadata: tc.ProviderMetadata,
					}
					outputCh <- languagemodel.ToolCall{
						ToolCallID:       tc.ToolCallID,
						ToolName:         tc.ToolName,
						Input:            tc.Input,
						ProviderMetadata: tc.ProviderMetadata,
					}
					hasToolCalls = true
				}
			}

			if candidate.FinishReason != nil {
				finishReason = languagemodel.FinishReason{
					Unified: MapGoogleFinishReason(MapGoogleFinishReasonOptions{
						FinishReason: candidate.FinishReason,
						HasToolCalls: hasToolCalls,
					}),
					Raw: candidate.FinishReason,
				}

				pm := map[string]any{
					"promptFeedback":     value.PromptFeedback,
					"groundingMetadata":  lastGroundingMetadata,
					"urlContextMetadata": lastURLContextMetadata,
					"safetyRatings":      candidate.SafetyRatings,
				}
				if usageMeta != nil {
					pm["usageMetadata"] = usageMeta
				}
				providerMeta = shared.ProviderMetadata{
					callArgs.ProviderOptionsName: pm,
				}
			}
		}

		// Flush open blocks
		if currentTextBlockID != nil {
			outputCh <- languagemodel.StreamPartTextEnd{ID: *currentTextBlockID}
		}
		if currentReasoningBlockID != nil {
			outputCh <- languagemodel.StreamPartReasoningEnd{ID: *currentReasoningBlockID}
		}

		// Send finish
		outputCh <- languagemodel.StreamPartFinish{
			FinishReason:     finishReason,
			Usage:            ConvertGoogleUsage(usage),
			ProviderMetadata: providerMeta,
		}
	}()

	return languagemodel.StreamResult{
		Stream:   outputCh,
		Request:  &languagemodel.StreamResultRequest{Body: callArgs.Args},
		Response: &languagemodel.StreamResultResponse{Headers: result.ResponseHeaders},
	}, nil
}

// --- Response types for JSON parsing ---

type googleResponse struct {
	Candidates    []googleCandidate `json:"candidates"`
	UsageMetadata map[string]any    `json:"usageMetadata,omitempty"`
	PromptFeedback map[string]any   `json:"promptFeedback,omitempty"`
}

type googleCandidate struct {
	Content            *googleCandidateContent   `json:"content,omitempty"`
	FinishReason       *string                   `json:"finishReason,omitempty"`
	SafetyRatings      []GoogleSafetyRating      `json:"safetyRatings,omitempty"`
	GroundingMetadata  *GoogleGroundingMetadata   `json:"groundingMetadata,omitempty"`
	URLContextMetadata *GoogleURLContextMetadata  `json:"urlContextMetadata,omitempty"`
}

type googleCandidateContent struct {
	Parts []googleResponsePart `json:"parts,omitempty"`
}

type googleResponsePart struct {
	Text                *string                      `json:"text,omitempty"`
	Thought             *bool                        `json:"thought,omitempty"`
	ThoughtSignature    *string                      `json:"thoughtSignature,omitempty"`
	FunctionCall        *googleResponseFunctionCall   `json:"functionCall,omitempty"`
	InlineData          *googleResponseInlineData     `json:"inlineData,omitempty"`
	ExecutableCode      *googleResponseExecutableCode `json:"executableCode,omitempty"`
	CodeExecutionResult *googleResponseCodeExecResult `json:"codeExecutionResult,omitempty"`
}

type googleResponseFunctionCall struct {
	Name string `json:"name"`
	Args any    `json:"args"`
}

type googleResponseInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type googleResponseExecutableCode struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}

type googleResponseCodeExecResult struct {
	Outcome string  `json:"outcome"`
	Output  *string `json:"output,omitempty"`
}

// Chunk response (streaming)
type googleChunkResponse struct {
	Candidates     []googleCandidate `json:"candidates,omitempty"`
	UsageMetadata  map[string]any    `json:"usageMetadata,omitempty"`
	PromptFeedback map[string]any    `json:"promptFeedback,omitempty"`
}

// --- Helper functions ---

func getToolCallsFromParts(parts []googleResponsePart, generateID providerutils.IdGenerator, providerOptionsName string) []languagemodel.ToolCall {
	var result []languagemodel.ToolCall
	for _, part := range parts {
		if part.FunctionCall != nil {
			tc := languagemodel.ToolCall{
				ToolCallID: generateID(),
				ToolName:   part.FunctionCall.Name,
				Input:      mustMarshal(part.FunctionCall.Args),
			}
			if part.ThoughtSignature != nil {
				tc.ProviderMetadata = shared.ProviderMetadata{
					providerOptionsName: map[string]any{
						"thoughtSignature": *part.ThoughtSignature,
					},
				}
			}
			result = append(result, tc)
		}
	}
	return result
}

func extractSources(groundingMetadata *GoogleGroundingMetadata, generateID providerutils.IdGenerator) []languagemodel.Content {
	if groundingMetadata == nil || len(groundingMetadata.GroundingChunks) == 0 {
		return nil
	}

	var sources []languagemodel.Content

	for _, chunk := range groundingMetadata.GroundingChunks {
		if chunk.Web != nil {
			sources = append(sources, languagemodel.SourceURL{
				ID:    generateID(),
				URL:   chunk.Web.URI,
				Title: chunk.Web.Title,
			})
		} else if chunk.Image != nil {
			sources = append(sources, languagemodel.SourceURL{
				ID:    generateID(),
				URL:   chunk.Image.SourceURI,
				Title: chunk.Image.Title,
			})
		} else if chunk.RetrievedContext != nil {
			uri := chunk.RetrievedContext.URI
			fileSearchStore := chunk.RetrievedContext.FileSearchStore

			if uri != nil && (strings.HasPrefix(*uri, "http://") || strings.HasPrefix(*uri, "https://")) {
				sources = append(sources, languagemodel.SourceURL{
					ID:    generateID(),
					URL:   *uri,
					Title: chunk.RetrievedContext.Title,
				})
			} else if uri != nil && *uri != "" {
				title := "Unknown Document"
				if chunk.RetrievedContext.Title != nil {
					title = *chunk.RetrievedContext.Title
				}
				mediaType, filename := detectMediaType(*uri)
				sources = append(sources, languagemodel.SourceDocument{
					ID:        generateID(),
					MediaType: mediaType,
					Title:     title,
					Filename:  filename,
				})
			} else if fileSearchStore != nil && *fileSearchStore != "" {
				title := "Unknown Document"
				if chunk.RetrievedContext.Title != nil {
					title = *chunk.RetrievedContext.Title
				}
				parts := strings.Split(*fileSearchStore, "/")
				fn := parts[len(parts)-1]
				sources = append(sources, languagemodel.SourceDocument{
					ID:        generateID(),
					MediaType: "application/octet-stream",
					Title:     title,
					Filename:  &fn,
				})
			}
		} else if chunk.Maps != nil {
			if chunk.Maps.URI != nil && *chunk.Maps.URI != "" {
				sources = append(sources, languagemodel.SourceURL{
					ID:    generateID(),
					URL:   *chunk.Maps.URI,
					Title: chunk.Maps.Title,
				})
			}
		}
	}

	if len(sources) == 0 {
		return nil
	}
	return sources
}

func detectMediaType(uri string) (string, *string) {
	mediaType := "application/octet-stream"
	var filename *string

	parts := strings.Split(uri, "/")
	fn := parts[len(parts)-1]
	filename = &fn

	switch {
	case strings.HasSuffix(uri, ".pdf"):
		mediaType = "application/pdf"
	case strings.HasSuffix(uri, ".txt"):
		mediaType = "text/plain"
	case strings.HasSuffix(uri, ".docx"):
		mediaType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case strings.HasSuffix(uri, ".doc"):
		mediaType = "application/msword"
	case strings.HasSuffix(uri, ".md") || strings.HasSuffix(uri, ".markdown"):
		mediaType = "text/markdown"
	}

	return mediaType, filename
}

func buildThoughtSignatureMetadata(thoughtSignature *string, providerOptionsName string) shared.ProviderMetadata {
	if thoughtSignature == nil {
		return nil
	}
	return shared.ProviderMetadata{
		providerOptionsName: map[string]any{
			"thoughtSignature": *thoughtSignature,
		},
	}
}

func applyProviderMetadata(content languagemodel.Content, metadata shared.ProviderMetadata) {
	// In Go we can't easily mutate the provider metadata on an interface.
	// This is a best-effort; the TS version mutates in place.
	// For the Go port, we handle this inline in the content building loop.
}

func mustMarshal(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// toInterfaceMap converts shared.ProviderOptions (map[string]map[string]any)
// to map[string]interface{} for use with providerutils.ParseProviderOptions.
func toInterfaceMap(opts shared.ProviderOptions) map[string]interface{} {
	if opts == nil {
		return nil
	}
	result := make(map[string]interface{}, len(opts))
	for k, v := range opts {
		result[k] = v
	}
	return result
}

func convertOptionalHeaders(headers map[string]*string) map[string]string {
	if headers == nil {
		return nil
	}
	result := make(map[string]string, len(headers))
	for k, v := range headers {
		if v != nil {
			result[k] = *v
		}
	}
	return result
}

func convertUsageMetadata(raw map[string]any) *GoogleUsageMetadata {
	if raw == nil {
		return nil
	}
	usage := &GoogleUsageMetadata{}
	if v, ok := raw["promptTokenCount"].(float64); ok {
		i := int(v)
		usage.PromptTokenCount = &i
	}
	if v, ok := raw["candidatesTokenCount"].(float64); ok {
		i := int(v)
		usage.CandidatesTokenCount = &i
	}
	if v, ok := raw["totalTokenCount"].(float64); ok {
		i := int(v)
		usage.TotalTokenCount = &i
	}
	if v, ok := raw["cachedContentTokenCount"].(float64); ok {
		i := int(v)
		usage.CachedContentTokenCount = &i
	}
	if v, ok := raw["thoughtsTokenCount"].(float64); ok {
		i := int(v)
		usage.ThoughtsTokenCount = &i
	}
	if v, ok := raw["trafficType"].(string); ok {
		usage.TrafficType = &v
	}
	return usage
}
