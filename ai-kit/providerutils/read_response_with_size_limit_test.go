// Ported from: packages/provider-utils/src/read-response-with-size-limit.test.ts
package providerutils

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

// chunkReader wraps a []byte and delivers it in small chunks to simulate streaming.
type chunkReader struct {
	data      []byte
	offset    int
	chunkSize int
}

func (r *chunkReader) Read(p []byte) (int, error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}
	end := r.offset + r.chunkSize
	if end > len(r.data) {
		end = len(r.data)
	}
	n := copy(p, r.data[r.offset:end])
	r.offset += n
	return n, nil
}

func (r *chunkReader) Close() error { return nil }

// createMockResponse builds an *http.Response with an optional body and Content-Length header.
// body==nil means the response has a nil Body (like the TS test's null body case).
func createMockResponse(body []byte, contentLength *string) *http.Response {
	header := http.Header{}
	if contentLength != nil {
		header.Set("Content-Length", *contentLength)
	}

	var bodyReader io.ReadCloser
	if body != nil {
		bodyReader = &chunkReader{data: body, chunkSize: 4}
	}

	return &http.Response{
		Header: header,
		Body:   bodyReader,
	}
}

func strPtr(s string) *string { return &s }

func TestReadResponseWithSizeLimit(t *testing.T) {
	t.Run("should read response within limit successfully", func(t *testing.T) {
		data := []byte{1, 2, 3, 4, 5, 6, 7, 8}
		resp := createMockResponse(data, strPtr("8"))

		result, err := ReadResponseWithSizeLimit(ReadResponseWithSizeLimitOptions{
			Response: resp,
			URL:      "http://example.com/file",
			MaxBytes: 100,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(result, data) {
			t.Errorf("got %v, want %v", result, data)
		}
	})

	t.Run("should reject when Content-Length exceeds limit (early check)", func(t *testing.T) {
		body := make([]byte, 10)
		resp := createMockResponse(body, strPtr("1000"))

		_, err := ReadResponseWithSizeLimit(ReadResponseWithSizeLimitOptions{
			Response: resp,
			URL:      "http://example.com/large",
			MaxBytes: 100,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !IsDownloadError(err) {
			t.Fatalf("expected DownloadError, got %T: %v", err, err)
		}
		if !containsSubstring(err.Error(), "Content-Length: 1000") {
			t.Errorf("expected error message to contain %q, got %q", "Content-Length: 1000", err.Error())
		}
	})

	t.Run("should abort when streamed bytes exceed limit", func(t *testing.T) {
		// Body is larger than maxBytes, but Content-Length is not set
		largeBody := make([]byte, 200)
		for i := range largeBody {
			largeBody[i] = 42
		}

		resp := createMockResponse(largeBody, nil)

		_, err := ReadResponseWithSizeLimit(ReadResponseWithSizeLimitOptions{
			Response: resp,
			URL:      "http://example.com/streaming",
			MaxBytes: 50,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !IsDownloadError(err) {
			t.Fatalf("expected DownloadError, got %T: %v", err, err)
		}
		if !containsSubstring(err.Error(), "exceeded maximum size of 50 bytes") {
			t.Errorf("expected error message to contain %q, got %q", "exceeded maximum size of 50 bytes", err.Error())
		}
	})

	t.Run("should handle lying Content-Length (says small, sends large)", func(t *testing.T) {
		largeBody := make([]byte, 200)
		for i := range largeBody {
			largeBody[i] = 42
		}

		resp := createMockResponse(largeBody, strPtr("10")) // Claims to be small

		_, err := ReadResponseWithSizeLimit(ReadResponseWithSizeLimitOptions{
			Response: resp,
			URL:      "http://example.com/liar",
			MaxBytes: 50,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !IsDownloadError(err) {
			t.Fatalf("expected DownloadError, got %T: %v", err, err)
		}
		if !containsSubstring(err.Error(), "exceeded maximum size of 50 bytes") {
			t.Errorf("expected error message to contain %q, got %q", "exceeded maximum size of 50 bytes", err.Error())
		}
	})

	t.Run("should handle empty body (null)", func(t *testing.T) {
		resp := createMockResponse(nil, nil)

		result, err := ReadResponseWithSizeLimit(ReadResponseWithSizeLimitOptions{
			Response: resp,
			URL:      "http://example.com/empty",
			MaxBytes: 100,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty result, got %v", result)
		}
	})

	t.Run("should handle empty body (zero-length)", func(t *testing.T) {
		resp := createMockResponse([]byte{}, nil)

		result, err := ReadResponseWithSizeLimit(ReadResponseWithSizeLimitOptions{
			Response: resp,
			URL:      "http://example.com/empty",
			MaxBytes: 100,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty result, got %v", result)
		}
	})

	t.Run("should respect custom maxBytes", func(t *testing.T) {
		data := make([]byte, 10)
		for i := range data {
			data[i] = 1
		}

		resp := createMockResponse(data, strPtr("10"))

		result, err := ReadResponseWithSizeLimit(ReadResponseWithSizeLimitOptions{
			Response: resp,
			URL:      "http://example.com/custom",
			MaxBytes: 10,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(result, data) {
			t.Errorf("got %v, want %v", result, data)
		}
	})

	t.Run("should reject at exact boundary (maxBytes + 1)", func(t *testing.T) {
		data := make([]byte, 11)
		for i := range data {
			data[i] = 1
		}

		resp := createMockResponse(data, nil)

		_, err := ReadResponseWithSizeLimit(ReadResponseWithSizeLimitOptions{
			Response: resp,
			URL:      "http://example.com/boundary",
			MaxBytes: 10,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !IsDownloadError(err) {
			t.Fatalf("expected DownloadError, got %T: %v", err, err)
		}
	})
}

// containsSubstring checks if s contains substr.
func containsSubstring(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
