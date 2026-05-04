package schedules

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSchedulerCloseContextCancelsAndWaitsForInFlightFire(t *testing.T) {
	pub := &contextPublisher{
		started: make(chan struct{}),
		done:    make(chan struct{}),
	}
	s := newScheduler(pub, nil, nil, nil, nil, nil)

	if _, err := s.Schedule(context.Background(), ScheduleConfig{
		ID:         "tick",
		Expression: "every 1ms",
		Topic:      "events.tick",
		Payload:    json.RawMessage(`{}`),
	}); err != nil {
		t.Fatalf("schedule: %v", err)
	}

	select {
	case <-pub.started:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("scheduled fire did not start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.CloseContext(ctx); err != nil {
		t.Fatalf("close scheduler: %v", err)
	}
	select {
	case <-pub.done:
	default:
		t.Fatal("in-flight schedule publish was not joined")
	}
	if !errors.Is(pub.err(), context.Canceled) {
		t.Fatalf("publish context error = %v, want context canceled", pub.err())
	}
	if got := s.debugSnapshot(); got.activeFires != 0 || got.activeTimers != 0 || !got.closed {
		t.Fatalf("debug snapshot after close = %#v, want closed with no active fires/timers", got)
	}
}

func TestSchedulerCloseContextReturnsDeadlineForStuckFire(t *testing.T) {
	pub := &blockingPublisher{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	s := newScheduler(pub, nil, nil, nil, nil, nil)

	if _, err := s.Schedule(context.Background(), ScheduleConfig{
		ID:         "tick",
		Expression: "every 1ms",
		Topic:      "events.tick",
		Payload:    json.RawMessage(`{}`),
	}); err != nil {
		t.Fatalf("schedule: %v", err)
	}

	select {
	case <-pub.started:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("scheduled fire did not start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	closeErr := make(chan error, 1)
	go func() {
		closeErr <- s.CloseContext(ctx)
	}()
	waitScheduleCondition(t, func() bool {
		return s.debugSnapshot().closing
	}, "scheduler closing state")
	if err := <-closeErr; !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("close scheduler error = %v, want context deadline exceeded", err)
	}
	if got := s.debugSnapshot(); got.activeFires != 1 || got.activeTimers != 0 || !got.closed {
		t.Fatalf("debug snapshot after close deadline = %#v, want closed with one active fire and no timers", got)
	}
	close(pub.release)
	if err := s.CloseContext(context.Background()); err != nil {
		t.Fatalf("close scheduler after release: %v", err)
	}

	time.Sleep(25 * time.Millisecond)
	if got := pub.count.Load(); got != 1 {
		t.Fatalf("publish count after close = %d, want 1", got)
	}
}

func TestSchedulerCloseContextRetriesWithSinglePendingFireWait(t *testing.T) {
	pub := &blockingPublisher{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	s := newScheduler(pub, nil, nil, nil, nil, nil)

	if _, err := s.Schedule(context.Background(), ScheduleConfig{
		ID:         "tick",
		Expression: "every 1ms",
		Topic:      "events.tick",
		Payload:    json.RawMessage(`{}`),
	}); err != nil {
		t.Fatalf("schedule: %v", err)
	}

	select {
	case <-pub.started:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("scheduled fire did not start")
	}

	for i := 0; i < 2; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		err := s.CloseContext(ctx)
		cancel()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("close scheduler attempt %d error = %v, want context deadline exceeded", i+1, err)
		}
		if got := s.debugSnapshot(); got.activeFires != 1 || got.activeTimers != 0 || !got.closed {
			t.Fatalf("debug snapshot after close attempt %d = %#v, want closed with one active fire and no timers", i+1, got)
		}
	}

	close(pub.release)
	if err := s.CloseContext(context.Background()); err != nil {
		t.Fatalf("close scheduler after release: %v", err)
	}
	if got := s.debugSnapshot(); got.activeFires != 0 || got.activeTimers != 0 || !got.closed {
		t.Fatalf("debug snapshot after release = %#v, want closed with no active fires/timers", got)
	}
}

func waitScheduleCondition(t *testing.T, ok func() bool, label string) {
	t.Helper()
	deadline := time.After(time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %s", label)
		case <-ticker.C:
			if ok() {
				return
			}
		}
	}
}

func TestSchedulerDebugSnapshotReportsOwnedCounters(t *testing.T) {
	s := newScheduler(&blockingPublisher{}, nil, nil, nil, nil, nil)

	if _, err := s.Schedule(context.Background(), ScheduleConfig{
		ID:         "repeat",
		Expression: "every 1h",
		Topic:      "events.repeat",
		Payload:    json.RawMessage(`{}`),
	}); err != nil {
		t.Fatalf("schedule repeat: %v", err)
	}
	if _, err := s.Schedule(context.Background(), ScheduleConfig{
		ID:         "once",
		Expression: "in 1h",
		Topic:      "events.once",
		Payload:    json.RawMessage(`{}`),
	}); err != nil {
		t.Fatalf("schedule once: %v", err)
	}

	got := s.debugSnapshot()
	if got.configuredSchedules != 2 {
		t.Fatalf("configuredSchedules = %d, want 2", got.configuredSchedules)
	}
	if got.activeTimers != 2 {
		t.Fatalf("activeTimers = %d, want 2", got.activeTimers)
	}
	if got.oneTimeSchedules != 1 || got.repeatingSchedules != 1 {
		t.Fatalf("schedule kinds = one-time %d repeating %d, want 1 and 1", got.oneTimeSchedules, got.repeatingSchedules)
	}
	if got.closed {
		t.Fatalf("closed = true, want false")
	}

	if err := s.Close(); err != nil {
		t.Fatalf("close scheduler: %v", err)
	}
	got = s.debugSnapshot()
	if !got.closed {
		t.Fatalf("closed = false, want true")
	}
	if got.configuredSchedules != 0 || got.activeTimers != 0 {
		t.Fatalf("closed counters = configured %d timers %d, want 0 and 0", got.configuredSchedules, got.activeTimers)
	}
}

type contextPublisher struct {
	count   atomic.Int32
	once    sync.Once
	started chan struct{}
	done    chan struct{}
	errMu   sync.Mutex
	ctxErr  error
}

func (p *contextPublisher) PublishRaw(ctx context.Context, _ string, _ json.RawMessage) (string, error) {
	p.count.Add(1)
	p.once.Do(func() { close(p.started) })
	<-ctx.Done()
	p.errMu.Lock()
	p.ctxErr = ctx.Err()
	p.errMu.Unlock()
	close(p.done)
	return "", ctx.Err()
}

func (p *contextPublisher) err() error {
	p.errMu.Lock()
	defer p.errMu.Unlock()
	return p.ctxErr
}

type blockingPublisher struct {
	count   atomic.Int32
	once    sync.Once
	started chan struct{}
	release chan struct{}
}

func (p *blockingPublisher) PublishRaw(context.Context, string, json.RawMessage) (string, error) {
	p.count.Add(1)
	p.once.Do(func() {
		if p.started != nil {
			close(p.started)
		}
		if p.release != nil {
			<-p.release
		}
	})
	return "", nil
}
