package infra_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoDirect_FS(t *testing.T) {
	for _, factory := range []struct {
		name string
		make func(t *testing.T) *testutil.TestKernel
	}{
		{"Kernel", testutil.NewTestKernelFull},
	} {
		t.Run(factory.name, func(t *testing.T) {
			tk := factory.make(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			t.Run("Write_Read_Roundtrip", func(t *testing.T) {
				result, err := tk.EvalTS(ctx, "__test.ts", `fs.writeFileSync("test.txt", "hello fs"); return fs.readFileSync("test.txt", "utf8");`)
				require.NoError(t, err)
				assert.Equal(t, "hello fs", result)
			})

			t.Run("Write_Overwrite", func(t *testing.T) {
				result, err := tk.EvalTS(ctx, "__test.ts", `
					fs.writeFileSync("overwrite.txt", "v1");
					fs.writeFileSync("overwrite.txt", "v2");
					return fs.readFileSync("overwrite.txt", "utf8");
				`)
				require.NoError(t, err)
				assert.Equal(t, "v2", result)
			})

			t.Run("Mkdir_Recursive", func(t *testing.T) {
				result, err := tk.EvalTS(ctx, "__test.ts", `
					fs.mkdirSync("a/b/c", {recursive: true});
					fs.writeFileSync("a/b/c/deep.txt", "deep");
					return fs.readFileSync("a/b/c/deep.txt", "utf8");
				`)
				require.NoError(t, err)
				assert.Equal(t, "deep", result)
			})

			t.Run("List_WithPattern", func(t *testing.T) {
				result, err := tk.EvalTS(ctx, "__test.ts", `
					fs.mkdirSync("listdir", {recursive: true});
					fs.writeFileSync("listdir/a.txt", "a");
					fs.writeFileSync("listdir/b.json", "{}");
					fs.writeFileSync("listdir/c.txt", "c");
					var files = fs.readdirSync("listdir");
					return JSON.stringify(files);
				`)
				require.NoError(t, err)
				var files []string
				json.Unmarshal([]byte(result), &files)
				assert.Len(t, files, 3) // a.txt, b.json, c.txt — readdirSync lists all
			})

			t.Run("Stat_File", func(t *testing.T) {
				result, err := tk.EvalTS(ctx, "__test.ts", `
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
			})

			t.Run("Stat_Directory", func(t *testing.T) {
				result, err := tk.EvalTS(ctx, "__test.ts", `
					fs.mkdirSync("stat-dir", {recursive: true});
					var s = fs.statSync("stat-dir");
					return JSON.stringify({isDir: s.isDirectory()});
				`)
				require.NoError(t, err)
				var stat struct {
					IsDir bool `json:"isDir"`
				}
				json.Unmarshal([]byte(result), &stat)
				assert.True(t, stat.IsDir)
			})

			t.Run("Delete", func(t *testing.T) {
				result, err := tk.EvalTS(ctx, "__test.ts", `
					fs.writeFileSync("delete-me.txt", "x");
					fs.unlinkSync("delete-me.txt");
					try { fs.readFileSync("delete-me.txt", "utf8"); return "SHOULD_FAIL"; }
					catch(e) { return e.code || "error"; }
				`)
				require.NoError(t, err)
				assert.NotEqual(t, "SHOULD_FAIL", result)
			})

			t.Run("Delete_NotFound", func(t *testing.T) {
				result, err := tk.EvalTS(ctx, "__test.ts", `
					try { fs.unlinkSync("ghost.txt"); return "SHOULD_FAIL"; }
					catch(e) { return e.code || "error"; }
				`)
				require.NoError(t, err)
				assert.NotEqual(t, "SHOULD_FAIL", result)
			})

			t.Run("Read_NotFound", func(t *testing.T) {
				result, err := tk.EvalTS(ctx, "__test.ts", `
					try { fs.readFileSync("nope.txt", "utf8"); return "SHOULD_FAIL"; }
					catch(e) { return e.code || "error"; }
				`)
				require.NoError(t, err)
				assert.NotEqual(t, "SHOULD_FAIL", result)
			})

			t.Run("PathTraversal_Rejected", func(t *testing.T) {
				result, err := tk.EvalTS(ctx, "__test.ts", `
					try { fs.readFileSync("../../etc/passwd"); return "SHOULD_FAIL"; }
					catch(e) { return e.code || "error"; }
				`)
				require.NoError(t, err)
				assert.NotEqual(t, "SHOULD_FAIL", result)
			})
		})
	}
}
