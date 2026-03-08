// Ported from: packages/core/src/processors/processors/pii-detector.ts
package concreteprocessors

import (
	"crypto/sha256"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"

	processors "github.com/brainlet/brainkit/agent-kit/core/processors"
)

// ---------------------------------------------------------------------------
// PIICategories
// ---------------------------------------------------------------------------

// PIICategories holds PII detection flags per category.
type PIICategories map[string]bool

// PIICategoryScore holds a single PII category score.
type PIICategoryScore struct {
	Type  string  `json:"type"`
	Score float64 `json:"score"`
}

// PIIDetection holds a single PII detection with location and redaction info.
type PIIDetection struct {
	Type          string  `json:"type"`
	Value         string  `json:"value"`
	Confidence    float64 `json:"confidence"`
	Start         int     `json:"start"`
	End           int     `json:"end"`
	RedactedValue *string `json:"redacted_value,omitempty"`
}

// PIIDetectionResult holds the result of PII detection.
type PIIDetectionResult struct {
	Categories      []PIICategoryScore `json:"categories"`
	Detections      []PIIDetection     `json:"detections"`
	RedactedContent *string            `json:"redacted_content,omitempty"`
}

// ---------------------------------------------------------------------------
// PIIDetectorOptions
// ---------------------------------------------------------------------------

// PIIDetectorStructuredOutputOptions holds structured output options for the detection agent.
type PIIDetectorStructuredOutputOptions struct {
	JSONPromptInjection bool `json:"jsonPromptInjection,omitempty"`
}

// PIIDetectorOptions configures the PIIDetector processor.
type PIIDetectorOptions struct {
	// Model configuration for the detection agent.
	Model MastraModelConfig

	// DetectionTypes are PII types to detect. Default: comprehensive list.
	DetectionTypes []string

	// Threshold for flagging (0-1). Default: 0.6.
	Threshold float64

	// Strategy when PII is detected: "block", "warn", "filter", "redact". Default: "redact".
	Strategy string

	// RedactionMethod: "mask", "hash", "remove", "placeholder". Default: "mask".
	RedactionMethod string

	// Instructions is custom detection instructions.
	Instructions string

	// IncludeDetections includes detection details in logs. Default: false.
	IncludeDetections bool

	// PreserveFormat preserves PII format during redaction. Default: true.
	PreserveFormat *bool

	// StructuredOutputOptions for the detection agent.
	StructuredOutputOptions *PIIDetectorStructuredOutputOptions

	// ProviderOptions are provider-specific options.
	ProviderOptions ProviderOptions
}

// ---------------------------------------------------------------------------
// PIIDetector
// ---------------------------------------------------------------------------

// defaultPIIDetectionTypes are the default PII types to detect.
var defaultPIIDetectionTypes = []string{
	"email",
	"phone",
	"credit-card",
	"ssn",
	"api-key",
	"ip-address",
	"name",
	"address",
	"date-of-birth",
	"url",
	"uuid",
	"crypto-wallet",
	"iban",
}

// PIIDetector identifies and redacts personally identifiable information
// for privacy compliance.
type PIIDetector struct {
	processors.BaseProcessor
	detectionAgent          Agent
	detectionTypes          []string
	threshold               float64
	strategy                string
	redactionMethod         string
	includeDetections       bool
	preserveFormat          bool
	structuredOutputOptions *PIIDetectorStructuredOutputOptions
	providerOptions         ProviderOptions
}

