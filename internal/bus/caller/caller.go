package caller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/ctxkeys"
	"github.com/google/uuid"
)

// BufferPolicy controls how the Caller handles stream chunks when a user
// callback lags behind producers and the per-pending buffer fills.
type BufferPolicy int

const (
	// BufferBlock (zero value) makes the transport callback block until the
	// consumer drains a slot. Applies back-pressure to the publisher.
	BufferBlock BufferPolicy = iota

	// BufferDropNewest silently drops the incoming chunk when the buffer
	// is full. Oldest chunks survive.
	BufferDropNewest

	// BufferDropOldest evicts the oldest buffered chunk to make room for
	// the newest. Newest chunks survive.
	BufferDropOldest

	// BufferError finalizes the call with a *BufferOverflowError when the
	// buffer is full.
	BufferError
)

// DefaultBufferSize is the default per-pending stream channel capacity.
const DefaultBufferSize = 64

// CancelTopic is the logical topic used for best-effort cancellation
// notices. `.ts` and Go handlers opt in by subscribing to it and
// matching on correlationID.
const CancelTopic = "_brainkit.cancel"

// CancelNotice is the payload published to CancelTopic when a Call's
// ctx is cancelled before a terminal reply arrives.
type CancelNotice struct {
	CorrelationID string `json:"correlationId"`
	Topic         string `json:"topic"`
	Reason        string `json:"reason"`
}

// Caller is the shared-inbox reply router for one Kit.
type Caller struct {
	rt      sdk.Runtime
	inbox   string
	pending sync.Map // correlationID → *pendingCall
	unsub   func()
	logger  *slog.Logger
	closed  atomic.Bool
	metrics Metrics
}

// Metrics exposes counters for observability. Read via Snapshot().
type Metrics struct {
	Inflight        atomic.Int64
	Completed       atomic.Int64
	TimedOut        atomic.Int64
	Cancelled       atomic.Int64
	Unmatched       atomic.Int64
	DecodeErrs      atomic.Int64
	BufferOverflows atomic.Int64
	ChunksDelivered atomic.Int64
	ChunksDropped   atomic.Int64
	FailedFast      atomic.Int64
}

// MetricsSnapshot is a point-in-time copy of Metrics.
type MetricsSnapshot struct {
	Inflight        int64
	Completed       int64
	TimedOut        int64
	Cancelled       int64
	Unmatched       int64
	DecodeErrs      int64
	BufferOverflows int64
	ChunksDelivered int64
	ChunksDropped   int64
	FailedFast      int64
}

type pendingCall struct {
	correlationID string
	topic         string
	done          chan result
	doneOnce      sync.Once
	ctx           context.Context

	// Streaming fields — zero when no StreamHandler was configured.
	stream    chan sdk.Message
	streamH   func(sdk.Message) error
	policy    BufferPolicy
	metrics   *Metrics
	drainDone chan struct{} // closed by drain when it exits

	// sendMu serializes stream sends with finalize's close so enqueue
	// never panics with "send on closed channel" after finalize fires.
	sendMu    sync.Mutex
	finalized atomic.Bool

	// envelope is set when the terminal reply arrived with
	// metadata["envelope"]="true" and Call should decode it as an
	// sdk.Envelope rather than return raw bytes.
	envelope bool
}

type result struct {
	payload json.RawMessage
	err     error
}

// Config configures a single Call invocation.
type Config struct {
	// TargetNamespace routes the call cross-namespace via
	// sdk.CrossNamespaceRuntime. Empty = publish in local namespace.
	TargetNamespace string

	// Metadata is added to the published message's metadata.
	Metadata map[string]string

	// StreamHandler, when non-nil, marks this call as streaming.
	StreamHandler func(sdk.Message) error

	// BufferSize is the per-pending stream channel capacity. 0 =
	// DefaultBufferSize. Ignored when StreamHandler is nil.
	BufferSize int

	// BufferPolicy selects behaviour when a chunk arrives with a full
	// buffer. Defaults to BufferBlock.
	BufferPolicy BufferPolicy

	// NoCancelSignal disables the best-effort `_brainkit.cancel` publish
	// that normally fires when ctx is cancelled before a terminal reply.
	NoCancelSignal bool
}

