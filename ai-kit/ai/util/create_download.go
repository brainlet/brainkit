// Ported from: packages/ai/src/util/download/create-download.ts
package util

import (
	"context"
	"net/url"
)

// CreateDownload creates a download function with configurable options.
// The returned function downloads from a URL with the given maxBytes limit.
func CreateDownload(maxBytes int) func(ctx context.Context, downloadURL *url.URL) (*DownloadResult, error) {
	return func(ctx context.Context, downloadURL *url.URL) (*DownloadResult, error) {
		return Download(ctx, downloadURL, maxBytes)
	}
}
