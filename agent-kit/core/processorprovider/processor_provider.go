// Ported from: packages/core/src/processor-provider/providers/index.ts
//
// This file defines the built-in processor provider registry and individual
// provider instances. In the TS source, each provider wraps a concrete
// processor class (UnicodeNormalizer, TokenLimiterProcessor, etc.) with
// a Zod config schema and available phases.
//
// The concrete processor implementations live in the processors/processors/
// sub-packages which are not yet ported. Each provider below is defined as a
// struct with stubbed CreateProcessor methods that panic until the underlying
// processor types are available.
package processorprovider

import (
	"github.com/brainlet/brainkit/agent-kit/core/processors"
	concreteprocessors "github.com/brainlet/brainkit/agent-kit/core/processors/processors"
)

// ---------------------------------------------------------------------------
// builtInProvider is a simple struct-based ProcessorProvider implementation.
// ---------------------------------------------------------------------------

type builtInProvider struct {
	info            ProcessorProviderInfo
	configSchema    map[string]any
	availablePhases []ProcessorPhase
	factory         func(config map[string]any) processors.Processor
}

func (p *builtInProvider) Info() ProcessorProviderInfo     { return p.info }
func (p *builtInProvider) ConfigSchema() map[string]any    { return p.configSchema }
func (p *builtInProvider) AvailablePhases() []ProcessorPhase { return p.availablePhases }
func (p *builtInProvider) CreateProcessor(config map[string]any) processors.Processor {
	return p.factory(config)
}

// ---------------------------------------------------------------------------
// 1. unicode-normalizer
// ---------------------------------------------------------------------------

// UnicodeNormalizerProvider supplies a UnicodeNormalizer processor.
var UnicodeNormalizerProvider ProcessorProvider = &builtInProvider{
	info: ProcessorProviderInfo{
		ID:          "unicode-normalizer",
		Name:        "Unicode Normalizer",
		Description: "Normalizes Unicode text by stripping control characters, collapsing whitespace, and trimming.",
	},
	configSchema: map[string]any{
		"stripControlChars":  "boolean, optional",
		"preserveEmojis":     "boolean, optional",
		"collapseWhitespace": "boolean, optional",
		"trim":               "boolean, optional",
	},
	availablePhases: []ProcessorPhase{PhaseProcessInput},
	factory: func(config map[string]any) processors.Processor {
		opts := &concreteprocessors.UnicodeNormalizerOptions{}
		if v, ok := config["stripControlChars"].(bool); ok {
			opts.StripControlChars = v
		}
		if v, ok := config["preserveEmojis"].(bool); ok {
			opts.PreserveEmojis = v
		}
		if v, ok := config["collapseWhitespace"].(bool); ok {
			opts.CollapseWhitespace = v
		}
		if v, ok := config["trim"].(bool); ok {
			opts.Trim = v
		}
		return concreteprocessors.NewUnicodeNormalizer(opts)
	},
}

// ---------------------------------------------------------------------------
// 2. token-limiter
// ---------------------------------------------------------------------------

// TokenLimiterProvider supplies a TokenLimiterProcessor.
var TokenLimiterProvider ProcessorProvider = &builtInProvider{
	info: ProcessorProviderInfo{
		ID:          "token-limiter",
		Name:        "Token Limiter",
		Description: "Limits the number of tokens in messages, supporting both input filtering and output truncation.",
	},
	configSchema: map[string]any{
		"limit":     "number, required",
		"strategy":  "enum(truncate|abort), optional",
		"countMode": "enum(cumulative|part), optional",
	},
	availablePhases: []ProcessorPhase{PhaseProcessInput, PhaseProcessOutputStream, PhaseProcessOutputResult},
	factory: func(config map[string]any) processors.Processor {
		limit := 0
		if v, ok := config["limit"].(float64); ok {
			limit = int(v)
		} else if v, ok := config["limit"].(int); ok {
			limit = v
		}
		opts := &concreteprocessors.TokenLimiterOptions{}
		if v, ok := config["strategy"].(string); ok {
			opts.Strategy = v
		}
		if v, ok := config["countMode"].(string); ok {
			opts.CountMode = v
		}
		return concreteprocessors.NewTokenLimiterProcessor(limit, opts)
	},
}

// ---------------------------------------------------------------------------
// 3. tool-call-filter
// ---------------------------------------------------------------------------

