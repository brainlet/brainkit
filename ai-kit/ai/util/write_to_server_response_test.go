// Ported from: packages/ai/src/util/write-to-server-response.test.ts
package util

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteToServerResponse_WriteData(t *testing.T) {
	recorder := httptest.NewRecorder()

	body := bytes.NewReader(append([]byte("chunk1"), []byte("chunk2")...))

	WriteToServerResponse(WriteToServerResponseOptions{
		Response:   recorder,
		Status:     200,
		StatusText: "OK",
		Headers:    map[string]string{"Content-Type": "text/plain"},
		Body:       body,
	})

	if recorder.Code != 200 {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	result := recorder.Body.String()
	if result != "chunk1chunk2" {
		t.Fatalf("expected chunk1chunk2, got %s", result)
	}
}

func TestWriteToServerResponse_HeadersWithoutStatusText(t *testing.T) {
	recorder := httptest.NewRecorder()

	body := bytes.NewReader([]byte("test data"))

	WriteToServerResponse(WriteToServerResponseOptions{
		Response: recorder,
		Status:   200,
		Headers: map[string]string{
			"X-Example-Header":     "example-value",
			"X-Example-Chat-Title": "My Conversation",
		},
		Body: body,
	})

	if recorder.Code != 200 {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("X-Example-Header"); got != "example-value" {
		t.Fatalf("expected X-Example-Header=example-value, got %s", got)
	}
	if got := recorder.Header().Get("X-Example-Chat-Title"); got != "My Conversation" {
		t.Fatalf("expected X-Example-Chat-Title=My Conversation, got %s", got)
	}
	if recorder.Body.String() != "test data" {
		t.Fatalf("expected 'test data', got %s", recorder.Body.String())
	}
}

func TestWriteToServerResponse_HeadersWithStatusText(t *testing.T) {
	recorder := httptest.NewRecorder()

	body := bytes.NewReader([]byte("test data"))

	WriteToServerResponse(WriteToServerResponseOptions{
		Response:   recorder,
		Status:     201,
		StatusText: "Created",
		Headers: map[string]string{
			"X-Example-Header":     "example-value",
			"X-Example-Chat-Title": "New Chat Session",
		},
		Body: body,
	})

	if recorder.Code != 201 {
		t.Fatalf("expected status 201, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("X-Example-Header"); got != "example-value" {
		t.Fatalf("expected X-Example-Header=example-value, got %s", got)
	}
	if got := recorder.Header().Get("X-Example-Chat-Title"); got != "New Chat Session" {
		t.Fatalf("expected X-Example-Chat-Title=New Chat Session, got %s", got)
	}
	if recorder.Body.String() != "test data" {
		t.Fatalf("expected 'test data', got %s", recorder.Body.String())
	}
}

func TestWriteToServerResponse_DefaultStatus(t *testing.T) {
	recorder := httptest.NewRecorder()

	body := bytes.NewReader([]byte("test data"))

	WriteToServerResponse(WriteToServerResponseOptions{
		Response: recorder,
		Headers: map[string]string{
			"X-Example-Header":  "example-value",
			"X-Example-Message": "Hello World",
		},
		Body: body,
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("X-Example-Header"); got != "example-value" {
		t.Fatalf("expected X-Example-Header=example-value, got %s", got)
	}
	if got := recorder.Header().Get("X-Example-Message"); got != "Hello World" {
		t.Fatalf("expected X-Example-Message=Hello World, got %s", got)
	}
	if recorder.Body.String() != "test data" {
		t.Fatalf("expected 'test data', got %s", recorder.Body.String())
	}
}
