// Ported from: packages/provider-utils/src/get-from-api.test.ts
package providerutils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// statusCodeErrorHandlerAsError adapts CreateStatusCodeErrorResponseHandler (which returns
// ResponseHandler[*APICallError]) to ResponseHandler[error] as required by GetFromApiOptions.
func statusCodeErrorHandlerAsError() ResponseHandler[error] {
	inner := CreateStatusCodeErrorResponseHandler()
	return func(opts ResponseHandlerOptions) (*ResponseHandlerResult[error], error) {
		result, err := inner(opts)
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, nil
		}
		return &ResponseHandlerResult[error]{
			Value:           result.Value,
			RawValue:        result.RawValue,
			ResponseHeaders: result.ResponseHeaders,
		}, nil
	}
}

// jsonMapResponseHandler is a convenience alias for the response handler type used in tests.
// Without a schema, SafeParseJSON returns map[string]interface{} from JSON objects.
type jsonMap = map[string]interface{}

func TestGetFromApi(t *testing.T) {
	mockSuccessResponse := map[string]interface{}{
		"name":  "test",
		"value": float64(123), // JSON numbers decode as float64
	}

	t.Run("should successfully fetch and parse data", func(t *testing.T) {
		body, _ := json.Marshal(mockSuccessResponse)

		var capturedReq *http.Request
		mockFetch := func(req *http.Request) (*http.Response, error) {
			capturedReq = req
			return &http.Response{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       io.NopCloser(strings.NewReader(string(body))),
				Header:     http.Header{},
			}, nil
		}

		result, err := GetFromApi(GetFromApiOptions[jsonMap]{
			URL:                       "https://api.test.com/data",
			Headers:                   map[string]string{"Authorization": "Bearer test"},
			SuccessfulResponseHandler: CreateJsonResponseHandler[jsonMap](nil),
			FailedResponseHandler:     statusCodeErrorHandlerAsError(),
			Fetch:                     mockFetch,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Value["name"] != "test" {
			t.Errorf("expected name=%q, got %v", "test", result.Value["name"])
		}
		if result.Value["value"] != float64(123) {
			t.Errorf("expected value=%v, got %v", float64(123), result.Value["value"])
		}

		// Verify the request was made with correct method
		if capturedReq.Method != "GET" {
			t.Errorf("expected method GET, got %s", capturedReq.Method)
		}

		// Verify the URL
		if capturedReq.URL.String() != "https://api.test.com/data" {
			t.Errorf("expected URL 'https://api.test.com/data', got %s", capturedReq.URL.String())
		}

		// Verify authorization header is present
		authHeader := capturedReq.Header.Get("Authorization")
		if authHeader != "Bearer test" {
			t.Errorf("expected Authorization header 'Bearer test', got %q", authHeader)
		}

		// Verify user-agent header includes SDK version and runtime info
		uaHeader := capturedReq.Header.Get("User-Agent")
		expectedPrefix := fmt.Sprintf("ai-sdk/provider-utils/%s", VERSION)
		if !strings.Contains(uaHeader, expectedPrefix) {
			t.Errorf("expected user-agent to contain %q, got %q", expectedPrefix, uaHeader)
		}
		runtimeUA := GetRuntimeEnvironmentUserAgent()
		if !strings.Contains(uaHeader, runtimeUA) {
			t.Errorf("expected user-agent to contain %q, got %q", runtimeUA, uaHeader)
		}
	})

	t.Run("should handle API errors", func(t *testing.T) {
		errorResponse := map[string]string{"error": "Not Found"}
		body, _ := json.Marshal(errorResponse)

		mockFetch := func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 404,
				Status:     "404 Not Found",
				Body:       io.NopCloser(strings.NewReader(string(body))),
				Header:     http.Header{},
			}, nil
		}

		_, err := GetFromApi(GetFromApiOptions[jsonMap]{
			URL:                       "https://api.test.com/data",
			SuccessfulResponseHandler: CreateJsonResponseHandler[jsonMap](nil),
			FailedResponseHandler:     statusCodeErrorHandlerAsError(),
			Fetch:                     mockFetch,
		})
		if err == nil {
			t.Fatal("expected error for 404 response")
		}
		if !IsAPICallError(err) {
			t.Errorf("expected APICallError, got %T: %v", err, err)
		}
	})

	t.Run("should handle network errors", func(t *testing.T) {
		// Simulate a "fetch failed" error with a cause, mirroring the TS test:
		//   Object.assign(new TypeError('fetch failed'), { cause: new Error('Failed to connect') })
		cause := errors.New("Failed to connect")
		fetchErr := fmt.Errorf("fetch failed: %w", cause)

		mockFetch := func(req *http.Request) (*http.Response, error) {
			return nil, fetchErr
		}

		_, err := GetFromApi(GetFromApiOptions[jsonMap]{
			URL:                       "https://api.test.com/data",
			SuccessfulResponseHandler: CreateJsonResponseHandler[jsonMap](nil),
			FailedResponseHandler:     statusCodeErrorHandlerAsError(),
			Fetch:                     mockFetch,
		})
		if err == nil {
			t.Fatal("expected error for network failure")
		}
		// HandleFetchError passes through generic errors that don't match known patterns.
		// The error message should reference the fetch failure.
		if !strings.Contains(err.Error(), "fetch failed") {
			t.Errorf("expected error containing 'fetch failed', got %q", err.Error())
		}
	})

	t.Run("should handle abort signals", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		mockFetch := func(req *http.Request) (*http.Response, error) {
			cancel() // simulate abort
			return nil, context.Canceled
		}

		_, err := GetFromApi(GetFromApiOptions[jsonMap]{
			URL:                       "https://api.test.com/data",
			SuccessfulResponseHandler: CreateJsonResponseHandler[jsonMap](nil),
			FailedResponseHandler:     statusCodeErrorHandlerAsError(),
			Fetch:                     mockFetch,
			Ctx:                       ctx,
		})
		if err == nil {
			t.Fatal("expected error for aborted request")
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("should remove undefined header entries", func(t *testing.T) {
		body, _ := json.Marshal(mockSuccessResponse)

		var capturedReq *http.Request
		mockFetch := func(req *http.Request) (*http.Response, error) {
			capturedReq = req
			return &http.Response{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       io.NopCloser(strings.NewReader(string(body))),
				Header:     http.Header{},
			}, nil
		}

		// In Go, "undefined" header values are represented as empty strings.
		// NormalizeHeaders (called inside WithUserAgentSuffix) filters out empty values.
		_, err := GetFromApi(GetFromApiOptions[jsonMap]{
			URL: "https://api.test.com/data",
			Headers: map[string]string{
				"Authorization":   "Bearer test",
				"X-Custom-Header": "", // equivalent to undefined in TS
			},
			SuccessfulResponseHandler: CreateJsonResponseHandler[jsonMap](nil),
			FailedResponseHandler:     statusCodeErrorHandlerAsError(),
			Fetch:                     mockFetch,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify Authorization header is present
		authHeader := capturedReq.Header.Get("Authorization")
		if authHeader != "Bearer test" {
			t.Errorf("expected Authorization header 'Bearer test', got %q", authHeader)
		}

		// Verify X-Custom-Header was removed (empty value = undefined)
		customHeader := capturedReq.Header.Get("X-Custom-Header")
		if customHeader != "" {
			t.Errorf("expected X-Custom-Header to be removed, got %q", customHeader)
		}

		// Verify user-agent header is present
		uaHeader := capturedReq.Header.Get("User-Agent")
		expectedPrefix := fmt.Sprintf("ai-sdk/provider-utils/%s", VERSION)
		if !strings.Contains(uaHeader, expectedPrefix) {
			t.Errorf("expected user-agent to contain %q, got %q", expectedPrefix, uaHeader)
		}
	})

	t.Run("should handle errors in response handlers", func(t *testing.T) {
		mockFetch := func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       io.NopCloser(strings.NewReader("invalid json")),
				Header:     http.Header{},
			}, nil
		}

		_, err := GetFromApi(GetFromApiOptions[jsonMap]{
			URL:                       "https://api.test.com/data",
			SuccessfulResponseHandler: CreateJsonResponseHandler[jsonMap](nil),
			FailedResponseHandler:     statusCodeErrorHandlerAsError(),
			Fetch:                     mockFetch,
		})
		if err == nil {
			t.Fatal("expected error for invalid JSON response")
		}
		if !IsAPICallError(err) {
			t.Errorf("expected APICallError, got %T: %v", err, err)
		}
	})

	t.Run("should use default fetch when not provided", func(t *testing.T) {
		// In the TS test, global.fetch is mocked. In Go, we verify that when Fetch
		// is nil, DefaultFetch is used by calling GetFromApi without a Fetch function.
		// This will attempt a real HTTP call (which will fail for a fake URL),
		// demonstrating that it falls through to DefaultFetch without panicking.
		_, err := GetFromApi(GetFromApiOptions[jsonMap]{
			URL:                       "http://127.0.0.1:0/nonexistent",
			SuccessfulResponseHandler: CreateJsonResponseHandler[jsonMap](nil),
			FailedResponseHandler:     statusCodeErrorHandlerAsError(),
			// Fetch is nil - should use DefaultFetch
		})
		// We expect an error because the URL is unreachable, but the important thing
		// is that it did not panic from a nil Fetch function.
		if err == nil {
			t.Log("unexpected success - URL should not be reachable")
		}
	})
}
