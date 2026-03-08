package graymatter

import (
	"testing"
)

func TestTest(t *testing.T) {
	t.Run("should return true if the string has front-matter", func(t *testing.T) {
		if !Test("---\nabc: xyz\n---") {
			t.Error("expected true")
		}

		if Test("---\nabc: xyz\n---", Options{Delimiters: "~~~"}) {
			t.Error("expected false for non-matching delimiters")
		}

		if !Test("~~~\nabc: xyz\n~~~", Options{Delimiters: "~~~"}) {
			t.Error("expected true for matching delimiters")
		}

		if Test("\nabc: xyz\n---") {
			t.Error("expected false for string starting without delimiter")
		}
	})
}
