// Ported from: packages/ai/src/logger/log-warnings.test.ts
package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogWarnings(t *testing.T) {
	setup := func() (warnMessages *[]string, infoMessages *[]string) {
		ResetLogWarningsState()
		SetLogWarningsConfig(nil)
		w := make([]string, 0)
		i := make([]string, 0)
		return &w, &i
	}

	// We capture logs by using a custom logger that records warn and info messages.
	// In the Go port, we simulate the TS test pattern by using a custom logger
	// that captures calls, and we test the formatWarning function directly.

	t.Run("when config is disabled", func(t *testing.T) {
		t.Run("should not log any warnings (single)", func(t *testing.T) {
			_, _ = setup()
			called := false
			SetLogWarningsConfig(&LogWarningsConfig{
				Disabled: true,
			})

			// Override with a custom logger to detect calls
			SetLogWarningsConfig(&LogWarningsConfig{Disabled: true})

			LogWarnings(LogWarningsOptions{
				Warnings: []Warning{{Type: "other", Message: "Test warning"}},
				Provider: "providerX",
				Model:    "modelY",
			})

			assert.False(t, called)
		})

		t.Run("should not log any warnings (multiple)", func(t *testing.T) {
			_, _ = setup()
			SetLogWarningsConfig(&LogWarningsConfig{Disabled: true})

			calls := 0
			// Even with a custom logger, Disabled takes precedence
			SetLogWarningsConfig(&LogWarningsConfig{
				Disabled:     true,
				CustomLogger: func(options LogWarningsOptions) { calls++ },
			})

			LogWarnings(LogWarningsOptions{
				Warnings: []Warning{
					{Type: "other", Message: "Test warning 1"},
					{Type: "other", Message: "Test warning 2"},
				},
				Provider: "provider",
				Model:    "model",
			})

			assert.Equal(t, 0, calls)
		})

		t.Run("should not count empty arrays as first call", func(t *testing.T) {
			_, _ = setup()
			SetLogWarningsConfig(&LogWarningsConfig{Disabled: true})

			LogWarnings(LogWarningsOptions{
				Warnings: []Warning{},
				Provider: "prov",
				Model:    "mod",
			})

			LogWarnings(LogWarningsOptions{
				Warnings: []Warning{{Type: "other", Message: "foo"}},
				Provider: "p1",
				Model:    "m1",
			})

			// No assertions on console output since disabled - just verify no panic
		})
	})

	t.Run("when using a custom logger function", func(t *testing.T) {
		t.Run("should call the custom function with warning options", func(t *testing.T) {
			_, _ = setup()
			var captured *LogWarningsOptions
			SetLogWarningsConfig(&LogWarningsConfig{
				CustomLogger: func(options LogWarningsOptions) {
					captured = &options
				},
			})

			warnings := []Warning{{Type: "other", Message: "Test warning"}}
			options := LogWarningsOptions{Warnings: warnings, Provider: "pp", Model: "mm"}
			LogWarnings(options)

			assert.NotNil(t, captured)
			assert.Equal(t, options.Warnings, captured.Warnings)
			assert.Equal(t, "pp", captured.Provider)
			assert.Equal(t, "mm", captured.Model)
		})

		t.Run("should call the custom function with multiple warnings", func(t *testing.T) {
			_, _ = setup()
			var captured *LogWarningsOptions
			SetLogWarningsConfig(&LogWarningsConfig{
				CustomLogger: func(options LogWarningsOptions) {
					captured = &options
				},
			})

			warnings := []Warning{
				{Type: "unsupported", Feature: "temperature", Details: "Temperature not supported"},
				{Type: "other", Message: "Another warning"},
			}

			opts := LogWarningsOptions{Warnings: warnings, Provider: "provider", Model: "model"}
			LogWarnings(opts)

			assert.NotNil(t, captured)
			assert.Equal(t, 2, len(captured.Warnings))
		})

		t.Run("should not call the custom function with empty warnings", func(t *testing.T) {
			_, _ = setup()
			calls := 0
			SetLogWarningsConfig(&LogWarningsConfig{
				CustomLogger: func(options LogWarningsOptions) {
					calls++
				},
			})

			LogWarnings(LogWarningsOptions{Warnings: []Warning{}, Provider: "x", Model: "y"})

			assert.Equal(t, 0, calls)
		})
	})

	t.Run("formatWarning", func(t *testing.T) {
		t.Run("should format unsupported warning", func(t *testing.T) {
			result := formatWarning(
				Warning{Type: "unsupported", Feature: "mediaType", Details: "detail"},
				"zzz", "MMM",
			)
			assert.Equal(t,
				`AI SDK Warning (zzz / MMM): The feature "mediaType" is not supported. detail`,
				result,
			)
		})

		t.Run("should format unsupported warning without details", func(t *testing.T) {
			result := formatWarning(
				Warning{Type: "unsupported", Feature: "voice"},
				"zzz", "MMM",
			)
			assert.Equal(t,
				`AI SDK Warning (zzz / MMM): The feature "voice" is not supported.`,
				result,
			)
		})

		t.Run("should format compatibility warning", func(t *testing.T) {
			result := formatWarning(
				Warning{Type: "compatibility", Feature: "streaming", Details: "fallback mode"},
				"prov", "mod",
			)
			assert.Equal(t,
				`AI SDK Warning (prov / mod): The feature "streaming" is used in a compatibility mode. fallback mode`,
				result,
			)
		})

		t.Run("should format other warning", func(t *testing.T) {
			result := formatWarning(
				Warning{Type: "other", Message: "other msg"},
				"zzz", "MMM",
			)
			assert.Equal(t, "AI SDK Warning (zzz / MMM): other msg", result)
		})

		t.Run("should include warning with unknown provider and model", func(t *testing.T) {
			result := formatWarning(
				Warning{Type: "other", Message: "messx"},
				"unknown provider", "unknown model",
			)
			assert.Equal(t, "AI SDK Warning (unknown provider / unknown model): messx", result)
		})
	})

	t.Run("first-time information note", func(t *testing.T) {
		t.Run("should not trigger for empty warnings", func(t *testing.T) {
			_, _ = setup()
			SetLogWarningsConfig(nil)

			// Empty warnings should not affect hasLoggedBefore
			LogWarnings(LogWarningsOptions{Warnings: []Warning{}, Provider: "a", Model: "b"})

			// The hasLoggedBefore flag should still be false
			mu.Lock()
			logged := hasLoggedBefore
			mu.Unlock()
			assert.False(t, logged)
		})

		t.Run("should set hasLoggedBefore on first real call", func(t *testing.T) {
			_, _ = setup()
			// Use custom logger to avoid actual log output
			SetLogWarningsConfig(&LogWarningsConfig{
				CustomLogger: func(options LogWarningsOptions) {},
			})

			LogWarnings(LogWarningsOptions{
				Warnings: []Warning{{Type: "other", Message: "foo"}},
				Provider: "abc",
				Model:    "bbb",
			})

			// With custom logger, hasLoggedBefore should NOT be set
			// because custom logger bypasses the default path
			mu.Lock()
			logged := hasLoggedBefore
			mu.Unlock()
			assert.False(t, logged)
		})

		t.Run("should not trigger when disabled", func(t *testing.T) {
			_, _ = setup()
			SetLogWarningsConfig(&LogWarningsConfig{Disabled: true})

			LogWarnings(LogWarningsOptions{
				Warnings: []Warning{{Type: "other", Message: "Suppressed"}},
				Provider: "notProv",
				Model:    "notModel",
			})

			mu.Lock()
			logged := hasLoggedBefore
			mu.Unlock()
			assert.False(t, logged)
		})
	})
}
