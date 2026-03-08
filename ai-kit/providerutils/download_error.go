// Ported from: packages/provider-utils/src/download-error.ts
package providerutils

import "fmt"

// DownloadError represents an error that occurred during a file download.
type DownloadError struct {
	// URL that was being downloaded.
	URL string
	// StatusCode from the HTTP response, if available.
	StatusCode *int
	// StatusText from the HTTP response, if available.
	StatusText string
	// Message is the human-readable error message.
	Message string
	// Cause is the underlying error, if any.
	Cause error
}

// DownloadErrorOptions are the options for creating a DownloadError.
type DownloadErrorOptions struct {
	URL        string
	StatusCode *int
	StatusText string
	Message    string
	Cause      error
}

// NewDownloadError creates a new DownloadError.
func NewDownloadError(opts DownloadErrorOptions) *DownloadError {
	msg := opts.Message
	if msg == "" {
		if opts.Cause == nil {
			sc := 0
			if opts.StatusCode != nil {
				sc = *opts.StatusCode
			}
			msg = fmt.Sprintf("Failed to download %s: %d %s", opts.URL, sc, opts.StatusText)
		} else {
			msg = fmt.Sprintf("Failed to download %s: %v", opts.URL, opts.Cause)
		}
	}
	return &DownloadError{
		URL:        opts.URL,
		StatusCode: opts.StatusCode,
		StatusText: opts.StatusText,
		Message:    msg,
		Cause:      opts.Cause,
	}
}

func (e *DownloadError) Error() string {
	return e.Message
}

func (e *DownloadError) Unwrap() error {
	return e.Cause
}

// IsDownloadError checks whether the given error is a DownloadError.
func IsDownloadError(err error) bool {
	_, ok := err.(*DownloadError)
	return ok
}
