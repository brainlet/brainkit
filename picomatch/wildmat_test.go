// wildmat_test.go — Faithful 1:1 port of picomatch/test/wildmat.js
package picomatch

import (
	"testing"
)

func TestWildmat(t *testing.T) {
	t.Run("Wildmat (git) tests", func(t *testing.T) {
		t.Run("Basic wildmat features", func(t *testing.T) {
			// wildmat.js line 8
			assertMatch(t, false, "foo", "*f")
			// wildmat.js line 9
			assertMatch(t, false, "foo", "??")
			// wildmat.js line 10
			assertMatch(t, false, "foo", "bar")
			// wildmat.js line 11
			assertMatch(t, false, "foobar", "foo\\*bar")
			// wildmat.js line 12
			assertMatch(t, true, "?a?b", "\\??\\?b")
			// wildmat.js line 13
			assertMatch(t, true, "aaaaaaabababab", "*ab")
			// wildmat.js line 14
			assertMatch(t, true, "foo", "*")
			// wildmat.js line 15
			assertMatch(t, true, "foo", "*foo*")
			// wildmat.js line 16
			assertMatch(t, true, "foo", "???")
			// wildmat.js line 17
			assertMatch(t, true, "foo", "f*")
			// wildmat.js line 18
			assertMatch(t, true, "foo", "foo")
			// wildmat.js line 19
			assertMatch(t, true, "foobar", "*ob*a*r*")
		})

		t.Run("should support recursion", func(t *testing.T) {
			// wildmat.js line 23
			assertMatch(t, false, "-adobe-courier-bold-o-normal--12-120-75-75-/-70-iso8859-1", "-*-*-*-*-*-*-12-*-*-*-m-*-*-*")
			// wildmat.js line 24
			assertMatch(t, false, "-adobe-courier-bold-o-normal--12-120-75-75-X-70-iso8859-1", "-*-*-*-*-*-*-12-*-*-*-m-*-*-*")
			// wildmat.js line 25
			assertMatch(t, false, "ab/cXd/efXg/hi", "*X*i")
			// wildmat.js line 26
			assertMatch(t, false, "ab/cXd/efXg/hi", "*Xg*i")
			// wildmat.js line 27
			assertMatch(t, false, "abcd/abcdefg/abcdefghijk/abcdefghijklmnop.txtz", "**/*a*b*g*n*t")
			// wildmat.js line 28
			assertMatch(t, false, "foo", "*/*/*")
			// wildmat.js line 29
			assertMatch(t, false, "foo", "fo")
			// wildmat.js line 30
			assertMatch(t, false, "foo/bar", "*/*/*")
			// wildmat.js line 31
			assertMatch(t, false, "foo/bar", "foo?bar")
			// wildmat.js line 32
			assertMatch(t, false, "foo/bb/aa/rr", "*/*/*")
			// wildmat.js line 33
			assertMatch(t, false, "foo/bba/arr", "foo*")
			// wildmat.js line 34
			assertMatch(t, false, "foo/bba/arr", "foo**")
			// wildmat.js line 35
			assertMatch(t, false, "foo/bba/arr", "foo/*")
			// wildmat.js line 36
			assertMatch(t, false, "foo/bba/arr", "foo/**arr")
			// wildmat.js line 37
			assertMatch(t, false, "foo/bba/arr", "foo/**z")
			// wildmat.js line 38
			assertMatch(t, false, "foo/bba/arr", "foo/*arr")
			// wildmat.js line 39
			assertMatch(t, false, "foo/bba/arr", "foo/*z")
			// wildmat.js line 40
			assertMatch(t, false, "XXX/adobe/courier/bold/o/normal//12/120/75/75/X/70/iso8859/1", "XXX/*/*/*/*/*/*/12/*/*/*/m/*/*/*")
			// wildmat.js line 41
			assertMatch(t, true, "-adobe-courier-bold-o-normal--12-120-75-75-m-70-iso8859-1", "-*-*-*-*-*-*-12-*-*-*-m-*-*-*")
			// wildmat.js line 42
			assertMatch(t, true, "ab/cXd/efXg/hi", "**/*X*/**/*i")
			// wildmat.js line 43
			assertMatch(t, true, "ab/cXd/efXg/hi", "*/*X*/*/*i")
			// wildmat.js line 44
			assertMatch(t, true, "abcd/abcdefg/abcdefghijk/abcdefghijklmnop.txt", "**/*a*b*g*n*t")
			// wildmat.js line 45
			assertMatch(t, true, "abcXdefXghi", "*X*i")
			// wildmat.js line 46
			assertMatch(t, true, "foo", "foo")
			// wildmat.js line 47
			assertMatch(t, true, "foo/bar", "foo/*")
			// wildmat.js line 48
			assertMatch(t, true, "foo/bar", "foo/bar")
			// wildmat.js line 49
			assertMatch(t, true, "foo/bar", "foo[/]bar")
			// wildmat.js line 50
			assertMatch(t, true, "foo/bb/aa/rr", "**/**/**")
			// wildmat.js line 51
			assertMatch(t, true, "foo/bba/arr", "*/*/*")
			// wildmat.js line 52
			assertMatch(t, true, "foo/bba/arr", "foo/**")
		})
	})
}
