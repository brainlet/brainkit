// Ported from: packages/provider-utils/src/get-from-api.ts
package providerutils

import (
	"context"
	"fmt"
	"net/http"
)

// GetFromApiOptions are the options for GetFromApi.
type GetFromApiOptions[T any] struct {
	URL                       string
	Headers                   map[string]string
	SuccessfulResponseHandler ResponseHandler[T]
	FailedResponseHandler     ResponseHandler[error]
	Ctx                       context.Context
	Fetch                     FetchFunction
}

// GetFromApiResult is the result from GetFromApi.
type GetFromApiResult[T any] struct {
	Value           T
	RawValue        interface{}
	ResponseHeaders map[string]string
}

// GetFromApi sends a GET request to an API endpoint and processes the response.
func GetFromApi[T any](opts GetFromApiOptions[T]) (*GetFromApiResult[T], error) {
	ctx := opts.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	headers := opts.Headers
	if headers == nil {
		headers = map[string]string{}
	}

	headers = WithUserAgentSuffix(
		headers,
		fmt.Sprintf("ai-sdk/provider-utils/%s", VERSION),
		GetRuntimeEnvironmentUserAgent(),
	)

	var fetch FetchFunction
	if opts.Fetch != nil {
		fetch = opts.Fetch
	} else {
		fetch = DefaultFetch
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, opts.URL, nil)
	if err != nil {
		return nil, HandleFetchError(HandleFetchErrorOptions{
			Error:             err,
			URL:               opts.URL,
			RequestBodyValues: nil,
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
			RequestBodyValues: nil,
		})
	}

	responseHeaders := ExtractResponseHeaders(resp)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if opts.FailedResponseHandler != nil {
			errorResult, handlerErr := opts.FailedResponseHandler(ResponseHandlerOptions{
				Response:          resp,
				URL:               opts.URL,
				RequestBodyValues: nil,
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
					RequestBodyValues: nil,
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
			RequestBodyValues: nil,
		})
	}

	result, err := opts.SuccessfulResponseHandler(ResponseHandlerOptions{
		Response:          resp,
		URL:               opts.URL,
		RequestBodyValues: nil,
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
			RequestBodyValues: nil,
		})
	}

	return &GetFromApiResult[T]{
		Value:           result.Value,
		RawValue:        result.RawValue,
		ResponseHeaders: result.ResponseHeaders,
	}, nil
}
