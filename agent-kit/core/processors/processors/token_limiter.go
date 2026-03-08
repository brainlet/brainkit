// Ported from: packages/core/src/processors/processors/token-limiter.ts
package concreteprocessors

import (
	"encoding/json"
	"fmt"

	processors "github.com/brainlet/brainkit/agent-kit/core/processors"
)

// ---------------------------------------------------------------------------
// Stub types for unported dependencies
// ---------------------------------------------------------------------------

// TokenEncoder is a stub for js-tiktoken/lite.Tiktoken.
// TODO: replace with actual tiktoken Go implementation once available.
type TokenEncoder interface {
	// Encode tokenizes a string and returns the token count.
	Encode(text string) []int
}

// defaultTokenEncoder is a simple whitespace-based token estimator.
// This is a placeholder until a proper tiktoken Go port is integrated.
type defaultTokenEncoder struct{}

func (e *defaultTokenEncoder) Encode(text string) []int {
	// Rough approximation: ~4 characters per token (GPT-4 average).
	tokenCount := len(text) / 4
	if tokenCount == 0 && len(text) > 0 {
		tokenCount = 1
	}
	result := make([]int, tokenCount)
	for i := range result {
		result[i] = i
	}
	return result
}

// ---------------------------------------------------------------------------
// TokenLimiterOptions
// ---------------------------------------------------------------------------

// TokenLimiterOptions configures the TokenLimiterProcessor.
type TokenLimiterOptions struct {
	// Limit is the maximum number of tokens to allow (required).
	Limit int

	// Encoder is the token encoder to use.
	// Default: simple character-based estimator (~4 chars/token).
	// TODO: replace default with o200k_base tiktoken encoding once ported.
	Encoder TokenEncoder

	// Strategy when token limit is reached: "truncate" or "abort". Default: "truncate".
	Strategy string

	// CountMode: "cumulative" (count all tokens from start) or "part" (only current part).
	// Default: "cumulative".
	CountMode string
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	// tokensPerMessage is the overhead per message in token counting.
	tokensPerMessage = 3.8

	// tokensPerConversation is the fixed overhead per conversation.
	tokensPerConversation = 24
)

// ---------------------------------------------------------------------------
// TokenLimiterProcessor
// ---------------------------------------------------------------------------

// TokenLimiterProcessor limits the number of tokens in messages.
//
// Can be used as:
//   - Input processor: Filters historical messages to fit within context window, prioritizing recent messages.
//   - Output processor: Limits generated response tokens via streaming (ProcessOutputStream) or
//     non-streaming (ProcessOutputResult).
type TokenLimiterProcessor struct {
	processors.BaseProcessor
	encoder   TokenEncoder
	maxTokens int
	strategy  string
	countMode string
}

// NewTokenLimiterProcessor creates a new TokenLimiterProcessor.
// Accepts either a simple int (token limit with defaults) or TokenLimiterOptions.
func NewTokenLimiterProcessor(limit int, opts *TokenLimiterOptions) *TokenLimiterProcessor {
	var encoder TokenEncoder
	strategy := "truncate"
	countMode := "cumulative"

	if opts != nil {
		if opts.Limit > 0 {
			limit = opts.Limit
		}
		if opts.Encoder != nil {
			encoder = opts.Encoder
		}
		if opts.Strategy != "" {
			strategy = opts.Strategy
		}
		if opts.CountMode != "" {
			countMode = opts.CountMode
		}
	}

	if encoder == nil {
		encoder = &defaultTokenEncoder{}
	}

	return &TokenLimiterProcessor{
		BaseProcessor: processors.NewBaseProcessor("token-limiter", "Token Limiter"),
		encoder:       encoder,
		maxTokens:     limit,
		strategy:      strategy,
		countMode:     countMode,
	}
}

// ProcessInput processes input messages to limit them to the configured token limit.
// This filters historical messages to fit within the token budget,
// prioritizing the most recent messages.
func (tlp *TokenLimiterProcessor) ProcessInput(args processors.ProcessInputArgs) (
	[]processors.MastraDBMessage, *processors.MessageList, *processors.ProcessInputResultWithSystemMessages, error,
) {
	messages := args.Messages

	// If no messages or empty array, return a trip wire error.
	if len(messages) == 0 {
		return nil, nil, nil, fmt.Errorf("TokenLimiterProcessor: No messages to process. Cannot send LLM a request with no messages")
	}

	limit := tlp.maxTokens

	// Calculate token count for system messages (always included, never filtered).
	var systemTokens float64
	if args.SystemMessages != nil {
		for _, msg := range args.SystemMessages {
			systemTokens += tlp.countCoreSystemMessageTokens(msg)
		}
	}

	// If system messages alone exceed the limit, return error.
	if systemTokens+tokensPerConversation >= float64(limit) {
		return nil, nil, nil, fmt.Errorf(
			"TokenLimiterProcessor: System messages alone exceed token limit. systemTokens=%.0f, limit=%d",
			systemTokens, limit,
		)
	}

	// Calculate remaining budget for non-system messages.
	remainingBudget := float64(limit) - systemTokens - tokensPerConversation

	// Process non-system messages in reverse order (newest first).
	var messagesToKeep []processors.MastraDBMessage
	var currentTokens float64

	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]
		messageTokens := tlp.countInputMessageTokens(message)

		if currentTokens+messageTokens <= remainingBudget {
			// Prepend to maintain order.
			messagesToKeep = append([]processors.MastraDBMessage{message}, messagesToKeep...)
			currentTokens += messageTokens
		}
		// Continue checking all messages, don't break early.
	}

	return messagesToKeep, nil, nil, nil
}