// NewPIIDetector creates a new PIIDetector processor.
func NewPIIDetector(opts PIIDetectorOptions) *PIIDetector {
	detectionTypes := opts.DetectionTypes
	if len(detectionTypes) == 0 {
		detectionTypes = defaultPIIDetectionTypes
	}

	threshold := opts.Threshold
	if threshold == 0 {
		threshold = 0.6
	}

	strategy := opts.Strategy
	if strategy == "" {
		strategy = "redact"
	}

	redactionMethod := opts.RedactionMethod
	if redactionMethod == "" {
		redactionMethod = "mask"
	}

	preserveFormat := true
	if opts.PreserveFormat != nil {
		preserveFormat = *opts.PreserveFormat
	}

	// TODO: Create actual detection agent once Agent is ported.

	return &PIIDetector{
		BaseProcessor:           processors.NewBaseProcessor("pii-detector", "PII Detector"),
		detectionAgent:          nil,
		detectionTypes:          detectionTypes,
		threshold:               threshold,
		strategy:                strategy,
		redactionMethod:         redactionMethod,
		includeDetections:       opts.IncludeDetections,
		preserveFormat:          preserveFormat,
		structuredOutputOptions: opts.StructuredOutputOptions,
		providerOptions:         opts.ProviderOptions,
	}
}

// ProcessInput evaluates each message for PII.
func (pd *PIIDetector) ProcessInput(args processors.ProcessInputArgs) (
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

		detectionResult, err := pd.detectPII(textContent)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("PII detection failed: %w", err)
		}

		if pd.isPIIFlagged(detectionResult) {
			processedMsg := pd.handleDetectedPII(message, detectionResult, pd.strategy, args.Abort)

			if pd.strategy == "filter" {
				continue
			} else if pd.strategy == "redact" {
				if processedMsg != nil {
					processedMessages = append(processedMessages, *processedMsg)
				} else {
					processedMessages = append(processedMessages, message)
				}
				continue
			}
		}

		processedMessages = append(processedMessages, message)
	}

	return processedMessages, nil, nil, nil
}

// ProcessInputStep is not implemented for this processor.
func (pd *PIIDetector) ProcessInputStep(args processors.ProcessInputStepArgs) (*processors.ProcessInputStepResult, []processors.MastraDBMessage, error) {
	return nil, nil, nil
}

// ProcessOutputStream processes streaming chunks for PII.
func (pd *PIIDetector) ProcessOutputStream(args processors.ProcessOutputStreamArgs) (*processors.ChunkType, error) {
	part := args.Part

	if part.Type != "text-delta" {
		return &part, nil
	}

	var textContent string
	if payload, ok := part.Payload.(map[string]any); ok {
		textContent, _ = payload["text"].(string)
	}
	if strings.TrimSpace(textContent) == "" {
		return &part, nil
	}

	detectionResult, err := pd.detectPII(textContent)
	if err != nil {
		log.Printf("[PIIDetector] Streaming detection failed, allowing content: %v", err)
		return &part, nil
	}

	if pd.isPIIFlagged(detectionResult) {
		detectedTypes := pd.getDetectedTypes(detectionResult)

		switch pd.strategy {
		case "block":
			if args.Abort != nil {
				err := args.Abort(fmt.Sprintf("PII detected in streaming content. Types: %s", strings.Join(detectedTypes, ", ")), nil)
				return nil, err
			}
		case "warn":
			log.Printf("[PIIDetector] PII detected in streaming content: %s", strings.Join(detectedTypes, ", "))
			return &part, nil
		case "filter":
			log.Printf("[PIIDetector] Filtered streaming part with PII: %s", strings.Join(detectedTypes, ", "))
			return nil, nil
		case "redact":
			if detectionResult.RedactedContent != nil {
				log.Printf("[PIIDetector] Redacted PII in streaming content: %s", strings.Join(detectedTypes, ", "))
				newPayload := map[string]any{"text": *detectionResult.RedactedContent}
				if payload, ok := part.Payload.(map[string]any); ok {
					for k, v := range payload {
						if k != "text" {
							newPayload[k] = v
						}
					}
				}
				result := processors.ChunkType{Type: part.Type, Payload: newPayload}
				return &result, nil
			}
			log.Println("[PIIDetector] No redaction available for streaming part, filtering")
			return nil, nil
		}
	}

	return &part, nil
}

