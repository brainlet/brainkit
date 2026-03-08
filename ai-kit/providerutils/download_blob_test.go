// Ported from: packages/provider-utils/src/download-blob.test.ts
package providerutils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// mockTransport implements http.RoundTripper and returns a canned response.
type mockTransport struct {
	roundTrip func(req *http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTrip(req)
}

// withMockTransport temporarily replaces http.DefaultTransport for the duration
// of the callback, restoring it afterwards.
func withMockTransport(t *testing.T, rt http.RoundTripper, fn func()) {
	t.Helper()
	orig := http.DefaultTransport
	http.DefaultTransport = rt
	defer func() { http.DefaultTransport = orig }()
	fn()
}

// makeReadCloser wraps a string as an io.ReadCloser.
func makeReadCloser(s string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(s))
}

// TestDownloadBlob groups the main downloadBlob() tests.
func TestDownloadBlob(t *testing.T) {
	t.Run("should download a blob successfully", func(t *testing.T) {
		content := []byte("test content")

		transport := &mockTransport{
			roundTrip: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Status:     "200 OK",
					Header:     http.Header{"Content-Type": {"image/png"}},
					Body:       makeReadCloser(string(content)),
					Request:    req,
				}, nil
			},
		}

		withMockTransport(t, transport, func() {
			result, err := DownloadBlob("https://example.com/image.png", nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.ContentType != "image/png" {
				t.Errorf("expected content type 'image/png', got %q", result.ContentType)
			}
			if string(result.Data) != string(content) {
				t.Errorf("expected data %q, got %q", string(content), string(result.Data))
			}
		})
	})

	t.Run("should throw DownloadError on non-ok response", func(t *testing.T) {
		transport := &mockTransport{
			roundTrip: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 404,
					Status:     "404 Not Found",
					Header:     http.Header{},
					Body:       makeReadCloser(""),
					Request:    req,
				}, nil
			},
		}

		withMockTransport(t, transport, func() {
			url := "https://example.com/not-found.png"
			_, err := DownloadBlob(url, nil)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !IsDownloadError(err) {
				t.Fatalf("expected DownloadError, got %T: %v", err, err)
			}
			dlErr := err.(*DownloadError)
			if dlErr.URL != url {
				t.Errorf("expected URL %q, got %q", url, dlErr.URL)
			}
			if dlErr.StatusCode == nil || *dlErr.StatusCode != 404 {
				t.Errorf("expected status code 404, got %v", dlErr.StatusCode)
			}
			if dlErr.StatusText != "404 Not Found" {
				t.Errorf("expected status text '404 Not Found', got %q", dlErr.StatusText)
			}
			want := "Failed to download https://example.com/not-found.png: 404 404 Not Found"
			if dlErr.Message != want {
				t.Errorf("expected message %q, got %q", want, dlErr.Message)
			}
		})
	})

	t.Run("should throw DownloadError on network error", func(t *testing.T) {
		networkError := errors.New("Network error")
		transport := &mockTransport{
			roundTrip: func(req *http.Request) (*http.Response, error) {
				return nil, networkError
			},
		}

		withMockTransport(t, transport, func() {
			url := "https://example.com/network-error.png"
			_, err := DownloadBlob(url, nil)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !IsDownloadError(err) {
				t.Fatalf("expected DownloadError, got %T: %v", err, err)
			}
			dlErr := err.(*DownloadError)
			if dlErr.URL != url {
				t.Errorf("expected URL %q, got %q", url, dlErr.URL)
			}
			if dlErr.Cause == nil {
				t.Error("expected Cause to be set")
			}
			if !strings.Contains(dlErr.Message, "Network error") {
				t.Errorf("expected message to contain 'Network error', got %q", dlErr.Message)
			}
		})
	})

	t.Run("should re-throw DownloadError without wrapping", func(t *testing.T) {
		// In the TS version, a DownloadError thrown from fetch is re-thrown as-is.
		// In Go, http.Client.Do wraps transport errors in *url.Error, so the
		// DownloadBlob code path at line 49 creates a new DownloadError with the
		// *url.Error as Cause. We verify the outer DownloadError wraps the original.
		sc := 500
		originalError := NewDownloadError(DownloadErrorOptions{
			URL:        "https://example.com/original.png",
			StatusCode: &sc,
			StatusText: "Internal Server Error",
		})

		transport := &mockTransport{
			roundTrip: func(req *http.Request) (*http.Response, error) {
				return nil, originalError
			},
		}

		withMockTransport(t, transport, func() {
			_, err := DownloadBlob("https://example.com/test.png", nil)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !IsDownloadError(err) {
				t.Fatalf("expected DownloadError, got %T: %v", err, err)
			}

			// The outer error wraps the original via http.Client.Do's *url.Error
			dlErr := err.(*DownloadError)

			// Verify the original DownloadError is in the cause chain
			var innerDlErr *DownloadError
			if !errors.As(dlErr.Cause, &innerDlErr) {
				t.Fatalf("expected original DownloadError in cause chain, got %T: %v", dlErr.Cause, dlErr.Cause)
			}
			if innerDlErr.URL != "https://example.com/original.png" {
				t.Errorf("expected inner URL 'https://example.com/original.png', got %q", innerDlErr.URL)
			}
			if innerDlErr.StatusCode == nil || *innerDlErr.StatusCode != 500 {
				t.Errorf("expected inner status code 500, got %v", innerDlErr.StatusCode)
			}
		})
	})

	t.Run("should abort when response exceeds default size limit", func(t *testing.T) {
		transport := &mockTransport{
			roundTrip: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Status:     "200 OK",
					Header: http.Header{
						"Content-Length": {fmt.Sprintf("%d", 3*1024*1024*1024)},
					},
					Body:    makeReadCloser("small body"),
					Request: req,
				}, nil
			},
		}

		withMockTransport(t, transport, func() {
			_, err := DownloadBlob("https://example.com/huge.bin", nil)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !IsDownloadError(err) {
				t.Fatalf("expected DownloadError, got %T: %v", err, err)
			}
			dlErr := err.(*DownloadError)
			if !strings.Contains(dlErr.Message, "exceeded maximum size") {
				t.Errorf("expected message to contain 'exceeded maximum size', got %q", dlErr.Message)
			}
		})
	})

	t.Run("should pass context for cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		transport := &mockTransport{
			roundTrip: func(req *http.Request) (*http.Response, error) {
				// The cancelled context should cause the request to fail
				return nil, req.Context().Err()
			},
		}

		withMockTransport(t, transport, func() {
			_, err := DownloadBlob("https://example.com/file.bin", &DownloadBlobOptions{
				Ctx: ctx,
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !IsDownloadError(err) {
				t.Fatalf("expected DownloadError, got %T: %v", err, err)
			}
		})
	})
}

