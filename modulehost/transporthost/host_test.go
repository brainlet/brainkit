package transporthost

import (
	"errors"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/types"
)

func TestDebugSnapshotReportsClosingOwnedTransport(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	transportSet := transport.NewManagedTransport("test", nil, nil, nil, nil, func() error {
		close(started)
		<-release
		return nil
	})
	host := &Host{transport: transportSet, ownsTransport: true}

	done := make(chan error, 1)
	go func() {
		done <- host.CloseOwnedTransport()
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for owned transport close to start")
	}

	if got := host.DebugSnapshot(); !got.ClosingTransport {
		t.Fatalf("ClosingTransport = false, want true")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("CloseOwnedTransport: %v", err)
	}
	if got := host.DebugSnapshot(); got.ClosingTransport {
		t.Fatalf("ClosingTransport = true after close, want false")
	}
}

func TestDebugSnapshotReportsClosedTransportResources(t *testing.T) {
	host, err := New(types.KernelConfig{
		Namespace: "transporthost-closed-test",
		RuntimeID: "transporthost-closed-test",
		CallerID:  "transporthost-closed-test",
	}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer host.Close()

	if got := host.DebugSnapshot(); !got.OwnsTransport || got.ClosedRouter || got.ClosedCaller || got.ClosedTransport {
		t.Fatalf("initial snapshot = %#v, want owned open transport resources", got)
	}
	if err := host.CloseRouter(); err != nil {
		t.Fatalf("CloseRouter: %v", err)
	}
	if err := host.CloseCaller(); err != nil {
		t.Fatalf("CloseCaller: %v", err)
	}
	if err := host.CloseOwnedTransport(); err != nil {
		t.Fatalf("CloseOwnedTransport: %v", err)
	}

	got := host.DebugSnapshot()
	if !got.ClosedRouter || !got.ClosedCaller || !got.ClosedTransport {
		t.Fatalf("closed snapshot = %#v, want all transport resources closed", got)
	}
	if got.ClosingRouter || got.ClosingCaller || got.ClosingTransport {
		t.Fatalf("closed snapshot = %#v, want no close in progress", got)
	}
}

func TestDebugSnapshotDoesNotMarkTransportClosedOnCloseError(t *testing.T) {
	closeErr := errors.New("close failed")
	transportSet := transport.NewManagedTransport("test", nil, nil, nil, nil, func() error {
		return closeErr
	})
	host := &Host{transport: transportSet, ownsTransport: true}

	if err := host.CloseOwnedTransport(); !errors.Is(err, closeErr) {
		t.Fatalf("CloseOwnedTransport error = %v, want %v", err, closeErr)
	}
	got := host.DebugSnapshot()
	if got.ClosedTransport {
		t.Fatalf("ClosedTransport = true after failed close, want false")
	}
	if got.ClosingTransport {
		t.Fatalf("ClosingTransport = true after failed close, want false")
	}
}
