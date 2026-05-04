package transport

import (
	"context"
	"fmt"
	"sync"
)

// Router dispatches subscribed transport messages through Brainkit middleware.
type Router struct {
	mu              sync.Mutex
	handlers        []*Handler
	middleware      []Middleware
	running         chan struct{}
	runningOnce     sync.Once
	closeOnce       sync.Once
	closed          chan struct{}
	runCtx          context.Context
	runCancel       context.CancelFunc
	startedHandlers map[*Handler]struct{}
}

// RouterDebugSnapshot is a test/debug view of router bookkeeping.
type RouterDebugSnapshot struct {
	Handlers        int
	StartedHandlers int
	StoppedHandlers int
	Topics          map[string]int
}

// NewRouter builds a message router with Brainkit's core middleware stack.
func NewRouter(callerID string, metrics *Metrics, maxConcurrency int) (*Router, error) {
	router := &Router{
		running:         make(chan struct{}),
		closed:          make(chan struct{}),
		startedHandlers: make(map[*Handler]struct{}),
	}
	router.AddMiddleware(
		DepthMiddleware,
		CallerIDMiddleware(callerID),
		MetricsMiddleware(metrics),
	)
	if maxConcurrency > 0 {
		router.AddMiddleware(MaxConcurrencyMiddleware(maxConcurrency))
	}
	return router, nil
}

// AddMiddleware appends middleware to handlers registered on this router.
func (r *Router) AddMiddleware(middleware ...Middleware) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.middleware = append(r.middleware, middleware...)
	r.mu.Unlock()
}

func (r *Router) addConsumerHandler(handlerName, topic string, sub Subscriber, fn NoPublishHandlerFunc) *Handler {
	return r.addConsumerHandlerWithMetricTopic(handlerName, topic, topic, sub, fn)
}

func (r *Router) addConsumerHandlerWithMetricTopic(handlerName, topic, metricTopic string, sub Subscriber, fn NoPublishHandlerFunc) *Handler {
	handler := &Handler{
		name:        handlerName,
		topic:       topic,
		metricTopic: metricTopic,
		sub:         sub,
		fn:          fn,
		started:     make(chan struct{}),
		stopped:     make(chan struct{}),
	}
	r.mu.Lock()
	r.handlers = append(r.handlers, handler)
	r.mu.Unlock()
	return handler
}

func (r *Router) removeHandler(handler *Handler) {
	if r == nil || handler == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.startedHandlers, handler)
	for i, candidate := range r.handlers {
		if candidate != handler {
			continue
		}
		copy(r.handlers[i:], r.handlers[i+1:])
		r.handlers[len(r.handlers)-1] = nil
		r.handlers = r.handlers[:len(r.handlers)-1]
		return
	}
}

// DebugSnapshot returns router bookkeeping counts for lifecycle tests.
func (r *Router) DebugSnapshot() RouterDebugSnapshot {
	if r == nil {
		return RouterDebugSnapshot{}
	}
	r.mu.Lock()
	handlers := append([]*Handler(nil), r.handlers...)
	started := len(r.startedHandlers)
	r.mu.Unlock()

	snapshot := RouterDebugSnapshot{
		Handlers:        len(handlers),
		StartedHandlers: started,
		Topics:          make(map[string]int),
	}
	for _, handler := range handlers {
		if handler == nil {
			continue
		}
		snapshot.Topics[handler.topic]++
		if handler.IsStopped() {
			snapshot.StoppedHandlers++
		}
	}
	if len(snapshot.Topics) == 0 {
		snapshot.Topics = nil
	}
	return snapshot
}

// IsRunning reports whether the router has started its handlers.
func (r *Router) IsRunning() bool {
	if r == nil {
		return false
	}
	select {
	case <-r.running:
		return true
	default:
		return false
	}
}

