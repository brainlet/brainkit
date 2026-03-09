// Ported from: packages/core/src/processors/processors/prompt-injection-detector.ts
package concreteprocessors

import (
	"fmt"
	"log"
	"math"
	"strings"

	processors "github.com/brainlet/brainkit/agent-kit/core/processors"
)

// ---------------------------------------------------------------------------
// PromptInjectionCategoryScore
// ---------------------------------------------------------------------------

// PromptInjectionCategoryScore holds a single detection category score.
type PromptInjectionCategoryScore struct {
	Type  string  `json:"type"`
	Score float64 `json:"score"`
}

// PromptInjectionResult holds the result of prompt injection detection.
type PromptInjectionResult struct {
	Categories       []PromptInjectionCategoryScore `json:"categories"`
	Reason           *string                        `json:"reason"`
	RewrittenContent *string                        `json:"rewritten_content,omitempty"`
}

// ---------------------------------------------------------------------------
// PromptInjectionOptions
// ---------------------------------------------------------------------------

// PromptInjectionStructuredOutputOptions holds structured output options for the detection agent.
type PromptInjectionStructuredOutputOptions struct {
	JSONPromptInjection bool `json:"jsonPromptInjection,omitempty"`
}

// PromptInjectionOptions configures the PromptInjectionDetector.
type PromptInjectionOptions struct {
	// Model configuration for the detection agent.
	Model MastraModelConfig

	// DetectionTypes are attack types to check for. Default: OWASP LLM01 categories.
	DetectionTypes []string

	// Threshold for flagging (0-1). Default: 0.7.
	Threshold float64

	// Strategy when injection is detected: "block", "warn", "filter", "rewrite". Default: "block".
	Strategy string

	// Instructions is custom detection instructions.
	Instructions string

	// IncludeScores includes confidence scores in logs. Default: false.
	IncludeScores bool

	// StructuredOutputOptions for the detection agent.
	StructuredOutputOptions *PromptInjectionStructuredOutputOptions

	// ProviderOptions are provider-specific options.
	ProviderOptions ProviderOptions
}

// ---------------------------------------------------------------------------
// PromptInjectionDetector
// ---------------------------------------------------------------------------

// defaultDetectionTypes are the default detection categories based on OWASP LLM01.
var defaultDetectionTypes = []string{
	"injection",
	"jailbreak",
	"tool-exfiltration",
	"data-exfiltration",
	"system-override",
	"role-manipulation",
}

// PromptInjectionDetector identifies and handles prompt injection attacks,
// jailbreaks, and tool/data exfiltration attempts.
type PromptInjectionDetector struct {
	processors.BaseProcessor
	detectionAgent          Agent
	detectionTypes          []string
	threshold               float64
	strategy                string
	includeScores           bool
	structuredOutputOptions *PromptInjectionStructuredOutputOptions
	providerOptions         ProviderOptions
}

// NewPromptInjectionDetector creates a new PromptInjectionDetector.
func NewPromptInjectionDetector(opts PromptInjectionOptions) *PromptInjectionDetector {
	detectionTypes := opts.DetectionTypes
	if len(detectionTypes) == 0 {
		detectionTypes = defaultDetectionTypes
	}

	threshold := opts.Threshold
	if threshold == 0 {
		threshold = 0.7
	}

	strategy := opts.Strategy
	if strategy == "" {
		strategy = "block"
	}

	// TODO: Create actual detection agent once Agent is ported.

	return &PromptInjectionDetector{
		BaseProcessor:           processors.NewBaseProcessor("prompt-injection-detector", "Prompt Injection Detector"),
		detectionAgent:          nil,
		detectionTypes:          detectionTypes,
		threshold:               threshold,
		strategy:                strategy,
		includeScores:           opts.IncludeScores,
		structuredOutputOptions: opts.StructuredOutputOptions,
		providerOptions:         opts.ProviderOptions,
	}
}

// ProcessInput evaluates each message for prompt injection.
func (pid *PromptInjectionDetector) ProcessInput(args processors.ProcessInputArgs) (
	[]processors.MastraDBMessage, *processors.MessageList, *processors.ProcessInputResultWithSystemMessages, error,
) {
	messages := args.Messages

	if len(messages) == 0 {
		return messages, nil, nil, nil
	}

	var processedMessages []processors.MastraDBMessage

	for _, message := range messages {
		textContent := extractTextContentFromMessage(message)
		if strings.TrimSpace(textContent) == "" {
			processedMessages = append(processedMessages, message)
			continue
		}

		detectionResult, err := pid.detectPromptInjection(textContent)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("prompt injection detection failed: %w", err)
		}

		if pid.isInjectionFlagged(detectionResult) {
			processedMsg := pid.handleDetectedInjection(message, detectionResult, pid.strategy, args.Abort)

			if pid.strategy == "filter" {
				continue
			} else if pid.strategy == "rewrite" {
				if processedMsg != nil {
					processedMessages = append(processedMessages, *processedMsg)
				}
				continue
			}
		}

		processedMessages = append(processedMessages, message)
	}

	return processedMessages, nil, nil, nil
}

