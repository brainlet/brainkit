// Ported from: packages/provider-utils/src/handle-fetch-error.test.ts
package providerutils

import (
	"context"
	"errors"
	"testing"
)

func TestHandleFetchError_NilError(t *testing.T) {
	err := HandleFetchError(HandleFetchErrorOptions{
		Error: nil,
		URL:   "https://example.com",
	})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestHandleFetchError_AbortError(t *testing.T) {
	err := HandleFetchError(HandleFetchErrorOptions{
		Error: context.Canceled,
		URL:   "https://example.com",
	})
	if err != context.Canceled {
		t.Errorf("expected context.Canceled to be passed through, got %v", err)
	}
}

func TestHandleFetchError_DeadlineExceeded(t *testing.T) {
	err := HandleFetchError(HandleFetchErrorOptions{
		Error: context.DeadlineExceeded,
		URL:   "https://example.com",
	})
	if err != context.DeadlineExceeded {
		t.Errorf("expected context.DeadlineExceeded to be passed through, got %v", err)
	}
}

func TestHandleFetchError_GenericError(t *testing.T) {
	origErr := errors.New("some error")
	err := HandleFetchError(HandleFetchErrorOptions{
		Error: origErr,
		URL:   "https://example.com",
	})
	if err != origErr {
		t.Errorf("expected original error to be passed through, got %v", err)
	}
}
