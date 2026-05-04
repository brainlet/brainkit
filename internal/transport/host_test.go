package transport

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestCommandHandleStopRetainsHandlerWhenStopTimesOut(t *testing.T) {
	router, err := NewRouter("test-caller", nil, 0)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	broker := newMemoryBroker()
	defer broker.Close()

	host := NewHost("test", router, broker, broker)
	entered := make(chan struct{})
	release := make(chan struct{})
	handle, err := host.RegisterCommand(context.Background(), RawCommandBinding{
		Name:  "blocking",
		Topic: "commands.block",
		Handle: func(context.Context, json.RawMessage) (json.RawMessage, error) {
			close(entered)
			<-release
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("RegisterCommand: %v", err)
	}

	runCtx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()
	go func() {
		if err := router.Run(runCtx); err != nil {
			t.Errorf("router.Run: %v", err)
		}
	}()
	t.Cleanup(func() {
		cancelRun()
		_ = router.Close()
	})

	select {
	case <-router.Running():
	case <-time.After(time.Second):
		t.Fatal("router did not start")
	}

	if err := broker.Publish(NamespacedTopic("test", "commands.block"), NewMessage([]byte(`{}`))); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("handler did not receive command")
	}

	stopCtx, cancelStop := context.WithTimeout(context.Background(), 20*time.Millisecond)
	err = handle.Stop(stopCtx)
	cancelStop()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Stop error = %v, want context deadline", err)
	}

	got := router.DebugSnapshot()
	if got.Handlers != 1 || got.StartedHandlers != 1 || got.StoppedHandlers != 0 {
		t.Fatalf("snapshot after timeout = %+v, want retained running handler", got)
	}

	close(release)
	retryCtx, cancelRetry := context.WithTimeout(context.Background(), time.Second)
	defer cancelRetry()
	if err := handle.Stop(retryCtx); err != nil {
		t.Fatalf("retry Stop: %v", err)
	}

	got = router.DebugSnapshot()
	if got.Handlers != 0 || got.StartedHandlers != 0 {
		t.Fatalf("snapshot after successful retry = %+v, want removed handler", got)
	}
}
