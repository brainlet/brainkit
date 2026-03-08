// Ported from: packages/ai/src/util/get-potential-start-index.test.ts
package util

import "testing"

func TestGetPotentialStartIndex_EmptySearchedText(t *testing.T) {
	_, found := GetPotentialStartIndex("1234567890", "")
	if found {
		t.Fatal("expected not found for empty searchedText")
	}
}

func TestGetPotentialStartIndex_NotInText(t *testing.T) {
	_, found := GetPotentialStartIndex("1234567890", "a")
	if found {
		t.Fatal("expected not found")
	}
}

func TestGetPotentialStartIndex_FullMatch(t *testing.T) {
	idx, found := GetPotentialStartIndex("1234567890", "1234567890")
	if !found || idx != 0 {
		t.Fatalf("expected 0, got %d (found=%v)", idx, found)
	}
}

func TestGetPotentialStartIndex_PartialAtEnd1(t *testing.T) {
	idx, found := GetPotentialStartIndex("1234567890", "0123")
	if !found || idx != 9 {
		t.Fatalf("expected 9, got %d (found=%v)", idx, found)
	}
}

func TestGetPotentialStartIndex_PartialAtEnd2(t *testing.T) {
	idx, found := GetPotentialStartIndex("1234567890", "90123")
	if !found || idx != 8 {
		t.Fatalf("expected 8, got %d (found=%v)", idx, found)
	}
}

func TestGetPotentialStartIndex_PartialAtEnd3(t *testing.T) {
	idx, found := GetPotentialStartIndex("1234567890", "890123")
	if !found || idx != 7 {
		t.Fatalf("expected 7, got %d (found=%v)", idx, found)
	}
}
