// Ported from: packages/ai/src/util/prepare-headers.test.ts
package util

import (
	"net/http"
	"testing"
)

func TestPrepareHeaders_SetDefaultIfNotPresent(t *testing.T) {
	headers := PrepareHeaders(http.Header{}, map[string]string{
		"Content-Type": "application/json",
	})
	if got := headers.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected application/json, got %s", got)
	}
}

func TestPrepareHeaders_DontOverwriteExisting(t *testing.T) {
	existing := http.Header{}
	existing.Set("Content-Type", "text/html")
	headers := PrepareHeaders(existing, map[string]string{
		"Content-Type": "application/json",
	})
	if got := headers.Get("Content-Type"); got != "text/html" {
		t.Fatalf("expected text/html, got %s", got)
	}
}

func TestPrepareHeaders_NilInit(t *testing.T) {
	headers := PrepareHeaders(nil, map[string]string{
		"Content-Type": "application/json",
	})
	if got := headers.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected application/json, got %s", got)
	}
}

func TestPrepareHeaders_MultipleHeaders(t *testing.T) {
	existing := http.Header{}
	existing.Set("Init", "foo")
	headers := PrepareHeaders(existing, map[string]string{
		"Content-Type": "application/json",
	})
	if got := headers.Get("Init"); got != "foo" {
		t.Fatalf("expected foo, got %s", got)
	}
	if got := headers.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected application/json, got %s", got)
	}
}
