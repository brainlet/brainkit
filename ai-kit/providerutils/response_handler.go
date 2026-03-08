// Ported from: packages/provider-utils/src/response-handler.ts
package providerutils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ResponseHandler is a function that processes an HTTP response and returns a typed result.
type ResponseHandler[T any] func(opts ResponseHandlerOptions) (*ResponseHandlerResult[T], error)

// ResponseHandlerOptions are the options passed to a ResponseHandler.
type ResponseHandlerOptions struct {
	URL               string
	RequestBodyValues interface{}
	Response          *http.Response
}

// ResponseHandlerResult is the result from a ResponseHandler.
type ResponseHandlerResult[T any] struct {
	Value           T
	RawValue        interface{}
	ResponseHeaders map[string]string
}

// CreateJsonErrorResponseHandler creates a ResponseHandler for JSON error responses.
func CreateJsonErrorResponseHandler[T any](
	errorSchema *Schema[T],
	errorToMessage func(T) string,
	isRetryable func(resp *http.Response, errorVal *T) bool,
) ResponseHandler[*APICallError] {
	return func(opts ResponseHandlerOptions) (*ResponseHandlerResult[*APICallError], error) {
		bodyBytes, err := io.ReadAll(opts.Response.Body)
		if err != nil {
			bodyBytes = []byte{}
		}
		defer opts.Response.Body.Close()

		responseBody := string(bodyBytes)
		responseHeaders := ExtractResponseHeaders(opts.Response)

		// Some providers return an empty response body for some errors
		if len(responseBody) == 0 || responseBody == "" {
			retryable := false
			if isRetryable != nil {
				retryable = isRetryable(opts.Response, nil)
			}
			return &ResponseHandlerResult[*APICallError]{
				ResponseHeaders: responseHeaders,
				Value: NewAPICallError(APICallErrorOptions{
					Message:           opts.Response.Status,
					URL:               opts.URL,
					RequestBodyValues: opts.RequestBodyValues,
					StatusCode:        opts.Response.StatusCode,
					ResponseHeaders:   responseHeaders,
					ResponseBody:      responseBody,
					IsRetryable:       retryable,
				}),
			}, nil
		}

		// Resilient parsing in case the response is not JSON or does not match the schema
		parsedError, parseErr := ParseJSON(responseBody, errorSchema)
		if parseErr != nil {
			retryable := false
			if isRetryable != nil {
				retryable = isRetryable(opts.Response, nil)
			}
			return &ResponseHandlerResult[*APICallError]{
				ResponseHeaders: responseHeaders,
				Value: NewAPICallError(APICallErrorOptions{
					Message:           opts.Response.Status,
					URL:               opts.URL,
					RequestBodyValues: opts.RequestBodyValues,
					StatusCode:        opts.Response.StatusCode,
					ResponseHeaders:   responseHeaders,
					ResponseBody:      responseBody,
					IsRetryable:       retryable,
				}),
			}, nil
		}

		retryable := false
		if isRetryable != nil {
			retryable = isRetryable(opts.Response, &parsedError)
		}
		return &ResponseHandlerResult[*APICallError]{
			ResponseHeaders: responseHeaders,
			Value: NewAPICallError(APICallErrorOptions{
				Message:           errorToMessage(parsedError),
				URL:               opts.URL,
				RequestBodyValues: opts.RequestBodyValues,
				StatusCode:        opts.Response.StatusCode,
				ResponseHeaders:   responseHeaders,
				ResponseBody:      responseBody,
				Data:              parsedError,
				IsRetryable:       retryable,
			}),
		}, nil
	}
}

// CreateEventSourceResponseHandler creates a ResponseHandler for SSE event streams.
func CreateEventSourceResponseHandler[T any](
	chunkSchema *Schema[T],
) ResponseHandler[<-chan ParseResult[T]] {
	return func(opts ResponseHandlerOptions) (*ResponseHandlerResult[<-chan ParseResult[T]], error) {
		responseHeaders := ExtractResponseHeaders(opts.Response)

		if opts.Response.Body == nil {
			return nil, &EmptyResponseBodyError{}
		}

		ch := ParseJsonEventStream(opts.Response.Body, chunkSchema)
		return &ResponseHandlerResult[<-chan ParseResult[T]]{
			ResponseHeaders: responseHeaders,
			Value:           ch,
		}, nil
	}
}

