// api_posix_test.go — Faithful 1:1 port of picomatch/test/api.posix.js
package picomatch

import (
	"testing"
)

func TestPicomatchPosix(t *testing.T) {
	t.Run("should use posix paths only by default", func(t *testing.T) {
		// api.posix.js line 8
		matcher := CompilePosix("a/**", nil)
		// api.posix.js line 9
		if !matcher("a/b") {
			t.Errorf("CompilePosix(%q)(\"a/b\") should be true", "a/**")
		}
		// api.posix.js line 10
		// In posix mode, backslash is NOT a path separator — it's an escape char.
		// So "a\\b" should NOT match "a/**" in posix mode.
		if matcher("a\\b") {
			t.Errorf("CompilePosix(%q)(\"a\\\\b\") should be false in posix mode", "a/**")
		}
	})

	t.Run("should still be manually configurable to accept non-posix paths", func(t *testing.T) {
		// api.posix.js line 13
		matcher := CompilePosix("a/**", &Options{Windows: true})
		// api.posix.js line 14
		if !matcher("a\\b") {
			t.Errorf("CompilePosix(%q, {Windows:true})(\"a\\\\b\") should be true", "a/**")
		}
		// api.posix.js line 15
		if !matcher("a/b") {
			t.Errorf("CompilePosix(%q, {Windows:true})(\"a/b\") should be true", "a/**")
		}
	})
}
