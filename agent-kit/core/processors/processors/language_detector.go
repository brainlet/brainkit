// Ported from: packages/core/src/processors/processors/language-detector.ts
package concreteprocessors

import (
	"fmt"
	"log"
	"strings"

	processors "github.com/brainlet/brainkit/agent-kit/core/processors"
)

// ---------------------------------------------------------------------------
// Stub types for unported dependencies
// ---------------------------------------------------------------------------

// Agent is a stub for ../../agent.Agent.
// TODO: import from agent package once ported.
type Agent interface {
	Generate(prompt string, opts map[string]any) (*AgentGenerateResult, error)
}

// AgentGenerateResult is a stub for the agent generate result.
// TODO: import from agent package once ported.
type AgentGenerateResult struct {
	Object any `json:"object"`
}

// MastraModelConfig is a stub for ../../llm/model/shared.types.MastraModelConfig.
// TODO: import from llm package once ported.
type MastraModelConfig = any

// ProviderOptions is a stub for ../../llm/model/provider-options.ProviderOptions.
// TODO: import from llm package once ported.
type ProviderOptions = map[string]any

// TripWireError is a sentinel error type for trip wire aborts.
// TODO: import from agent package once ported.
type TripWireError struct {
	Message string
}

func (e *TripWireError) Error() string { return e.Message }

// ---------------------------------------------------------------------------
// LanguageDetection
// ---------------------------------------------------------------------------

// LanguageDetection holds a single language detection result.
type LanguageDetection struct {
	Language   string  `json:"language"`
	Confidence float64 `json:"confidence"`
	ISOCode    string  `json:"iso_code"`
}

// TranslationResult holds the result of a translation.
type TranslationResult struct {
	OriginalText     string  `json:"original_text"`
	OriginalLanguage string  `json:"original_language"`
	TranslatedText   string  `json:"translated_text"`
	TargetLanguage   string  `json:"target_language"`
	Confidence       float64 `json:"confidence"`
}

// LanguageDetectionResult holds language detection and optional translation result.
type LanguageDetectionResult struct {
	ISOCode        *string `json:"iso_code"`
	Confidence     *float64 `json:"confidence"`
	TranslatedText *string `json:"translated_text,omitempty"`
}

// ---------------------------------------------------------------------------
// LanguageDetectorOptions
// ---------------------------------------------------------------------------

// LanguageDetectorOptions configures the LanguageDetector processor.
type LanguageDetectorOptions struct {
	// Model configuration for the detection/translation agent.
	Model MastraModelConfig

	// TargetLanguages for the project (language name or ISO code).
	TargetLanguages []string

	// Threshold is the confidence threshold for language detection (0-1). Default: 0.7.
	Threshold float64

	// Strategy when non-target language is detected: "detect", "translate", "block", "warn".
	// Default: "detect".
	Strategy string

	// PreserveOriginal preserves original content in message metadata. Default: true.
	PreserveOriginal *bool

	// Instructions is custom detection instructions for the agent.
	Instructions string

	// MinTextLength is the minimum text length to perform detection. Default: 10.
	MinTextLength int

	// IncludeDetectionDetails includes detailed detection info in logs. Default: false.
	IncludeDetectionDetails bool

	// TranslationQuality preference: "speed", "quality", "balanced". Default: "quality".
	TranslationQuality string

	// ProviderOptions are provider-specific options passed to the internal detection agent.
	ProviderOptions ProviderOptions
}

// ---------------------------------------------------------------------------
// LanguageDetector
// ---------------------------------------------------------------------------

// LanguageDetector identifies the language of input text and optionally
// translates it to a target language for consistent processing.
type LanguageDetector struct {
	processors.BaseProcessor
	detectionAgent          Agent
	targetLanguages         []string
	threshold               float64
	strategy                string
	preserveOriginal        bool
	minTextLength           int
	includeDetectionDetails bool
	translationQuality      string
	providerOptions         ProviderOptions
}

// defaultTargetLanguages is the default set of target languages.
var defaultTargetLanguages = []string{"English", "en"}

