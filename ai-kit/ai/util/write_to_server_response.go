// Ported from: packages/ai/src/util/write-to-server-response.ts
package util

import (
	"io"
	"net/http"
)

// WriteToServerResponseOptions configures writing a stream to an HTTP response.
type WriteToServerResponseOptions struct {
	Response   http.ResponseWriter
	Status     int
	StatusText string
	Headers    map[string]string
	Body       io.Reader
}

// WriteToServerResponse writes the content of a reader to an HTTP response.
func WriteToServerResponse(opts WriteToServerResponseOptions) {
	statusCode := opts.Status
	if statusCode == 0 {
		statusCode = 200
	}

	// Set headers
	for key, value := range opts.Headers {
		opts.Response.Header().Set(key, value)
	}

	opts.Response.WriteHeader(statusCode)

	if opts.Body != nil {
		buf := make([]byte, 4096)
		for {
			n, err := opts.Body.Read(buf)
			if n > 0 {
				_, _ = opts.Response.Write(buf[:n])
				if f, ok := opts.Response.(http.Flusher); ok {
					f.Flush()
				}
			}
			if err != nil {
				break
			}
		}
	}
}
