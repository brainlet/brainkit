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

func TestGoDirect_Memory(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			tk := newTestKernelWithStorage(t)
			rt := sdk.Runtime(tk)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Initialize memory directly via EvalTS so it's set on the Kit's globalThis
			// (Deploy creates a Compartment with isolated globals — __kit_memory wouldn't propagate)
			_, err := tk.EvalTS(ctx, "__memory_init.ts", `
				const mem = createMemory({
					storage: new InMemoryStore(),
					lastMessages: 10,
				});
				globalThis.__kit_memory = mem;
				return "ok";
			`)
			require.NoError(t, err, "memory init must succeed")

			t.Run("CreateThread", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.MemoryCreateThreadMsg, messages.MemoryCreateThreadResp](rt, ctx, messages.MemoryCreateThreadMsg{})
				require.NoError(t, err)
				assert.NotEmpty(t, resp.ThreadID, "should return a thread ID")
			})

			t.Run("CreateThread_Save_Recall", func(t *testing.T) {
				createResp, err := sdk.PublishAwait[messages.MemoryCreateThreadMsg, messages.MemoryCreateThreadResp](rt, ctx, messages.MemoryCreateThreadMsg{})
				require.NoError(t, err)
				threadID := createResp.ThreadID

				_, err = sdk.PublishAwait[messages.MemorySaveMsg, messages.MemorySaveResp](rt, ctx, messages.MemorySaveMsg{
					ThreadID: threadID,
					Messages: []messages.MemoryMessage{
						{Role: "user", Content: "What is brainkit?"},
						{Role: "assistant", Content: "Brainkit is a Go runtime for AI agents."},
					},
				})
				require.NoError(t, err)

				recallResp, err := sdk.PublishAwait[messages.MemoryRecallMsg, messages.MemoryRecallResp](rt, ctx, messages.MemoryRecallMsg{
					ThreadID: threadID,
					Query:    "brainkit",
				})
				require.NoError(t, err)
				assert.NotEmpty(t, recallResp.Messages, "should recall saved messages")
			})

			t.Run("GetThread", func(t *testing.T) {
				createResp, err := sdk.PublishAwait[messages.MemoryCreateThreadMsg, messages.MemoryCreateThreadResp](rt, ctx, messages.MemoryCreateThreadMsg{})
				require.NoError(t, err)

				getResp, err := sdk.PublishAwait[messages.MemoryGetThreadMsg, messages.MemoryGetThreadResp](rt, ctx, messages.MemoryGetThreadMsg{
					ThreadID: createResp.ThreadID,
				})
				require.NoError(t, err)
				assert.NotNil(t, getResp.Thread, "should return thread data")
			})

			t.Run("ListThreads", func(t *testing.T) {
				listResp, err := sdk.PublishAwait[messages.MemoryListThreadsMsg, messages.MemoryListThreadsResp](rt, ctx, messages.MemoryListThreadsMsg{})
				require.NoError(t, err)
				assert.NotNil(t, listResp.Threads, "should return threads array")
			})

			t.Run("DeleteThread", func(t *testing.T) {
				createResp, err := sdk.PublishAwait[messages.MemoryCreateThreadMsg, messages.MemoryCreateThreadResp](rt, ctx, messages.MemoryCreateThreadMsg{})
				require.NoError(t, err)

				deleteResp, err := sdk.PublishAwait[messages.MemoryDeleteThreadMsg, messages.MemoryDeleteThreadResp](rt, ctx, messages.MemoryDeleteThreadMsg{
					ThreadID: createResp.ThreadID,
				})
				require.NoError(t, err)
				assert.True(t, deleteResp.OK)
			})
		})
	}
}