// ProcessOutputStream processes output stream chunks with token limiting.
func (tlp *TokenLimiterProcessor) ProcessOutputStream(args processors.ProcessOutputStreamArgs) (*processors.ChunkType, error) {
	part := args.Part
	state := args.State
	limit := tlp.maxTokens

	// Initialize currentTokens in state if not present.
	if _, ok := state["currentTokens"]; !ok {
		state["currentTokens"] = 0.0
	}

	// Count tokens in the current part.
	chunkTokens := tlp.countTokensInChunk(part)

	currentTokens, _ := state["currentTokens"].(float64)

	if tlp.countMode == "cumulative" {
		currentTokens += float64(chunkTokens)
		state["currentTokens"] = currentTokens
	} else {
		currentTokens = float64(chunkTokens)
		state["currentTokens"] = currentTokens
	}

	// Check if we've exceeded the limit.
	if int(currentTokens) > limit {
		if tlp.strategy == "abort" {
			if args.Abort != nil {
				err := args.Abort(fmt.Sprintf("Token limit of %d exceeded (current: %d)", limit, int(currentTokens)), nil)
				return nil, err
			}
		} else {
			// truncate strategy - don't emit this part.
			if tlp.countMode == "part" {
				state["currentTokens"] = 0.0
			}
			return nil, nil
		}
	}

	// If we're in part mode, reset the count for next part.
	if tlp.countMode == "part" {
		state["currentTokens"] = 0.0
	}

	return &part, nil
}

// ProcessOutputResult processes the final result (non-streaming).
// Truncates the text content if it exceeds the token limit.
func (tlp *TokenLimiterProcessor) ProcessOutputResult(args processors.ProcessOutputResultArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	messages := args.Messages
	limit := tlp.maxTokens
	cumulativeTokens := 0

	var processedMessages []processors.MastraDBMessage

	for _, message := range messages {
		if message.Role != "assistant" || len(message.Content.Parts) == 0 {
			processedMessages = append(processedMessages, message)
			continue
		}

		var processedParts []processors.MessagePart
		for _, part := range message.Content.Parts {
			if part.Type == "text" {
				textContent := part.Text
				tokens := len(tlp.encoder.Encode(textContent))

				if cumulativeTokens+tokens <= limit {
					cumulativeTokens += tokens
					processedParts = append(processedParts, part)
				} else {
					if tlp.strategy == "abort" {
						if args.Abort != nil {
							_ = args.Abort(fmt.Sprintf("Token limit of %d exceeded (current: %d)", limit, cumulativeTokens+tokens), nil)
						}
					} else {
						// Truncate the text to fit within the remaining token limit.
						remainingTokens := limit - cumulativeTokens
						truncatedText := tlp.truncateToTokenLimit(textContent, remainingTokens)
						cumulativeTokens += len(tlp.encoder.Encode(truncatedText))
						processedParts = append(processedParts, processors.MessagePart{
							Type: "text",
							Text: truncatedText,
						})
					}
				}
			} else {
				processedParts = append(processedParts, part)
			}
		}

		msg := message
		msg.Content.Parts = processedParts
		processedMessages = append(processedMessages, msg)
	}

	return processedMessages, nil, nil
}

// ProcessInputStep is not implemented for this processor.
func (tlp *TokenLimiterProcessor) ProcessInputStep(args processors.ProcessInputStepArgs) (*processors.ProcessInputStepResult, []processors.MastraDBMessage, error) {
	return nil, nil, nil
}

// ProcessOutputStep is not implemented for this processor.
func (tlp *TokenLimiterProcessor) ProcessOutputStep(args processors.ProcessOutputStepArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}

