package graymatter

import "testing"

func dataMap(t *testing.T, value any) map[string]any {
	t.Helper()

	m, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", value)
	}
	return m
}

func jsFunction(t *testing.T, value any) JSFunction {
	t.Helper()

	fn, ok := value.(JSFunction)
	if !ok {
		t.Fatalf("expected JSFunction, got %T", value)
	}
	return fn
}