// NewCaller creates a Caller bound to rt's inbox topic derived from
// runtimeID. Subscribes to the inbox and to `bus.handler.exhausted` for
// fail-fast shortcutting. Callers must Close() to release both.
func NewCaller(rt sdk.Runtime, runtimeID string, logger *slog.Logger) (*Caller, error) {
	if runtimeID == "" {
		return nil, fmt.Errorf("caller: runtimeID is required")
	}
	return NewCallerWithInbox(rt, fmt.Sprintf("_brainkit.inbox.%s", runtimeID), logger)
}

// NewCallerWithInbox creates a Caller bound to an explicit inbox
// topic — used by the plugin SDK, which builds
// `_brainkit.plugin-inbox.<owner>.<name>` instead of the Kit's
// `_brainkit.inbox.<runtimeID>` scheme.
func NewCallerWithInbox(rt sdk.Runtime, inbox string, logger *slog.Logger) (*Caller, error) {
	if rt == nil {
		return nil, fmt.Errorf("caller: rt is required")
	}
	if inbox == "" {
		return nil, fmt.Errorf("caller: inbox is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	c := &Caller{
		rt:     rt,
		inbox:  inbox,
		logger: logger,
	}
	inboxUnsub, err := rt.SubscribeRaw(context.Background(), c.inbox, c.onInbox)
	if err != nil {
		return nil, fmt.Errorf("caller: subscribe %s: %w", c.inbox, err)
	}
	failUnsub, err := rt.SubscribeRaw(context.Background(), "bus.handler.exhausted", c.onFailure)
	if err != nil {
		inboxUnsub()
		return nil, fmt.Errorf("caller: subscribe bus.handler.exhausted: %w", err)
	}
	c.unsub = func() { inboxUnsub(); failUnsub() }
	return c, nil
}

// Inbox returns the logical inbox topic this Caller listens on.
func (c *Caller) Inbox() string { return c.inbox }

// Snapshot returns a copy of the current metric counters.
func (c *Caller) Snapshot() MetricsSnapshot {
	return MetricsSnapshot{
		Inflight:        c.metrics.Inflight.Load(),
		Completed:       c.metrics.Completed.Load(),
		TimedOut:        c.metrics.TimedOut.Load(),
		Cancelled:       c.metrics.Cancelled.Load(),
		Unmatched:       c.metrics.Unmatched.Load(),
		DecodeErrs:      c.metrics.DecodeErrs.Load(),
		BufferOverflows: c.metrics.BufferOverflows.Load(),
		ChunksDelivered: c.metrics.ChunksDelivered.Load(),
		ChunksDropped:   c.metrics.ChunksDropped.Load(),
		FailedFast:      c.metrics.FailedFast.Load(),
	}
}

// Close unsubscribes the inbox and finalizes all in-flight calls with
// ErrCallerClosed. Safe to call multiple times.
func (c *Caller) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}
	if c.unsub != nil {
		c.unsub()
	}
	c.pending.Range(func(_, v any) bool {
		v.(*pendingCall).finalize(result{err: ErrCallerClosed})
		return true
	})
	return nil
}