// ToolCallFilterProvider supplies a ToolCallFilter processor.
var ToolCallFilterProvider ProcessorProvider = &builtInProvider{
	info: ProcessorProviderInfo{
		ID:          "tool-call-filter",
		Name:        "Tool Call Filter",
		Description: "Filters out tool calls and results from messages, optionally targeting specific tools.",
	},
	configSchema: map[string]any{
		"exclude": "array(string), optional",
	},
	availablePhases: []ProcessorPhase{PhaseProcessInput},
	factory: func(config map[string]any) processors.Processor {
		var opts *concreteprocessors.ToolCallFilterOptions
		if exclude, ok := config["exclude"].([]any); ok {
			strs := make([]string, 0, len(exclude))
			for _, e := range exclude {
				if s, ok := e.(string); ok {
					strs = append(strs, s)
				}
			}
			opts = &concreteprocessors.ToolCallFilterOptions{Exclude: strs}
		}
		return concreteprocessors.NewToolCallFilter(opts)
	},
}

// ---------------------------------------------------------------------------
// 4. batch-parts
// ---------------------------------------------------------------------------

// BatchPartsProvider supplies a BatchPartsProcessor.
var BatchPartsProvider ProcessorProvider = &builtInProvider{
	info: ProcessorProviderInfo{
		ID:          "batch-parts",
		Name:        "Batch Parts",
		Description: "Batches multiple stream parts together to reduce stream overhead.",
	},
	configSchema: map[string]any{
		"batchSize":    "number, optional",
		"maxWaitTime":  "number, optional",
		"emitOnNonText": "boolean, optional",
	},
	availablePhases: []ProcessorPhase{PhaseProcessOutputStream},
	factory: func(config map[string]any) processors.Processor {
		opts := &concreteprocessors.BatchPartsOptions{}
		if v, ok := config["batchSize"].(float64); ok {
			opts.BatchSize = int(v)
		} else if v, ok := config["batchSize"].(int); ok {
			opts.BatchSize = v
		}
		if v, ok := config["maxWaitTime"].(float64); ok {
			opts.MaxWaitTime = int(v)
		} else if v, ok := config["maxWaitTime"].(int); ok {
			opts.MaxWaitTime = v
		}
		if v, ok := config["emitOnNonText"].(bool); ok {
			opts.EmitOnNonText = v
		}
		return concreteprocessors.NewBatchPartsProcessor(opts)
	},
}

// ---------------------------------------------------------------------------
// 5. moderation
// ---------------------------------------------------------------------------

// ModerationProvider supplies a ModerationProcessor.
var ModerationProvider ProcessorProvider = &builtInProvider{
	info: ProcessorProviderInfo{
		ID:          "moderation",
		Name:        "Moderation",
		Description: "Evaluates content against configurable moderation categories for content safety.",
	},
	configSchema: map[string]any{
		"model":                  "string, required",
		"categories":             "array(string), optional",
		"threshold":              "number, optional",
		"strategy":               "enum(block|warn|filter), optional",
		"instructions":           "string, optional",
		"includeScores":          "boolean, optional",
		"chunkWindow":            "number, optional",
		"structuredOutputOptions": "object{jsonPromptInjection: boolean}, optional",
		"providerOptions":        "record(string, any), optional",
	},
	availablePhases: []ProcessorPhase{PhaseProcessInput, PhaseProcessOutputResult, PhaseProcessOutputStream},
	factory: func(config map[string]any) processors.Processor {
		opts := concreteprocessors.ModerationOptions{
			Model: config["model"],
		}
		if v, ok := config["categories"].([]any); ok {
			opts.Categories = toStringSlice(v)
		}
		if v, ok := config["threshold"].(float64); ok {
			opts.Threshold = v
		}
		if v, ok := config["strategy"].(string); ok {
			opts.Strategy = v
		}
		if v, ok := config["instructions"].(string); ok {
			opts.Instructions = v
		}
		if v, ok := config["includeScores"].(bool); ok {
			opts.IncludeScores = v
		}
		if v, ok := config["chunkWindow"].(float64); ok {
			opts.ChunkWindow = int(v)
		}
		if v, ok := config["providerOptions"].(map[string]any); ok {
			opts.ProviderOptions = v
		}
		return concreteprocessors.NewModerationProcessor(opts)
	},
}

// ---------------------------------------------------------------------------
// 6. prompt-injection-detector
// ---------------------------------------------------------------------------

