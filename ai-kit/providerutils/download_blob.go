// Ported from: packages/provider-utils/src/download-blob.ts
package providerutils

import (
	"context"
	"net/http"
)

// DownloadBlobOptions are the options for DownloadBlob.
type DownloadBlobOptions struct {
	// MaxBytes is the maximum allowed download size in bytes. Defaults to DefaultMaxDownloadSize.
	MaxBytes int64
	// Ctx is the context for cancellation.
	Ctx context.Context
}

// DownloadBlobResult contains the downloaded data and its content type.
type DownloadBlobResult struct {
	// Data is the downloaded bytes.
	Data []byte
	// ContentType is the MIME type from the response, if available.
	ContentType string
}

// DownloadBlob downloads a file from a URL and returns the data with its content type.
func DownloadBlob(url string, opts *DownloadBlobOptions) (*DownloadBlobResult, error) {
	if err := ValidateDownloadUrl(url); err != nil {
		return nil, err
	}

	ctx := context.Background()
	var maxBytes int64 = DefaultMaxDownloadSize
	if opts != nil {
		if opts.Ctx != nil {
			ctx = opts.Ctx
		}
		if opts.MaxBytes > 0 {
			maxBytes = opts.MaxBytes
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, NewDownloadError(DownloadErrorOptions{URL: url, Cause: err})
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, NewDownloadError(DownloadErrorOptions{URL: url, Cause: err})
	}

	// Validate final URL after redirects to prevent SSRF via open redirect
	if resp.Request != nil && resp.Request.URL != nil && resp.Request.URL.String() != url {
		if err := ValidateDownloadUrl(resp.Request.URL.String()); err != nil {
			resp.Body.Close()
			return nil, err
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		sc := resp.StatusCode
		return nil, NewDownloadError(DownloadErrorOptions{
			URL:        url,
			StatusCode: &sc,
			StatusText: resp.Status,
		})
	}

	data, err := ReadResponseWithSizeLimit(ReadResponseWithSizeLimitOptions{
		Response: resp,
		URL:      url,
		MaxBytes: maxBytes,
	})
	if err != nil {
		if IsDownloadError(err) {
			return nil, err
		}
		return nil, NewDownloadError(DownloadErrorOptions{URL: url, Cause: err})
	}

	contentType := resp.Header.Get("Content-Type")
	return &DownloadBlobResult{
		Data:        data,
		ContentType: contentType,
	}, nil
}