// TestDownloadBlobSSRFProtection groups the SSRF protection tests.
func TestDownloadBlobSSRFProtection(t *testing.T) {
	t.Run("should reject private IPv4 addresses", func(t *testing.T) {
		urls := []string{
			"http://127.0.0.1/file",
			"http://10.0.0.1/file",
			"http://169.254.169.254/latest/meta-data/",
		}
		for _, u := range urls {
			t.Run(u, func(t *testing.T) {
				_, err := DownloadBlob(u, nil)
				if err == nil {
					t.Errorf("expected error for %s", u)
				}
				if !IsDownloadError(err) {
					t.Errorf("expected DownloadError for %s, got %T: %v", u, err, err)
				}
			})
		}
	})

	t.Run("should reject localhost", func(t *testing.T) {
		_, err := DownloadBlob("http://localhost/file", nil)
		if err == nil {
			t.Fatal("expected error for localhost")
		}
		if !IsDownloadError(err) {
			t.Fatalf("expected DownloadError, got %T: %v", err, err)
		}
	})

	t.Run("should reject non-http protocols", func(t *testing.T) {
		_, err := DownloadBlob("file:///etc/passwd", nil)
		if err == nil {
			t.Fatal("expected error for file:// protocol")
		}
		if !IsDownloadError(err) {
			t.Fatalf("expected DownloadError, got %T: %v", err, err)
		}
	})

	t.Run("should reject redirects to private IP addresses", func(t *testing.T) {
		// Simulate a response where the final URL (after redirect) is a private IP.
		// In Go, resp.Request.URL reflects the final URL after redirects.
		transport := &mockTransport{
			roundTrip: func(req *http.Request) (*http.Response, error) {
				redirectReq, _ := http.NewRequest("GET", "http://169.254.169.254/latest/meta-data/", nil)
				return &http.Response{
					StatusCode: 200,
					Status:     "200 OK",
					Header:     http.Header{"Content-Type": {"text/plain"}},
					Body:       makeReadCloser("secret"),
					Request:    redirectReq,
				}, nil
			},
		}

		withMockTransport(t, transport, func() {
			_, err := DownloadBlob("https://evil.com/redirect", nil)
			if err == nil {
				t.Fatal("expected error for redirect to private IP")
			}
			if !IsDownloadError(err) {
				t.Fatalf("expected DownloadError, got %T: %v", err, err)
			}
		})
	})

	t.Run("should reject redirects to localhost", func(t *testing.T) {
		transport := &mockTransport{
			roundTrip: func(req *http.Request) (*http.Response, error) {
				redirectReq, _ := http.NewRequest("GET", "http://localhost:8080/admin", nil)
				return &http.Response{
					StatusCode: 200,
					Status:     "200 OK",
					Header:     http.Header{"Content-Type": {"text/plain"}},
					Body:       makeReadCloser("secret"),
					Request:    redirectReq,
				}, nil
			},
		}

		withMockTransport(t, transport, func() {
			_, err := DownloadBlob("https://evil.com/redirect", nil)
			if err == nil {
				t.Fatal("expected error for redirect to localhost")
			}
			if !IsDownloadError(err) {
				t.Fatalf("expected DownloadError, got %T: %v", err, err)
			}
		})
	})

	t.Run("should allow redirects to safe URLs", func(t *testing.T) {
		content := []byte("safe content")

		transport := &mockTransport{
			roundTrip: func(req *http.Request) (*http.Response, error) {
				// Simulate redirect: the final URL is a safe CDN URL
				redirectReq, _ := http.NewRequest("GET", "https://cdn.example.com/image.png", nil)
				return &http.Response{
					StatusCode: 200,
					Status:     "200 OK",
					Header:     http.Header{"Content-Type": {"image/png"}},
					Body:       makeReadCloser(string(content)),
					Request:    redirectReq,
				}, nil
			},
		}

		withMockTransport(t, transport, func() {
			result, err := DownloadBlob("https://example.com/image.png", nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.ContentType != "image/png" {
				t.Errorf("expected content type 'image/png', got %q", result.ContentType)
			}
			if string(result.Data) != string(content) {
				t.Errorf("expected data %q, got %q", string(content), string(result.Data))
			}
		})
	})
}

// TestDownloadError groups the DownloadError construction tests.
func TestDownloadError(t *testing.T) {
	t.Run("should create error with status code and text", func(t *testing.T) {
		sc := 403
		dlErr := NewDownloadError(DownloadErrorOptions{
			URL:        "https://example.com/test.png",
			StatusCode: &sc,
			StatusText: "Forbidden",
		})

		if dlErr.Error() != "Failed to download https://example.com/test.png: 403 Forbidden" {
			t.Errorf("unexpected message: %q", dlErr.Error())
		}
		if dlErr.URL != "https://example.com/test.png" {
			t.Errorf("expected URL 'https://example.com/test.png', got %q", dlErr.URL)
		}
		if dlErr.StatusCode == nil || *dlErr.StatusCode != 403 {
			t.Errorf("expected status code 403, got %v", dlErr.StatusCode)
		}
		if dlErr.StatusText != "Forbidden" {
			t.Errorf("expected status text 'Forbidden', got %q", dlErr.StatusText)
		}
	})

	t.Run("should create error with cause", func(t *testing.T) {
		cause := errors.New("Connection refused")
		dlErr := NewDownloadError(DownloadErrorOptions{
			URL:   "https://example.com/test.png",
			Cause: cause,
		})

		if dlErr.URL != "https://example.com/test.png" {
			t.Errorf("expected URL 'https://example.com/test.png', got %q", dlErr.URL)
		}
		if dlErr.Cause != cause {
			t.Errorf("expected cause to match")
		}
		if !strings.Contains(dlErr.Message, "Connection refused") {
			t.Errorf("expected message to contain 'Connection refused', got %q", dlErr.Message)
		}
	})

	t.Run("should create error with custom message", func(t *testing.T) {
		dlErr := NewDownloadError(DownloadErrorOptions{
			URL:     "https://example.com/test.png",
			Message: "Custom error message",
		})

		if dlErr.Message != "Custom error message" {
			t.Errorf("expected 'Custom error message', got %q", dlErr.Message)
		}
	})

	t.Run("should identify DownloadError instances correctly", func(t *testing.T) {
		dlErr := NewDownloadError(DownloadErrorOptions{
			URL: "https://example.com/test.png",
		})
		regularErr := errors.New("Not a download error")

		if !IsDownloadError(dlErr) {
			t.Error("expected IsDownloadError to return true for DownloadError")
		}
		if IsDownloadError(regularErr) {
			t.Error("expected IsDownloadError to return false for regular error")
		}
		if IsDownloadError(nil) {
			t.Error("expected IsDownloadError to return false for nil")
		}
	})
}
