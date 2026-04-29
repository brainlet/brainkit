package engine

import "fmt"

// AttachJSRuntime installs the optional JS/TS runtime attachment. It is the
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

// DetachJSRuntime removes the runtime attachment. The runtime implementation
// owns cleanup of its own resources.
func (k *Kernel) DetachJSRuntime(rt JSRuntimeAttachment) {
	if k == nil || rt == nil {
		return
	}
	k.mu.Lock()
	k.jsRuntime = nil
	k.mu.Unlock()
}