// Call sends payload to topic and waits for the terminal reply.
func (c *Caller) Call(ctx context.Context, topic string, payload json.RawMessage, cfg Config) (json.RawMessage, error) {
	if c.closed.Load() {
		return nil, ErrCallerClosed
	}

	cid := uuid.NewString()
	p := &pendingCall{
		correlationID: cid,
		topic:         topic,
		done:          make(chan result, 1),
		ctx:           ctx,
		policy:        cfg.BufferPolicy,
		metrics:       &c.metrics,
	}
	if cfg.StreamHandler != nil {
		size := cfg.BufferSize
		if size <= 0 {
			size = DefaultBufferSize
		}
		p.stream = make(chan sdk.Message, size)
		p.streamH = cfg.StreamHandler
		p.drainDone = make(chan struct{})
		go p.drain()
	}

	c.pending.Store(cid, p)
	c.metrics.Inflight.Add(1)
	defer func() {
		c.pending.Delete(cid)
		c.metrics.Inflight.Add(-1)
	}()

	pubCtx := ctxkeys.WithPublishMeta(ctx, cid, c.inbox)
	start := time.Now()

	var err error
	if cfg.TargetNamespace != "" {
		xrt, ok := c.rt.(sdk.CrossNamespaceRuntime)
		if !ok {
			return nil, fmt.Errorf("caller: cross-namespace call requires CrossNamespaceRuntime")
		}
		_, err = xrt.PublishRawTo(pubCtx, cfg.TargetNamespace, topic, payload)
	} else {
		_, err = c.rt.PublishRaw(pubCtx, topic, payload)
	}
	if err != nil {
		return nil, err
	}

	select {
	case r := <-p.done:
		if p.drainDone != nil {
			<-p.drainDone
		}
		if r.err != nil {
			// Fail-fast via HandlerFailedError is counted here too.
			var hf *HandlerFailedError
			if errors.As(r.err, &hf) {
				c.metrics.FailedFast.Add(1)
			}
			return nil, r.err
		}
		// Envelope unwrap: only when the reply was stamped with
		// metadata["envelope"]="true" by a producer that speaks the wire
		// envelope (transport/host.go, .ts bus.js). Raw replies pass
		// through untouched so tests that roll their own reply shape keep
		// working during the migration.
		if p.envelope {
			env, derr := sdk.DecodeEnvelope(r.payload)
			if derr != nil {
				c.metrics.DecodeErrs.Add(1)
				return nil, &DecodeError{Topic: topic, Payload: r.payload, Cause: derr}
			}
			if !env.Ok {
				if e := sdk.FromEnvelope(env); e != nil {
					return nil, e
				}
			}
			c.metrics.Completed.Add(1)
			return env.Data, nil
		}
		c.metrics.Completed.Add(1)
		return r.payload, nil
	case <-ctx.Done():
		elapsed := time.Since(start)
		// Best-effort cancel notice — let remote handlers that opted
		// into `_brainkit.cancel` stop wasted work.
		if !cfg.NoCancelSignal {
			c.emitCancel(cid, topic, ctx.Err())
		}
		if ctx.Err() == context.DeadlineExceeded {
			c.metrics.TimedOut.Add(1)
			return nil, &CallTimeoutError{Topic: topic, Elapsed: elapsed}
		}
		c.metrics.Cancelled.Add(1)
		return nil, &CallCancelledError{Topic: topic, Cause: ctx.Err()}
	}
}

func (c *Caller) onInbox(msg sdk.Message) {
	cid := ""
	done := false
	if msg.Metadata != nil {
		cid = msg.Metadata["correlationId"]
		done = msg.Metadata["done"] == "true"
	}
	if cid == "" {
		c.metrics.Unmatched.Add(1)
		return
	}

	if done {
		v, ok := c.pending.LoadAndDelete(cid)
		if !ok {
			c.metrics.Unmatched.Add(1)
			return
		}
		p := v.(*pendingCall)
		if msg.Metadata["envelope"] == "true" {
			p.envelope = true
		}
		p.finalize(result{payload: json.RawMessage(msg.Payload)})
		return
	}

	v, ok := c.pending.Load(cid)
	if !ok {
		c.metrics.Unmatched.Add(1)
		return
	}
	p := v.(*pendingCall)
	if p.stream == nil {
		c.metrics.ChunksDropped.Add(1)
		return
	}
	p.enqueue(msg)
}

