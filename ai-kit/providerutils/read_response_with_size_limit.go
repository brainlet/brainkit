// Ported from: packages/provider-utils/src/read-response-with-size-limit.ts
package providerutils

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// DefaultMaxDownloadSize is the default maximum download size: 2 GiB.
const DefaultMaxDownloadSize = 2 * 1024 * 1024 * 1024

// ReadResponseWithSizeLimitOptions are the options for ReadResponseWithSizeLimit.
type ReadResponseWithSizeLimitOptions struct {
	// Response is the HTTP response to read.
	Response *http.Response
	// URL is the URL being downloaded (used in error messages).
	URL string
	// MaxBytes is the maximum allowed bytes. Defaults to DefaultMaxDownloadSize.
	MaxBytes int64
}

// ReadResponseWithSizeLimit reads an HTTP response body with a size limit to prevent
// memory exhaustion. Checks the Content-Length header for early rejection, then reads
// the body incrementally.
func ReadResponseWithSizeLimit(opts ReadResponseWithSizeLimitOptions) ([]byte, error) {
	maxBytes := opts.MaxBytes
	if maxBytes <= 0 {
		maxBytes = DefaultMaxDownloadSize
	}

	// Early rejection based on Content-Length header
	if opts.Response != nil {
		contentLength := opts.Response.Header.Get("Content-Length")
		if contentLength != "" {
			length, err := strconv.ParseInt(contentLength, 10, 64)
			if err == nil && length > maxBytes {
				return nil, NewDownloadError(DownloadErrorOptions{
					URL:     opts.URL,
					Message: fmt.Sprintf("Download of %s exceeded maximum size of %d bytes (Content-Length: %d).", opts.URL, maxBytes, length),
				})
			}
		}
	}

	if opts.Response == nil || opts.Response.Body == nil {
		return []byte{}, nil
	}
	defer opts.Response.Body.Close()

	// Read incrementally with size checking
	var totalBytes int64
	buf := make([]byte, 0, 32*1024)
	tmp := make([]byte, 32*1024)

	for {
		n, err := opts.Response.Body.Read(tmp)
		if n > 0 {
			totalBytes += int64(n)
			if totalBytes > maxBytes {
				return nil, NewDownloadError(DownloadErrorOptions{
					URL:     opts.URL,
					Message: fmt.Sprintf("Download of %s exceeded maximum size of %d bytes.", opts.URL, maxBytes),
				})
			}
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}

	return buf, nil
}
