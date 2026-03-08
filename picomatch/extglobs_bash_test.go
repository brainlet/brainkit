package picomatch

// extglobs_bash_test.go — Faithfully ported from picomatch/test/extglobs-bash.js
// Some of tests were converted from bash 4.3, 4.4, and minimatch unit tests.
// Every test case includes the original source file and line number.

import (
	"testing"
)

var (
	// bashOpts — common options used across most extglob bash tests: { bash: true, windows: true }
	bashOpts = &Options{Bash: true, Windows: true}
	// bashNonWinOpts — { bash: true, windows: false }
	bashNonWinOpts = &Options{Bash: true}
	// winOpts — { windows: true }
	winOpts = &Options{Windows: true}
)

func TestExtglobsBash(t *testing.T) {
	t.Run("should not match empty string with *(0|1|3|5|7|9)", func(t *testing.T) {
		// extglobs-bash.js:12
		assertMatch(t, false, "", "*(0|1|3|5|7|9)", bashOpts)
	})

	t.Run("*(a|b[) should not match *(a|b\\\\\\\\[)", func(t *testing.T) {
		// extglobs-bash.js:16
		assertMatch(t, false, "*(a|b[)", "*(a|b\\[)", bashOpts)
	})

	t.Run("*(a|b[) should not match \\\\\\\\*\\\\\\\\(a|b\\\\\\\\[\\\\\\\\)", func(t *testing.T) {
		// extglobs-bash.js:20
		assertMatch(t, false, "*(a|b[)", "\\*\\(a|b\\[\\)", bashOpts)
	})

	t.Run("*** should match \\\\\\\\*\\\\\\\\*\\\\\\\\*", func(t *testing.T) {
		// extglobs-bash.js:24
		assertMatch(t, true, "***", "\\*\\*\\*", bashOpts)
	})

	t.Run("-adobe-courier-bold-o-normal--12-120-75-75-/-70-iso8859-1 should not match -*-*-*-*-*-*-12-*-*-*-m-*-*-*", func(t *testing.T) {
		// extglobs-bash.js:28
		assertMatch(t, false, "-adobe-courier-bold-o-normal--12-120-75-75-/-70-iso8859-1", "-*-*-*-*-*-*-12-*-*-*-m-*-*-*", bashOpts)
	})

	t.Run("-adobe-courier-bold-o-normal--12-120-75-75-m-70-iso8859-1 should match -*-*-*-*-*-*-12-*-*-*-m-*-*-*", func(t *testing.T) {
		// extglobs-bash.js:32
		assertMatch(t, true, "-adobe-courier-bold-o-normal--12-120-75-75-m-70-iso8859-1", "-*-*-*-*-*-*-12-*-*-*-m-*-*-*", bashOpts)
	})

	t.Run("-adobe-courier-bold-o-normal--12-120-75-75-X-70-iso8859-1 should not match -*-*-*-*-*-*-12-*-*-*-m-*-*-*", func(t *testing.T) {
		// extglobs-bash.js:36
		assertMatch(t, false, "-adobe-courier-bold-o-normal--12-120-75-75-X-70-iso8859-1", "-*-*-*-*-*-*-12-*-*-*-m-*-*-*", bashOpts)
	})

	t.Run("/dev/udp/129.22.8.102/45 should match /dev\\\\\\\\/@(tcp|udp)\\\\\\\\/*\\\\\\\\/*", func(t *testing.T) {
		// extglobs-bash.js:40
		assertMatch(t, true, "/dev/udp/129.22.8.102/45", "/dev\\/@(tcp|udp)\\/*\\/*", bashOpts)
	})

	t.Run("/x/y/z should match /x/y/z", func(t *testing.T) {
		// extglobs-bash.js:44
		assertMatch(t, true, "/x/y/z", "/x/y/z", bashOpts)
	})

	t.Run("0377 should match +([0-7])", func(t *testing.T) {
		// extglobs-bash.js:48
		assertMatch(t, true, "0377", "+([0-7])", bashOpts)
	})

	t.Run("07 should match +([0-7])", func(t *testing.T) {
		// extglobs-bash.js:52
		assertMatch(t, true, "07", "+([0-7])", bashOpts)
	})

	t.Run("09 should not match +([0-7])", func(t *testing.T) {
		// extglobs-bash.js:56
		assertMatch(t, false, "09", "+([0-7])", bashOpts)
	})

	t.Run("1 should match 0|[1-9]*([0-9])", func(t *testing.T) {
		// extglobs-bash.js:60
		assertMatch(t, true, "1", "0|[1-9]*([0-9])", bashOpts)
	})

	t.Run("12 should match 0|[1-9]*([0-9])", func(t *testing.T) {
		// extglobs-bash.js:64
		assertMatch(t, true, "12", "0|[1-9]*([0-9])", bashOpts)
	})

	t.Run("123abc should not match (a+|b)*", func(t *testing.T) {
		// extglobs-bash.js:68
		assertMatch(t, false, "123abc", "(a+|b)*", bashOpts)
	})

	t.Run("123abc should not match (a+|b)+", func(t *testing.T) {
		// extglobs-bash.js:72
		assertMatch(t, false, "123abc", "(a+|b)+", bashOpts)
	})

	t.Run("123abc should match *?(a)bc", func(t *testing.T) {
		// extglobs-bash.js:76
		assertMatch(t, true, "123abc", "*?(a)bc", bashOpts)
	})

	t.Run("123abc should not match a(b*(foo|bar))d", func(t *testing.T) {
		// extglobs-bash.js:80
		assertMatch(t, false, "123abc", "a(b*(foo|bar))d", bashOpts)
	})

	t.Run("123abc should not match ab*(e|f)", func(t *testing.T) {
		// extglobs-bash.js:84
		assertMatch(t, false, "123abc", "ab*(e|f)", bashOpts)
	})

	t.Run("123abc should not match ab**", func(t *testing.T) {
		// extglobs-bash.js:88
		assertMatch(t, false, "123abc", "ab**", bashOpts)
	})

	t.Run("123abc should not match ab**(e|f)", func(t *testing.T) {
		// extglobs-bash.js:92
		assertMatch(t, false, "123abc", "ab**(e|f)", bashOpts)
	})

	t.Run("123abc should not match ab**(e|f)g", func(t *testing.T) {
		// extglobs-bash.js:96
		assertMatch(t, false, "123abc", "ab**(e|f)g", bashOpts)
	})

	t.Run("123abc should not match ab***ef", func(t *testing.T) {
		// extglobs-bash.js:100
		assertMatch(t, false, "123abc", "ab***ef", bashOpts)
	})

	t.Run("123abc should not match ab*+(e|f)", func(t *testing.T) {
		// extglobs-bash.js:104
		assertMatch(t, false, "123abc", "ab*+(e|f)", bashOpts)
	})

	t.Run("123abc should not match ab*d+(e|f)", func(t *testing.T) {
		// extglobs-bash.js:108
		assertMatch(t, false, "123abc", "ab*d+(e|f)", bashOpts)
	})

	t.Run("123abc should not match ab?*(e|f)", func(t *testing.T) {
		// extglobs-bash.js:112
		assertMatch(t, false, "123abc", "ab?*(e|f)", bashOpts)
	})

	t.Run("12abc should not match 0|[1-9]*([0-9])", func(t *testing.T) {
		// extglobs-bash.js:116
		assertMatch(t, false, "12abc", "0|[1-9]*([0-9])", bashOpts)
	})

	t.Run("137577991 should match *(0|1|3|5|7|9)", func(t *testing.T) {
		// extglobs-bash.js:120
		assertMatch(t, true, "137577991", "*(0|1|3|5|7|9)", bashOpts)
	})

	t.Run("2468 should not match *(0|1|3|5|7|9)", func(t *testing.T) {
		// extglobs-bash.js:124
		assertMatch(t, false, "2468", "*(0|1|3|5|7|9)", bashOpts)
	})

	t.Run("?a?b should match \\\\\\\\??\\\\\\\\?b", func(t *testing.T) {
		// extglobs-bash.js:128
		assertMatch(t, true, "?a?b", "\\??\\?b", bashOpts)
	})

	t.Run("\\\\\\\\a\\\\\\\\b\\\\\\\\c should not match abc", func(t *testing.T) {
		// extglobs-bash.js:132
		assertMatch(t, false, "\\a\\b\\c", "abc", bashOpts)
	})

	t.Run("a should match !(*.a|*.b|*.c)", func(t *testing.T) {
		// extglobs-bash.js:136
		assertMatch(t, true, "a", "!(*.a|*.b|*.c)", bashOpts)
	})

	t.Run("a should not match !(a)", func(t *testing.T) {
		// extglobs-bash.js:140
		assertMatch(t, false, "a", "!(a)", bashOpts)
	})

	t.Run("a should not match !(a)*", func(t *testing.T) {
		// extglobs-bash.js:144
		assertMatch(t, false, "a", "!(a)*", bashOpts)
	})

	t.Run("a should match (a)", func(t *testing.T) {
		// extglobs-bash.js:148
		assertMatch(t, true, "a", "(a)", bashOpts)
	})

	t.Run("a should not match (b)", func(t *testing.T) {
		// extglobs-bash.js:152
		assertMatch(t, false, "a", "(b)", bashOpts)
	})

	t.Run("a should match *(a)", func(t *testing.T) {
		// extglobs-bash.js:156
		assertMatch(t, true, "a", "*(a)", bashOpts)
	})

	t.Run("a should match +(a)", func(t *testing.T) {
		// extglobs-bash.js:160
		assertMatch(t, true, "a", "+(a)", bashOpts)
	})

	t.Run("a should match ?", func(t *testing.T) {
		// extglobs-bash.js:164
		assertMatch(t, true, "a", "?", bashOpts)
	})

	t.Run("a should match ?(a|b)", func(t *testing.T) {
		// extglobs-bash.js:168
		assertMatch(t, true, "a", "?(a|b)", bashOpts)
	})

	t.Run("a should not match ??", func(t *testing.T) {
		// extglobs-bash.js:172
		assertMatch(t, false, "a", "??", bashOpts)
	})

	t.Run("a should match a!(b)*", func(t *testing.T) {
		// extglobs-bash.js:176
		assertMatch(t, true, "a", "a!(b)*", bashOpts)
	})

	t.Run("a should match a?(a|b)", func(t *testing.T) {
		// extglobs-bash.js:180
		assertMatch(t, true, "a", "a?(a|b)", bashOpts)
	})

	t.Run("a should match a?(x)", func(t *testing.T) {
		// extglobs-bash.js:184
		assertMatch(t, true, "a", "a?(x)", bashOpts)
	})

	t.Run("a should not match a??b", func(t *testing.T) {
		// extglobs-bash.js:188
		assertMatch(t, false, "a", "a??b", bashOpts)
	})

	t.Run("a should not match b?(a|b)", func(t *testing.T) {
		// extglobs-bash.js:192
		assertMatch(t, false, "a", "b?(a|b)", bashOpts)
	})

	t.Run("a((((b should match a(*b", func(t *testing.T) {
		// extglobs-bash.js:196
		assertMatch(t, true, "a((((b", "a(*b", bashOpts)
	})

	t.Run("a((((b should not match a(b", func(t *testing.T) {
		// extglobs-bash.js:200
		assertMatch(t, false, "a((((b", "a(b", bashOpts)
	})

	t.Run("a((((b should not match a\\\\\\\\(b", func(t *testing.T) {
		// extglobs-bash.js:204
		assertMatch(t, false, "a((((b", "a\\(b", bashOpts)
	})

	t.Run("a((b should match a(*b", func(t *testing.T) {
		// extglobs-bash.js:208
		assertMatch(t, true, "a((b", "a(*b", bashOpts)
	})

	t.Run("a((b should not match a(b", func(t *testing.T) {
		// extglobs-bash.js:212
		assertMatch(t, false, "a((b", "a(b", bashOpts)
	})

	t.Run("a((b should not match a\\\\\\\\(b", func(t *testing.T) {
		// extglobs-bash.js:216
		assertMatch(t, false, "a((b", "a\\(b", bashOpts)
	})

	t.Run("a(b should match a(*b", func(t *testing.T) {
		// extglobs-bash.js:220
		assertMatch(t, true, "a(b", "a(*b", bashOpts)
	})

	t.Run("a(b should match a(b", func(t *testing.T) {
		// extglobs-bash.js:224
		assertMatch(t, true, "a(b", "a(b", bashOpts)
	})

	t.Run("a\\\\\\\\(b should match a\\\\\\\\(b", func(t *testing.T) {
		// extglobs-bash.js:228
		assertMatch(t, true, "a\\(b", "a\\(b", bashOpts)
	})

	t.Run("a(b should match a\\\\\\\\(b", func(t *testing.T) {
		// extglobs-bash.js:232
		assertMatch(t, true, "a(b", "a\\(b", bashOpts)
	})

	t.Run("a. should match !(*.a|*.b|*.c)", func(t *testing.T) {
		// extglobs-bash.js:236
		assertMatch(t, true, "a.", "!(*.a|*.b|*.c)", bashOpts)
	})

	t.Run("a. should match *!(.a|.b|.c)", func(t *testing.T) {
		// extglobs-bash.js:240
		assertMatch(t, true, "a.", "*!(.a|.b|.c)", bashOpts)
	})

	t.Run("a. should match *.!(a)", func(t *testing.T) {
		// extglobs-bash.js:244
		assertMatch(t, true, "a.", "*.!(a)", bashOpts)
	})

	t.Run("a. should match *.!(a|b|c)", func(t *testing.T) {
		// extglobs-bash.js:248
		assertMatch(t, true, "a.", "*.!(a|b|c)", bashOpts)
	})

	t.Run("a. should not match *.(a|b|@(ab|a*@(b))*(c)d)", func(t *testing.T) {
		// extglobs-bash.js:252
		assertMatch(t, false, "a.", "*.(a|b|@(ab|a*@(b))*(c)d)", bashOpts)
	})

	t.Run("a. should not match *.+(b|d)", func(t *testing.T) {
		// extglobs-bash.js:256
		assertMatch(t, false, "a.", "*.+(b|d)", bashOpts)
	})

	t.Run("a.a should not match !(*.[a-b]*)", func(t *testing.T) {
		// extglobs-bash.js:260
		assertMatch(t, false, "a.a", "!(*.[a-b]*)", bashOpts)
	})

	t.Run("a.a should not match !(*.a|*.b|*.c)", func(t *testing.T) {
		// extglobs-bash.js:264
		assertMatch(t, false, "a.a", "!(*.a|*.b|*.c)", bashOpts)
	})

	t.Run("a.a should not match !(*[a-b].[a-b]*)", func(t *testing.T) {
		// extglobs-bash.js:268
		assertMatch(t, false, "a.a", "!(*[a-b].[a-b]*)", bashOpts)
	})

	t.Run("a.a should not match !*.(a|b)", func(t *testing.T) {
		// extglobs-bash.js:272
		assertMatch(t, false, "a.a", "!*.(a|b)", bashOpts)
	})

	t.Run("a.a should not match !*.(a|b)*", func(t *testing.T) {
		// extglobs-bash.js:276
		assertMatch(t, false, "a.a", "!*.(a|b)*", bashOpts)
	})

	t.Run("a.a should match (a|d).(a|b)*", func(t *testing.T) {
		// extglobs-bash.js:280
		assertMatch(t, true, "a.a", "(a|d).(a|b)*", bashOpts)
	})

	t.Run("a.a should match (b|a).(a)", func(t *testing.T) {
		// extglobs-bash.js:284
		assertMatch(t, true, "a.a", "(b|a).(a)", bashOpts)
	})

	t.Run("a.a should match *!(.a|.b|.c)", func(t *testing.T) {
		// extglobs-bash.js:288
		assertMatch(t, true, "a.a", "*!(.a|.b|.c)", bashOpts)
	})

	t.Run("a.a should not match *.!(a)", func(t *testing.T) {
		// extglobs-bash.js:292
		assertMatch(t, false, "a.a", "*.!(a)", bashOpts)
	})

	t.Run("a.a should not match *.!(a|b|c)", func(t *testing.T) {
		// extglobs-bash.js:296
		assertMatch(t, false, "a.a", "*.!(a|b|c)", bashOpts)
	})

	t.Run("a.a should match *.(a|b|@(ab|a*@(b))*(c)d)", func(t *testing.T) {
		// extglobs-bash.js:300
		assertMatch(t, true, "a.a", "*.(a|b|@(ab|a*@(b))*(c)d)", bashOpts)
	})

	t.Run("a.a should not match *.+(b|d)", func(t *testing.T) {
		// extglobs-bash.js:304
		assertMatch(t, false, "a.a", "*.+(b|d)", bashOpts)
	})

	t.Run("a.a should match @(b|a).@(a)", func(t *testing.T) {
		// extglobs-bash.js:308
		assertMatch(t, true, "a.a", "@(b|a).@(a)", bashOpts)
	})

	t.Run("a.a.a should not match !(*.[a-b]*)", func(t *testing.T) {
		// extglobs-bash.js:312
		assertMatch(t, false, "a.a.a", "!(*.[a-b]*)", bashOpts)
	})

	t.Run("a.a.a should not match !(*[a-b].[a-b]*)", func(t *testing.T) {
		// extglobs-bash.js:316
		assertMatch(t, false, "a.a.a", "!(*[a-b].[a-b]*)", bashOpts)
	})

	t.Run("a.a.a should not match !*.(a|b)", func(t *testing.T) {
		// extglobs-bash.js:320
		assertMatch(t, false, "a.a.a", "!*.(a|b)", bashOpts)
	})

	t.Run("a.a.a should not match !*.(a|b)*", func(t *testing.T) {
		// extglobs-bash.js:324
		assertMatch(t, false, "a.a.a", "!*.(a|b)*", bashOpts)
	})

	t.Run("a.a.a should match *.!(a)", func(t *testing.T) {
		// extglobs-bash.js:328
		assertMatch(t, true, "a.a.a", "*.!(a)", bashOpts)
	})

	t.Run("a.a.a should not match *.+(b|d)", func(t *testing.T) {
		// extglobs-bash.js:332
		assertMatch(t, false, "a.a.a", "*.+(b|d)", bashOpts)
	})

	t.Run("a.aa.a should not match (b|a).(a)", func(t *testing.T) {
		// extglobs-bash.js:336
		assertMatch(t, false, "a.aa.a", "(b|a).(a)", bashOpts)
	})

	t.Run("a.aa.a should not match @(b|a).@(a)", func(t *testing.T) {
		// extglobs-bash.js:340
		assertMatch(t, false, "a.aa.a", "@(b|a).@(a)", bashOpts)
	})

	t.Run("a.abcd should match !(*.a|*.b|*.c)", func(t *testing.T) {
		// extglobs-bash.js:344
		assertMatch(t, true, "a.abcd", "!(*.a|*.b|*.c)", bashOpts)
	})

	t.Run("a.abcd should not match !(*.a|*.b|*.c)*", func(t *testing.T) {
		// extglobs-bash.js:348
		assertMatch(t, false, "a.abcd", "!(*.a|*.b|*.c)*", bashOpts)
	})

	t.Run("a.abcd should match *!(*.a|*.b|*.c)*", func(t *testing.T) {
		// extglobs-bash.js:352
		assertMatch(t, true, "a.abcd", "*!(*.a|*.b|*.c)*", bashOpts)
	})

	t.Run("a.abcd should match *!(.a|.b|.c)", func(t *testing.T) {
		// extglobs-bash.js:356
		assertMatch(t, true, "a.abcd", "*!(.a|.b|.c)", bashOpts)
	})

	t.Run("a.abcd should match *.!(a|b|c)", func(t *testing.T) {
		// extglobs-bash.js:360
		assertMatch(t, true, "a.abcd", "*.!(a|b|c)", bashOpts)
	})

	t.Run("a.abcd should not match *.!(a|b|c)*", func(t *testing.T) {
		// extglobs-bash.js:364
		assertMatch(t, false, "a.abcd", "*.!(a|b|c)*", bashOpts)
	})

	t.Run("a.abcd should match *.(a|b|@(ab|a*@(b))*(c)d)", func(t *testing.T) {
		// extglobs-bash.js:368
		assertMatch(t, true, "a.abcd", "*.(a|b|@(ab|a*@(b))*(c)d)", bashOpts)
	})

	t.Run("a.b should not match !(*.*)", func(t *testing.T) {
		// extglobs-bash.js:372
		assertMatch(t, false, "a.b", "!(*.*)", bashOpts)
	})

	t.Run("a.b should not match !(*.[a-b]*)", func(t *testing.T) {
		// extglobs-bash.js:376
		assertMatch(t, false, "a.b", "!(*.[a-b]*)", bashOpts)
	})

	t.Run("a.b should not match !(*.a|*.b|*.c)", func(t *testing.T) {
		// extglobs-bash.js:380
		assertMatch(t, false, "a.b", "!(*.a|*.b|*.c)", bashOpts)
	})

	t.Run("a.b should not match !(*[a-b].[a-b]*)", func(t *testing.T) {
		// extglobs-bash.js:384
		assertMatch(t, false, "a.b", "!(*[a-b].[a-b]*)", bashOpts)
	})

	t.Run("a.b should not match !*.(a|b)", func(t *testing.T) {
		// extglobs-bash.js:388
		assertMatch(t, false, "a.b", "!*.(a|b)", bashOpts)
	})

	t.Run("a.b should not match !*.(a|b)*", func(t *testing.T) {
		// extglobs-bash.js:392
		assertMatch(t, false, "a.b", "!*.(a|b)*", bashOpts)
	})

	t.Run("a.b should match (a|d).(a|b)*", func(t *testing.T) {
		// extglobs-bash.js:396
		assertMatch(t, true, "a.b", "(a|d).(a|b)*", bashOpts)
	})

	t.Run("a.b should match *!(.a|.b|.c)", func(t *testing.T) {
		// extglobs-bash.js:400
		assertMatch(t, true, "a.b", "*!(.a|.b|.c)", bashOpts)
	})

	t.Run("a.b should match *.!(a)", func(t *testing.T) {
		// extglobs-bash.js:404
		assertMatch(t, true, "a.b", "*.!(a)", bashOpts)
	})

	t.Run("a.b should not match *.!(a|b|c)", func(t *testing.T) {
		// extglobs-bash.js:408
		assertMatch(t, false, "a.b", "*.!(a|b|c)", bashOpts)
	})

	t.Run("a.b should match *.(a|b|@(ab|a*@(b))*(c)d)", func(t *testing.T) {
		// extglobs-bash.js:412
		assertMatch(t, true, "a.b", "*.(a|b|@(ab|a*@(b))*(c)d)", bashOpts)
	})

	t.Run("a.b should match *.+(b|d)", func(t *testing.T) {
		// extglobs-bash.js:416
		assertMatch(t, true, "a.b", "*.+(b|d)", bashOpts)
	})

	t.Run("a.bb should not match !(*.[a-b]*)", func(t *testing.T) {
		// extglobs-bash.js:420
		assertMatch(t, false, "a.bb", "!(*.[a-b]*)", bashOpts)
	})

	t.Run("a.bb should not match !(*[a-b].[a-b]*)", func(t *testing.T) {
		// extglobs-bash.js:424
		assertMatch(t, false, "a.bb", "!(*[a-b].[a-b]*)", bashOpts)
	})

	t.Run("a.bb should match !*.(a|b)", func(t *testing.T) {
		// extglobs-bash.js:428
		assertMatch(t, true, "a.bb", "!*.(a|b)", bashOpts)
	})

	t.Run("a.bb should not match !*.(a|b)*", func(t *testing.T) {
		// extglobs-bash.js:432
		assertMatch(t, false, "a.bb", "!*.(a|b)*", bashOpts)
	})

	t.Run("a.bb should not match !*.*(a|b)", func(t *testing.T) {
		// extglobs-bash.js:436
		assertMatch(t, false, "a.bb", "!*.*(a|b)", bashOpts)
	})

	t.Run("a.bb should match (a|d).(a|b)*", func(t *testing.T) {
		// extglobs-bash.js:440
		assertMatch(t, true, "a.bb", "(a|d).(a|b)*", bashOpts)
	})

	t.Run("a.bb should not match (b|a).(a)", func(t *testing.T) {
		// extglobs-bash.js:444
		assertMatch(t, false, "a.bb", "(b|a).(a)", bashOpts)
	})

	t.Run("a.bb should match *.+(b|d)", func(t *testing.T) {
		// extglobs-bash.js:448
		assertMatch(t, true, "a.bb", "*.+(b|d)", bashOpts)
	})

	t.Run("a.bb should not match @(b|a).@(a)", func(t *testing.T) {
		// extglobs-bash.js:452
		assertMatch(t, false, "a.bb", "@(b|a).@(a)", bashOpts)
	})

	t.Run("a.c should not match !(*.a|*.b|*.c)", func(t *testing.T) {
		// extglobs-bash.js:456
		assertMatch(t, false, "a.c", "!(*.a|*.b|*.c)", bashOpts)
	})

	t.Run("a.c should match *!(.a|.b|.c)", func(t *testing.T) {
		// extglobs-bash.js:460
		assertMatch(t, true, "a.c", "*!(.a|.b|.c)", bashOpts)
	})

	t.Run("a.c should not match *.!(a|b|c)", func(t *testing.T) {
		// extglobs-bash.js:464
		assertMatch(t, false, "a.c", "*.!(a|b|c)", bashOpts)
	})

	t.Run("a.c should not match *.(a|b|@(ab|a*@(b))*(c)d)", func(t *testing.T) {
		// extglobs-bash.js:468
		assertMatch(t, false, "a.c", "*.(a|b|@(ab|a*@(b))*(c)d)", bashOpts)
	})

	t.Run("a.c.d should match !(*.a|*.b|*.c)", func(t *testing.T) {
		// extglobs-bash.js:472
		assertMatch(t, true, "a.c.d", "!(*.a|*.b|*.c)", bashOpts)
	})

	t.Run("a.c.d should match *!(.a|.b|.c)", func(t *testing.T) {
		// extglobs-bash.js:476
		assertMatch(t, true, "a.c.d", "*!(.a|.b|.c)", bashOpts)
	})

	t.Run("a.c.d should match *.!(a|b|c)", func(t *testing.T) {
		// extglobs-bash.js:480
		assertMatch(t, true, "a.c.d", "*.!(a|b|c)", bashOpts)
	})

	t.Run("a.c.d should not match *.(a|b|@(ab|a*@(b))*(c)d)", func(t *testing.T) {
		// extglobs-bash.js:484
		assertMatch(t, false, "a.c.d", "*.(a|b|@(ab|a*@(b))*(c)d)", bashOpts)
	})

	t.Run("a.ccc should match !(*.[a-b]*)", func(t *testing.T) {
		// extglobs-bash.js:488
		assertMatch(t, true, "a.ccc", "!(*.[a-b]*)", bashOpts)
	})

	t.Run("a.ccc should match !(*[a-b].[a-b]*)", func(t *testing.T) {
		// extglobs-bash.js:492
		assertMatch(t, true, "a.ccc", "!(*[a-b].[a-b]*)", bashOpts)
	})

	t.Run("a.ccc should match !*.(a|b)", func(t *testing.T) {
		// extglobs-bash.js:496
		assertMatch(t, true, "a.ccc", "!*.(a|b)", bashOpts)
	})

	t.Run("a.ccc should match !*.(a|b)*", func(t *testing.T) {
		// extglobs-bash.js:500
		assertMatch(t, true, "a.ccc", "!*.(a|b)*", bashOpts)
	})

	t.Run("a.ccc should not match *.+(b|d)", func(t *testing.T) {
		// extglobs-bash.js:504
		assertMatch(t, false, "a.ccc", "*.+(b|d)", bashOpts)
	})

	t.Run("a.js should not match !(*.js)", func(t *testing.T) {
		// extglobs-bash.js:508
		assertMatch(t, false, "a.js", "!(*.js)", bashOpts)
	})

	t.Run("a.js should match *!(.js)", func(t *testing.T) {
		// extglobs-bash.js:512
		assertMatch(t, true, "a.js", "*!(.js)", bashOpts)
	})

	t.Run("a.js should not match *.!(js)", func(t *testing.T) {
		// extglobs-bash.js:516
		assertMatch(t, false, "a.js", "*.!(js)", bashOpts)
	})

	t.Run("a.js should not match a.!(js)", func(t *testing.T) {
		// extglobs-bash.js:520
		assertMatch(t, false, "a.js", "a.!(js)", bashOpts)
	})

	t.Run("a.js should not match a.!(js)*", func(t *testing.T) {
		// extglobs-bash.js:524
		assertMatch(t, false, "a.js", "a.!(js)*", bashOpts)
	})

	t.Run("a.js.js should not match !(*.js)", func(t *testing.T) {
		// extglobs-bash.js:528
		assertMatch(t, false, "a.js.js", "!(*.js)", bashOpts)
	})

	t.Run("a.js.js should match *!(.js)", func(t *testing.T) {
		// extglobs-bash.js:532
		assertMatch(t, true, "a.js.js", "*!(.js)", bashOpts)
	})

	t.Run("a.js.js should match *.!(js)", func(t *testing.T) {
		// extglobs-bash.js:536
		assertMatch(t, true, "a.js.js", "*.!(js)", bashOpts)
	})

	t.Run("a.js.js should match *.*(js).js", func(t *testing.T) {
		// extglobs-bash.js:540
		assertMatch(t, true, "a.js.js", "*.*(js).js", bashOpts)
	})

	t.Run("a.md should match !(*.js)", func(t *testing.T) {
		// extglobs-bash.js:544
		assertMatch(t, true, "a.md", "!(*.js)", bashOpts)
	})

	t.Run("a.md should match *!(.js)", func(t *testing.T) {
		// extglobs-bash.js:548
		assertMatch(t, true, "a.md", "*!(.js)", bashOpts)
	})

	t.Run("a.md should match *.!(js)", func(t *testing.T) {
		// extglobs-bash.js:552
		assertMatch(t, true, "a.md", "*.!(js)", bashOpts)
	})

	t.Run("a.md should match a.!(js)", func(t *testing.T) {
		// extglobs-bash.js:556
		assertMatch(t, true, "a.md", "a.!(js)", bashOpts)
	})

	t.Run("a.md should match a.!(js)*", func(t *testing.T) {
		// extglobs-bash.js:560
		assertMatch(t, true, "a.md", "a.!(js)*", bashOpts)
	})

	t.Run("a.md.js should not match *.*(js).js", func(t *testing.T) {
		// extglobs-bash.js:564
		assertMatch(t, false, "a.md.js", "*.*(js).js", bashOpts)
	})

	t.Run("a.txt should match a.!(js)", func(t *testing.T) {
		// extglobs-bash.js:568
		assertMatch(t, true, "a.txt", "a.!(js)", bashOpts)
	})

	t.Run("a.txt should match a.!(js)*", func(t *testing.T) {
		// extglobs-bash.js:572
		assertMatch(t, true, "a.txt", "a.!(js)*", bashOpts)
	})

	t.Run("a/!(z) should match a/!(z)", func(t *testing.T) {
		// extglobs-bash.js:576
		assertMatch(t, true, "a/!(z)", "a/!(z)", bashOpts)
	})

	t.Run("a/b should match a/!(z)", func(t *testing.T) {
		// extglobs-bash.js:580
		assertMatch(t, true, "a/b", "a/!(z)", bashOpts)
	})

	t.Run("a/b/c.txt should not match */b/!(*).txt", func(t *testing.T) {
		// extglobs-bash.js:584
		assertMatch(t, false, "a/b/c.txt", "*/b/!(*).txt", bashOpts)
	})

	t.Run("a/b/c.txt should not match */b/!(c).txt", func(t *testing.T) {
		// extglobs-bash.js:588
		assertMatch(t, false, "a/b/c.txt", "*/b/!(c).txt", bashOpts)
	})

	t.Run("a/b/c.txt should match */b/!(cc).txt", func(t *testing.T) {
		// extglobs-bash.js:592
		assertMatch(t, true, "a/b/c.txt", "*/b/!(cc).txt", bashOpts)
	})

	t.Run("a/b/cc.txt should not match */b/!(*).txt", func(t *testing.T) {
		// extglobs-bash.js:596
		assertMatch(t, false, "a/b/cc.txt", "*/b/!(*).txt", bashOpts)
	})

	t.Run("a/b/cc.txt should not match */b/!(c).txt", func(t *testing.T) {
		// extglobs-bash.js:600
		assertMatch(t, false, "a/b/cc.txt", "*/b/!(c).txt", bashOpts)
	})

	t.Run("a/b/cc.txt should not match */b/!(cc).txt", func(t *testing.T) {
		// extglobs-bash.js:604
		assertMatch(t, false, "a/b/cc.txt", "*/b/!(cc).txt", bashOpts)
	})

	t.Run("a/dir/foo.txt should match */dir/**/!(bar).txt", func(t *testing.T) {
		// extglobs-bash.js:608
		assertMatch(t, true, "a/dir/foo.txt", "*/dir/**/!(bar).txt", bashOpts)
	})

	t.Run("a/z should not match a/!(z)", func(t *testing.T) {
		// extglobs-bash.js:612
		assertMatch(t, false, "a/z", "a/!(z)", bashOpts)
	})

	t.Run("a\\\\\\\\(b should not match a(*b", func(t *testing.T) {
		// extglobs-bash.js:616
		assertMatch(t, false, "a\\(b", "a(*b", bashOpts)
	})

	t.Run("a\\\\\\\\(b should not match a(b", func(t *testing.T) {
		// extglobs-bash.js:620
		assertMatch(t, false, "a\\(b", "a(b", bashOpts)
	})

	t.Run("a\\\\\\\\z should match a\\\\\\\\z", func(t *testing.T) {
		// extglobs-bash.js:624
		assertMatch(t, true, "a\\\\z", "a\\\\z", bashNonWinOpts)
	})

	t.Run("a\\\\\\\\z should match a\\\\\\\\z (2)", func(t *testing.T) {
		// extglobs-bash.js:628
		assertMatch(t, true, "a\\\\z", "a\\\\z", bashOpts)
	})

	t.Run("a\\\\\\\\b should match a/b", func(t *testing.T) {
		// extglobs-bash.js:632
		assertMatch(t, true, "a\\b", "a/b", winOpts)
	})

	t.Run("a\\\\\\\\z should match a\\\\\\\\z (3)", func(t *testing.T) {
		// extglobs-bash.js:636
		assertMatch(t, true, "a\\\\z", "a\\\\z", bashOpts)
		// extglobs-bash.js:637
		assertMatch(t, true, "a\\z", "a\\z", bashOpts)
	})

	t.Run("a\\\\\\\\z should not match a\\\\\\\\z", func(t *testing.T) {
		// extglobs-bash.js:641
		assertMatch(t, true, "a\\z", "a\\z", bashOpts)
	})

	t.Run("aa should not match !(a!(b))", func(t *testing.T) {
		// extglobs-bash.js:645
		assertMatch(t, false, "aa", "!(a!(b))", bashOpts)
	})

	t.Run("aa should match !(a)", func(t *testing.T) {
		// extglobs-bash.js:649
		assertMatch(t, true, "aa", "!(a)", bashOpts)
	})

	t.Run("aa should not match !(a)*", func(t *testing.T) {
		// extglobs-bash.js:653
		assertMatch(t, false, "aa", "!(a)*", bashOpts)
	})

	t.Run("aa should not match ?", func(t *testing.T) {
		// extglobs-bash.js:657
		assertMatch(t, false, "aa", "?", bashOpts)
	})

	t.Run("aa should not match @(a)b", func(t *testing.T) {
		// extglobs-bash.js:661
		assertMatch(t, false, "aa", "@(a)b", bashOpts)
	})

	t.Run("aa should match a!(b)*", func(t *testing.T) {
		// extglobs-bash.js:665
		assertMatch(t, true, "aa", "a!(b)*", bashOpts)
	})

	t.Run("aa should not match a??b", func(t *testing.T) {
		// extglobs-bash.js:669
		assertMatch(t, false, "aa", "a??b", bashOpts)
	})

	t.Run("aa.aa should not match (b|a).(a)", func(t *testing.T) {
		// extglobs-bash.js:673
		assertMatch(t, false, "aa.aa", "(b|a).(a)", bashOpts)
	})

	t.Run("aa.aa should not match @(b|a).@(a)", func(t *testing.T) {
		// extglobs-bash.js:677
		assertMatch(t, false, "aa.aa", "@(b|a).@(a)", bashOpts)
	})

	t.Run("aaa should not match !(a)*", func(t *testing.T) {
		// extglobs-bash.js:681
		assertMatch(t, false, "aaa", "!(a)*", bashOpts)
	})

	t.Run("aaa should match a!(b)*", func(t *testing.T) {
		// extglobs-bash.js:685
		assertMatch(t, true, "aaa", "a!(b)*", bashOpts)
	})

	t.Run("aaaaaaabababab should match *ab", func(t *testing.T) {
		// extglobs-bash.js:689
		assertMatch(t, true, "aaaaaaabababab", "*ab", bashOpts)
	})

	t.Run("aaac should match *(@(a))a@(c)", func(t *testing.T) {
		// extglobs-bash.js:693
		assertMatch(t, true, "aaac", "*(@(a))a@(c)", bashOpts)
	})

	t.Run("aaaz should match [a*(]*z", func(t *testing.T) {
		// extglobs-bash.js:697
		assertMatch(t, true, "aaaz", "[a*(]*z", bashOpts)
	})

	t.Run("aab should not match !(a)*", func(t *testing.T) {
		// extglobs-bash.js:701
		assertMatch(t, false, "aab", "!(a)*", bashOpts)
	})

	t.Run("aab should not match ?", func(t *testing.T) {
		// extglobs-bash.js:705
		assertMatch(t, false, "aab", "?", bashOpts)
	})

	t.Run("aab should not match ??", func(t *testing.T) {
		// extglobs-bash.js:709
		assertMatch(t, false, "aab", "??", bashOpts)
	})

	t.Run("aab should not match @(c)b", func(t *testing.T) {
		// extglobs-bash.js:713
		assertMatch(t, false, "aab", "@(c)b", bashOpts)
	})

	t.Run("aab should match a!(b)*", func(t *testing.T) {
		// extglobs-bash.js:717
		assertMatch(t, true, "aab", "a!(b)*", bashOpts)
	})

	t.Run("aab should not match a??b", func(t *testing.T) {
		// extglobs-bash.js:721
		assertMatch(t, false, "aab", "a??b", bashOpts)
	})

	t.Run("aac should match *(@(a))a@(c)", func(t *testing.T) {
		// extglobs-bash.js:725
		assertMatch(t, true, "aac", "*(@(a))a@(c)", bashOpts)
	})

	t.Run("aac should not match *(@(a))b@(c)", func(t *testing.T) {
		// extglobs-bash.js:729
		assertMatch(t, false, "aac", "*(@(a))b@(c)", bashOpts)
	})

	t.Run("aax should not match a!(a*|b)", func(t *testing.T) {
		// extglobs-bash.js:733
		assertMatch(t, false, "aax", "a!(a*|b)", bashOpts)
	})

	t.Run("aax should match a!(x*|b)", func(t *testing.T) {
		// extglobs-bash.js:737
		assertMatch(t, true, "aax", "a!(x*|b)", bashOpts)
	})

	t.Run("aax should match a?(a*|b)", func(t *testing.T) {
		// extglobs-bash.js:741
		assertMatch(t, true, "aax", "a?(a*|b)", bashOpts)
	})

	t.Run("aaz should match [a*(]*z", func(t *testing.T) {
		// extglobs-bash.js:745
		assertMatch(t, true, "aaz", "[a*(]*z", bashOpts)
	})

	t.Run("ab should match !(*.*)", func(t *testing.T) {
		// extglobs-bash.js:749
		assertMatch(t, true, "ab", "!(*.*)", bashOpts)
	})

	t.Run("ab should match !(a!(b))", func(t *testing.T) {
		// extglobs-bash.js:753
		assertMatch(t, true, "ab", "!(a!(b))", bashOpts)
	})

	t.Run("ab should not match !(a)*", func(t *testing.T) {
		// extglobs-bash.js:757
		assertMatch(t, false, "ab", "!(a)*", bashOpts)
	})

	t.Run("ab should match @(a+|b)*", func(t *testing.T) {
		// extglobs-bash.js:761
		assertMatch(t, true, "ab", "@(a+|b)*", bashOpts)
	})

	t.Run("ab should match (a+|b)+", func(t *testing.T) {
		// extglobs-bash.js:765
		assertMatch(t, true, "ab", "(a+|b)+", bashOpts)
	})

	t.Run("ab should not match *?(a)bc", func(t *testing.T) {
		// extglobs-bash.js:769
		assertMatch(t, false, "ab", "*?(a)bc", bashOpts)
	})

	t.Run("ab should not match a!(*(b|B))", func(t *testing.T) {
		// extglobs-bash.js:773
		assertMatch(t, false, "ab", "a!(*(b|B))", bashOpts)
	})

	t.Run("ab should not match a!(@(b|B))", func(t *testing.T) {
		// extglobs-bash.js:777
		assertMatch(t, false, "ab", "a!(@(b|B))", bashOpts)
	})

	t.Run("aB should not match a!(@(b|B))", func(t *testing.T) {
		// extglobs-bash.js:781
		assertMatch(t, false, "aB", "a!(@(b|B))", bashOpts)
	})

	t.Run("ab should not match a!(b)*", func(t *testing.T) {
		// extglobs-bash.js:785
		assertMatch(t, false, "ab", "a!(b)*", bashOpts)
	})

	t.Run("ab should not match a(*b", func(t *testing.T) {
		// extglobs-bash.js:789
		assertMatch(t, false, "ab", "a(*b", bashOpts)
	})

	t.Run("ab should not match a(b", func(t *testing.T) {
		// extglobs-bash.js:793
		assertMatch(t, false, "ab", "a(b", bashOpts)
	})

	t.Run("ab should not match a(b*(foo|bar))d", func(t *testing.T) {
		// extglobs-bash.js:797
		assertMatch(t, false, "ab", "a(b*(foo|bar))d", bashOpts)
	})

	t.Run("ab should not match a/b", func(t *testing.T) {
		// extglobs-bash.js:801
		assertMatch(t, false, "ab", "a/b", winOpts)
	})

	t.Run("ab should not match a\\\\\\\\(b", func(t *testing.T) {
		// extglobs-bash.js:805
		assertMatch(t, false, "ab", "a\\(b", bashOpts)
	})

	t.Run("ab should match ab*(e|f)", func(t *testing.T) {
		// extglobs-bash.js:809
		assertMatch(t, true, "ab", "ab*(e|f)", bashOpts)
	})

	t.Run("ab should match ab**", func(t *testing.T) {
		// extglobs-bash.js:813
		assertMatch(t, true, "ab", "ab**", bashOpts)
	})

	t.Run("ab should match ab**(e|f)", func(t *testing.T) {
		// extglobs-bash.js:817
		assertMatch(t, true, "ab", "ab**(e|f)", bashOpts)
	})

	t.Run("ab should not match ab**(e|f)g", func(t *testing.T) {
		// extglobs-bash.js:821
		assertMatch(t, false, "ab", "ab**(e|f)g", bashOpts)
	})

	t.Run("ab should not match ab***ef", func(t *testing.T) {
		// extglobs-bash.js:825
		assertMatch(t, false, "ab", "ab***ef", bashOpts)
	})

	t.Run("ab should not match ab*+(e|f)", func(t *testing.T) {
		// extglobs-bash.js:829
		assertMatch(t, false, "ab", "ab*+(e|f)", bashOpts)
	})

	t.Run("ab should not match ab*d+(e|f)", func(t *testing.T) {
		// extglobs-bash.js:833
		assertMatch(t, false, "ab", "ab*d+(e|f)", bashOpts)
	})

	t.Run("ab should not match ab?*(e|f)", func(t *testing.T) {
		// extglobs-bash.js:837
		assertMatch(t, false, "ab", "ab?*(e|f)", bashOpts)
	})

	t.Run("ab/cXd/efXg/hi should match **/*X*/**/*i", func(t *testing.T) {
		// extglobs-bash.js:841
		assertMatch(t, true, "ab/cXd/efXg/hi", "**/*X*/**/*i", bashOpts)
	})

	t.Run("ab/cXd/efXg/hi should match */*X*/*/*i", func(t *testing.T) {
		// extglobs-bash.js:845
		assertMatch(t, true, "ab/cXd/efXg/hi", "*/*X*/*/*i", bashOpts)
	})

	t.Run("ab/cXd/efXg/hi should match *X*i", func(t *testing.T) {
		// extglobs-bash.js:849
		assertMatch(t, true, "ab/cXd/efXg/hi", "*X*i", bashOpts)
	})

	t.Run("ab/cXd/efXg/hi should match *Xg*i", func(t *testing.T) {
		// extglobs-bash.js:853
		assertMatch(t, true, "ab/cXd/efXg/hi", "*Xg*i", bashOpts)
	})

	t.Run("ab] should match a!(@(b|B))", func(t *testing.T) {
		// extglobs-bash.js:857
		assertMatch(t, true, "ab]", "a!(@(b|B))", bashOpts)
	})

	t.Run("abab should match (a+|b)*", func(t *testing.T) {
		// extglobs-bash.js:861
		assertMatch(t, true, "abab", "(a+|b)*", bashOpts)
	})

	t.Run("abab should match (a+|b)+", func(t *testing.T) {
		// extglobs-bash.js:865
		assertMatch(t, true, "abab", "(a+|b)+", bashOpts)
	})

	t.Run("abab should not match *?(a)bc", func(t *testing.T) {
		// extglobs-bash.js:869
		assertMatch(t, false, "abab", "*?(a)bc", bashOpts)
	})

	t.Run("abab should not match a(b*(foo|bar))d", func(t *testing.T) {
		// extglobs-bash.js:873
		assertMatch(t, false, "abab", "a(b*(foo|bar))d", bashOpts)
	})

	t.Run("abab should not match ab*(e|f)", func(t *testing.T) {
		// extglobs-bash.js:877
		assertMatch(t, false, "abab", "ab*(e|f)", bashOpts)
	})

	t.Run("abab should match ab**", func(t *testing.T) {
		// extglobs-bash.js:881
		assertMatch(t, true, "abab", "ab**", bashOpts)
	})

	t.Run("abab should match ab**(e|f)", func(t *testing.T) {
		// extglobs-bash.js:885
		assertMatch(t, true, "abab", "ab**(e|f)", bashOpts)
	})

	t.Run("abab should not match ab**(e|f)g", func(t *testing.T) {
		// extglobs-bash.js:889
		assertMatch(t, false, "abab", "ab**(e|f)g", bashOpts)
	})

	t.Run("abab should not match ab***ef", func(t *testing.T) {
		// extglobs-bash.js:893
		assertMatch(t, false, "abab", "ab***ef", bashOpts)
	})

	t.Run("abab should not match ab*+(e|f)", func(t *testing.T) {
		// extglobs-bash.js:897
		assertMatch(t, false, "abab", "ab*+(e|f)", bashOpts)
	})

	t.Run("abab should not match ab*d+(e|f)", func(t *testing.T) {
		// extglobs-bash.js:901
		assertMatch(t, false, "abab", "ab*d+(e|f)", bashOpts)
	})

	t.Run("abab should not match ab?*(e|f)", func(t *testing.T) {
		// extglobs-bash.js:905
		assertMatch(t, false, "abab", "ab?*(e|f)", bashOpts)
	})

	t.Run("abb should match !(*.*)", func(t *testing.T) {
		// extglobs-bash.js:909
		assertMatch(t, true, "abb", "!(*.*)", bashOpts)
	})

	t.Run("abb should not match !(a)*", func(t *testing.T) {
		// extglobs-bash.js:913
		assertMatch(t, false, "abb", "!(a)*", bashOpts)
	})

	t.Run("abb should not match a!(b)*", func(t *testing.T) {
		// extglobs-bash.js:917
		assertMatch(t, false, "abb", "a!(b)*", bashOpts)
	})

	t.Run("abbcd should match @(ab|a*(b))*(c)d", func(t *testing.T) {
		// extglobs-bash.js:921
		assertMatch(t, true, "abbcd", "@(ab|a*(b))*(c)d", bashOpts)
	})

	t.Run("abc should not match \\\\\\\\a\\\\\\\\b\\\\\\\\c", func(t *testing.T) {
		// extglobs-bash.js:925
		assertMatch(t, false, "abc", "\\a\\b\\c", bashOpts)
	})

	t.Run("aBc should match a!(@(b|B))", func(t *testing.T) {
		// extglobs-bash.js:929
		assertMatch(t, true, "aBc", "a!(@(b|B))", bashOpts)
	})

	t.Run("abcd should match ?@(a|b)*@(c)d", func(t *testing.T) {
		// extglobs-bash.js:933
		assertMatch(t, true, "abcd", "?@(a|b)*@(c)d", bashOpts)
	})

	t.Run("abcd should match @(ab|a*@(b))*(c)d", func(t *testing.T) {
		// extglobs-bash.js:937
		assertMatch(t, true, "abcd", "@(ab|a*@(b))*(c)d", bashOpts)
	})

	t.Run("abcd/abcdefg/abcdefghijk/abcdefghijklmnop.txt should match **/*a*b*g*n*t", func(t *testing.T) {
		// extglobs-bash.js:941
		assertMatch(t, true, "abcd/abcdefg/abcdefghijk/abcdefghijklmnop.txt", "**/*a*b*g*n*t", bashOpts)
	})

	t.Run("abcd/abcdefg/abcdefghijk/abcdefghijklmnop.txtz should not match **/*a*b*g*n*t", func(t *testing.T) {
		// extglobs-bash.js:945
		assertMatch(t, false, "abcd/abcdefg/abcdefghijk/abcdefghijklmnop.txtz", "**/*a*b*g*n*t", bashOpts)
	})

	t.Run("abcdef should match (a+|b)*", func(t *testing.T) {
		// extglobs-bash.js:949
		assertMatch(t, true, "abcdef", "(a+|b)*", bashOpts)
	})

	t.Run("abcdef should not match (a+|b)+", func(t *testing.T) {
		// extglobs-bash.js:953
		assertMatch(t, false, "abcdef", "(a+|b)+", bashOpts)
	})

	t.Run("abcdef should not match *?(a)bc", func(t *testing.T) {
		// extglobs-bash.js:957
		assertMatch(t, false, "abcdef", "*?(a)bc", bashOpts)
	})

	t.Run("abcdef should not match a(b*(foo|bar))d", func(t *testing.T) {
		// extglobs-bash.js:961
		assertMatch(t, false, "abcdef", "a(b*(foo|bar))d", bashOpts)
	})

	t.Run("abcdef should not match ab*(e|f)", func(t *testing.T) {
		// extglobs-bash.js:965
		assertMatch(t, false, "abcdef", "ab*(e|f)", bashOpts)
	})

	t.Run("abcdef should match ab**", func(t *testing.T) {
		// extglobs-bash.js:969
		assertMatch(t, true, "abcdef", "ab**", bashOpts)
	})

	t.Run("abcdef should match ab**(e|f)", func(t *testing.T) {
		// extglobs-bash.js:973
		assertMatch(t, true, "abcdef", "ab**(e|f)", bashOpts)
	})

	t.Run("abcdef should not match ab**(e|f)g", func(t *testing.T) {
		// extglobs-bash.js:977
		assertMatch(t, false, "abcdef", "ab**(e|f)g", bashOpts)
	})

	t.Run("abcdef should match ab***ef", func(t *testing.T) {
		// extglobs-bash.js:981
		assertMatch(t, true, "abcdef", "ab***ef", bashOpts)
	})

	t.Run("abcdef should match ab*+(e|f)", func(t *testing.T) {
		// extglobs-bash.js:985
		assertMatch(t, true, "abcdef", "ab*+(e|f)", bashOpts)
	})

	t.Run("abcdef should match ab*d+(e|f)", func(t *testing.T) {
		// extglobs-bash.js:989
		assertMatch(t, true, "abcdef", "ab*d+(e|f)", bashOpts)
	})

	t.Run("abcdef should not match ab?*(e|f)", func(t *testing.T) {
		// extglobs-bash.js:993
		assertMatch(t, false, "abcdef", "ab?*(e|f)", bashOpts)
	})

	t.Run("abcfef should match (a+|b)*", func(t *testing.T) {
		// extglobs-bash.js:997
		assertMatch(t, true, "abcfef", "(a+|b)*", bashOpts)
	})

	t.Run("abcfef should not match (a+|b)+", func(t *testing.T) {
		// extglobs-bash.js:1001
		assertMatch(t, false, "abcfef", "(a+|b)+", bashOpts)
	})

	t.Run("abcfef should not match *?(a)bc", func(t *testing.T) {
		// extglobs-bash.js:1005
		assertMatch(t, false, "abcfef", "*?(a)bc", bashOpts)
	})

	t.Run("abcfef should not match a(b*(foo|bar))d", func(t *testing.T) {
		// extglobs-bash.js:1009
		assertMatch(t, false, "abcfef", "a(b*(foo|bar))d", bashOpts)
	})

	t.Run("abcfef should not match ab*(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1013
		assertMatch(t, false, "abcfef", "ab*(e|f)", bashOpts)
	})

	t.Run("abcfef should match ab**", func(t *testing.T) {
		// extglobs-bash.js:1017
		assertMatch(t, true, "abcfef", "ab**", bashOpts)
	})

	t.Run("abcfef should match ab**(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1021
		assertMatch(t, true, "abcfef", "ab**(e|f)", bashOpts)
	})

	t.Run("abcfef should not match ab**(e|f)g", func(t *testing.T) {
		// extglobs-bash.js:1025
		assertMatch(t, false, "abcfef", "ab**(e|f)g", bashOpts)
	})

	t.Run("abcfef should match ab***ef", func(t *testing.T) {
		// extglobs-bash.js:1029
		assertMatch(t, true, "abcfef", "ab***ef", bashOpts)
	})

	t.Run("abcfef should match ab*+(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1033
		assertMatch(t, true, "abcfef", "ab*+(e|f)", bashOpts)
	})

	t.Run("abcfef should not match ab*d+(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1037
		assertMatch(t, false, "abcfef", "ab*d+(e|f)", bashOpts)
	})

	t.Run("abcfef should match ab?*(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1041
		assertMatch(t, true, "abcfef", "ab?*(e|f)", bashOpts)
	})

	t.Run("abcfefg should match (a+|b)*", func(t *testing.T) {
		// extglobs-bash.js:1045
		assertMatch(t, true, "abcfefg", "(a+|b)*", bashOpts)
	})

	t.Run("abcfefg should not match (a+|b)+", func(t *testing.T) {
		// extglobs-bash.js:1049
		assertMatch(t, false, "abcfefg", "(a+|b)+", bashOpts)
	})

	t.Run("abcfefg should not match *?(a)bc", func(t *testing.T) {
		// extglobs-bash.js:1053
		assertMatch(t, false, "abcfefg", "*?(a)bc", bashOpts)
	})

	t.Run("abcfefg should not match a(b*(foo|bar))d", func(t *testing.T) {
		// extglobs-bash.js:1057
		assertMatch(t, false, "abcfefg", "a(b*(foo|bar))d", bashOpts)
	})

	t.Run("abcfefg should not match ab*(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1061
		assertMatch(t, false, "abcfefg", "ab*(e|f)", bashOpts)
	})

	t.Run("abcfefg should match ab**", func(t *testing.T) {
		// extglobs-bash.js:1065
		assertMatch(t, true, "abcfefg", "ab**", bashOpts)
	})

	t.Run("abcfefg should match ab**(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1069
		assertMatch(t, true, "abcfefg", "ab**(e|f)", bashOpts)
	})

	t.Run("abcfefg should match ab**(e|f)g", func(t *testing.T) {
		// extglobs-bash.js:1073
		assertMatch(t, true, "abcfefg", "ab**(e|f)g", bashOpts)
	})

	t.Run("abcfefg should not match ab***ef", func(t *testing.T) {
		// extglobs-bash.js:1077
		assertMatch(t, false, "abcfefg", "ab***ef", bashOpts)
	})

	t.Run("abcfefg should not match ab*+(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1081
		assertMatch(t, false, "abcfefg", "ab*+(e|f)", bashOpts)
	})

	t.Run("abcfefg should not match ab*d+(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1085
		assertMatch(t, false, "abcfefg", "ab*d+(e|f)", bashOpts)
	})

	t.Run("abcfefg should not match ab?*(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1089
		assertMatch(t, false, "abcfefg", "ab?*(e|f)", bashOpts)
	})

	t.Run("abcx should match !([[*])*", func(t *testing.T) {
		// extglobs-bash.js:1093
		assertMatch(t, true, "abcx", "!([[*])*", bashOpts)
	})

	t.Run("abcx should match +(a|b\\\\\\\\[)*", func(t *testing.T) {
		// extglobs-bash.js:1097
		assertMatch(t, true, "abcx", "+(a|b\\[)*", bashOpts)
	})

	t.Run("abcx should not match [a*(]*z", func(t *testing.T) {
		// extglobs-bash.js:1101
		assertMatch(t, false, "abcx", "[a*(]*z", bashOpts)
	})

	t.Run("abcXdefXghi should match *X*i", func(t *testing.T) {
		// extglobs-bash.js:1105
		assertMatch(t, true, "abcXdefXghi", "*X*i", bashOpts)
	})

	t.Run("abcz should match !([[*])*", func(t *testing.T) {
		// extglobs-bash.js:1109
		assertMatch(t, true, "abcz", "!([[*])*", bashOpts)
	})

	t.Run("abcz should match +(a|b\\\\\\\\[)*", func(t *testing.T) {
		// extglobs-bash.js:1113
		assertMatch(t, true, "abcz", "+(a|b\\[)*", bashOpts)
	})

	t.Run("abcz should match [a*(]*z", func(t *testing.T) {
		// extglobs-bash.js:1117
		assertMatch(t, true, "abcz", "[a*(]*z", bashOpts)
	})

	t.Run("abd should match (a+|b)*", func(t *testing.T) {
		// extglobs-bash.js:1121
		assertMatch(t, true, "abd", "(a+|b)*", bashOpts)
	})

	t.Run("abd should not match (a+|b)+", func(t *testing.T) {
		// extglobs-bash.js:1125
		assertMatch(t, false, "abd", "(a+|b)+", bashOpts)
	})

	t.Run("abd should not match *?(a)bc", func(t *testing.T) {
		// extglobs-bash.js:1129
		assertMatch(t, false, "abd", "*?(a)bc", bashOpts)
	})

	t.Run("abd should match a!(*(b|B))", func(t *testing.T) {
		// extglobs-bash.js:1133
		assertMatch(t, true, "abd", "a!(*(b|B))", bashOpts)
	})

	t.Run("abd should match a!(@(b|B))", func(t *testing.T) {
		// extglobs-bash.js:1137
		assertMatch(t, true, "abd", "a!(@(b|B))", bashOpts)
	})

	t.Run("abd should not match a!(@(b|B))d", func(t *testing.T) {
		// extglobs-bash.js:1141
		assertMatch(t, false, "abd", "a!(@(b|B))d", bashOpts)
	})

	t.Run("abd should match a(b*(foo|bar))d", func(t *testing.T) {
		// extglobs-bash.js:1145
		assertMatch(t, true, "abd", "a(b*(foo|bar))d", bashOpts)
	})

	t.Run("abd should match a+(b|c)d", func(t *testing.T) {
		// extglobs-bash.js:1149
		assertMatch(t, true, "abd", "a+(b|c)d", bashOpts)
	})

	t.Run("abd should match a[b*(foo|bar)]d", func(t *testing.T) {
		// extglobs-bash.js:1153
		assertMatch(t, true, "abd", "a[b*(foo|bar)]d", bashOpts)
	})

	t.Run("abd should not match ab*(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1157
		assertMatch(t, false, "abd", "ab*(e|f)", bashOpts)
	})

	t.Run("abd should match ab**", func(t *testing.T) {
		// extglobs-bash.js:1161
		assertMatch(t, true, "abd", "ab**", bashOpts)
	})

	t.Run("abd should match ab**(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1165
		assertMatch(t, true, "abd", "ab**(e|f)", bashOpts)
	})

	t.Run("abd should not match ab**(e|f)g", func(t *testing.T) {
		// extglobs-bash.js:1169
		assertMatch(t, false, "abd", "ab**(e|f)g", bashOpts)
	})

	t.Run("abd should not match ab***ef", func(t *testing.T) {
		// extglobs-bash.js:1173
		assertMatch(t, false, "abd", "ab***ef", bashOpts)
	})

	t.Run("abd should not match ab*+(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1177
		assertMatch(t, false, "abd", "ab*+(e|f)", bashOpts)
	})

	t.Run("abd should not match ab*d+(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1181
		assertMatch(t, false, "abd", "ab*d+(e|f)", bashOpts)
	})

	t.Run("abd should match ab?*(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1185
		assertMatch(t, true, "abd", "ab?*(e|f)", bashOpts)
	})

	t.Run("abef should match (a+|b)*", func(t *testing.T) {
		// extglobs-bash.js:1189
		assertMatch(t, true, "abef", "(a+|b)*", bashOpts)
	})

	t.Run("abef should not match (a+|b)+", func(t *testing.T) {
		// extglobs-bash.js:1193
		assertMatch(t, false, "abef", "(a+|b)+", bashOpts)
	})

	t.Run("abef should not match *(a+|b)", func(t *testing.T) {
		// extglobs-bash.js:1197
		assertMatch(t, false, "abef", "*(a+|b)", bashOpts)
	})

	t.Run("abef should not match *?(a)bc", func(t *testing.T) {
		// extglobs-bash.js:1201
		assertMatch(t, false, "abef", "*?(a)bc", bashOpts)
	})

	t.Run("abef should not match a(b*(foo|bar))d", func(t *testing.T) {
		// extglobs-bash.js:1205
		assertMatch(t, false, "abef", "a(b*(foo|bar))d", bashOpts)
	})

	t.Run("abef should match ab*(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1209
		assertMatch(t, true, "abef", "ab*(e|f)", bashOpts)
	})

	t.Run("abef should match ab**", func(t *testing.T) {
		// extglobs-bash.js:1213
		assertMatch(t, true, "abef", "ab**", bashOpts)
	})

	t.Run("abef should match ab**(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1217
		assertMatch(t, true, "abef", "ab**(e|f)", bashOpts)
	})

	t.Run("abef should not match ab**(e|f)g", func(t *testing.T) {
		// extglobs-bash.js:1221
		assertMatch(t, false, "abef", "ab**(e|f)g", bashOpts)
	})

	t.Run("abef should match ab***ef", func(t *testing.T) {
		// extglobs-bash.js:1225
		assertMatch(t, true, "abef", "ab***ef", bashOpts)
	})

	t.Run("abef should match ab*+(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1229
		assertMatch(t, true, "abef", "ab*+(e|f)", bashOpts)
	})

	t.Run("abef should not match ab*d+(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1233
		assertMatch(t, false, "abef", "ab*d+(e|f)", bashOpts)
	})

	t.Run("abef should match ab?*(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1237
		assertMatch(t, true, "abef", "ab?*(e|f)", bashOpts)
	})

	t.Run("abz should not match a!(*)", func(t *testing.T) {
		// extglobs-bash.js:1241
		assertMatch(t, false, "abz", "a!(*)", bashOpts)
	})

	t.Run("abz should match a!(z)", func(t *testing.T) {
		// extglobs-bash.js:1245
		assertMatch(t, true, "abz", "a!(z)", bashOpts)
	})

	t.Run("abz should match a*!(z)", func(t *testing.T) {
		// extglobs-bash.js:1249
		assertMatch(t, true, "abz", "a*!(z)", bashOpts)
	})

	t.Run("abz should not match a*(z)", func(t *testing.T) {
		// extglobs-bash.js:1253
		assertMatch(t, false, "abz", "a*(z)", bashOpts)
	})

	t.Run("abz should match a**(z)", func(t *testing.T) {
		// extglobs-bash.js:1257
		assertMatch(t, true, "abz", "a**(z)", bashOpts)
	})

	t.Run("abz should match a*@(z)", func(t *testing.T) {
		// extglobs-bash.js:1261
		assertMatch(t, true, "abz", "a*@(z)", bashOpts)
	})

	t.Run("abz should not match a+(z)", func(t *testing.T) {
		// extglobs-bash.js:1265
		assertMatch(t, false, "abz", "a+(z)", bashOpts)
	})

	t.Run("abz should not match a?(z)", func(t *testing.T) {
		// extglobs-bash.js:1269
		assertMatch(t, false, "abz", "a?(z)", bashOpts)
	})

	t.Run("abz should not match a@(z)", func(t *testing.T) {
		// extglobs-bash.js:1273
		assertMatch(t, false, "abz", "a@(z)", bashOpts)
	})

	t.Run("ac should not match !(a)*", func(t *testing.T) {
		// extglobs-bash.js:1277
		assertMatch(t, false, "ac", "!(a)*", bashOpts)
	})

	t.Run("ac should match *(@(a))a@(c)", func(t *testing.T) {
		// extglobs-bash.js:1281
		assertMatch(t, true, "ac", "*(@(a))a@(c)", bashOpts)
	})

	t.Run("ac should match a!(*(b|B))", func(t *testing.T) {
		// extglobs-bash.js:1285
		assertMatch(t, true, "ac", "a!(*(b|B))", bashOpts)
	})

	t.Run("ac should match a!(@(b|B))", func(t *testing.T) {
		// extglobs-bash.js:1289
		assertMatch(t, true, "ac", "a!(@(b|B))", bashOpts)
	})

	t.Run("ac should match a!(b)*", func(t *testing.T) {
		// extglobs-bash.js:1293
		assertMatch(t, true, "ac", "a!(b)*", bashOpts)
	})

	t.Run("accdef should match (a+|b)*", func(t *testing.T) {
		// extglobs-bash.js:1297
		assertMatch(t, true, "accdef", "(a+|b)*", bashOpts)
	})

	t.Run("accdef should not match (a+|b)+", func(t *testing.T) {
		// extglobs-bash.js:1301
		assertMatch(t, false, "accdef", "(a+|b)+", bashOpts)
	})

	t.Run("accdef should not match *?(a)bc", func(t *testing.T) {
		// extglobs-bash.js:1305
		assertMatch(t, false, "accdef", "*?(a)bc", bashOpts)
	})

	t.Run("accdef should not match a(b*(foo|bar))d", func(t *testing.T) {
		// extglobs-bash.js:1309
		assertMatch(t, false, "accdef", "a(b*(foo|bar))d", bashOpts)
	})

	t.Run("accdef should not match ab*(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1313
		assertMatch(t, false, "accdef", "ab*(e|f)", bashOpts)
	})

	t.Run("accdef should not match ab**", func(t *testing.T) {
		// extglobs-bash.js:1317
		assertMatch(t, false, "accdef", "ab**", bashOpts)
	})

	t.Run("accdef should not match ab**(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1321
		assertMatch(t, false, "accdef", "ab**(e|f)", bashOpts)
	})

	t.Run("accdef should not match ab**(e|f)g", func(t *testing.T) {
		// extglobs-bash.js:1325
		assertMatch(t, false, "accdef", "ab**(e|f)g", bashOpts)
	})

	t.Run("accdef should not match ab***ef", func(t *testing.T) {
		// extglobs-bash.js:1329
		assertMatch(t, false, "accdef", "ab***ef", bashOpts)
	})

	t.Run("accdef should not match ab*+(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1333
		assertMatch(t, false, "accdef", "ab*+(e|f)", bashOpts)
	})

	t.Run("accdef should not match ab*d+(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1337
		assertMatch(t, false, "accdef", "ab*d+(e|f)", bashOpts)
	})

	t.Run("accdef should not match ab?*(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1341
		assertMatch(t, false, "accdef", "ab?*(e|f)", bashOpts)
	})

	t.Run("acd should match (a+|b)*", func(t *testing.T) {
		// extglobs-bash.js:1345
		assertMatch(t, true, "acd", "(a+|b)*", bashOpts)
	})

	t.Run("acd should not match (a+|b)+", func(t *testing.T) {
		// extglobs-bash.js:1349
		assertMatch(t, false, "acd", "(a+|b)+", bashOpts)
	})

	t.Run("acd should not match *?(a)bc", func(t *testing.T) {
		// extglobs-bash.js:1353
		assertMatch(t, false, "acd", "*?(a)bc", bashOpts)
	})

	t.Run("acd should match @(ab|a*(b))*(c)d", func(t *testing.T) {
		// extglobs-bash.js:1357
		assertMatch(t, true, "acd", "@(ab|a*(b))*(c)d", bashOpts)
	})

	t.Run("acd should match a!(*(b|B))", func(t *testing.T) {
		// extglobs-bash.js:1361
		assertMatch(t, true, "acd", "a!(*(b|B))", bashOpts)
	})

	t.Run("acd should match a!(@(b|B))", func(t *testing.T) {
		// extglobs-bash.js:1365
		assertMatch(t, true, "acd", "a!(@(b|B))", bashOpts)
	})

	t.Run("acd should match a!(@(b|B))d", func(t *testing.T) {
		// extglobs-bash.js:1369
		assertMatch(t, true, "acd", "a!(@(b|B))d", bashOpts)
	})

	t.Run("acd should not match a(b*(foo|bar))d", func(t *testing.T) {
		// extglobs-bash.js:1373
		assertMatch(t, false, "acd", "a(b*(foo|bar))d", bashOpts)
	})

	t.Run("acd should match a+(b|c)d", func(t *testing.T) {
		// extglobs-bash.js:1377
		assertMatch(t, true, "acd", "a+(b|c)d", bashOpts)
	})

	t.Run("acd should not match a[b*(foo|bar)]d", func(t *testing.T) {
		// extglobs-bash.js:1381
		assertMatch(t, false, "acd", "a[b*(foo|bar)]d", bashOpts)
	})

	t.Run("acd should not match ab*(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1385
		assertMatch(t, false, "acd", "ab*(e|f)", bashOpts)
	})

	t.Run("acd should not match ab**", func(t *testing.T) {
		// extglobs-bash.js:1389
		assertMatch(t, false, "acd", "ab**", bashOpts)
	})

	t.Run("acd should not match ab**(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1393
		assertMatch(t, false, "acd", "ab**(e|f)", bashOpts)
	})

	t.Run("acd should not match ab**(e|f)g", func(t *testing.T) {
		// extglobs-bash.js:1397
		assertMatch(t, false, "acd", "ab**(e|f)g", bashOpts)
	})

	t.Run("acd should not match ab***ef", func(t *testing.T) {
		// extglobs-bash.js:1401
		assertMatch(t, false, "acd", "ab***ef", bashOpts)
	})

	t.Run("acd should not match ab*+(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1405
		assertMatch(t, false, "acd", "ab*+(e|f)", bashOpts)
	})

	t.Run("acd should not match ab*d+(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1409
		assertMatch(t, false, "acd", "ab*d+(e|f)", bashOpts)
	})

	t.Run("acd should not match ab?*(e|f)", func(t *testing.T) {
		// extglobs-bash.js:1413
		assertMatch(t, false, "acd", "ab?*(e|f)", bashOpts)
	})

	t.Run("ax should match ?(a*|b)", func(t *testing.T) {
		// extglobs-bash.js:1417
		assertMatch(t, true, "ax", "?(a*|b)", bashOpts)
	})

	t.Run("ax should not match a?(b*)", func(t *testing.T) {
		// extglobs-bash.js:1421
		assertMatch(t, false, "ax", "a?(b*)", bashOpts)
	})

	t.Run("axz should not match a+(z)", func(t *testing.T) {
		// extglobs-bash.js:1425
		assertMatch(t, false, "axz", "a+(z)", bashOpts)
	})

	t.Run("az should not match a!(*)", func(t *testing.T) {
		// extglobs-bash.js:1429
		assertMatch(t, false, "az", "a!(*)", bashOpts)
	})

	t.Run("az should not match a!(z)", func(t *testing.T) {
		// extglobs-bash.js:1433
		assertMatch(t, false, "az", "a!(z)", bashOpts)
	})

	t.Run("az should match a*!(z)", func(t *testing.T) {
		// extglobs-bash.js:1437
		assertMatch(t, true, "az", "a*!(z)", bashOpts)
	})

	t.Run("az should match a*(z)", func(t *testing.T) {
		// extglobs-bash.js:1441
		assertMatch(t, true, "az", "a*(z)", bashOpts)
	})

	t.Run("az should match a**(z)", func(t *testing.T) {
		// extglobs-bash.js:1445
		assertMatch(t, true, "az", "a**(z)", bashOpts)
	})

	t.Run("az should match a*@(z)", func(t *testing.T) {
		// extglobs-bash.js:1449
		assertMatch(t, true, "az", "a*@(z)", bashOpts)
	})

	t.Run("az should match a+(z)", func(t *testing.T) {
		// extglobs-bash.js:1453
		assertMatch(t, true, "az", "a+(z)", bashOpts)
	})

	t.Run("az should match a?(z)", func(t *testing.T) {
		// extglobs-bash.js:1457
		assertMatch(t, true, "az", "a?(z)", bashOpts)
	})

	t.Run("az should match a@(z)", func(t *testing.T) {
		// extglobs-bash.js:1461
		assertMatch(t, true, "az", "a@(z)", bashOpts)
	})

	t.Run("az should not match a\\\\\\\\z", func(t *testing.T) {
		// extglobs-bash.js:1465
		assertMatch(t, false, "az", "a\\\\z", bashNonWinOpts)
	})

	t.Run("az should not match a\\\\\\\\z (2)", func(t *testing.T) {
		// extglobs-bash.js:1469
		assertMatch(t, false, "az", "a\\\\z", bashOpts)
	})

	t.Run("b should match !(a)*", func(t *testing.T) {
		// extglobs-bash.js:1473
		assertMatch(t, true, "b", "!(a)*", bashOpts)
	})

	t.Run("b should match (a+|b)*", func(t *testing.T) {
		// extglobs-bash.js:1477
		assertMatch(t, true, "b", "(a+|b)*", bashOpts)
	})

	t.Run("b should not match a!(b)*", func(t *testing.T) {
		// extglobs-bash.js:1481
		assertMatch(t, false, "b", "a!(b)*", bashOpts)
	})

	t.Run("b.a should match (b|a).(a)", func(t *testing.T) {
		// extglobs-bash.js:1485
		assertMatch(t, true, "b.a", "(b|a).(a)", bashOpts)
	})

	t.Run("b.a should match @(b|a).@(a)", func(t *testing.T) {
		// extglobs-bash.js:1489
		assertMatch(t, true, "b.a", "@(b|a).@(a)", bashOpts)
	})

	t.Run("b/a should not match !(b/a)", func(t *testing.T) {
		// extglobs-bash.js:1493
		assertMatch(t, false, "b/a", "!(b/a)", bashOpts)
	})

	t.Run("b/b should match !(b/a)", func(t *testing.T) {
		// extglobs-bash.js:1497
		assertMatch(t, true, "b/b", "!(b/a)", bashOpts)
	})

	t.Run("b/c should match !(b/a)", func(t *testing.T) {
		// extglobs-bash.js:1501
		assertMatch(t, true, "b/c", "!(b/a)", bashOpts)
	})

	t.Run("b/c should not match b/!(c)", func(t *testing.T) {
		// extglobs-bash.js:1505
		assertMatch(t, false, "b/c", "b/!(c)", bashOpts)
	})

	t.Run("b/c should match b/!(cc)", func(t *testing.T) {
		// extglobs-bash.js:1509
		assertMatch(t, true, "b/c", "b/!(cc)", bashOpts)
	})

	t.Run("b/c.txt should not match b/!(c).txt", func(t *testing.T) {
		// extglobs-bash.js:1513
		assertMatch(t, false, "b/c.txt", "b/!(c).txt", bashOpts)
	})

	t.Run("b/c.txt should match b/!(cc).txt", func(t *testing.T) {
		// extglobs-bash.js:1517
		assertMatch(t, true, "b/c.txt", "b/!(cc).txt", bashOpts)
	})

	t.Run("b/cc should match b/!(c)", func(t *testing.T) {
		// extglobs-bash.js:1521
		assertMatch(t, true, "b/cc", "b/!(c)", bashOpts)
	})

	t.Run("b/cc should not match b/!(cc)", func(t *testing.T) {
		// extglobs-bash.js:1525
		assertMatch(t, false, "b/cc", "b/!(cc)", bashOpts)
	})

	t.Run("b/cc.txt should not match b/!(c).txt", func(t *testing.T) {
		// extglobs-bash.js:1529
		assertMatch(t, false, "b/cc.txt", "b/!(c).txt", bashOpts)
	})

	t.Run("b/cc.txt should not match b/!(cc).txt", func(t *testing.T) {
		// extglobs-bash.js:1533
		assertMatch(t, false, "b/cc.txt", "b/!(cc).txt", bashOpts)
	})

	t.Run("b/ccc should match b/!(c)", func(t *testing.T) {
		// extglobs-bash.js:1537
		assertMatch(t, true, "b/ccc", "b/!(c)", bashOpts)
	})

	t.Run("ba should match !(a!(b))", func(t *testing.T) {
		// extglobs-bash.js:1541
		assertMatch(t, true, "ba", "!(a!(b))", bashOpts)
	})

	t.Run("ba should match b?(a|b)", func(t *testing.T) {
		// extglobs-bash.js:1545
		assertMatch(t, true, "ba", "b?(a|b)", bashOpts)
	})

	t.Run("baaac should not match *(@(a))a@(c)", func(t *testing.T) {
		// extglobs-bash.js:1549
		assertMatch(t, false, "baaac", "*(@(a))a@(c)", bashOpts)
	})

	t.Run("bar should match !(foo)", func(t *testing.T) {
		// extglobs-bash.js:1553
		assertMatch(t, true, "bar", "!(foo)", bashOpts)
	})

	t.Run("bar should match !(foo)*", func(t *testing.T) {
		// extglobs-bash.js:1557
		assertMatch(t, true, "bar", "!(foo)*", bashOpts)
	})

	t.Run("bar should match !(foo)b*", func(t *testing.T) {
		// extglobs-bash.js:1561
		assertMatch(t, true, "bar", "!(foo)b*", bashOpts)
	})

	t.Run("bar should match *(!(foo))", func(t *testing.T) {
		// extglobs-bash.js:1565
		assertMatch(t, true, "bar", "*(!(foo))", bashOpts)
	})

	t.Run("baz should match !(foo)*", func(t *testing.T) {
		// extglobs-bash.js:1569
		assertMatch(t, true, "baz", "!(foo)*", bashOpts)
	})

	t.Run("baz should match !(foo)b*", func(t *testing.T) {
		// extglobs-bash.js:1573
		assertMatch(t, true, "baz", "!(foo)b*", bashOpts)
	})

	t.Run("baz should match *(!(foo))", func(t *testing.T) {
		// extglobs-bash.js:1577
		assertMatch(t, true, "baz", "*(!(foo))", bashOpts)
	})

	t.Run("bb should match !(a!(b))", func(t *testing.T) {
		// extglobs-bash.js:1581
		assertMatch(t, true, "bb", "!(a!(b))", bashOpts)
	})

	t.Run("bb should match !(a)*", func(t *testing.T) {
		// extglobs-bash.js:1585
		assertMatch(t, true, "bb", "!(a)*", bashOpts)
	})

	t.Run("bb should not match a!(b)*", func(t *testing.T) {
		// extglobs-bash.js:1589
		assertMatch(t, false, "bb", "a!(b)*", bashOpts)
	})

	t.Run("bb should not match a?(a|b)", func(t *testing.T) {
		// extglobs-bash.js:1593
		assertMatch(t, false, "bb", "a?(a|b)", bashOpts)
	})

	t.Run("bbc should match !([[*])*", func(t *testing.T) {
		// extglobs-bash.js:1597
		assertMatch(t, true, "bbc", "!([[*])*", bashOpts)
	})

	t.Run("bbc should not match +(a|b\\\\\\\\[)*", func(t *testing.T) {
		// extglobs-bash.js:1601
		assertMatch(t, false, "bbc", "+(a|b\\[)*", bashOpts)
	})

	t.Run("bbc should not match [a*(]*z", func(t *testing.T) {
		// extglobs-bash.js:1605
		assertMatch(t, false, "bbc", "[a*(]*z", bashOpts)
	})

	t.Run("bz should not match a+(z)", func(t *testing.T) {
		// extglobs-bash.js:1609
		assertMatch(t, false, "bz", "a+(z)", bashOpts)
	})

	t.Run("c should not match *(@(a))a@(c)", func(t *testing.T) {
		// extglobs-bash.js:1613
		assertMatch(t, false, "c", "*(@(a))a@(c)", bashOpts)
	})

	t.Run("c.a should not match !(*.[a-b]*)", func(t *testing.T) {
		// extglobs-bash.js:1617
		assertMatch(t, false, "c.a", "!(*.[a-b]*)", bashOpts)
	})

	t.Run("c.a should match !(*[a-b].[a-b]*)", func(t *testing.T) {
		// extglobs-bash.js:1621
		assertMatch(t, true, "c.a", "!(*[a-b].[a-b]*)", bashOpts)
	})

	t.Run("c.a should not match !*.(a|b)", func(t *testing.T) {
		// extglobs-bash.js:1625
		assertMatch(t, false, "c.a", "!*.(a|b)", bashOpts)
	})

	t.Run("c.a should not match !*.(a|b)*", func(t *testing.T) {
		// extglobs-bash.js:1629
		assertMatch(t, false, "c.a", "!*.(a|b)*", bashOpts)
	})

	t.Run("c.a should not match (b|a).(a)", func(t *testing.T) {
		// extglobs-bash.js:1633
		assertMatch(t, false, "c.a", "(b|a).(a)", bashOpts)
	})

	t.Run("c.a should not match *.!(a)", func(t *testing.T) {
		// extglobs-bash.js:1637
		assertMatch(t, false, "c.a", "*.!(a)", bashOpts)
	})

	t.Run("c.a should not match *.+(b|d)", func(t *testing.T) {
		// extglobs-bash.js:1641
		assertMatch(t, false, "c.a", "*.+(b|d)", bashOpts)
	})

	t.Run("c.a should not match @(b|a).@(a)", func(t *testing.T) {
		// extglobs-bash.js:1645
		assertMatch(t, false, "c.a", "@(b|a).@(a)", bashOpts)
	})

	t.Run("c.c should not match !(*.a|*.b|*.c)", func(t *testing.T) {
		// extglobs-bash.js:1649
		assertMatch(t, false, "c.c", "!(*.a|*.b|*.c)", bashOpts)
	})

	t.Run("c.c should match *!(.a|.b|.c)", func(t *testing.T) {
		// extglobs-bash.js:1653
		assertMatch(t, true, "c.c", "*!(.a|.b|.c)", bashOpts)
	})

	t.Run("c.c should not match *.!(a|b|c)", func(t *testing.T) {
		// extglobs-bash.js:1657
		assertMatch(t, false, "c.c", "*.!(a|b|c)", bashOpts)
	})

	t.Run("c.c should not match *.(a|b|@(ab|a*@(b))*(c)d)", func(t *testing.T) {
		// extglobs-bash.js:1661
		assertMatch(t, false, "c.c", "*.(a|b|@(ab|a*@(b))*(c)d)", bashOpts)
	})

	t.Run("c.ccc should match !(*.[a-b]*)", func(t *testing.T) {
		// extglobs-bash.js:1665
		assertMatch(t, true, "c.ccc", "!(*.[a-b]*)", bashOpts)
	})

	t.Run("c.ccc should match !(*[a-b].[a-b]*)", func(t *testing.T) {
		// extglobs-bash.js:1669
		assertMatch(t, true, "c.ccc", "!(*[a-b].[a-b]*)", bashOpts)
	})

	t.Run("c.js should not match !(*.js)", func(t *testing.T) {
		// extglobs-bash.js:1673
		assertMatch(t, false, "c.js", "!(*.js)", bashOpts)
	})

	t.Run("c.js should match *!(.js)", func(t *testing.T) {
		// extglobs-bash.js:1677
		assertMatch(t, true, "c.js", "*!(.js)", bashOpts)
	})

	t.Run("c.js should not match *.!(js)", func(t *testing.T) {
		// extglobs-bash.js:1681
		assertMatch(t, false, "c.js", "*.!(js)", bashOpts)
	})

	t.Run("c/a/v should match c/!(z)/v", func(t *testing.T) {
		// extglobs-bash.js:1685
		assertMatch(t, true, "c/a/v", "c/!(z)/v", bashOpts)
	})

	t.Run("c/a/v should not match c/*(z)/v", func(t *testing.T) {
		// extglobs-bash.js:1689
		assertMatch(t, false, "c/a/v", "c/*(z)/v", bashOpts)
	})

	t.Run("c/a/v should not match c/+(z)/v", func(t *testing.T) {
		// extglobs-bash.js:1693
		assertMatch(t, false, "c/a/v", "c/+(z)/v", bashOpts)
	})

	t.Run("c/a/v should not match c/@(z)/v", func(t *testing.T) {
		// extglobs-bash.js:1697
		assertMatch(t, false, "c/a/v", "c/@(z)/v", bashOpts)
	})

	t.Run("c/z/v should not match *(z)", func(t *testing.T) {
		// extglobs-bash.js:1701
		assertMatch(t, false, "c/z/v", "*(z)", bashOpts)
	})

	t.Run("c/z/v should not match +(z)", func(t *testing.T) {
		// extglobs-bash.js:1705
		assertMatch(t, false, "c/z/v", "+(z)", bashOpts)
	})

	t.Run("c/z/v should not match ?(z)", func(t *testing.T) {
		// extglobs-bash.js:1709
		assertMatch(t, false, "c/z/v", "?(z)", bashOpts)
	})

	t.Run("c/z/v should not match c/!(z)/v", func(t *testing.T) {
		// extglobs-bash.js:1713
		assertMatch(t, false, "c/z/v", "c/!(z)/v", bashOpts)
	})

	t.Run("c/z/v should match c/*(z)/v", func(t *testing.T) {
		// extglobs-bash.js:1717
		assertMatch(t, true, "c/z/v", "c/*(z)/v", bashOpts)
	})

	t.Run("c/z/v should match c/+(z)/v", func(t *testing.T) {
		// extglobs-bash.js:1721
		assertMatch(t, true, "c/z/v", "c/+(z)/v", bashOpts)
	})

	t.Run("c/z/v should match c/@(z)/v", func(t *testing.T) {
		// extglobs-bash.js:1725
		assertMatch(t, true, "c/z/v", "c/@(z)/v", bashOpts)
	})

	t.Run("c/z/v should match c/z/v", func(t *testing.T) {
		// extglobs-bash.js:1729
		assertMatch(t, true, "c/z/v", "c/z/v", bashOpts)
	})

	t.Run("cc.a should not match (b|a).(a)", func(t *testing.T) {
		// extglobs-bash.js:1733
		assertMatch(t, false, "cc.a", "(b|a).(a)", bashOpts)
	})

	t.Run("cc.a should not match @(b|a).@(a)", func(t *testing.T) {
		// extglobs-bash.js:1737
		assertMatch(t, false, "cc.a", "@(b|a).@(a)", bashOpts)
	})

	t.Run("ccc should match !(a)*", func(t *testing.T) {
		// extglobs-bash.js:1741
		assertMatch(t, true, "ccc", "!(a)*", bashOpts)
	})

	t.Run("ccc should not match a!(b)*", func(t *testing.T) {
		// extglobs-bash.js:1745
		assertMatch(t, false, "ccc", "a!(b)*", bashOpts)
	})

	t.Run("cow should match !(*.*)", func(t *testing.T) {
		// extglobs-bash.js:1749
		assertMatch(t, true, "cow", "!(*.*)", bashOpts)
	})

	t.Run("cow should not match !(*.*).", func(t *testing.T) {
		// extglobs-bash.js:1753
		assertMatch(t, false, "cow", "!(*.*).", bashOpts)
	})

	t.Run("cow should not match .!(*.*)", func(t *testing.T) {
		// extglobs-bash.js:1757
		assertMatch(t, false, "cow", ".!(*.*)", bashOpts)
	})

	t.Run("cz should not match a!(*)", func(t *testing.T) {
		// extglobs-bash.js:1761
		assertMatch(t, false, "cz", "a!(*)", bashOpts)
	})

	t.Run("cz should not match a!(z)", func(t *testing.T) {
		// extglobs-bash.js:1765
		assertMatch(t, false, "cz", "a!(z)", bashOpts)
	})

	t.Run("cz should not match a*!(z)", func(t *testing.T) {
		// extglobs-bash.js:1769
		assertMatch(t, false, "cz", "a*!(z)", bashOpts)
	})

	t.Run("cz should not match a*(z)", func(t *testing.T) {
		// extglobs-bash.js:1773
		assertMatch(t, false, "cz", "a*(z)", bashOpts)
	})

	t.Run("cz should not match a**(z)", func(t *testing.T) {
		// extglobs-bash.js:1777
		assertMatch(t, false, "cz", "a**(z)", bashOpts)
	})

	t.Run("cz should not match a*@(z)", func(t *testing.T) {
		// extglobs-bash.js:1781
		assertMatch(t, false, "cz", "a*@(z)", bashOpts)
	})

	t.Run("cz should not match a+(z)", func(t *testing.T) {
		// extglobs-bash.js:1785
		assertMatch(t, false, "cz", "a+(z)", bashOpts)
	})

	t.Run("cz should not match a?(z)", func(t *testing.T) {
		// extglobs-bash.js:1789
		assertMatch(t, false, "cz", "a?(z)", bashOpts)
	})

	t.Run("cz should not match a@(z)", func(t *testing.T) {
		// extglobs-bash.js:1793
		assertMatch(t, false, "cz", "a@(z)", bashOpts)
	})

	t.Run("d.a.d should not match !(*.[a-b]*)", func(t *testing.T) {
		// extglobs-bash.js:1797
		assertMatch(t, false, "d.a.d", "!(*.[a-b]*)", bashOpts)
	})

	t.Run("d.a.d should match !(*[a-b].[a-b]*)", func(t *testing.T) {
		// extglobs-bash.js:1801
		assertMatch(t, true, "d.a.d", "!(*[a-b].[a-b]*)", bashOpts)
	})

	t.Run("d.a.d should not match !*.(a|b)*", func(t *testing.T) {
		// extglobs-bash.js:1805
		assertMatch(t, false, "d.a.d", "!*.(a|b)*", bashOpts)
	})

	t.Run("d.a.d should match !*.*(a|b)", func(t *testing.T) {
		// extglobs-bash.js:1809
		assertMatch(t, true, "d.a.d", "!*.*(a|b)", bashOpts)
	})

	t.Run("d.a.d should not match !*.{a,b}*", func(t *testing.T) {
		// extglobs-bash.js:1813
		assertMatch(t, false, "d.a.d", "!*.{a,b}*", bashOpts)
	})

	t.Run("d.a.d should match *.!(a)", func(t *testing.T) {
		// extglobs-bash.js:1817
		assertMatch(t, true, "d.a.d", "*.!(a)", bashOpts)
	})

	t.Run("d.a.d should match *.+(b|d)", func(t *testing.T) {
		// extglobs-bash.js:1821
		assertMatch(t, true, "d.a.d", "*.+(b|d)", bashOpts)
	})

	t.Run("d.d should match !(*.a|*.b|*.c)", func(t *testing.T) {
		// extglobs-bash.js:1825
		assertMatch(t, true, "d.d", "!(*.a|*.b|*.c)", bashOpts)
	})

	t.Run("d.d should match *!(.a|.b|.c)", func(t *testing.T) {
		// extglobs-bash.js:1829
		assertMatch(t, true, "d.d", "*!(.a|.b|.c)", bashOpts)
	})

	t.Run("d.d should match *.!(a|b|c)", func(t *testing.T) {
		// extglobs-bash.js:1833
		assertMatch(t, true, "d.d", "*.!(a|b|c)", bashOpts)
	})

	t.Run("d.d should not match *.(a|b|@(ab|a*@(b))*(c)d)", func(t *testing.T) {
		// extglobs-bash.js:1837
		assertMatch(t, false, "d.d", "*.(a|b|@(ab|a*@(b))*(c)d)", bashOpts)
	})

	t.Run("d.js.d should match !(*.js)", func(t *testing.T) {
		// extglobs-bash.js:1841
		assertMatch(t, true, "d.js.d", "!(*.js)", bashOpts)
	})

	t.Run("d.js.d should match *!(.js)", func(t *testing.T) {
		// extglobs-bash.js:1845
		assertMatch(t, true, "d.js.d", "*!(.js)", bashOpts)
	})

	t.Run("d.js.d should match *.!(js)", func(t *testing.T) {
		// extglobs-bash.js:1849
		assertMatch(t, true, "d.js.d", "*.!(js)", bashOpts)
	})

	t.Run("dd.aa.d should not match (b|a).(a)", func(t *testing.T) {
		// extglobs-bash.js:1853
		assertMatch(t, false, "dd.aa.d", "(b|a).(a)", bashOpts)
	})

	t.Run("dd.aa.d should not match @(b|a).@(a)", func(t *testing.T) {
		// extglobs-bash.js:1857
		assertMatch(t, false, "dd.aa.d", "@(b|a).@(a)", bashOpts)
	})

	t.Run("def should not match ()ef", func(t *testing.T) {
		// extglobs-bash.js:1861
		assertMatch(t, false, "def", "()ef", bashOpts)
	})

	t.Run("e.e should match !(*.a|*.b|*.c)", func(t *testing.T) {
		// extglobs-bash.js:1865
		assertMatch(t, true, "e.e", "!(*.a|*.b|*.c)", bashOpts)
	})

	t.Run("e.e should match *!(.a|.b|.c)", func(t *testing.T) {
		// extglobs-bash.js:1869
		assertMatch(t, true, "e.e", "*!(.a|.b|.c)", bashOpts)
	})

	t.Run("e.e should match *.!(a|b|c)", func(t *testing.T) {
		// extglobs-bash.js:1873
		assertMatch(t, true, "e.e", "*.!(a|b|c)", bashOpts)
	})

	t.Run("e.e should not match *.(a|b|@(ab|a*@(b))*(c)d)", func(t *testing.T) {
		// extglobs-bash.js:1877
		assertMatch(t, false, "e.e", "*.(a|b|@(ab|a*@(b))*(c)d)", bashOpts)
	})

	t.Run("ef should match ()ef", func(t *testing.T) {
		// extglobs-bash.js:1881
		assertMatch(t, true, "ef", "()ef", bashOpts)
	})

	t.Run("effgz should match @(b+(c)d|e*(f)g?|?(h)i@(j|k))", func(t *testing.T) {
		// extglobs-bash.js:1885
		assertMatch(t, true, "effgz", "@(b+(c)d|e*(f)g?|?(h)i@(j|k))", bashOpts)
	})

	t.Run("efgz should match @(b+(c)d|e*(f)g?|?(h)i@(j|k))", func(t *testing.T) {
		// extglobs-bash.js:1889
		assertMatch(t, true, "efgz", "@(b+(c)d|e*(f)g?|?(h)i@(j|k))", bashOpts)
	})

	t.Run("egz should match @(b+(c)d|e*(f)g?|?(h)i@(j|k))", func(t *testing.T) {
		// extglobs-bash.js:1893
		assertMatch(t, true, "egz", "@(b+(c)d|e*(f)g?|?(h)i@(j|k))", bashOpts)
	})

	t.Run("egz should not match @(b+(c)d|e+(f)g?|?(h)i@(j|k))", func(t *testing.T) {
		// extglobs-bash.js:1897
		assertMatch(t, false, "egz", "@(b+(c)d|e+(f)g?|?(h)i@(j|k))", bashOpts)
	})

	t.Run("egzefffgzbcdij should match *(b+(c)d|e*(f)g?|?(h)i@(j|k))", func(t *testing.T) {
		// extglobs-bash.js:1901
		assertMatch(t, true, "egzefffgzbcdij", "*(b+(c)d|e*(f)g?|?(h)i@(j|k))", bashOpts)
	})

	t.Run("f should not match !(f!(o))", func(t *testing.T) {
		// extglobs-bash.js:1905
		assertMatch(t, false, "f", "!(f!(o))", bashOpts)
	})

	t.Run("f should match !(f(o))", func(t *testing.T) {
		// extglobs-bash.js:1909
		assertMatch(t, true, "f", "!(f(o))", bashOpts)
	})

	t.Run("f should not match !(f)", func(t *testing.T) {
		// extglobs-bash.js:1913
		assertMatch(t, false, "f", "!(f)", bashOpts)
	})

	t.Run("f should not match *(!(f))", func(t *testing.T) {
		// extglobs-bash.js:1917
		assertMatch(t, false, "f", "*(!(f))", bashOpts)
	})

	t.Run("f should not match +(!(f))", func(t *testing.T) {
		// extglobs-bash.js:1921
		assertMatch(t, false, "f", "+(!(f))", bashOpts)
	})

	t.Run("f.a should not match !(*.a|*.b|*.c)", func(t *testing.T) {
		// extglobs-bash.js:1925
		assertMatch(t, false, "f.a", "!(*.a|*.b|*.c)", bashOpts)
	})

	t.Run("f.a should match *!(.a|.b|.c)", func(t *testing.T) {
		// extglobs-bash.js:1929
		assertMatch(t, true, "f.a", "*!(.a|.b|.c)", bashOpts)
	})

	t.Run("f.a should not match *.!(a|b|c)", func(t *testing.T) {
		// extglobs-bash.js:1933
		assertMatch(t, false, "f.a", "*.!(a|b|c)", bashOpts)
	})

	t.Run("f.f should match !(*.a|*.b|*.c)", func(t *testing.T) {
		// extglobs-bash.js:1937
		assertMatch(t, true, "f.f", "!(*.a|*.b|*.c)", bashOpts)
	})

	t.Run("f.f should match *!(.a|.b|.c)", func(t *testing.T) {
		// extglobs-bash.js:1941
		assertMatch(t, true, "f.f", "*!(.a|.b|.c)", bashOpts)
	})

	t.Run("f.f should match *.!(a|b|c)", func(t *testing.T) {
		// extglobs-bash.js:1945
		assertMatch(t, true, "f.f", "*.!(a|b|c)", bashOpts)
	})

	t.Run("f.f should not match *.(a|b|@(ab|a*@(b))*(c)d)", func(t *testing.T) {
		// extglobs-bash.js:1949
		assertMatch(t, false, "f.f", "*.(a|b|@(ab|a*@(b))*(c)d)", bashOpts)
	})

	t.Run("fa should not match !(f!(o))", func(t *testing.T) {
		// extglobs-bash.js:1953
		assertMatch(t, false, "fa", "!(f!(o))", bashOpts)
	})

	t.Run("fa should match !(f(o))", func(t *testing.T) {
		// extglobs-bash.js:1957
		assertMatch(t, true, "fa", "!(f(o))", bashOpts)
	})

	t.Run("fb should not match !(f!(o))", func(t *testing.T) {
		// extglobs-bash.js:1961
		assertMatch(t, false, "fb", "!(f!(o))", bashOpts)
	})

	t.Run("fb should match !(f(o))", func(t *testing.T) {
		// extglobs-bash.js:1965
		assertMatch(t, true, "fb", "!(f(o))", bashOpts)
	})

	t.Run("fff should match !(f)", func(t *testing.T) {
		// extglobs-bash.js:1969
		assertMatch(t, true, "fff", "!(f)", bashOpts)
	})

	t.Run("fff should match *(!(f))", func(t *testing.T) {
		// extglobs-bash.js:1973
		assertMatch(t, true, "fff", "*(!(f))", bashOpts)
	})

	t.Run("fff should match +(!(f))", func(t *testing.T) {
		// extglobs-bash.js:1977
		assertMatch(t, true, "fff", "+(!(f))", bashOpts)
	})

	t.Run("fffooofoooooffoofffooofff should match *(*(f)*(o))", func(t *testing.T) {
		// extglobs-bash.js:1981
		assertMatch(t, true, "fffooofoooooffoofffooofff", "*(*(f)*(o))", bashOpts)
	})

	t.Run("ffo should match *(f*(o))", func(t *testing.T) {
		// extglobs-bash.js:1985
		assertMatch(t, true, "ffo", "*(f*(o))", bashOpts)
	})

	t.Run("file.C should not match *.c?(c)", func(t *testing.T) {
		// extglobs-bash.js:1989
		assertMatch(t, false, "file.C", "*.c?(c)", bashOpts)
	})

	t.Run("file.c should match *.c?(c)", func(t *testing.T) {
		// extglobs-bash.js:1993
		assertMatch(t, true, "file.c", "*.c?(c)", bashOpts)
	})

	t.Run("file.cc should match *.c?(c)", func(t *testing.T) {
		// extglobs-bash.js:1997
		assertMatch(t, true, "file.cc", "*.c?(c)", bashOpts)
	})

	t.Run("file.ccc should not match *.c?(c)", func(t *testing.T) {
		// extglobs-bash.js:2001
		assertMatch(t, false, "file.ccc", "*.c?(c)", bashOpts)
	})

	t.Run("fo should match !(f!(o))", func(t *testing.T) {
		// extglobs-bash.js:2005
		assertMatch(t, true, "fo", "!(f!(o))", bashOpts)
	})

	t.Run("fo should not match !(f(o))", func(t *testing.T) {
		// extglobs-bash.js:2009
		assertMatch(t, false, "fo", "!(f(o))", bashOpts)
	})

	t.Run("fofo should match *(f*(o))", func(t *testing.T) {
		// extglobs-bash.js:2013
		assertMatch(t, true, "fofo", "*(f*(o))", bashOpts)
	})

	t.Run("fofoofoofofoo should match *(fo|foo)", func(t *testing.T) {
		// extglobs-bash.js:2017
		assertMatch(t, true, "fofoofoofofoo", "*(fo|foo)", bashOpts)
	})

	t.Run("fofoofoofofoo should match *(fo|foo) (2)", func(t *testing.T) {
		// extglobs-bash.js:2021
		assertMatch(t, true, "fofoofoofofoo", "*(fo|foo)", bashOpts)
	})

	t.Run("foo should match !(!(foo))", func(t *testing.T) {
		// extglobs-bash.js:2025
		assertMatch(t, true, "foo", "!(!(foo))", bashOpts)
	})

	t.Run("foo should match !(f)", func(t *testing.T) {
		// extglobs-bash.js:2029
		assertMatch(t, true, "foo", "!(f)", bashOpts)
	})

	t.Run("foo should not match !(foo)", func(t *testing.T) {
		// extglobs-bash.js:2033
		assertMatch(t, false, "foo", "!(foo)", bashOpts)
	})

	t.Run("foo should not match !(foo)*", func(t *testing.T) {
		// extglobs-bash.js:2037
		assertMatch(t, false, "foo", "!(foo)*", bashOpts)
	})

	t.Run("foo should not match !(foo)* (2)", func(t *testing.T) {
		// extglobs-bash.js:2041
		assertMatch(t, false, "foo", "!(foo)*", bashOpts)
	})

	t.Run("foo should not match !(foo)+", func(t *testing.T) {
		// extglobs-bash.js:2045
		assertMatch(t, false, "foo", "!(foo)+", bashOpts)
	})

	t.Run("foo should not match !(foo)b*", func(t *testing.T) {
		// extglobs-bash.js:2049
		assertMatch(t, false, "foo", "!(foo)b*", bashOpts)
	})

	t.Run("foo should match !(x)", func(t *testing.T) {
		// extglobs-bash.js:2053
		assertMatch(t, true, "foo", "!(x)", bashOpts)
	})

	t.Run("foo should match !(x)*", func(t *testing.T) {
		// extglobs-bash.js:2057
		assertMatch(t, true, "foo", "!(x)*", bashOpts)
	})

	t.Run("foo should match *", func(t *testing.T) {
		// extglobs-bash.js:2061
		assertMatch(t, true, "foo", "*", bashOpts)
	})

	t.Run("foo should match *(!(f))", func(t *testing.T) {
		// extglobs-bash.js:2065
		assertMatch(t, true, "foo", "*(!(f))", bashOpts)
	})

	t.Run("foo should not match *(!(foo))", func(t *testing.T) {
		// extglobs-bash.js:2069
		assertMatch(t, false, "foo", "*(!(foo))", bashOpts)
	})

	t.Run("foo should not match *(@(a))a@(c)", func(t *testing.T) {
		// extglobs-bash.js:2073
		assertMatch(t, false, "foo", "*(@(a))a@(c)", bashOpts)
	})

	t.Run("foo should match *(@(foo))", func(t *testing.T) {
		// extglobs-bash.js:2077
		assertMatch(t, true, "foo", "*(@(foo))", bashOpts)
	})

	t.Run("foo should not match *(a|b\\\\\\\\[)", func(t *testing.T) {
		// extglobs-bash.js:2081
		assertMatch(t, false, "foo", "*(a|b\\[)", bashOpts)
	})

	t.Run("foo should match *(a|b\\\\\\\\[)|f*", func(t *testing.T) {
		// extglobs-bash.js:2085
		assertMatch(t, true, "foo", "*(a|b\\[)|f*", bashOpts)
	})

	t.Run("foo should match @(*(a|b\\\\\\\\[)|f*)", func(t *testing.T) {
		// extglobs-bash.js:2089
		assertMatch(t, true, "foo", "@(*(a|b\\[)|f*)", bashOpts)
	})

	t.Run("foo should not match */*/*", func(t *testing.T) {
		// extglobs-bash.js:2093
		assertMatch(t, false, "foo", "*/*/*", bashOpts)
	})

	t.Run("foo should not match *f", func(t *testing.T) {
		// extglobs-bash.js:2097
		assertMatch(t, false, "foo", "*f", bashOpts)
	})

	t.Run("foo should match *foo*", func(t *testing.T) {
		// extglobs-bash.js:2101
		assertMatch(t, true, "foo", "*foo*", bashOpts)
	})

	t.Run("foo should match +(!(f))", func(t *testing.T) {
		// extglobs-bash.js:2105
		assertMatch(t, true, "foo", "+(!(f))", bashOpts)
	})

	t.Run("foo should not match ??", func(t *testing.T) {
		// extglobs-bash.js:2109
		assertMatch(t, false, "foo", "??", bashOpts)
	})

	t.Run("foo should match ???", func(t *testing.T) {
		// extglobs-bash.js:2113
		assertMatch(t, true, "foo", "???", bashOpts)
	})

	t.Run("foo should not match bar", func(t *testing.T) {
		// extglobs-bash.js:2117
		assertMatch(t, false, "foo", "bar", bashOpts)
	})

	t.Run("foo should match f*", func(t *testing.T) {
		// extglobs-bash.js:2121
		assertMatch(t, true, "foo", "f*", bashOpts)
	})

	t.Run("foo should not match fo", func(t *testing.T) {
		// extglobs-bash.js:2125
		assertMatch(t, false, "foo", "fo", bashOpts)
	})

	t.Run("foo should match foo", func(t *testing.T) {
		// extglobs-bash.js:2129
		assertMatch(t, true, "foo", "foo", bashOpts)
	})

	t.Run("foo should match {*(a|b\\\\\\\\[),f*}", func(t *testing.T) {
		// extglobs-bash.js:2133
		assertMatch(t, true, "foo", "{*(a|b\\[),f*}", bashOpts)
	})

	t.Run("foo* should match foo\\\\\\\\*", func(t *testing.T) {
		// extglobs-bash.js:2137
		assertMatch(t, true, "foo*", "foo\\*", bashNonWinOpts)
	})

	t.Run("foo*bar should match foo\\\\\\\\*bar", func(t *testing.T) {
		// extglobs-bash.js:2141
		assertMatch(t, true, "foo*bar", "foo\\*bar", bashOpts)
	})

	t.Run("foo.js should not match !(foo).js", func(t *testing.T) {
		// extglobs-bash.js:2145
		assertMatch(t, false, "foo.js", "!(foo).js", bashOpts)
	})

	t.Run("foo.js.js should match *.!(js)", func(t *testing.T) {
		// extglobs-bash.js:2149
		assertMatch(t, true, "foo.js.js", "*.!(js)", bashOpts)
	})

	t.Run("foo.js.js should not match *.!(js)*", func(t *testing.T) {
		// extglobs-bash.js:2153
		assertMatch(t, false, "foo.js.js", "*.!(js)*", bashOpts)
	})

	t.Run("foo.js.js should not match *.!(js)*.!(js)", func(t *testing.T) {
		// extglobs-bash.js:2157
		assertMatch(t, false, "foo.js.js", "*.!(js)*.!(js)", bashOpts)
	})

	t.Run("foo.js.js should not match *.!(js)+", func(t *testing.T) {
		// extglobs-bash.js:2161
		assertMatch(t, false, "foo.js.js", "*.!(js)+", bashOpts)
	})

	t.Run("foo.txt should match **/!(bar).txt", func(t *testing.T) {
		// extglobs-bash.js:2165
		assertMatch(t, true, "foo.txt", "**/!(bar).txt", bashOpts)
	})

	t.Run("foo/bar should not match */*/*", func(t *testing.T) {
		// extglobs-bash.js:2169
		assertMatch(t, false, "foo/bar", "*/*/*", bashOpts)
	})

	t.Run("foo/bar should match foo/!(foo)", func(t *testing.T) {
		// extglobs-bash.js:2173
		assertMatch(t, true, "foo/bar", "foo/!(foo)", bashOpts)
	})

	t.Run("foo/bar should match foo/*", func(t *testing.T) {
		// extglobs-bash.js:2177
		assertMatch(t, true, "foo/bar", "foo/*", bashOpts)
	})

	t.Run("foo/bar should match foo/bar", func(t *testing.T) {
		// extglobs-bash.js:2181
		assertMatch(t, true, "foo/bar", "foo/bar", bashOpts)
	})

	t.Run("foo/bar should not match foo?bar", func(t *testing.T) {
		// extglobs-bash.js:2185
		assertMatch(t, false, "foo/bar", "foo?bar", bashOpts)
	})

	t.Run("foo/bar should match foo[/]bar", func(t *testing.T) {
		// extglobs-bash.js:2189
		assertMatch(t, true, "foo/bar", "foo[/]bar", bashOpts)
	})

	t.Run("foo/bar/baz.jsx should match foo/bar/**/*.+(js|jsx)", func(t *testing.T) {
		// extglobs-bash.js:2193
		assertMatch(t, true, "foo/bar/baz.jsx", "foo/bar/**/*.+(js|jsx)", bashOpts)
	})

	t.Run("foo/bar/baz.jsx should match foo/bar/*.+(js|jsx)", func(t *testing.T) {
		// extglobs-bash.js:2197
		assertMatch(t, true, "foo/bar/baz.jsx", "foo/bar/*.+(js|jsx)", bashOpts)
	})

	t.Run("foo/bb/aa/rr should match **/**/**", func(t *testing.T) {
		// extglobs-bash.js:2201
		assertMatch(t, true, "foo/bb/aa/rr", "**/**/**", bashOpts)
	})

	t.Run("foo/bb/aa/rr should match */*/*", func(t *testing.T) {
		// extglobs-bash.js:2205
		assertMatch(t, true, "foo/bb/aa/rr", "*/*/*", bashOpts)
	})

	t.Run("foo/bba/arr should match */*/*", func(t *testing.T) {
		// extglobs-bash.js:2209
		assertMatch(t, true, "foo/bba/arr", "*/*/*", bashOpts)
	})

	t.Run("foo/bba/arr should match foo*", func(t *testing.T) {
		// extglobs-bash.js:2213
		assertMatch(t, true, "foo/bba/arr", "foo*", bashOpts)
	})

	t.Run("foo/bba/arr should match foo**", func(t *testing.T) {
		// extglobs-bash.js:2217
		assertMatch(t, true, "foo/bba/arr", "foo**", bashOpts)
	})

	t.Run("foo/bba/arr should match foo/*", func(t *testing.T) {
		// extglobs-bash.js:2221
		assertMatch(t, true, "foo/bba/arr", "foo/*", bashOpts)
	})

	t.Run("foo/bba/arr should match foo/**", func(t *testing.T) {
		// extglobs-bash.js:2225
		assertMatch(t, true, "foo/bba/arr", "foo/**", bashOpts)
	})

	t.Run("foo/bba/arr should match foo/**arr", func(t *testing.T) {
		// extglobs-bash.js:2229
		assertMatch(t, true, "foo/bba/arr", "foo/**arr", bashOpts)
	})

	t.Run("foo/bba/arr should not match foo/**z", func(t *testing.T) {
		// extglobs-bash.js:2233
		assertMatch(t, false, "foo/bba/arr", "foo/**z", bashOpts)
	})

	t.Run("foo/bba/arr should match foo/*arr", func(t *testing.T) {
		// extglobs-bash.js:2237
		assertMatch(t, true, "foo/bba/arr", "foo/*arr", bashOpts)
	})

	t.Run("foo/bba/arr should not match foo/*z", func(t *testing.T) {
		// extglobs-bash.js:2241
		assertMatch(t, false, "foo/bba/arr", "foo/*z", bashOpts)
	})

	t.Run("foob should not match !(foo)b*", func(t *testing.T) {
		// extglobs-bash.js:2245
		assertMatch(t, false, "foob", "!(foo)b*", bashOpts)
	})

	t.Run("foob should not match (foo)bb", func(t *testing.T) {
		// extglobs-bash.js:2249
		assertMatch(t, false, "foob", "(foo)bb", bashOpts)
	})

	t.Run("foobar should match !(foo)", func(t *testing.T) {
		// extglobs-bash.js:2253
		assertMatch(t, true, "foobar", "!(foo)", bashOpts)
	})

	t.Run("foobar should not match !(foo)*", func(t *testing.T) {
		// extglobs-bash.js:2257
		assertMatch(t, false, "foobar", "!(foo)*", bashOpts)
	})

	t.Run("foobar should not match !(foo)* (2)", func(t *testing.T) {
		// extglobs-bash.js:2261
		assertMatch(t, false, "foobar", "!(foo)*", bashOpts)
	})

	t.Run("foobar should not match !(foo)b*", func(t *testing.T) {
		// extglobs-bash.js:2265
		assertMatch(t, false, "foobar", "!(foo)b*", bashOpts)
	})

	t.Run("foobar should match *(!(foo))", func(t *testing.T) {
		// extglobs-bash.js:2269
		assertMatch(t, true, "foobar", "*(!(foo))", bashOpts)
	})

	t.Run("foobar should match *ob*a*r*", func(t *testing.T) {
		// extglobs-bash.js:2273
		assertMatch(t, true, "foobar", "*ob*a*r*", bashOpts)
	})

	t.Run("foobar should match foo\\\\\\\\*bar", func(t *testing.T) {
		// extglobs-bash.js:2277
		assertMatch(t, true, "foobar", "foo*bar", bashOpts)
	})

	t.Run("foobb should not match !(foo)b*", func(t *testing.T) {
		// extglobs-bash.js:2281
		assertMatch(t, false, "foobb", "!(foo)b*", bashOpts)
	})

	t.Run("foobb should match (foo)bb", func(t *testing.T) {
		// extglobs-bash.js:2285
		assertMatch(t, true, "foobb", "(foo)bb", bashOpts)
	})

	t.Run("(foo)bb should match \\\\\\\\(foo\\\\\\\\)bb", func(t *testing.T) {
		// extglobs-bash.js:2289
		assertMatch(t, true, "(foo)bb", "\\(foo\\)bb", bashOpts)
	})

	t.Run("foofoofo should match @(foo|f|fo)*(f|of+(o))", func(t *testing.T) {
		// extglobs-bash.js:2293
		assertMatch(t, true, "foofoofo", "@(foo|f|fo)*(f|of+(o))", bashOpts)
	})

	t.Run("foofoofo should match @(foo|f|fo)*(f|of+(o)) (2)", func(t *testing.T) {
		// extglobs-bash.js:2297
		assertMatch(t, true, "foofoofo", "@(foo|f|fo)*(f|of+(o))", bashOpts)
	})

	t.Run("fooofoofofooo should match *(f*(o))", func(t *testing.T) {
		// extglobs-bash.js:2301
		assertMatch(t, true, "fooofoofofooo", "*(f*(o))", bashOpts)
	})

	t.Run("foooofo should match *(f*(o))", func(t *testing.T) {
		// extglobs-bash.js:2305
		assertMatch(t, true, "foooofo", "*(f*(o))", bashOpts)
	})

	t.Run("foooofof should match *(f*(o))", func(t *testing.T) {
		// extglobs-bash.js:2309
		assertMatch(t, true, "foooofof", "*(f*(o))", bashOpts)
	})

	t.Run("foooofof should not match *(f+(o))", func(t *testing.T) {
		// extglobs-bash.js:2313
		assertMatch(t, false, "foooofof", "*(f+(o))", bashOpts)
	})

	t.Run("foooofofx should not match *(f*(o))", func(t *testing.T) {
		// extglobs-bash.js:2317
		assertMatch(t, false, "foooofofx", "*(f*(o))", bashOpts)
	})

	t.Run("foooxfooxfoxfooox should match *(f*(o)x)", func(t *testing.T) {
		// extglobs-bash.js:2321
		assertMatch(t, true, "foooxfooxfoxfooox", "*(f*(o)x)", bashOpts)
	})

	t.Run("foooxfooxfxfooox should match *(f*(o)x)", func(t *testing.T) {
		// extglobs-bash.js:2325
		assertMatch(t, true, "foooxfooxfxfooox", "*(f*(o)x)", bashOpts)
	})

	t.Run("foooxfooxofoxfooox should not match *(f*(o)x)", func(t *testing.T) {
		// extglobs-bash.js:2329
		assertMatch(t, false, "foooxfooxofoxfooox", "*(f*(o)x)", bashOpts)
	})

	t.Run("foot should match @(!(z*)|*x)", func(t *testing.T) {
		// extglobs-bash.js:2333
		assertMatch(t, true, "foot", "@(!(z*)|*x)", bashOpts)
	})

	t.Run("foox should match @(!(z*)|*x)", func(t *testing.T) {
		// extglobs-bash.js:2337
		assertMatch(t, true, "foox", "@(!(z*)|*x)", bashOpts)
	})

	t.Run("fz should not match *(z)", func(t *testing.T) {
		// extglobs-bash.js:2341
		assertMatch(t, false, "fz", "*(z)", bashOpts)
	})

	t.Run("fz should not match +(z)", func(t *testing.T) {
		// extglobs-bash.js:2345
		assertMatch(t, false, "fz", "+(z)", bashOpts)
	})

	t.Run("fz should not match ?(z)", func(t *testing.T) {
		// extglobs-bash.js:2349
		assertMatch(t, false, "fz", "?(z)", bashOpts)
	})

	t.Run("moo.cow should not match !(moo).!(cow)", func(t *testing.T) {
		// extglobs-bash.js:2353
		assertMatch(t, false, "moo.cow", "!(moo).!(cow)", bashOpts)
	})

	t.Run("moo.cow should not match !(*).!(*)", func(t *testing.T) {
		// extglobs-bash.js:2357
		assertMatch(t, false, "moo.cow", "!(*).!(*)", bashOpts)
	})

	t.Run("moo.cow should not match !(*.*).!(*.*)", func(t *testing.T) {
		// extglobs-bash.js:2361
		assertMatch(t, false, "moo.cow", "!(*.*).!(*.*)", bashOpts)
	})

	t.Run("mad.moo.cow should not match !(*.*).!(*.*)", func(t *testing.T) {
		// extglobs-bash.js:2365
		assertMatch(t, false, "mad.moo.cow", "!(*.*).!(*.*)", bashOpts)
	})

	t.Run("mad.moo.cow should not match .!(*.*)", func(t *testing.T) {
		// extglobs-bash.js:2369
		assertMatch(t, false, "mad.moo.cow", ".!(*.*)", bashOpts)
	})

	t.Run("Makefile should match !(*.c|*.h|Makefile.in|config*|README)", func(t *testing.T) {
		// extglobs-bash.js:2373
		assertMatch(t, true, "Makefile", "!(*.c|*.h|Makefile.in|config*|README)", bashOpts)
	})

	t.Run("Makefile.in should not match !(*.c|*.h|Makefile.in|config*|README)", func(t *testing.T) {
		// extglobs-bash.js:2377
		assertMatch(t, false, "Makefile.in", "!(*.c|*.h|Makefile.in|config*|README)", bashOpts)
	})

	t.Run("moo should match !(*.*)", func(t *testing.T) {
		// extglobs-bash.js:2381
		assertMatch(t, true, "moo", "!(*.*)", bashOpts)
	})

	t.Run("moo should not match !(*.*).", func(t *testing.T) {
		// extglobs-bash.js:2385
		assertMatch(t, false, "moo", "!(*.*).", bashOpts)
	})

	t.Run("moo should not match .!(*.*)", func(t *testing.T) {
		// extglobs-bash.js:2389
		assertMatch(t, false, "moo", ".!(*.*)", bashOpts)
	})

	t.Run("moo.cow should not match !(*.*)", func(t *testing.T) {
		// extglobs-bash.js:2393
		assertMatch(t, false, "moo.cow", "!(*.*)", bashOpts)
	})

	t.Run("moo.cow should not match !(*.*).", func(t *testing.T) {
		// extglobs-bash.js:2397
		assertMatch(t, false, "moo.cow", "!(*.*).", bashOpts)
	})

	t.Run("moo.cow should not match .!(*.*)", func(t *testing.T) {
		// extglobs-bash.js:2401
		assertMatch(t, false, "moo.cow", ".!(*.*)", bashOpts)
	})

	t.Run("mucca.pazza should not match mu!(*(c))?.pa!(*(z))?", func(t *testing.T) {
		// extglobs-bash.js:2405
		assertMatch(t, false, "mucca.pazza", "mu!(*(c))?.pa!(*(z))?", bashOpts)
	})

	t.Run("ofoofo should match *(of+(o))", func(t *testing.T) {
		// extglobs-bash.js:2409
		assertMatch(t, true, "ofoofo", "*(of+(o))", bashOpts)
	})

	t.Run("ofoofo should match *(of+(o)|f)", func(t *testing.T) {
		// extglobs-bash.js:2413
		assertMatch(t, true, "ofoofo", "*(of+(o)|f)", bashOpts)
	})

	t.Run("ofooofoofofooo should not match *(f*(o))", func(t *testing.T) {
		// extglobs-bash.js:2417
		assertMatch(t, false, "ofooofoofofooo", "*(f*(o))", bashOpts)
	})

	t.Run("ofoooxoofxo should match *(*(of*(o)x)o)", func(t *testing.T) {
		// extglobs-bash.js:2421
		assertMatch(t, true, "ofoooxoofxo", "*(*(of*(o)x)o)", bashOpts)
	})

	t.Run("ofoooxoofxoofoooxoofxo should match *(*(of*(o)x)o)", func(t *testing.T) {
		// extglobs-bash.js:2425
		assertMatch(t, true, "ofoooxoofxoofoooxoofxo", "*(*(of*(o)x)o)", bashOpts)
	})

	t.Run("ofoooxoofxoofoooxoofxofo should not match *(*(of*(o)x)o)", func(t *testing.T) {
		// extglobs-bash.js:2429
		assertMatch(t, false, "ofoooxoofxoofoooxoofxofo", "*(*(of*(o)x)o)", bashOpts)
	})

	t.Run("ofoooxoofxoofoooxoofxoo should match *(*(of*(o)x)o)", func(t *testing.T) {
		// extglobs-bash.js:2433
		assertMatch(t, true, "ofoooxoofxoofoooxoofxoo", "*(*(of*(o)x)o)", bashOpts)
	})

	t.Run("ofoooxoofxoofoooxoofxooofxofxo should match *(*(of*(o)x)o)", func(t *testing.T) {
		// extglobs-bash.js:2437
		assertMatch(t, true, "ofoooxoofxoofoooxoofxooofxofxo", "*(*(of*(o)x)o)", bashOpts)
	})

	t.Run("ofxoofxo should match *(*(of*(o)x)o)", func(t *testing.T) {
		// extglobs-bash.js:2441
		assertMatch(t, true, "ofxoofxo", "*(*(of*(o)x)o)", bashOpts)
	})

	t.Run("oofooofo should match *(of|oof+(o))", func(t *testing.T) {
		// extglobs-bash.js:2445
		assertMatch(t, true, "oofooofo", "*(of|oof+(o))", bashOpts)
	})

	t.Run("ooo should match !(f)", func(t *testing.T) {
		// extglobs-bash.js:2449
		assertMatch(t, true, "ooo", "!(f)", bashOpts)
	})

	t.Run("ooo should match *(!(f))", func(t *testing.T) {
		// extglobs-bash.js:2453
		assertMatch(t, true, "ooo", "*(!(f))", bashOpts)
	})

	t.Run("ooo should match +(!(f))", func(t *testing.T) {
		// extglobs-bash.js:2457
		assertMatch(t, true, "ooo", "+(!(f))", bashOpts)
	})

	t.Run("oxfoxfox should not match *(oxf+(ox))", func(t *testing.T) {
		// extglobs-bash.js:2461
		assertMatch(t, false, "oxfoxfox", "*(oxf+(ox))", bashOpts)
	})

	t.Run("oxfoxoxfox should match *(oxf+(ox))", func(t *testing.T) {
		// extglobs-bash.js:2465
		assertMatch(t, true, "oxfoxoxfox", "*(oxf+(ox))", bashOpts)
	})

	t.Run("para should match para*([0-9])", func(t *testing.T) {
		// extglobs-bash.js:2469
		assertMatch(t, true, "para", "para*([0-9])", bashOpts)
	})

	t.Run("para should not match para+([0-9])", func(t *testing.T) {
		// extglobs-bash.js:2473
		assertMatch(t, false, "para", "para+([0-9])", bashOpts)
	})

	t.Run("para.38 should match para!(*.[00-09])", func(t *testing.T) {
		// extglobs-bash.js:2477
		assertMatch(t, true, "para.38", "para!(*.[00-09])", bashOpts)
	})

	t.Run("para.graph should match para!(*.[0-9])", func(t *testing.T) {
		// extglobs-bash.js:2481
		assertMatch(t, true, "para.graph", "para!(*.[0-9])", bashOpts)
	})

	t.Run("para13829383746592 should match para*([0-9])", func(t *testing.T) {
		// extglobs-bash.js:2485
		assertMatch(t, true, "para13829383746592", "para*([0-9])", bashOpts)
	})

	t.Run("para381 should not match para?([345]|99)1", func(t *testing.T) {
		// extglobs-bash.js:2489
		assertMatch(t, false, "para381", "para?([345]|99)1", bashOpts)
	})

	t.Run("para39 should match para!(*.[0-9])", func(t *testing.T) {
		// extglobs-bash.js:2493
		assertMatch(t, true, "para39", "para!(*.[0-9])", bashOpts)
	})

	t.Run("para987346523 should match para+([0-9])", func(t *testing.T) {
		// extglobs-bash.js:2497
		assertMatch(t, true, "para987346523", "para+([0-9])", bashOpts)
	})

	t.Run("para991 should match para?([345]|99)1", func(t *testing.T) {
		// extglobs-bash.js:2501
		assertMatch(t, true, "para991", "para?([345]|99)1", bashOpts)
	})

	t.Run("paragraph should match para!(*.[0-9])", func(t *testing.T) {
		// extglobs-bash.js:2505
		assertMatch(t, true, "paragraph", "para!(*.[0-9])", bashOpts)
	})

	t.Run("paragraph should not match para*([0-9])", func(t *testing.T) {
		// extglobs-bash.js:2509
		assertMatch(t, false, "paragraph", "para*([0-9])", bashOpts)
	})

	t.Run("paragraph should match para@(chute|graph)", func(t *testing.T) {
		// extglobs-bash.js:2513
		assertMatch(t, true, "paragraph", "para@(chute|graph)", bashOpts)
	})

	t.Run("paramour should not match para@(chute|graph)", func(t *testing.T) {
		// extglobs-bash.js:2517
		assertMatch(t, false, "paramour", "para@(chute|graph)", bashOpts)
	})

	t.Run("parse.y should match !(*.c|*.h|Makefile.in|config*|README)", func(t *testing.T) {
		// extglobs-bash.js:2521
		assertMatch(t, true, "parse.y", "!(*.c|*.h|Makefile.in|config*|README)", bashOpts)
	})

	t.Run("shell.c should not match !(*.c|*.h|Makefile.in|config*|README)", func(t *testing.T) {
		// extglobs-bash.js:2525
		assertMatch(t, false, "shell.c", "!(*.c|*.h|Makefile.in|config*|README)", bashOpts)
	})

	t.Run("VMS.FILE; should not match *\\\\\\\\;[1-9]*([0-9])", func(t *testing.T) {
		// extglobs-bash.js:2529
		assertMatch(t, false, "VMS.FILE;", "*\\;[1-9]*([0-9])", bashOpts)
	})

	t.Run("VMS.FILE;0 should not match *\\\\\\\\;[1-9]*([0-9])", func(t *testing.T) {
		// extglobs-bash.js:2533
		assertMatch(t, false, "VMS.FILE;0", "*\\;[1-9]*([0-9])", bashOpts)
	})

	t.Run("VMS.FILE;9 should match *\\\\\\\\;[1-9]*([0-9])", func(t *testing.T) {
		// extglobs-bash.js:2537
		assertMatch(t, true, "VMS.FILE;9", "*\\;[1-9]*([0-9])", bashOpts)
	})

	t.Run("VMS.FILE;1 should match *\\\\\\\\;[1-9]*([0-9])", func(t *testing.T) {
		// extglobs-bash.js:2541
		assertMatch(t, true, "VMS.FILE;1", "*\\;[1-9]*([0-9])", bashOpts)
	})

	t.Run("VMS.FILE;1 should match *;[1-9]*([0-9])", func(t *testing.T) {
		// extglobs-bash.js:2545
		assertMatch(t, true, "VMS.FILE;1", "*;[1-9]*([0-9])", bashOpts)
	})

	t.Run("VMS.FILE;139 should match *\\\\\\\\;[1-9]*([0-9])", func(t *testing.T) {
		// extglobs-bash.js:2549
		assertMatch(t, true, "VMS.FILE;139", "*\\;[1-9]*([0-9])", bashOpts)
	})

	t.Run("VMS.FILE;1N should not match *\\\\\\\\;[1-9]*([0-9])", func(t *testing.T) {
		// extglobs-bash.js:2553
		assertMatch(t, false, "VMS.FILE;1N", "*\\;[1-9]*([0-9])", bashOpts)
	})

	t.Run("xfoooofof should not match *(f*(o))", func(t *testing.T) {
		// extglobs-bash.js:2557
		assertMatch(t, false, "xfoooofof", "*(f*(o))", bashOpts)
	})

	t.Run("XXX/adobe/courier/bold/o/normal//12/120/75/75/m/70/iso8859/1 should match XXX/*/*/*/*/*/*/12/*/*/*/m/*/*/*", func(t *testing.T) {
		// extglobs-bash.js:2561
		assertMatch(t, true, "XXX/adobe/courier/bold/o/normal//12/120/75/75/m/70/iso8859/1", "XXX/*/*/*/*/*/*/12/*/*/*/m/*/*/*", bashNonWinOpts)
	})

	t.Run("XXX/adobe/courier/bold/o/normal//12/120/75/75/X/70/iso8859/1 should not match XXX/*/*/*/*/*/*/12/*/*/*/m/*/*/*", func(t *testing.T) {
		// extglobs-bash.js:2565
		assertMatch(t, false, "XXX/adobe/courier/bold/o/normal//12/120/75/75/X/70/iso8859/1", "XXX/*/*/*/*/*/*/12/*/*/*/m/*/*/*", bashOpts)
	})

	t.Run("z should match *(z)", func(t *testing.T) {
		// extglobs-bash.js:2569
		assertMatch(t, true, "z", "*(z)", bashOpts)
	})

	t.Run("z should match +(z)", func(t *testing.T) {
		// extglobs-bash.js:2573
		assertMatch(t, true, "z", "+(z)", bashOpts)
	})

	t.Run("z should match ?(z)", func(t *testing.T) {
		// extglobs-bash.js:2577
		assertMatch(t, true, "z", "?(z)", bashOpts)
	})

	t.Run("zf should not match *(z)", func(t *testing.T) {
		// extglobs-bash.js:2581
		assertMatch(t, false, "zf", "*(z)", bashOpts)
	})

	t.Run("zf should not match +(z)", func(t *testing.T) {
		// extglobs-bash.js:2585
		assertMatch(t, false, "zf", "+(z)", bashOpts)
	})

	t.Run("zf should not match ?(z)", func(t *testing.T) {
		// extglobs-bash.js:2589
		assertMatch(t, false, "zf", "?(z)", bashOpts)
	})

	t.Run("zoot should not match @(!(z*)|*x)", func(t *testing.T) {
		// extglobs-bash.js:2593
		assertMatch(t, false, "zoot", "@(!(z*)|*x)", bashOpts)
	})

	t.Run("zoox should match @(!(z*)|*x)", func(t *testing.T) {
		// extglobs-bash.js:2597
		assertMatch(t, true, "zoox", "@(!(z*)|*x)", bashOpts)
	})

	t.Run("zz should not match (a+|b)*", func(t *testing.T) {
		// extglobs-bash.js:2601
		assertMatch(t, false, "zz", "(a+|b)*", bashOpts)
	})

}
