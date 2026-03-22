package test

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoDirect_AI(t *testing.T) {
	loadEnv(t)
	if !hasAIKey() {
		t.Skip("OPENAI_API_KEY required — set in .env")
	}

	rt := newTestKernel(t)

	t.Run("Generate", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := sdk.PublishAwait[messages.AiGenerateMsg, messages.AiGenerateResp](rt, ctx, messages.AiGenerateMsg{
			Model:  "openai/gpt-4o-mini",
			Prompt: "Reply with exactly one word: hello",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Text, "should generate text")
		t.Logf("AI response: %q (tokens: %d)", resp.Text, resp.Usage.TotalTokens)
	})

	t.Run("GenerateObject", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := sdk.PublishAwait[messages.AiGenerateObjectMsg, messages.AiGenerateObjectResp](rt, ctx, messages.AiGenerateObjectMsg{
			Model:  "openai/gpt-4o-mini",
			Prompt: "Generate a person with name 'Alice' and age 30",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
					"age":  map[string]any{"type": "number"},
				},
				"required": []string{"name", "age"},
			},
		})
		require.NoError(t, err)
		assert.NotNil(t, resp.Object)
		t.Logf("AI object: %v", resp.Object)
	})

	t.Run("Embed", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := sdk.PublishAwait[messages.AiEmbedMsg, messages.AiEmbedResp](rt, ctx, messages.AiEmbedMsg{
			Model: "openai/text-embedding-3-small",
			Value: "hello world",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Embedding, "should return embedding vector")
		t.Logf("Embedding dimensions: %d", len(resp.Embedding))
	})

	t.Run("EmbedMany", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := sdk.PublishAwait[messages.AiEmbedManyMsg, messages.AiEmbedManyResp](rt, ctx, messages.AiEmbedManyMsg{
			Model:  "openai/text-embedding-3-small",
			Values: []string{"hello", "world", "foo"},
		})
		require.NoError(t, err)
		assert.Len(t, resp.Embeddings, 3, "should return 3 embeddings")
		for i, emb := range resp.Embeddings {
			assert.NotEmpty(t, emb, "embedding %d should not be empty", i)
		}
		t.Logf("EmbedMany: 3 embeddings, %d dimensions each", len(resp.Embeddings[0]))
	})
}
