// options_noglobstar_test.go — Faithful 1:1 port of picomatch/test/options.noglobstar.js
package picomatch

import "testing"

func TestOptionsNoglobstar(t *testing.T) {
	t.Run("options.noglobstar", func(t *testing.T) {
		t.Run("should disable extglob support when options.noglobstar is true", func(t *testing.T) {
			// options.noglobstar.js line 8
			assertMatch(t, true, "a/b/c", "**", &Options{Noglobstar: false})
			// options.noglobstar.js line 9
			assertMatch(t, false, "a/b/c", "**", &Options{Noglobstar: true})
			// options.noglobstar.js line 10
			assertMatch(t, true, "a/b/c", "a/**", &Options{Noglobstar: false})
			// options.noglobstar.js line 11
			assertMatch(t, false, "a/b/c", "a/**", &Options{Noglobstar: true})
		})
	})
}
