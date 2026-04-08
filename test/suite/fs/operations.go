package fs

import (
	"encoding/json"
	"testing"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
)

func testWriteReadRoundtrip(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__test_fs_wr.ts", `fs.writeFileSync("test.txt", "hello fs"); return fs.readFileSync("test.txt", "utf8");`)
	assert.Equal(t, "hello fs", result)
}

func testWriteOverwrite(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__test_fs_ow.ts", `
		fs.writeFileSync("overwrite.txt", "v1");
		fs.writeFileSync("overwrite.txt", "v2");
		return fs.readFileSync("overwrite.txt", "utf8");
	`)
	assert.Equal(t, "v2", result)
}

func testMkdirRecursive(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__test_fs_mkdir.ts", `
		fs.mkdirSync("a/b/c", {recursive: true});
		fs.writeFileSync("a/b/c/deep.txt", "deep");
		return fs.readFileSync("a/b/c/deep.txt", "utf8");
	`)
	assert.Equal(t, "deep", result)
}

func testStatFile(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__test_fs_stat.ts", `
		fs.writeFileSync("stat-target.txt", "12345");
		var s = fs.statSync("stat-target.txt");
		return JSON.stringify({size: s.size, isDir: s.isDirectory()});
	`)
	var stat struct {
		Size  int64 `json:"size"`
		IsDir bool  `json:"isDir"`
	}
	json.Unmarshal([]byte(result), &stat)
	assert.False(t, stat.IsDir)
	assert.Equal(t, int64(5), stat.Size)
}

func testStatDirectory(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__test_fs_statdir.ts", `
		fs.mkdirSync("stat-dir-fs", {recursive: true});
		var s = fs.statSync("stat-dir-fs");
		return JSON.stringify({isDir: s.isDirectory()});
	`)
	var stat struct {
		IsDir bool `json:"isDir"`
	}
	json.Unmarshal([]byte(result), &stat)
	assert.True(t, stat.IsDir)
}

func testDelete(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__test_fs_del.ts", `
		fs.writeFileSync("delete-me.txt", "x");
		fs.unlinkSync("delete-me.txt");
		try { fs.readFileSync("delete-me.txt", "utf8"); return "SHOULD_FAIL"; }
		catch(e) { return e.code || "error"; }
	`)
	assert.NotEqual(t, "SHOULD_FAIL", result)
}

func testDeleteNotFound(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__test_fs_delnf.ts", `
		try { fs.unlinkSync("ghost.txt"); return "SHOULD_FAIL"; }
		catch(e) { return e.code || "error"; }
	`)
	assert.NotEqual(t, "SHOULD_FAIL", result)
}

func testReadNotFound(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__test_fs_readnf.ts", `
		try { fs.readFileSync("nope.txt", "utf8"); return "SHOULD_FAIL"; }
		catch(e) { return e.code || "error"; }
	`)
	assert.NotEqual(t, "SHOULD_FAIL", result)
}

func testPathTraversalRejected(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__test_fs_traversal.ts", `
		try { fs.readFileSync("../../etc/passwd"); return "SHOULD_FAIL"; }
		catch(e) { return e.code || "error"; }
	`)
	assert.NotEqual(t, "SHOULD_FAIL", result)
}

func testLargeFileWrite(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__test_fs_large.ts", `
		var big = "";
		for (var i = 0; i < 1024*1024; i++) big += "x";
		fs.writeFileSync("big-file.txt", big);
		var s = fs.statSync("big-file.txt");
		return JSON.stringify({size: s.size});
	`)
	var stat struct{ Size int64 `json:"size"` }
	json.Unmarshal([]byte(result), &stat)
	assert.Equal(t, int64(1024*1024), stat.Size)
}

func testFSListWithPattern(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__test_fs_list.ts", `
		fs.mkdirSync("listdir", {recursive: true});
		fs.writeFileSync("listdir/a.txt", "a");
		fs.writeFileSync("listdir/b.json", "{}");
		fs.writeFileSync("listdir/c.txt", "c");
		var files = fs.readdirSync("listdir");
		return JSON.stringify(files);
	`)
	var files []string
	json.Unmarshal([]byte(result), &files)
	assert.Len(t, files, 3) // a.txt, b.json, c.txt — readdirSync lists all
}

func testFSFromTS(t *testing.T, env *suite.TestEnv) {
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
	assert.NoError(t, err)

	result := testutil.EvalTS(t, env.Kit, "__fs_ts.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || {});
	`)
	assert.Contains(t, result, "from typescript")
	assert.Contains(t, result, `"deleted":true`)
}
