// Ported from: packages/core/src/processors/processors/index.ts
package concreteprocessors

// This package will contain the concrete processor implementations.
//
// TODO: Port the following from packages/core/src/processors/processors/:
//
// 1. UnicodeNormalizer (unicode-normalizer.ts)
//    Options: UnicodeNormalizerOptions
//    Purpose: Normalizes Unicode text by stripping control chars, collapsing whitespace, trimming.
//
// 2. ModerationProcessor (moderation.ts)
//    Options: ModerationOptions
//    Types: ModerationResult, ModerationCategoryScores
//    Purpose: Evaluates content against configurable moderation categories.
//
// 3. PromptInjectionDetector (prompt-injection-detector.ts)
//    Options: PromptInjectionOptions
//    Types: PromptInjectionResult, PromptInjectionCategoryScores
//    Purpose: Identifies and handles prompt injection attacks, jailbreaks.
//
// 4. PIIDetector (pii-detector.ts)
//    Options: PIIDetectorOptions
//    Types: PIIDetectionResult, PIICategories, PIICategoryScores, PIIDetection
//    Purpose: Identifies and redacts personally identifiable information.
//
// 5. LanguageDetector (language-detector.ts)
//    Options: LanguageDetectorOptions
//    Types: LanguageDetectionResult, LanguageDetection, TranslationResult
//    Purpose: Detects language and optionally translates to a target language.
//
// 6. StructuredOutputProcessor (structured-output.ts)
//    Options: StructuredOutputOptions
//    Purpose: Handles structured output schema validation and formatting.
//
// 7. BatchPartsProcessor (batch-parts.ts)
//    Options: BatchPartsOptions
//    Types: BatchPartsState
//    Purpose: Batches multiple stream parts together to reduce stream overhead.
//
// 8. TokenLimiterProcessor (token-limiter.ts)
//    Options: TokenLimiterOptions
//    Purpose: Limits the number of tokens in messages (truncate or abort strategy).
//
// 9. SystemPromptScrubber (system-prompt-scrubber.ts)
//    Options: SystemPromptScrubberOptions
//    Types: SystemPromptDetectionResult, SystemPromptDetection
//    Purpose: Detects and removes system prompt leakage from model outputs.
//
// 10. ToolCallFilter (tool-call-filter.ts)
//     Purpose: Filters out tool calls and results from messages.
//
// 11. ToolSearchProcessor (tool-search.ts)
//     Options: ToolSearchProcessorOptions
//     Purpose: Searches for and dynamically selects tools based on user query.
//
// 12. SkillsProcessor (skills.ts)
//     Options: SkillsProcessorOptions
//     Purpose: Injects skill instructions into the conversation based on context.
//
// 13. WorkspaceInstructionsProcessor (workspace-instructions.ts)
//     Options: WorkspaceInstructionsProcessorOptions
//     Purpose: Injects workspace-level instructions into the conversation.
//
// 14. PrepareStep (prepare-step.ts)
//     Purpose: Prepares step data for the agentic loop.
