// Ported from: packages/ai/src/util/download/download.ts
package util

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// DownloadError represents an error that occurred during a download.
type DownloadError struct {
	URL        string
	StatusCode int
	StatusText string
	Cause      error
	Message    string
}

func (e *DownloadError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.StatusCode != 0 {
		return fmt.Sprintf("download failed: %s %d %s", e.URL, e.StatusCode, e.StatusText)
	}
	if e.Cause != nil {
		return fmt.Sprintf("download failed: %s: %v", e.URL, e.Cause)
	}
	return fmt.Sprintf("download failed: %s", e.URL)
}

func (e *DownloadError) Unwrap() error {
	return e.Cause
}

// IsDownloadError checks if an error is a DownloadError.
func IsDownloadError(err error) bool {
	_, ok := err.(*DownloadError)
	return ok
}

// DefaultMaxDownloadSize is the default maximum download size (100 MiB).
const DefaultMaxDownloadSize = 100 * 1024 * 1024

// DownloadResult holds the result of a successful download.
type DownloadResult struct {
	Data      []byte
	MediaType string
}

// validateDownloadURL checks whether a URL is safe (not a private/localhost address).
func validateDownloadURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return &DownloadError{URL: rawURL, Message: fmt.Sprintf("invalid URL: %v", err)}
	}

	host := u.Hostname()

	// Reject localhost
	if strings.EqualFold(host, "localhost") {
		return &DownloadError{URL: rawURL, Message: "download URL points to localhost"}
	}

	// Reject private IPs
	ip := net.ParseIP(host)
	if ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return &DownloadError{URL: rawURL, Message: fmt.Sprintf("download URL points to private address: %s", host)}
		}
	}

	return nil
}

// DownloadOptions configures the Download function.
type DownloadOptions struct {
	URL      *url.URL
	MaxBytes int
	// Client is an optional HTTP client to use. Defaults to http.DefaultClient.
	Client *http.Client
	// skipSSRFCheck disables SSRF URL validation. Used only in tests.
	skipSSRFCheck bool
}

// Download downloads a file from a URL.
func Download(ctx context.Context, downloadURL *url.URL, maxBytes int, opts ...func(*DownloadOptions)) (*DownloadResult, error) {
	urlText := downloadURL.String()

	dopts := &DownloadOptions{URL: downloadURL, MaxBytes: maxBytes}
	for _, opt := range opts {
		opt(dopts)
	}

	if !dopts.skipSSRFCheck {
		if err := validateDownloadURL(urlText); err != nil {
			return nil, err
		}
	}

	if maxBytes <= 0 {
		maxBytes = DefaultMaxDownloadSize
	}

	client := dopts.Client
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlText, nil)
	if err != nil {
		return nil, &DownloadError{URL: urlText, Cause: err}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, &DownloadError{URL: urlText, Cause: err}
	}
	defer resp.Body.Close()

	// Validate final URL after redirects to prevent SSRF via open redirect.
	if !dopts.skipSSRFCheck && resp.Request.URL.String() != urlText {
		if err := validateDownloadURL(resp.Request.URL.String()); err != nil {
			return nil, err
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &DownloadError{
			URL:        urlText,
			StatusCode: resp.StatusCode,
			StatusText: resp.Status,
		}
	}

	// Check Content-Length header for early rejection.
	if resp.ContentLength > int64(maxBytes) {
		return nil, &DownloadError{
			URL:     urlText,
			Message: fmt.Sprintf("download exceeded maximum size of %d bytes", maxBytes),
		}
	}

	// Read with size limit.
	limitedReader := io.LimitReader(resp.Body, int64(maxBytes)+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, &DownloadError{URL: urlText, Cause: err}
	}

	if len(data) > maxBytes {
		return nil, &DownloadError{
			URL:     urlText,
			Message: fmt.Sprintf("download exceeded maximum size of %d bytes", maxBytes),
		}
	}

	mediaType := resp.Header.Get("Content-Type")

	return &DownloadResult{
		Data:      data,
		MediaType: mediaType,
	}, nil
}
