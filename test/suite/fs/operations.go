package fs

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testWriteReadRoundtrip(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := env.Kernel.EvalTS(ctx, "__test.ts", `fs.writeFileSync("test.txt", "hello fs"); return fs.readFileSync("test.txt", "utf8");`)
	require.NoError(t, err)
	assert.Equal(t, "hello fs", result)
}

func testWriteOverwrite(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := env.Kernel.EvalTS(ctx, "__test.ts", `
		fs.writeFileSync("overwrite.txt", "v1");
		fs.writeFileSync("overwrite.txt", "v2");
		return fs.readFileSync("overwrite.txt", "utf8");
	`)
	require.NoError(t, err)
	assert.Equal(t, "v2", result)
}

func testMkdirRecursive(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := env.Kernel.EvalTS(ctx, "__test.ts", `
		fs.mkdirSync("a/b/c", {recursive: true});
		fs.writeFileSync("a/b/c/deep.txt", "deep");
		return fs.readFileSync("a/b/c/deep.txt", "utf8");
	`)
	require.NoError(t, err)
	assert.Equal(t, "deep", result)
}

func testStatFile(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := env.Kernel.EvalTS(ctx, "__test.ts", `
		fs.writeFileSync("stat-target.txt", "12345");
		var s = fs.statSync("stat-target.txt");
		return JSON.stringify({size: s.size, isDir: s.isDirectory()});
	`)
	require.NoError(t, err)
	var stat struct {
		Size  int64 `json:"size"`
		IsDir bool  `json:"isDir"`
	}
	json.Unmarshal([]byte(result), &stat)
	assert.False(t, stat.IsDir)
	assert.Equal(t, int64(5), stat.Size)
}

func testStatDirectory(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := env.Kernel.EvalTS(ctx, "__test.ts", `
		fs.mkdirSync("stat-dir-fs", {recursive: true});
		var s = fs.statSync("stat-dir-fs");
		return JSON.stringify({isDir: s.isDirectory()});
	`)
	require.NoError(t, err)
	var stat struct {
		IsDir bool `json:"isDir"`
	}
	json.Unmarshal([]byte(result), &stat)
	assert.True(t, stat.IsDir)
}

func testDelete(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := env.Kernel.EvalTS(ctx, "__test.ts", `
		fs.writeFileSync("delete-me.txt", "x");
		fs.unlinkSync("delete-me.txt");
		try { fs.readFileSync("delete-me.txt", "utf8"); return "SHOULD_FAIL"; }
		catch(e) { return e.code || "error"; }
	`)
	require.NoError(t, err)
	assert.NotEqual(t, "SHOULD_FAIL", result)
}

func testDeleteNotFound(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := env.Kernel.EvalTS(ctx, "__test.ts", `
		try { fs.unlinkSync("ghost.txt"); return "SHOULD_FAIL"; }
		catch(e) { return e.code || "error"; }
	`)
	require.NoError(t, err)
	assert.NotEqual(t, "SHOULD_FAIL", result)
}

func testReadNotFound(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := env.Kernel.EvalTS(ctx, "__test.ts", `
		try { fs.readFileSync("nope.txt", "utf8"); return "SHOULD_FAIL"; }
		catch(e) { return e.code || "error"; }
	`)
	require.NoError(t, err)
	assert.NotEqual(t, "SHOULD_FAIL", result)
}

func testPathTraversalRejected(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := env.Kernel.EvalTS(ctx, "__test.ts", `
		try { fs.readFileSync("../../etc/passwd"); return "SHOULD_FAIL"; }
		catch(e) { return e.code || "error"; }
	`)
	require.NoError(t, err)
	assert.NotEqual(t, "SHOULD_FAIL", result)
}

func testLargeFileWrite(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := env.Kernel.EvalTS(ctx, "__test.ts", `
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

func testFSFromTS(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()
	err := env.Deploy("fs-ts-test.ts", `
		fs.writeFileSync("ts-file.txt", "from typescript");
		var data = fs.readFileSync("ts-file.txt", "utf8");
		var stat = fs.statSync("ts-file.txt");
		var list = fs.readdirSync(".");
		fs.unlinkSync("ts-file.txt");
		output({
			data: data, size: stat.size, filesFound: list.length, deleted: true,
		});
	`)
	require.NoError(t, err)

	result, _ := env.Kernel.EvalTS(ctx, "__fs_ts.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || {});
	`)
	assert.Contains(t, result, "from typescript")
	assert.Contains(t, result, `"deleted":true`)
}
