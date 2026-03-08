// Ported from: packages/provider-utils/src/delayed-promise.test.ts
package providerutils

import (
	"errors"
	"testing"
	"time"
)

func TestDelayedPromise_ResolveAfterCreation(t *testing.T) {
	dp := NewDelayedPromise[string]()
	dp.Resolve("success")
	val, err := dp.Await()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "success" {
		t.Errorf("expected 'success', got '%s'", val)
	}
}

func TestDelayedPromise_RejectAfterCreation(t *testing.T) {
	dp := NewDelayedPromise[string]()
	dp.Reject(errors.New("failure"))
	_, err := dp.Await()
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "failure" {
		t.Errorf("expected 'failure', got '%s'", err.Error())
	}
}

func TestDelayedPromise_ResolveBeforeAccess(t *testing.T) {
	dp := NewDelayedPromise[string]()
	go func() {
		time.Sleep(10 * time.Millisecond)
		dp.Resolve("delayed-success")
	}()
	val, err := dp.Await()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "delayed-success" {
		t.Errorf("expected 'delayed-success', got '%s'", val)
	}
}

func TestDelayedPromise_RejectBeforeAccess(t *testing.T) {
	dp := NewDelayedPromise[string]()
	go func() {
		time.Sleep(10 * time.Millisecond)
		dp.Reject(errors.New("delayed-failure"))
	}()
	_, err := dp.Await()
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "delayed-failure" {
		t.Errorf("expected 'delayed-failure', got '%s'", err.Error())
	}
}

func TestDelayedPromise_MultipleAccess(t *testing.T) {
	dp := NewDelayedPromise[string]()
	dp.Resolve("success")

	val1, err := dp.Await()
	if err != nil {
		t.Fatalf("first access error: %v", err)
	}
	val2, err := dp.Await()
	if err != nil {
		t.Fatalf("second access error: %v", err)
	}
	if val1 != "success" || val2 != "success" {
		t.Errorf("expected 'success' for both, got '%s' and '%s'", val1, val2)
	}
}
