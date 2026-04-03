package brainkit

import (
	"context"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// streamTracker manages heartbeat goroutines for active stream replyTo topics.
// Started by bus_reply bridge on first done=false with type discriminator.
// Stopped on done=true or self-terminates after maxLife.
type streamTracker struct {
	mu       sync.Mutex
	active   map[string]context.CancelFunc
	kernel   *Kernel
	interval time.Duration
	maxLife  time.Duration
}

func newStreamTracker(kernel *Kernel, interval, maxLife time.Duration) *streamTracker {
	if interval == 0 {
		interval = 10 * time.Second
	}
	if maxLife == 0 {
		maxLife = 10 * time.Minute
	}
	return &streamTracker{
		active:   make(map[string]context.CancelFunc),
		kernel:   kernel,
		interval: interval,
		maxLife:  maxLife,
	}
}

// StartHeartbeat begins sending {"type":"heartbeat"} to replyTo every interval.
// Idempotent — no-op if already started for this replyTo.
// Self-terminates after maxLife as safety net.
func (st *streamTracker) StartHeartbeat(replyTo, correlationID string) {
	st.mu.Lock()
	if _, exists := st.active[replyTo]; exists {
		st.mu.Unlock()
		return
	}
	// Create a context that cancels on either StopHeartbeat or maxLife timeout.
	// Derived from bridge.GoContext() so it also cancels on bridge.Close().
	ctx, cancel := context.WithTimeout(st.kernel.bridge.GoContext(), st.maxLife)
	st.active[replyTo] = cancel
	st.mu.Unlock()

	st.kernel.bridge.Go(func(goCtx context.Context) {
		ticker := time.NewTicker(st.interval)
		defer ticker.Stop()
		// Self-remove from active map on exit — prevents map growth when
		// goroutines self-terminate via maxLife timeout or bridge close.
		defer func() {
			st.mu.Lock()
			delete(st.active, replyTo)
			st.mu.Unlock()
		}()
		for {
			select {
			case <-ticker.C:
				wmsg := message.NewMessage(watermill.NewUUID(), []byte(`{"type":"heartbeat"}`))
				wmsg.Metadata.Set("correlationId", correlationID)
				st.kernel.transport.Publisher.Publish(replyTo, wmsg)
			case <-ctx.Done():
				return
			case <-goCtx.Done():
				return
			}
		}
	})
}

// StopHeartbeat cancels the heartbeat goroutine for a replyTo topic.
func (st *streamTracker) StopHeartbeat(replyTo string) {
	st.mu.Lock()
	if cancel, ok := st.active[replyTo]; ok {
		cancel()
		delete(st.active, replyTo)
	}
	st.mu.Unlock()
}

// CloseAll cancels all active heartbeat goroutines. Called during Kernel.Close.
func (st *streamTracker) CloseAll() {
	st.mu.Lock()
	for replyTo, cancel := range st.active {
		cancel()
		delete(st.active, replyTo)
	}
	st.mu.Unlock()
}