// onFailure handles `bus.handler.exhausted` events. When an event carries
// a correlationID that matches a pending call, finalize it with
// *HandlerFailedError so the caller shortcuts out of its wait.
func (c *Caller) onFailure(msg sdk.Message) {
	cid := ""
	if msg.Metadata != nil {
		cid = msg.Metadata["correlationId"]
	}
	if cid == "" {
		return
	}
	v, ok := c.pending.LoadAndDelete(cid)
	if !ok {
		return
	}
	p := v.(*pendingCall)
	var evt sdk.HandlerExhaustedEvent
	_ = json.Unmarshal(msg.Payload, &evt)
	cause := errors.New(evt.Error)
	if evt.Error == "" {
		cause = fmt.Errorf("handler exhausted")
	}
	p.finalize(result{err: &HandlerFailedError{
		Topic:   evt.Topic,
		Retries: evt.RetryCount,
		Cause:   cause,
	}})
}

// emitCancel publishes a best-effort cancellation notice on a detached
// context so an already-cancelled parent doesn't block the emit.
func (c *Caller) emitCancel(cid, topic string, cause error) {
	reason := ""
	if cause != nil {
		reason = cause.Error()
	}
	payload, err := json.Marshal(CancelNotice{
		CorrelationID: cid,
		Topic:         topic,
		Reason:        reason,
	})
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if _, err := c.rt.PublishRaw(ctx, CancelTopic, payload); err != nil {
		c.logger.Warn("caller: emit cancel",
			slog.String("topic", topic),
			slog.String("error", err.Error()))
	}
}

// enqueue routes a chunk onto the stream channel according to policy.
func (p *pendingCall) enqueue(msg sdk.Message) {
	p.sendMu.Lock()
	defer p.sendMu.Unlock()
	if p.finalized.Load() {
		p.metrics.ChunksDropped.Add(1)
		return
	}
	switch p.policy {
	case BufferBlock:
		select {
		case p.stream <- msg:
		case <-p.ctx.Done():
			p.metrics.ChunksDropped.Add(1)
		}
	case BufferDropNewest:
		select {
		case p.stream <- msg:
		default:
			p.metrics.ChunksDropped.Add(1)
		}
	case BufferDropOldest:
		for {
			select {
			case p.stream <- msg:
				return
			default:
				select {
				case <-p.stream:
					p.metrics.ChunksDropped.Add(1)
				default:
				}
			}
		}
	case BufferError:
		select {
		case p.stream <- msg:
		default:
			p.metrics.BufferOverflows.Add(1)
			p.finalizeLocked(result{err: &BufferOverflowError{
				CorrelationID: p.correlationID,
				Topic:         p.topic,
			}})
		}
	}
}

// drain feeds the user's stream handler from the bounded channel until it
// closes. If the handler returns an error, finalize with that error and
// drain remaining items so the producer doesn't block.
func (p *pendingCall) drain() {
	defer close(p.drainDone)
	for msg := range p.stream {
		if err := p.streamH(msg); err != nil {
			p.finalize(result{err: err})
			for range p.stream {
			}
			return
		}
		p.metrics.ChunksDelivered.Add(1)
	}
}

// finalize resolves the pending call. Safe to call multiple times and from
// any goroutine.
func (p *pendingCall) finalize(r result) {
	p.doneOnce.Do(func() {
		p.sendMu.Lock()
		p.finalized.Store(true)
		if p.stream != nil {
			close(p.stream)
		}
		p.sendMu.Unlock()
		p.done <- r
	})
}

// finalizeLocked is finalize's body for callers that already hold sendMu.
func (p *pendingCall) finalizeLocked(r result) {
	p.doneOnce.Do(func() {
		p.finalized.Store(true)
		if p.stream != nil {
			close(p.stream)
		}
		p.done <- r
	})
}
