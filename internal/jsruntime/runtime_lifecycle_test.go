package jsruntime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/brainlet/brainkit/modulehost/resourcehost"
)

func TestRuntimeUnmountReturnsStrictCleanupError(t *testing.T) {
	want := errors.New("restore configured registry failed")
	var gotMode runtimeCloseMode
	called := false
	r := &Runtime{
		cleanup: func(_ context.Context, mode runtimeCloseMode) error {
			called = true
			gotMode = mode
			return want
		},
	}

	err := r.Unmount(context.Background())
	if !errors.Is(err, want) {
		t.Fatalf("Unmount error = %v, want %v", err, want)
	}
	if !called {
		t.Fatal("cleanup was not called")
	}
	if gotMode != runtimeCloseModeUnmount {
		t.Fatalf("cleanup mode = %v, want unmount", gotMode)
	}
	snapshot := r.JSRuntimeDebugSnapshot()
	if snapshot.Closed || snapshot.Phase != "closing" {
		t.Fatalf("snapshot after failed strict unmount = %+v, want closing and not closed", snapshot)
	}
}

func TestRuntimeUnmountRetriesAfterCleanupDeadline(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	calls := 0
	r := &Runtime{
		cleanup: func(ctx context.Context, mode runtimeCloseMode) error {
			calls++
			if calls == 1 {
				close(started)
			}
			if mode != runtimeCloseModeUnmount {
				t.Fatalf("cleanup mode = %v, want unmount", mode)
			}
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	err := r.Unmount(closeCtx)
	cancel()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Unmount error = %v, want %v", err, context.DeadlineExceeded)
	}
	<-started
	if r.closed || !r.closing {
		t.Fatalf("runtime state after timed-out unmount: closing=%v closed=%v", r.closing, r.closed)
	}

	close(release)
	retryCtx, retryCancel := context.WithTimeout(context.Background(), time.Second)
	defer retryCancel()
	if err := r.Unmount(retryCtx); err != nil {
		t.Fatalf("retry Unmount: %v", err)
	}
	if !r.closed || r.closing {
		t.Fatalf("runtime state after successful retry: closing=%v closed=%v", r.closing, r.closed)
	}
	if calls != 2 {
		t.Fatalf("cleanup calls = %d, want 2", calls)
	}
}

func TestRuntimeShutdownIgnoresBestEffortCleanupError(t *testing.T) {
	want := errors.New("shutdown cleanup failed")
	var gotMode runtimeCloseMode
	called := false
	r := &Runtime{
		cleanup: func(_ context.Context, mode runtimeCloseMode) error {
			called = true
			gotMode = mode
			return want
		},
	}

	if err := r.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown error = %v, want nil", err)
	}
	if !called {
		t.Fatal("cleanup was not called")
	}
	if gotMode != runtimeCloseModeShutdown {
		t.Fatalf("cleanup mode = %v, want shutdown", gotMode)
	}
}

func TestDeploymentCleanupReturnsScheduleErrors(t *testing.T) {
	want := errors.New("unschedule failed")
	m := &DeploymentManager{
		scheduleCleanup: func(string) error {
			return want
		},
	}

	err := m.dispatchCleanups([]resourcehost.Entry{{Type: "schedule", ID: "nightly"}})
	if !errors.Is(err, want) {
		t.Fatalf("dispatchCleanups error = %v, want %v", err, want)
	}
}
