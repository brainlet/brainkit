// Ported from: packages/ai/src/agent/pipe-agent-ui-stream-to-response.ts
package agent

import (
	"context"
	"fmt"
	"net/http"

	gt "github.com/brainlet/brainkit/ai-kit/ai/generatetext"
)

// PipeAgentUIStreamToResponseOptions contains options for PipeAgentUIStreamToResponse.
type PipeAgentUIStreamToResponseOptions struct {
	// Response is the http.ResponseWriter to pipe to.
	// In TS, this is a Node.js ServerResponse.
	Response http.ResponseWriter

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

// PipeAgentUIStreamToResponse pipes the agent UI message stream to an
// http.ResponseWriter.
//
// In the TypeScript source, this pipes to a Node.js ServerResponse object.
// In Go, this writes to an http.ResponseWriter.
//
// TODO: This function requires pipeUIMessageStreamToResponse from the
// ui-message-stream package, which is not yet fully ported.
func PipeAgentUIStreamToResponse(opts PipeAgentUIStreamToResponseOptions) error {
	if opts.Agent == nil {
		return fmt.Errorf("agent is required")
	}
	if opts.Response == nil {
		return fmt.Errorf("response is required")
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

	// TODO: pipeUIMessageStreamToResponse({
	//   response, headers, status, statusText, consumeSseStream, stream,
	// })
	//
	// The TS source calls pipeUIMessageStreamToResponse which pipes the
	// SSE-formatted stream directly to the Node.js ServerResponse.
	// In Go, this would write SSE-formatted data to the http.ResponseWriter.

	// Set headers
	for k, v := range opts.Headers {
		opts.Response.Header().Set(k, v)
	}

	status := opts.Status
	if status == 0 {
		status = http.StatusOK
	}
	opts.Response.WriteHeader(status)

	return nil
}
