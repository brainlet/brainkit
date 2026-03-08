// Ported from: packages/core/src/agent/message-list/prompt/download-assets.ts
package prompt

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
)

// DownloadFromUrlOptions contains options for downloading a URL.
type DownloadFromUrlOptions struct {
	URL             *url.URL
	DownloadRetries int
}

// DownloadFromUrl downloads content from a URL with retry support.
func DownloadFromUrl(opts DownloadFromUrlOptions) (DownloadedAsset, error) {
	urlText := opts.URL.String()

	var lastErr error
	for attempt := 0; attempt <= opts.DownloadRetries; attempt++ {
		resp, err := http.Get(urlText)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("failed to download asset: HTTP %d", resp.StatusCode)
			continue
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		return DownloadedAsset{
			Data:      data,
			MediaType: resp.Header.Get("Content-Type"),
		}, nil
	}

	return DownloadedAsset{}, fmt.Errorf("failed to download asset after %d retries: %w", opts.DownloadRetries, lastErr)
}

// DownloadAssetsFromMessagesOptions contains options for downloading assets from messages.
type DownloadAssetsFromMessagesOptions struct {
	Messages             []map[string]any
	DownloadConcurrency  int
	DownloadRetries      int
	SupportedUrls        map[string][]string
}

// DownloadAssetsFromMessages downloads image/file assets from model messages.
// TODO: This is a simplified port. The TS version uses p-map for concurrency
// and isUrlSupported from @ai-sdk/provider-utils-v5.
func DownloadAssetsFromMessages(opts DownloadAssetsFromMessagesOptions) map[string]DownloadedAsset {
	concurrency := opts.DownloadConcurrency
	if concurrency <= 0 {
		concurrency = 10
	}
	retries := opts.DownloadRetries
	if retries <= 0 {
		retries = 3
	}

	// Collect URLs to download
	type downloadItem struct {
		urlObj *url.URL
	}
	var items []downloadItem

	for _, msg := range opts.Messages {
		role, _ := msg["role"].(string)
		if role != "user" {
			continue
		}
		content, ok := msg["content"].([]any)
		if !ok {
			continue
		}
		for _, partRaw := range content {
			part, ok := partRaw.(map[string]any)
			if !ok {
				continue
			}
			partType, _ := part["type"].(string)
			if partType != "image" && partType != "file" {
				continue
			}
			var dataStr string
			if partType == "image" {
				dataStr, _ = part["image"].(string)
			} else {
				dataStr, _ = part["data"].(string)
			}
			if dataStr == "" {
				continue
			}
			u, err := url.Parse(dataStr)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
				continue
			}
			items = append(items, downloadItem{urlObj: u})
		}
	}

	// Download concurrently using a semaphore
	result := make(map[string]DownloadedAsset)
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	for _, item := range items {
		wg.Add(1)
		sem <- struct{}{}
		go func(u *url.URL) {
			defer wg.Done()
			defer func() { <-sem }()

			asset, err := DownloadFromUrl(DownloadFromUrlOptions{
				URL:             u,
				DownloadRetries: retries,
			})
			if err != nil {
				return
			}
			mu.Lock()
			result[u.String()] = asset
			mu.Unlock()
		}(item.urlObj)
	}
	wg.Wait()

	return result
}