// PromptInjectionDetectorProvider supplies a PromptInjectionDetector processor.
var PromptInjectionDetectorProvider ProcessorProvider = &builtInProvider{
	info: ProcessorProviderInfo{
		ID:          "prompt-injection-detector",
		Name:        "Prompt Injection Detector",
		Description: "Identifies and handles prompt injection attacks, jailbreaks, and data exfiltration attempts.",
	},
	configSchema: map[string]any{
		"model":                  "string, required",
		"detectionTypes":         "array(string), optional",
		"threshold":              "number, optional",
		"strategy":               "enum(block|warn|filter|rewrite), optional",
		"instructions":           "string, optional",
		"includeScores":          "boolean, optional",
		"structuredOutputOptions": "object{jsonPromptInjection: boolean}, optional",
		"providerOptions":        "record(string, any), optional",
	},
	availablePhases: []ProcessorPhase{PhaseProcessInput},
	factory: func(config map[string]any) processors.Processor {
		opts := concreteprocessors.PromptInjectionOptions{
			Model: config["model"],
		}
		if v, ok := config["detectionTypes"].([]any); ok {
			opts.DetectionTypes = toStringSlice(v)
		}
		if v, ok := config["threshold"].(float64); ok {
			opts.Threshold = v
		}
		if v, ok := config["strategy"].(string); ok {
			opts.Strategy = v
		}
		if v, ok := config["instructions"].(string); ok {
			opts.Instructions = v
		}
		if v, ok := config["includeScores"].(bool); ok {
			opts.IncludeScores = v
		}
		if v, ok := config["providerOptions"].(map[string]any); ok {
			opts.ProviderOptions = v
		}
		return concreteprocessors.NewPromptInjectionDetector(opts)
	},
}

// ---------------------------------------------------------------------------
// 7. pii-detector
// ---------------------------------------------------------------------------

// PIIDetectorProvider supplies a PIIDetector processor.
var PIIDetectorProvider ProcessorProvider = &builtInProvider{
	info: ProcessorProviderInfo{
		ID:          "pii-detector",
		Name:        "PII Detector",
		Description: "Identifies and redacts personally identifiable information for privacy compliance.",
	},
	configSchema: map[string]any{
		"model":                  "string, required",
		"detectionTypes":         "array(string), optional",
		"threshold":              "number, optional",
		"strategy":               "enum(block|warn|filter|redact), optional",
		"redactionMethod":        "enum(mask|hash|remove|placeholder), optional",
		"instructions":           "string, optional",
		"includeDetections":      "boolean, optional",
		"preserveFormat":         "boolean, optional",
		"structuredOutputOptions": "object{jsonPromptInjection: boolean}, optional",
		"providerOptions":        "record(string, any), optional",
	},
	availablePhases: []ProcessorPhase{PhaseProcessInput},
	factory: func(config map[string]any) processors.Processor {
		opts := concreteprocessors.PIIDetectorOptions{
			Model: config["model"],
		}
		if v, ok := config["detectionTypes"].([]any); ok {
			opts.DetectionTypes = toStringSlice(v)
		}
		if v, ok := config["threshold"].(float64); ok {
			opts.Threshold = v
		}
		if v, ok := config["strategy"].(string); ok {
			opts.Strategy = v
		}
		if v, ok := config["redactionMethod"].(string); ok {
			opts.RedactionMethod = v
		}
		if v, ok := config["instructions"].(string); ok {
			opts.Instructions = v
		}
		if v, ok := config["includeDetections"].(bool); ok {
			opts.IncludeDetections = v
		}
		if v, ok := config["preserveFormat"].(bool); ok {
			b := v
			opts.PreserveFormat = &b
		}
		if v, ok := config["providerOptions"].(map[string]any); ok {
			opts.ProviderOptions = v
		}
		return concreteprocessors.NewPIIDetector(opts)
	},
}

// ---------------------------------------------------------------------------
// 8. language-detector
// ---------------------------------------------------------------------------

