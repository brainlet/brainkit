package engine

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/brainlet/brainkit/internal/types"
	runtimecap "github.com/brainlet/brainkit/modulecap/runtime"
)

type fakeJSAttachment struct {
	unmountErr  error
	shutdownErr error
	closeErr    error

	unmounts   int
	shutdowns  int
	closes     int
	interrupts int

	currentSource string
	order         int
	seed          int32
}

func (f *fakeJSAttachment) Deploy(context.Context, string, string, ...types.DeployOption) ([]types.ResourceInfo, error) {
	return nil, nil
}
func (f *fakeJSAttachment) Teardown(context.Context, string) (int, error) { return 0, nil }
func (f *fakeJSAttachment) ListDeployments() []runtimecap.DeploymentInfo  { return nil }
func (f *fakeJSAttachment) EvalTS(context.Context, string, string) (string, error) {
	return "", nil
}
func (f *fakeJSAttachment) EvalModule(context.Context, string, string) (string, error) {
	return "", nil
}
func (f *fakeJSAttachment) ListResources(...string) ([]types.ResourceInfo, error) {
	return nil, nil
}
func (f *fakeJSAttachment) ResourcesFrom(string) ([]types.ResourceInfo, error) { return nil, nil }
func (f *fakeJSAttachment) TeardownFile(string) (int, error)                   { return 0, nil }
func (f *fakeJSAttachment) RemoveResource(string, string) error                { return nil }
func (f *fakeJSAttachment) CallJS(context.Context, string, any) (json.RawMessage, error) {
	return nil, nil
}
func (f *fakeJSAttachment) HarnessRuntime() any { return nil }
func (f *fakeJSAttachment) Interrupt()          { f.interrupts++ }
func (f *fakeJSAttachment) Unmount(context.Context) error {
	f.unmounts++
	return f.unmountErr
}
func (f *fakeJSAttachment) Shutdown(context.Context) error {
	f.shutdowns++
	return f.shutdownErr
}
func (f *fakeJSAttachment) Close() error {
	f.closes++
	return f.closeErr
}
func (f *fakeJSAttachment) CurrentSource() string { return f.currentSource }
func (f *fakeJSAttachment) SetCurrentSource(source string) {
	f.currentSource = source
}
func (f *fakeJSAttachment) NextDeployOrder() int {
	f.order++
	return f.order
}
func (f *fakeJSAttachment) SetDeployOrderSeed(seed int32) { f.seed = seed }

func TestDisableJSRuntimeUsesStrictUnmountAndKeepsAttachmentAfterError(t *testing.T) {
	want := errors.New("strict unmount failed")
	rt := &fakeJSAttachment{unmountErr: want}
	k := &Kernel{jsRuntime: rt, config: types.KernelConfig{JSRuntime: true}}

	err := k.DisableJSRuntime(context.Background())
	if !errors.Is(err, want) {
		t.Fatalf("DisableJSRuntime error = %v, want %v", err, want)
	}
	if rt.unmounts != 1 {
		t.Fatalf("Unmount calls = %d, want 1", rt.unmounts)
	}
	if rt.shutdowns != 0 || rt.closes != 0 {
		t.Fatalf("DisableJSRuntime used shutdown=%d close=%d, want neither", rt.shutdowns, rt.closes)
	}
	if k.jsRuntime != rt {
		t.Fatal("runtime attachment was detached after failed strict unmount")
	}
	if !k.config.JSRuntime {
		t.Fatal("runtime config no longer marks JSRuntime active after failed strict unmount")
	}

	rt.unmountErr = nil
	if err := k.DisableJSRuntime(context.Background()); err != nil {
		t.Fatalf("retry DisableJSRuntime: %v", err)
	}
	if rt.unmounts != 2 {
		t.Fatalf("Unmount calls after retry = %d, want 2", rt.unmounts)
	}
	if k.jsRuntime != nil {
		t.Fatal("runtime attachment was not detached after successful retry")
	}
	if k.config.JSRuntime {
		t.Fatal("runtime config still marks JSRuntime active after successful retry")
	}
}

func TestDisableJSRuntimeUsesShutdownWhileDraining(t *testing.T) {
	rt := &fakeJSAttachment{}
	k := &Kernel{jsRuntime: rt, config: types.KernelConfig{JSRuntime: true}}
	k.draining.Store(true)

	if err := k.DisableJSRuntime(context.Background()); err != nil {
		t.Fatalf("DisableJSRuntime while draining: %v", err)
	}
	if rt.shutdowns != 1 {
		t.Fatalf("Shutdown calls = %d, want 1", rt.shutdowns)
	}
	if rt.unmounts != 0 || rt.closes != 0 {
		t.Fatalf("draining DisableJSRuntime used unmount=%d close=%d, want neither", rt.unmounts, rt.closes)
	}
	if k.jsRuntime != nil {
		t.Fatal("runtime attachment was not detached")
	}
	if k.config.JSRuntime {
		t.Fatal("runtime config still marks JSRuntime active")
	}
}

func TestKernelCloseUsesRuntimeShutdownNotUnmount(t *testing.T) {
	rt := &fakeJSAttachment{}
	k := &Kernel{jsRuntime: rt, config: types.KernelConfig{JSRuntime: true}}

	if err := k.close(context.Background()); err != nil {
		t.Fatalf("kernel close: %v", err)
	}
	if rt.shutdowns != 1 {
		t.Fatalf("Shutdown calls = %d, want 1", rt.shutdowns)
	}
	if rt.unmounts != 0 || rt.closes != 0 {
		t.Fatalf("kernel close used unmount=%d close=%d, want neither", rt.unmounts, rt.closes)
	}
	if k.jsRuntime != nil {
		t.Fatal("runtime attachment was not detached")
	}
	if k.config.JSRuntime {
		t.Fatal("runtime config still marks JSRuntime active")
	}
}
