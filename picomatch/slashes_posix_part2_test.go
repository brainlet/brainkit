package picomatch

// slashes_posix_part2_test.go — Ported from picomatch/test/slashes-posix.js lines 601-1211
// Tests for POSIX slash handling: globstars, wildcards, trailing slashes, escaped stars,
// file paths, leading dots, leading slashes, and double slashes.

import (
	"strings"
	"testing"
)

func TestSlashesPosixPart2(t *testing.T) {
	t.Run("should match one or more directories with a globstar", func(t *testing.T) {
		assertMatch(t, false, "a/", "**/a")                // slashes-posix.js:607
		assertMatch(t, false, "/a/", "**/a")               // slashes-posix.js:608
		assertMatch(t, false, "a/a/", "**/a")              // slashes-posix.js:609
		assertMatch(t, false, "/a/a/", "**/a")             // slashes-posix.js:610
		assertMatch(t, false, "a/a/a/", "**/a")            // slashes-posix.js:611

		assertMatch(t, true, "a", "**/a")                  // slashes-posix.js:613
		assertMatch(t, true, "a/a", "**/a")                // slashes-posix.js:614
		assertMatch(t, true, "a/a/a", "**/a")              // slashes-posix.js:615
		assertMatch(t, true, "/a", "**/a")                 // slashes-posix.js:616
		assertMatch(t, true, "a/a/", "**/a/*{,/}")         // slashes-posix.js:617
		assertMatch(t, false, "a/a/", "**/a/*", &Options{StrictSlashes: true}) // slashes-posix.js:618
		assertMatch(t, true, "/a/a", "**/a")               // slashes-posix.js:619

		assertMatch(t, true, "a", "a/**")                  // slashes-posix.js:621
		assertMatch(t, false, "/a", "a/**")                // slashes-posix.js:622
		assertMatch(t, false, "/a/", "a/**")               // slashes-posix.js:623
		assertMatch(t, false, "/a/a", "a/**")              // slashes-posix.js:624
		assertMatch(t, false, "/a/a/", "a/**")             // slashes-posix.js:625
		assertMatch(t, true, "/a", "/a/**")                // slashes-posix.js:626
		assertMatch(t, true, "/a/", "/a/**")               // slashes-posix.js:627
		assertMatch(t, true, "/a/a", "/a/**")              // slashes-posix.js:628
		assertMatch(t, true, "/a/a/", "/a/**")             // slashes-posix.js:629
		assertMatch(t, true, "a/", "a/**")                 // slashes-posix.js:630
		assertMatch(t, true, "a/a", "a/**")                // slashes-posix.js:631
		assertMatch(t, true, "a/a/", "a/**")               // slashes-posix.js:632
		assertMatch(t, true, "a/a/a", "a/**")              // slashes-posix.js:633
		assertMatch(t, true, "a/a/a/", "a/**")             // slashes-posix.js:634

		assertMatch(t, true, "a", "**/a/**")               // slashes-posix.js:636
		assertMatch(t, true, "/a", "**/a/**")              // slashes-posix.js:637
		assertMatch(t, true, "/a/", "**/a/**")             // slashes-posix.js:638
		assertMatch(t, true, "/a/a", "**/a/**")            // slashes-posix.js:639
		assertMatch(t, true, "/a/a/", "**/a/**")           // slashes-posix.js:640
		assertMatch(t, true, "a/", "**/a/**")              // slashes-posix.js:641
		assertMatch(t, true, "a/a", "**/a/**")             // slashes-posix.js:642
		assertMatch(t, true, "a/a/", "**/a/**")            // slashes-posix.js:643
		assertMatch(t, true, "a/a/a", "**/a/**")           // slashes-posix.js:644
		assertMatch(t, true, "a/a/a/", "**/a/**")          // slashes-posix.js:645
	})

	t.Run("should match one or more characters", func(t *testing.T) {
		// Pattern: *
		assertMatch(t, true, "a", "*")                     // slashes-posix.js:649
		assertMatch(t, true, "aa", "*")                    // slashes-posix.js:650
		assertMatch(t, true, "aaa", "*")                   // slashes-posix.js:651
		assertMatch(t, true, "aaaa", "*")                  // slashes-posix.js:652
		assertMatch(t, true, "ab", "*")                    // slashes-posix.js:653
		assertMatch(t, true, "b", "*")                     // slashes-posix.js:654
		assertMatch(t, true, "bb", "*")                    // slashes-posix.js:655
		assertMatch(t, true, "c", "*")                     // slashes-posix.js:656
		assertMatch(t, true, "cc", "*")                    // slashes-posix.js:657
		assertMatch(t, true, "cac", "*")                   // slashes-posix.js:658
		assertMatch(t, false, "a/a", "*")                  // slashes-posix.js:659
		assertMatch(t, false, "a/b", "*")                  // slashes-posix.js:660
		assertMatch(t, false, "a/c", "*")                  // slashes-posix.js:661
		assertMatch(t, false, "a/x", "*")                  // slashes-posix.js:662
		assertMatch(t, false, "a/a/a", "*")                // slashes-posix.js:663
		assertMatch(t, false, "a/a/b", "*")                // slashes-posix.js:664
		assertMatch(t, false, "a/a/a/a", "*")              // slashes-posix.js:665
		assertMatch(t, false, "a/a/a/a/a", "*")            // slashes-posix.js:666
		assertMatch(t, false, "x/y", "*")                  // slashes-posix.js:667
		assertMatch(t, false, "z/z", "*")                  // slashes-posix.js:668

		// Pattern: a*
		assertMatch(t, true, "a", "a*")                    // slashes-posix.js:670
		assertMatch(t, true, "aa", "a*")                   // slashes-posix.js:671
		assertMatch(t, true, "aaa", "a*")                  // slashes-posix.js:672
		assertMatch(t, true, "aaaa", "a*")                 // slashes-posix.js:673
		assertMatch(t, true, "ab", "a*")                   // slashes-posix.js:674
		assertMatch(t, false, "b", "a*")                   // slashes-posix.js:675
		assertMatch(t, false, "bb", "a*")                  // slashes-posix.js:676
		assertMatch(t, false, "c", "a*")                   // slashes-posix.js:677
		assertMatch(t, false, "cc", "a*")                  // slashes-posix.js:678
		assertMatch(t, false, "cac", "a*")                 // slashes-posix.js:679
		assertMatch(t, false, "a/a", "a*")                 // slashes-posix.js:680
		assertMatch(t, false, "a/b", "a*")                 // slashes-posix.js:681
		assertMatch(t, false, "a/c", "a*")                 // slashes-posix.js:682
		assertMatch(t, false, "a/x", "a*")                 // slashes-posix.js:683
		assertMatch(t, false, "a/a/a", "a*")               // slashes-posix.js:684
		assertMatch(t, false, "a/a/b", "a*")               // slashes-posix.js:685
		assertMatch(t, false, "a/a/a/a", "a*")             // slashes-posix.js:686
		assertMatch(t, false, "a/a/a/a/a", "a*")           // slashes-posix.js:687
		assertMatch(t, false, "x/y", "a*")                 // slashes-posix.js:688
		assertMatch(t, false, "z/z", "a*")                 // slashes-posix.js:689

		// Pattern: *b
		assertMatch(t, false, "a", "*b")                   // slashes-posix.js:691
		assertMatch(t, false, "aa", "*b")                  // slashes-posix.js:692
		assertMatch(t, false, "aaa", "*b")                 // slashes-posix.js:693
		assertMatch(t, false, "aaaa", "*b")                // slashes-posix.js:694
		assertMatch(t, true, "ab", "*b")                   // slashes-posix.js:695
		assertMatch(t, true, "b", "*b")                    // slashes-posix.js:696
		assertMatch(t, true, "bb", "*b")                   // slashes-posix.js:697
		assertMatch(t, false, "c", "*b")                   // slashes-posix.js:698
		assertMatch(t, false, "cc", "*b")                  // slashes-posix.js:699
		assertMatch(t, false, "cac", "*b")                 // slashes-posix.js:700
		assertMatch(t, false, "a/a", "*b")                 // slashes-posix.js:701
		assertMatch(t, false, "a/b", "*b")                 // slashes-posix.js:702
		assertMatch(t, false, "a/c", "*b")                 // slashes-posix.js:703
		assertMatch(t, false, "a/x", "*b")                 // slashes-posix.js:704
		assertMatch(t, false, "a/a/a", "*b")               // slashes-posix.js:705
		assertMatch(t, false, "a/a/b", "*b")               // slashes-posix.js:706
		assertMatch(t, false, "a/a/a/a", "*b")             // slashes-posix.js:707
		assertMatch(t, false, "a/a/a/a/a", "*b")           // slashes-posix.js:708
		assertMatch(t, false, "x/y", "*b")                 // slashes-posix.js:709
		assertMatch(t, false, "z/z", "*b")                 // slashes-posix.js:710
	})

	t.Run("should match one or zero characters", func(t *testing.T) {
		// Pattern: *
		assertMatch(t, true, "a", "*")                     // slashes-posix.js:714
		assertMatch(t, true, "aa", "*")                    // slashes-posix.js:715
		assertMatch(t, true, "aaa", "*")                   // slashes-posix.js:716
		assertMatch(t, true, "aaaa", "*")                  // slashes-posix.js:717
		assertMatch(t, true, "ab", "*")                    // slashes-posix.js:718
		assertMatch(t, true, "b", "*")                     // slashes-posix.js:719
		assertMatch(t, true, "bb", "*")                    // slashes-posix.js:720
		assertMatch(t, true, "c", "*")                     // slashes-posix.js:721
		assertMatch(t, true, "cc", "*")                    // slashes-posix.js:722
		assertMatch(t, true, "cac", "*")                   // slashes-posix.js:723
		assertMatch(t, false, "a/a", "*")                  // slashes-posix.js:724
		assertMatch(t, false, "a/b", "*")                  // slashes-posix.js:725
		assertMatch(t, false, "a/c", "*")                  // slashes-posix.js:726
		assertMatch(t, false, "a/x", "*")                  // slashes-posix.js:727
		assertMatch(t, false, "a/a/a", "*")                // slashes-posix.js:728
		assertMatch(t, false, "a/a/b", "*")                // slashes-posix.js:729
		assertMatch(t, false, "a/a/a/a", "*")              // slashes-posix.js:730
		assertMatch(t, false, "a/a/a/a/a", "*")            // slashes-posix.js:731
		assertMatch(t, false, "x/y", "*")                  // slashes-posix.js:732
		assertMatch(t, false, "z/z", "*")                  // slashes-posix.js:733

		// Pattern: *a*
		assertMatch(t, true, "a", "*a*")                   // slashes-posix.js:735
		assertMatch(t, true, "aa", "*a*")                  // slashes-posix.js:736
		assertMatch(t, true, "aaa", "*a*")                 // slashes-posix.js:737
		assertMatch(t, true, "aaaa", "*a*")                // slashes-posix.js:738
		assertMatch(t, true, "ab", "*a*")                  // slashes-posix.js:739
		assertMatch(t, false, "b", "*a*")                  // slashes-posix.js:740
		assertMatch(t, false, "bb", "*a*")                 // slashes-posix.js:741
		assertMatch(t, false, "c", "*a*")                  // slashes-posix.js:742
		assertMatch(t, false, "cc", "*a*")                 // slashes-posix.js:743
		assertMatch(t, true, "cac", "*a*")                 // slashes-posix.js:744
		assertMatch(t, false, "a/a", "*a*")                // slashes-posix.js:745
		assertMatch(t, false, "a/b", "*a*")                // slashes-posix.js:746
		assertMatch(t, false, "a/c", "*a*")                // slashes-posix.js:747
		assertMatch(t, false, "a/x", "*a*")                // slashes-posix.js:748
		assertMatch(t, false, "a/a/a", "*a*")              // slashes-posix.js:749
		assertMatch(t, false, "a/a/b", "*a*")              // slashes-posix.js:750
		assertMatch(t, false, "a/a/a/a", "*a*")            // slashes-posix.js:751
		assertMatch(t, false, "a/a/a/a/a", "*a*")          // slashes-posix.js:752
		assertMatch(t, false, "x/y", "*a*")                // slashes-posix.js:753
		assertMatch(t, false, "z/z", "*a*")                // slashes-posix.js:754

		// Pattern: *b*
		assertMatch(t, false, "a", "*b*")                  // slashes-posix.js:756
		assertMatch(t, false, "aa", "*b*")                 // slashes-posix.js:757
		assertMatch(t, false, "aaa", "*b*")                // slashes-posix.js:758
		assertMatch(t, false, "aaaa", "*b*")               // slashes-posix.js:759
		assertMatch(t, true, "ab", "*b*")                  // slashes-posix.js:760
		assertMatch(t, true, "b", "*b*")                   // slashes-posix.js:761
		assertMatch(t, true, "bb", "*b*")                  // slashes-posix.js:762
		assertMatch(t, false, "c", "*b*")                  // slashes-posix.js:763
		assertMatch(t, false, "cc", "*b*")                 // slashes-posix.js:764
		assertMatch(t, false, "cac", "*b*")                // slashes-posix.js:765
		assertMatch(t, false, "a/a", "*b*")                // slashes-posix.js:766
		assertMatch(t, false, "a/b", "*b*")                // slashes-posix.js:767
		assertMatch(t, false, "a/c", "*b*")                // slashes-posix.js:768
		assertMatch(t, false, "a/x", "*b*")                // slashes-posix.js:769
		assertMatch(t, false, "a/a/a", "*b*")              // slashes-posix.js:770
		assertMatch(t, false, "a/a/b", "*b*")              // slashes-posix.js:771
		assertMatch(t, false, "a/a/a/a", "*b*")            // slashes-posix.js:772
		assertMatch(t, false, "a/a/a/a/a", "*b*")          // slashes-posix.js:773
		assertMatch(t, false, "x/y", "*b*")                // slashes-posix.js:774
		assertMatch(t, false, "z/z", "*b*")                // slashes-posix.js:775

		// Pattern: *c*
		assertMatch(t, false, "a", "*c*")                  // slashes-posix.js:777
		assertMatch(t, false, "aa", "*c*")                 // slashes-posix.js:778
		assertMatch(t, false, "aaa", "*c*")                // slashes-posix.js:779
		assertMatch(t, false, "aaaa", "*c*")               // slashes-posix.js:780
		assertMatch(t, false, "ab", "*c*")                 // slashes-posix.js:781
		assertMatch(t, false, "b", "*c*")                  // slashes-posix.js:782
		assertMatch(t, false, "bb", "*c*")                 // slashes-posix.js:783
		assertMatch(t, true, "c", "*c*")                   // slashes-posix.js:784
		assertMatch(t, true, "cc", "*c*")                  // slashes-posix.js:785
		assertMatch(t, true, "cac", "*c*")                 // slashes-posix.js:786
		assertMatch(t, false, "a/a", "*c*")                // slashes-posix.js:787
		assertMatch(t, false, "a/b", "*c*")                // slashes-posix.js:788
		assertMatch(t, false, "a/c", "*c*")                // slashes-posix.js:789
		assertMatch(t, false, "a/x", "*c*")                // slashes-posix.js:790
		assertMatch(t, false, "a/a/a", "*c*")              // slashes-posix.js:791
		assertMatch(t, false, "a/a/b", "*c*")              // slashes-posix.js:792
		assertMatch(t, false, "a/a/a/a", "*c*")            // slashes-posix.js:793
		assertMatch(t, false, "a/a/a/a/a", "*c*")          // slashes-posix.js:794
		assertMatch(t, false, "x/y", "*c*")                // slashes-posix.js:795
		assertMatch(t, false, "z/z", "*c*")                // slashes-posix.js:796
	})

	t.Run("should respect trailing slashes on patterns", func(t *testing.T) {
		// Pattern: */
		assertMatch(t, false, "a", "*/")                   // slashes-posix.js:800
		assertMatch(t, true, "a/", "*/")                   // slashes-posix.js:801
		assertMatch(t, false, "b", "*/")                   // slashes-posix.js:802
		assertMatch(t, true, "b/", "*/")                   // slashes-posix.js:803
		assertMatch(t, false, "a/a", "*/")                 // slashes-posix.js:804
		assertMatch(t, false, "a/a/", "*/")                // slashes-posix.js:805
		assertMatch(t, false, "a/b", "*/")                 // slashes-posix.js:806
		assertMatch(t, false, "a/b/", "*/")                // slashes-posix.js:807
		assertMatch(t, false, "a/c", "*/")                 // slashes-posix.js:808
		assertMatch(t, false, "a/c/", "*/")                // slashes-posix.js:809
		assertMatch(t, false, "a/x", "*/")                 // slashes-posix.js:810
		assertMatch(t, false, "a/x/", "*/")                // slashes-posix.js:811
		assertMatch(t, false, "a/a/a", "*/")               // slashes-posix.js:812
		assertMatch(t, false, "a/a/b", "*/")               // slashes-posix.js:813
		assertMatch(t, false, "a/a/b/", "*/")              // slashes-posix.js:814
		assertMatch(t, false, "a/a/a/", "*/")              // slashes-posix.js:815
		assertMatch(t, false, "a/a/a/a", "*/")             // slashes-posix.js:816
		assertMatch(t, false, "a/a/a/a/", "*/")            // slashes-posix.js:817
		assertMatch(t, false, "a/a/a/a/a", "*/")           // slashes-posix.js:818
		assertMatch(t, false, "a/a/a/a/a/", "*/")          // slashes-posix.js:819
		assertMatch(t, false, "x/y", "*/")                 // slashes-posix.js:820
		assertMatch(t, false, "z/z", "*/")                 // slashes-posix.js:821
		assertMatch(t, false, "x/y/", "*/")                // slashes-posix.js:822
		assertMatch(t, false, "z/z/", "*/")                // slashes-posix.js:823
		assertMatch(t, false, "a/b/c/.d/e/", "*/")         // slashes-posix.js:824

		// Pattern: */*/
		assertMatch(t, false, "a", "*/*/")                 // slashes-posix.js:826
		assertMatch(t, false, "a/", "*/*/")                // slashes-posix.js:827
		assertMatch(t, false, "b", "*/*/")                 // slashes-posix.js:828
		assertMatch(t, false, "b/", "*/*/")                // slashes-posix.js:829
		assertMatch(t, false, "a/a", "*/*/")               // slashes-posix.js:830
		assertMatch(t, true, "a/a/", "*/*/")               // slashes-posix.js:831
		assertMatch(t, false, "a/b", "*/*/")               // slashes-posix.js:832
		assertMatch(t, true, "a/b/", "*/*/")               // slashes-posix.js:833
		assertMatch(t, false, "a/c", "*/*/")               // slashes-posix.js:834
		assertMatch(t, true, "a/c/", "*/*/")               // slashes-posix.js:835
		assertMatch(t, false, "a/x", "*/*/")               // slashes-posix.js:836
		assertMatch(t, true, "a/x/", "*/*/")               // slashes-posix.js:837
		assertMatch(t, false, "a/a/a", "*/*/")             // slashes-posix.js:838
		assertMatch(t, false, "a/a/b", "*/*/")             // slashes-posix.js:839
		assertMatch(t, false, "a/a/b/", "*/*/")            // slashes-posix.js:840
		assertMatch(t, false, "a/a/a/", "*/*/")            // slashes-posix.js:841
		assertMatch(t, false, "a/a/a/a", "*/*/")           // slashes-posix.js:842
		assertMatch(t, false, "a/a/a/a/", "*/*/")          // slashes-posix.js:843
		assertMatch(t, false, "a/a/a/a/a", "*/*/")         // slashes-posix.js:844
		assertMatch(t, false, "a/a/a/a/a/", "*/*/")        // slashes-posix.js:845
		assertMatch(t, false, "x/y", "*/*/")               // slashes-posix.js:846
		assertMatch(t, false, "z/z", "*/*/")               // slashes-posix.js:847
		assertMatch(t, true, "x/y/", "*/*/")               // slashes-posix.js:848
		assertMatch(t, true, "z/z/", "*/*/")               // slashes-posix.js:849
		assertMatch(t, false, "a/b/c/.d/e/", "*/*/")       // slashes-posix.js:850

		// Pattern: */*/*/
		assertMatch(t, false, "a", "*/*/*/")               // slashes-posix.js:852
		assertMatch(t, false, "a/", "*/*/*/")              // slashes-posix.js:853
		assertMatch(t, false, "b", "*/*/*/")               // slashes-posix.js:854
		assertMatch(t, false, "b/", "*/*/*/")              // slashes-posix.js:855
		assertMatch(t, false, "a/a", "*/*/*/")             // slashes-posix.js:856
		assertMatch(t, false, "a/a/", "*/*/*/")            // slashes-posix.js:857
		assertMatch(t, false, "a/b", "*/*/*/")             // slashes-posix.js:858
		assertMatch(t, false, "a/b/", "*/*/*/")            // slashes-posix.js:859
		assertMatch(t, false, "a/c", "*/*/*/")             // slashes-posix.js:860
		assertMatch(t, false, "a/c/", "*/*/*/")            // slashes-posix.js:861
		assertMatch(t, false, "a/x", "*/*/*/")             // slashes-posix.js:862
		assertMatch(t, false, "a/x/", "*/*/*/")            // slashes-posix.js:863
		assertMatch(t, false, "a/a/a", "*/*/*/")           // slashes-posix.js:864
		assertMatch(t, false, "a/a/b", "*/*/*/")           // slashes-posix.js:865
		assertMatch(t, true, "a/a/b/", "*/*/*/")           // slashes-posix.js:866
		assertMatch(t, true, "a/a/a/", "*/*/*/")           // slashes-posix.js:867
		assertMatch(t, false, "a/a/a/a", "*/*/*/")         // slashes-posix.js:868
		assertMatch(t, false, "a/a/a/a/", "*/*/*/")        // slashes-posix.js:869
		assertMatch(t, false, "a/a/a/a/a", "*/*/*/")       // slashes-posix.js:870
		assertMatch(t, false, "a/a/a/a/a/", "*/*/*/")      // slashes-posix.js:871
		assertMatch(t, false, "x/y", "*/*/*/")             // slashes-posix.js:872
		assertMatch(t, false, "z/z", "*/*/*/")             // slashes-posix.js:873
		assertMatch(t, false, "x/y/", "*/*/*/")            // slashes-posix.js:874
		assertMatch(t, false, "z/z/", "*/*/*/")            // slashes-posix.js:875
		assertMatch(t, false, "a/b/c/.d/e/", "*/*/*/")     // slashes-posix.js:876

		// Pattern: */*/*/*/
		assertMatch(t, false, "a", "*/*/*/*/")             // slashes-posix.js:878
		assertMatch(t, false, "a/", "*/*/*/*/")            // slashes-posix.js:879
		assertMatch(t, false, "b", "*/*/*/*/")             // slashes-posix.js:880
		assertMatch(t, false, "b/", "*/*/*/*/")            // slashes-posix.js:881
		assertMatch(t, false, "a/a", "*/*/*/*/")           // slashes-posix.js:882
		assertMatch(t, false, "a/a/", "*/*/*/*/")          // slashes-posix.js:883
		assertMatch(t, false, "a/b", "*/*/*/*/")           // slashes-posix.js:884
		assertMatch(t, false, "a/b/", "*/*/*/*/")          // slashes-posix.js:885
		assertMatch(t, false, "a/c", "*/*/*/*/")           // slashes-posix.js:886
		assertMatch(t, false, "a/c/", "*/*/*/*/")          // slashes-posix.js:887
		assertMatch(t, false, "a/x", "*/*/*/*/")           // slashes-posix.js:888
		assertMatch(t, false, "a/x/", "*/*/*/*/")          // slashes-posix.js:889
		assertMatch(t, false, "a/a/a", "*/*/*/*/")         // slashes-posix.js:890
		assertMatch(t, false, "a/a/b", "*/*/*/*/")         // slashes-posix.js:891
		assertMatch(t, false, "a/a/b/", "*/*/*/*/")        // slashes-posix.js:892
		assertMatch(t, false, "a/a/a/", "*/*/*/*/")        // slashes-posix.js:893
		assertMatch(t, false, "a/a/a/a", "*/*/*/*/")       // slashes-posix.js:894
		assertMatch(t, true, "a/a/a/a/", "*/*/*/*/")       // slashes-posix.js:895
		assertMatch(t, false, "a/a/a/a/a", "*/*/*/*/")     // slashes-posix.js:896
		assertMatch(t, false, "a/a/a/a/a/", "*/*/*/*/")    // slashes-posix.js:897
		assertMatch(t, false, "x/y", "*/*/*/*/")           // slashes-posix.js:898
		assertMatch(t, false, "z/z", "*/*/*/*/")           // slashes-posix.js:899
		assertMatch(t, false, "x/y/", "*/*/*/*/")          // slashes-posix.js:900
		assertMatch(t, false, "z/z/", "*/*/*/*/")          // slashes-posix.js:901
		assertMatch(t, false, "a/b/c/.d/e/", "*/*/*/*/")   // slashes-posix.js:902

		// Pattern: */*/*/*/*/
		assertMatch(t, false, "a", "*/*/*/*/*/")           // slashes-posix.js:904
		assertMatch(t, false, "a/", "*/*/*/*/*/")          // slashes-posix.js:905
		assertMatch(t, false, "b", "*/*/*/*/*/")           // slashes-posix.js:906
		assertMatch(t, false, "b/", "*/*/*/*/*/")          // slashes-posix.js:907
		assertMatch(t, false, "a/a", "*/*/*/*/*/")         // slashes-posix.js:908
		assertMatch(t, false, "a/a/", "*/*/*/*/*/")        // slashes-posix.js:909
		assertMatch(t, false, "a/b", "*/*/*/*/*/")         // slashes-posix.js:910
		assertMatch(t, false, "a/b/", "*/*/*/*/*/")        // slashes-posix.js:911
		assertMatch(t, false, "a/c", "*/*/*/*/*/")         // slashes-posix.js:912
		assertMatch(t, false, "a/c/", "*/*/*/*/*/")        // slashes-posix.js:913
		assertMatch(t, false, "a/x", "*/*/*/*/*/")         // slashes-posix.js:914
		assertMatch(t, false, "a/x/", "*/*/*/*/*/")        // slashes-posix.js:915
		assertMatch(t, false, "a/a/a", "*/*/*/*/*/")       // slashes-posix.js:916
		assertMatch(t, false, "a/a/b", "*/*/*/*/*/")       // slashes-posix.js:917
		assertMatch(t, false, "a/a/b/", "*/*/*/*/*/")      // slashes-posix.js:918
		assertMatch(t, false, "a/a/a/", "*/*/*/*/*/")      // slashes-posix.js:919
		assertMatch(t, false, "a/a/a/a", "*/*/*/*/*/")     // slashes-posix.js:920
		assertMatch(t, false, "a/a/a/a/", "*/*/*/*/*/")    // slashes-posix.js:921
		assertMatch(t, false, "a/a/a/a/a", "*/*/*/*/*/")   // slashes-posix.js:922
		assertMatch(t, true, "a/a/a/a/a/", "*/*/*/*/*/")   // slashes-posix.js:923
		assertMatch(t, false, "x/y", "*/*/*/*/*/")         // slashes-posix.js:924
		assertMatch(t, false, "z/z", "*/*/*/*/*/")         // slashes-posix.js:925
		assertMatch(t, false, "x/y/", "*/*/*/*/*/")        // slashes-posix.js:926
		assertMatch(t, false, "z/z/", "*/*/*/*/*/")        // slashes-posix.js:927
		assertMatch(t, false, "a/b/c/.d/e/", "*/*/*/*/*/") // slashes-posix.js:928

		// Pattern: a/*/
		assertMatch(t, false, "a", "a/*/")                 // slashes-posix.js:930
		assertMatch(t, false, "a/", "a/*/")                // slashes-posix.js:931
		assertMatch(t, false, "b", "a/*/")                 // slashes-posix.js:932
		assertMatch(t, false, "b/", "a/*/")                // slashes-posix.js:933
		assertMatch(t, false, "a/a", "a/*/")               // slashes-posix.js:934
		assertMatch(t, true, "a/a/", "a/*/")               // slashes-posix.js:935
		assertMatch(t, false, "a/b", "a/*/")               // slashes-posix.js:936
		assertMatch(t, true, "a/b/", "a/*/")               // slashes-posix.js:937
		assertMatch(t, false, "a/c", "a/*/")               // slashes-posix.js:938
		assertMatch(t, true, "a/c/", "a/*/")               // slashes-posix.js:939
		assertMatch(t, false, "a/x", "a/*/")               // slashes-posix.js:940
		assertMatch(t, true, "a/x/", "a/*/")               // slashes-posix.js:941
		assertMatch(t, false, "a/a/a", "a/*/")             // slashes-posix.js:942
		assertMatch(t, false, "a/a/b", "a/*/")             // slashes-posix.js:943
		assertMatch(t, false, "a/a/b/", "a/*/")            // slashes-posix.js:944
		assertMatch(t, false, "a/a/a/", "a/*/")            // slashes-posix.js:945
		assertMatch(t, false, "a/a/a/a", "a/*/")           // slashes-posix.js:946
		assertMatch(t, false, "a/a/a/a/", "a/*/")          // slashes-posix.js:947
		assertMatch(t, false, "a/a/a/a/a", "a/*/")         // slashes-posix.js:948
		assertMatch(t, false, "a/a/a/a/a/", "a/*/")        // slashes-posix.js:949
		assertMatch(t, false, "x/y", "a/*/")               // slashes-posix.js:950
		assertMatch(t, false, "z/z", "a/*/")               // slashes-posix.js:951
		assertMatch(t, false, "x/y/", "a/*/")              // slashes-posix.js:952
		assertMatch(t, false, "z/z/", "a/*/")              // slashes-posix.js:953
		assertMatch(t, false, "a/b/c/.d/e/", "a/*/")       // slashes-posix.js:954

		// Pattern: a/*/*/
		assertMatch(t, false, "a", "a/*/*/")               // slashes-posix.js:956
		assertMatch(t, false, "a/", "a/*/*/")              // slashes-posix.js:957
		assertMatch(t, false, "b", "a/*/*/")               // slashes-posix.js:958
		assertMatch(t, false, "b/", "a/*/*/")              // slashes-posix.js:959
		assertMatch(t, false, "a/a", "a/*/*/")             // slashes-posix.js:960
		assertMatch(t, false, "a/a/", "a/*/*/")            // slashes-posix.js:961
		assertMatch(t, false, "a/b", "a/*/*/")             // slashes-posix.js:962
		assertMatch(t, false, "a/b/", "a/*/*/")            // slashes-posix.js:963
		assertMatch(t, false, "a/c", "a/*/*/")             // slashes-posix.js:964
		assertMatch(t, false, "a/c/", "a/*/*/")            // slashes-posix.js:965
		assertMatch(t, false, "a/x", "a/*/*/")             // slashes-posix.js:966
		assertMatch(t, false, "a/x/", "a/*/*/")            // slashes-posix.js:967
		assertMatch(t, false, "a/a/a", "a/*/*/")           // slashes-posix.js:968
		assertMatch(t, false, "a/a/b", "a/*/*/")           // slashes-posix.js:969
		assertMatch(t, true, "a/a/b/", "a/*/*/")           // slashes-posix.js:970
		assertMatch(t, true, "a/a/a/", "a/*/*/")           // slashes-posix.js:971
		assertMatch(t, false, "a/a/a/a", "a/*/*/")         // slashes-posix.js:972
		assertMatch(t, false, "a/a/a/a/", "a/*/*/")        // slashes-posix.js:973
		assertMatch(t, false, "a/a/a/a/a", "a/*/*/")       // slashes-posix.js:974
		assertMatch(t, false, "a/a/a/a/a/", "a/*/*/")      // slashes-posix.js:975
		assertMatch(t, false, "x/y", "a/*/*/")             // slashes-posix.js:976
		assertMatch(t, false, "z/z", "a/*/*/")             // slashes-posix.js:977
		assertMatch(t, false, "x/y/", "a/*/*/")            // slashes-posix.js:978
		assertMatch(t, false, "z/z/", "a/*/*/")            // slashes-posix.js:979
		assertMatch(t, false, "a/b/c/.d/e/", "a/*/*/")     // slashes-posix.js:980

		// Pattern: a/*/*/*/
		assertMatch(t, false, "a", "a/*/*/*/")             // slashes-posix.js:982
		assertMatch(t, false, "a/", "a/*/*/*/")            // slashes-posix.js:983
		assertMatch(t, false, "b", "a/*/*/*/")             // slashes-posix.js:984
		assertMatch(t, false, "b/", "a/*/*/*/")            // slashes-posix.js:985
		assertMatch(t, false, "a/a", "a/*/*/*/")           // slashes-posix.js:986
		assertMatch(t, false, "a/a/", "a/*/*/*/")          // slashes-posix.js:987
		assertMatch(t, false, "a/b", "a/*/*/*/")           // slashes-posix.js:988
		assertMatch(t, false, "a/b/", "a/*/*/*/")          // slashes-posix.js:989
		assertMatch(t, false, "a/c", "a/*/*/*/")           // slashes-posix.js:990
		assertMatch(t, false, "a/c/", "a/*/*/*/")          // slashes-posix.js:991
		assertMatch(t, false, "a/x", "a/*/*/*/")           // slashes-posix.js:992
		assertMatch(t, false, "a/x/", "a/*/*/*/")          // slashes-posix.js:993
		assertMatch(t, false, "a/a/a", "a/*/*/*/")         // slashes-posix.js:994
		assertMatch(t, false, "a/a/b", "a/*/*/*/")         // slashes-posix.js:995
		assertMatch(t, false, "a/a/b/", "a/*/*/*/")        // slashes-posix.js:996
		assertMatch(t, false, "a/a/a/", "a/*/*/*/")        // slashes-posix.js:997
		assertMatch(t, false, "a/a/a/a", "a/*/*/*/")       // slashes-posix.js:998
		assertMatch(t, true, "a/a/a/a/", "a/*/*/*/")       // slashes-posix.js:999
		assertMatch(t, false, "a/a/a/a/a", "a/*/*/*/")     // slashes-posix.js:1000
		assertMatch(t, false, "a/a/a/a/a/", "a/*/*/*/")    // slashes-posix.js:1001
		assertMatch(t, false, "x/y", "a/*/*/*/")           // slashes-posix.js:1002
		assertMatch(t, false, "z/z", "a/*/*/*/")           // slashes-posix.js:1003
		assertMatch(t, false, "x/y/", "a/*/*/*/")          // slashes-posix.js:1004
		assertMatch(t, false, "z/z/", "a/*/*/*/")          // slashes-posix.js:1005
		assertMatch(t, false, "a/b/c/.d/e/", "a/*/*/*/")   // slashes-posix.js:1006

		// Pattern: a/*/*/*/*/
		assertMatch(t, false, "a", "a/*/*/*/*/")           // slashes-posix.js:1008
		assertMatch(t, false, "a/", "a/*/*/*/*/")          // slashes-posix.js:1009
		assertMatch(t, false, "b", "a/*/*/*/*/")           // slashes-posix.js:1010
		assertMatch(t, false, "b/", "a/*/*/*/*/")          // slashes-posix.js:1011
		assertMatch(t, false, "a/a", "a/*/*/*/*/")         // slashes-posix.js:1012
		assertMatch(t, false, "a/a/", "a/*/*/*/*/")        // slashes-posix.js:1013
		assertMatch(t, false, "a/b", "a/*/*/*/*/")         // slashes-posix.js:1014
		assertMatch(t, false, "a/b/", "a/*/*/*/*/")        // slashes-posix.js:1015
		assertMatch(t, false, "a/c", "a/*/*/*/*/")         // slashes-posix.js:1016
		assertMatch(t, false, "a/c/", "a/*/*/*/*/")        // slashes-posix.js:1017
		assertMatch(t, false, "a/x", "a/*/*/*/*/")         // slashes-posix.js:1018
		assertMatch(t, false, "a/x/", "a/*/*/*/*/")        // slashes-posix.js:1019
		assertMatch(t, false, "a/a/a", "a/*/*/*/*/")       // slashes-posix.js:1020
		assertMatch(t, false, "a/a/b", "a/*/*/*/*/")       // slashes-posix.js:1021
		assertMatch(t, false, "a/a/b/", "a/*/*/*/*/")      // slashes-posix.js:1022
		assertMatch(t, false, "a/a/a/", "a/*/*/*/*/")      // slashes-posix.js:1023
		assertMatch(t, false, "a/a/a/a", "a/*/*/*/*/")     // slashes-posix.js:1024
		assertMatch(t, false, "a/a/a/a/", "a/*/*/*/*/")    // slashes-posix.js:1025
		assertMatch(t, false, "a/a/a/a/a", "a/*/*/*/*/")   // slashes-posix.js:1026
		assertMatch(t, true, "a/a/a/a/a/", "a/*/*/*/*/")   // slashes-posix.js:1027
		assertMatch(t, false, "x/y", "a/*/*/*/*/")         // slashes-posix.js:1028
		assertMatch(t, false, "z/z", "a/*/*/*/*/")         // slashes-posix.js:1029
		assertMatch(t, false, "x/y/", "a/*/*/*/*/")        // slashes-posix.js:1030
		assertMatch(t, false, "z/z/", "a/*/*/*/*/")        // slashes-posix.js:1031
		assertMatch(t, false, "a/b/c/.d/e/", "a/*/*/*/*/") // slashes-posix.js:1032

		// Pattern: a/*/a/
		assertMatch(t, false, "a", "a/*/a/")               // slashes-posix.js:1034
		assertMatch(t, false, "a/", "a/*/a/")              // slashes-posix.js:1035
		assertMatch(t, false, "b", "a/*/a/")               // slashes-posix.js:1036
		assertMatch(t, false, "b/", "a/*/a/")              // slashes-posix.js:1037
		assertMatch(t, false, "a/a", "a/*/a/")             // slashes-posix.js:1038
		assertMatch(t, false, "a/a/", "a/*/a/")            // slashes-posix.js:1039
		assertMatch(t, false, "a/b", "a/*/a/")             // slashes-posix.js:1040
		assertMatch(t, false, "a/b/", "a/*/a/")            // slashes-posix.js:1041
		assertMatch(t, false, "a/c", "a/*/a/")             // slashes-posix.js:1042
		assertMatch(t, false, "a/c/", "a/*/a/")            // slashes-posix.js:1043
		assertMatch(t, false, "a/x", "a/*/a/")             // slashes-posix.js:1044
		assertMatch(t, false, "a/x/", "a/*/a/")            // slashes-posix.js:1045
		assertMatch(t, false, "a/a/a", "a/*/a/")           // slashes-posix.js:1046
		assertMatch(t, false, "a/a/b", "a/*/a/")           // slashes-posix.js:1047
		assertMatch(t, false, "a/a/b/", "a/*/a/")          // slashes-posix.js:1048
		assertMatch(t, true, "a/a/a/", "a/*/a/")           // slashes-posix.js:1049
		assertMatch(t, false, "a/a/a/a", "a/*/a/")         // slashes-posix.js:1050
		assertMatch(t, false, "a/a/a/a/", "a/*/a/")        // slashes-posix.js:1051
		assertMatch(t, false, "a/a/a/a/a", "a/*/a/")       // slashes-posix.js:1052
		assertMatch(t, false, "a/a/a/a/a/", "a/*/a/")      // slashes-posix.js:1053
		assertMatch(t, false, "x/y", "a/*/a/")             // slashes-posix.js:1054
		assertMatch(t, false, "z/z", "a/*/a/")             // slashes-posix.js:1055
		assertMatch(t, false, "x/y/", "a/*/a/")            // slashes-posix.js:1056
		assertMatch(t, false, "z/z/", "a/*/a/")            // slashes-posix.js:1057
		assertMatch(t, false, "a/b/c/.d/e/", "a/*/a/")     // slashes-posix.js:1058

		// Pattern: a/*/b/
		assertMatch(t, false, "a", "a/*/b/")               // slashes-posix.js:1060
		assertMatch(t, false, "a/", "a/*/b/")              // slashes-posix.js:1061
		assertMatch(t, false, "b", "a/*/b/")               // slashes-posix.js:1062
		assertMatch(t, false, "b/", "a/*/b/")              // slashes-posix.js:1063
		assertMatch(t, false, "a/a", "a/*/b/")             // slashes-posix.js:1064
		assertMatch(t, false, "a/a/", "a/*/b/")            // slashes-posix.js:1065
		assertMatch(t, false, "a/b", "a/*/b/")             // slashes-posix.js:1066
		assertMatch(t, false, "a/b/", "a/*/b/")            // slashes-posix.js:1067
		assertMatch(t, false, "a/c", "a/*/b/")             // slashes-posix.js:1068
		assertMatch(t, false, "a/c/", "a/*/b/")            // slashes-posix.js:1069
		assertMatch(t, false, "a/x", "a/*/b/")             // slashes-posix.js:1070
		assertMatch(t, false, "a/x/", "a/*/b/")            // slashes-posix.js:1071
		assertMatch(t, false, "a/a/a", "a/*/b/")           // slashes-posix.js:1072
		assertMatch(t, false, "a/a/b", "a/*/b/")           // slashes-posix.js:1073
		assertMatch(t, true, "a/a/b/", "a/*/b/")           // slashes-posix.js:1074
		assertMatch(t, false, "a/a/a/", "a/*/b/")          // slashes-posix.js:1075
		assertMatch(t, false, "a/a/a/a", "a/*/b/")         // slashes-posix.js:1076
		assertMatch(t, false, "a/a/a/a/", "a/*/b/")        // slashes-posix.js:1077
		assertMatch(t, false, "a/a/a/a/a", "a/*/b/")       // slashes-posix.js:1078
		assertMatch(t, false, "a/a/a/a/a/", "a/*/b/")      // slashes-posix.js:1079
		assertMatch(t, false, "x/y", "a/*/b/")             // slashes-posix.js:1080
		assertMatch(t, false, "z/z", "a/*/b/")             // slashes-posix.js:1081
		assertMatch(t, false, "x/y/", "a/*/b/")            // slashes-posix.js:1082
		assertMatch(t, false, "z/z/", "a/*/b/")            // slashes-posix.js:1083
		assertMatch(t, false, "a/b/c/.d/e/", "a/*/b/")     // slashes-posix.js:1084
	})

	t.Run("should match a literal star when escaped", func(t *testing.T) {
		// Pattern: \*
		assertMatch(t, false, ".md", "\\*")                // slashes-posix.js:1088
		assertMatch(t, false, "a**a.md", "\\*")            // slashes-posix.js:1089
		assertMatch(t, false, "**a.md", "\\*")             // slashes-posix.js:1090
		assertMatch(t, false, "**/a.md", "\\*")            // slashes-posix.js:1091
		assertMatch(t, false, "**.md", "\\*")              // slashes-posix.js:1092
		assertMatch(t, false, ".md", "\\*")                // slashes-posix.js:1093
		assertMatch(t, true, "*", "\\*")                   // slashes-posix.js:1094
		assertMatch(t, false, "**", "\\*")                 // slashes-posix.js:1095
		assertMatch(t, false, "*.md", "\\*")               // slashes-posix.js:1096

		// Pattern: \*.md
		assertMatch(t, false, ".md", "\\*.md")             // slashes-posix.js:1098
		assertMatch(t, false, "a**a.md", "\\*.md")         // slashes-posix.js:1099
		assertMatch(t, false, "**a.md", "\\*.md")          // slashes-posix.js:1100
		assertMatch(t, false, "**/a.md", "\\*.md")         // slashes-posix.js:1101
		assertMatch(t, false, "**.md", "\\*.md")           // slashes-posix.js:1102
		assertMatch(t, false, ".md", "\\*.md")             // slashes-posix.js:1103
		assertMatch(t, false, "*", "\\*.md")               // slashes-posix.js:1104
		assertMatch(t, false, "**", "\\*.md")              // slashes-posix.js:1105
		assertMatch(t, true, "*.md", "\\*.md")             // slashes-posix.js:1106

		// Pattern: \**.md
		assertMatch(t, false, ".md", "\\**.md")            // slashes-posix.js:1108
		assertMatch(t, false, "a**a.md", "\\**.md")        // slashes-posix.js:1109
		assertMatch(t, true, "**a.md", "\\**.md")          // slashes-posix.js:1110
		assertMatch(t, false, "**/a.md", "\\**.md")        // slashes-posix.js:1111
		assertMatch(t, true, "**.md", "\\**.md")           // slashes-posix.js:1112
		assertMatch(t, false, ".md", "\\**.md")            // slashes-posix.js:1113
		assertMatch(t, false, "*", "\\**.md")              // slashes-posix.js:1114
		assertMatch(t, false, "**", "\\**.md")             // slashes-posix.js:1115
		assertMatch(t, true, "*.md", "\\**.md")            // slashes-posix.js:1116

		// Pattern: a\**.md
		assertMatch(t, false, ".md", "a\\**.md")           // slashes-posix.js:1118
		assertMatch(t, true, "a**a.md", "a\\**.md")        // slashes-posix.js:1119
		assertMatch(t, false, "**a.md", "a\\**.md")        // slashes-posix.js:1120
		assertMatch(t, false, "**/a.md", "a\\**.md")       // slashes-posix.js:1121
		assertMatch(t, false, "**.md", "a\\**.md")         // slashes-posix.js:1122
		assertMatch(t, false, ".md", "a\\**.md")           // slashes-posix.js:1123
		assertMatch(t, false, "*", "a\\**.md")             // slashes-posix.js:1124
		assertMatch(t, false, "**", "a\\**.md")            // slashes-posix.js:1125
		assertMatch(t, false, "*.md", "a\\**.md")          // slashes-posix.js:1126
	})

	t.Run("should match file paths", func(t *testing.T) {
		assertMatch(t, false, "a/.b", "a/**/z/*.md")                        // slashes-posix.js:1130
		assertMatch(t, false, "a/b/c/j/e/z/c.txt", "a/**/j/**/z/*.md")     // slashes-posix.js:1131
		assertMatch(t, false, "a/b/z/.a", "a/**/z/*.a")                     // slashes-posix.js:1132
		assertMatch(t, false, "a/b/z/.a", "a/*/z/*.a")                      // slashes-posix.js:1133
		assertMatch(t, false, "foo.txt", "*/*.txt")                          // slashes-posix.js:1134
		assertMatch(t, true, "a/.b", "a/.*")                                // slashes-posix.js:1135
		assertMatch(t, true, "a/b/c/d/e/j/n/p/o/z/c.md", "a/**/j/**/z/*.md") // slashes-posix.js:1136
		assertMatch(t, true, "a/b/c/d/e/z/c.md", "a/**/z/*.md")             // slashes-posix.js:1137
		assertMatch(t, true, "a/b/c/xyz.md", "a/b/c/*.md")                  // slashes-posix.js:1138
		assertMatch(t, true, "a/b/z/.a", "a/*/z/.a")                        // slashes-posix.js:1139
		assertMatch(t, true, "a/bb.bb/aa/b.b/aa/c/xyz.md", "a/**/c/*.md")   // slashes-posix.js:1140
		assertMatch(t, true, "a/bb.bb/aa/bb/aa/c/xyz.md", "a/**/c/*.md")    // slashes-posix.js:1141
		assertMatch(t, true, "a/bb.bb/c/xyz.md", "a/*/c/*.md")              // slashes-posix.js:1142
		assertMatch(t, true, "a/bb/c/xyz.md", "a/*/c/*.md")                 // slashes-posix.js:1143
		assertMatch(t, true, "a/bbbb/c/xyz.md", "a/*/c/*.md")               // slashes-posix.js:1144
		assertMatch(t, true, "foo.txt", "**/foo.txt")                        // slashes-posix.js:1145
		assertMatch(t, true, "foo/bar.txt", "**/*.txt")                      // slashes-posix.js:1146
		assertMatch(t, true, "foo/bar/baz.txt", "**/*.txt")                  // slashes-posix.js:1147
	})

	t.Run("should match paths with leading ./ when pattern has ./", func(t *testing.T) {
		// JS: const format = str => str.replace(/^\.\//, '');
		// slashes-posix.js:1151
		format := func(s string) string {
			return strings.TrimPrefix(s, "./")
		}
		assertMatch(t, false, "./a/b/c/d/e/z/c.md", "./a/**/j/**/z/*.md", &Options{Format: format})         // slashes-posix.js:1152
		assertMatch(t, false, "./a/b/c/j/e/z/c.txt", "./a/**/j/**/z/*.md", &Options{Format: format})        // slashes-posix.js:1153
		assertMatch(t, true, "./a/b/c/d/e/j/n/p/o/z/c.md", "./a/**/j/**/z/*.md", &Options{Format: format})  // slashes-posix.js:1154
		assertMatch(t, true, "./a/b/c/d/e/z/c.md", "./a/**/z/*.md", &Options{Format: format})                // slashes-posix.js:1155
		assertMatch(t, true, "./a/b/c/j/e/z/c.md", "./a/**/j/**/z/*.md", &Options{Format: format})          // slashes-posix.js:1156
		assertMatch(t, true, "./a/b/z/.a", "./a/**/z/.a", &Options{Format: format})                          // slashes-posix.js:1157
	})

	t.Run("should match leading slashes", func(t *testing.T) {
		assertMatch(t, false, "ef", "/*")                                    // slashes-posix.js:1161
		assertMatch(t, true, "/ef", "/*")                                    // slashes-posix.js:1162
		assertMatch(t, true, "/foo/bar.txt", "/foo/*")                       // slashes-posix.js:1163
		assertMatch(t, true, "/foo/bar.txt", "/foo/**")                      // slashes-posix.js:1164
		assertMatch(t, true, "/foo/bar.txt", "/foo/**/**/*.txt")             // slashes-posix.js:1165
		assertMatch(t, true, "/foo/bar.txt", "/foo/**/**/bar.txt")           // slashes-posix.js:1166
		assertMatch(t, true, "/foo/bar.txt", "/foo/**/*.txt")                // slashes-posix.js:1167
		assertMatch(t, true, "/foo/bar.txt", "/foo/**/bar.txt")              // slashes-posix.js:1168
		assertMatch(t, false, "/foo/bar.txt", "/foo/*/bar.txt")              // slashes-posix.js:1169
		assertMatch(t, false, "/foo/bar/baz.txt", "/foo/*")                  // slashes-posix.js:1170
		assertMatch(t, true, "/foo/bar/baz.txt", "/foo/**")                  // slashes-posix.js:1171
		assertMatch(t, true, "/foo/bar/baz.txt", "/foo/**")                  // slashes-posix.js:1172
		assertMatch(t, true, "/foo/bar/baz.txt", "/foo/**/*.txt")            // slashes-posix.js:1173
		assertMatch(t, true, "/foo/bar/baz.txt", "/foo/**/*/*.txt")          // slashes-posix.js:1174
		assertMatch(t, true, "/foo/bar/baz.txt", "/foo/**/*/baz.txt")        // slashes-posix.js:1175
		assertMatch(t, false, "/foo/bar/baz.txt", "/foo/*.txt")              // slashes-posix.js:1176
		assertMatch(t, true, "/foo/bar/baz.txt", "/foo/*/*.txt")             // slashes-posix.js:1177
		assertMatch(t, false, "/foo/bar/baz.txt", "/foo/*/*/baz.txt")        // slashes-posix.js:1178
		assertMatch(t, false, "/foo/bar/baz.txt", "/foo/bar**")              // slashes-posix.js:1179
		assertMatch(t, true, "/foo/bar/baz/qux.txt", "**/*.txt")            // slashes-posix.js:1180
		assertMatch(t, false, "/foo/bar/baz/qux.txt", "**/.txt")            // slashes-posix.js:1181
		assertMatch(t, false, "/foo/bar/baz/qux.txt", "*/*.txt")            // slashes-posix.js:1182
		assertMatch(t, false, "/foo/bar/baz/qux.txt", "/foo/**.txt")         // slashes-posix.js:1183
		assertMatch(t, true, "/foo/bar/baz/qux.txt", "/foo/**/*.txt")       // slashes-posix.js:1184
		assertMatch(t, false, "/foo/bar/baz/qux.txt", "/foo/*/*.txt")       // slashes-posix.js:1185
		assertMatch(t, false, "/foo/bar/baz/qux.txt", "/foo/bar**/*.txt")   // slashes-posix.js:1186
		assertMatch(t, false, "/.txt", "*.txt")                              // slashes-posix.js:1187
		assertMatch(t, false, "/.txt", "/*.txt")                             // slashes-posix.js:1188
		assertMatch(t, false, "/.txt", "*/*.txt")                            // slashes-posix.js:1189
		assertMatch(t, false, "/.txt", "**/*.txt")                           // slashes-posix.js:1190
		assertMatch(t, false, "/.txt", "*.txt", &Options{Dot: true})         // slashes-posix.js:1191
		assertMatch(t, true, "/.txt", "/*.txt", &Options{Dot: true})         // slashes-posix.js:1192
		assertMatch(t, true, "/.txt", "*/*.txt", &Options{Dot: true})        // slashes-posix.js:1193
		assertMatch(t, true, "/.txt", "**/*.txt", &Options{Dot: true})       // slashes-posix.js:1194
	})

	t.Run("should match double slashes", func(t *testing.T) {
		assertMatch(t, false, "https://foo.com/bar/baz/app.min.js", "https://foo.com/*")                                           // slashes-posix.js:1198
		assertMatch(t, false, "https://foo.com/bar/baz/app.min.js", "https://foo.com/*")                                           // slashes-posix.js:1199
		assertMatch(t, true, "https://foo.com/bar/baz/app.min.js", "https://foo.com/**")                                           // slashes-posix.js:1200
		assertMatch(t, false, "https://foo.com/bar/baz/app.min.js", "https://foo.com/**", &Options{Noglobstar: true})               // slashes-posix.js:1201
		assertMatch(t, true, "https://foo.com/bar/baz/app.min.js", "https://foo.com/**")                                           // slashes-posix.js:1202
		assertMatch(t, true, "https://foo.com/bar/baz/app.min.js", "https://foo.com/**/app.min.js")                                // slashes-posix.js:1203
		assertMatch(t, true, "https://foo.com/bar/baz/app.min.js", "https://foo.com/*/*/app.min.js")                               // slashes-posix.js:1204
		assertMatch(t, true, "https://foo.com/bar/baz/app.min.js", "https://foo.com/*/*/app.min.js", &Options{Noglobstar: true})    // slashes-posix.js:1205
		assertMatch(t, false, "https://foo.com/bar/baz/app.min.js", "https://foo.com/*/app.min.js")                                // slashes-posix.js:1206
		assertMatch(t, false, "https://foo.com/bar/baz/app.min.js", "https://foo.com/*/app.min.js")                                // slashes-posix.js:1207
		assertMatch(t, true, "https://foo.com/bar/baz/app.min.js", "https://foo.com/**/app.min.js")                                // slashes-posix.js:1208
		assertMatch(t, false, "https://foo.com/bar/baz/app.min.js", "https://foo.com/**/app.min.js", &Options{Noglobstar: true})    // slashes-posix.js:1209
	})
}
