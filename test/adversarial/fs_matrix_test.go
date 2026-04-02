package adversarial_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"context"
	"time"
)

// TestFSMatrix_WriteReadDelete — full lifecycle via polyfill.
func TestFSMatrix_WriteReadDelete(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Write + Read
	result, err := tk.EvalTS(ctx, "__test.ts", `
		fs.writeFileSync("fs-test.txt", "hello fs");
		return fs.readFileSync("fs-test.txt", "utf8");
	`)
	require.NoError(t, err)
	assert.Equal(t, "hello fs", result)

	// Stat
	statResult, err := tk.EvalTS(ctx, "__test.ts", `
		var s = fs.statSync("fs-test.txt");
		return JSON.stringify({size: s.size});
	`)
	require.NoError(t, err)
	assert.Contains(t, statResult, "size")

	// Delete
	delResult, err := tk.EvalTS(ctx, "__test.ts", `
		fs.unlinkSync("fs-test.txt");
		return "ok";
	`)
	require.NoError(t, err)
	assert.Equal(t, "ok", delResult)

	// Read again — should fail
	readResult, err := tk.EvalTS(ctx, "__test.ts", `
		try { fs.readFileSync("fs-test.txt", "utf8"); return "SHOULD_FAIL"; }
		catch(e) { return e.code || "error"; }
	`)
	require.NoError(t, err)
	assert.NotEqual(t, "SHOULD_FAIL", readResult)
}

// TestFSMatrix_MkdirAndList — create dir, list contents.
func TestFSMatrix_MkdirAndList(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := tk.EvalTS(ctx, "__test.ts", `
		fs.mkdirSync("test-dir", {recursive: true});
		fs.writeFileSync("test-dir/file.txt", "inside");
		var files = fs.readdirSync("test-dir");
		return JSON.stringify(files);
	`)
	require.NoError(t, err)
	assert.Contains(t, result, "file.txt")
}

// TestFSMatrix_ReadNonexistent — read nonexistent file returns error.
func TestFSMatrix_ReadNonexistent(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := tk.EvalTS(ctx, "__test.ts", `
		try { fs.readFileSync("ghost-file-xyz.txt", "utf8"); return "SHOULD_FAIL"; }
		catch(e) { return e.code || "error"; }
	`)
	require.NoError(t, err)
	assert.NotEqual(t, "SHOULD_FAIL", result)
}

// TestFSMatrix_WorkspaceEscape — path traversal blocked.
func TestFSMatrix_WorkspaceEscape(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := tk.EvalTS(ctx, "__test.ts", `
		try { fs.readFileSync("../../../etc/passwd"); return "SHOULD_FAIL"; }
		catch(e) { return e.code || "error"; }
	`)
	require.NoError(t, err)
	assert.NotEqual(t, "SHOULD_FAIL", result)
}

// TestFSMatrix_LargeFileWrite — write and read 1MB file.
func TestFSMatrix_LargeFileWrite(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = strings.Repeat("x", 1024*1024) // keep strings import used

	result, err := tk.EvalTS(ctx, "__test.ts", `
		var big = "";
		for (var i = 0; i < 1024*1024; i++) big += "x";
		fs.writeFileSync("big-file.txt", big);
		var s = fs.statSync("big-file.txt");
		return JSON.stringify({size: s.size});
	`)
	require.NoError(t, err)
	var stat struct{ Size int64 `json:"size"` }
	json.Unmarshal([]byte(result), &stat)
	assert.Equal(t, int64(1024*1024), stat.Size)
}

// TestFSMatrix_ListWithPattern — list directory contents.
func TestFSMatrix_ListWithPattern(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := tk.EvalTS(ctx, "__test.ts", `
		fs.writeFileSync("alpha.txt", "content");
		fs.writeFileSync("beta.txt", "content");
		fs.writeFileSync("alpha.json", "content");
		fs.writeFileSync("gamma.txt", "content");
		var files = fs.readdirSync(".");
		return JSON.stringify(files);
	`)
	require.NoError(t, err)
	assert.Contains(t, result, "alpha.txt")
	assert.Contains(t, result, "beta.txt")
	assert.Contains(t, result, "gamma.txt")
	assert.Contains(t, result, "alpha.json")
}

// TestFSMatrix_StatDirectory — stat on directory returns isDir=true.
func TestFSMatrix_StatDirectory(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := tk.EvalTS(ctx, "__test.ts", `
		fs.mkdirSync("stat-dir", {recursive: true});
		var s = fs.statSync("stat-dir");
		return JSON.stringify({isDir: s.isDirectory()});
	`)
	require.NoError(t, err)
	assert.Contains(t, result, `"isDir":true`)
}

// TestFSMatrix_FromTS — all FS ops from .ts surface.
func TestFSMatrix_FromTS(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "fs-ts-test.ts", `
		fs.writeFileSync("ts-file.txt", "from typescript");
		var data = fs.readFileSync("ts-file.txt", "utf8");
		var stat = fs.statSync("ts-file.txt");
		var list = fs.readdirSync(".");
		fs.unlinkSync("ts-file.txt");

		output({
			data: data,
			size: stat.size,
			filesFound: list.length,
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