// ProcessOutputResult processes output messages for PII (same logic as input).
func (pd *PIIDetector) ProcessOutputResult(args processors.ProcessOutputResultArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	result, _, _, err := pd.ProcessInput(processors.ProcessInputArgs{
		ProcessorMessageContext: args.ProcessorMessageContext,
		State:                   args.State,
	})
	if err != nil {
		return nil, nil, err
	}
	return result, nil, nil
}

// ProcessOutputStep is not implemented for this processor.
func (pd *PIIDetector) ProcessOutputStep(args processors.ProcessOutputStepArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------

// detectPII detects PII using the internal agent.
func (pd *PIIDetector) detectPII(content string) (*PIIDetectionResult, error) {
	// TODO: Once Agent is ported, use the detection agent with structured output.
	log.Println("[PIIDetector] Detection agent not yet ported, no PII detected")
	return &PIIDetectionResult{
		Categories: nil,
		Detections: nil,
	}, nil
}

// isPIIFlagged checks if PII was detected.
func (pd *PIIDetector) isPIIFlagged(result *PIIDetectionResult) bool {
	if len(result.Detections) > 0 {
		return true
	}

	if len(result.Categories) > 0 {
		maxScore := 0.0
		for _, cat := range result.Categories {
			maxScore = math.Max(maxScore, cat.Score)
		}
		return maxScore >= pd.threshold
	}

	return false
}

// handleDetectedPII handles detected PII based on strategy.
func (pd *PIIDetector) handleDetectedPII(
	message processors.MastraDBMessage,
	result *PIIDetectionResult,
	strategy string,
	abort func(string, *processors.TripWireOptions) error,
) *processors.MastraDBMessage {
	var detectedTypes []string
	for _, cat := range result.Categories {
		if cat.Score >= pd.threshold {
			detectedTypes = append(detectedTypes, cat.Type)
		}
	}

	alertMessage := fmt.Sprintf("PII detected. Types: %s", strings.Join(detectedTypes, ", "))
	if pd.includeDetections && len(result.Detections) > 0 {
		alertMessage += fmt.Sprintf(". Detections: %d items", len(result.Detections))
	}

	switch strategy {
	case "block":
		if abort != nil {
			_ = abort(alertMessage, nil)
		}
		return nil
	case "warn":
		log.Printf("[PIIDetector] %s", alertMessage)
		return nil
	case "filter":
		log.Printf("[PIIDetector] Filtered message: %s", alertMessage)
		return nil
	case "redact":
		if result.RedactedContent != nil {
			log.Printf("[PIIDetector] Redacted PII: %s", alertMessage)
			msg := pd.createRedactedMessage(message, *result.RedactedContent)
			return &msg
		}
		log.Printf("[PIIDetector] No redaction available, filtering: %s", alertMessage)
		return nil
	default:
		return nil
	}
}

// createRedactedMessage creates a redacted message.
func (pd *PIIDetector) createRedactedMessage(originalMessage processors.MastraDBMessage, redactedContent string) processors.MastraDBMessage {
	msg := originalMessage
	msg.Content.Parts = []processors.MessagePart{{Type: "text", Text: redactedContent}}
	msg.Content.Content = redactedContent
	return msg
}

// ApplyRedactionMethod applies the redaction method to content with detections.
func (pd *PIIDetector) ApplyRedactionMethod(content string, detections []PIIDetection) string {
	redacted := content

	// Sort detections by start position in reverse order to maintain indices.
	sorted := make([]PIIDetection, len(detections))
	copy(sorted, detections)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Start > sorted[j].Start
	})

	for _, detection := range sorted {
		redactedValue := pd.RedactValue(detection.Value, detection.Type)
		redacted = redacted[:detection.Start] + redactedValue + redacted[detection.End:]
	}

	return redacted
}

