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

				resp, err := sdk.PublishAwait[messages.FsReadMsg, messages.FsReadResp](rt, ctx, messages.FsReadMsg{Path: "test.txt"})
				require.NoError(t, err)
				assert.Equal(t, "hello fs", resp.Data)
			})

			t.Run("Write_Overwrite", func(t *testing.T) {
				sdk.PublishAwait[messages.FsWriteMsg, messages.FsWriteResp](rt, ctx, messages.FsWriteMsg{Path: "overwrite.txt", Data: "v1"})
				sdk.PublishAwait[messages.FsWriteMsg, messages.FsWriteResp](rt, ctx, messages.FsWriteMsg{Path: "overwrite.txt", Data: "v2"})

				resp, err := sdk.PublishAwait[messages.FsReadMsg, messages.FsReadResp](rt, ctx, messages.FsReadMsg{Path: "overwrite.txt"})
				require.NoError(t, err)
				assert.Equal(t, "v2", resp.Data)
			})

			t.Run("Mkdir_Recursive", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.FsMkdirMsg, messages.FsMkdirResp](rt, ctx, messages.FsMkdirMsg{Path: "a/b/c"})
				require.NoError(t, err)
				assert.True(t, resp.OK)

				_, err = sdk.PublishAwait[messages.FsWriteMsg, messages.FsWriteResp](rt, ctx, messages.FsWriteMsg{Path: "a/b/c/deep.txt", Data: "deep"})
				require.NoError(t, err)

				readResp, err := sdk.PublishAwait[messages.FsReadMsg, messages.FsReadResp](rt, ctx, messages.FsReadMsg{Path: "a/b/c/deep.txt"})
				require.NoError(t, err)
				assert.Equal(t, "deep", readResp.Data)
			})

			t.Run("List_WithPattern", func(t *testing.T) {
				sdk.PublishAwait[messages.FsWriteMsg, messages.FsWriteResp](rt, ctx, messages.FsWriteMsg{Path: "listdir/a.txt", Data: "a"})
				sdk.PublishAwait[messages.FsWriteMsg, messages.FsWriteResp](rt, ctx, messages.FsWriteMsg{Path: "listdir/b.json", Data: "{}"})
				sdk.PublishAwait[messages.FsWriteMsg, messages.FsWriteResp](rt, ctx, messages.FsWriteMsg{Path: "listdir/c.txt", Data: "c"})

				resp, err := sdk.PublishAwait[messages.FsListMsg, messages.FsListResp](rt, ctx, messages.FsListMsg{Path: "listdir", Pattern: "*.txt"})
				require.NoError(t, err)
				assert.Len(t, resp.Files, 2) // a.txt, c.txt
			})

			t.Run("Stat_File", func(t *testing.T) {
				sdk.PublishAwait[messages.FsWriteMsg, messages.FsWriteResp](rt, ctx, messages.FsWriteMsg{Path: "stat-target.txt", Data: "12345"})

				resp, err := sdk.PublishAwait[messages.FsStatMsg, messages.FsStatResp](rt, ctx, messages.FsStatMsg{Path: "stat-target.txt"})
				require.NoError(t, err)
				assert.False(t, resp.IsDir)
				assert.Equal(t, int64(5), resp.Size)
				assert.NotEmpty(t, resp.ModTime)
			})

			t.Run("Stat_Directory", func(t *testing.T) {
				sdk.PublishAwait[messages.FsMkdirMsg, messages.FsMkdirResp](rt, ctx, messages.FsMkdirMsg{Path: "stat-dir"})

				resp, err := sdk.PublishAwait[messages.FsStatMsg, messages.FsStatResp](rt, ctx, messages.FsStatMsg{Path: "stat-dir"})
				require.NoError(t, err)
				assert.True(t, resp.IsDir)
			})

			t.Run("Delete", func(t *testing.T) {
				sdk.PublishAwait[messages.FsWriteMsg, messages.FsWriteResp](rt, ctx, messages.FsWriteMsg{Path: "delete-me.txt", Data: "x"})

				resp, err := sdk.PublishAwait[messages.FsDeleteMsg, messages.FsDeleteResp](rt, ctx, messages.FsDeleteMsg{Path: "delete-me.txt"})
				require.NoError(t, err)
				assert.True(t, resp.OK)

				_, err = sdk.PublishAwait[messages.FsReadMsg, messages.FsReadResp](rt, ctx, messages.FsReadMsg{Path: "delete-me.txt"})
				assert.Error(t, err)
			})

			t.Run("Delete_NotFound", func(t *testing.T) {
				_, err := sdk.PublishAwait[messages.FsDeleteMsg, messages.FsDeleteResp](rt, ctx, messages.FsDeleteMsg{Path: "ghost.txt"})
				assert.Error(t, err)
			})

			t.Run("Read_NotFound", func(t *testing.T) {
				_, err := sdk.PublishAwait[messages.FsReadMsg, messages.FsReadResp](rt, ctx, messages.FsReadMsg{Path: "nope.txt"})
				assert.Error(t, err)
			})

			t.Run("PathTraversal_Rejected", func(t *testing.T) {
				_, err := sdk.PublishAwait[messages.FsReadMsg, messages.FsReadResp](rt, ctx, messages.FsReadMsg{Path: "../../etc/passwd"})
				assert.Error(t, err)
			})
		})
	}
}
