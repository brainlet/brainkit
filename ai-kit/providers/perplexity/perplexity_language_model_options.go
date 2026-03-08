// Ported from: packages/perplexity/src/perplexity-language-model-options.ts
package perplexity

// PerplexityLanguageModelID represents Perplexity model identifiers.
// https://docs.perplexity.ai/models/model-cards
//
// Known values:
//   - "sonar-deep-research"
//   - "sonar-reasoning-pro"
//   - "sonar-reasoning"
//   - "sonar-pro"
//   - "sonar"
//
// Any string is accepted to allow new models.
type PerplexityLanguageModelID = string