// languageMap maps ISO codes to language names.
var languageMap = map[string]string{
	"en":    "English",
	"es":    "Spanish",
	"fr":    "French",
	"de":    "German",
	"it":    "Italian",
	"pt":    "Portuguese",
	"ru":    "Russian",
	"ja":    "Japanese",
	"ko":    "Korean",
	"zh":    "Chinese",
	"zh-cn": "Chinese (Simplified)",
	"zh-tw": "Chinese (Traditional)",
	"ar":    "Arabic",
	"hi":    "Hindi",
	"th":    "Thai",
	"vi":    "Vietnamese",
	"tr":    "Turkish",
	"pl":    "Polish",
	"nl":    "Dutch",
	"sv":    "Swedish",
	"da":    "Danish",
	"no":    "Norwegian",
	"fi":    "Finnish",
	"el":    "Greek",
	"he":    "Hebrew",
	"cs":    "Czech",
	"hu":    "Hungarian",
	"ro":    "Romanian",
	"bg":    "Bulgarian",
	"hr":    "Croatian",
	"sk":    "Slovak",
	"sl":    "Slovenian",
	"et":    "Estonian",
	"lv":    "Latvian",
	"lt":    "Lithuanian",
	"uk":    "Ukrainian",
	"be":    "Belarusian",
}

// NewLanguageDetector creates a new LanguageDetector processor.
func NewLanguageDetector(opts LanguageDetectorOptions) *LanguageDetector {
	targetLangs := opts.TargetLanguages
	if len(targetLangs) == 0 {
		targetLangs = defaultTargetLanguages
	}

	threshold := opts.Threshold
	if threshold == 0 {
		threshold = 0.7
	}

	strategy := opts.Strategy
	if strategy == "" {
		strategy = "detect"
	}

	preserveOriginal := true
	if opts.PreserveOriginal != nil {
		preserveOriginal = *opts.PreserveOriginal
	}

	minTextLength := opts.MinTextLength
	if minTextLength <= 0 {
		minTextLength = 10
	}

	translationQuality := opts.TranslationQuality
	if translationQuality == "" {
		translationQuality = "quality"
	}

	// TODO: Create actual detection agent once Agent is ported.
	// For now, detectionAgent is nil.

	return &LanguageDetector{
		BaseProcessor:           processors.NewBaseProcessor("language-detector", "Language Detector"),
		detectionAgent:          nil,
		targetLanguages:         targetLangs,
		threshold:               threshold,
		strategy:                strategy,
		preserveOriginal:        preserveOriginal,
		minTextLength:           minTextLength,
		includeDetectionDetails: opts.IncludeDetectionDetails,
		translationQuality:      translationQuality,
		providerOptions:         opts.ProviderOptions,
	}
}

// ProcessInput detects languages and optionally translates messages.
func (ld *LanguageDetector) ProcessInput(args processors.ProcessInputArgs) (
	[]processors.MastraDBMessage, *processors.MessageList, *processors.ProcessInputResultWithSystemMessages, error,
) {
	messages := args.Messages

	if len(messages) == 0 {
		return messages, nil, nil, nil
	}

	var processedMessages []processors.MastraDBMessage

	for _, message := range messages {
		textContent := extractTextContentFromMessage(message)
		if len(textContent) < ld.minTextLength {
			processedMessages = append(processedMessages, message)
			continue
		}

		detectionResult, err := ld.detectLanguage(textContent)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("language detection failed: %w", err)
		}

		// Check confidence threshold.
		if detectionResult.Confidence != nil && *detectionResult.Confidence < ld.threshold {
			processedMessages = append(processedMessages, message)
			continue
		}

		if !ld.isNonTargetLanguage(detectionResult) {
			targetLangCode := ld.getLanguageCode(ld.targetLanguages[0])
			confidence := 0.95
			targetMsg := ld.addLanguageMetadata(message, &LanguageDetectionResult{
				ISOCode:    &targetLangCode,
				Confidence: &confidence,
			}, nil)

			if ld.includeDetectionDetails {
				log.Printf("[LanguageDetector] Content in target language: Language detected: %s (%s) with confidence 0.95",
					ld.getLanguageName(targetLangCode), targetLangCode)
			}

			processedMessages = append(processedMessages, targetMsg)
			continue
		}

		processedMsg, err := ld.handleDetectedLanguage(message, detectionResult, args.Abort)
		if err != nil {
			return nil, nil, nil, err
		}
		if processedMsg != nil {
			processedMessages = append(processedMessages, *processedMsg)
		}
	}

	return processedMessages, nil, nil, nil
}

