# processors/ Fixtures

Tests Mastra processor constructors and key runtime shapes exposed through the
`agent` bundle. The category is marked AI-gated because many processors are
LLM-backed even when a specific fixture only checks constructor/API shape.

## Fixtures

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| agents-md-injector/basic | yes | none | `AgentsMDInjector` construction and `processInputStep` availability |
| batch-parts/basic | yes | none | `BatchPartsProcessor` construction and output stream processing shape |
| language-detector/basic | yes | none | `LanguageDetector` construction with model and input processor shape |
| moderation/basic | yes | none | `ModerationProcessor` construction with model |
| pii-detector/basic | yes | none | `PIIDetector` construction with redact strategy and detection types |
| prompt-injection/basic | yes | none | `PromptInjectionDetector` construction with rewrite strategy |
| response-cache/basic | yes | none | `ResponseCache` construction, `InMemoryServerCache` operations, per-call context helpers, and deterministic key shape |
| skill-search/basic | yes | none | `SkillSearchProcessor` prototype exposes `processInputStep` |
| skills/basic | yes | none | `SkillsProcessor` prototype exposes `processInputStep` |
| structured-output/basic | yes | none | `StructuredOutputProcessor` construction with schema and model |
| system-prompt-scrubber/basic | yes | none | `SystemPromptScrubber` construction and output result processing shape |
| token-limiter/basic | yes | none | `TokenLimiterProcessor` construction and output stream processing shape |
| tool-call-filter/basic | yes | none | `ToolCallFilter` construction and input processing shape |
| tool-search/basic | yes | none | `ToolSearchProcessor` construction with an in-memory tool set |
| unicode-normalizer/basic | yes | none | `UnicodeNormalizer` construction and input processing shape |
| workspace-instructions/basic | yes | none | `WorkspaceInstructionsProcessor` construction and `processInputStep` shape |

## Notes

- These fixtures mostly lock importability and API shape, not full agent
  processor behavior.
- Full processor-in-agent behavior is partly covered by
  `memory/processor/custom` and agent tripwire fixtures.
- Add behavior fixtures only when they prove a real Brainkit runtime risk, such
  as tripwire propagation, stream batching, or request context propagation.
