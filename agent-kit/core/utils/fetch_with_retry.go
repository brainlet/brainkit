// Ported from: packages/core/src/utils/fetchWithRetry.ts
package utils

import (
	"fmt"
	"net/http"
	"time"
)

// FetchWithRetry executes an HTTP request with exponential backoff retry logic.
//
// 4xx responses are treated as client errors and returned immediately without retry.
// 5xx responses and network errors are retried up to maxRetries times.
//
// The request must be cloneable for retries: if the request has a body,
// req.GetBody must be set so the body can be re-read on each attempt.
// Requests created via http.NewRequest/http.NewRequestWithContext set GetBody
// automatically for common body types (bytes.Reader, strings.Reader, etc.).
//
// If client is nil, http.DefaultClient is used.
// If maxRetries <= 0, it defaults to 3.
func FetchWithRetry(client *http.Client, req *http.Request, maxRetries int) (*http.Response, error) {
	if client == nil {
		client = http.DefaultClient
	}
	if maxRetries <= 0 {
		maxRetries = 3
	}

	var lastErr error
	retryCount := 0

	for retryCount < maxRetries {
		// Clone the request for this attempt so the original remains reusable.
		attemptReq := req.Clone(req.Context())
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, fmt.Errorf("failed to clone request body: %w", err)
			}
			attemptReq.Body = body
		}

		resp, err := client.Do(attemptReq)
		if err != nil {
			// Network / transport error — retry unless exhausted.
			lastErr = err
			retryCount++
			if retryCount >= maxRetries {
				break
			}
			delay := min(time.Second*time.Duration(1<<retryCount), 10*time.Second)
			time.Sleep(delay)
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// Success — return immediately.
			return resp, nil
		}

		// 4xx client errors — do not retry.
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			resp.Body.Close()
			return nil, fmt.Errorf("request failed with status: %d %s", resp.StatusCode, resp.Status)
		}

		// 5xx or other non-success — retry with backoff.
		resp.Body.Close()
		lastErr = fmt.Errorf("request failed with status: %d %s", resp.StatusCode, resp.Status)
		retryCount++
		if retryCount >= maxRetries {
			break
		}
		delay := min(time.Second*time.Duration(1<<retryCount), 10*time.Second)
		time.Sleep(delay)
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("request failed after multiple retry attempts")
}
