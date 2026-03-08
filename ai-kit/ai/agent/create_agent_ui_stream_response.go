// Ported from: packages/ai/src/agent/create-agent-ui-stream-response.ts
package agent

import (
	"context"
	"fmt"
	"net/http"

	gt "github.com/brainlet/brainkit/ai-kit/ai/generatetext"
)

// CreateAgentUIStreamResponseOptions contains options for CreateAgentUIStreamResponse.
type CreateAgentUIStreamResponseOptions struct {
	// Agent is the agent to run.
	Agent Agent

	// UIMessages is the input UI messages.
	UIMessages []interface{}

	// Ctx is the Go context for cancellation (replaces AbortSignal in TS).
	Ctx context.Context

	// Timeout in milliseconds. Optional.
	Timeout *gt.TimeoutConfiguration

	// Options are call-level options specific to the agent implementation.
	Options interface{}

	// ExperimentalTransform contains optional stream transformations.
	ExperimentalTransform []gt.StreamTextTransform

	// OnStepFinish is called when each step is finished. Optional.
	OnStepFinish OnStepFinishCallback

	// Response init fields (from UIMessageStreamResponseInit in TS)

	// Headers are additional headers for the response.
	Headers map[string]string

	// Status is the HTTP status code for the response.
	Status int

	// StatusText is the HTTP status text for the response.
	StatusText string

	// ConsumeSseStream controls whether to consume the SSE stream.
	ConsumeSseStream bool
}

// CreateAgentUIStreamResponse runs the agent and returns an HTTP response
// with a UI message stream.
//
// In the TypeScript source, this returns a Response object (Web API).
// In Go, this writes to an http.ResponseWriter.
//
// TODO: This function requires createUIMessageStreamResponse from the
// ui-message-stream package, which is not yet fully ported.
func CreateAgentUIStreamResponse(w http.ResponseWriter, opts CreateAgentUIStreamResponseOptions) error {
	if opts.Agent == nil {
		return fmt.Errorf("agent is required")
	}

	ctx := opts.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	// Create the UI stream
	_, err := CreateAgentUIStream(CreateAgentUIStreamOptions{
		Agent:                 opts.Agent,
		UIMessages:           opts.UIMessages,
		Ctx:                  ctx,
		Timeout:              opts.Timeout,
		Options:              opts.Options,
		ExperimentalTransform: opts.ExperimentalTransform,
		OnStepFinish:         opts.OnStepFinish,
	})
	if err != nil {
		return fmt.Errorf("create agent UI stream: %w", err)
	}

	// TODO: createUIMessageStreamResponse({
	//   headers, status, statusText, consumeSseStream, stream,
	// })
	//
	// The TS source wraps the stream in a createUIMessageStreamResponse call
	// which formats the stream as SSE and returns a Response object.
	// In Go, this would write SSE-formatted data to the http.ResponseWriter.

	// Set headers
	for k, v := range opts.Headers {
		w.Header().Set(k, v)
	}

	status := opts.Status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)

	return nil
}
