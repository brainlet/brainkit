package fs

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("fs", func(t *testing.T) {
		t.Run("write_read_roundtrip", func(t *testing.T) { testWriteReadRoundtrip(t, env) })
		t.Run("write_overwrite", func(t *testing.T) { testWriteOverwrite(t, env) })
		t.Run("mkdir_recursive", func(t *testing.T) { testMkdirRecursive(t, env) })
		t.Run("stat_file", func(t *testing.T) { testStatFile(t, env) })
		t.Run("stat_directory", func(t *testing.T) { testStatDirectory(t, env) })
		t.Run("delete", func(t *testing.T) { testDelete(t, env) })
		t.Run("delete_not_found", func(t *testing.T) { testDeleteNotFound(t, env) })
		t.Run("read_not_found", func(t *testing.T) { testReadNotFound(t, env) })
		t.Run("path_traversal_rejected", func(t *testing.T) { testPathTraversalRejected(t, env) })
		t.Run("large_file_write", func(t *testing.T) { testLargeFileWrite(t, env) })
		t.Run("list_with_pattern", func(t *testing.T) { testFSListWithPattern(t, env) })
		t.Run("fs_from_ts", func(t *testing.T) { testFSFromTS(t, env) })
	})
}
