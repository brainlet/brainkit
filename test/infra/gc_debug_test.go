package infra_test

import (
	"testing"

	"github.com/brainlet/brainkit/kit"
)

// TestGC_SingleKernelCleanClose creates a single Kernel and closes it.
// This tests whether JS_FreeContext crashes on SES-hardened objects.
func TestGC_SingleKernelCleanClose(t *testing.T) {
	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "gc-test",
		CallerID:     "gc-test",
		WorkspaceDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewKernel: %v", err)
	}
	t.Log("Kernel created, closing...")
	if err := k.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	t.Log("Kernel closed cleanly")
}

// TestGC_MultipleKernelCleanClose creates and destroys 5 Kernels sequentially.
// Tests whether accumulated bridge lifecycles cause thread/memory issues.
func TestGC_MultipleKernelCleanClose(t *testing.T) {
	for i := 0; i < 5; i++ {
		k, err := kit.NewKernel(kit.KernelConfig{
			Namespace:    "gc-multi",
			CallerID:     "gc-multi",
			WorkspaceDir: t.TempDir(),
		})
		if err != nil {
			t.Fatalf("NewKernel %d: %v", i, err)
		}
		if err := k.Close(); err != nil {
			t.Fatalf("Close %d: %v", i, err)
		}
		t.Logf("Kernel %d closed", i)
	}
	t.Log("All 5 Kernels closed cleanly")
}

// TestGC_TenKernelCleanClose stress test — 10 Kernels.
func TestGC_TenKernelCleanClose(t *testing.T) {
	for i := 0; i < 10; i++ {
		k, err := kit.NewKernel(kit.KernelConfig{
			Namespace:    "gc-stress",
			CallerID:     "gc-stress",
			WorkspaceDir: t.TempDir(),
		})
		if err != nil {
			t.Fatalf("NewKernel %d: %v", i, err)
		}
		if err := k.Close(); err != nil {
			t.Fatalf("Close %d: %v", i, err)
		}
	}
	t.Log("All 10 Kernels closed cleanly")
}
