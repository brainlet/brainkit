// Ported from: packages/provider-utils/src/post-to-api.ts
package providerutils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// PostToApiBody represents the body for a POST request.
type PostToApiBody struct {
	// Content is the raw content to send (JSON string, form data bytes, etc.).
	Content io.Reader
	// Values is the original values used to construct the body (used in error messages).
	Values interface{}
}

// PostToApiOptions are the options for PostToApi.
type PostToApiOptions[T any] struct {
	URL                       string
	Headers                   map[string]string
	Body                      PostToApiBody
	FailedResponseHandler     ResponseHandler[error]
	SuccessfulResponseHandler ResponseHandler[T]
	Ctx                       context.Context
	Fetch                     FetchFunction
}

// PostToApiResult is the result from PostToApi.
type PostToApiResult[T any] struct {
	Value           T
	RawValue        interface{}
	ResponseHeaders map[string]string
}

// PostJsonToApi sends a JSON POST request to an API endpoint.
func PostJsonToApi[T any](opts PostJsonToApiOptions[T]) (*PostToApiResult[T], error) {
	bodyJSON, err := json.Marshal(opts.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	for k, v := range opts.Headers {
		headers[k] = v
	}

	return PostToApi(PostToApiOptions[T]{
		URL:     opts.URL,
		Headers: headers,
		Body: PostToApiBody{
			Content: bytes.NewReader(bodyJSON),
			Values:  opts.Body,
		},
		FailedResponseHandler:     opts.FailedResponseHandler,
		SuccessfulResponseHandler: opts.SuccessfulResponseHandler,
		Ctx:                       opts.Ctx,
		Fetch:                     opts.Fetch,
	})
}

// PostJsonToApiOptions are the options for PostJsonToApi.
type PostJsonToApiOptions[T any] struct {
	URL                       string
	Headers                   map[string]string
	Body                      interface{}
	FailedResponseHandler     ResponseHandler[error]
	SuccessfulResponseHandler ResponseHandler[T]
	Ctx                       context.Context
	Fetch                     FetchFunction
}

// PostToApi sends a POST request to an API endpoint and processes the response.
func PostToApi[T any](opts PostToApiOptions[T]) (*PostToApiResult[T], error) {
	ctx := opts.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	headers := WithUserAgentSuffix(
		opts.Headers,
		fmt.Sprintf("ai-sdk/provider-utils/%s", VERSION),
		GetRuntimeEnvironmentUserAgent(),
	)

	var fetch FetchFunction
	if opts.Fetch != nil {
		fetch = opts.Fetch
	} else {
		fetch = DefaultFetch
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, opts.URL, opts.Body.Content)
	if err != nil {
		return nil, HandleFetchError(HandleFetchErrorOptions{
			Error:             err,
			URL:               opts.URL,
			RequestBodyValues: opts.Body.Values,
		})
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := fetch(req)
	if err != nil {
		return nil, HandleFetchError(HandleFetchErrorOptions{
			Error:             err,
			URL:               opts.URL,
			RequestBodyValues: opts.Body.Values,
		})
	}

	responseHeaders := ExtractResponseHeaders(resp)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if opts.FailedResponseHandler != nil {
			errorResult, handlerErr := opts.FailedResponseHandler(ResponseHandlerOptions{
				Response:          resp,
				URL:               opts.URL,
				RequestBodyValues: opts.Body.Values,
			})

			if handlerErr != nil {
				if IsAbortError(handlerErr) || IsAPICallError(handlerErr) {
					return nil, handlerErr
				}
				return nil, NewAPICallError(APICallErrorOptions{
					Message:           "Failed to process error response",
					Cause:             handlerErr,
					StatusCode:        resp.StatusCode,
					URL:               opts.URL,
					ResponseHeaders:   responseHeaders,
					RequestBodyValues: opts.Body.Values,
				})
			}

			if errorResult != nil {
				return nil, errorResult.Value
			}
		}

		return nil, NewAPICallError(APICallErrorOptions{
			Message:           resp.Status,
			StatusCode:        resp.StatusCode,
			URL:               opts.URL,
			ResponseHeaders:   responseHeaders,
			RequestBodyValues: opts.Body.Values,
		})
	}

	result, err := opts.SuccessfulResponseHandler(ResponseHandlerOptions{
		Response:          resp,
		URL:               opts.URL,
		RequestBodyValues: opts.Body.Values,
	})
	if err != nil {
		if IsAbortError(err) || IsAPICallError(err) {
			return nil, err
		}
		return nil, NewAPICallError(APICallErrorOptions{
			Message:           "Failed to process successful response",
			Cause:             err,
			StatusCode:        resp.StatusCode,
			URL:               opts.URL,
			ResponseHeaders:   responseHeaders,
			RequestBodyValues: opts.Body.Values,
		})
	}

	return &PostToApiResult[T]{
		Value:           result.Value,
		RawValue:        result.RawValue,
		ResponseHeaders: result.ResponseHeaders,
	}, nil
}

// DefaultFetch is the default fetch function using http.DefaultClient.
func DefaultFetch(req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req)
}