// ProcessInputStep is not implemented for this processor.
func (ld *LanguageDetector) ProcessInputStep(args processors.ProcessInputStepArgs) (*processors.ProcessInputStepResult, []processors.MastraDBMessage, error) {
	return nil, nil, nil
}

// ProcessOutputStream is not implemented for this processor.
func (ld *LanguageDetector) ProcessOutputStream(args processors.ProcessOutputStreamArgs) (*processors.ChunkType, error) {
	return &args.Part, nil
}

// ProcessOutputResult is not implemented for this processor.
func (ld *LanguageDetector) ProcessOutputResult(args processors.ProcessOutputResultArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}

// ProcessOutputStep is not implemented for this processor.
func (ld *LanguageDetector) ProcessOutputStep(args processors.ProcessOutputStepArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------

// detectLanguage detects the language of content using the internal agent.
func (ld *LanguageDetector) detectLanguage(content string) (*LanguageDetectionResult, error) {
	// TODO: Once Agent is ported, use the detection agent with structured output.
	// For now, return a nil result (assume target language).
	log.Println("[LanguageDetector] Detection agent not yet ported, assuming target language")
	return &LanguageDetectionResult{
		ISOCode:    nil,
		Confidence: nil,
	}, nil
}

// isNonTargetLanguage determines if the detection result indicates a non-target language.
func (ld *LanguageDetector) isNonTargetLanguage(result *LanguageDetectionResult) bool {
	if result.ISOCode != nil && result.Confidence != nil && *result.Confidence >= ld.threshold {
		return !ld.isTargetLanguage(result.ISOCode)
	}
	return false
}

// getLanguageName returns the language name for an ISO code.
func (ld *LanguageDetector) getLanguageName(isoCode string) string {
	if name, ok := languageMap[strings.ToLower(isoCode)]; ok {
		return name
	}
	return isoCode
}

// handleDetectedLanguage handles a detected non-target language based on strategy.
func (ld *LanguageDetector) handleDetectedLanguage(
	message processors.MastraDBMessage,
	result *LanguageDetectionResult,
	abort func(string, *processors.TripWireOptions) error,
) (*processors.MastraDBMessage, error) {
	detectedLanguage := "Unknown"
	if result.ISOCode != nil {
		detectedLanguage = ld.getLanguageName(*result.ISOCode)
	}

	isoCode := ""
	if result.ISOCode != nil {
		isoCode = *result.ISOCode
	}

	confStr := "N/A"
	if result.Confidence != nil {
		confStr = fmt.Sprintf("%.2f", *result.Confidence)
	}

	alertMessage := fmt.Sprintf("Language detected: %s (%s) with confidence %s", detectedLanguage, isoCode, confStr)

	switch ld.strategy {
	case "detect":
		log.Printf("[LanguageDetector] %s", alertMessage)
		msg := ld.addLanguageMetadata(message, result, nil)
		return &msg, nil

	case "warn":
		log.Printf("[LanguageDetector] Non-target language: %s", alertMessage)
		msg := ld.addLanguageMetadata(message, result, nil)
		return &msg, nil

	case "block":
		blockMessage := fmt.Sprintf("Non-target language detected: %s", alertMessage)
		log.Printf("[LanguageDetector] Blocking: %s", blockMessage)
		if abort != nil {
			return nil, abort(blockMessage, nil)
		}
		return nil, fmt.Errorf("%s", blockMessage)

	case "translate":
		if result.TranslatedText != nil {
			log.Printf("[LanguageDetector] Translated from %s: %s", detectedLanguage, alertMessage)
			msg := ld.createTranslatedMessage(message, result)
			return &msg, nil
		}
		log.Printf("[LanguageDetector] No translation available, keeping original: %s", alertMessage)
		msg := ld.addLanguageMetadata(message, result, nil)
		return &msg, nil

	default:
		msg := ld.addLanguageMetadata(message, result, nil)
		return &msg, nil
	}
}

// createTranslatedMessage creates a translated message with original preserved in metadata.
func (ld *LanguageDetector) createTranslatedMessage(originalMessage processors.MastraDBMessage, result *LanguageDetectionResult) processors.MastraDBMessage {
	if result.TranslatedText == nil {
		return ld.addLanguageMetadata(originalMessage, result, nil)
	}

	translatedMsg := originalMessage
	translatedMsg.Content.Parts = []processors.MessagePart{{Type: "text", Text: *result.TranslatedText}}
	translatedMsg.Content.Content = *result.TranslatedText

	return ld.addLanguageMetadata(translatedMsg, result, &originalMessage)
}

// addLanguageMetadata adds language detection metadata to a message.
func (ld *LanguageDetector) addLanguageMetadata(
	message processors.MastraDBMessage,
	result *LanguageDetectionResult,
	originalMessage *processors.MastraDBMessage,
) processors.MastraDBMessage {
	isTarget := ld.isTargetLanguage(result.ISOCode)

	detection := map[string]any{
		"is_target_language": isTarget,
		"target_languages":  ld.targetLanguages,
	}

	if result.ISOCode != nil {
		detection["detected_language"] = ld.getLanguageName(*result.ISOCode)
		detection["iso_code"] = *result.ISOCode
	}
	if result.Confidence != nil {
		detection["confidence"] = *result.Confidence
	}
	if result.TranslatedText != nil {
		translation := map[string]any{
			"target_language": ld.targetLanguages[0],
		}
		if result.ISOCode != nil {
			translation["original_language"] = ld.getLanguageName(*result.ISOCode)
		} else {
			translation["original_language"] = "Unknown"
		}
		if result.Confidence != nil {
			translation["translation_confidence"] = *result.Confidence
		}
		detection["translation"] = translation
	}
	if ld.preserveOriginal && originalMessage != nil {
		detection["original_content"] = extractTextContentFromMessage(*originalMessage)
	}

	msg := message
	if msg.Content.Metadata == nil {
		msg.Content.Metadata = make(map[string]any)
	}
	msg.Content.Metadata["language_detection"] = detection
	return msg
}

// isTargetLanguage checks if a language ISO code matches any target language.
func (ld *LanguageDetector) isTargetLanguage(isoCode *string) bool {
	if isoCode == nil {
		return true // Assume target if no detection
	}

	for _, target := range ld.targetLanguages {
		targetCode := ld.getLanguageCode(target)
		if targetCode == strings.ToLower(*isoCode) {
			return true
		}
		if strings.EqualFold(target, ld.getLanguageName(*isoCode)) {
			return true
		}
	}
	return false
}

// getLanguageCode returns the ISO code for a language name.
func (ld *LanguageDetector) getLanguageCode(language string) string {
	lowerLang := strings.ToLower(language)

	if _, ok := languageMap[lowerLang]; ok {
		return lowerLang
	}

	for code, name := range languageMap {
		if strings.EqualFold(name, language) {
			return code
		}
	}

	if len(lowerLang) <= 3 {
		return lowerLang
	}
	return "unknown"
}

// extractTextContentFromMessage extracts text content from a message.
func extractTextContentFromMessage(message processors.MastraDBMessage) string {
	var text string

	for _, part := range message.Content.Parts {
		if part.Type == "text" && part.Text != "" {
			text += part.Text + " "
		}
	}

	if strings.TrimSpace(text) == "" && message.Content.Content != "" {
		text = message.Content.Content
	}

	return strings.TrimSpace(text)
}
