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

		_pr1, err := sdk.Publish(rt, ctx, messages.AiGenerateMsg{
			Model:  "openai/gpt-4o-mini",
			Prompt: "Reply with exactly one word: hello",
		})
		require.NoError(t, err)
		_ch1 := make(chan messages.AiGenerateResp, 1)
		_us1, err := sdk.SubscribeTo[messages.AiGenerateResp](rt, ctx, _pr1.ReplyTo, func(r messages.AiGenerateResp, m messages.Message) { _ch1 <- r })
		require.NoError(t, err)
		defer _us1()
		var resp messages.AiGenerateResp
		select {
		case resp = <-_ch1:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotEmpty(t, resp.Text, "should generate text")
		t.Logf("AI response: %q (tokens: %d)", resp.Text, resp.Usage.TotalTokens)
	})

	t.Run("GenerateObject", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_pr2, err := sdk.Publish(rt, ctx, messages.AiGenerateObjectMsg{
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
		_ch2 := make(chan messages.AiGenerateObjectResp, 1)
		_us2, err := sdk.SubscribeTo[messages.AiGenerateObjectResp](rt, ctx, _pr2.ReplyTo, func(r messages.AiGenerateObjectResp, m messages.Message) { _ch2 <- r })
		require.NoError(t, err)
		defer _us2()
		var resp messages.AiGenerateObjectResp
		select {
		case resp = <-_ch2:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotNil(t, resp.Object)
		t.Logf("AI object: %v", resp.Object)
	})

	t.Run("Embed", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_pr3, err := sdk.Publish(rt, ctx, messages.AiEmbedMsg{
			Model: "openai/text-embedding-3-small",
			Value: "hello world",
		})
		require.NoError(t, err)
		_ch3 := make(chan messages.AiEmbedResp, 1)
		_us3, err := sdk.SubscribeTo[messages.AiEmbedResp](rt, ctx, _pr3.ReplyTo, func(r messages.AiEmbedResp, m messages.Message) { _ch3 <- r })
		require.NoError(t, err)
		defer _us3()
		var resp messages.AiEmbedResp
		select {
		case resp = <-_ch3:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotEmpty(t, resp.Embedding, "should return embedding vector")
		t.Logf("Embedding dimensions: %d", len(resp.Embedding))
	})

	t.Run("EmbedMany", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_pr4, err := sdk.Publish(rt, ctx, messages.AiEmbedManyMsg{
			Model:  "openai/text-embedding-3-small",
			Values: []string{"hello", "world", "foo"},
		})
		require.NoError(t, err)
		_ch4 := make(chan messages.AiEmbedManyResp, 1)
		_us4, err := sdk.SubscribeTo[messages.AiEmbedManyResp](rt, ctx, _pr4.ReplyTo, func(r messages.AiEmbedManyResp, m messages.Message) { _ch4 <- r })
		require.NoError(t, err)
		defer _us4()
		var resp messages.AiEmbedManyResp
		select {
		case resp = <-_ch4:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.Len(t, resp.Embeddings, 3, "should return 3 embeddings")
		for i, emb := range resp.Embeddings {
			assert.NotEmpty(t, emb, "embedding %d should not be empty", i)
		}
		t.Logf("EmbedMany: 3 embeddings, %d dimensions each", len(resp.Embeddings[0]))
	})
}