// GetMaxTokens returns the maximum token limit.
func (tlp *TokenLimiterProcessor) GetMaxTokens() int {
	return tlp.maxTokens
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------

// countCoreSystemMessageTokens counts tokens for a system message.
func (tlp *TokenLimiterProcessor) countCoreSystemMessageTokens(message processors.CoreMessageV4) float64 {
	var content string
	switch c := message.Content.(type) {
	case string:
		content = c
	default:
		return tokensPerMessage
	}

	tokenString := message.Role + content
	return float64(len(tlp.encoder.Encode(tokenString))) + tokensPerMessage
}

// countInputMessageTokens counts tokens for an input message including overhead.
func (tlp *TokenLimiterProcessor) countInputMessageTokens(message processors.MastraDBMessage) float64 {
	tokenString := message.Role
	var overhead float64

	toolResultCount := 0

	if message.Content.Content != "" && len(message.Content.Parts) == 0 {
		tokenString += message.Content.Content
	} else if len(message.Content.Parts) > 0 {
		for _, part := range message.Content.Parts {
			switch part.Type {
			case "text":
				tokenString += part.Text
			case "tool-invocation":
				if part.ToolInvocationData != nil {
					inv := part.ToolInvocationData
					if inv.State == "call" || inv.State == "partial-call" {
						tokenString += inv.ToolName
						if inv.Args != nil {
							switch a := inv.Args.(type) {
							case string:
								tokenString += a
							default:
								jsonBytes, err := json.Marshal(a)
								if err == nil {
									tokenString += string(jsonBytes)
									overhead -= 12
								}
							}
						}
					} else if inv.State == "result" {
						toolResultCount++
						if inv.Result != nil {
							switch r := inv.Result.(type) {
							case string:
								tokenString += r
							default:
								jsonBytes, err := json.Marshal(r)
								if err == nil {
									tokenString += string(jsonBytes)
									overhead -= 12
								}
							}
						}
					}
				}
			default:
				jsonBytes, err := json.Marshal(part)
				if err == nil {
					tokenString += string(jsonBytes)
				}
			}
		}
	}

	// Add message formatting overhead.
	overhead += tokensPerMessage

	// Additional overhead for each tool result (which adds an extra CoreMessage).
	if toolResultCount > 0 {
		overhead += float64(toolResultCount) * tokensPerMessage
	}

	tokenCount := float64(len(tlp.encoder.Encode(tokenString)))
	return tokenCount + overhead
}

// countTokensInChunk counts tokens in a single stream chunk.
func (tlp *TokenLimiterProcessor) countTokensInChunk(part processors.ChunkType) int {
	switch part.Type {
	case "text-delta":
		if payload, ok := part.Payload.(map[string]any); ok {
			if text, ok := payload["text"].(string); ok {
				return len(tlp.encoder.Encode(text))
			}
		}
	case "object":
		jsonBytes, err := json.Marshal(part.Payload)
		if err == nil {
			return len(tlp.encoder.Encode(string(jsonBytes)))
		}
	case "tool-call":
		if payload, ok := part.Payload.(map[string]any); ok {
			var tokenString string
			if toolName, ok := payload["toolName"].(string); ok {
				tokenString = toolName
			}
			if args, ok := payload["args"]; ok {
				switch a := args.(type) {
				case string:
					tokenString += a
				default:
					jsonBytes, err := json.Marshal(a)
					if err == nil {
						tokenString += string(jsonBytes)
					}
				}
			}
			return len(tlp.encoder.Encode(tokenString))
		}
	case "tool-result":
		if payload, ok := part.Payload.(map[string]any); ok {
			if result, ok := payload["result"]; ok {
				switch r := result.(type) {
				case string:
					return len(tlp.encoder.Encode(r))
				default:
					jsonBytes, err := json.Marshal(r)
					if err == nil {
						return len(tlp.encoder.Encode(string(jsonBytes)))
					}
				}
			}
		}
	default:
		jsonBytes, err := json.Marshal(part)
		if err == nil {
			return len(tlp.encoder.Encode(string(jsonBytes)))
		}
	}

	return 0
}

// truncateToTokenLimit truncates text to fit within a token limit using binary search.
func (tlp *TokenLimiterProcessor) truncateToTokenLimit(text string, maxTokens int) string {
	if maxTokens <= 0 {
		return ""
	}

	tokens := len(tlp.encoder.Encode(text))
	if tokens <= maxTokens {
		return text
	}

	// Binary search for the cutoff point.
	runes := []rune(text)
	left := 0
	right := len(runes)
	bestLength := 0

	for left <= right {
		mid := (left + right) / 2
		testText := string(runes[:mid])
		testTokens := len(tlp.encoder.Encode(testText))

		if testTokens <= maxTokens {
			bestLength = mid
			left = mid + 1
		} else {
			right = mid - 1
		}
	}

	return string(runes[:bestLength])
}

