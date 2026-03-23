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
			tk := newTestKernelWithStorageAndBackend(t, backend)
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
				_pr1, err := sdk.Publish(rt, ctx, messages.MemoryCreateThreadMsg{})
				require.NoError(t, err)
				_ch1 := make(chan messages.MemoryCreateThreadResp, 1)
				_us1, err := sdk.SubscribeTo[messages.MemoryCreateThreadResp](rt, ctx, _pr1.ReplyTo, func(r messages.MemoryCreateThreadResp, m messages.Message) { _ch1 <- r })
				require.NoError(t, err)
				defer _us1()
				var resp messages.MemoryCreateThreadResp
				select {
				case resp = <-_ch1:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.NotEmpty(t, resp.ThreadID, "should return a thread ID")
			})

			t.Run("CreateThread_Save_Recall", func(t *testing.T) {
				_pr2, err := sdk.Publish(rt, ctx, messages.MemoryCreateThreadMsg{})
				require.NoError(t, err)
				_ch2 := make(chan messages.MemoryCreateThreadResp, 1)
				_us2, err := sdk.SubscribeTo[messages.MemoryCreateThreadResp](rt, ctx, _pr2.ReplyTo, func(r messages.MemoryCreateThreadResp, m messages.Message) { _ch2 <- r })
				require.NoError(t, err)
				defer _us2()
				var createResp messages.MemoryCreateThreadResp
				select {
				case createResp = <-_ch2:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				threadID := createResp.ThreadID

				_, err = sdk.PublishAwait[messages.MemorySaveMsg, messages.MemorySaveResp](rt, ctx, messages.MemorySaveMsg{
					ThreadID: threadID,
					Messages: []messages.MemoryMessage{
						{Role: "user", Content: "What is brainkit?"},
						{Role: "assistant", Content: "Brainkit is a Go runtime for AI agents."},
					},
				})
				require.NoError(t, err)

				// Recall succeeds (returns empty without vector store — semantic search needs embedder)
				_, err = sdk.PublishAwait[messages.MemoryRecallMsg, messages.MemoryRecallResp](rt, ctx, messages.MemoryRecallMsg{
					ThreadID: threadID,
					Query:    "brainkit",
				})
				require.NoError(t, err)
			})

			t.Run("GetThread", func(t *testing.T) {
				_pr3, err := sdk.Publish(rt, ctx, messages.MemoryCreateThreadMsg{})
				require.NoError(t, err)
				_ch3 := make(chan messages.MemoryCreateThreadResp, 1)
				_us3, err := sdk.SubscribeTo[messages.MemoryCreateThreadResp](rt, ctx, _pr3.ReplyTo, func(r messages.MemoryCreateThreadResp, m messages.Message) { _ch3 <- r })
				require.NoError(t, err)
				defer _us3()
				var createResp messages.MemoryCreateThreadResp
				select {
				case createResp = <-_ch3:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				getResp, err := sdk.PublishAwait[messages.MemoryGetThreadMsg, messages.MemoryGetThreadResp](rt, ctx, messages.MemoryGetThreadMsg{
					ThreadID: createResp.ThreadID,
				})
				require.NoError(t, err)
				assert.NotNil(t, getResp.Thread, "should return thread data")
			})

			t.Run("ListThreads", func(t *testing.T) {
				_pr4, err := sdk.Publish(rt, ctx, messages.MemoryListThreadsMsg{})
				require.NoError(t, err)
				_ch4 := make(chan messages.MemoryListThreadsResp, 1)
				_us4, err := sdk.SubscribeTo[messages.MemoryListThreadsResp](rt, ctx, _pr4.ReplyTo, func(r messages.MemoryListThreadsResp, m messages.Message) { _ch4 <- r })
				require.NoError(t, err)
				defer _us4()
				var listResp messages.MemoryListThreadsResp
				select {
				case listResp = <-_ch4:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.NotNil(t, listResp.Threads, "should return threads array")
			})

			t.Run("DeleteThread", func(t *testing.T) {
				_pr5, err := sdk.Publish(rt, ctx, messages.MemoryCreateThreadMsg{})
				require.NoError(t, err)
				_ch5 := make(chan messages.MemoryCreateThreadResp, 1)
				_us5, err := sdk.SubscribeTo[messages.MemoryCreateThreadResp](rt, ctx, _pr5.ReplyTo, func(r messages.MemoryCreateThreadResp, m messages.Message) { _ch5 <- r })
				require.NoError(t, err)
				defer _us5()
				var createResp messages.MemoryCreateThreadResp
				select {
				case createResp = <-_ch5:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				deleteResp, err := sdk.PublishAwait[messages.MemoryDeleteThreadMsg, messages.MemoryDeleteThreadResp](rt, ctx, messages.MemoryDeleteThreadMsg{
					ThreadID: createResp.ThreadID,
				})
				require.NoError(t, err)
				assert.True(t, deleteResp.OK)
			})
		})
	}
}
