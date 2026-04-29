package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// CallerRuntime is implemented by runtimes that expose a shared-inbox Caller.
type CallerRuntime interface {
	Runtime
	Caller() *Caller
}

// CallOption configures a typed Call or CallStream invocation.
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

// WithCallTo routes the call to a target namespace.
func WithCallTo(namespace string) CallOption {
	return func(c *callConfig) { c.targetNS = namespace }
}

// WithCallMeta adds metadata key/values to the published message.
func WithCallMeta(meta map[string]string) CallOption {
	return func(c *callConfig) { c.meta = meta }
}

// WithCallBuffer sets the per-pending stream channel capacity.
func WithCallBuffer(n int) CallOption {
	return func(c *callConfig) { c.bufferSize = n }
}

// WithCallBufferPolicy selects how the stream channel handles overflow.
func WithCallBufferPolicy(p BufferPolicy) CallOption {
	return func(c *callConfig) { c.bufferPolicy = p }
}

// WithCallNoCancelSignal disables the best-effort cancellation publish.
func WithCallNoCancelSignal() CallOption {
	return func(c *callConfig) { c.noCancelSignal = true }
}

// Call sends a typed request to the target topic and waits for a typed response.
func Call[Req BrainkitMessage, Resp any](rt CallerRuntime, ctx context.Context, req Req, opts ...CallOption) (Resp, error) {
	var zero Resp
	cfg := callConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline && cfg.timeout <= 0 {
		return zero, &NoDeadlineError{}
	}
	if cfg.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.timeout)
		defer cancel()
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return zero, fmt.Errorf("sdk.Call: marshal %T: %w", req, err)
	}
	c := rt.Caller()
	if c == nil {
		return zero, fmt.Errorf("sdk.Call: caller not initialized")
	}
	replyPayload, err := c.Call(ctx, req.BusTopic(), payload, CallerConfig{
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
		return zero, &CallDecodeError{Topic: req.BusTopic(), Payload: replyPayload, Cause: err}
	}
	return resp, nil
}

// CallStream sends a typed request, forwards intermediate chunks through
// onChunk, then returns the terminal reply decoded into Resp.
func CallStream[Req BrainkitMessage, Chunk any, Resp any](
	rt CallerRuntime,
	ctx context.Context,
	req Req,
	onChunk func(Chunk) error,
	opts ...CallOption,
) (Resp, error) {
	var zero Resp
	if onChunk == nil {
		return zero, fmt.Errorf("sdk.CallStream: onChunk is required")
	}
	cfg := callConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline && cfg.timeout <= 0 {
		return zero, &NoDeadlineError{}
	}
	if cfg.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.timeout)
		defer cancel()
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return zero, fmt.Errorf("sdk.CallStream: marshal %T: %w", req, err)
	}
	c := rt.Caller()
	if c == nil {
		return zero, fmt.Errorf("sdk.CallStream: caller not initialized")
	}
	topic := req.BusTopic()
	streamH := func(msg Message) error {
		var chunk Chunk
		if rm, ok := any(&chunk).(*json.RawMessage); ok {
			*rm = append(json.RawMessage(nil), msg.Payload...)
			return onChunk(chunk)
		}
		if err := json.Unmarshal(msg.Payload, &chunk); err != nil {
			return &CallDecodeError{Topic: topic, Payload: msg.Payload, Cause: err}
		}
		return onChunk(chunk)
	}
	replyPayload, err := c.Call(ctx, topic, payload, CallerConfig{
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
		return zero, &CallDecodeError{Topic: topic, Payload: replyPayload, Cause: err}
	}
	return resp, nil
}
