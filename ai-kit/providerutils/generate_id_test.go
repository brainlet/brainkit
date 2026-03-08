// Ported from: packages/provider-utils/src/generate-id.test.ts
package providerutils

import "testing"

func TestCreateIdGenerator_CorrectLength(t *testing.T) {
	size := 10
	gen := CreateIdGenerator(&CreateIdGeneratorOptions{Size: &size})
	id := gen()
	if len(id) != 10 {
		t.Errorf("expected length 10, got %d", len(id))
	}
}

func TestCreateIdGenerator_DefaultLength(t *testing.T) {
	gen := CreateIdGenerator(nil)
	id := gen()
	if len(id) != 16 {
		t.Errorf("expected length 16, got %d", len(id))
	}
}

func TestCreateIdGenerator_SeparatorInAlphabet(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when separator is part of alphabet")
		}
	}()
	sep := "a"
	prefix := "b"
	CreateIdGenerator(&CreateIdGeneratorOptions{Separator: &sep, Prefix: &prefix})
}

func TestGenerateId_UniqueIDs(t *testing.T) {
	id1 := GenerateId()
	id2 := GenerateId()
	if id1 == id2 {
		t.Errorf("expected unique IDs, got %s and %s", id1, id2)
	}
}
