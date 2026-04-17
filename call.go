package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/brainlet/brainkit/internal/bus/caller"
	"github.com/brainlet/brainkit/sdk"
)

// BufferPolicy controls how a streaming Call handles chunks when its buffer
// is full. Re-exported from the internal caller package.
type BufferPolicy = caller.BufferPolicy

const (
	// BufferBlock applies back-pressure to the producer until a slot frees.
	BufferBlock = caller.BufferBlock
	// BufferDropNewest drops incoming chunks when the buffer is full.
	BufferDropNewest = caller.BufferDropNewest
	// BufferDropOldest evicts the oldest buffered chunk to make room.
	BufferDropOldest = caller.BufferDropOldest
	// BufferError finalizes the call with *caller.BufferOverflowError.
	BufferError = caller.BufferError
)

// CallOption configures a Call or CallStream invocation.
type CallOption func(*callConfig)

type callConfig struct {
	timeout        time.Duration
	targetNS       string
	meta           map[string]string
	bufferSize     int
	bufferPolicy   BufferPolicy
	noCancelSignal bool
}

// WithCallTimeout injects an absolute timeout. If ctx already has an earlier
// deadline, that wins.
func WithCallTimeout(d time.Duration) CallOption {
	return func(c *callConfig) { c.timeout = d }
}

// WithCallTo routes the call to a different namespace. Requires the runtime
// to implement sdk.CrossNamespaceRuntime.
func WithCallTo(namespace string) CallOption {
	return func(c *callConfig) { c.targetNS = namespace }
}

// WithCallMeta adds metadata key/values to the published message.
func WithCallMeta(meta map[string]string) CallOption {
	return func(c *callConfig) { c.meta = meta }
}

// WithCallBuffer sets the per-pending stream channel capacity. Only meaningful
// for CallStream. 0 (the default) uses caller.DefaultBufferSize.
func WithCallBuffer(n int) CallOption {
	return func(c *callConfig) { c.bufferSize = n }
}

// WithCallBufferPolicy selects how the stream channel handles overflow.
// Only meaningful for CallStream. Defaults to BufferBlock.
func WithCallBufferPolicy(p BufferPolicy) CallOption {
	return func(c *callConfig) { c.bufferPolicy = p }
}

// WithCallNoCancelSignal disables the best-effort `_brainkit.cancel`
// publish that normally fires when ctx is cancelled before a terminal
// reply. Use when the remote side opts out of cancellation.
func WithCallNoCancelSignal() CallOption {
	return func(c *callConfig) { c.noCancelSignal = true }
}

// Caller returns the Kit's shared-inbox reply router.
// Used by modules and advanced callers; most users should prefer Call.
func (k *Kit) Caller() *caller.Caller {
	return k.kernel.Caller()
}

// Call sends a typed request to the target topic and waits for a typed
// response. Requires either a ctx with a deadline or WithCallTimeout.
//
// Behaviour:
//
//   - Generates a correlationID, registers a pending entry on the Kit's
//     shared inbox, publishes with replyTo=inbox.
//   - Reply arrives on the inbox, is demultiplexed by correlationID, and
//     unmarshalled into Resp.
//   - If ctx expires via deadline → *caller.CallTimeoutError.
//   - If ctx is cancelled for any other reason → *caller.CallCancelledError.
//   - If the payload can't decode into Resp → *caller.DecodeError (with raw
//     payload preserved).
//
// Resp of json.RawMessage short-circuits the decode and returns raw bytes.
func Call[Req sdk.BrainkitMessage, Resp any](k *Kit, ctx context.Context, req Req, opts ...CallOption) (Resp, error) {
	var zero Resp
	cfg := callConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	if _, hasDeadline := ctx.Deadline(); !hasDeadline && cfg.timeout <= 0 {
		return zero, &caller.NoDeadlineError{}
	}
	if cfg.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.timeout)
		defer cancel()
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return zero, fmt.Errorf("brainkit.Call: marshal %T: %w", req, err)
	}

	c := k.Caller()
	if c == nil {
		return zero, fmt.Errorf("brainkit.Call: caller not initialized")
	}

	replyPayload, err := c.Call(ctx, req.BusTopic(), payload, caller.Config{
		TargetNamespace: cfg.targetNS,
		Metadata:        cfg.meta,
		NoCancelSignal:  cfg.noCancelSignal,
	})
	if err != nil {
		return zero, err
	}

	if rm, ok := any(&zero).(*json.RawMessage); ok {
		*rm = replyPayload
		return zero, nil
	}

	var resp Resp
	if err := json.Unmarshal(replyPayload, &resp); err != nil {
		return zero, &caller.DecodeError{Topic: req.BusTopic(), Payload: replyPayload, Cause: err}
	}
	return resp, nil
}

// CallStream sends a typed request, forwards intermediate chunks through
// onChunk in arrival order, then returns the terminal reply decoded into
// Resp. Returning a non-nil error from onChunk finalizes the call with that
// error.
//
// The same deadline rules apply as Call. Use WithCallBuffer and
// WithCallBufferPolicy to tune back-pressure; default is a 64-slot buffer
// with BufferBlock.
func CallStream[Req sdk.BrainkitMessage, Chunk any, Resp any](
	k *Kit,
	ctx context.Context,
	req Req,
	onChunk func(Chunk) error,
	opts ...CallOption,
) (Resp, error) {
	var zero Resp
	if onChunk == nil {
		return zero, fmt.Errorf("brainkit.CallStream: onChunk is required")
	}
	cfg := callConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	if _, hasDeadline := ctx.Deadline(); !hasDeadline && cfg.timeout <= 0 {
		return zero, &caller.NoDeadlineError{}
	}
	if cfg.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.timeout)
		defer cancel()
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return zero, fmt.Errorf("brainkit.CallStream: marshal %T: %w", req, err)
	}

	c := k.Caller()
	if c == nil {
		return zero, fmt.Errorf("brainkit.CallStream: caller not initialized")
	}

	topic := req.BusTopic()
	streamH := func(msg sdk.Message) error {
		var chunk Chunk
		if rm, ok := any(&chunk).(*json.RawMessage); ok {
			*rm = append(json.RawMessage(nil), msg.Payload...)
			return onChunk(chunk)
		}
		if err := json.Unmarshal(msg.Payload, &chunk); err != nil {
			return &caller.DecodeError{Topic: topic, Payload: msg.Payload, Cause: err}
		}
		return onChunk(chunk)
	}

	replyPayload, err := c.Call(ctx, topic, payload, caller.Config{
		TargetNamespace: cfg.targetNS,
		Metadata:        cfg.meta,
		StreamHandler:   streamH,
		BufferSize:      cfg.bufferSize,
		BufferPolicy:    cfg.bufferPolicy,
		NoCancelSignal:  cfg.noCancelSignal,
	})
	if err != nil {
		return zero, err
	}

	if rm, ok := any(&zero).(*json.RawMessage); ok {
		*rm = replyPayload
		return zero, nil
	}

	var resp Resp
	if err := json.Unmarshal(replyPayload, &resp); err != nil {
		return zero, &caller.DecodeError{Topic: topic, Payload: replyPayload, Cause: err}
	}
	return resp, nil
}
