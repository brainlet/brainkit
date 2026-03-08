// Ported from: packages/ai/src/util/notify.test.ts
package util

import (
	"errors"
	"reflect"
	"testing"
)

func TestNotify_SingleCallback(t *testing.T) {
	var calls []string
	Notify("hello", func(event string) error {
		calls = append(calls, event)
		return nil
	})
	expected := []string{"hello"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("expected %v, got %v", expected, calls)
	}
}

func TestNotify_MultipleCallbacks(t *testing.T) {
	var calls []string
	Notify("hello",
		func(event string) error {
			calls = append(calls, "first: "+event)
			return nil
		},
		func(event string) error {
			calls = append(calls, "second: "+event)
			return nil
		},
	)
	expected := []string{"first: hello", "second: hello"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("expected %v, got %v", expected, calls)
	}
}

func TestNotify_NoCallbacks(t *testing.T) {
	// Should not panic
	Notify[string]("hello")
}

func TestNotify_ErrorsSwallowed(t *testing.T) {
	var calls []string
	Notify("test",
		func(event string) error {
			calls = append(calls, "before throw")
			return errors.New("callback error")
		},
	)
	// Should continue without panic
	calls = append(calls, "after notify")
	expected := []string{"before throw", "after notify"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("expected %v, got %v", expected, calls)
	}
}

func TestNotify_ErrorsContinueToNext(t *testing.T) {
	var calls []string
	Notify("test",
		func(_ string) error {
			calls = append(calls, "first before throw")
			return errors.New("first error")
		},
		func(_ string) error {
			calls = append(calls, "second runs")
			return nil
		},
	)
	expected := []string{"first before throw", "second runs"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("expected %v, got %v", expected, calls)
	}
}

func TestNotify_RepeatedCalls(t *testing.T) {
	var events []string
	cb := func(event string) error {
		events = append(events, event)
		return nil
	}

	Notify("first", cb)
	Notify("second", cb)
	Notify("third", cb)

	expected := []string{"first", "second", "third"}
	if !reflect.DeepEqual(events, expected) {
		t.Fatalf("expected %v, got %v", expected, events)
	}
}
