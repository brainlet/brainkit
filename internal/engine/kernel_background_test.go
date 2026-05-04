package engine

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestKernelCloseWaitsForBackgroundTasks(t *testing.T) {
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	k := &Kernel{
		shutdownCtx:    shutdownCtx,
		shutdownCancel: shutdownCancel,
	}

	started := make(chan struct{})
	var finished atomic.Bool
	k.startBackground(func(ctx context.Context) {
		close(started)
		<-ctx.Done()
		time.Sleep(20 * time.Millisecond)
		finished.Store(true)
	})

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("background task did not start")
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := k.close(closeCtx); err != nil {
		t.Fatalf("close returned error: %v", err)
	}
	if !finished.Load() {
		t.Fatal("close returned before background task finished")
	}
}

func TestKernelCloseReturnsContextErrorWhenBackgroundTaskIgnoresShutdown(t *testing.T) {
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	k := &Kernel{
		shutdownCtx:    shutdownCtx,
		shutdownCancel: shutdownCancel,
	}

	started := make(chan struct{})
	release := make(chan struct{})
	released := false
	t.Cleanup(func() {
		if !released {
			close(release)
		}
	})
	k.startBackground(func(context.Context) {
		close(started)
		<-release
	})

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("background task did not start")
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := k.close(closeCtx); err != context.DeadlineExceeded {
		t.Fatalf("close error = %v, want %v", err, context.DeadlineExceeded)
	}

	retryDeadlineCtx, retryDeadlineCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	if err := k.close(retryDeadlineCtx); err != context.DeadlineExceeded {
		t.Fatalf("second close error = %v, want %v", err, context.DeadlineExceeded)
	}
	retryDeadlineCancel()

	released = true
	close(release)
	retryCtx, retryCancel := context.WithTimeout(context.Background(), time.Second)
	defer retryCancel()
	if err := k.close(retryCtx); err != nil {
		t.Fatalf("retry close after background release: %v", err)
	}
}
