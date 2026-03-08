// Ported from: packages/ai/src/util/download/download.test.ts
package util

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// withSkipSSRF returns a Download option that skips SSRF validation.
// This is the Go equivalent of the TS tests mocking globalThis.fetch.
func withSkipSSRF(o *DownloadOptions) {
	o.skipSSRFCheck = true
}

// withClient returns a Download option that sets a custom HTTP client.
func withClient(c *http.Client) func(*DownloadOptions) {
	return func(o *DownloadOptions) {
		o.Client = c
	}
}

// --- describe('download SSRF protection') ---

// TS: it('should reject private IPv4 addresses')
func TestDownloadSSRFProtection_RejectPrivateIPv4(t *testing.T) {
	ctx := context.Background()
	privateURLs := []string{
		"http://127.0.0.1/file",
		"http://10.0.0.1/file",
		"http://169.254.169.254/latest/meta-data/",
	}

	for _, rawURL := range privateURLs {
		t.Run(rawURL, func(t *testing.T) {
			u, err := url.Parse(rawURL)
			if err != nil {
				t.Fatalf("failed to parse URL %s: %v", rawURL, err)
			}
			_, dlErr := Download(ctx, u, 0)
			if dlErr == nil {
				t.Fatalf("expected error for %s, got nil", rawURL)
			}
			var de *DownloadError
			if !errors.As(dlErr, &de) {
				t.Fatalf("expected DownloadError for %s, got %T: %v", rawURL, dlErr, dlErr)
			}
		})
	}
}

// TS: it('should reject localhost')
func TestDownloadSSRFProtection_RejectLocalhost(t *testing.T) {
	ctx := context.Background()
	u, _ := url.Parse("http://localhost/file")
	_, err := Download(ctx, u, 0)
	if err == nil {
		t.Fatal("expected error for localhost, got nil")
	}
	var de *DownloadError
	if !errors.As(err, &de) {
		t.Fatalf("expected DownloadError, got %T: %v", err, err)
	}
}

// --- describe('download SSRF redirect protection') ---

// TS: it('should reject redirects to private IP addresses')
// The TS test mocks globalThis.fetch to return {redirected: true, url: "http://169.254.169.254/..."}.
// In Go, we use httptest to create a server that redirects to a second server whose URL
// we then validate. Since we cannot actually redirect to 169.254.169.254 in a test,
// we validate the redirect URL directly through validateDownloadURL, which is the same
// code path Download uses after following redirects.
func TestDownloadSSRFRedirectProtection_RejectRedirectToPrivateIP(t *testing.T) {
	err := validateDownloadURL("http://169.254.169.254/latest/meta-data/")
	if err == nil {
		t.Fatal("expected error for redirect to private IP, got nil")
	}
	var de *DownloadError
	if !errors.As(err, &de) {
		t.Fatalf("expected DownloadError, got %T: %v", err, err)
	}
}

// TS: it('should reject redirects to localhost')
func TestDownloadSSRFRedirectProtection_RejectRedirectToLocalhost(t *testing.T) {
	err := validateDownloadURL("http://localhost:8080/admin")
	if err == nil {
		t.Fatal("expected error for redirect to localhost, got nil")
	}
	var de *DownloadError
	if !errors.As(err, &de) {
		t.Fatalf("expected DownloadError, got %T: %v", err, err)
	}
}

// TS: it('should allow redirects to safe URLs')
// The TS test mocks fetch to return a redirected response from a safe URL with data.
// In Go, we use two httptest servers: one redirects to the other (the "CDN").
func TestDownloadSSRFRedirectProtection_AllowSafeRedirect(t *testing.T) {
	content := []byte{1, 2, 3}

	// "CDN" server that serves the actual content.
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(200)
		_, _ = w.Write(content)
	}))
	defer cdn.Close()

	// Origin server that redirects to the CDN.
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, cdn.URL+"/image.png", http.StatusFound)
	}))
	defer origin.Close()

	ctx := context.Background()
	u, _ := url.Parse(origin.URL + "/image.png")
	result, err := Download(ctx, u, 0, withSkipSSRF)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(result.Data, content) {
		t.Fatalf("expected data %v, got %v", content, result.Data)
	}
	if result.MediaType != "image/png" {
		t.Fatalf("expected media type image/png, got %s", result.MediaType)
	}
}

