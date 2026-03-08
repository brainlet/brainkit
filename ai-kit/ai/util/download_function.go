// Ported from: packages/ai/src/util/download/download-function.ts
package util

import (
	"context"
	"net/url"
)

// DownloadRequest represents a single download request item.
type DownloadRequest struct {
	URL                   *url.URL
	IsURLSupportedByModel bool
}

// DownloadFunctionResult represents the result of a download attempt.
// Data is nil if the URL should be passed through as-is.
type DownloadFunctionResult struct {
	Data      []byte
	MediaType string
}

// DownloadFunction is a function that decides for each URL whether to download
// the asset or pass it through to the model.
type DownloadFunction func(ctx context.Context, requests []DownloadRequest) ([]*DownloadFunctionResult, error)

// CreateDefaultDownloadFunction creates a DownloadFunction that downloads files
// only when they are not supported by the model.
func CreateDefaultDownloadFunction() DownloadFunction {
	return func(ctx context.Context, requests []DownloadRequest) ([]*DownloadFunctionResult, error) {
		results := make([]*DownloadFunctionResult, len(requests))

		for i, req := range requests {
			if req.IsURLSupportedByModel {
				results[i] = nil
				continue
			}

			result, err := Download(ctx, req.URL, 0)
			if err != nil {
				return nil, err
			}

			results[i] = &DownloadFunctionResult{
				Data:      result.Data,
				MediaType: result.MediaType,
			}
		}

		return results, nil
	}
}
