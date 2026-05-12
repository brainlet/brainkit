package engine

import (
	"context"
	"errors"
	"fmt"
)

// AttachJSRuntime installs the optional JavaScript runtime attachment. It is the
// narrow handoff point used by the jsruntime module/package boundary.
func (k *Kernel) AttachJSRuntime(rt JSRuntimeAttachment) error {
	if k == nil {
		return fmt.Errorf("brainkit: kernel is nil")
	}
	if rt == nil {
		return fmt.Errorf("brainkit: js runtime attachment is nil")
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.jsRuntime != nil {
		return nil
	}
	k.jsRuntime = rt
	return nil
}

// DisableJSRuntime detaches the optional runtime attachment. Live hot-unmount
// uses strict cleanup; process shutdown/drain uses the best-effort shutdown
// path so a dying kernel is not blocked by restore-only cleanup.
func (k *Kernel) DisableJSRuntime(ctx context.Context) error {
	if k == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	k.mu.Lock()
	rt := k.jsRuntime
	k.mu.Unlock()
	if rt == nil {
		k.SetRuntimeConfigJSRuntime(false)
		return nil
	}

	var err error
	if k.runtimeHost != nil {
		err = errors.Join(err, k.runtimeHost.CloseContext(ctx))
	}
	if k.draining.Load() || k.IsClosed() {
		err = errors.Join(err, rt.Shutdown(ctx))
	} else {
		err = errors.Join(err, rt.Unmount(ctx))
	}
	if err != nil {
		return err
	}
	k.DetachJSRuntime(rt)
	k.SetRuntimeConfigJSRuntime(false)
	return nil
}

// DetachJSRuntime removes the runtime attachment. The runtime implementation
// owns cleanup of its own resources.
func (k *Kernel) DetachJSRuntime(rt JSRuntimeAttachment) {
	if k == nil || rt == nil {
		return
	}
	k.mu.Lock()
	if k.jsRuntime == rt {
		k.jsRuntime = nil
	}
	k.mu.Unlock()
}
