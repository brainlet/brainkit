package adversarial_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFSMatrix_WriteReadDelete — full lifecycle via bus.
func TestFSMatrix_WriteReadDelete(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Write
	pr1, _ := sdk.Publish(tk, ctx, messages.FsWriteMsg{Path: "fs-test.txt", Data: "hello fs"})
	ch1 := make(chan []byte, 1)
	unsub1, _ := tk.SubscribeRaw(ctx, pr1.ReplyTo, func(m messages.Message) { ch1 <- m.Payload })
	<-ch1
	unsub1()

	// Read
	pr2, _ := sdk.Publish(tk, ctx, messages.FsReadMsg{Path: "fs-test.txt"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	p2 := <-ch2
	unsub2()
	assert.Contains(t, string(p2), "hello fs")

	// Stat
	pr3, _ := sdk.Publish(tk, ctx, messages.FsStatMsg{Path: "fs-test.txt"})
	ch3 := make(chan []byte, 1)
	unsub3, _ := tk.SubscribeRaw(ctx, pr3.ReplyTo, func(m messages.Message) { ch3 <- m.Payload })
	p3 := <-ch3
	unsub3()
	assert.Contains(t, string(p3), "size")

	// Delete
	pr4, _ := sdk.Publish(tk, ctx, messages.FsDeleteMsg{Path: "fs-test.txt"})
	ch4 := make(chan []byte, 1)
	unsub4, _ := tk.SubscribeRaw(ctx, pr4.ReplyTo, func(m messages.Message) { ch4 <- m.Payload })
	p4 := <-ch4
	unsub4()
	assert.Contains(t, string(p4), "ok")

	// Read again — should fail
	pr5, _ := sdk.Publish(tk, ctx, messages.FsReadMsg{Path: "fs-test.txt"})
	ch5 := make(chan json.RawMessage, 1)
	unsub5, _ := tk.SubscribeRaw(ctx, pr5.ReplyTo, func(m messages.Message) { ch5 <- json.RawMessage(m.Payload) })
	defer unsub5()
	p5 := <-ch5
	assert.True(t, responseHasError(p5), "reading deleted file should error")
}

// TestFSMatrix_MkdirAndList — create dir, list contents.
func TestFSMatrix_MkdirAndList(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Mkdir
	pr1, _ := sdk.Publish(tk, ctx, messages.FsMkdirMsg{Path: "test-dir"})
	ch1 := make(chan []byte, 1)
	unsub1, _ := tk.SubscribeRaw(ctx, pr1.ReplyTo, func(m messages.Message) { ch1 <- m.Payload })
	<-ch1
	unsub1()

	// Write file inside
	pr2, _ := sdk.Publish(tk, ctx, messages.FsWriteMsg{Path: "test-dir/file.txt", Data: "inside"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	<-ch2
	unsub2()

	// List
	pr3, _ := sdk.Publish(tk, ctx, messages.FsListMsg{Path: "test-dir"})
	ch3 := make(chan []byte, 1)
	unsub3, _ := tk.SubscribeRaw(ctx, pr3.ReplyTo, func(m messages.Message) { ch3 <- m.Payload })
	defer unsub3()
	p3 := <-ch3
	assert.Contains(t, string(p3), "file.txt")
}

// TestFSMatrix_ReadNonexistent — read nonexistent file returns error.
func TestFSMatrix_ReadNonexistent(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, _ := sdk.Publish(tk, ctx, messages.FsReadMsg{Path: "ghost-file-xyz.txt"})
	ch := make(chan json.RawMessage, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- json.RawMessage(m.Payload) })
	defer unsub()

	select {
	case p := <-ch:
		assert.True(t, responseHasError(p))
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// TestFSMatrix_WorkspaceEscape — path traversal blocked.
func TestFSMatrix_WorkspaceEscape(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// filepath.Clean normalizes ../../../ to stay within root, so the escape check
	// in resolveWorkspacePath only fires if the resolved path literally exits the prefix.
	// With filepath.Join(workspace, filepath.Clean("/"+path)), ../../../ → /etc/passwd → workspace/etc/passwd.
	// So escape only happens if Clean produces a path outside the workspace.
	// Test that we at least get an error (file not found within workspace).
	pr, _ := sdk.Publish(tk, ctx, messages.FsReadMsg{Path: "../../../etc/passwd"})
	ch := make(chan json.RawMessage, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- json.RawMessage(m.Payload) })
	defer unsub()

	select {
	case p := <-ch:
		assert.True(t, responseHasError(p), "traversal should error (escape or not found)")
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// TestFSMatrix_LargeFileWrite — write and read 1MB file.
func TestFSMatrix_LargeFileWrite(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	big := strings.Repeat("x", 1024*1024)

	pr1, _ := sdk.Publish(tk, ctx, messages.FsWriteMsg{Path: "big-file.txt", Data: big})
	ch1 := make(chan []byte, 1)
	unsub1, _ := tk.SubscribeRaw(ctx, pr1.ReplyTo, func(m messages.Message) { ch1 <- m.Payload })
	<-ch1
	unsub1()

	pr2, _ := sdk.Publish(tk, ctx, messages.FsStatMsg{Path: "big-file.txt"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	defer unsub2()
	p2 := <-ch2

	var stat struct{ Size int64 `json:"size"` }
	json.Unmarshal(p2, &stat)
	assert.Equal(t, int64(1024*1024), stat.Size)
}

// TestFSMatrix_ListWithPattern — list with glob pattern.
func TestFSMatrix_ListWithPattern(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Write several files
	for _, name := range []string{"alpha.txt", "beta.txt", "alpha.json", "gamma.txt"} {
		pr, _ := sdk.Publish(tk, ctx, messages.FsWriteMsg{Path: name, Data: "content"})
		ch := make(chan []byte, 1)
		unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
		<-ch
		unsub()
	}

	// List with pattern
	pr, _ := sdk.Publish(tk, ctx, messages.FsListMsg{Path: ".", Pattern: "*.txt"})
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()
	p := <-ch

	assert.Contains(t, string(p), "alpha.txt")
	assert.Contains(t, string(p), "beta.txt")
	assert.Contains(t, string(p), "gamma.txt")
	assert.NotContains(t, string(p), "alpha.json")
}

// TestFSMatrix_StatDirectory — stat on directory returns isDir=true.
func TestFSMatrix_StatDirectory(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr1, _ := sdk.Publish(tk, ctx, messages.FsMkdirMsg{Path: "stat-dir"})
	ch1 := make(chan []byte, 1)
	unsub1, _ := tk.SubscribeRaw(ctx, pr1.ReplyTo, func(m messages.Message) { ch1 <- m.Payload })
	<-ch1
	unsub1()

	pr2, _ := sdk.Publish(tk, ctx, messages.FsStatMsg{Path: "stat-dir"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	defer unsub2()
	p2 := <-ch2

	assert.Contains(t, string(p2), `"isDir":true`)
}

// TestFSMatrix_FromTS — all FS ops from .ts surface.
func TestFSMatrix_FromTS(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "fs-ts-test.ts", `
		await fs.write("ts-file.txt", "from typescript");
		var read = await fs.read("ts-file.txt");
		var stat = await fs.stat("ts-file.txt");
		var list = await fs.list(".", "ts-*");
		await fs.delete("ts-file.txt");

		output({
			data: read.data,
			size: stat.size,
			filesFound: list.files.length,
			deleted: true,
		});
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__fs_ts.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || {});
	`)
	assert.Contains(t, result, "from typescript")
	assert.Contains(t, result, `"deleted":true`)
}
