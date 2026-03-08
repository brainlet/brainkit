// Ported from: packages/provider-utils/src/resolve.test.ts
package providerutils

import "testing"

func TestResolveValue_RawValue(t *testing.T) {
	r := NewResolvableValue(42)
	result := ResolveValue(r)
	if result != 42 {
		t.Errorf("expected 42, got %v", result)
	}
}

func TestResolveValue_Function(t *testing.T) {
	r := NewResolvableFunc(func() (int, error) { return 42, nil })
	result := ResolveValue(r)
	if result != 42 {
		t.Errorf("expected 42, got %v", result)
	}
}

func TestResolveValue_StringValue(t *testing.T) {
	r := NewResolvableValue("hello")
	result := ResolveValue(r)
	if result != "hello" {
		t.Errorf("expected 'hello', got %v", result)
	}
}

func TestResolveValue_FunctionReturningObject(t *testing.T) {
	type obj struct{ Foo string }
	r := NewResolvableFunc(func() (obj, error) { return obj{Foo: "bar"}, nil })
	result := ResolveValue(r)
	if result.Foo != "bar" {
		t.Errorf("expected Foo='bar', got %v", result.Foo)
	}
}

func TestResolveValue_FunctionCalledEachTime(t *testing.T) {
	counter := 0
	r := NewResolvableFunc(func() (int, error) {
		counter++
		return counter, nil
	})
	if ResolveValue(r) != 1 {
		t.Error("expected 1")
	}
	if ResolveValue(r) != 2 {
		t.Error("expected 2")
	}
	if ResolveValue(r) != 3 {
		t.Error("expected 3")
	}
}
