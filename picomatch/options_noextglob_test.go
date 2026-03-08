// options_noextglob_test.go — Faithful 1:1 port of picomatch/test/options.noextglob.js
package picomatch

import "testing"

func TestOptionsNoextglob(t *testing.T) {
	t.Run("options.noextglob", func(t *testing.T) {
		t.Run("should disable extglob support when options.noextglob is true", func(t *testing.T) {
			// options.noextglob.js line 8
			assertMatch(t, true, "a+z", "a+(z)", &Options{Noextglob: true})
			// options.noextglob.js line 9
			assertMatch(t, false, "az", "a+(z)", &Options{Noextglob: true})
			// options.noextglob.js line 10
			assertMatch(t, false, "azz", "a+(z)", &Options{Noextglob: true})
			// options.noextglob.js line 11
			assertMatch(t, false, "azzz", "a+(z)", &Options{Noextglob: true})
		})

		t.Run("should work with noext alias to support minimatch", func(t *testing.T) {
			// options.noextglob.js line 15
			assertMatch(t, true, "a+z", "a+(z)", &Options{Noext: true})
			// options.noextglob.js line 16
			assertMatch(t, false, "az", "a+(z)", &Options{Noext: true})
			// options.noextglob.js line 17
			assertMatch(t, false, "azz", "a+(z)", &Options{Noext: true})
			// options.noextglob.js line 18
			assertMatch(t, false, "azzz", "a+(z)", &Options{Noext: true})
		})
	})
}
