// malicious_test.go — Faithful 1:1 port of picomatch/test/malicious.js
package picomatch

import (
	"fmt"
	"strings"
	"testing"
)

// repeat produces a string of n backslashes, matching the JS helper:
//
//	const repeat = n => '\\'.repeat(n);
//
// malicious.js line 5
func repeat(n int) string {
	return strings.Repeat(`\`, n)
}

func TestMalicious(t *testing.T) {
	t.Run("handling of potential regex exploits", func(t *testing.T) {
		t.Run("should support long escape sequences", func(t *testing.T) {
			// malicious.js line 13-14: skip win32 check, test on all platforms
			// On non-Windows, isMatch("\\A", repeat(65500) + "A") should be true.
			// In Go, backslash is not a path separator, so this should work.
			// malicious.js line 14
			assertMatch(t, true, `\A`, repeat(65500)+"A")

			// malicious.js line 16
			assertMatch(t, true, "A", "!"+repeat(65500)+"A")
			// malicious.js line 17
			assertMatch(t, true, "A", "!("+repeat(65500)+"A)")
			// malicious.js line 18
			assertMatch(t, false, "A", "[!("+repeat(65500)+"A")
		})

		t.Run("should throw an error when the pattern is too long", func(t *testing.T) {
			// malicious.js line 22: isMatch('foo', '*'.repeat(65537)) should panic
			func() {
				defer func() {
					r := recover()
					if r == nil {
						t.Errorf("expected panic for pattern length 65537")
					}
					s := fmt.Sprintf("%v", r)
					if !strings.Contains(s, "exceeds maximum allowed") {
						t.Errorf("expected panic message to contain %q, got %q", "exceeds maximum allowed", s)
					}
				}()
				// malicious.js line 22
				IsMatch("foo", strings.Repeat("*", 65537), nil)
			}()

			// malicious.js line 23-25: isMatch('A', '!(' + repeat(65536) + 'A)') should panic
			func() {
				defer func() {
					r := recover()
					if r == nil {
						t.Errorf("expected panic for pattern length 65540")
					}
					s := fmt.Sprintf("%v", r)
					// malicious.js line 25: /Input length: 65540, exceeds maximum allowed length: 65536/
					if !strings.Contains(s, "Input length: 65540, exceeds maximum allowed length: 65536") {
						t.Errorf("expected exact panic message, got %q", s)
					}
				}()
				// malicious.js line 24
				IsMatch("A", "!("+repeat(65536)+"A)", nil)
			}()
		})

		t.Run("should allow max bytes to be customized", func(t *testing.T) {
			// malicious.js line 29-31
			func() {
				defer func() {
					r := recover()
					if r == nil {
						t.Errorf("expected panic for custom maxLength 499")
					}
					s := fmt.Sprintf("%v", r)
					// malicious.js line 31: /Input length: 504, exceeds maximum allowed length: 499/
					if !strings.Contains(s, "Input length: 504, exceeds maximum allowed length: 499") {
						t.Errorf("expected exact panic message, got %q", s)
					}
				}()
				// malicious.js line 30
				IsMatch("A", "!("+repeat(500)+"A)", &Options{MaxLength: 499})
			}()
		})

		t.Run("should be able to accept Object instance properties", func(t *testing.T) {
			// malicious.js line 34
			assertMatch(t, true, "constructor", "constructor")
			// malicious.js line 35
			assertMatch(t, true, "__proto__", "__proto__")
			// malicious.js line 36
			assertMatch(t, true, "toString", "toString")
		})
	})
}
