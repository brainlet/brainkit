package testing

import (
	"testing"
)

func TestExpect_ToBe(t *testing.T) {
	if err := Expect(42).ToBe(42); err != nil {
		t.Fatal(err)
	}
	if err := Expect("hello").ToBe("hello"); err != nil {
		t.Fatal(err)
	}
	if err := Expect(42).ToBe(43); err == nil {
		t.Fatal("expected failure")
	}
}

func TestExpect_ToEqual(t *testing.T) {
	if err := Expect([]int{1, 2, 3}).ToEqual([]int{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
	if err := Expect([]int{1, 2}).ToEqual([]int{1, 3}); err == nil {
		t.Fatal("expected failure")
	}
}

func TestExpect_ToContain(t *testing.T) {
	if err := Expect("hello world").ToContain("world"); err != nil {
		t.Fatal(err)
	}
	if err := Expect("hello").ToContain("xyz"); err == nil {
		t.Fatal("expected failure")
	}
}

func TestExpect_ToMatch(t *testing.T) {
	if err := Expect("hello-123").ToMatch(`\d+`); err != nil {
		t.Fatal(err)
	}
	if err := Expect("hello").ToMatch(`^\d+$`); err == nil {
		t.Fatal("expected failure")
	}
}

func TestExpect_ToBeTruthy(t *testing.T) {
	if err := Expect(true).ToBeTruthy(); err != nil {
		t.Fatal(err)
	}
	if err := Expect("x").ToBeTruthy(); err != nil {
		t.Fatal(err)
	}
	if err := Expect(false).ToBeTruthy(); err == nil {
		t.Fatal("expected failure")
	}
	if err := Expect("").ToBeTruthy(); err == nil {
		t.Fatal("expected failure for empty string")
	}
}

func TestExpect_ToBeFalsy(t *testing.T) {
	if err := Expect(false).ToBeFalsy(); err != nil {
		t.Fatal(err)
	}
	if err := Expect("").ToBeFalsy(); err != nil {
		t.Fatal(err)
	}
	if err := Expect(nil).ToBeFalsy(); err != nil {
		t.Fatal(err)
	}
}

func TestExpect_ToBeDefined(t *testing.T) {
	if err := Expect("something").ToBeDefined(); err != nil {
		t.Fatal(err)
	}
	if err := Expect(nil).ToBeDefined(); err == nil {
		t.Fatal("expected failure for nil")
	}
}

func TestExpect_ToBeGreaterThan(t *testing.T) {
	if err := Expect(10).ToBeGreaterThan(5); err != nil {
		t.Fatal(err)
	}
	if err := Expect(3).ToBeGreaterThan(5); err == nil {
		t.Fatal("expected failure")
	}
}

func TestExpect_ToBeLessThan(t *testing.T) {
	if err := Expect(3).ToBeLessThan(5); err != nil {
		t.Fatal(err)
	}
	if err := Expect(10).ToBeLessThan(5); err == nil {
		t.Fatal("expected failure")
	}
}

func TestExpect_ToHaveLength(t *testing.T) {
	if err := Expect("abc").ToHaveLength(3); err != nil {
		t.Fatal(err)
	}
	if err := Expect([]int{1, 2}).ToHaveLength(2); err != nil {
		t.Fatal(err)
	}
	if err := Expect("ab").ToHaveLength(3); err == nil {
		t.Fatal("expected failure")
	}
}

func TestExpect_Not(t *testing.T) {
	if err := Expect(42).Not().ToBe(43); err != nil {
		t.Fatal(err)
	}
	if err := Expect(42).Not().ToBe(42); err == nil {
		t.Fatal("expected failure for negated match")
	}
	if err := Expect("hello").Not().ToContain("xyz"); err != nil {
		t.Fatal(err)
	}
}