// Run starts the router.
func (r *Router) Run(ctx context.Context) error {
	if r == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	r.mu.Lock()
	if r.runCtx == nil {
		r.runCtx, r.runCancel = context.WithCancel(ctx)
	}
	r.mu.Unlock()

	if err := r.RunHandlers(ctx); err != nil {
		return err
	}
	r.runningOnce.Do(func() { close(r.running) })

	select {
	case <-r.runCtx.Done():
		return nil
	case <-r.closed:
		return nil
	case <-ctx.Done():
		return nil
	}
}

// RunHandlers starts handlers added after the router has already started.
func (r *Router) RunHandlers(ctx context.Context) error {
	if r == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	r.mu.Lock()
	runCtx := r.runCtx
	if runCtx == nil {
		runCtx = ctx
	}
	middleware := append([]Middleware(nil), r.middleware...)
	var toStart []*Handler
	for _, handler := range r.handlers {
		if _, ok := r.startedHandlers[handler]; ok {
			continue
		}
		r.startedHandlers[handler] = struct{}{}
		toStart = append(toStart, handler)
	}
	r.mu.Unlock()

	for _, handler := range toStart {
		handler.start(runCtx, middleware)
	}
	for _, handler := range toStart {
		select {
		case <-handler.Started():
			if err := handler.startErr(); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// Running returns a channel closed once handlers are subscribed.
func (r *Router) Running() <-chan struct{} {
	if r == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return r.running
}

// Close stops the router.
func (r *Router) Close() error {
	if r == nil {
		return nil
	}
	r.closeOnce.Do(func() {
		r.mu.Lock()
		if r.runCancel != nil {
			r.runCancel()
		}
		handlers := append([]*Handler(nil), r.handlers...)
		r.mu.Unlock()
		for _, handler := range handlers {
			handler.Stop()
		}
		close(r.closed)
	})
	return nil
}

// Handler is a running router subscription.
type Handler struct {
	name        string
	topic       string
	metricTopic string
	sub         Subscriber
	fn          NoPublishHandlerFunc

	started chan struct{}
	stopped chan struct{}

	startOnce sync.Once
	stopOnce  sync.Once
	cancel    context.CancelFunc

	mu  sync.Mutex
	err error
}

func (h *Handler) start(parent context.Context, middleware []Middleware) {
	h.startOnce.Do(func() {
		ctx, cancel := context.WithCancel(parent)
		h.cancel = cancel
		go h.run(ctx, middleware)
	})
}

func (h *Handler) run(ctx context.Context, middleware []Middleware) {
	defer close(h.stopped)

	ch, err := h.sub.Subscribe(ctx, h.topic)
	if err != nil {
		h.setErr(fmt.Errorf("handler %s subscribe %s: %w", h.name, h.topic, err))
		close(h.started)
		return
	}
	close(h.started)

	handler := HandlerFunc(func(msg *Message) ([]*Message, error) {
		if h.fn == nil {
			return nil, nil
		}
		return nil, h.fn(msg)
	})
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			metricTopic := h.metricTopic
			if metricTopic == "" {
				metricTopic = h.topic
			}
			msg.Metadata.Set("_subscription_topic", metricTopic)
			if _, err := handler(msg); err != nil {
				msg.Nack()
				continue
			}
			msg.Ack()
		}
	}
}

func (h *Handler) setErr(err error) {
	h.mu.Lock()
	h.err = err
	h.mu.Unlock()
}

func (h *Handler) startErr() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.err
}

// Started returns a channel closed once the subscription is active.
func (h *Handler) Started() <-chan struct{} {
	return h.started
}

// Stopped returns a channel closed once the handler exits.
func (h *Handler) Stopped() <-chan struct{} {
	return h.stopped
}

// IsStopped reports whether the handler goroutine has exited.
func (h *Handler) IsStopped() bool {
	if h == nil {
		return true
	}
	select {
	case <-h.stopped:
		return true
	default:
		return false
	}
}

// Stop cancels this handler.
func (h *Handler) Stop() {
	if h == nil {
		return
	}
	h.stopOnce.Do(func() {
		if h.cancel != nil {
			h.cancel()
		}
	})
}
