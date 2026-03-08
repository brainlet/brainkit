// Ported from: packages/provider-utils/src/response-handler.test.ts
package providerutils

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func makeResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{},
	}
}

func TestCreateJsonResponseHandler_ValidJSON(t *testing.T) {
	handler := CreateJsonResponseHandler[map[string]interface{}](nil)

	result, err := handler(ResponseHandlerOptions{
		URL:               "https://example.com",
		RequestBodyValues: nil,
		Response:          makeResponse(200, `{"name": "John"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.Value
	if m["name"] != "John" {
		t.Errorf("expected name='John', got %v", m["name"])
	}
}

func TestCreateJsonResponseHandler_InvalidJSON(t *testing.T) {
	handler := CreateJsonResponseHandler[map[string]interface{}](nil)

	_, err := handler(ResponseHandlerOptions{
		URL:               "https://example.com",
		RequestBodyValues: nil,
		Response:          makeResponse(200, "not json"),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !IsAPICallError(err) {
		t.Errorf("expected APICallError, got %T", err)
	}
}

func TestCreateBinaryResponseHandler_Success(t *testing.T) {
	handler := CreateBinaryResponseHandler()

	result, err := handler(ResponseHandlerOptions{
		URL:               "https://example.com",
		RequestBodyValues: nil,
		Response:          makeResponse(200, "binary data"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result.Value) != "binary data" {
		t.Errorf("expected 'binary data', got %q", string(result.Value))
	}
}

func TestCreateBinaryResponseHandler_EmptyBody(t *testing.T) {
	handler := CreateBinaryResponseHandler()

	resp := &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Body:       nil,
		Header:     http.Header{},
	}

	_, err := handler(ResponseHandlerOptions{
		URL:               "https://example.com",
		RequestBodyValues: nil,
		Response:          resp,
	})
	if err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestCreateStatusCodeErrorResponseHandler_Success(t *testing.T) {
	handler := CreateStatusCodeErrorResponseHandler()

	result, err := handler(ResponseHandlerOptions{
		URL:               "https://example.com",
		RequestBodyValues: nil,
		Response:          makeResponse(500, "server error"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Value == nil {
		t.Fatal("expected error value")
	}
	if result.Value.StatusCode != 500 {
		t.Errorf("expected status code 500, got %d", result.Value.StatusCode)
	}
}
