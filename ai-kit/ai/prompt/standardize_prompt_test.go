// Ported from: packages/ai/src/prompt/standardize-prompt.test.ts
package prompt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStandardizePrompt(t *testing.T) {
	t.Run("should return error when messages array is empty", func(t *testing.T) {
		_, err := StandardizePrompt(Prompt{
			Messages: []ModelMessage{},
		})
		assert.Error(t, err)
		assert.True(t, IsInvalidPromptError(err))
	})

	t.Run("should return error when neither prompt nor messages defined", func(t *testing.T) {
		_, err := StandardizePrompt(Prompt{})
		assert.Error(t, err)
		assert.True(t, IsInvalidPromptError(err))
	})

	t.Run("should return error when both prompt and messages defined", func(t *testing.T) {
		_, err := StandardizePrompt(Prompt{
			PromptValue: "hello",
			Messages:    []ModelMessage{UserModelMessage{Role: "user", Content: "hello"}},
		})
		assert.Error(t, err)
		assert.True(t, IsInvalidPromptError(err))
	})

	t.Run("should support SystemModelMessage system message", func(t *testing.T) {
		result, err := StandardizePrompt(Prompt{
			System: SystemModelMessage{
				Role:    "system",
				Content: "INSTRUCTIONS",
			},
			PromptValue: "Hello, world!",
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, len(result.Messages))

		sys := result.System.(SystemModelMessage)
		assert.Equal(t, "INSTRUCTIONS", sys.Content)

		msg := result.Messages[0].(UserModelMessage)
		assert.Equal(t, "user", msg.Role)
		assert.Equal(t, "Hello, world!", msg.Content)
	})

	t.Run("should support array of SystemModelMessage system messages", func(t *testing.T) {
		result, err := StandardizePrompt(Prompt{
			System: []SystemModelMessage{
				{Role: "system", Content: "INSTRUCTIONS"},
				{Role: "system", Content: "INSTRUCTIONS 2"},
			},
			PromptValue: "Hello, world!",
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, len(result.Messages))

		sysArr := result.System.([]SystemModelMessage)
		assert.Equal(t, 2, len(sysArr))
		assert.Equal(t, "INSTRUCTIONS", sysArr[0].Content)
		assert.Equal(t, "INSTRUCTIONS 2", sysArr[1].Content)
	})

	t.Run("should support string prompt", func(t *testing.T) {
		result, err := StandardizePrompt(Prompt{
			PromptValue: "Hello, world!",
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, len(result.Messages))

		msg := result.Messages[0].(UserModelMessage)
		assert.Equal(t, "user", msg.Role)
		assert.Equal(t, "Hello, world!", msg.Content)
	})
}
