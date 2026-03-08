package picomatch

// extglobs_temp_part2_test.go — Ported from picomatch/test/extglobs-temp.js lines 601-1215
// This file contains the second half of the extglobs-temp test suite.

import (
	"testing"
)

func TestExtglobsTempPart2(t *testing.T) {
	win := &Options{Windows: true}
	posix := &Options{Posix: true}

	t.Run("pattern *(of+(o)|f) windows continued", func(t *testing.T) {
		// extglobs-temp.js:601
		assertMatch(t, true, "ofooofoofofooo", "*(of+(o)|f)", win)
		// extglobs-temp.js:602
		assertMatch(t, false, "ofoooxoofxo", "*(of+(o)|f)", win)
		// extglobs-temp.js:603
		assertMatch(t, false, "ofoooxoofxoofoooxoofxo", "*(of+(o)|f)", win)
		// extglobs-temp.js:604
		assertMatch(t, false, "ofoooxoofxoofoooxoofxofo", "*(of+(o)|f)", win)
		// extglobs-temp.js:605
		assertMatch(t, false, "ofoooxoofxoofoooxoofxoo", "*(of+(o)|f)", win)
		// extglobs-temp.js:606
		assertMatch(t, false, "ofoooxoofxoofoooxoofxooofxofxo", "*(of+(o)|f)", win)
		// extglobs-temp.js:607
		assertMatch(t, false, "ofxoofxo", "*(of+(o)|f)", win)
		// extglobs-temp.js:608
		assertMatch(t, false, "oofooofo", "*(of+(o)|f)", win)
		// extglobs-temp.js:609
		assertMatch(t, false, "ooo", "*(of+(o)|f)", win)
		// extglobs-temp.js:610
		assertMatch(t, false, "oxfoxfox", "*(of+(o)|f)", win)
		// extglobs-temp.js:611
		assertMatch(t, false, "oxfoxoxfox", "*(of+(o)|f)", win)
		// extglobs-temp.js:612
		assertMatch(t, false, "xfoooofof", "*(of+(o)|f)", win)
	})

	t.Run("pattern *(of|oof+(o)) windows", func(t *testing.T) {
		// extglobs-temp.js:614
		assertMatch(t, false, "ffffffo", "*(of|oof+(o))", win)
		// extglobs-temp.js:615
		assertMatch(t, false, "fffooofoooooffoofffooofff", "*(of|oof+(o))", win)
		// extglobs-temp.js:616
		assertMatch(t, false, "ffo", "*(of|oof+(o))", win)
		// extglobs-temp.js:617
		assertMatch(t, false, "fofo", "*(of|oof+(o))", win)
		// extglobs-temp.js:618
		assertMatch(t, false, "fofoofoofofoo", "*(of|oof+(o))", win)
		// extglobs-temp.js:619
		assertMatch(t, false, "foo", "*(of|oof+(o))", win)
		// extglobs-temp.js:620
		assertMatch(t, false, "foob", "*(of|oof+(o))", win)
		// extglobs-temp.js:621
		assertMatch(t, false, "foobb", "*(of|oof+(o))", win)
		// extglobs-temp.js:622
		assertMatch(t, false, "foofoofo", "*(of|oof+(o))", win)
		// extglobs-temp.js:623
		assertMatch(t, false, "fooofoofofooo", "*(of|oof+(o))", win)
		// extglobs-temp.js:624
		assertMatch(t, false, "foooofo", "*(of|oof+(o))", win)
		// extglobs-temp.js:625
		assertMatch(t, false, "foooofof", "*(of|oof+(o))", win)
		// extglobs-temp.js:626
		assertMatch(t, false, "foooofofx", "*(of|oof+(o))", win)
		// extglobs-temp.js:627
		assertMatch(t, false, "foooxfooxfoxfooox", "*(of|oof+(o))", win)
		// extglobs-temp.js:628
		assertMatch(t, false, "foooxfooxfxfooox", "*(of|oof+(o))", win)
		// extglobs-temp.js:629
		assertMatch(t, false, "foooxfooxofoxfooox", "*(of|oof+(o))", win)
		// extglobs-temp.js:630
		assertMatch(t, false, "foot", "*(of|oof+(o))", win)
		// extglobs-temp.js:631
		assertMatch(t, false, "foox", "*(of|oof+(o))", win)
		// extglobs-temp.js:632
		assertMatch(t, true, "ofoofo", "*(of|oof+(o))", win)
		// extglobs-temp.js:633
		assertMatch(t, false, "ofooofoofofooo", "*(of|oof+(o))", win)
		// extglobs-temp.js:634
		assertMatch(t, false, "ofoooxoofxo", "*(of|oof+(o))", win)
		// extglobs-temp.js:635
		assertMatch(t, false, "ofoooxoofxoofoooxoofxo", "*(of|oof+(o))", win)
		// extglobs-temp.js:636
		assertMatch(t, false, "ofoooxoofxoofoooxoofxofo", "*(of|oof+(o))", win)
		// extglobs-temp.js:637
		assertMatch(t, false, "ofoooxoofxoofoooxoofxoo", "*(of|oof+(o))", win)
		// extglobs-temp.js:638
		assertMatch(t, false, "ofoooxoofxoofoooxoofxooofxofxo", "*(of|oof+(o))", win)
		// extglobs-temp.js:639
		assertMatch(t, false, "ofxoofxo", "*(of|oof+(o))", win)
		// extglobs-temp.js:640
		assertMatch(t, true, "oofooofo", "*(of|oof+(o))", win)
		// extglobs-temp.js:641
		assertMatch(t, false, "ooo", "*(of|oof+(o))", win)
		// extglobs-temp.js:642
		assertMatch(t, false, "oxfoxfox", "*(of|oof+(o))", win)
		// extglobs-temp.js:643
		assertMatch(t, false, "oxfoxoxfox", "*(of|oof+(o))", win)
		// extglobs-temp.js:644
		assertMatch(t, false, "xfoooofof", "*(of|oof+(o))", win)
	})

	t.Run("pattern *(oxf+(ox)) windows", func(t *testing.T) {
		// extglobs-temp.js:646
		assertMatch(t, false, "ffffffo", "*(oxf+(ox))", win)
		// extglobs-temp.js:647
		assertMatch(t, false, "fffooofoooooffoofffooofff", "*(oxf+(ox))", win)
		// extglobs-temp.js:648
		assertMatch(t, false, "ffo", "*(oxf+(ox))", win)
		// extglobs-temp.js:649
		assertMatch(t, false, "fofo", "*(oxf+(ox))", win)
		// extglobs-temp.js:650
		assertMatch(t, false, "fofoofoofofoo", "*(oxf+(ox))", win)
		// extglobs-temp.js:651
		assertMatch(t, false, "foo", "*(oxf+(ox))", win)
		// extglobs-temp.js:652
		assertMatch(t, false, "foob", "*(oxf+(ox))", win)
		// extglobs-temp.js:653
		assertMatch(t, false, "foobb", "*(oxf+(ox))", win)
		// extglobs-temp.js:654
		assertMatch(t, false, "foofoofo", "*(oxf+(ox))", win)
		// extglobs-temp.js:655
		assertMatch(t, false, "fooofoofofooo", "*(oxf+(ox))", win)
		// extglobs-temp.js:656
		assertMatch(t, false, "foooofo", "*(oxf+(ox))", win)
		// extglobs-temp.js:657
		assertMatch(t, false, "foooofof", "*(oxf+(ox))", win)
		// extglobs-temp.js:658
		assertMatch(t, false, "foooofofx", "*(oxf+(ox))", win)
		// extglobs-temp.js:659
		assertMatch(t, false, "foooxfooxfoxfooox", "*(oxf+(ox))", win)
		// extglobs-temp.js:660
		assertMatch(t, false, "foooxfooxfxfooox", "*(oxf+(ox))", win)
		// extglobs-temp.js:661
		assertMatch(t, false, "foooxfooxofoxfooox", "*(oxf+(ox))", win)
		// extglobs-temp.js:662
		assertMatch(t, false, "foot", "*(oxf+(ox))", win)
		// extglobs-temp.js:663
		assertMatch(t, false, "foox", "*(oxf+(ox))", win)
		// extglobs-temp.js:664
		assertMatch(t, false, "ofoofo", "*(oxf+(ox))", win)
		// extglobs-temp.js:665
		assertMatch(t, false, "ofooofoofofooo", "*(oxf+(ox))", win)
		// extglobs-temp.js:666
		assertMatch(t, false, "ofoooxoofxo", "*(oxf+(ox))", win)
		// extglobs-temp.js:667
		assertMatch(t, false, "ofoooxoofxoofoooxoofxo", "*(oxf+(ox))", win)
		// extglobs-temp.js:668
		assertMatch(t, false, "ofoooxoofxoofoooxoofxofo", "*(oxf+(ox))", win)
		// extglobs-temp.js:669
		assertMatch(t, false, "ofoooxoofxoofoooxoofxoo", "*(oxf+(ox))", win)
		// extglobs-temp.js:670
		assertMatch(t, false, "ofoooxoofxoofoooxoofxooofxofxo", "*(oxf+(ox))", win)
		// extglobs-temp.js:671
		assertMatch(t, false, "ofxoofxo", "*(oxf+(ox))", win)
		// extglobs-temp.js:672
		assertMatch(t, false, "oofooofo", "*(oxf+(ox))", win)
		// extglobs-temp.js:673
		assertMatch(t, false, "ooo", "*(oxf+(ox))", win)
		// extglobs-temp.js:674
		assertMatch(t, false, "oxfoxfox", "*(oxf+(ox))", win)
		// extglobs-temp.js:675
		assertMatch(t, true, "oxfoxoxfox", "*(oxf+(ox))", win)
		// extglobs-temp.js:676
		assertMatch(t, false, "xfoooofof", "*(oxf+(ox))", win)
	})

	t.Run("pattern @(!(z*)|*x) windows", func(t *testing.T) {
		// extglobs-temp.js:678
		assertMatch(t, true, "ffffffo", "@(!(z*)|*x)", win)
		// extglobs-temp.js:679
		assertMatch(t, true, "fffooofoooooffoofffooofff", "@(!(z*)|*x)", win)
		// extglobs-temp.js:680
		assertMatch(t, true, "ffo", "@(!(z*)|*x)", win)
		// extglobs-temp.js:681
		assertMatch(t, true, "fofo", "@(!(z*)|*x)", win)
		// extglobs-temp.js:682
		assertMatch(t, true, "fofoofoofofoo", "@(!(z*)|*x)", win)
		// extglobs-temp.js:683
		assertMatch(t, true, "foo", "@(!(z*)|*x)", win)
		// extglobs-temp.js:684
		assertMatch(t, true, "foob", "@(!(z*)|*x)", win)
		// extglobs-temp.js:685
		assertMatch(t, true, "foobb", "@(!(z*)|*x)", win)
		// extglobs-temp.js:686
		assertMatch(t, true, "foofoofo", "@(!(z*)|*x)", win)
		// extglobs-temp.js:687
		assertMatch(t, true, "fooofoofofooo", "@(!(z*)|*x)", win)
		// extglobs-temp.js:688
		assertMatch(t, true, "foooofo", "@(!(z*)|*x)", win)
		// extglobs-temp.js:689
		assertMatch(t, true, "foooofof", "@(!(z*)|*x)", win)
		// extglobs-temp.js:690
		assertMatch(t, true, "foooofofx", "@(!(z*)|*x)", win)
		// extglobs-temp.js:691
		assertMatch(t, true, "foooxfooxfoxfooox", "@(!(z*)|*x)", win)
		// extglobs-temp.js:692
		assertMatch(t, true, "foooxfooxfxfooox", "@(!(z*)|*x)", win)
		// extglobs-temp.js:693
		assertMatch(t, true, "foooxfooxofoxfooox", "@(!(z*)|*x)", win)
		// extglobs-temp.js:694
		assertMatch(t, true, "foot", "@(!(z*)|*x)", win)
		// extglobs-temp.js:695
		assertMatch(t, true, "foox", "@(!(z*)|*x)", win)
		// extglobs-temp.js:696
		assertMatch(t, true, "ofoofo", "@(!(z*)|*x)", win)
		// extglobs-temp.js:697
		assertMatch(t, true, "ofooofoofofooo", "@(!(z*)|*x)", win)
		// extglobs-temp.js:698
		assertMatch(t, true, "ofoooxoofxo", "@(!(z*)|*x)", win)
		// extglobs-temp.js:699
		assertMatch(t, true, "ofoooxoofxoofoooxoofxo", "@(!(z*)|*x)", win)
		// extglobs-temp.js:700
		assertMatch(t, true, "ofoooxoofxoofoooxoofxofo", "@(!(z*)|*x)", win)
		// extglobs-temp.js:701
		assertMatch(t, true, "ofoooxoofxoofoooxoofxoo", "@(!(z*)|*x)", win)
		// extglobs-temp.js:702
		assertMatch(t, true, "ofoooxoofxoofoooxoofxooofxofxo", "@(!(z*)|*x)", win)
		// extglobs-temp.js:703
		assertMatch(t, true, "ofxoofxo", "@(!(z*)|*x)", win)
		// extglobs-temp.js:704
		assertMatch(t, true, "oofooofo", "@(!(z*)|*x)", win)
		// extglobs-temp.js:705
		assertMatch(t, true, "ooo", "@(!(z*)|*x)", win)
		// extglobs-temp.js:706
		assertMatch(t, true, "oxfoxfox", "@(!(z*)|*x)", win)
		// extglobs-temp.js:707
		assertMatch(t, true, "oxfoxoxfox", "@(!(z*)|*x)", win)
		// extglobs-temp.js:708
		assertMatch(t, true, "xfoooofof", "@(!(z*)|*x)", win)
	})

	t.Run("pattern @(foo|f|fo)*(f|of+(o)) windows", func(t *testing.T) {
		// extglobs-temp.js:710
		assertMatch(t, false, "ffffffo", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:711
		assertMatch(t, false, "fffooofoooooffoofffooofff", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:712
		assertMatch(t, false, "ffo", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:713
		assertMatch(t, true, "fofo", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:714
		assertMatch(t, true, "fofoofoofofoo", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:715
		assertMatch(t, true, "foo", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:716
		assertMatch(t, false, "foob", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:717
		assertMatch(t, false, "foobb", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:718
		assertMatch(t, true, "foofoofo", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:719
		assertMatch(t, true, "fooofoofofooo", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:720
		assertMatch(t, false, "foooofo", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:721
		assertMatch(t, false, "foooofof", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:722
		assertMatch(t, false, "foooofofx", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:723
		assertMatch(t, false, "foooxfooxfoxfooox", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:724
		assertMatch(t, false, "foooxfooxfxfooox", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:725
		assertMatch(t, false, "foooxfooxofoxfooox", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:726
		assertMatch(t, false, "foot", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:727
		assertMatch(t, false, "foox", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:728
		assertMatch(t, false, "ofoofo", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:729
		assertMatch(t, false, "ofooofoofofooo", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:730
		assertMatch(t, false, "ofoooxoofxo", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:731
		assertMatch(t, false, "ofoooxoofxoofoooxoofxo", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:732
		assertMatch(t, false, "ofoooxoofxoofoooxoofxofo", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:733
		assertMatch(t, false, "ofoooxoofxoofoooxoofxoo", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:734
		assertMatch(t, false, "ofoooxoofxoofoooxoofxooofxofxo", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:735
		assertMatch(t, false, "ofxoofxo", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:736
		assertMatch(t, false, "oofooofo", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:737
		assertMatch(t, false, "ooo", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:738
		assertMatch(t, false, "oxfoxfox", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:739
		assertMatch(t, false, "oxfoxoxfox", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:740
		assertMatch(t, false, "xfoooofof", "@(foo|f|fo)*(f|of+(o))", win)
	})

	t.Run("pattern *(@(a))a@(c) windows", func(t *testing.T) {
		// extglobs-temp.js:742
		assertMatch(t, true, "aaac", "*(@(a))a@(c)", win)
		// extglobs-temp.js:743
		assertMatch(t, true, "aac", "*(@(a))a@(c)", win)
		// extglobs-temp.js:744
		assertMatch(t, true, "ac", "*(@(a))a@(c)", win)
		// extglobs-temp.js:745
		assertMatch(t, false, "abbcd", "*(@(a))a@(c)", win)
		// extglobs-temp.js:746
		assertMatch(t, false, "abcd", "*(@(a))a@(c)", win)
		// extglobs-temp.js:747
		assertMatch(t, false, "acd", "*(@(a))a@(c)", win)
		// extglobs-temp.js:748
		assertMatch(t, false, "baaac", "*(@(a))a@(c)", win)
		// extglobs-temp.js:749
		assertMatch(t, false, "c", "*(@(a))a@(c)", win)
		// extglobs-temp.js:750
		assertMatch(t, false, "foo", "*(@(a))a@(c)", win)
	})

	t.Run("pattern @(ab|a*(b))*(c)d windows", func(t *testing.T) {
		// extglobs-temp.js:752
		assertMatch(t, false, "aaac", "@(ab|a*(b))*(c)d", win)
		// extglobs-temp.js:753
		assertMatch(t, false, "aac", "@(ab|a*(b))*(c)d", win)
		// extglobs-temp.js:754
		assertMatch(t, false, "ac", "@(ab|a*(b))*(c)d", win)
		// extglobs-temp.js:755
		assertMatch(t, true, "abbcd", "@(ab|a*(b))*(c)d", win)
		// extglobs-temp.js:756
		assertMatch(t, true, "abcd", "@(ab|a*(b))*(c)d", win)
		// extglobs-temp.js:757
		assertMatch(t, true, "acd", "@(ab|a*(b))*(c)d", win)
		// extglobs-temp.js:758
		assertMatch(t, false, "baaac", "@(ab|a*(b))*(c)d", win)
		// extglobs-temp.js:759
		assertMatch(t, false, "c", "@(ab|a*(b))*(c)d", win)
		// extglobs-temp.js:760
		assertMatch(t, false, "foo", "@(ab|a*(b))*(c)d", win)
	})

	t.Run("pattern ?@(a|b)*@(c)d windows", func(t *testing.T) {
		// extglobs-temp.js:762
		assertMatch(t, false, "aaac", "?@(a|b)*@(c)d", win)
		// extglobs-temp.js:763
		assertMatch(t, false, "aac", "?@(a|b)*@(c)d", win)
		// extglobs-temp.js:764
		assertMatch(t, false, "ac", "?@(a|b)*@(c)d", win)
		// extglobs-temp.js:765
		assertMatch(t, true, "abbcd", "?@(a|b)*@(c)d", win)
		// extglobs-temp.js:766
		assertMatch(t, true, "abcd", "?@(a|b)*@(c)d", win)
		// extglobs-temp.js:767
		assertMatch(t, false, "acd", "?@(a|b)*@(c)d", win)
		// extglobs-temp.js:768
		assertMatch(t, false, "baaac", "?@(a|b)*@(c)d", win)
		// extglobs-temp.js:769
		assertMatch(t, false, "c", "?@(a|b)*@(c)d", win)
		// extglobs-temp.js:770
		assertMatch(t, false, "foo", "?@(a|b)*@(c)d", win)
	})

	t.Run("pattern @(ab|a*@(b))*(c)d windows", func(t *testing.T) {
		// extglobs-temp.js:772
		assertMatch(t, false, "aaac", "@(ab|a*@(b))*(c)d", win)
		// extglobs-temp.js:773
		assertMatch(t, false, "aac", "@(ab|a*@(b))*(c)d", win)
		// extglobs-temp.js:774
		assertMatch(t, false, "ac", "@(ab|a*@(b))*(c)d", win)
		// extglobs-temp.js:775
		assertMatch(t, true, "abbcd", "@(ab|a*@(b))*(c)d", win)
		// extglobs-temp.js:776
		assertMatch(t, true, "abcd", "@(ab|a*@(b))*(c)d", win)
		// extglobs-temp.js:777
		assertMatch(t, false, "acd", "@(ab|a*@(b))*(c)d", win)
		// extglobs-temp.js:778
		assertMatch(t, false, "baaac", "@(ab|a*@(b))*(c)d", win)
		// extglobs-temp.js:779
		assertMatch(t, false, "c", "@(ab|a*@(b))*(c)d", win)
		// extglobs-temp.js:780
		assertMatch(t, false, "foo", "@(ab|a*@(b))*(c)d", win)
	})

	t.Run("pattern *(@(a))b@(c) windows", func(t *testing.T) {
		// extglobs-temp.js:782
		assertMatch(t, false, "aac", "*(@(a))b@(c)", win)
	})

	t.Run("other - should support backtracking in alternation matches", func(t *testing.T) {
		// extglobs-temp.js:789
		assertMatch(t, true, "fofoofoofofoo", "*(fo|foo)", win)
		// extglobs-temp.js:790
		assertMatch(t, false, "ffffffo", "*(fo|foo)", win)
		// extglobs-temp.js:791
		assertMatch(t, false, "fffooofoooooffoofffooofff", "*(fo|foo)", win)
		// extglobs-temp.js:792
		assertMatch(t, false, "ffo", "*(fo|foo)", win)
		// extglobs-temp.js:793
		assertMatch(t, true, "fofo", "*(fo|foo)", win)
		// extglobs-temp.js:794
		assertMatch(t, true, "fofoofoofofoo", "*(fo|foo)", win)
		// extglobs-temp.js:795
		assertMatch(t, true, "foo", "*(fo|foo)", win)
		// extglobs-temp.js:796
		assertMatch(t, false, "foob", "*(fo|foo)", win)
		// extglobs-temp.js:797
		assertMatch(t, false, "foobb", "*(fo|foo)", win)
		// extglobs-temp.js:798
		assertMatch(t, true, "foofoofo", "*(fo|foo)", win)
		// extglobs-temp.js:799
		assertMatch(t, false, "fooofoofofooo", "*(fo|foo)", win)
		// extglobs-temp.js:800
		assertMatch(t, false, "foooofo", "*(fo|foo)", win)
		// extglobs-temp.js:801
		assertMatch(t, false, "foooofof", "*(fo|foo)", win)
		// extglobs-temp.js:802
		assertMatch(t, false, "foooofofx", "*(fo|foo)", win)
		// extglobs-temp.js:803
		assertMatch(t, false, "foooxfooxfoxfooox", "*(fo|foo)", win)
		// extglobs-temp.js:804
		assertMatch(t, false, "foooxfooxfxfooox", "*(fo|foo)", win)
		// extglobs-temp.js:805
		assertMatch(t, false, "foooxfooxofoxfooox", "*(fo|foo)", win)
		// extglobs-temp.js:806
		assertMatch(t, false, "foot", "*(fo|foo)", win)
		// extglobs-temp.js:807
		assertMatch(t, false, "foox", "*(fo|foo)", win)
		// extglobs-temp.js:808
		assertMatch(t, false, "ofoofo", "*(fo|foo)", win)
		// extglobs-temp.js:809
		assertMatch(t, false, "ofooofoofofooo", "*(fo|foo)", win)
		// extglobs-temp.js:810
		assertMatch(t, false, "ofoooxoofxo", "*(fo|foo)", win)
		// extglobs-temp.js:811
		assertMatch(t, false, "ofoooxoofxoofoooxoofxo", "*(fo|foo)", win)
		// extglobs-temp.js:812
		assertMatch(t, false, "ofoooxoofxoofoooxoofxofo", "*(fo|foo)", win)
		// extglobs-temp.js:813
		assertMatch(t, false, "ofoooxoofxoofoooxoofxoo", "*(fo|foo)", win)
		// extglobs-temp.js:814
		assertMatch(t, false, "ofoooxoofxoofoooxoofxooofxofxo", "*(fo|foo)", win)
		// extglobs-temp.js:815
		assertMatch(t, false, "ofxoofxo", "*(fo|foo)", win)
		// extglobs-temp.js:816
		assertMatch(t, false, "oofooofo", "*(fo|foo)", win)
		// extglobs-temp.js:817
		assertMatch(t, false, "ooo", "*(fo|foo)", win)
		// extglobs-temp.js:818
		assertMatch(t, false, "oxfoxfox", "*(fo|foo)", win)
		// extglobs-temp.js:819
		assertMatch(t, false, "oxfoxoxfox", "*(fo|foo)", win)
		// extglobs-temp.js:820
		assertMatch(t, false, "xfoooofof", "*(fo|foo)", win)
	})

	t.Run("other - should support exclusions", func(t *testing.T) {
		// extglobs-temp.js:824
		assertMatch(t, false, "foob", "!(foo)b*", win)
		// extglobs-temp.js:825
		assertMatch(t, false, "foobb", "!(foo)b*", win)
		// extglobs-temp.js:826
		assertMatch(t, false, "foo", "!(foo)b*", win)
		// extglobs-temp.js:827
		assertMatch(t, true, "bar", "!(foo)b*", win)
		// extglobs-temp.js:828
		assertMatch(t, true, "baz", "!(foo)b*", win)
		// extglobs-temp.js:829
		assertMatch(t, false, "foobar", "!(foo)b*", win)

		// extglobs-temp.js:831
		assertMatch(t, false, "foo", "*(!(foo))", win)
		// extglobs-temp.js:832
		assertMatch(t, true, "bar", "*(!(foo))", win)
		// extglobs-temp.js:833
		assertMatch(t, true, "baz", "*(!(foo))", win)
		// extglobs-temp.js:834
		assertMatch(t, true, "foobar", "*(!(foo))", win)

		// Bash 4.3 says this should match `foo` and `foobar`, which makes no sense
		// extglobs-temp.js:837
		assertMatch(t, false, "foo", "!(foo)*", win)
		// extglobs-temp.js:838
		assertMatch(t, false, "foobar", "!(foo)*", win)
		// extglobs-temp.js:839
		assertMatch(t, true, "bar", "!(foo)*", win)
		// extglobs-temp.js:840
		assertMatch(t, true, "baz", "!(foo)*", win)

		// extglobs-temp.js:842
		assertMatch(t, false, "moo.cow", "!(*.*)", win)
		// extglobs-temp.js:843
		assertMatch(t, true, "moo", "!(*.*)", win)
		// extglobs-temp.js:844
		assertMatch(t, true, "cow", "!(*.*)", win)

		// extglobs-temp.js:846
		assertMatch(t, true, "moo.cow", "!(a*).!(b*)", win)
		// extglobs-temp.js:847
		assertMatch(t, false, "moo.cow", "!(*).!(*)", win)
		// extglobs-temp.js:848
		assertMatch(t, false, "moo.cow.moo.cow", "!(*.*).!(*.*)", win)
		// extglobs-temp.js:849
		assertMatch(t, false, "mad.moo.cow", "!(*.*).!(*.*)", win)

		// extglobs-temp.js:851
		assertMatch(t, false, "moo.cow", "!(*.*).", win)
		// extglobs-temp.js:852
		assertMatch(t, false, "moo", "!(*.*).", win)
		// extglobs-temp.js:853
		assertMatch(t, false, "cow", "!(*.*).", win)

		// extglobs-temp.js:855
		assertMatch(t, false, "moo.cow", ".!(*.*)", win)
		// extglobs-temp.js:856
		assertMatch(t, false, "moo", ".!(*.*)", win)
		// extglobs-temp.js:857
		assertMatch(t, false, "cow", ".!(*.*)", win)

		// extglobs-temp.js:859
		assertMatch(t, false, "mucca.pazza", "mu!(*(c))?.pa!(*(z))?", win)

		// extglobs-temp.js:861
		assertMatch(t, true, "effgz", "@(b+(c)d|e*(f)g?|?(h)i@(j|k))", win)
		// extglobs-temp.js:862
		assertMatch(t, true, "efgz", "@(b+(c)d|e*(f)g?|?(h)i@(j|k))", win)
		// extglobs-temp.js:863
		assertMatch(t, true, "egz", "@(b+(c)d|e*(f)g?|?(h)i@(j|k))", win)
		// extglobs-temp.js:864
		assertMatch(t, false, "egz", "@(b+(c)d|e+(f)g?|?(h)i@(j|k))", win)
		// extglobs-temp.js:865
		assertMatch(t, true, "egzefffgzbcdij", "*(b+(c)d|e*(f)g?|?(h)i@(j|k))", win)
	})

	t.Run("other - valid numbers", func(t *testing.T) {
		// extglobs-temp.js:869
		assertMatch(t, true, "/dev/udp/129.22.8.102/45", "/dev/@(tcp|udp)/*/*", win)

		// extglobs-temp.js:871
		assertMatch(t, false, "0", "[1-6]([0-9])", win)
		// extglobs-temp.js:872
		assertMatch(t, true, "12", "[1-6]([0-9])", win)
		// extglobs-temp.js:873
		assertMatch(t, false, "1", "[1-6]([0-9])", win)
		// extglobs-temp.js:874
		assertMatch(t, false, "12abc", "[1-6]([0-9])", win)
		// extglobs-temp.js:875
		assertMatch(t, false, "555", "[1-6]([0-9])", win)

		// extglobs-temp.js:877
		assertMatch(t, false, "0", "[1-6]*([0-9])", win)
		// extglobs-temp.js:878
		assertMatch(t, true, "12", "[1-6]*([0-9])", win)
		// extglobs-temp.js:879
		assertMatch(t, true, "1", "[1-6]*([0-9])", win)
		// extglobs-temp.js:880
		assertMatch(t, false, "12abc", "[1-6]*([0-9])", win)
		// extglobs-temp.js:881
		assertMatch(t, true, "555", "[1-6]*([0-9])", win)

		// extglobs-temp.js:883
		assertMatch(t, false, "0", "[1-5]*([6-9])", win)
		// extglobs-temp.js:884
		assertMatch(t, false, "12", "[1-5]*([6-9])", win)
		// extglobs-temp.js:885
		assertMatch(t, true, "1", "[1-5]*([6-9])", win)
		// extglobs-temp.js:886
		assertMatch(t, false, "12abc", "[1-5]*([6-9])", win)
		// extglobs-temp.js:887
		assertMatch(t, false, "555", "[1-5]*([6-9])", win)

		// extglobs-temp.js:889
		assertMatch(t, true, "0", "0|[1-6]*([0-9])", win)
		// extglobs-temp.js:890
		assertMatch(t, true, "12", "0|[1-6]*([0-9])", win)
		// extglobs-temp.js:891
		assertMatch(t, true, "1", "0|[1-6]*([0-9])", win)
		// extglobs-temp.js:892
		assertMatch(t, false, "12abc", "0|[1-6]*([0-9])", win)
		// extglobs-temp.js:893
		assertMatch(t, true, "555", "0|[1-6]*([0-9])", win)

		// extglobs-temp.js:895
		assertMatch(t, true, "07", "+([0-7])", win)
		// extglobs-temp.js:896
		assertMatch(t, true, "0377", "+([0-7])", win)
		// extglobs-temp.js:897
		assertMatch(t, false, "09", "+([0-7])", win)
	})

	t.Run("other - check extended globbing in pattern removal", func(t *testing.T) {
		// extglobs-temp.js:901
		assertMatch(t, true, "a", "+(a|abc)", win)
		// extglobs-temp.js:902
		assertMatch(t, true, "abc", "+(a|abc)", win)

		// extglobs-temp.js:904
		assertMatch(t, false, "abcd", "+(a|abc)", win)
		// extglobs-temp.js:905
		assertMatch(t, false, "abcde", "+(a|abc)", win)
		// extglobs-temp.js:906
		assertMatch(t, false, "abcedf", "+(a|abc)", win)

		// extglobs-temp.js:908
		assertMatch(t, true, "f", "+(def|f)", win)
		// extglobs-temp.js:909
		assertMatch(t, true, "def", "+(f|def)", win)

		// extglobs-temp.js:911
		assertMatch(t, false, "cdef", "+(f|def)", win)
		// extglobs-temp.js:912
		assertMatch(t, false, "bcdef", "+(f|def)", win)
		// extglobs-temp.js:913
		assertMatch(t, false, "abcedf", "+(f|def)", win)

		// extglobs-temp.js:915
		assertMatch(t, true, "abcd", "*(a|b)cd", win)

		// extglobs-temp.js:917
		assertMatch(t, false, "a", "*(a|b)cd", win)
		// extglobs-temp.js:918
		assertMatch(t, false, "ab", "*(a|b)cd", win)
		// extglobs-temp.js:919
		assertMatch(t, false, "abc", "*(a|b)cd", win)

		// extglobs-temp.js:921
		assertMatch(t, false, "a", "\"*(a|b)cd\"", win)
		// extglobs-temp.js:922
		assertMatch(t, false, "ab", "\"*(a|b)cd\"", win)
		// extglobs-temp.js:923
		assertMatch(t, false, "abc", "\"*(a|b)cd\"", win)
		// extglobs-temp.js:924
		assertMatch(t, false, "abcde", "\"*(a|b)cd\"", win)
		// extglobs-temp.js:925
		assertMatch(t, false, "abcdef", "\"*(a|b)cd\"", win)
	})

	t.Run("other - bug report extended glob patterns following a star", func(t *testing.T) {
		// extglobs-temp.js:929
		assertMatch(t, true, "/dev/udp/129.22.8.102/45", "/dev\\/@(tcp|udp)\\/*\\/*", win)
		// extglobs-temp.js:930
		assertMatch(t, false, "123abc", "(a+|b)*", win)
		// extglobs-temp.js:931
		assertMatch(t, true, "ab", "(a+|b)*", win)
		// extglobs-temp.js:932
		assertMatch(t, true, "abab", "(a+|b)*", win)
		// extglobs-temp.js:933
		assertMatch(t, true, "abcdef", "(a+|b)*", win)
		// extglobs-temp.js:934
		assertMatch(t, true, "accdef", "(a+|b)*", win)
		// extglobs-temp.js:935
		assertMatch(t, true, "abcfefg", "(a+|b)*", win)
		// extglobs-temp.js:936
		assertMatch(t, true, "abef", "(a+|b)*", win)
		// extglobs-temp.js:937
		assertMatch(t, true, "abcfef", "(a+|b)*", win)
		// extglobs-temp.js:938
		assertMatch(t, true, "abd", "(a+|b)*", win)
		// extglobs-temp.js:939
		assertMatch(t, true, "acd", "(a+|b)*", win)

		// extglobs-temp.js:941
		assertMatch(t, false, "123abc", "(a+|b)+", win)
		// extglobs-temp.js:942
		assertMatch(t, true, "ab", "(a+|b)+", win)
		// extglobs-temp.js:943
		assertMatch(t, true, "abab", "(a+|b)+", win)
		// extglobs-temp.js:944
		assertMatch(t, false, "abcdef", "(a+|b)+", win)
		// extglobs-temp.js:945
		assertMatch(t, false, "accdef", "(a+|b)+", win)
		// extglobs-temp.js:946
		assertMatch(t, false, "abcfefg", "(a+|b)+", win)
		// extglobs-temp.js:947
		assertMatch(t, false, "abef", "(a+|b)+", win)
		// extglobs-temp.js:948
		assertMatch(t, false, "abcfef", "(a+|b)+", win)
		// extglobs-temp.js:949
		assertMatch(t, false, "abd", "(a+|b)+", win)
		// extglobs-temp.js:950
		assertMatch(t, false, "acd", "(a+|b)+", win)

		// extglobs-temp.js:952
		assertMatch(t, false, "123abc", "a(b*(foo|bar))d", win)
		// extglobs-temp.js:953
		assertMatch(t, false, "ab", "a(b*(foo|bar))d", win)
		// extglobs-temp.js:954
		assertMatch(t, false, "abab", "a(b*(foo|bar))d", win)
		// extglobs-temp.js:955
		assertMatch(t, false, "abcdef", "a(b*(foo|bar))d", win)
		// extglobs-temp.js:956
		assertMatch(t, false, "accdef", "a(b*(foo|bar))d", win)
		// extglobs-temp.js:957
		assertMatch(t, false, "abcfefg", "a(b*(foo|bar))d", win)
		// extglobs-temp.js:958
		assertMatch(t, false, "abef", "a(b*(foo|bar))d", win)
		// extglobs-temp.js:959
		assertMatch(t, false, "abcfef", "a(b*(foo|bar))d", win)
		// extglobs-temp.js:960
		assertMatch(t, true, "abd", "a(b*(foo|bar))d", win)
		// extglobs-temp.js:961
		assertMatch(t, false, "acd", "a(b*(foo|bar))d", win)

		// extglobs-temp.js:963
		assertMatch(t, false, "123abc", "ab*(e|f)", win)
		// extglobs-temp.js:964
		assertMatch(t, true, "ab", "ab*(e|f)", win)
		// extglobs-temp.js:965
		assertMatch(t, false, "abab", "ab*(e|f)", win)
		// extglobs-temp.js:966
		assertMatch(t, false, "abcdef", "ab*(e|f)", win)
		// extglobs-temp.js:967
		assertMatch(t, false, "accdef", "ab*(e|f)", win)
		// extglobs-temp.js:968
		assertMatch(t, false, "abcfefg", "ab*(e|f)", win)
		// extglobs-temp.js:969
		assertMatch(t, true, "abef", "ab*(e|f)", win)
		// extglobs-temp.js:970
		assertMatch(t, false, "abcfef", "ab*(e|f)", win)
		// extglobs-temp.js:971
		assertMatch(t, false, "abd", "ab*(e|f)", win)
		// extglobs-temp.js:972
		assertMatch(t, false, "acd", "ab*(e|f)", win)

		// extglobs-temp.js:974
		assertMatch(t, false, "123abc", "ab**(e|f)", win)
		// extglobs-temp.js:975
		assertMatch(t, true, "ab", "ab**(e|f)", win)
		// extglobs-temp.js:976
		assertMatch(t, true, "abab", "ab**(e|f)", win)
		// extglobs-temp.js:977
		assertMatch(t, true, "abcdef", "ab**(e|f)", win)
		// extglobs-temp.js:978
		assertMatch(t, false, "accdef", "ab**(e|f)", win)
		// extglobs-temp.js:979
		assertMatch(t, true, "abcfefg", "ab**(e|f)", win)
		// extglobs-temp.js:980
		assertMatch(t, true, "abef", "ab**(e|f)", win)
		// extglobs-temp.js:981
		assertMatch(t, true, "abcfef", "ab**(e|f)", win)
		// extglobs-temp.js:982
		assertMatch(t, true, "abd", "ab**(e|f)", win)
		// extglobs-temp.js:983
		assertMatch(t, false, "acd", "ab**(e|f)", win)

		// extglobs-temp.js:985
		assertMatch(t, false, "123abc", "ab**(e|f)g", win)
		// extglobs-temp.js:986
		assertMatch(t, false, "ab", "ab**(e|f)g", win)
		// extglobs-temp.js:987
		assertMatch(t, false, "abab", "ab**(e|f)g", win)
		// extglobs-temp.js:988
		assertMatch(t, false, "abcdef", "ab**(e|f)g", win)
		// extglobs-temp.js:989
		assertMatch(t, false, "accdef", "ab**(e|f)g", win)
		// extglobs-temp.js:990
		assertMatch(t, true, "abcfefg", "ab**(e|f)g", win)
		// extglobs-temp.js:991
		assertMatch(t, false, "abef", "ab**(e|f)g", win)
		// extglobs-temp.js:992
		assertMatch(t, false, "abcfef", "ab**(e|f)g", win)
		// extglobs-temp.js:993
		assertMatch(t, false, "abd", "ab**(e|f)g", win)
		// extglobs-temp.js:994
		assertMatch(t, false, "acd", "ab**(e|f)g", win)

		// extglobs-temp.js:996
		assertMatch(t, false, "123abc", "ab***ef", win)
		// extglobs-temp.js:997
		assertMatch(t, false, "ab", "ab***ef", win)
		// extglobs-temp.js:998
		assertMatch(t, false, "abab", "ab***ef", win)
		// extglobs-temp.js:999
		assertMatch(t, true, "abcdef", "ab***ef", win)
		// extglobs-temp.js:1000
		assertMatch(t, false, "accdef", "ab***ef", win)
		// extglobs-temp.js:1001
		assertMatch(t, false, "abcfefg", "ab***ef", win)
		// extglobs-temp.js:1002
		assertMatch(t, true, "abef", "ab***ef", win)
		// extglobs-temp.js:1003
		assertMatch(t, true, "abcfef", "ab***ef", win)
		// extglobs-temp.js:1004
		assertMatch(t, false, "abd", "ab***ef", win)
		// extglobs-temp.js:1005
		assertMatch(t, false, "acd", "ab***ef", win)

		// extglobs-temp.js:1007
		assertMatch(t, false, "123abc", "ab*+(e|f)", win)
		// extglobs-temp.js:1008
		assertMatch(t, false, "ab", "ab*+(e|f)", win)
		// extglobs-temp.js:1009
		assertMatch(t, false, "abab", "ab*+(e|f)", win)
		// extglobs-temp.js:1010
		assertMatch(t, true, "abcdef", "ab*+(e|f)", win)
		// extglobs-temp.js:1011
		assertMatch(t, false, "accdef", "ab*+(e|f)", win)
		// extglobs-temp.js:1012
		assertMatch(t, false, "abcfefg", "ab*+(e|f)", win)
		// extglobs-temp.js:1013
		assertMatch(t, true, "abef", "ab*+(e|f)", win)
		// extglobs-temp.js:1014
		assertMatch(t, true, "abcfef", "ab*+(e|f)", win)
		// extglobs-temp.js:1015
		assertMatch(t, false, "abd", "ab*+(e|f)", win)
		// extglobs-temp.js:1016
		assertMatch(t, false, "acd", "ab*+(e|f)", win)

		// extglobs-temp.js:1018
		assertMatch(t, false, "123abc", "ab*d*(e|f)", win)
		// extglobs-temp.js:1019
		assertMatch(t, false, "ab", "ab*d*(e|f)", win)
		// extglobs-temp.js:1020
		assertMatch(t, false, "abab", "ab*d*(e|f)", win)
		// extglobs-temp.js:1021
		assertMatch(t, true, "abcdef", "ab*d*(e|f)", win)
		// extglobs-temp.js:1022
		assertMatch(t, false, "accdef", "ab*d*(e|f)", win)
		// extglobs-temp.js:1023
		assertMatch(t, false, "abcfefg", "ab*d*(e|f)", win)
		// extglobs-temp.js:1024
		assertMatch(t, false, "abef", "ab*d*(e|f)", win)
		// extglobs-temp.js:1025
		assertMatch(t, false, "abcfef", "ab*d*(e|f)", win)
		// extglobs-temp.js:1026
		assertMatch(t, true, "abd", "ab*d*(e|f)", win)
		// extglobs-temp.js:1027
		assertMatch(t, false, "acd", "ab*d*(e|f)", win)

		// extglobs-temp.js:1029
		assertMatch(t, false, "123abc", "ab*d+(e|f)", win)
		// extglobs-temp.js:1030
		assertMatch(t, false, "ab", "ab*d+(e|f)", win)
		// extglobs-temp.js:1031
		assertMatch(t, false, "abab", "ab*d+(e|f)", win)
		// extglobs-temp.js:1032
		assertMatch(t, true, "abcdef", "ab*d+(e|f)", win)
		// extglobs-temp.js:1033
		assertMatch(t, false, "accdef", "ab*d+(e|f)", win)
		// extglobs-temp.js:1034
		assertMatch(t, false, "abcfefg", "ab*d+(e|f)", win)
		// extglobs-temp.js:1035
		assertMatch(t, false, "abef", "ab*d+(e|f)", win)
		// extglobs-temp.js:1036
		assertMatch(t, false, "abcfef", "ab*d+(e|f)", win)
		// extglobs-temp.js:1037
		assertMatch(t, false, "abd", "ab*d+(e|f)", win)
		// extglobs-temp.js:1038
		assertMatch(t, false, "acd", "ab*d+(e|f)", win)

		// extglobs-temp.js:1040
		assertMatch(t, false, "123abc", "ab?*(e|f)", win)
		// extglobs-temp.js:1041
		assertMatch(t, false, "ab", "ab?*(e|f)", win)
		// extglobs-temp.js:1042
		assertMatch(t, false, "abab", "ab?*(e|f)", win)
		// extglobs-temp.js:1043
		assertMatch(t, false, "abcdef", "ab?*(e|f)", win)
		// extglobs-temp.js:1044
		assertMatch(t, false, "accdef", "ab?*(e|f)", win)
		// extglobs-temp.js:1045
		assertMatch(t, false, "abcfefg", "ab?*(e|f)", win)
		// extglobs-temp.js:1046
		assertMatch(t, true, "abef", "ab?*(e|f)", win)
		// extglobs-temp.js:1047
		assertMatch(t, true, "abcfef", "ab?*(e|f)", win)
		// extglobs-temp.js:1048
		assertMatch(t, true, "abd", "ab?*(e|f)", win)
		// extglobs-temp.js:1049
		assertMatch(t, false, "acd", "ab?*(e|f)", win)
	})

	t.Run("other - bug in all versions up to and including bash-2.05b", func(t *testing.T) {
		// extglobs-temp.js:1053
		assertMatch(t, true, "123abc", "*?(a)bc", win)
	})

	t.Run("other - should work with character classes", func(t *testing.T) {
		// extglobs-temp.js:1058
		assertMatch(t, true, "a.b", "a[^[:alnum:]]b", posix)
		// extglobs-temp.js:1059
		assertMatch(t, true, "a,b", "a[^[:alnum:]]b", posix)
		// extglobs-temp.js:1060
		assertMatch(t, true, "a:b", "a[^[:alnum:]]b", posix)
		// extglobs-temp.js:1061
		assertMatch(t, true, "a-b", "a[^[:alnum:]]b", posix)
		// extglobs-temp.js:1062
		assertMatch(t, true, "a;b", "a[^[:alnum:]]b", posix)
		// extglobs-temp.js:1063
		assertMatch(t, true, "a b", "a[^[:alnum:]]b", posix)
		// extglobs-temp.js:1064
		assertMatch(t, true, "a_b", "a[^[:alnum:]]b", posix)

		// extglobs-temp.js:1066
		assertMatch(t, true, "a.b", "a[-.,:\\;\\ _]b", win)
		// extglobs-temp.js:1067
		assertMatch(t, true, "a,b", "a[-.,:\\;\\ _]b", win)
		// extglobs-temp.js:1068
		assertMatch(t, true, "a:b", "a[-.,:\\;\\ _]b", win)
		// extglobs-temp.js:1069
		assertMatch(t, true, "a-b", "a[-.,:\\;\\ _]b", win)
		// extglobs-temp.js:1070
		assertMatch(t, true, "a;b", "a[-.,:\\;\\ _]b", win)
		// extglobs-temp.js:1071
		assertMatch(t, true, "a b", "a[-.,:\\;\\ _]b", win)
		// extglobs-temp.js:1072
		assertMatch(t, true, "a_b", "a[-.,:\\;\\ _]b", win)

		// extglobs-temp.js:1074
		assertMatch(t, true, "a.b", "a@([^[:alnum:]])b", posix)
		// extglobs-temp.js:1075
		assertMatch(t, true, "a,b", "a@([^[:alnum:]])b", posix)
		// extglobs-temp.js:1076
		assertMatch(t, true, "a:b", "a@([^[:alnum:]])b", posix)
		// extglobs-temp.js:1077
		assertMatch(t, true, "a-b", "a@([^[:alnum:]])b", posix)
		// extglobs-temp.js:1078
		assertMatch(t, true, "a;b", "a@([^[:alnum:]])b", posix)
		// extglobs-temp.js:1079
		assertMatch(t, true, "a b", "a@([^[:alnum:]])b", posix)
		// extglobs-temp.js:1080
		assertMatch(t, true, "a_b", "a@([^[:alnum:]])b", posix)

		// extglobs-temp.js:1082
		assertMatch(t, true, "a.b", "a@([-.,:; _])b", win)
		// extglobs-temp.js:1083
		assertMatch(t, true, "a,b", "a@([-.,:; _])b", win)
		// extglobs-temp.js:1084
		assertMatch(t, true, "a:b", "a@([-.,:; _])b", win)
		// extglobs-temp.js:1085
		assertMatch(t, true, "a-b", "a@([-.,:; _])b", win)
		// extglobs-temp.js:1086
		assertMatch(t, true, "a;b", "a@([-.,:; _])b", win)
		// extglobs-temp.js:1087
		assertMatch(t, true, "a b", "a@([-.,:; _])b", win)
		// extglobs-temp.js:1088
		assertMatch(t, true, "a_b", "a@([-.,:; _])b", win)

		// extglobs-temp.js:1090
		assertMatch(t, true, "a.b", "a@([.])b", win)
		// extglobs-temp.js:1091
		assertMatch(t, false, "a,b", "a@([.])b", win)
		// extglobs-temp.js:1092
		assertMatch(t, false, "a:b", "a@([.])b", win)
		// extglobs-temp.js:1093
		assertMatch(t, false, "a-b", "a@([.])b", win)
		// extglobs-temp.js:1094
		assertMatch(t, false, "a;b", "a@([.])b", win)
		// extglobs-temp.js:1095
		assertMatch(t, false, "a b", "a@([.])b", win)
		// extglobs-temp.js:1096
		assertMatch(t, false, "a_b", "a@([.])b", win)

		// extglobs-temp.js:1098
		assertMatch(t, false, "a.b", "a@([^.])b", win)
		// extglobs-temp.js:1099
		assertMatch(t, true, "a,b", "a@([^.])b", win)
		// extglobs-temp.js:1100
		assertMatch(t, true, "a:b", "a@([^.])b", win)
		// extglobs-temp.js:1101
		assertMatch(t, true, "a-b", "a@([^.])b", win)
		// extglobs-temp.js:1102
		assertMatch(t, true, "a;b", "a@([^.])b", win)
		// extglobs-temp.js:1103
		assertMatch(t, true, "a b", "a@([^.])b", win)
		// extglobs-temp.js:1104
		assertMatch(t, true, "a_b", "a@([^.])b", win)

		// extglobs-temp.js:1106
		assertMatch(t, true, "a.b", "a@([^x])b", win)
		// extglobs-temp.js:1107
		assertMatch(t, true, "a,b", "a@([^x])b", win)
		// extglobs-temp.js:1108
		assertMatch(t, true, "a:b", "a@([^x])b", win)
		// extglobs-temp.js:1109
		assertMatch(t, true, "a-b", "a@([^x])b", win)
		// extglobs-temp.js:1110
		assertMatch(t, true, "a;b", "a@([^x])b", win)
		// extglobs-temp.js:1111
		assertMatch(t, true, "a b", "a@([^x])b", win)
		// extglobs-temp.js:1112
		assertMatch(t, true, "a_b", "a@([^x])b", win)

		// extglobs-temp.js:1114
		assertMatch(t, true, "a.b", "a+([^[:alnum:]])b", posix)
		// extglobs-temp.js:1115
		assertMatch(t, true, "a,b", "a+([^[:alnum:]])b", posix)
		// extglobs-temp.js:1116
		assertMatch(t, true, "a:b", "a+([^[:alnum:]])b", posix)
		// extglobs-temp.js:1117
		assertMatch(t, true, "a-b", "a+([^[:alnum:]])b", posix)
		// extglobs-temp.js:1118
		assertMatch(t, true, "a;b", "a+([^[:alnum:]])b", posix)
		// extglobs-temp.js:1119
		assertMatch(t, true, "a b", "a+([^[:alnum:]])b", posix)
		// extglobs-temp.js:1120
		assertMatch(t, true, "a_b", "a+([^[:alnum:]])b", posix)

		// extglobs-temp.js:1122
		assertMatch(t, true, "a.b", "a@(.|[^[:alnum:]])b", posix)
		// extglobs-temp.js:1123
		assertMatch(t, true, "a,b", "a@(.|[^[:alnum:]])b", posix)
		// extglobs-temp.js:1124
		assertMatch(t, true, "a:b", "a@(.|[^[:alnum:]])b", posix)
		// extglobs-temp.js:1125
		assertMatch(t, true, "a-b", "a@(.|[^[:alnum:]])b", posix)
		// extglobs-temp.js:1126
		assertMatch(t, true, "a;b", "a@(.|[^[:alnum:]])b", posix)
		// extglobs-temp.js:1127
		assertMatch(t, true, "a b", "a@(.|[^[:alnum:]])b", posix)
		// extglobs-temp.js:1128
		assertMatch(t, true, "a_b", "a@(.|[^[:alnum:]])b", posix)
	})

	t.Run("other - should support POSIX character classes in extglobs", func(t *testing.T) {
		// extglobs-temp.js:1133
		assertMatch(t, true, "a.c", "+([[:alpha:].])", posix)
		// extglobs-temp.js:1134
		assertMatch(t, true, "a.c", "+([[:alpha:].])+([[:alpha:].])", posix)
		// extglobs-temp.js:1135
		assertMatch(t, true, "a.c", "*([[:alpha:].])", posix)
		// extglobs-temp.js:1136
		assertMatch(t, true, "a.c", "*([[:alpha:].])*([[:alpha:].])", posix)
		// extglobs-temp.js:1137
		assertMatch(t, true, "a.c", "?([[:alpha:].])?([[:alpha:].])?([[:alpha:].])", posix)
		// extglobs-temp.js:1138
		assertMatch(t, true, "a.c", "@([[:alpha:].])@([[:alpha:].])@([[:alpha:].])", posix)
		// extglobs-temp.js:1139
		assertMatch(t, false, ".", "!(\\.)", posix)
		// extglobs-temp.js:1140
		assertMatch(t, false, ".", "!([[:alpha:].])", posix)
		// extglobs-temp.js:1141
		assertMatch(t, true, ".", "?([[:alpha:].])", posix)
		// extglobs-temp.js:1142
		assertMatch(t, true, ".", "@([[:alpha:].])", posix)
	})

	t.Run("other - should pass extglob2 tests", func(t *testing.T) {
		// extglobs-temp.js:1147
		assertMatch(t, false, "baaac", "*(@(a))a@(c)", win)
		// extglobs-temp.js:1148
		assertMatch(t, false, "c", "*(@(a))a@(c)", win)
		// extglobs-temp.js:1149
		assertMatch(t, false, "egz", "@(b+(c)d|e+(f)g?|?(h)i@(j|k))", win)
		// extglobs-temp.js:1150
		assertMatch(t, false, "foooofof", "*(f+(o))", win)
		// extglobs-temp.js:1151
		assertMatch(t, false, "foooofofx", "*(f*(o))", win)
		// extglobs-temp.js:1152
		assertMatch(t, false, "foooxfooxofoxfooox", "*(f*(o)x)", win)
		// extglobs-temp.js:1153
		assertMatch(t, false, "ofooofoofofooo", "*(f*(o))", win)
		// extglobs-temp.js:1154
		assertMatch(t, false, "ofoooxoofxoofoooxoofxofo", "*(*(of*(o)x)o)", win)
		// extglobs-temp.js:1155
		assertMatch(t, false, "oxfoxfox", "*(oxf+(ox))", win)
		// extglobs-temp.js:1156
		assertMatch(t, false, "xfoooofof", "*(f*(o))", win)
		// extglobs-temp.js:1157
		assertMatch(t, true, "aaac", "*(@(a))a@(c)", win)
		// extglobs-temp.js:1158
		assertMatch(t, true, "aac", "*(@(a))a@(c)", win)
		// extglobs-temp.js:1159
		assertMatch(t, true, "abbcd", "@(ab|a*(b))*(c)d", win)
		// extglobs-temp.js:1160
		assertMatch(t, true, "abcd", "?@(a|b)*@(c)d", win)
		// extglobs-temp.js:1161
		assertMatch(t, true, "abcd", "@(ab|a*@(b))*(c)d", win)
		// extglobs-temp.js:1162
		assertMatch(t, true, "ac", "*(@(a))a@(c)", win)
		// extglobs-temp.js:1163
		assertMatch(t, true, "acd", "@(ab|a*(b))*(c)d", win)
		// extglobs-temp.js:1164
		assertMatch(t, true, "effgz", "@(b+(c)d|e*(f)g?|?(h)i@(j|k))", win)
		// extglobs-temp.js:1165
		assertMatch(t, true, "efgz", "@(b+(c)d|e*(f)g?|?(h)i@(j|k))", win)
		// extglobs-temp.js:1166
		assertMatch(t, true, "egz", "@(b+(c)d|e*(f)g?|?(h)i@(j|k))", win)
		// extglobs-temp.js:1167
		assertMatch(t, true, "egzefffgzbcdij", "*(b+(c)d|e*(f)g?|?(h)i@(j|k))", win)
		// extglobs-temp.js:1168
		assertMatch(t, true, "fffooofoooooffoofffooofff", "*(*(f)*(o))", win)
		// extglobs-temp.js:1169
		assertMatch(t, true, "ffo", "*(f*(o))", win)
		// extglobs-temp.js:1170
		assertMatch(t, true, "fofo", "*(f*(o))", win)
		// extglobs-temp.js:1171
		assertMatch(t, true, "foofoofo", "@(foo|f|fo)*(f|of+(o))", win)
		// extglobs-temp.js:1172
		assertMatch(t, true, "fooofoofofooo", "*(f*(o))", win)
		// extglobs-temp.js:1173
		assertMatch(t, true, "foooofo", "*(f*(o))", win)
		// extglobs-temp.js:1174
		assertMatch(t, true, "foooofof", "*(f*(o))", win)
		// extglobs-temp.js:1175
		assertMatch(t, true, "foooxfooxfoxfooox", "*(f*(o)x)", win)
		// extglobs-temp.js:1176
		assertMatch(t, true, "foooxfooxfxfooox", "*(f*(o)x)", win)
		// extglobs-temp.js:1177
		assertMatch(t, true, "ofoofo", "*(of+(o))", win)
		// extglobs-temp.js:1178
		assertMatch(t, true, "ofoofo", "*(of+(o)|f)", win)
		// extglobs-temp.js:1179
		assertMatch(t, true, "ofoooxoofxo", "*(*(of*(o)x)o)", win)
		// extglobs-temp.js:1180
		assertMatch(t, true, "ofoooxoofxoofoooxoofxo", "*(*(of*(o)x)o)", win)
		// extglobs-temp.js:1181
		assertMatch(t, true, "ofoooxoofxoofoooxoofxoo", "*(*(of*(o)x)o)", win)
		// extglobs-temp.js:1182
		assertMatch(t, true, "ofoooxoofxoofoooxoofxooofxofxo", "*(*(of*(o)x)o)", win)
		// extglobs-temp.js:1183
		assertMatch(t, true, "ofxoofxo", "*(*(of*(o)x)o)", win)
		// extglobs-temp.js:1184
		assertMatch(t, true, "oofooofo", "*(of|oof+(o))", win)
		// extglobs-temp.js:1185
		assertMatch(t, true, "oxfoxoxfox", "*(oxf+(ox))", win)
	})

	t.Run("other - should support exclusions final", func(t *testing.T) {
		// extglobs-temp.js:1189
		assertMatch(t, false, "f", "!(f)", win)
		// extglobs-temp.js:1190
		assertMatch(t, false, "f", "*(!(f))", win)
		// extglobs-temp.js:1191
		assertMatch(t, false, "f", "+(!(f))", win)
		// extglobs-temp.js:1192
		assertMatch(t, false, "foo", "!(foo)", win)
		// extglobs-temp.js:1193
		assertMatch(t, false, "foob", "!(foo)b*", win)
		// extglobs-temp.js:1194
		assertMatch(t, false, "mad.moo.cow", "!(*.*).!(*.*)", win)
		// extglobs-temp.js:1195
		assertMatch(t, false, "mucca.pazza", "mu!(*(c))?.pa!(*(z))?", win)
		// extglobs-temp.js:1196
		assertMatch(t, false, "zoot", "@(!(z*)|*x)", win)
		// extglobs-temp.js:1197
		assertMatch(t, true, "fff", "!(f)", win)
		// extglobs-temp.js:1198
		assertMatch(t, true, "fff", "*(!(f))", win)
		// extglobs-temp.js:1199
		assertMatch(t, true, "fff", "+(!(f))", win)
		// extglobs-temp.js:1200
		assertMatch(t, true, "foo", "!(f)", win)
		// extglobs-temp.js:1201
		assertMatch(t, true, "foo", "!(x)", win)
		// extglobs-temp.js:1202
		assertMatch(t, true, "foo", "!(x)*", win)
		// extglobs-temp.js:1203
		assertMatch(t, true, "foo", "*(!(f))", win)
		// extglobs-temp.js:1204
		assertMatch(t, true, "foo", "+(!(f))", win)
		// extglobs-temp.js:1205
		assertMatch(t, true, "foobar", "!(foo)", win)
		// extglobs-temp.js:1206
		assertMatch(t, true, "foot", "@(!(z*)|*x)", win)
		// extglobs-temp.js:1207
		assertMatch(t, true, "foox", "@(!(z*)|*x)", win)
		// extglobs-temp.js:1208
		assertMatch(t, true, "ooo", "!(f)", win)
		// extglobs-temp.js:1209
		assertMatch(t, true, "ooo", "*(!(f))", win)
		// extglobs-temp.js:1210
		assertMatch(t, true, "ooo", "+(!(f))", win)
		// extglobs-temp.js:1211
		assertMatch(t, true, "zoox", "@(!(z*)|*x)", win)
	})
}
