// Ported from: packages/core/src/workspace/lifecycle.test.ts
package workspace

import (
	"errors"
	"testing"
)

func TestProviderStatus(t *testing.T) {
	t.Run("has correct status values", func(t *testing.T) {
		cases := []struct {
			status ProviderStatus
			want   string
		}{
			{ProviderStatusPending, "pending"},
			{ProviderStatusInitializing, "initializing"},
			{ProviderStatusReady, "ready"},
			{ProviderStatusStarting, "starting"},
			{ProviderStatusRunning, "running"},
			{ProviderStatusStopping, "stopping"},
			{ProviderStatusStopped, "stopped"},
			{ProviderStatusDestroying, "destroying"},
			{ProviderStatusDestroyed, "destroyed"},
			{ProviderStatusError, "error"},
		}
		for _, tc := range cases {
			if string(tc.status) != tc.want {
				t.Errorf("status = %q, want %q", tc.status, tc.want)
			}
		}
	})
}

// Test providers for CallLifecycle.

// mockStatus is a helper that returns a ready status for all mock providers.
// All mock providers embed this to satisfy the LifecycleProvider.Status() requirement.
type mockStatus struct{}

func (m *mockStatus) Status() ProviderStatus { return ProviderStatusReady }

type mockInitProvider struct {
	mockStatus
	initCalled bool
}

func (m *mockInitProvider) Init() error {
	m.initCalled = true
	return nil
}

type mockWrappedInitProvider struct {
	mockStatus
	wrappedInitCalled bool
	plainInitCalled   bool
}

func (m *mockWrappedInitProvider) _Init() error {
	m.wrappedInitCalled = true
	return nil
}

func (m *mockWrappedInitProvider) Init() error {
	m.plainInitCalled = true
	return nil
}

type mockStartProvider struct {
	mockStatus
	startCalled bool
}

func (m *mockStartProvider) Start() error {
	m.startCalled = true
	return nil
}

type mockStopProvider struct {
	mockStatus
	stopCalled bool
}

func (m *mockStopProvider) Stop() error {
	m.stopCalled = true
	return nil
}

type mockDestroyProvider struct {
	mockStatus
	destroyCalled bool
}

func (m *mockDestroyProvider) Destroy() error {
	m.destroyCalled = true
	return nil
}

type mockErrorProvider struct {
	mockStatus
}

func (m *mockErrorProvider) Init() error {
	return errors.New("init failed")
}

type mockEmptyProvider struct {
	mockStatus
}

func TestCallLifecycle(t *testing.T) {
	t.Run("calls Init on plain provider", func(t *testing.T) {
		p := &mockInitProvider{}
		err := CallLifecycle(p, "init")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !p.initCalled {
			t.Error("Init should have been called")
		}
	})

	t.Run("prefers _Init over Init", func(t *testing.T) {
		p := &mockWrappedInitProvider{}
		err := CallLifecycle(p, "init")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !p.wrappedInitCalled {
			t.Error("_Init should have been called")
		}
		if p.plainInitCalled {
			t.Error("Init should NOT have been called when _Init is available")
		}
	})

	t.Run("calls Start on provider", func(t *testing.T) {
		p := &mockStartProvider{}
		err := CallLifecycle(p, "start")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !p.startCalled {
			t.Error("Start should have been called")
		}
	})

	t.Run("calls Stop on provider", func(t *testing.T) {
		p := &mockStopProvider{}
		err := CallLifecycle(p, "stop")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !p.stopCalled {
			t.Error("Stop should have been called")
		}
	})

	t.Run("calls Destroy on provider", func(t *testing.T) {
		p := &mockDestroyProvider{}
		err := CallLifecycle(p, "destroy")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !p.destroyCalled {
			t.Error("Destroy should have been called")
		}
	})

	t.Run("returns error from lifecycle method", func(t *testing.T) {
		p := &mockErrorProvider{}
		err := CallLifecycle(p, "init")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "init failed" {
			t.Errorf("error = %q, want %q", err.Error(), "init failed")
		}
	})

	t.Run("returns nil for provider without matching method", func(t *testing.T) {
		p := &mockEmptyProvider{}
		err := CallLifecycle(p, "init")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns nil for unknown method", func(t *testing.T) {
		p := &mockInitProvider{}
		err := CallLifecycle(p, "unknown")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