// LanguageDetectorProvider supplies a LanguageDetector processor.
var LanguageDetectorProvider ProcessorProvider = &builtInProvider{
	info: ProcessorProviderInfo{
		ID:          "language-detector",
		Name:        "Language Detector",
		Description: "Detects the language of input text and optionally translates it to a target language.",
	},
	configSchema: map[string]any{
		"model":                  "string, required",
		"targetLanguages":        "array(string), required",
		"threshold":              "number, optional",
		"strategy":               "enum(detect|translate|block|warn), optional",
		"preserveOriginal":       "boolean, optional",
		"instructions":           "string, optional",
		"minTextLength":          "number, optional",
		"includeDetectionDetails": "boolean, optional",
		"translationQuality":     "enum(speed|quality|balanced), optional",
		"providerOptions":        "record(string, any), optional",
	},
	availablePhases: []ProcessorPhase{PhaseProcessInput},
	factory: func(config map[string]any) processors.Processor {
		opts := concreteprocessors.LanguageDetectorOptions{
			Model: config["model"],
		}
		if v, ok := config["targetLanguages"].([]any); ok {
			opts.TargetLanguages = toStringSlice(v)
		}
		if v, ok := config["threshold"].(float64); ok {
			opts.Threshold = v
		}
		if v, ok := config["strategy"].(string); ok {
			opts.Strategy = v
		}
		if v, ok := config["preserveOriginal"].(bool); ok {
			b := v
			opts.PreserveOriginal = &b
		}
		if v, ok := config["instructions"].(string); ok {
			opts.Instructions = v
		}
		if v, ok := config["minTextLength"].(float64); ok {
			opts.MinTextLength = int(v)
		}
		if v, ok := config["includeDetectionDetails"].(bool); ok {
			opts.IncludeDetectionDetails = v
		}
		if v, ok := config["translationQuality"].(string); ok {
			opts.TranslationQuality = v
		}
		if v, ok := config["providerOptions"].(map[string]any); ok {
			opts.ProviderOptions = v
		}
		return concreteprocessors.NewLanguageDetector(opts)
	},
}

// ---------------------------------------------------------------------------
// 9. system-prompt-scrubber
// ---------------------------------------------------------------------------

// SystemPromptScrubberProvider supplies a SystemPromptScrubber processor.
var SystemPromptScrubberProvider ProcessorProvider = &builtInProvider{
	info: ProcessorProviderInfo{
		ID:          "system-prompt-scrubber",
		Name:        "System Prompt Scrubber",
		Description: "Detects and removes system prompt leakage from model outputs.",
	},
	configSchema: map[string]any{
		"model":                  "string, required",
		"strategy":               "enum(block|warn|filter|redact), optional",
		"customPatterns":         "array(string), optional",
		"includeDetections":      "boolean, optional",
		"instructions":           "string, optional",
		"redactionMethod":        "enum(mask|placeholder|remove), optional",
		"placeholderText":        "string, optional",
		"structuredOutputOptions": "object{jsonPromptInjection: boolean}, optional",
	},
	availablePhases: []ProcessorPhase{PhaseProcessOutputStream, PhaseProcessOutputResult},
	factory: func(config map[string]any) processors.Processor {
		opts := concreteprocessors.SystemPromptScrubberOptions{
			Model: config["model"],
		}
		if v, ok := config["strategy"].(string); ok {
			opts.Strategy = v
		}
		if v, ok := config["customPatterns"].([]any); ok {
			opts.CustomPatterns = toStringSlice(v)
		}
		if v, ok := config["includeDetections"].(bool); ok {
			opts.IncludeDetections = v
		}
		if v, ok := config["instructions"].(string); ok {
			opts.Instructions = v
		}
		if v, ok := config["redactionMethod"].(string); ok {
			opts.RedactionMethod = v
		}
		if v, ok := config["placeholderText"].(string); ok {
			opts.PlaceholderText = v
		}
		p, err := concreteprocessors.NewSystemPromptScrubber(opts)
		if err != nil {
			panic("processorprovider: SystemPromptScrubber: " + err.Error())
		}
		return p
	},
}

// ---------------------------------------------------------------------------
// Aggregated registry of all built-in providers
// ---------------------------------------------------------------------------

// BuiltInProcessorProviders maps provider IDs to their ProcessorProvider
// implementations. Mirrors the TS BUILT_IN_PROCESSOR_PROVIDERS record.
var BuiltInProcessorProviders = map[string]ProcessorProvider{
	"unicode-normalizer":        UnicodeNormalizerProvider,
	"token-limiter":             TokenLimiterProvider,
	"tool-call-filter":          ToolCallFilterProvider,
	"batch-parts":               BatchPartsProvider,
	"moderation":                ModerationProvider,
	"prompt-injection-detector": PromptInjectionDetectorProvider,
	"pii-detector":              PIIDetectorProvider,
	"language-detector":         LanguageDetectorProvider,
	"system-prompt-scrubber":    SystemPromptScrubberProvider,
}

// toStringSlice converts []any to []string, skipping non-string values.
func toStringSlice(v []any) []string {
	result := make([]string, 0, len(v))
	for _, item := range v {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}
