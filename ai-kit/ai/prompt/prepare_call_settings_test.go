// Ported from: packages/ai/src/prompt/prepare-call-settings.test.ts
package prompt

import (
	"testing"

	aierror "github.com/brainlet/brainkit/ai-kit/ai/error"

	"github.com/stretchr/testify/assert"
)

func intPtr(i int) *int       { return &i }
func floatPtr(f float64) *float64 { return &f }

func TestPrepareCallSettings(t *testing.T) {
	t.Run("valid inputs", func(t *testing.T) {
		t.Run("should not return error for valid settings", func(t *testing.T) {
			_, err := PrepareCallSettings(PrepareCallSettingsInput{
				MaxOutputTokens:  intPtr(100),
				Temperature:      floatPtr(0.7),
				TopP:             floatPtr(0.9),
				TopK:             intPtr(50),
				PresencePenalty:  floatPtr(0.5),
				FrequencyPenalty: floatPtr(0.3),
				Seed:             intPtr(42),
			})
			assert.NoError(t, err)
		})

		t.Run("should allow nil values for optional settings", func(t *testing.T) {
			_, err := PrepareCallSettings(PrepareCallSettingsInput{})
			assert.NoError(t, err)
		})
	})

	t.Run("invalid inputs", func(t *testing.T) {
		t.Run("maxOutputTokens", func(t *testing.T) {
			t.Run("should return error if maxOutputTokens is less than 1", func(t *testing.T) {
				_, err := PrepareCallSettings(PrepareCallSettingsInput{
					MaxOutputTokens: intPtr(0),
				})
				assert.Error(t, err)
				assert.True(t, aierror.IsInvalidArgumentError(err))
			})
		})
	})

	t.Run("should return a new object with limited values", func(t *testing.T) {
		result, err := PrepareCallSettings(PrepareCallSettingsInput{
			MaxOutputTokens: intPtr(100),
			Temperature:     floatPtr(0.7),
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, intPtr(100), result.MaxOutputTokens)
		assert.Equal(t, floatPtr(0.7), result.Temperature)
		assert.Nil(t, result.TopP)
		assert.Nil(t, result.TopK)
		assert.Nil(t, result.PresencePenalty)
		assert.Nil(t, result.FrequencyPenalty)
		assert.Nil(t, result.Seed)
		assert.Nil(t, result.StopSequences)
	})
}