// ProcessInputStep is not implemented for this processor.
func (pid *PromptInjectionDetector) ProcessInputStep(args processors.ProcessInputStepArgs) (*processors.ProcessInputStepResult, []processors.MastraDBMessage, error) {
	return nil, nil, nil
}

// ProcessOutputStream is not implemented for this processor.
func (pid *PromptInjectionDetector) ProcessOutputStream(args processors.ProcessOutputStreamArgs) (*processors.ChunkType, error) {
	return &args.Part, nil
}

// ProcessOutputResult is not implemented for this processor.
func (pid *PromptInjectionDetector) ProcessOutputResult(args processors.ProcessOutputResultArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}

// ProcessOutputStep is not implemented for this processor.
func (pid *PromptInjectionDetector) ProcessOutputStep(args processors.ProcessOutputStepArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------

// detectPromptInjection detects prompt injection using the internal agent.
func (pid *PromptInjectionDetector) detectPromptInjection(content string) (*PromptInjectionResult, error) {
	// TODO: Once Agent is ported, use the detection agent with structured output.
	log.Println("[PromptInjectionDetector] Detection agent not yet ported, allowing content")
	return &PromptInjectionResult{
		Categories:       nil,
		Reason:           nil,
		RewrittenContent: nil,
	}, nil
}

// isInjectionFlagged checks if any category score exceeds the threshold.
func (pid *PromptInjectionDetector) isInjectionFlagged(result *PromptInjectionResult) bool {
	if len(result.Categories) == 0 {
		return false
	}

	maxScore := 0.0
	for _, cat := range result.Categories {
		maxScore = math.Max(maxScore, cat.Score)
	}
	return maxScore >= pid.threshold
}

// handleDetectedInjection handles detected prompt injection based on strategy.
func (pid *PromptInjectionDetector) handleDetectedInjection(
	message processors.MastraDBMessage,
	result *PromptInjectionResult,
	strategy string,
	abort func(string, *processors.TripWireOptions) error,
) *processors.MastraDBMessage {
	var flaggedTypes []string
	for _, cat := range result.Categories {
		if cat.Score >= pid.threshold {
			flaggedTypes = append(flaggedTypes, cat.Type)
		}
	}

	alertMessage := fmt.Sprintf("Prompt injection detected. Types: %s", strings.Join(flaggedTypes, ", "))
	if result.Reason != nil {
		alertMessage += fmt.Sprintf(". Reason: %s", *result.Reason)
	}
	if pid.includeScores {
		var scores []string
		for _, cat := range result.Categories {
			scores = append(scores, fmt.Sprintf("%s: %.2f", cat.Type, cat.Score))
		}
		alertMessage += fmt.Sprintf(". Scores: %s", strings.Join(scores, ", "))
	}

	switch strategy {
	case "block":
		if abort != nil {
			_ = abort(alertMessage, nil)
		}
		return nil
	case "warn":
		log.Printf("[PromptInjectionDetector] %s", alertMessage)
		return nil
	case "filter":
		log.Printf("[PromptInjectionDetector] Filtered message: %s", alertMessage)
		return nil
	case "rewrite":
		if result.RewrittenContent != nil {
			log.Printf("[PromptInjectionDetector] Rewrote message: %s", alertMessage)
			msg := pid.createRewrittenMessage(message, *result.RewrittenContent)
			return &msg
		}
		log.Printf("[PromptInjectionDetector] No rewrite available, filtering: %s", alertMessage)
		return nil
	default:
		return nil
	}
}

// createRewrittenMessage creates a message with neutralized content.
func (pid *PromptInjectionDetector) createRewrittenMessage(originalMessage processors.MastraDBMessage, rewrittenContent string) processors.MastraDBMessage {
	msg := originalMessage
	msg.Content.Parts = []processors.MastraMessagePart{{Type: "text", Text: rewrittenContent}}
	msg.Content.Content = rewrittenContent
	return msg
}
