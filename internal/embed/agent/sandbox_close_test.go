package agentembed

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/jsbridge"
)

func TestSandboxCloseContextRetriesBridgeClose(t *testing.T) {
	bridge, err := jsbridge.New(jsbridge.Config{})
	if err != nil {
		t.Fatalf("new bridge: %v", err)
	}
	sandbox := &Sandbox{
		bridge: bridge,
		agents: map[string]*Agent{},
	}

	started := make(chan struct{})
	release := make(chan struct{})
	bridge.Go(func(context.Context) {
		close(started)
		<-release
	})
	<-started

	closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	err = sandbox.CloseContext(closeCtx)
	cancel()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CloseContext error = %v, want %v", err, context.DeadlineExceeded)
	}
	if !sandbox.closing || sandbox.closed {
		t.Fatalf("sandbox state after timed-out close: closing=%v closed=%v", sandbox.closing, sandbox.closed)
	}

	close(release)
	retryCtx, retryCancel := context.WithTimeout(context.Background(), time.Second)
	defer retryCancel()
	if err := sandbox.CloseContext(retryCtx); err != nil {
		t.Fatalf("retry CloseContext: %v", err)
	}
	if sandbox.closing || !sandbox.closed || sandbox.bridge != nil {
		t.Fatalf("sandbox state after successful retry: closing=%v closed=%v bridge=%v", sandbox.closing, sandbox.closed, sandbox.bridge)
	}
}
