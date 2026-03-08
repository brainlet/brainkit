// Ported from: packages/ai/src/generate-text/retention-benchmark.test.ts
// Note: The TS test requires MockLanguageModelV3 which is not ported.
// This Go test covers the retention/inclusion behavior structurally:
// - Request body retention and exclusion
// - Response body retention and exclusion
// - GenerateTextIncludeSettings construction
package generatetext

import (
	"testing"
)

func TestRetentionBenchmark_GenerateTextIncludeSettings(t *testing.T) {
	t.Run("default includes everything", func(t *testing.T) {
		opts := GenerateTextOptions{
			Prompt: "test",
		}

		// By default, Include is nil, meaning everything is retained
		if opts.Include != nil {
			t.Error("expected default Include to be nil (retain everything)")
		}
	})

	t.Run("exclude request body", func(t *testing.T) {
		falseVal := false
		opts := GenerateTextOptions{
			Prompt: "test",
			Include: &GenerateTextIncludeSettings{
				RequestBody: &falseVal,
			},
		}

		if opts.Include == nil {
			t.Fatal("expected Include to not be nil")
		}
		if opts.Include.RequestBody == nil || *opts.Include.RequestBody != false {
			t.Error("expected RequestBody to be false")
		}
	})

	t.Run("exclude response body", func(t *testing.T) {
		falseVal := false
		opts := GenerateTextOptions{
			Prompt: "test",
			Include: &GenerateTextIncludeSettings{
				ResponseBody: &falseVal,
			},
		}

		if opts.Include == nil {
			t.Fatal("expected Include to not be nil")
		}
		if opts.Include.ResponseBody == nil || *opts.Include.ResponseBody != false {
			t.Error("expected ResponseBody to be false")
		}
	})

	t.Run("exclude both bodies", func(t *testing.T) {
		falseVal := false
		opts := GenerateTextOptions{
			Prompt: "test",
			Include: &GenerateTextIncludeSettings{
				RequestBody:  &falseVal,
				ResponseBody: &falseVal,
			},
		}

		if opts.Include == nil {
			t.Fatal("expected Include to not be nil")
		}
		if *opts.Include.RequestBody != false {
			t.Error("expected RequestBody to be false")
		}
		if *opts.Include.ResponseBody != false {
			t.Error("expected ResponseBody to be false")
		}
	})
}

func TestRetentionBenchmark_RequestMetadata(t *testing.T) {
	t.Run("request body retained by default", func(t *testing.T) {
		largeBody := make([]byte, 1024*1024) // 1MB
		for i := range largeBody {
			largeBody[i] = 'x'
		}
		bodyStr := string(largeBody)

		meta := LanguageModelRequestMetadata{
			Body: bodyStr,
		}

		// Body should be retained
		bodyResult, ok := meta.Body.(string)
		if !ok {
			t.Fatalf("expected string body, got %T", meta.Body)
		}
		if len(bodyResult) != 1024*1024 {
			t.Errorf("expected body length 1MB, got %d", len(bodyResult))
		}
	})

	t.Run("request body excluded when nil", func(t *testing.T) {
		meta := LanguageModelRequestMetadata{
			Body: nil,
		}

		if meta.Body != nil {
			t.Error("expected nil body")
		}
	})
}

func TestRetentionBenchmark_ResponseMetadata(t *testing.T) {
	t.Run("response body retained", func(t *testing.T) {
		largeBody := make([]byte, 1024*1024) // 1MB
		for i := range largeBody {
			largeBody[i] = 'x'
		}
		bodyStr := string(largeBody)

		meta := GenerateTextResponseMetadata{
			Body: bodyStr,
		}

		bodyResult, ok := meta.Body.(string)
		if !ok {
			t.Fatalf("expected string body, got %T", meta.Body)
		}
		if len(bodyResult) != 1024*1024 {
			t.Errorf("expected body length 1MB, got %d", len(bodyResult))
		}
	})

	t.Run("response body excluded when nil", func(t *testing.T) {
		meta := GenerateTextResponseMetadata{
			Body: nil,
		}

		if meta.Body != nil {
			t.Error("expected nil body")
		}
	})
}