// RedactValue redacts an individual PII value based on method and type.
func (pd *PIIDetector) RedactValue(value, piiType string) string {
	switch pd.redactionMethod {
	case "mask":
		return pd.maskValue(value, piiType)
	case "hash":
		return pd.hashValue(value)
	case "remove":
		return ""
	case "placeholder":
		return fmt.Sprintf("[%s]", strings.ToUpper(piiType))
	default:
		return pd.maskValue(value, piiType)
	}
}

// maskValue masks PII value while optionally preserving format.
func (pd *PIIDetector) maskValue(value, piiType string) string {
	if !pd.preserveFormat {
		maskLen := len(value)
		if maskLen > 8 {
			maskLen = 8
		}
		return strings.Repeat("*", maskLen)
	}

	switch piiType {
	case "email":
		parts := strings.SplitN(value, "@", 2)
		if len(parts) == 2 {
			local := parts[0]
			domain := parts[1]
			var maskedLocal string
			if len(local) > 2 {
				maskedLocal = string(local[0]) + strings.Repeat("*", len(local)-2) + string(local[len(local)-1])
			} else {
				maskedLocal = "***"
			}
			domainParts := strings.SplitN(domain, ".", 2)
			var maskedDomain string
			if len(domainParts) > 1 {
				maskedDomain = strings.Repeat("*", len(domainParts[0])) + "." + domainParts[1]
			} else {
				maskedDomain = "***"
			}
			return maskedLocal + "@" + maskedDomain
		}

	case "phone":
		runes := []rune(value)
		result := make([]rune, len(runes))
		digitIndex := 0
		totalDigits := 0
		for _, r := range runes {
			if r >= '0' && r <= '9' {
				totalDigits++
			}
		}
		for i, r := range runes {
			if r >= '0' && r <= '9' {
				if digitIndex >= totalDigits-4 {
					result[i] = r
				} else {
					result[i] = 'X'
				}
				digitIndex++
			} else {
				result[i] = r
			}
		}
		return string(result)

	case "credit-card", "ssn":
		runes := []rune(value)
		result := make([]rune, len(runes))
		digitIndex := 0
		totalDigits := 0
		for _, r := range runes {
			if r >= '0' && r <= '9' {
				totalDigits++
			}
		}
		for i, r := range runes {
			if r >= '0' && r <= '9' {
				if digitIndex >= totalDigits-4 {
					result[i] = r
				} else {
					result[i] = '*'
				}
				digitIndex++
			} else {
				result[i] = r
			}
		}
		return string(result)

	case "uuid":
		runes := []rune(value)
		result := make([]rune, len(runes))
		for i, r := range runes {
			if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
				result[i] = '*'
			} else {
				result[i] = r
			}
		}
		return string(result)

	case "crypto-wallet":
		if len(value) > 8 {
			return value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
		}
		return strings.Repeat("*", len(value))

	case "iban":
		if len(value) > 6 {
			return value[:2] + strings.Repeat("*", len(value)-6) + value[len(value)-4:]
		}
		return strings.Repeat("*", len(value))
	}

	// Generic masking.
	if len(value) <= 3 {
		return strings.Repeat("*", len(value))
	}
	return string(value[0]) + strings.Repeat("*", len(value)-2) + string(value[len(value)-1])
}

// hashValue hashes PII value using SHA256.
func (pd *PIIDetector) hashValue(value string) string {
	h := sha256.Sum256([]byte(value))
	return fmt.Sprintf("[HASH:%x]", h[:4])
}

// getDetectedTypes returns unique detected PII types from a detection result.
func (pd *PIIDetector) getDetectedTypes(result *PIIDetectionResult) []string {
	if len(result.Detections) > 0 {
		seen := make(map[string]bool)
		var types []string
		for _, d := range result.Detections {
			if !seen[d.Type] {
				seen[d.Type] = true
				types = append(types, d.Type)
			}
		}
		return types
	}

	if len(result.Categories) > 0 {
		var types []string
		for _, cat := range result.Categories {
			if cat.Score >= pd.threshold {
				types = append(types, cat.Type)
			}
		}
		return types
	}

	return nil
}