// --- describe('download') ---

// TS: it('should download data successfully and match expected bytes')
func TestDownload_SuccessfulDownload(t *testing.T) {
	expectedBytes := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(200)
		_, _ = w.Write(expectedBytes)
	}))
	defer server.Close()

	ctx := context.Background()
	u, _ := url.Parse(server.URL + "/file")
	result, err := Download(ctx, u, 0, withSkipSSRF)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !bytes.Equal(result.Data, expectedBytes) {
		t.Fatalf("expected data %v, got %v", expectedBytes, result.Data)
	}
	if result.MediaType != "application/octet-stream" {
		t.Fatalf("expected media type application/octet-stream, got %s", result.MediaType)
	}
}

// TS: it('should throw DownloadError when response is not ok')
func TestDownload_ErrorOnNonOkResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	ctx := context.Background()
	u, _ := url.Parse(server.URL + "/file")
	_, err := Download(ctx, u, 0, withSkipSSRF)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var de *DownloadError
	if !errors.As(err, &de) {
		t.Fatalf("expected DownloadError, got %T: %v", err, err)
	}
	if de.StatusCode != 404 {
		t.Fatalf("expected status code 404, got %d", de.StatusCode)
	}
	// TS: expect((error as DownloadError).statusText).toBe('Not Found')
	// Go's http package formats StatusText as "404 Not Found" in resp.Status.
	if !strings.Contains(de.StatusText, "Not Found") {
		t.Fatalf("expected StatusText to contain 'Not Found', got %q", de.StatusText)
	}
}

// TS: it('should throw DownloadError when fetch throws an error')
func TestDownload_ErrorOnFetchFailure(t *testing.T) {
	ctx := context.Background()
	// Use a URL pointing to a closed port to simulate network error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close() // close immediately to cause connection refused

	u, _ := url.Parse(serverURL + "/file")
	_, err := Download(ctx, u, 0, withSkipSSRF)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var de *DownloadError
	if !errors.As(err, &de) {
		t.Fatalf("expected DownloadError, got %T: %v", err, err)
	}
}

// TS: it('should abort when response exceeds default size limit')
func TestDownload_ExceedsDefaultSizeLimit(t *testing.T) {
	// TS uses 3 * 1024 * 1024 * 1024 in Content-Length header.
	// We use a small maxBytes for testability, same approach.
	maxBytes := 100

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", maxBytes+1000))
		w.WriteHeader(200)
		_, _ = w.Write(make([]byte, 10))
	}))
	defer server.Close()

	ctx := context.Background()
	u, _ := url.Parse(server.URL + "/large")
	_, err := Download(ctx, u, maxBytes, withSkipSSRF)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsDownloadError(err) {
		t.Fatal("expected IsDownloadError to return true")
	}
	var de *DownloadError
	if !errors.As(err, &de) {
		t.Fatalf("expected DownloadError, got %T: %v", err, err)
	}
	if !strings.Contains(de.Error(), "exceeded maximum size") {
		t.Fatalf("expected 'exceeded maximum size' in error, got: %s", de.Error())
	}
}

// TS: it('should pass abortSignal to fetch')
// In Go, AbortSignal maps to context.Context cancellation.
func TestDownload_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately, equivalent to controller.abort()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("data"))
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL + "/file")
	_, err := Download(ctx, u, 0, withSkipSSRF)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
	if !IsDownloadError(err) {
		t.Fatalf("expected DownloadError, got %T: %v", err, err)
	}
}