// CreateJsonResponseHandler creates a ResponseHandler for JSON responses.
func CreateJsonResponseHandler[T any](
	responseSchema *Schema[T],
) ResponseHandler[T] {
	return func(opts ResponseHandlerOptions) (*ResponseHandlerResult[T], error) {
		bodyBytes, err := io.ReadAll(opts.Response.Body)
		if err != nil {
			bodyBytes = []byte{}
		}
		defer opts.Response.Body.Close()

		responseBody := string(bodyBytes)
		responseHeaders := ExtractResponseHeaders(opts.Response)

		// If the schema has a Validate function, use the full parse+validate pipeline.
		if responseSchema != nil && responseSchema.Validate != nil {
			parsedResult := SafeParseJSON(responseBody, responseSchema)
			if !parsedResult.Success {
				return nil, NewAPICallError(APICallErrorOptions{
					Message:           "Invalid JSON response",
					Cause:             parsedResult.Error,
					StatusCode:        opts.Response.StatusCode,
					ResponseHeaders:   responseHeaders,
					ResponseBody:      responseBody,
					URL:               opts.URL,
					RequestBodyValues: opts.RequestBodyValues,
				})
			}
			return &ResponseHandlerResult[T]{
				ResponseHeaders: responseHeaders,
				Value:           parsedResult.Value,
				RawValue:        parsedResult.RawValue,
			}, nil
		}

		// No schema validation — unmarshal directly into the target struct type T.
		// SecureJsonParse returns map[string]interface{} which cannot be type-asserted
		// to concrete struct types, so we use json.Unmarshal instead.
		var result T
		if err := json.Unmarshal(bodyBytes, &result); err != nil {
			return nil, NewAPICallError(APICallErrorOptions{
				Message:           "Invalid JSON response",
				Cause:             NewJSONParseError(responseBody, err),
				StatusCode:        opts.Response.StatusCode,
				ResponseHeaders:   responseHeaders,
				ResponseBody:      responseBody,
				URL:               opts.URL,
				RequestBodyValues: opts.RequestBodyValues,
			})
		}

		return &ResponseHandlerResult[T]{
			ResponseHeaders: responseHeaders,
			Value:           result,
			RawValue:        responseBody,
		}, nil
	}
}

// CreateBinaryResponseHandler creates a ResponseHandler for binary responses.
func CreateBinaryResponseHandler() ResponseHandler[[]byte] {
	return func(opts ResponseHandlerOptions) (*ResponseHandlerResult[[]byte], error) {
		responseHeaders := ExtractResponseHeaders(opts.Response)

		if opts.Response.Body == nil {
			return nil, NewAPICallError(APICallErrorOptions{
				Message:           "Response body is empty",
				URL:               opts.URL,
				RequestBodyValues: opts.RequestBodyValues,
				StatusCode:        opts.Response.StatusCode,
				ResponseHeaders:   responseHeaders,
			})
		}

		data, err := io.ReadAll(opts.Response.Body)
		defer opts.Response.Body.Close()

		if err != nil {
			return nil, NewAPICallError(APICallErrorOptions{
				Message:           fmt.Sprintf("Failed to read response body: %v", err),
				URL:               opts.URL,
				RequestBodyValues: opts.RequestBodyValues,
				StatusCode:        opts.Response.StatusCode,
				ResponseHeaders:   responseHeaders,
				Cause:             err,
			})
		}

		return &ResponseHandlerResult[[]byte]{
			ResponseHeaders: responseHeaders,
			Value:           data,
		}, nil
	}
}

// CreateStatusCodeErrorResponseHandler creates a ResponseHandler for status code errors.
func CreateStatusCodeErrorResponseHandler() ResponseHandler[*APICallError] {
	return func(opts ResponseHandlerOptions) (*ResponseHandlerResult[*APICallError], error) {
		responseHeaders := ExtractResponseHeaders(opts.Response)

		bodyBytes, _ := io.ReadAll(opts.Response.Body)
		defer opts.Response.Body.Close()
		responseBody := string(bodyBytes)

		return &ResponseHandlerResult[*APICallError]{
			ResponseHeaders: responseHeaders,
			Value: NewAPICallError(APICallErrorOptions{
				Message:           opts.Response.Status,
				URL:               opts.URL,
				RequestBodyValues: opts.RequestBodyValues,
				StatusCode:        opts.Response.StatusCode,
				ResponseHeaders:   responseHeaders,
				ResponseBody:      responseBody,
			}),
		}, nil
	}
}
