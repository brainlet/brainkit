package engine

import (
	"context"
	"sync"
	"time"

	"github.com/brainlet/brainkit/internal/syncx"
)

// streamTracker manages heartbeat goroutines for active stream replyTo topics.
// Started by bus_reply bridge on first done=false with type discriminator.
// Stopped on done=true or self-terminates after maxLife.
type streamTracker struct {
	mu        syncx.Mutex
	active    map[string]context.CancelFunc
	kernel    *Kernel
	interval  time.Duration
	maxLife   time.Duration
	wg        sync.WaitGroup
	closeWait sync.Once
	closeDone chan struct{}
}

func newStreamTracker(kernel *Kernel, interval, maxLife time.Duration) *streamTracker {
	if interval == 0 {
		interval = 10 * time.Second
	}
	if maxLife == 0 {
		maxLife = 10 * time.Minute
	}
	return &streamTracker{
		active:    make(map[string]context.CancelFunc),
		kernel:    kernel,
		interval:  interval,
		maxLife:   maxLife,
		closeDone: make(chan struct{}),
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
	ctx, cancel := context.WithTimeout(st.kernel.shutdownCtx, st.maxLife)
	st.active[replyTo] = cancel
	st.wg.Add(1)
	st.mu.Unlock()

	run := func(goCtx context.Context) {
		defer st.wg.Done()
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
				_ = st.kernel.transportHost.PublishReply(goCtx, replyTo, correlationID, []byte(`{"type":"heartbeat"}`), false, false)
			case <-ctx.Done():
				return
			case <-goCtx.Done():
				return
			}
		}
	}
	go run(st.kernel.shutdownCtx)
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
func (st *streamTracker) CloseAll(ctx context.Context) error {
	if st == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	st.mu.Lock()
	for replyTo, cancel := range st.active {
		cancel()
		delete(st.active, replyTo)
	}
	closeDone := st.closeDone
	if closeDone == nil {
		closeDone = make(chan struct{})
		st.closeDone = closeDone
	}
	st.closeWait.Do(func() {
		go func() {
			st.wg.Wait()
			close(closeDone)
		}()
	})
	st.mu.Unlock()
	select {
	case <-closeDone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Active reports active heartbeat goroutines for lifecycle diagnostics.
func (st *streamTracker) Active() int {
	if st == nil {
		return 0
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	return len(st.active)
}
