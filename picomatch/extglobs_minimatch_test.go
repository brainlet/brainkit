package picomatch

// extglobs_minimatch_test.go — Ported from picomatch/test/extglobs-minimatch.js
// Some of tests were converted from bash 4.3, 4.4, and minimatch unit tests.
// 642 assertions total.

import (
	"testing"
)

func TestExtglobsMinimatch(t *testing.T) {

	// Source: extglobs-minimatch.js line 12
	t.Run("L12__not_matches___0_1_3_5_7_9_", func(t *testing.T) {
		assertMatch(t, false, "", "*(0|1|3|5|7|9)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 16
	t.Run("L16___a_b___not_matches___a_b____", func(t *testing.T) {
		assertMatch(t, false, "*(a|b[)", "*(a|b\\[)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 20
	t.Run("L20___a_b___matches_______a___b______", func(t *testing.T) {
		assertMatch(t, true, "*(a|b[)", "\\*\\(a\\|b\\[\\)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 24
	t.Run("L24_____matches__________", func(t *testing.T) {
		assertMatch(t, true, "***", "\\*\\*\\*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 28
	t.Run("L28__adobe_courier_bold_o_normal__12_120_75_75___70_iso8859_1_not_matches___________", func(t *testing.T) {
		assertMatch(t, false, "-adobe-courier-bold-o-normal--12-120-75-75-/-70-iso8859-1", "-*-*-*-*-*-*-12-*-*-*-m-*-*-*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 32
	t.Run("L32__adobe_courier_bold_o_normal__12_120_75_75_m_70_iso8859_1_matches______________1", func(t *testing.T) {
		assertMatch(t, true, "-adobe-courier-bold-o-normal--12-120-75-75-m-70-iso8859-1", "-*-*-*-*-*-*-12-*-*-*-m-*-*-*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 36
	t.Run("L36__adobe_courier_bold_o_normal__12_120_75_75_X_70_iso8859_1_not_matches___________", func(t *testing.T) {
		assertMatch(t, false, "-adobe-courier-bold-o-normal--12-120-75-75-X-70-iso8859-1", "-*-*-*-*-*-*-12-*-*-*-m-*-*-*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 40
	t.Run("L40__dev_udp_129_22_8_102_45_matches__dev_____tcp_udp_________", func(t *testing.T) {
		assertMatch(t, true, "/dev/udp/129.22.8.102/45", "/dev\\/@(tcp|udp)\\/*\\/*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 44
	t.Run("L44__x_y_z_matches__x_y_z", func(t *testing.T) {
		assertMatch(t, true, "/x/y/z", "/x/y/z", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 48
	t.Run("L48_0377_matches____0_7__", func(t *testing.T) {
		assertMatch(t, true, "0377", "+([0-7])", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 52
	t.Run("L52_07_matches____0_7__", func(t *testing.T) {
		assertMatch(t, true, "07", "+([0-7])", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 56
	t.Run("L56_09_not_matches____0_7__", func(t *testing.T) {
		assertMatch(t, false, "09", "+([0-7])", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 60
	t.Run("L60_1_matches_0__1_9____0_9__", func(t *testing.T) {
		assertMatch(t, true, "1", "0|[1-9]*([0-9])", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 64
	t.Run("L64_12_matches_0__1_9____0_9__", func(t *testing.T) {
		assertMatch(t, true, "12", "0|[1-9]*([0-9])", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 68
	t.Run("L68_123abc_not_matches__a__b__", func(t *testing.T) {
		assertMatch(t, false, "123abc", "(a+|b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 72
	t.Run("L72_123abc_not_matches__a__b__", func(t *testing.T) {
		assertMatch(t, false, "123abc", "(a+|b)+", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 76
	t.Run("L76_123abc_matches____a_bc", func(t *testing.T) {
		assertMatch(t, true, "123abc", "*?(a)bc", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 80
	t.Run("L80_123abc_not_matches_a_b__foo_bar__d", func(t *testing.T) {
		assertMatch(t, false, "123abc", "a(b*(foo|bar))d", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 84
	t.Run("L84_123abc_not_matches_ab__e_f_", func(t *testing.T) {
		assertMatch(t, false, "123abc", "ab*(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 88
	t.Run("L88_123abc_not_matches_ab__", func(t *testing.T) {
		assertMatch(t, false, "123abc", "ab**", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 92
	t.Run("L92_123abc_not_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, false, "123abc", "ab**(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 96
	t.Run("L96_123abc_not_matches_ab___e_f_g", func(t *testing.T) {
		assertMatch(t, false, "123abc", "ab**(e|f)g", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 100
	t.Run("L100_123abc_not_matches_ab___ef", func(t *testing.T) {
		assertMatch(t, false, "123abc", "ab***ef", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 104
	t.Run("L104_123abc_not_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, false, "123abc", "ab*+(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 108
	t.Run("L108_123abc_not_matches_ab_d__e_f_", func(t *testing.T) {
		assertMatch(t, false, "123abc", "ab*d+(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 112
	t.Run("L112_123abc_not_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, false, "123abc", "ab?*(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 116
	t.Run("L116_12abc_not_matches_0__1_9____0_9__", func(t *testing.T) {
		assertMatch(t, false, "12abc", "0|[1-9]*([0-9])", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 120
	t.Run("L120_137577991_matches___0_1_3_5_7_9_", func(t *testing.T) {
		assertMatch(t, true, "137577991", "*(0|1|3|5|7|9)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 124
	t.Run("L124_2468_not_matches___0_1_3_5_7_9_", func(t *testing.T) {
		assertMatch(t, false, "2468", "*(0|1|3|5|7|9)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 128
	t.Run("L128__a_b_matches________b", func(t *testing.T) {
		assertMatch(t, true, "?a?b", "\\??\\?b", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 132
	t.Run("L132___a__b__c_not_matches_abc", func(t *testing.T) {
		assertMatch(t, false, "\\a\\b\\c", "abc", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 136
	t.Run("L136_a_matches_____a___b___c_", func(t *testing.T) {
		assertMatch(t, true, "a", "!(*.a|*.b|*.c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 140
	t.Run("L140_a_not_matches___a_", func(t *testing.T) {
		assertMatch(t, false, "a", "!(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 144
	t.Run("L144_a_not_matches___a__", func(t *testing.T) {
		assertMatch(t, false, "a", "!(a)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 148
	t.Run("L148_a_matches__a_", func(t *testing.T) {
		assertMatch(t, true, "a", "(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 152
	t.Run("L152_a_not_matches__b_", func(t *testing.T) {
		assertMatch(t, false, "a", "(b)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 156
	t.Run("L156_a_matches___a_", func(t *testing.T) {
		assertMatch(t, true, "a", "*(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 160
	t.Run("L160_a_matches___a_", func(t *testing.T) {
		assertMatch(t, true, "a", "+(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 164
	t.Run("L164_a_matches__", func(t *testing.T) {
		assertMatch(t, true, "a", "?", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 168
	t.Run("L168_a_matches___a_b_", func(t *testing.T) {
		assertMatch(t, true, "a", "?(a|b)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 172
	t.Run("L172_a_not_matches___", func(t *testing.T) {
		assertMatch(t, false, "a", "??", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 176
	t.Run("L176_a_matches_a__b__", func(t *testing.T) {
		assertMatch(t, true, "a", "a!(b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 180
	t.Run("L180_a_matches_a__a_b_", func(t *testing.T) {
		assertMatch(t, true, "a", "a?(a|b)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 184
	t.Run("L184_a_matches_a__x_", func(t *testing.T) {
		assertMatch(t, true, "a", "a?(x)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 188
	t.Run("L188_a_not_matches_a__b", func(t *testing.T) {
		assertMatch(t, false, "a", "a??b", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 192
	t.Run("L192_a_not_matches_b__a_b_", func(t *testing.T) {
		assertMatch(t, false, "a", "b?(a|b)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 196
	t.Run("L196_a____b_matches_a__b", func(t *testing.T) {
		assertMatch(t, true, "a((((b", "a(*b", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 200
	t.Run("L200_a____b_not_matches_a_b", func(t *testing.T) {
		assertMatch(t, false, "a((((b", "a(b", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 204
	t.Run("L204_a____b_not_matches_a___b", func(t *testing.T) {
		assertMatch(t, false, "a((((b", "a\\(b", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 208
	t.Run("L208_a__b_matches_a__b", func(t *testing.T) {
		assertMatch(t, true, "a((b", "a(*b", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 212
	t.Run("L212_a__b_not_matches_a_b", func(t *testing.T) {
		assertMatch(t, false, "a((b", "a(b", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 216
	t.Run("L216_a__b_not_matches_a___b", func(t *testing.T) {
		assertMatch(t, false, "a((b", "a\\(b", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 220
	t.Run("L220_a_b_matches_a__b", func(t *testing.T) {
		assertMatch(t, true, "a(b", "a(*b", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 224
	t.Run("L224_a_b_matches_a_b", func(t *testing.T) {
		assertMatch(t, true, "a(b", "a(b", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 228
	t.Run("L228_a_b_matches_a___b", func(t *testing.T) {
		assertMatch(t, true, "a(b", "a\\(b", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 232
	t.Run("L232_a__matches_____a___b___c_", func(t *testing.T) {
		assertMatch(t, true, "a.", "!(*.a|*.b|*.c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 236
	t.Run("L236_a__matches_____a__b__c_", func(t *testing.T) {
		assertMatch(t, true, "a.", "*!(.a|.b|.c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 240
	t.Run("L240_a__matches_____a_", func(t *testing.T) {
		assertMatch(t, true, "a.", "*.!(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 244
	t.Run("L244_a__matches_____a_b_c_", func(t *testing.T) {
		assertMatch(t, true, "a.", "*.!(a|b|c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 248
	t.Run("L248_a__not_matches____a_b___ab_a___b____c_d_", func(t *testing.T) {
		assertMatch(t, false, "a.", "*.(a|b|@(ab|a*@(b))*(c)d)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 252
	t.Run("L252_a__not_matches_____b_d_", func(t *testing.T) {
		assertMatch(t, false, "a.", "*.+(b|d)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 256
	t.Run("L256_a_a_not_matches______a_b___", func(t *testing.T) {
		assertMatch(t, false, "a.a", "!(*.[a-b]*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 260
	t.Run("L260_a_a_not_matches_____a___b___c_", func(t *testing.T) {
		assertMatch(t, false, "a.a", "!(*.a|*.b|*.c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 264
	t.Run("L264_a_a_not_matches_____a_b___a_b___", func(t *testing.T) {
		assertMatch(t, false, "a.a", "!(*[a-b].[a-b]*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 268
	t.Run("L268_a_a_not_matches_____a_b_", func(t *testing.T) {
		assertMatch(t, false, "a.a", "!*.(a|b)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 272
	t.Run("L272_a_a_not_matches_____a_b__", func(t *testing.T) {
		assertMatch(t, false, "a.a", "!*.(a|b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 276
	t.Run("L276_a_a_matches__a_d___a_b__", func(t *testing.T) {
		assertMatch(t, true, "a.a", "(a|d).(a|b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 280
	t.Run("L280_a_a_matches__b_a___a_", func(t *testing.T) {
		assertMatch(t, true, "a.a", "(b|a).(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 284
	t.Run("L284_a_a_matches_____a__b__c_", func(t *testing.T) {
		assertMatch(t, true, "a.a", "*!(.a|.b|.c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 288
	t.Run("L288_a_a_not_matches_____a_", func(t *testing.T) {
		assertMatch(t, false, "a.a", "*.!(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 292
	t.Run("L292_a_a_not_matches_____a_b_c_", func(t *testing.T) {
		assertMatch(t, false, "a.a", "*.!(a|b|c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 296
	t.Run("L296_a_a_matches____a_b___ab_a___b____c_d_", func(t *testing.T) {
		assertMatch(t, true, "a.a", "*.(a|b|@(ab|a*@(b))*(c)d)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 300
	t.Run("L300_a_a_not_matches_____b_d_", func(t *testing.T) {
		assertMatch(t, false, "a.a", "*.+(b|d)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 304
	t.Run("L304_a_a_matches___b_a____a_", func(t *testing.T) {
		assertMatch(t, true, "a.a", "@(b|a).@(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 308
	t.Run("L308_a_a_a_not_matches______a_b___", func(t *testing.T) {
		assertMatch(t, false, "a.a.a", "!(*.[a-b]*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 312
	t.Run("L312_a_a_a_not_matches_____a_b___a_b___", func(t *testing.T) {
		assertMatch(t, false, "a.a.a", "!(*[a-b].[a-b]*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 316
	t.Run("L316_a_a_a_not_matches_____a_b_", func(t *testing.T) {
		assertMatch(t, false, "a.a.a", "!*.(a|b)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 320
	t.Run("L320_a_a_a_not_matches_____a_b__", func(t *testing.T) {
		assertMatch(t, false, "a.a.a", "!*.(a|b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 324
	t.Run("L324_a_a_a_matches_____a_", func(t *testing.T) {
		assertMatch(t, true, "a.a.a", "*.!(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 328
	t.Run("L328_a_a_a_not_matches_____b_d_", func(t *testing.T) {
		assertMatch(t, false, "a.a.a", "*.+(b|d)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 332
	t.Run("L332_a_aa_a_not_matches__b_a___a_", func(t *testing.T) {
		assertMatch(t, false, "a.aa.a", "(b|a).(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 336
	t.Run("L336_a_aa_a_not_matches___b_a____a_", func(t *testing.T) {
		assertMatch(t, false, "a.aa.a", "@(b|a).@(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 340
	t.Run("L340_a_abcd_matches_____a___b___c_", func(t *testing.T) {
		assertMatch(t, true, "a.abcd", "!(*.a|*.b|*.c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 344
	t.Run("L344_a_abcd_not_matches_____a___b___c__", func(t *testing.T) {
		assertMatch(t, false, "a.abcd", "!(*.a|*.b|*.c)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 348
	t.Run("L348_a_abcd_matches______a___b___c__", func(t *testing.T) {
		assertMatch(t, true, "a.abcd", "*!(*.a|*.b|*.c)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 352
	t.Run("L352_a_abcd_matches_____a__b__c_", func(t *testing.T) {
		assertMatch(t, true, "a.abcd", "*!(.a|.b|.c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 356
	t.Run("L356_a_abcd_matches_____a_b_c_", func(t *testing.T) {
		assertMatch(t, true, "a.abcd", "*.!(a|b|c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 360
	t.Run("L360_a_abcd_not_matches_____a_b_c__", func(t *testing.T) {
		assertMatch(t, false, "a.abcd", "*.!(a|b|c)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 364
	t.Run("L364_a_abcd_matches____a_b___ab_a___b____c_d_", func(t *testing.T) {
		assertMatch(t, true, "a.abcd", "*.(a|b|@(ab|a*@(b))*(c)d)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 368
	t.Run("L368_a_b_not_matches_______", func(t *testing.T) {
		assertMatch(t, false, "a.b", "!(*.*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 372
	t.Run("L372_a_b_not_matches______a_b___", func(t *testing.T) {
		assertMatch(t, false, "a.b", "!(*.[a-b]*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 376
	t.Run("L376_a_b_not_matches_____a___b___c_", func(t *testing.T) {
		assertMatch(t, false, "a.b", "!(*.a|*.b|*.c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 380
	t.Run("L380_a_b_not_matches_____a_b___a_b___", func(t *testing.T) {
		assertMatch(t, false, "a.b", "!(*[a-b].[a-b]*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 384
	t.Run("L384_a_b_not_matches_____a_b_", func(t *testing.T) {
		assertMatch(t, false, "a.b", "!*.(a|b)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 388
	t.Run("L388_a_b_not_matches_____a_b__", func(t *testing.T) {
		assertMatch(t, false, "a.b", "!*.(a|b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 392
	t.Run("L392_a_b_matches__a_d___a_b__", func(t *testing.T) {
		assertMatch(t, true, "a.b", "(a|d).(a|b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 396
	t.Run("L396_a_b_matches_____a__b__c_", func(t *testing.T) {
		assertMatch(t, true, "a.b", "*!(.a|.b|.c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 400
	t.Run("L400_a_b_matches_____a_", func(t *testing.T) {
		assertMatch(t, true, "a.b", "*.!(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 404
	t.Run("L404_a_b_not_matches_____a_b_c_", func(t *testing.T) {
		assertMatch(t, false, "a.b", "*.!(a|b|c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 408
	t.Run("L408_a_b_matches____a_b___ab_a___b____c_d_", func(t *testing.T) {
		assertMatch(t, true, "a.b", "*.(a|b|@(ab|a*@(b))*(c)d)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 412
	t.Run("L412_a_b_matches_____b_d_", func(t *testing.T) {
		assertMatch(t, true, "a.b", "*.+(b|d)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 416
	t.Run("L416_a_bb_not_matches______a_b___", func(t *testing.T) {
		assertMatch(t, false, "a.bb", "!(*.[a-b]*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 420
	t.Run("L420_a_bb_not_matches_____a_b___a_b___", func(t *testing.T) {
		assertMatch(t, false, "a.bb", "!(*[a-b].[a-b]*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 424
	t.Run("L424_a_bb_matches_____a_b_", func(t *testing.T) {
		assertMatch(t, true, "a.bb", "!*.(a|b)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 428
	t.Run("L428_a_bb_not_matches_____a_b__", func(t *testing.T) {
		assertMatch(t, false, "a.bb", "!*.(a|b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 432
	t.Run("L432_a_bb_not_matches______a_b_", func(t *testing.T) {
		assertMatch(t, false, "a.bb", "!*.*(a|b)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 436
	t.Run("L436_a_bb_matches__a_d___a_b__", func(t *testing.T) {
		assertMatch(t, true, "a.bb", "(a|d).(a|b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 440
	t.Run("L440_a_bb_not_matches__b_a___a_", func(t *testing.T) {
		assertMatch(t, false, "a.bb", "(b|a).(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 444
	t.Run("L444_a_bb_matches_____b_d_", func(t *testing.T) {
		assertMatch(t, true, "a.bb", "*.+(b|d)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 448
	t.Run("L448_a_bb_not_matches___b_a____a_", func(t *testing.T) {
		assertMatch(t, false, "a.bb", "@(b|a).@(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 452
	t.Run("L452_a_c_not_matches_____a___b___c_", func(t *testing.T) {
		assertMatch(t, false, "a.c", "!(*.a|*.b|*.c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 456
	t.Run("L456_a_c_matches_____a__b__c_", func(t *testing.T) {
		assertMatch(t, true, "a.c", "*!(.a|.b|.c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 460
	t.Run("L460_a_c_not_matches_____a_b_c_", func(t *testing.T) {
		assertMatch(t, false, "a.c", "*.!(a|b|c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 464
	t.Run("L464_a_c_not_matches____a_b___ab_a___b____c_d_", func(t *testing.T) {
		assertMatch(t, false, "a.c", "*.(a|b|@(ab|a*@(b))*(c)d)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 468
	t.Run("L468_a_c_d_matches_____a___b___c_", func(t *testing.T) {
		assertMatch(t, true, "a.c.d", "!(*.a|*.b|*.c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 472
	t.Run("L472_a_c_d_matches_____a__b__c_", func(t *testing.T) {
		assertMatch(t, true, "a.c.d", "*!(.a|.b|.c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 476
	t.Run("L476_a_c_d_matches_____a_b_c_", func(t *testing.T) {
		assertMatch(t, true, "a.c.d", "*.!(a|b|c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 480
	t.Run("L480_a_c_d_not_matches____a_b___ab_a___b____c_d_", func(t *testing.T) {
		assertMatch(t, false, "a.c.d", "*.(a|b|@(ab|a*@(b))*(c)d)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 484
	t.Run("L484_a_ccc_matches______a_b___", func(t *testing.T) {
		assertMatch(t, true, "a.ccc", "!(*.[a-b]*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 488
	t.Run("L488_a_ccc_matches_____a_b___a_b___", func(t *testing.T) {
		assertMatch(t, true, "a.ccc", "!(*[a-b].[a-b]*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 492
	t.Run("L492_a_ccc_matches_____a_b_", func(t *testing.T) {
		assertMatch(t, true, "a.ccc", "!*.(a|b)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 496
	t.Run("L496_a_ccc_matches_____a_b__", func(t *testing.T) {
		assertMatch(t, true, "a.ccc", "!*.(a|b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 500
	t.Run("L500_a_ccc_not_matches_____b_d_", func(t *testing.T) {
		assertMatch(t, false, "a.ccc", "*.+(b|d)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 504
	t.Run("L504_a_js_not_matches_____js_", func(t *testing.T) {
		assertMatch(t, false, "a.js", "!(*.js)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 508
	t.Run("L508_a_js_matches_____js_", func(t *testing.T) {
		assertMatch(t, true, "a.js", "*!(.js)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 512
	t.Run("L512_a_js_not_matches_____js_", func(t *testing.T) {
		assertMatch(t, false, "a.js", "*.!(js)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 516
	t.Run("L516_a_js_not_matches_a___js_", func(t *testing.T) {
		assertMatch(t, false, "a.js", "a.!(js)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 520
	t.Run("L520_a_js_not_matches_a___js__", func(t *testing.T) {
		assertMatch(t, false, "a.js", "a.!(js)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 524
	t.Run("L524_a_js_js_not_matches_____js_", func(t *testing.T) {
		assertMatch(t, false, "a.js.js", "!(*.js)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 528
	t.Run("L528_a_js_js_matches_____js_", func(t *testing.T) {
		assertMatch(t, true, "a.js.js", "*!(.js)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 532
	t.Run("L532_a_js_js_matches_____js_", func(t *testing.T) {
		assertMatch(t, true, "a.js.js", "*.!(js)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 536
	t.Run("L536_a_js_js_matches_____js__js", func(t *testing.T) {
		assertMatch(t, true, "a.js.js", "*.*(js).js", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 540
	t.Run("L540_a_md_matches_____js_", func(t *testing.T) {
		assertMatch(t, true, "a.md", "!(*.js)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 544
	t.Run("L544_a_md_matches_____js_", func(t *testing.T) {
		assertMatch(t, true, "a.md", "*!(.js)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 548
	t.Run("L548_a_md_matches_____js_", func(t *testing.T) {
		assertMatch(t, true, "a.md", "*.!(js)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 552
	t.Run("L552_a_md_matches_a___js_", func(t *testing.T) {
		assertMatch(t, true, "a.md", "a.!(js)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 556
	t.Run("L556_a_md_matches_a___js__", func(t *testing.T) {
		assertMatch(t, true, "a.md", "a.!(js)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 560
	t.Run("L560_a_md_js_not_matches_____js__js", func(t *testing.T) {
		assertMatch(t, false, "a.md.js", "*.*(js).js", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 564
	t.Run("L564_a_txt_matches_a___js_", func(t *testing.T) {
		assertMatch(t, true, "a.txt", "a.!(js)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 568
	t.Run("L568_a_txt_matches_a___js__", func(t *testing.T) {
		assertMatch(t, true, "a.txt", "a.!(js)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 572
	t.Run("L572_a___z__matches_a___z_", func(t *testing.T) {
		assertMatch(t, true, "a/!(z)", "a/!(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 576
	t.Run("L576_a_b_matches_a___z_", func(t *testing.T) {
		assertMatch(t, true, "a/b", "a/!(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 580
	t.Run("L580_a_b_c_txt_not_matches___b______txt", func(t *testing.T) {
		assertMatch(t, false, "a/b/c.txt", "*/b/!(*).txt", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 584
	t.Run("L584_a_b_c_txt_not_matches___b___c__txt", func(t *testing.T) {
		assertMatch(t, false, "a/b/c.txt", "*/b/!(c).txt", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 588
	t.Run("L588_a_b_c_txt_matches___b___cc__txt", func(t *testing.T) {
		assertMatch(t, true, "a/b/c.txt", "*/b/!(cc).txt", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 592
	t.Run("L592_a_b_cc_txt_not_matches___b______txt", func(t *testing.T) {
		assertMatch(t, false, "a/b/cc.txt", "*/b/!(*).txt", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 596
	t.Run("L596_a_b_cc_txt_not_matches___b___c__txt", func(t *testing.T) {
		assertMatch(t, false, "a/b/cc.txt", "*/b/!(c).txt", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 600
	t.Run("L600_a_b_cc_txt_not_matches___b___cc__txt", func(t *testing.T) {
		assertMatch(t, false, "a/b/cc.txt", "*/b/!(cc).txt", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 604
	t.Run("L604_a_dir_foo_txt_matches___dir______bar__txt", func(t *testing.T) {
		assertMatch(t, true, "a/dir/foo.txt", "*/dir/**/!(bar).txt", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 608
	t.Run("L608_a_z_not_matches_a___z_", func(t *testing.T) {
		assertMatch(t, false, "a/z", "a/!(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 612
	t.Run("L612_a___b_not_matches_a__b", func(t *testing.T) {
		assertMatch(t, false, "a\\(b", "a(*b", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 616
	t.Run("L616_a___b_not_matches_a_b", func(t *testing.T) {
		assertMatch(t, false, "a\\(b", "a(b", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 620
	t.Run("L620_a____z_matches_a____z", func(t *testing.T) {
		assertMatch(t, true, "a\\\\z", "a\\\\z", &Options{Windows: false})
	})

	// Source: extglobs-minimatch.js line 624
	t.Run("L624_a____z_matches_a____z", func(t *testing.T) {
		assertMatch(t, true, "a\\\\z", "a\\\\z", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 628
	t.Run("L628_a__b_matches_a_b", func(t *testing.T) {
		assertMatch(t, true, "a\\b", "a/b", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 632
	t.Run("L632_a__z_matches_a____z", func(t *testing.T) {
		assertMatch(t, true, "a\\z", "a\\\\z", &Options{Windows: false})
	})

	// Source: extglobs-minimatch.js line 636
	t.Run("L636_a__z_matches_a__z", func(t *testing.T) {
		assertMatch(t, true, "a\\z", "a\\z", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 640
	t.Run("L640_aa_not_matches___a__b__", func(t *testing.T) {
		assertMatch(t, false, "aa", "!(a!(b))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 644
	t.Run("L644_aa_matches___a_", func(t *testing.T) {
		assertMatch(t, true, "aa", "!(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 648
	t.Run("L648_aa_not_matches___a__", func(t *testing.T) {
		assertMatch(t, false, "aa", "!(a)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 652
	t.Run("L652_aa_not_matches__", func(t *testing.T) {
		assertMatch(t, false, "aa", "?", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 656
	t.Run("L656_aa_not_matches___a_b", func(t *testing.T) {
		assertMatch(t, false, "aa", "@(a)b", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 660
	t.Run("L660_aa_matches_a__b__", func(t *testing.T) {
		assertMatch(t, true, "aa", "a!(b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 664
	t.Run("L664_aa_not_matches_a__b", func(t *testing.T) {
		assertMatch(t, false, "aa", "a??b", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 668
	t.Run("L668_aa_aa_not_matches__b_a___a_", func(t *testing.T) {
		assertMatch(t, false, "aa.aa", "(b|a).(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 672
	t.Run("L672_aa_aa_not_matches___b_a____a_", func(t *testing.T) {
		assertMatch(t, false, "aa.aa", "@(b|a).@(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 676
	t.Run("L676_aaa_not_matches___a__", func(t *testing.T) {
		assertMatch(t, false, "aaa", "!(a)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 680
	t.Run("L680_aaa_matches_a__b__", func(t *testing.T) {
		assertMatch(t, true, "aaa", "a!(b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 684
	t.Run("L684_aaaaaaabababab_matches__ab", func(t *testing.T) {
		assertMatch(t, true, "aaaaaaabababab", "*ab", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 688
	t.Run("L688_aaac_matches_____a__a__c_", func(t *testing.T) {
		assertMatch(t, true, "aaac", "*(@(a))a@(c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 692
	t.Run("L692_aaaz_matches__a____z", func(t *testing.T) {
		assertMatch(t, true, "aaaz", "[a*(]*z", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 696
	t.Run("L696_aab_not_matches___a__", func(t *testing.T) {
		assertMatch(t, false, "aab", "!(a)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 700
	t.Run("L700_aab_not_matches__", func(t *testing.T) {
		assertMatch(t, false, "aab", "?", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 704
	t.Run("L704_aab_not_matches___", func(t *testing.T) {
		assertMatch(t, false, "aab", "??", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 708
	t.Run("L708_aab_not_matches___c_b", func(t *testing.T) {
		assertMatch(t, false, "aab", "@(c)b", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 712
	t.Run("L712_aab_matches_a__b__", func(t *testing.T) {
		assertMatch(t, true, "aab", "a!(b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 716
	t.Run("L716_aab_not_matches_a__b", func(t *testing.T) {
		assertMatch(t, false, "aab", "a??b", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 720
	t.Run("L720_aac_matches_____a__a__c_", func(t *testing.T) {
		assertMatch(t, true, "aac", "*(@(a))a@(c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 724
	t.Run("L724_aac_not_matches_____a__b__c_", func(t *testing.T) {
		assertMatch(t, false, "aac", "*(@(a))b@(c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 728
	t.Run("L728_aax_not_matches_a__a__b_", func(t *testing.T) {
		assertMatch(t, false, "aax", "a!(a*|b)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 732
	t.Run("L732_aax_matches_a__x__b_", func(t *testing.T) {
		assertMatch(t, true, "aax", "a!(x*|b)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 736
	t.Run("L736_aax_matches_a__a__b_", func(t *testing.T) {
		assertMatch(t, true, "aax", "a?(a*|b)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 740
	t.Run("L740_aaz_matches__a____z", func(t *testing.T) {
		assertMatch(t, true, "aaz", "[a*(]*z", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 744
	t.Run("L744_ab_matches_______", func(t *testing.T) {
		assertMatch(t, true, "ab", "!(*.*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 748
	t.Run("L748_ab_matches___a__b__", func(t *testing.T) {
		assertMatch(t, true, "ab", "!(a!(b))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 752
	t.Run("L752_ab_not_matches___a__", func(t *testing.T) {
		assertMatch(t, false, "ab", "!(a)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 756
	t.Run("L756_ab_matches__a__b__", func(t *testing.T) {
		assertMatch(t, true, "ab", "(a+|b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 760
	t.Run("L760_ab_matches__a__b__", func(t *testing.T) {
		assertMatch(t, true, "ab", "(a+|b)+", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 764
	t.Run("L764_ab_not_matches____a_bc", func(t *testing.T) {
		assertMatch(t, false, "ab", "*?(a)bc", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 768
	t.Run("L768_ab_not_matches_a____b_B__", func(t *testing.T) {
		assertMatch(t, false, "ab", "a!(*(b|B))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 772
	t.Run("L772_ab_not_matches_a____b_B__", func(t *testing.T) {
		assertMatch(t, false, "ab", "a!(@(b|B))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 776
	t.Run("L776_aB_not_matches_a____b_B__", func(t *testing.T) {
		assertMatch(t, false, "aB", "a!(@(b|B))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 780
	t.Run("L780_ab_not_matches_a__b__", func(t *testing.T) {
		assertMatch(t, false, "ab", "a!(b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 784
	t.Run("L784_ab_not_matches_a__b", func(t *testing.T) {
		assertMatch(t, false, "ab", "a(*b", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 788
	t.Run("L788_ab_not_matches_a_b", func(t *testing.T) {
		assertMatch(t, false, "ab", "a(b", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 792
	t.Run("L792_ab_not_matches_a_b__foo_bar__d", func(t *testing.T) {
		assertMatch(t, false, "ab", "a(b*(foo|bar))d", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 796
	t.Run("L796_ab_not_matches_a_b", func(t *testing.T) {
		assertMatch(t, false, "ab", "a/b", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 800
	t.Run("L800_ab_not_matches_a___b", func(t *testing.T) {
		assertMatch(t, false, "ab", "a\\(b", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 804
	t.Run("L804_ab_matches_ab__e_f_", func(t *testing.T) {
		assertMatch(t, true, "ab", "ab*(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 808
	t.Run("L808_ab_matches_ab__", func(t *testing.T) {
		assertMatch(t, true, "ab", "ab**", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 812
	t.Run("L812_ab_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, true, "ab", "ab**(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 816
	t.Run("L816_ab_not_matches_ab___e_f_g", func(t *testing.T) {
		assertMatch(t, false, "ab", "ab**(e|f)g", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 820
	t.Run("L820_ab_not_matches_ab___ef", func(t *testing.T) {
		assertMatch(t, false, "ab", "ab***ef", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 824
	t.Run("L824_ab_not_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, false, "ab", "ab*+(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 828
	t.Run("L828_ab_not_matches_ab_d__e_f_", func(t *testing.T) {
		assertMatch(t, false, "ab", "ab*d+(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 832
	t.Run("L832_ab_not_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, false, "ab", "ab?*(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 836
	t.Run("L836_ab_cXd_efXg_hi_matches_____X______i", func(t *testing.T) {
		assertMatch(t, true, "ab/cXd/efXg/hi", "**/*X*/**/*i", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 840
	t.Run("L840_ab_cXd_efXg_hi_matches____X_____i", func(t *testing.T) {
		assertMatch(t, true, "ab/cXd/efXg/hi", "*/*X*/*/*i", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 844
	t.Run("L844_ab_cXd_efXg_hi_not_matches__X_i", func(t *testing.T) {
		assertMatch(t, false, "ab/cXd/efXg/hi", "*X*i", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 848
	t.Run("L848_ab_cXd_efXg_hi_not_matches__Xg_i", func(t *testing.T) {
		assertMatch(t, false, "ab/cXd/efXg/hi", "*Xg*i", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 852
	t.Run("L852_ab__matches_a____b_B__", func(t *testing.T) {
		assertMatch(t, true, "ab]", "a!(@(b|B))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 856
	t.Run("L856_abab_matches__a__b__", func(t *testing.T) {
		assertMatch(t, true, "abab", "(a+|b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 860
	t.Run("L860_abab_matches__a__b__", func(t *testing.T) {
		assertMatch(t, true, "abab", "(a+|b)+", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 864
	t.Run("L864_abab_not_matches____a_bc", func(t *testing.T) {
		assertMatch(t, false, "abab", "*?(a)bc", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 868
	t.Run("L868_abab_not_matches_a_b__foo_bar__d", func(t *testing.T) {
		assertMatch(t, false, "abab", "a(b*(foo|bar))d", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 872
	t.Run("L872_abab_not_matches_ab__e_f_", func(t *testing.T) {
		assertMatch(t, false, "abab", "ab*(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 876
	t.Run("L876_abab_matches_ab__", func(t *testing.T) {
		assertMatch(t, true, "abab", "ab**", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 880
	t.Run("L880_abab_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, true, "abab", "ab**(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 884
	t.Run("L884_abab_not_matches_ab___e_f_g", func(t *testing.T) {
		assertMatch(t, false, "abab", "ab**(e|f)g", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 888
	t.Run("L888_abab_not_matches_ab___ef", func(t *testing.T) {
		assertMatch(t, false, "abab", "ab***ef", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 892
	t.Run("L892_abab_not_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, false, "abab", "ab*+(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 896
	t.Run("L896_abab_not_matches_ab_d__e_f_", func(t *testing.T) {
		assertMatch(t, false, "abab", "ab*d+(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 900
	t.Run("L900_abab_not_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, false, "abab", "ab?*(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 904
	t.Run("L904_abb_matches_______", func(t *testing.T) {
		assertMatch(t, true, "abb", "!(*.*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 908
	t.Run("L908_abb_not_matches___a__", func(t *testing.T) {
		assertMatch(t, false, "abb", "!(a)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 912
	t.Run("L912_abb_not_matches_a__b__", func(t *testing.T) {
		assertMatch(t, false, "abb", "a!(b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 916
	t.Run("L916_abbcd_matches___ab_a__b____c_d", func(t *testing.T) {
		assertMatch(t, true, "abbcd", "@(ab|a*(b))*(c)d", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 920
	t.Run("L920_abc_not_matches___a__b__c", func(t *testing.T) {
		assertMatch(t, false, "abc", "\\a\\b\\c", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 924
	t.Run("L924_aBc_matches_a____b_B__", func(t *testing.T) {
		assertMatch(t, true, "aBc", "a!(@(b|B))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 928
	t.Run("L928_abcd_matches____a_b____c_d", func(t *testing.T) {
		assertMatch(t, true, "abcd", "?@(a|b)*@(c)d", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 932
	t.Run("L932_abcd_matches___ab_a___b____c_d", func(t *testing.T) {
		assertMatch(t, true, "abcd", "@(ab|a*@(b))*(c)d", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 936
	t.Run("L936_abcd_abcdefg_abcdefghijk_abcdefghijklmnop_txt_matches_____a_b_g_n_t", func(t *testing.T) {
		assertMatch(t, true, "abcd/abcdefg/abcdefghijk/abcdefghijklmnop.txt", "**/*a*b*g*n*t", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 940
	t.Run("L940_abcd_abcdefg_abcdefghijk_abcdefghijklmnop_txtz_not_matches_____a_b_g_n_t", func(t *testing.T) {
		assertMatch(t, false, "abcd/abcdefg/abcdefghijk/abcdefghijklmnop.txtz", "**/*a*b*g*n*t", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 944
	t.Run("L944_abcdef_matches__a__b__", func(t *testing.T) {
		assertMatch(t, true, "abcdef", "(a+|b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 948
	t.Run("L948_abcdef_not_matches__a__b__", func(t *testing.T) {
		assertMatch(t, false, "abcdef", "(a+|b)+", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 952
	t.Run("L952_abcdef_not_matches____a_bc", func(t *testing.T) {
		assertMatch(t, false, "abcdef", "*?(a)bc", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 956
	t.Run("L956_abcdef_not_matches_a_b__foo_bar__d", func(t *testing.T) {
		assertMatch(t, false, "abcdef", "a(b*(foo|bar))d", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 960
	t.Run("L960_abcdef_not_matches_ab__e_f_", func(t *testing.T) {
		assertMatch(t, false, "abcdef", "ab*(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 964
	t.Run("L964_abcdef_matches_ab__", func(t *testing.T) {
		assertMatch(t, true, "abcdef", "ab**", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 968
	t.Run("L968_abcdef_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, true, "abcdef", "ab**(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 972
	t.Run("L972_abcdef_not_matches_ab___e_f_g", func(t *testing.T) {
		assertMatch(t, false, "abcdef", "ab**(e|f)g", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 976
	t.Run("L976_abcdef_matches_ab___ef", func(t *testing.T) {
		assertMatch(t, true, "abcdef", "ab***ef", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 980
	t.Run("L980_abcdef_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, true, "abcdef", "ab*+(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 984
	t.Run("L984_abcdef_matches_ab_d__e_f_", func(t *testing.T) {
		assertMatch(t, true, "abcdef", "ab*d+(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 988
	t.Run("L988_abcdef_not_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, false, "abcdef", "ab?*(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 992
	t.Run("L992_abcfef_matches__a__b__", func(t *testing.T) {
		assertMatch(t, true, "abcfef", "(a+|b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 996
	t.Run("L996_abcfef_not_matches__a__b__", func(t *testing.T) {
		assertMatch(t, false, "abcfef", "(a+|b)+", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1000
	t.Run("L1000_abcfef_not_matches____a_bc", func(t *testing.T) {
		assertMatch(t, false, "abcfef", "*?(a)bc", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1004
	t.Run("L1004_abcfef_not_matches_a_b__foo_bar__d", func(t *testing.T) {
		assertMatch(t, false, "abcfef", "a(b*(foo|bar))d", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1008
	t.Run("L1008_abcfef_not_matches_ab__e_f_", func(t *testing.T) {
		assertMatch(t, false, "abcfef", "ab*(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1012
	t.Run("L1012_abcfef_matches_ab__", func(t *testing.T) {
		assertMatch(t, true, "abcfef", "ab**", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1016
	t.Run("L1016_abcfef_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, true, "abcfef", "ab**(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1020
	t.Run("L1020_abcfef_not_matches_ab___e_f_g", func(t *testing.T) {
		assertMatch(t, false, "abcfef", "ab**(e|f)g", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1024
	t.Run("L1024_abcfef_matches_ab___ef", func(t *testing.T) {
		assertMatch(t, true, "abcfef", "ab***ef", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1028
	t.Run("L1028_abcfef_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, true, "abcfef", "ab*+(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1032
	t.Run("L1032_abcfef_not_matches_ab_d__e_f_", func(t *testing.T) {
		assertMatch(t, false, "abcfef", "ab*d+(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1036
	t.Run("L1036_abcfef_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, true, "abcfef", "ab?*(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1040
	t.Run("L1040_abcfefg_matches__a__b__", func(t *testing.T) {
		assertMatch(t, true, "abcfefg", "(a+|b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1044
	t.Run("L1044_abcfefg_not_matches__a__b__", func(t *testing.T) {
		assertMatch(t, false, "abcfefg", "(a+|b)+", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1048
	t.Run("L1048_abcfefg_not_matches____a_bc", func(t *testing.T) {
		assertMatch(t, false, "abcfefg", "*?(a)bc", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1052
	t.Run("L1052_abcfefg_not_matches_a_b__foo_bar__d", func(t *testing.T) {
		assertMatch(t, false, "abcfefg", "a(b*(foo|bar))d", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1056
	t.Run("L1056_abcfefg_not_matches_ab__e_f_", func(t *testing.T) {
		assertMatch(t, false, "abcfefg", "ab*(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1060
	t.Run("L1060_abcfefg_matches_ab__", func(t *testing.T) {
		assertMatch(t, true, "abcfefg", "ab**", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1064
	t.Run("L1064_abcfefg_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, true, "abcfefg", "ab**(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1068
	t.Run("L1068_abcfefg_matches_ab___e_f_g", func(t *testing.T) {
		assertMatch(t, true, "abcfefg", "ab**(e|f)g", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1072
	t.Run("L1072_abcfefg_not_matches_ab___ef", func(t *testing.T) {
		assertMatch(t, false, "abcfefg", "ab***ef", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1076
	t.Run("L1076_abcfefg_not_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, false, "abcfefg", "ab*+(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1080
	t.Run("L1080_abcfefg_not_matches_ab_d__e_f_", func(t *testing.T) {
		assertMatch(t, false, "abcfefg", "ab*d+(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1084
	t.Run("L1084_abcfefg_not_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, false, "abcfefg", "ab?*(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1088
	t.Run("L1088_abcx_matches_________", func(t *testing.T) {
		assertMatch(t, true, "abcx", "!([[*])*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1092
	t.Run("L1092_abcx_matches___a_b_____", func(t *testing.T) {
		assertMatch(t, true, "abcx", "+(a|b\\[)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1096
	t.Run("L1096_abcx_not_matches__a____z", func(t *testing.T) {
		assertMatch(t, false, "abcx", "[a*(]*z", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1100
	t.Run("L1100_abcXdefXghi_matches__X_i", func(t *testing.T) {
		assertMatch(t, true, "abcXdefXghi", "*X*i", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1104
	t.Run("L1104_abcz_matches_________", func(t *testing.T) {
		assertMatch(t, true, "abcz", "!([[*])*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1108
	t.Run("L1108_abcz_matches___a_b_____", func(t *testing.T) {
		assertMatch(t, true, "abcz", "+(a|b\\[)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1112
	t.Run("L1112_abcz_matches__a____z", func(t *testing.T) {
		assertMatch(t, true, "abcz", "[a*(]*z", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1116
	t.Run("L1116_abd_matches__a__b__", func(t *testing.T) {
		assertMatch(t, true, "abd", "(a+|b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1120
	t.Run("L1120_abd_not_matches__a__b__", func(t *testing.T) {
		assertMatch(t, false, "abd", "(a+|b)+", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1124
	t.Run("L1124_abd_not_matches____a_bc", func(t *testing.T) {
		assertMatch(t, false, "abd", "*?(a)bc", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1128
	t.Run("L1128_abd_matches_a____b_B__", func(t *testing.T) {
		assertMatch(t, true, "abd", "a!(*(b|B))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1132
	t.Run("L1132_abd_matches_a____b_B__", func(t *testing.T) {
		assertMatch(t, true, "abd", "a!(@(b|B))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1136
	t.Run("L1136_abd_not_matches_a____b_B__d", func(t *testing.T) {
		assertMatch(t, false, "abd", "a!(@(b|B))d", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1140
	t.Run("L1140_abd_matches_a_b__foo_bar__d", func(t *testing.T) {
		assertMatch(t, true, "abd", "a(b*(foo|bar))d", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1144
	t.Run("L1144_abd_matches_a__b_c_d", func(t *testing.T) {
		assertMatch(t, true, "abd", "a+(b|c)d", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1148
	t.Run("L1148_abd_matches_a_b__foo_bar__d", func(t *testing.T) {
		assertMatch(t, true, "abd", "a[b*(foo|bar)]d", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1152
	t.Run("L1152_abd_not_matches_ab__e_f_", func(t *testing.T) {
		assertMatch(t, false, "abd", "ab*(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1156
	t.Run("L1156_abd_matches_ab__", func(t *testing.T) {
		assertMatch(t, true, "abd", "ab**", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1160
	t.Run("L1160_abd_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, true, "abd", "ab**(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1164
	t.Run("L1164_abd_not_matches_ab___e_f_g", func(t *testing.T) {
		assertMatch(t, false, "abd", "ab**(e|f)g", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1168
	t.Run("L1168_abd_not_matches_ab___ef", func(t *testing.T) {
		assertMatch(t, false, "abd", "ab***ef", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1172
	t.Run("L1172_abd_not_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, false, "abd", "ab*+(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1176
	t.Run("L1176_abd_not_matches_ab_d__e_f_", func(t *testing.T) {
		assertMatch(t, false, "abd", "ab*d+(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1180
	t.Run("L1180_abd_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, true, "abd", "ab?*(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1184
	t.Run("L1184_abef_matches__a__b__", func(t *testing.T) {
		assertMatch(t, true, "abef", "(a+|b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1188
	t.Run("L1188_abef_not_matches__a__b__", func(t *testing.T) {
		assertMatch(t, false, "abef", "(a+|b)+", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1192
	t.Run("L1192_abef_not_matches___a__b_", func(t *testing.T) {
		assertMatch(t, false, "abef", "*(a+|b)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1196
	t.Run("L1196_abef_not_matches____a_bc", func(t *testing.T) {
		assertMatch(t, false, "abef", "*?(a)bc", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1200
	t.Run("L1200_abef_not_matches_a_b__foo_bar__d", func(t *testing.T) {
		assertMatch(t, false, "abef", "a(b*(foo|bar))d", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1204
	t.Run("L1204_abef_matches_ab__e_f_", func(t *testing.T) {
		assertMatch(t, true, "abef", "ab*(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1208
	t.Run("L1208_abef_matches_ab__", func(t *testing.T) {
		assertMatch(t, true, "abef", "ab**", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1212
	t.Run("L1212_abef_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, true, "abef", "ab**(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1216
	t.Run("L1216_abef_not_matches_ab___e_f_g", func(t *testing.T) {
		assertMatch(t, false, "abef", "ab**(e|f)g", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1220
	t.Run("L1220_abef_matches_ab___ef", func(t *testing.T) {
		assertMatch(t, true, "abef", "ab***ef", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1224
	t.Run("L1224_abef_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, true, "abef", "ab*+(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1228
	t.Run("L1228_abef_not_matches_ab_d__e_f_", func(t *testing.T) {
		assertMatch(t, false, "abef", "ab*d+(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1232
	t.Run("L1232_abef_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, true, "abef", "ab?*(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1236
	t.Run("L1236_abz_not_matches_a____", func(t *testing.T) {
		assertMatch(t, false, "abz", "a!(*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1240
	t.Run("L1240_abz_matches_a__z_", func(t *testing.T) {
		assertMatch(t, true, "abz", "a!(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1244
	t.Run("L1244_abz_matches_a___z_", func(t *testing.T) {
		assertMatch(t, true, "abz", "a*!(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1248
	t.Run("L1248_abz_not_matches_a__z_", func(t *testing.T) {
		assertMatch(t, false, "abz", "a*(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1252
	t.Run("L1252_abz_matches_a___z_", func(t *testing.T) {
		assertMatch(t, true, "abz", "a**(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1256
	t.Run("L1256_abz_matches_a___z_", func(t *testing.T) {
		assertMatch(t, true, "abz", "a*@(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1260
	t.Run("L1260_abz_not_matches_a__z_", func(t *testing.T) {
		assertMatch(t, false, "abz", "a+(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1264
	t.Run("L1264_abz_not_matches_a__z_", func(t *testing.T) {
		assertMatch(t, false, "abz", "a?(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1268
	t.Run("L1268_abz_not_matches_a__z_", func(t *testing.T) {
		assertMatch(t, false, "abz", "a@(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1272
	t.Run("L1272_ac_not_matches___a__", func(t *testing.T) {
		assertMatch(t, false, "ac", "!(a)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1276
	t.Run("L1276_ac_matches_____a__a__c_", func(t *testing.T) {
		assertMatch(t, true, "ac", "*(@(a))a@(c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1280
	t.Run("L1280_ac_matches_a____b_B__", func(t *testing.T) {
		assertMatch(t, true, "ac", "a!(*(b|B))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1284
	t.Run("L1284_ac_matches_a____b_B__", func(t *testing.T) {
		assertMatch(t, true, "ac", "a!(@(b|B))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1288
	t.Run("L1288_ac_matches_a__b__", func(t *testing.T) {
		assertMatch(t, true, "ac", "a!(b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1292
	t.Run("L1292_accdef_matches__a__b__", func(t *testing.T) {
		assertMatch(t, true, "accdef", "(a+|b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1296
	t.Run("L1296_accdef_not_matches__a__b__", func(t *testing.T) {
		assertMatch(t, false, "accdef", "(a+|b)+", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1300
	t.Run("L1300_accdef_not_matches____a_bc", func(t *testing.T) {
		assertMatch(t, false, "accdef", "*?(a)bc", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1304
	t.Run("L1304_accdef_not_matches_a_b__foo_bar__d", func(t *testing.T) {
		assertMatch(t, false, "accdef", "a(b*(foo|bar))d", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1308
	t.Run("L1308_accdef_not_matches_ab__e_f_", func(t *testing.T) {
		assertMatch(t, false, "accdef", "ab*(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1312
	t.Run("L1312_accdef_not_matches_ab__", func(t *testing.T) {
		assertMatch(t, false, "accdef", "ab**", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1316
	t.Run("L1316_accdef_not_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, false, "accdef", "ab**(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1320
	t.Run("L1320_accdef_not_matches_ab___e_f_g", func(t *testing.T) {
		assertMatch(t, false, "accdef", "ab**(e|f)g", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1324
	t.Run("L1324_accdef_not_matches_ab___ef", func(t *testing.T) {
		assertMatch(t, false, "accdef", "ab***ef", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1328
	t.Run("L1328_accdef_not_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, false, "accdef", "ab*+(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1332
	t.Run("L1332_accdef_not_matches_ab_d__e_f_", func(t *testing.T) {
		assertMatch(t, false, "accdef", "ab*d+(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1336
	t.Run("L1336_accdef_not_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, false, "accdef", "ab?*(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1340
	t.Run("L1340_acd_matches__a__b__", func(t *testing.T) {
		assertMatch(t, true, "acd", "(a+|b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1344
	t.Run("L1344_acd_not_matches__a__b__", func(t *testing.T) {
		assertMatch(t, false, "acd", "(a+|b)+", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1348
	t.Run("L1348_acd_not_matches____a_bc", func(t *testing.T) {
		assertMatch(t, false, "acd", "*?(a)bc", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1352
	t.Run("L1352_acd_matches___ab_a__b____c_d", func(t *testing.T) {
		assertMatch(t, true, "acd", "@(ab|a*(b))*(c)d", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1356
	t.Run("L1356_acd_matches_a____b_B__", func(t *testing.T) {
		assertMatch(t, true, "acd", "a!(*(b|B))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1360
	t.Run("L1360_acd_matches_a____b_B__", func(t *testing.T) {
		assertMatch(t, true, "acd", "a!(@(b|B))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1364
	t.Run("L1364_acd_matches_a____b_B__d", func(t *testing.T) {
		assertMatch(t, true, "acd", "a!(@(b|B))d", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1368
	t.Run("L1368_acd_not_matches_a_b__foo_bar__d", func(t *testing.T) {
		assertMatch(t, false, "acd", "a(b*(foo|bar))d", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1372
	t.Run("L1372_acd_matches_a__b_c_d", func(t *testing.T) {
		assertMatch(t, true, "acd", "a+(b|c)d", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1376
	t.Run("L1376_acd_not_matches_a_b__foo_bar__d", func(t *testing.T) {
		assertMatch(t, false, "acd", "a[b*(foo|bar)]d", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1380
	t.Run("L1380_acd_not_matches_ab__e_f_", func(t *testing.T) {
		assertMatch(t, false, "acd", "ab*(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1384
	t.Run("L1384_acd_not_matches_ab__", func(t *testing.T) {
		assertMatch(t, false, "acd", "ab**", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1388
	t.Run("L1388_acd_not_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, false, "acd", "ab**(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1392
	t.Run("L1392_acd_not_matches_ab___e_f_g", func(t *testing.T) {
		assertMatch(t, false, "acd", "ab**(e|f)g", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1396
	t.Run("L1396_acd_not_matches_ab___ef", func(t *testing.T) {
		assertMatch(t, false, "acd", "ab***ef", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1400
	t.Run("L1400_acd_not_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, false, "acd", "ab*+(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1404
	t.Run("L1404_acd_not_matches_ab_d__e_f_", func(t *testing.T) {
		assertMatch(t, false, "acd", "ab*d+(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1408
	t.Run("L1408_acd_not_matches_ab___e_f_", func(t *testing.T) {
		assertMatch(t, false, "acd", "ab?*(e|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1412
	t.Run("L1412_axz_not_matches_a__z_", func(t *testing.T) {
		assertMatch(t, false, "axz", "a+(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1416
	t.Run("L1416_az_not_matches_a____", func(t *testing.T) {
		assertMatch(t, false, "az", "a!(*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1420
	t.Run("L1420_az_not_matches_a__z_", func(t *testing.T) {
		assertMatch(t, false, "az", "a!(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1424
	t.Run("L1424_az_matches_a___z_", func(t *testing.T) {
		assertMatch(t, true, "az", "a*!(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1428
	t.Run("L1428_az_matches_a__z_", func(t *testing.T) {
		assertMatch(t, true, "az", "a*(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1432
	t.Run("L1432_az_matches_a___z_", func(t *testing.T) {
		assertMatch(t, true, "az", "a**(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1436
	t.Run("L1436_az_matches_a___z_", func(t *testing.T) {
		assertMatch(t, true, "az", "a*@(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1440
	t.Run("L1440_az_matches_a__z_", func(t *testing.T) {
		assertMatch(t, true, "az", "a+(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1444
	t.Run("L1444_az_matches_a__z_", func(t *testing.T) {
		assertMatch(t, true, "az", "a?(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1448
	t.Run("L1448_az_matches_a__z_", func(t *testing.T) {
		assertMatch(t, true, "az", "a@(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1452
	t.Run("L1452_az_not_matches_a____z", func(t *testing.T) {
		assertMatch(t, false, "az", "a\\\\z", &Options{Windows: false})
	})

	// Source: extglobs-minimatch.js line 1456
	t.Run("L1456_az_not_matches_a____z", func(t *testing.T) {
		assertMatch(t, false, "az", "a\\\\z", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1460
	t.Run("L1460_b_matches___a__", func(t *testing.T) {
		assertMatch(t, true, "b", "!(a)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1464
	t.Run("L1464_b_matches__a__b__", func(t *testing.T) {
		assertMatch(t, true, "b", "(a+|b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1468
	t.Run("L1468_b_not_matches_a__b__", func(t *testing.T) {
		assertMatch(t, false, "b", "a!(b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1472
	t.Run("L1472_b_a_matches__b_a___a_", func(t *testing.T) {
		assertMatch(t, true, "b.a", "(b|a).(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1476
	t.Run("L1476_b_a_matches___b_a____a_", func(t *testing.T) {
		assertMatch(t, true, "b.a", "@(b|a).@(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1480
	t.Run("L1480_b_a_not_matches___b_a_", func(t *testing.T) {
		assertMatch(t, false, "b/a", "!(b/a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1484
	t.Run("L1484_b_b_matches___b_a_", func(t *testing.T) {
		assertMatch(t, true, "b/b", "!(b/a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1488
	t.Run("L1488_b_c_matches___b_a_", func(t *testing.T) {
		assertMatch(t, true, "b/c", "!(b/a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1492
	t.Run("L1492_b_c_not_matches_b___c_", func(t *testing.T) {
		assertMatch(t, false, "b/c", "b/!(c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1496
	t.Run("L1496_b_c_matches_b___cc_", func(t *testing.T) {
		assertMatch(t, true, "b/c", "b/!(cc)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1500
	t.Run("L1500_b_c_txt_not_matches_b___c__txt", func(t *testing.T) {
		assertMatch(t, false, "b/c.txt", "b/!(c).txt", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1504
	t.Run("L1504_b_c_txt_matches_b___cc__txt", func(t *testing.T) {
		assertMatch(t, true, "b/c.txt", "b/!(cc).txt", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1508
	t.Run("L1508_b_cc_matches_b___c_", func(t *testing.T) {
		assertMatch(t, true, "b/cc", "b/!(c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1512
	t.Run("L1512_b_cc_not_matches_b___cc_", func(t *testing.T) {
		assertMatch(t, false, "b/cc", "b/!(cc)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1516
	t.Run("L1516_b_cc_txt_not_matches_b___c__txt", func(t *testing.T) {
		assertMatch(t, false, "b/cc.txt", "b/!(c).txt", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1520
	t.Run("L1520_b_cc_txt_not_matches_b___cc__txt", func(t *testing.T) {
		assertMatch(t, false, "b/cc.txt", "b/!(cc).txt", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1524
	t.Run("L1524_b_ccc_matches_b___c_", func(t *testing.T) {
		assertMatch(t, true, "b/ccc", "b/!(c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1528
	t.Run("L1528_ba_matches___a__b__", func(t *testing.T) {
		assertMatch(t, true, "ba", "!(a!(b))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1532
	t.Run("L1532_ba_matches_b__a_b_", func(t *testing.T) {
		assertMatch(t, true, "ba", "b?(a|b)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1536
	t.Run("L1536_baaac_not_matches_____a__a__c_", func(t *testing.T) {
		assertMatch(t, false, "baaac", "*(@(a))a@(c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1540
	t.Run("L1540_bar_matches___foo_", func(t *testing.T) {
		assertMatch(t, true, "bar", "!(foo)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1544
	t.Run("L1544_bar_matches___foo__", func(t *testing.T) {
		assertMatch(t, true, "bar", "!(foo)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1548
	t.Run("L1548_bar_matches___foo_b_", func(t *testing.T) {
		assertMatch(t, true, "bar", "!(foo)b*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1552
	t.Run("L1552_bar_matches_____foo__", func(t *testing.T) {
		assertMatch(t, true, "bar", "*(!(foo))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1556
	t.Run("L1556_baz_matches___foo__", func(t *testing.T) {
		assertMatch(t, true, "baz", "!(foo)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1560
	t.Run("L1560_baz_matches___foo_b_", func(t *testing.T) {
		assertMatch(t, true, "baz", "!(foo)b*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1564
	t.Run("L1564_baz_matches_____foo__", func(t *testing.T) {
		assertMatch(t, true, "baz", "*(!(foo))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1568
	t.Run("L1568_bb_matches___a__b__", func(t *testing.T) {
		assertMatch(t, true, "bb", "!(a!(b))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1572
	t.Run("L1572_bb_matches___a__", func(t *testing.T) {
		assertMatch(t, true, "bb", "!(a)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1576
	t.Run("L1576_bb_not_matches_a__b__", func(t *testing.T) {
		assertMatch(t, false, "bb", "a!(b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1580
	t.Run("L1580_bb_not_matches_a__a_b_", func(t *testing.T) {
		assertMatch(t, false, "bb", "a?(a|b)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1584
	t.Run("L1584_bbc_matches_________", func(t *testing.T) {
		assertMatch(t, true, "bbc", "!([[*])*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1588
	t.Run("L1588_bbc_not_matches___a_b_____", func(t *testing.T) {
		assertMatch(t, false, "bbc", "+(a|b\\[)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1592
	t.Run("L1592_bbc_not_matches__a____z", func(t *testing.T) {
		assertMatch(t, false, "bbc", "[a*(]*z", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1596
	t.Run("L1596_bz_not_matches_a__z_", func(t *testing.T) {
		assertMatch(t, false, "bz", "a+(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1600
	t.Run("L1600_c_not_matches_____a__a__c_", func(t *testing.T) {
		assertMatch(t, false, "c", "*(@(a))a@(c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1604
	t.Run("L1604_c_a_not_matches______a_b___", func(t *testing.T) {
		assertMatch(t, false, "c.a", "!(*.[a-b]*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1608
	t.Run("L1608_c_a_matches_____a_b___a_b___", func(t *testing.T) {
		assertMatch(t, true, "c.a", "!(*[a-b].[a-b]*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1612
	t.Run("L1612_c_a_not_matches_____a_b_", func(t *testing.T) {
		assertMatch(t, false, "c.a", "!*.(a|b)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1616
	t.Run("L1616_c_a_not_matches_____a_b__", func(t *testing.T) {
		assertMatch(t, false, "c.a", "!*.(a|b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1620
	t.Run("L1620_c_a_not_matches__b_a___a_", func(t *testing.T) {
		assertMatch(t, false, "c.a", "(b|a).(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1624
	t.Run("L1624_c_a_not_matches_____a_", func(t *testing.T) {
		assertMatch(t, false, "c.a", "*.!(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1628
	t.Run("L1628_c_a_not_matches_____b_d_", func(t *testing.T) {
		assertMatch(t, false, "c.a", "*.+(b|d)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1632
	t.Run("L1632_c_a_not_matches___b_a____a_", func(t *testing.T) {
		assertMatch(t, false, "c.a", "@(b|a).@(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1636
	t.Run("L1636_c_c_not_matches_____a___b___c_", func(t *testing.T) {
		assertMatch(t, false, "c.c", "!(*.a|*.b|*.c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1640
	t.Run("L1640_c_c_matches_____a__b__c_", func(t *testing.T) {
		assertMatch(t, true, "c.c", "*!(.a|.b|.c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1644
	t.Run("L1644_c_c_not_matches_____a_b_c_", func(t *testing.T) {
		assertMatch(t, false, "c.c", "*.!(a|b|c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1648
	t.Run("L1648_c_c_not_matches____a_b___ab_a___b____c_d_", func(t *testing.T) {
		assertMatch(t, false, "c.c", "*.(a|b|@(ab|a*@(b))*(c)d)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1652
	t.Run("L1652_c_ccc_matches______a_b___", func(t *testing.T) {
		assertMatch(t, true, "c.ccc", "!(*.[a-b]*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1656
	t.Run("L1656_c_ccc_matches_____a_b___a_b___", func(t *testing.T) {
		assertMatch(t, true, "c.ccc", "!(*[a-b].[a-b]*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1660
	t.Run("L1660_c_js_not_matches_____js_", func(t *testing.T) {
		assertMatch(t, false, "c.js", "!(*.js)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1664
	t.Run("L1664_c_js_matches_____js_", func(t *testing.T) {
		assertMatch(t, true, "c.js", "*!(.js)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1668
	t.Run("L1668_c_js_not_matches_____js_", func(t *testing.T) {
		assertMatch(t, false, "c.js", "*.!(js)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1672
	t.Run("L1672_c_a_v_matches_c___z__v", func(t *testing.T) {
		assertMatch(t, true, "c/a/v", "c/!(z)/v", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1676
	t.Run("L1676_c_a_v_not_matches_c___z__v", func(t *testing.T) {
		assertMatch(t, false, "c/a/v", "c/*(z)/v", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1680
	t.Run("L1680_c_a_v_not_matches_c___z__v", func(t *testing.T) {
		assertMatch(t, false, "c/a/v", "c/+(z)/v", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1684
	t.Run("L1684_c_a_v_not_matches_c___z__v", func(t *testing.T) {
		assertMatch(t, false, "c/a/v", "c/@(z)/v", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1688
	t.Run("L1688_c_z_v_not_matches___z_", func(t *testing.T) {
		assertMatch(t, false, "c/z/v", "*(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1692
	t.Run("L1692_c_z_v_not_matches___z_", func(t *testing.T) {
		assertMatch(t, false, "c/z/v", "+(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1696
	t.Run("L1696_c_z_v_not_matches___z_", func(t *testing.T) {
		assertMatch(t, false, "c/z/v", "?(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1700
	t.Run("L1700_c_z_v_not_matches_c___z__v", func(t *testing.T) {
		assertMatch(t, false, "c/z/v", "c/!(z)/v", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1704
	t.Run("L1704_c_z_v_matches_c___z__v", func(t *testing.T) {
		assertMatch(t, true, "c/z/v", "c/*(z)/v", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1708
	t.Run("L1708_c_z_v_matches_c___z__v", func(t *testing.T) {
		assertMatch(t, true, "c/z/v", "c/+(z)/v", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1712
	t.Run("L1712_c_z_v_matches_c___z__v", func(t *testing.T) {
		assertMatch(t, true, "c/z/v", "c/@(z)/v", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1716
	t.Run("L1716_c_z_v_matches_c_z_v", func(t *testing.T) {
		assertMatch(t, true, "c/z/v", "c/z/v", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1720
	t.Run("L1720_cc_a_not_matches__b_a___a_", func(t *testing.T) {
		assertMatch(t, false, "cc.a", "(b|a).(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1724
	t.Run("L1724_cc_a_not_matches___b_a____a_", func(t *testing.T) {
		assertMatch(t, false, "cc.a", "@(b|a).@(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1728
	t.Run("L1728_ccc_matches___a__", func(t *testing.T) {
		assertMatch(t, true, "ccc", "!(a)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1732
	t.Run("L1732_ccc_not_matches_a__b__", func(t *testing.T) {
		assertMatch(t, false, "ccc", "a!(b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1736
	t.Run("L1736_cow_matches_______", func(t *testing.T) {
		assertMatch(t, true, "cow", "!(*.*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1740
	t.Run("L1740_cow_not_matches________", func(t *testing.T) {
		assertMatch(t, false, "cow", "!(*.*).", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1744
	t.Run("L1744_cow_not_matches________", func(t *testing.T) {
		assertMatch(t, false, "cow", ".!(*.*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1748
	t.Run("L1748_cz_not_matches_a____", func(t *testing.T) {
		assertMatch(t, false, "cz", "a!(*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1752
	t.Run("L1752_cz_not_matches_a__z_", func(t *testing.T) {
		assertMatch(t, false, "cz", "a!(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1756
	t.Run("L1756_cz_not_matches_a___z_", func(t *testing.T) {
		assertMatch(t, false, "cz", "a*!(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1760
	t.Run("L1760_cz_not_matches_a__z_", func(t *testing.T) {
		assertMatch(t, false, "cz", "a*(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1764
	t.Run("L1764_cz_not_matches_a___z_", func(t *testing.T) {
		assertMatch(t, false, "cz", "a**(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1768
	t.Run("L1768_cz_not_matches_a___z_", func(t *testing.T) {
		assertMatch(t, false, "cz", "a*@(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1772
	t.Run("L1772_cz_not_matches_a__z_", func(t *testing.T) {
		assertMatch(t, false, "cz", "a+(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1776
	t.Run("L1776_cz_not_matches_a__z_", func(t *testing.T) {
		assertMatch(t, false, "cz", "a?(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1780
	t.Run("L1780_cz_not_matches_a__z_", func(t *testing.T) {
		assertMatch(t, false, "cz", "a@(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1784
	t.Run("L1784_d_a_d_not_matches______a_b___", func(t *testing.T) {
		assertMatch(t, false, "d.a.d", "!(*.[a-b]*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1788
	t.Run("L1788_d_a_d_matches_____a_b___a_b___", func(t *testing.T) {
		assertMatch(t, true, "d.a.d", "!(*[a-b].[a-b]*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1792
	t.Run("L1792_d_a_d_not_matches_____a_b__", func(t *testing.T) {
		assertMatch(t, false, "d.a.d", "!*.(a|b)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1796
	t.Run("L1796_d_a_d_matches______a_b_", func(t *testing.T) {
		assertMatch(t, true, "d.a.d", "!*.*(a|b)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1800
	t.Run("L1800_d_a_d_not_matches_____a_b__", func(t *testing.T) {
		assertMatch(t, false, "d.a.d", "!*.{a,b}*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1804
	t.Run("L1804_d_a_d_matches_____a_", func(t *testing.T) {
		assertMatch(t, true, "d.a.d", "*.!(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1808
	t.Run("L1808_d_a_d_matches_____b_d_", func(t *testing.T) {
		assertMatch(t, true, "d.a.d", "*.+(b|d)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1812
	t.Run("L1812_d_d_matches_____a___b___c_", func(t *testing.T) {
		assertMatch(t, true, "d.d", "!(*.a|*.b|*.c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1816
	t.Run("L1816_d_d_matches_____a__b__c_", func(t *testing.T) {
		assertMatch(t, true, "d.d", "*!(.a|.b|.c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1820
	t.Run("L1820_d_d_matches_____a_b_c_", func(t *testing.T) {
		assertMatch(t, true, "d.d", "*.!(a|b|c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1824
	t.Run("L1824_d_d_not_matches____a_b___ab_a___b____c_d_", func(t *testing.T) {
		assertMatch(t, false, "d.d", "*.(a|b|@(ab|a*@(b))*(c)d)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1828
	t.Run("L1828_d_js_d_matches_____js_", func(t *testing.T) {
		assertMatch(t, true, "d.js.d", "!(*.js)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1832
	t.Run("L1832_d_js_d_matches_____js_", func(t *testing.T) {
		assertMatch(t, true, "d.js.d", "*!(.js)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1836
	t.Run("L1836_d_js_d_matches_____js_", func(t *testing.T) {
		assertMatch(t, true, "d.js.d", "*.!(js)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1840
	t.Run("L1840_dd_aa_d_not_matches__b_a___a_", func(t *testing.T) {
		assertMatch(t, false, "dd.aa.d", "(b|a).(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1844
	t.Run("L1844_dd_aa_d_not_matches___b_a____a_", func(t *testing.T) {
		assertMatch(t, false, "dd.aa.d", "@(b|a).@(a)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1848
	t.Run("L1848_def_not_matches___ef", func(t *testing.T) {
		assertMatch(t, false, "def", "()ef", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1852
	t.Run("L1852_e_e_matches_____a___b___c_", func(t *testing.T) {
		assertMatch(t, true, "e.e", "!(*.a|*.b|*.c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1856
	t.Run("L1856_e_e_matches_____a__b__c_", func(t *testing.T) {
		assertMatch(t, true, "e.e", "*!(.a|.b|.c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1860
	t.Run("L1860_e_e_matches_____a_b_c_", func(t *testing.T) {
		assertMatch(t, true, "e.e", "*.!(a|b|c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1864
	t.Run("L1864_e_e_not_matches____a_b___ab_a___b____c_d_", func(t *testing.T) {
		assertMatch(t, false, "e.e", "*.(a|b|@(ab|a*@(b))*(c)d)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1868
	t.Run("L1868_ef_matches___ef", func(t *testing.T) {
		assertMatch(t, true, "ef", "()ef", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1872
	t.Run("L1872_effgz_matches___b__c_d_e__f_g____h_i__j_k__", func(t *testing.T) {
		assertMatch(t, true, "effgz", "@(b+(c)d|e*(f)g?|?(h)i@(j|k))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1876
	t.Run("L1876_efgz_matches___b__c_d_e__f_g____h_i__j_k__", func(t *testing.T) {
		assertMatch(t, true, "efgz", "@(b+(c)d|e*(f)g?|?(h)i@(j|k))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1880
	t.Run("L1880_egz_matches___b__c_d_e__f_g____h_i__j_k__", func(t *testing.T) {
		assertMatch(t, true, "egz", "@(b+(c)d|e*(f)g?|?(h)i@(j|k))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1884
	t.Run("L1884_egz_not_matches___b__c_d_e__f_g____h_i__j_k__", func(t *testing.T) {
		assertMatch(t, false, "egz", "@(b+(c)d|e+(f)g?|?(h)i@(j|k))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1888
	t.Run("L1888_egzefffgzbcdij_matches___b__c_d_e__f_g____h_i__j_k__", func(t *testing.T) {
		assertMatch(t, true, "egzefffgzbcdij", "*(b+(c)d|e*(f)g?|?(h)i@(j|k))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1892
	t.Run("L1892_f_not_matches___f__o__", func(t *testing.T) {
		assertMatch(t, false, "f", "!(f!(o))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1896
	t.Run("L1896_f_matches___f_o__", func(t *testing.T) {
		assertMatch(t, true, "f", "!(f(o))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1900
	t.Run("L1900_f_not_matches___f_", func(t *testing.T) {
		assertMatch(t, false, "f", "!(f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1904
	t.Run("L1904_f_not_matches_____f__", func(t *testing.T) {
		assertMatch(t, false, "f", "*(!(f))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1908
	t.Run("L1908_f_not_matches_____f__", func(t *testing.T) {
		assertMatch(t, false, "f", "+(!(f))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1912
	t.Run("L1912_f_a_not_matches_____a___b___c_", func(t *testing.T) {
		assertMatch(t, false, "f.a", "!(*.a|*.b|*.c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1916
	t.Run("L1916_f_a_matches_____a__b__c_", func(t *testing.T) {
		assertMatch(t, true, "f.a", "*!(.a|.b|.c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1920
	t.Run("L1920_f_a_not_matches_____a_b_c_", func(t *testing.T) {
		assertMatch(t, false, "f.a", "*.!(a|b|c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1924
	t.Run("L1924_f_f_matches_____a___b___c_", func(t *testing.T) {
		assertMatch(t, true, "f.f", "!(*.a|*.b|*.c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1928
	t.Run("L1928_f_f_matches_____a__b__c_", func(t *testing.T) {
		assertMatch(t, true, "f.f", "*!(.a|.b|.c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1932
	t.Run("L1932_f_f_matches_____a_b_c_", func(t *testing.T) {
		assertMatch(t, true, "f.f", "*.!(a|b|c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1936
	t.Run("L1936_f_f_not_matches____a_b___ab_a___b____c_d_", func(t *testing.T) {
		assertMatch(t, false, "f.f", "*.(a|b|@(ab|a*@(b))*(c)d)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1940
	t.Run("L1940_fa_not_matches___f__o__", func(t *testing.T) {
		assertMatch(t, false, "fa", "!(f!(o))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1944
	t.Run("L1944_fa_matches___f_o__", func(t *testing.T) {
		assertMatch(t, true, "fa", "!(f(o))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1948
	t.Run("L1948_fb_not_matches___f__o__", func(t *testing.T) {
		assertMatch(t, false, "fb", "!(f!(o))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1952
	t.Run("L1952_fb_matches___f_o__", func(t *testing.T) {
		assertMatch(t, true, "fb", "!(f(o))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1956
	t.Run("L1956_fff_matches___f_", func(t *testing.T) {
		assertMatch(t, true, "fff", "!(f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1960
	t.Run("L1960_fff_matches_____f__", func(t *testing.T) {
		assertMatch(t, true, "fff", "*(!(f))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1964
	t.Run("L1964_fff_matches_____f__", func(t *testing.T) {
		assertMatch(t, true, "fff", "+(!(f))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1968
	t.Run("L1968_fffooofoooooffoofffooofff_matches_____f___o__", func(t *testing.T) {
		assertMatch(t, true, "fffooofoooooffoofffooofff", "*(*(f)*(o))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1972
	t.Run("L1972_ffo_matches___f__o__", func(t *testing.T) {
		assertMatch(t, true, "ffo", "*(f*(o))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1976
	t.Run("L1976_file_C_not_matches___c__c_", func(t *testing.T) {
		assertMatch(t, false, "file.C", "*.c?(c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1980
	t.Run("L1980_file_c_matches___c__c_", func(t *testing.T) {
		assertMatch(t, true, "file.c", "*.c?(c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1984
	t.Run("L1984_file_cc_matches___c__c_", func(t *testing.T) {
		assertMatch(t, true, "file.cc", "*.c?(c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1988
	t.Run("L1988_file_ccc_not_matches___c__c_", func(t *testing.T) {
		assertMatch(t, false, "file.ccc", "*.c?(c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1992
	t.Run("L1992_fo_matches___f__o__", func(t *testing.T) {
		assertMatch(t, true, "fo", "!(f!(o))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 1996
	t.Run("L1996_fo_not_matches___f_o__", func(t *testing.T) {
		assertMatch(t, false, "fo", "!(f(o))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2000
	t.Run("L2000_fofo_matches___f__o__", func(t *testing.T) {
		assertMatch(t, true, "fofo", "*(f*(o))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2004
	t.Run("L2004_fofoofoofofoo_matches___fo_foo_", func(t *testing.T) {
		assertMatch(t, true, "fofoofoofofoo", "*(fo|foo)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2008
	t.Run("L2008_fofoofoofofoo_matches___fo_foo_", func(t *testing.T) {
		assertMatch(t, true, "fofoofoofofoo", "*(fo|foo)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2012
	t.Run("L2012_foo_matches_____foo__", func(t *testing.T) {
		assertMatch(t, true, "foo", "!(!(foo))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2016
	t.Run("L2016_foo_matches___f_", func(t *testing.T) {
		assertMatch(t, true, "foo", "!(f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2020
	t.Run("L2020_foo_not_matches___foo_", func(t *testing.T) {
		assertMatch(t, false, "foo", "!(foo)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2024
	t.Run("L2024_foo_not_matches___foo__", func(t *testing.T) {
		assertMatch(t, false, "foo", "!(foo)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2028
	t.Run("L2028_foo_not_matches___foo__", func(t *testing.T) {
		assertMatch(t, false, "foo", "!(foo)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2032
	t.Run("L2032_foo_not_matches___foo__", func(t *testing.T) {
		assertMatch(t, false, "foo", "!(foo)+", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2036
	t.Run("L2036_foo_not_matches___foo_b_", func(t *testing.T) {
		assertMatch(t, false, "foo", "!(foo)b*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2040
	t.Run("L2040_foo_matches___x_", func(t *testing.T) {
		assertMatch(t, true, "foo", "!(x)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2044
	t.Run("L2044_foo_matches___x__", func(t *testing.T) {
		assertMatch(t, true, "foo", "!(x)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2048
	t.Run("L2048_foo_matches__", func(t *testing.T) {
		assertMatch(t, true, "foo", "*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2052
	t.Run("L2052_foo_matches_____f__", func(t *testing.T) {
		assertMatch(t, true, "foo", "*(!(f))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2056
	t.Run("L2056_foo_not_matches_____foo__", func(t *testing.T) {
		assertMatch(t, false, "foo", "*(!(foo))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2060
	t.Run("L2060_foo_not_matches_____a__a__c_", func(t *testing.T) {
		assertMatch(t, false, "foo", "*(@(a))a@(c)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2064
	t.Run("L2064_foo_matches_____foo__", func(t *testing.T) {
		assertMatch(t, true, "foo", "*(@(foo))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2068
	t.Run("L2068_foo_not_matches___a_b____", func(t *testing.T) {
		assertMatch(t, false, "foo", "*(a|b\\[)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2072
	t.Run("L2072_foo_matches___a_b_____f_", func(t *testing.T) {
		assertMatch(t, true, "foo", "*(a|b\\[)|f*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2076
	t.Run("L2076_foo_matches_____a_b_____f__", func(t *testing.T) {
		assertMatch(t, true, "foo", "@(*(a|b\\[)|f*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2080
	t.Run("L2080_foo_not_matches______", func(t *testing.T) {
		assertMatch(t, false, "foo", "*/*/*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2084
	t.Run("L2084_foo_not_matches__f", func(t *testing.T) {
		assertMatch(t, false, "foo", "*f", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2088
	t.Run("L2088_foo_matches__foo_", func(t *testing.T) {
		assertMatch(t, true, "foo", "*foo*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2092
	t.Run("L2092_foo_matches_____f__", func(t *testing.T) {
		assertMatch(t, true, "foo", "+(!(f))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2096
	t.Run("L2096_foo_not_matches___", func(t *testing.T) {
		assertMatch(t, false, "foo", "??", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2100
	t.Run("L2100_foo_matches____", func(t *testing.T) {
		assertMatch(t, true, "foo", "???", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2104
	t.Run("L2104_foo_not_matches_bar", func(t *testing.T) {
		assertMatch(t, false, "foo", "bar", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2108
	t.Run("L2108_foo_matches_f_", func(t *testing.T) {
		assertMatch(t, true, "foo", "f*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2112
	t.Run("L2112_foo_not_matches_fo", func(t *testing.T) {
		assertMatch(t, false, "foo", "fo", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2116
	t.Run("L2116_foo_matches_foo", func(t *testing.T) {
		assertMatch(t, true, "foo", "foo", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2120
	t.Run("L2120_foo_matches____a_b_____f__", func(t *testing.T) {
		assertMatch(t, true, "foo", "{*(a|b\\[),f*}", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2124
	t.Run("L2124_foo__matches_foo___", func(t *testing.T) {
		assertMatch(t, true, "foo*", "foo\\*", &Options{Windows: false})
	})

	// Source: extglobs-minimatch.js line 2128
	t.Run("L2128_foo_bar_matches_foo___bar", func(t *testing.T) {
		assertMatch(t, true, "foo*bar", "foo\\*bar", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2132
	t.Run("L2132_foo_js_not_matches___foo__js", func(t *testing.T) {
		assertMatch(t, false, "foo.js", "!(foo).js", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2136
	t.Run("L2136_foo_js_js_matches_____js_", func(t *testing.T) {
		assertMatch(t, true, "foo.js.js", "*.!(js)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2140
	t.Run("L2140_foo_js_js_not_matches_____js__", func(t *testing.T) {
		assertMatch(t, false, "foo.js.js", "*.!(js)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2144
	t.Run("L2144_foo_js_js_not_matches_____js_____js_", func(t *testing.T) {
		assertMatch(t, false, "foo.js.js", "*.!(js)*.!(js)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2148
	t.Run("L2148_foo_js_js_not_matches_____js__", func(t *testing.T) {
		assertMatch(t, false, "foo.js.js", "*.!(js)+", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2152
	t.Run("L2152_foo_txt_matches______bar__txt", func(t *testing.T) {
		assertMatch(t, true, "foo.txt", "**/!(bar).txt", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2156
	t.Run("L2156_foo_bar_not_matches______", func(t *testing.T) {
		assertMatch(t, false, "foo/bar", "*/*/*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2160
	t.Run("L2160_foo_bar_matches_foo___foo_", func(t *testing.T) {
		assertMatch(t, true, "foo/bar", "foo/!(foo)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2164
	t.Run("L2164_foo_bar_matches_foo__", func(t *testing.T) {
		assertMatch(t, true, "foo/bar", "foo/*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2168
	t.Run("L2168_foo_bar_matches_foo_bar", func(t *testing.T) {
		assertMatch(t, true, "foo/bar", "foo/bar", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2172
	t.Run("L2172_foo_bar_not_matches_foo_bar", func(t *testing.T) {
		assertMatch(t, false, "foo/bar", "foo?bar", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2176
	t.Run("L2176_foo_bar_matches_foo___bar", func(t *testing.T) {
		assertMatch(t, true, "foo/bar", "foo[/]bar", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2180
	t.Run("L2180_foo_bar_baz_jsx_matches_foo_bar________js_jsx_", func(t *testing.T) {
		assertMatch(t, true, "foo/bar/baz.jsx", "foo/bar/**/*.+(js|jsx)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2184
	t.Run("L2184_foo_bar_baz_jsx_matches_foo_bar_____js_jsx_", func(t *testing.T) {
		assertMatch(t, true, "foo/bar/baz.jsx", "foo/bar/*.+(js|jsx)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2188
	t.Run("L2188_foo_bb_aa_rr_matches_________", func(t *testing.T) {
		assertMatch(t, true, "foo/bb/aa/rr", "**/**/**", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2192
	t.Run("L2192_foo_bb_aa_rr_not_matches______", func(t *testing.T) {
		assertMatch(t, false, "foo/bb/aa/rr", "*/*/*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2196
	t.Run("L2196_foo_bba_arr_matches______", func(t *testing.T) {
		assertMatch(t, true, "foo/bba/arr", "*/*/*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2200
	t.Run("L2200_foo_bba_arr_not_matches_foo_", func(t *testing.T) {
		assertMatch(t, false, "foo/bba/arr", "foo*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2204
	t.Run("L2204_foo_bba_arr_not_matches_foo__", func(t *testing.T) {
		assertMatch(t, false, "foo/bba/arr", "foo**", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2208
	t.Run("L2208_foo_bba_arr_not_matches_foo__", func(t *testing.T) {
		assertMatch(t, false, "foo/bba/arr", "foo/*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2212
	t.Run("L2212_foo_bba_arr_matches_foo___", func(t *testing.T) {
		assertMatch(t, true, "foo/bba/arr", "foo/**", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2216
	t.Run("L2216_foo_bba_arr_not_matches_foo___arr", func(t *testing.T) {
		assertMatch(t, false, "foo/bba/arr", "foo/**arr", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2220
	t.Run("L2220_foo_bba_arr_not_matches_foo___z", func(t *testing.T) {
		assertMatch(t, false, "foo/bba/arr", "foo/**z", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2224
	t.Run("L2224_foo_bba_arr_not_matches_foo__arr", func(t *testing.T) {
		assertMatch(t, false, "foo/bba/arr", "foo/*arr", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2228
	t.Run("L2228_foo_bba_arr_not_matches_foo__z", func(t *testing.T) {
		assertMatch(t, false, "foo/bba/arr", "foo/*z", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2232
	t.Run("L2232_foob_not_matches___foo_b_", func(t *testing.T) {
		assertMatch(t, false, "foob", "!(foo)b*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2236
	t.Run("L2236_foob_not_matches__foo_bb", func(t *testing.T) {
		assertMatch(t, false, "foob", "(foo)bb", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2240
	t.Run("L2240_foobar_matches___foo_", func(t *testing.T) {
		assertMatch(t, true, "foobar", "!(foo)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2244
	t.Run("L2244_foobar_not_matches___foo__", func(t *testing.T) {
		assertMatch(t, false, "foobar", "!(foo)*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2248
	t.Run("L2248_foobar_not_matches___foo_b_", func(t *testing.T) {
		assertMatch(t, false, "foobar", "!(foo)b*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2252
	t.Run("L2252_foobar_matches_____foo__", func(t *testing.T) {
		assertMatch(t, true, "foobar", "*(!(foo))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2256
	t.Run("L2256_foobar_matches__ob_a_r_", func(t *testing.T) {
		assertMatch(t, true, "foobar", "*ob*a*r*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2260
	t.Run("L2260_foobar_not_matches_foo___bar", func(t *testing.T) {
		assertMatch(t, false, "foobar", "foo\\*bar", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2264
	t.Run("L2264_foobb_not_matches___foo_b_", func(t *testing.T) {
		assertMatch(t, false, "foobb", "!(foo)b*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2268
	t.Run("L2268_foobb_matches__foo_bb", func(t *testing.T) {
		assertMatch(t, true, "foobb", "(foo)bb", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2272
	t.Run("L2272__foo_bb_matches____foo___bb", func(t *testing.T) {
		assertMatch(t, true, "(foo)bb", "\\(foo\\)bb", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2276
	t.Run("L2276_foofoofo_matches___foo_f_fo___f_of__o__", func(t *testing.T) {
		assertMatch(t, true, "foofoofo", "@(foo|f|fo)*(f|of+(o))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2280
	t.Run("L2280_foofoofo_matches___foo_f_fo___f_of__o__", func(t *testing.T) {
		assertMatch(t, true, "foofoofo", "@(foo|f|fo)*(f|of+(o))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2284
	t.Run("L2284_fooofoofofooo_matches___f__o__", func(t *testing.T) {
		assertMatch(t, true, "fooofoofofooo", "*(f*(o))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2288
	t.Run("L2288_foooofo_matches___f__o__", func(t *testing.T) {
		assertMatch(t, true, "foooofo", "*(f*(o))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2292
	t.Run("L2292_foooofof_matches___f__o__", func(t *testing.T) {
		assertMatch(t, true, "foooofof", "*(f*(o))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2296
	t.Run("L2296_foooofof_not_matches___f__o__", func(t *testing.T) {
		assertMatch(t, false, "foooofof", "*(f+(o))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2300
	t.Run("L2300_foooofofx_not_matches___f__o__", func(t *testing.T) {
		assertMatch(t, false, "foooofofx", "*(f*(o))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2304
	t.Run("L2304_foooxfooxfoxfooox_matches___f__o_x_", func(t *testing.T) {
		assertMatch(t, true, "foooxfooxfoxfooox", "*(f*(o)x)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2308
	t.Run("L2308_foooxfooxfxfooox_matches___f__o_x_", func(t *testing.T) {
		assertMatch(t, true, "foooxfooxfxfooox", "*(f*(o)x)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2312
	t.Run("L2312_foooxfooxofoxfooox_not_matches___f__o_x_", func(t *testing.T) {
		assertMatch(t, false, "foooxfooxofoxfooox", "*(f*(o)x)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2316
	t.Run("L2316_foot_matches_____z____x_", func(t *testing.T) {
		assertMatch(t, true, "foot", "@(!(z*)|*x)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2320
	t.Run("L2320_foox_matches_____z____x_", func(t *testing.T) {
		assertMatch(t, true, "foox", "@(!(z*)|*x)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2324
	t.Run("L2324_fz_not_matches___z_", func(t *testing.T) {
		assertMatch(t, false, "fz", "*(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2328
	t.Run("L2328_fz_not_matches___z_", func(t *testing.T) {
		assertMatch(t, false, "fz", "+(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2332
	t.Run("L2332_fz_not_matches___z_", func(t *testing.T) {
		assertMatch(t, false, "fz", "?(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2336
	t.Run("L2336_moo_cow_not_matches___moo____cow_", func(t *testing.T) {
		assertMatch(t, false, "moo.cow", "!(moo).!(cow)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2340
	t.Run("L2340_moo_cow_not_matches__________", func(t *testing.T) {
		assertMatch(t, false, "moo.cow", "!(*).!(*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2344
	t.Run("L2344_mad_moo_cow_not_matches______________", func(t *testing.T) {
		assertMatch(t, false, "mad.moo.cow", "!(*.*).!(*.*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2348
	t.Run("L2348_mad_moo_cow_not_matches________", func(t *testing.T) {
		assertMatch(t, false, "mad.moo.cow", ".!(*.*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2352
	t.Run("L2352_Makefile_matches_____c___h_Makefile_in_config__README_", func(t *testing.T) {
		assertMatch(t, true, "Makefile", "!(*.c|*.h|Makefile.in|config*|README)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2356
	t.Run("L2356_Makefile_in_not_matches_____c___h_Makefile_in_config__README_", func(t *testing.T) {
		assertMatch(t, false, "Makefile.in", "!(*.c|*.h|Makefile.in|config*|README)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2360
	t.Run("L2360_moo_matches_______", func(t *testing.T) {
		assertMatch(t, true, "moo", "!(*.*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2364
	t.Run("L2364_moo_not_matches________", func(t *testing.T) {
		assertMatch(t, false, "moo", "!(*.*).", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2368
	t.Run("L2368_moo_not_matches________", func(t *testing.T) {
		assertMatch(t, false, "moo", ".!(*.*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2372
	t.Run("L2372_moo_cow_not_matches_______", func(t *testing.T) {
		assertMatch(t, false, "moo.cow", "!(*.*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2376
	t.Run("L2376_moo_cow_not_matches________", func(t *testing.T) {
		assertMatch(t, false, "moo.cow", "!(*.*).", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2380
	t.Run("L2380_moo_cow_not_matches________", func(t *testing.T) {
		assertMatch(t, false, "moo.cow", ".!(*.*)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2384
	t.Run("L2384_mucca_pazza_not_matches_mu____c____pa____z___", func(t *testing.T) {
		assertMatch(t, false, "mucca.pazza", "mu!(*(c))?.pa!(*(z))?", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2388
	t.Run("L2388_ofoofo_matches___of__o__", func(t *testing.T) {
		assertMatch(t, true, "ofoofo", "*(of+(o))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2392
	t.Run("L2392_ofoofo_matches___of__o__f_", func(t *testing.T) {
		assertMatch(t, true, "ofoofo", "*(of+(o)|f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2396
	t.Run("L2396_ofooofoofofooo_not_matches___f__o__", func(t *testing.T) {
		assertMatch(t, false, "ofooofoofofooo", "*(f*(o))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2400
	t.Run("L2400_ofoooxoofxo_matches_____of__o_x_o_", func(t *testing.T) {
		assertMatch(t, true, "ofoooxoofxo", "*(*(of*(o)x)o)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2404
	t.Run("L2404_ofoooxoofxoofoooxoofxo_matches_____of__o_x_o_", func(t *testing.T) {
		assertMatch(t, true, "ofoooxoofxoofoooxoofxo", "*(*(of*(o)x)o)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2408
	t.Run("L2408_ofoooxoofxoofoooxoofxofo_not_matches_____of__o_x_o_", func(t *testing.T) {
		assertMatch(t, false, "ofoooxoofxoofoooxoofxofo", "*(*(of*(o)x)o)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2412
	t.Run("L2412_ofoooxoofxoofoooxoofxoo_matches_____of__o_x_o_", func(t *testing.T) {
		assertMatch(t, true, "ofoooxoofxoofoooxoofxoo", "*(*(of*(o)x)o)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2416
	t.Run("L2416_ofoooxoofxoofoooxoofxooofxofxo_matches_____of__o_x_o_", func(t *testing.T) {
		assertMatch(t, true, "ofoooxoofxoofoooxoofxooofxofxo", "*(*(of*(o)x)o)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2420
	t.Run("L2420_ofxoofxo_matches_____of__o_x_o_", func(t *testing.T) {
		assertMatch(t, true, "ofxoofxo", "*(*(of*(o)x)o)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2424
	t.Run("L2424_oofooofo_matches___of_oof__o__", func(t *testing.T) {
		assertMatch(t, true, "oofooofo", "*(of|oof+(o))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2428
	t.Run("L2428_ooo_matches___f_", func(t *testing.T) {
		assertMatch(t, true, "ooo", "!(f)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2432
	t.Run("L2432_ooo_matches_____f__", func(t *testing.T) {
		assertMatch(t, true, "ooo", "*(!(f))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2436
	t.Run("L2436_ooo_matches_____f__", func(t *testing.T) {
		assertMatch(t, true, "ooo", "+(!(f))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2440
	t.Run("L2440_oxfoxfox_not_matches___oxf__ox__", func(t *testing.T) {
		assertMatch(t, false, "oxfoxfox", "*(oxf+(ox))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2444
	t.Run("L2444_oxfoxoxfox_matches___oxf__ox__", func(t *testing.T) {
		assertMatch(t, true, "oxfoxoxfox", "*(oxf+(ox))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2448
	t.Run("L2448_para_matches_para___0_9__", func(t *testing.T) {
		assertMatch(t, true, "para", "para*([0-9])", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2452
	t.Run("L2452_para_not_matches_para___0_9__", func(t *testing.T) {
		assertMatch(t, false, "para", "para+([0-9])", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2456
	t.Run("L2456_para_38_matches_para_____00_09__", func(t *testing.T) {
		assertMatch(t, true, "para.38", "para!(*.[00-09])", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2460
	t.Run("L2460_para_graph_matches_para_____0_9__", func(t *testing.T) {
		assertMatch(t, true, "para.graph", "para!(*.[0-9])", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2464
	t.Run("L2464_para13829383746592_matches_para___0_9__", func(t *testing.T) {
		assertMatch(t, true, "para13829383746592", "para*([0-9])", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2468
	t.Run("L2468_para381_not_matches_para___345__99_1", func(t *testing.T) {
		assertMatch(t, false, "para381", "para?([345]|99)1", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2472
	t.Run("L2472_para39_matches_para_____0_9__", func(t *testing.T) {
		assertMatch(t, true, "para39", "para!(*.[0-9])", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2476
	t.Run("L2476_para987346523_matches_para___0_9__", func(t *testing.T) {
		assertMatch(t, true, "para987346523", "para+([0-9])", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2480
	t.Run("L2480_para991_matches_para___345__99_1", func(t *testing.T) {
		assertMatch(t, true, "para991", "para?([345]|99)1", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2484
	t.Run("L2484_paragraph_matches_para_____0_9__", func(t *testing.T) {
		assertMatch(t, true, "paragraph", "para!(*.[0-9])", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2488
	t.Run("L2488_paragraph_not_matches_para___0_9__", func(t *testing.T) {
		assertMatch(t, false, "paragraph", "para*([0-9])", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2492
	t.Run("L2492_paragraph_matches_para__chute_graph_", func(t *testing.T) {
		assertMatch(t, true, "paragraph", "para@(chute|graph)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2496
	t.Run("L2496_paramour_not_matches_para__chute_graph_", func(t *testing.T) {
		assertMatch(t, false, "paramour", "para@(chute|graph)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2500
	t.Run("L2500_parse_y_matches_____c___h_Makefile_in_config__README_", func(t *testing.T) {
		assertMatch(t, true, "parse.y", "!(*.c|*.h|Makefile.in|config*|README)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2504
	t.Run("L2504_shell_c_not_matches_____c___h_Makefile_in_config__README_", func(t *testing.T) {
		assertMatch(t, false, "shell.c", "!(*.c|*.h|Makefile.in|config*|README)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2508
	t.Run("L2508_VMS_FILE__not_matches______1_9____0_9__", func(t *testing.T) {
		assertMatch(t, false, "VMS.FILE;", "*\\;[1-9]*([0-9])", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2512
	t.Run("L2512_VMS_FILE_0_not_matches______1_9____0_9__", func(t *testing.T) {
		assertMatch(t, false, "VMS.FILE;0", "*\\;[1-9]*([0-9])", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2516
	t.Run("L2516_VMS_FILE_1_matches______1_9____0_9__", func(t *testing.T) {
		assertMatch(t, true, "VMS.FILE;1", "*\\;[1-9]*([0-9])", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2520
	t.Run("L2520_VMS_FILE_1_matches____1_9____0_9__", func(t *testing.T) {
		assertMatch(t, true, "VMS.FILE;1", "*;[1-9]*([0-9])", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2524
	t.Run("L2524_VMS_FILE_139_matches______1_9____0_9__", func(t *testing.T) {
		assertMatch(t, true, "VMS.FILE;139", "*\\;[1-9]*([0-9])", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2528
	t.Run("L2528_VMS_FILE_1N_not_matches______1_9____0_9__", func(t *testing.T) {
		assertMatch(t, false, "VMS.FILE;1N", "*\\;[1-9]*([0-9])", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2532
	t.Run("L2532_xfoooofof_not_matches___f__o__", func(t *testing.T) {
		assertMatch(t, false, "xfoooofof", "*(f*(o))", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2536
	t.Run("L2536_XXX_adobe_courier_bold_o_normal__12_120_75_75_m_70_iso8859_1_matches_XXX________", func(t *testing.T) {
		assertMatch(t, true, "XXX/adobe/courier/bold/o/normal//12/120/75/75/m/70/iso8859/1", "XXX/*/*/*/*/*/*/12/*/*/*/m/*/*/*", &Options{Windows: false})
	})

	// Source: extglobs-minimatch.js line 2540
	t.Run("L2540_XXX_adobe_courier_bold_o_normal__12_120_75_75_X_70_iso8859_1_not_matches_XXX____", func(t *testing.T) {
		assertMatch(t, false, "XXX/adobe/courier/bold/o/normal//12/120/75/75/X/70/iso8859/1", "XXX/*/*/*/*/*/*/12/*/*/*/m/*/*/*", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2544
	t.Run("L2544_z_matches___z_", func(t *testing.T) {
		assertMatch(t, true, "z", "*(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2548
	t.Run("L2548_z_matches___z_", func(t *testing.T) {
		assertMatch(t, true, "z", "+(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2552
	t.Run("L2552_z_matches___z_", func(t *testing.T) {
		assertMatch(t, true, "z", "?(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2556
	t.Run("L2556_zf_not_matches___z_", func(t *testing.T) {
		assertMatch(t, false, "zf", "*(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2560
	t.Run("L2560_zf_not_matches___z_", func(t *testing.T) {
		assertMatch(t, false, "zf", "+(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2564
	t.Run("L2564_zf_not_matches___z_", func(t *testing.T) {
		assertMatch(t, false, "zf", "?(z)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2568
	t.Run("L2568_zoot_not_matches_____z____x_", func(t *testing.T) {
		assertMatch(t, false, "zoot", "@(!(z*)|*x)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2572
	t.Run("L2572_zoox_matches_____z____x_", func(t *testing.T) {
		assertMatch(t, true, "zoox", "@(!(z*)|*x)", &Options{Windows: true})
	})

	// Source: extglobs-minimatch.js line 2576
	t.Run("L2576_zz_not_matches__a__b__", func(t *testing.T) {
		assertMatch(t, false, "zz", "(a+|b)*", &Options{Windows: true})
	})

}
