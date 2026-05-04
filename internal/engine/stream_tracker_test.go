package engine

import (
	"context"
	"testing"
	"time"
)

func TestStreamTrackerCloseAllCancelsAndWaitsForHeartbeats(t *testing.T) {
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	defer shutdownCancel()

	k := &Kernel{
		shutdownCtx:    shutdownCtx,
		shutdownCancel: shutdownCancel,
	}
	st := newStreamTracker(k, time.Hour, time.Hour)

	st.StartHeartbeat("reply.close", "corr-close")
	if got := st.Active(); got != 1 {
		t.Fatalf("active heartbeats = %d, want 1", got)
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := st.CloseAll(closeCtx); err != nil {
		t.Fatalf("CloseAll returned error: %v", err)
	}
	if got := st.Active(); got != 0 {
		t.Fatalf("active heartbeats after CloseAll = %d, want 0", got)
	}
}

func TestStreamTrackerCloseAllRetriesWithSinglePendingHeartbeatWait(t *testing.T) {
	st := newStreamTracker(&Kernel{shutdownCtx: context.Background()}, time.Hour, time.Hour)
	st.mu.Lock()
	st.active["reply.blocked"] = func() {}
	st.wg.Add(1)
	st.mu.Unlock()

	for i := 0; i < 2; i++ {
		closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		err := st.CloseAll(closeCtx)
		cancel()
		if err != context.DeadlineExceeded {
			t.Fatalf("CloseAll attempt %d error = %v, want %v", i+1, err, context.DeadlineExceeded)
		}
		if got := st.Active(); got != 0 {
			t.Fatalf("active heartbeats after close attempt %d = %d, want 0", i+1, got)
		}
	}

	st.wg.Done()
	if err := st.CloseAll(context.Background()); err != nil {
		t.Fatalf("retry CloseAll: %v", err)
	}
}

func TestStreamTrackerMaxLifeSelfRemovesHeartbeat(t *testing.T) {
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	defer shutdownCancel()

	k := &Kernel{
		shutdownCtx:    shutdownCtx,
		shutdownCancel: shutdownCancel,
	}
	st := newStreamTracker(k, time.Hour, 20*time.Millisecond)

	st.StartHeartbeat("reply.maxlife", "corr-maxlife")
	if got := st.Active(); got != 1 {
		t.Fatalf("active heartbeats = %d, want 1", got)
	}

	deadline := time.After(time.Second)
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			t.Fatalf("active heartbeats after max life = %d, want 0", st.Active())
		case <-ticker.C:
			if st.Active() == 0 {
				return
			}
		}
	}
}
