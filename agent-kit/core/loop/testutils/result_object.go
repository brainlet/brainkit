// Ported from: packages/core/src/loop/test-utils/resultObject.ts
package testutils

// ---------------------------------------------------------------------------
// ResultObjectTestsConfig
// ---------------------------------------------------------------------------

// ResultObjectTestsConfig configures the resultObjectTests test suite.
type ResultObjectTestsConfig struct {
	LoopFn       LoopFn
	RunID        string
	ModelVersion string // "v2" or "v3"
}

// ResultObjectTests contains the test definitions for result object properties.
// In the TS source, this is a vitest describe block that validates:
//   - result.warnings (resolve with warnings)
//   - result.usage (resolve with token usage)
//   - result.finishReason (resolve with finish reason)
//   - result.text (resolve with generated text)
//   - result.reasoning (resolve with reasoning content)
//   - result.sources (resolve with source attributions)
//   - result.files (resolve with generated files)
//   - result.response (resolve with response metadata)
//   - result.response.headers (resolve with response headers)
//   - result.request (resolve with request metadata)
//   - result.providerMetadata
//
// Tests are parameterized by model version (v2/v3) to verify both
// specification versions produce the same result shape.
type ResultObjectTests struct {
	Config ResultObjectTestsConfig
}

// NewResultObjectTests creates a new ResultObjectTests instance.
func NewResultObjectTests(config ResultObjectTestsConfig) *ResultObjectTests {
	return &ResultObjectTests{Config: config}
}

// ---------------------------------------------------------------------------
// Result object test helpers
// ---------------------------------------------------------------------------

// CreateResultObjectModels returns the appropriate test models for the given
// model version. Uses CreateTestModels for v2 and CreateTestModelsV3 for v3.
func CreateResultObjectModels(modelVersion string, opts ...any) []ModelManagerModelConfig {
	if modelVersion == "v3" {
		return CreateTestModelsV3()
	}
	return CreateTestModels()
}

// GetModelWithSourcesForVersion returns the sources mock model for the given version.
func GetModelWithSourcesForVersion(modelVersion string) any {
	if modelVersion == "v3" {
		return ModelWithSourcesV3
	}
	return ModelWithSources
}

// GetModelWithFilesForVersion returns the files mock model for the given version.
func GetModelWithFilesForVersion(modelVersion string) any {
	if modelVersion == "v3" {
		return ModelWithFilesV3
	}
	return ModelWithFiles
}

// GetModelWithReasoningForVersion returns the reasoning mock model for the given version.
func GetModelWithReasoningForVersion(modelVersion string) any {
	if modelVersion == "v3" {
		return ModelWithReasoningV3
	}
	return ModelWithReasoning
}

// GetTestUsageForVersion returns the test usage for the given version.
func GetTestUsageForVersion(modelVersion string) any {
	if modelVersion == "v3" {
		return TestUsageV3
	}
	return TestUsage
}
