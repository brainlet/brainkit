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

func TestGoDirect_FS(t *testing.T) {
	for _, factory := range []struct {
		name string
		make func(t *testing.T) sdk.Runtime
	}{
		{"Kernel", newTestKernel},
		{"Node", newTestNode},
	} {
		t.Run(factory.name, func(t *testing.T) {
			rt := factory.make(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			t.Run("Write_Read_Roundtrip", func(t *testing.T) {
				_, err := sdk.PublishAwait[messages.FsWriteMsg, messages.FsWriteResp](rt, ctx, messages.FsWriteMsg{
					Path: "test.txt", Data: "hello fs",
				})
				require.NoError(t, err)

				_pr1, err := sdk.Publish(rt, ctx, messages.FsReadMsg{Path: "test.txt"})
				require.NoError(t, err)
				_ch1 := make(chan messages.FsReadResp, 1)
				_us1, err := sdk.SubscribeTo[messages.FsReadResp](rt, ctx, _pr1.ReplyTo, func(r messages.FsReadResp, m messages.Message) { _ch1 <- r })
				require.NoError(t, err)
				defer _us1()
				var resp messages.FsReadResp
				select {
				case resp = <-_ch1:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.Equal(t, "hello fs", resp.Data)
			})

			t.Run("Write_Overwrite", func(t *testing.T) {
				sdk.Publish(rt, ctx, messages.FsWriteMsg{Path: "overwrite.txt", Data: "v1"})
				sdk.Publish(rt, ctx, messages.FsWriteMsg{Path: "overwrite.txt", Data: "v2"})

				_pr2, err := sdk.Publish(rt, ctx, messages.FsReadMsg{Path: "overwrite.txt"})
				require.NoError(t, err)
				_ch2 := make(chan messages.FsReadResp, 1)
				_us2, err := sdk.SubscribeTo[messages.FsReadResp](rt, ctx, _pr2.ReplyTo, func(r messages.FsReadResp, m messages.Message) { _ch2 <- r })
				require.NoError(t, err)
				defer _us2()
				var resp messages.FsReadResp
				select {
				case resp = <-_ch2:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.Equal(t, "v2", resp.Data)
			})

			t.Run("Mkdir_Recursive", func(t *testing.T) {
				_pr3, err := sdk.Publish(rt, ctx, messages.FsMkdirMsg{Path: "a/b/c"})
				require.NoError(t, err)
				_ch3 := make(chan messages.FsMkdirResp, 1)
				_us3, err := sdk.SubscribeTo[messages.FsMkdirResp](rt, ctx, _pr3.ReplyTo, func(r messages.FsMkdirResp, m messages.Message) { _ch3 <- r })
				require.NoError(t, err)
				defer _us3()
				var resp messages.FsMkdirResp
				select {
				case resp = <-_ch3:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.True(t, resp.OK)

				_pr4, err := sdk.Publish(rt, ctx, messages.FsWriteMsg{Path: "a/b/c/deep.txt", Data: "deep"})
				require.NoError(t, err)
				_ch4 := make(chan messages.FsWriteResp, 1)
				_us4, _ := sdk.SubscribeTo[messages.FsWriteResp](rt, ctx, _pr4.ReplyTo, func(r messages.FsWriteResp, m messages.Message) { _ch4 <- r })
				defer _us4()
				select {
				case <-_ch4:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				_pr5, err := sdk.Publish(rt, ctx, messages.FsReadMsg{Path: "a/b/c/deep.txt"})
				require.NoError(t, err)
				_ch5 := make(chan messages.FsReadResp, 1)
				_us5, err := sdk.SubscribeTo[messages.FsReadResp](rt, ctx, _pr5.ReplyTo, func(r messages.FsReadResp, m messages.Message) { _ch5 <- r })
				require.NoError(t, err)
				defer _us5()
				var readResp messages.FsReadResp
				select {
				case readResp = <-_ch5:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.Equal(t, "deep", readResp.Data)
			})

			t.Run("List_WithPattern", func(t *testing.T) {
				sdk.Publish(rt, ctx, messages.FsWriteMsg{Path: "listdir/a.txt", Data: "a"})
				sdk.Publish(rt, ctx, messages.FsWriteMsg{Path: "listdir/b.json", Data: "{}"})
				sdk.Publish(rt, ctx, messages.FsWriteMsg{Path: "listdir/c.txt", Data: "c"})

				_pr6, err := sdk.Publish(rt, ctx, messages.FsListMsg{Path: "listdir", Pattern: "*.txt"})
				require.NoError(t, err)
				_ch6 := make(chan messages.FsListResp, 1)
				_us6, err := sdk.SubscribeTo[messages.FsListResp](rt, ctx, _pr6.ReplyTo, func(r messages.FsListResp, m messages.Message) { _ch6 <- r })
				require.NoError(t, err)
				defer _us6()
				var resp messages.FsListResp
				select {
				case resp = <-_ch6:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.Len(t, resp.Files, 2) // a.txt, c.txt
			})

			t.Run("Stat_File", func(t *testing.T) {
				sdk.Publish(rt, ctx, messages.FsWriteMsg{Path: "stat-target.txt", Data: "12345"})

				_pr7, err := sdk.Publish(rt, ctx, messages.FsStatMsg{Path: "stat-target.txt"})
				require.NoError(t, err)
				_ch7 := make(chan messages.FsStatResp, 1)
				_us7, err := sdk.SubscribeTo[messages.FsStatResp](rt, ctx, _pr7.ReplyTo, func(r messages.FsStatResp, m messages.Message) { _ch7 <- r })
				require.NoError(t, err)
				defer _us7()
				var resp messages.FsStatResp
				select {
				case resp = <-_ch7:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.False(t, resp.IsDir)
				assert.Equal(t, int64(5), resp.Size)
				assert.NotEmpty(t, resp.ModTime)
			})

			t.Run("Stat_Directory", func(t *testing.T) {
				sdk.Publish(rt, ctx, messages.FsMkdirMsg{Path: "stat-dir"})

				_pr8, err := sdk.Publish(rt, ctx, messages.FsStatMsg{Path: "stat-dir"})
				require.NoError(t, err)
				_ch8 := make(chan messages.FsStatResp, 1)
				_us8, err := sdk.SubscribeTo[messages.FsStatResp](rt, ctx, _pr8.ReplyTo, func(r messages.FsStatResp, m messages.Message) { _ch8 <- r })
				require.NoError(t, err)
				defer _us8()
				var resp messages.FsStatResp
				select {
				case resp = <-_ch8:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.True(t, resp.IsDir)
			})

			t.Run("Delete", func(t *testing.T) {
				sdk.Publish(rt, ctx, messages.FsWriteMsg{Path: "delete-me.txt", Data: "x"})

				_pr9, err := sdk.Publish(rt, ctx, messages.FsDeleteMsg{Path: "delete-me.txt"})
				require.NoError(t, err)
				_ch9 := make(chan messages.FsDeleteResp, 1)
				_us9, err := sdk.SubscribeTo[messages.FsDeleteResp](rt, ctx, _pr9.ReplyTo, func(r messages.FsDeleteResp, m messages.Message) { _ch9 <- r })
				require.NoError(t, err)
				defer _us9()
				var resp messages.FsDeleteResp
				select {
				case resp = <-_ch9:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.True(t, resp.OK)

				_pr10, err := sdk.Publish(rt, ctx, messages.FsReadMsg{Path: "delete-me.txt"})
				assert.Error(t, err)
			})

			t.Run("Delete_NotFound", func(t *testing.T) {
				_pr11, err := sdk.Publish(rt, ctx, messages.FsDeleteMsg{Path: "ghost.txt"})
				assert.Error(t, err)
			})

			t.Run("Read_NotFound", func(t *testing.T) {
				_pr12, err := sdk.Publish(rt, ctx, messages.FsReadMsg{Path: "nope.txt"})
				assert.Error(t, err)
			})

			t.Run("PathTraversal_Rejected", func(t *testing.T) {
				_pr13, err := sdk.Publish(rt, ctx, messages.FsReadMsg{Path: "../../etc/passwd"})
				assert.Error(t, err)
			})
		})
	}
}
