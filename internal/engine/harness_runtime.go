package engine

// HarnessRuntime returns the optional JS runtime's harness adapter without
// making core import the QuickJS-backed implementation.
func (k *Kernel) HarnessRuntime() any {
	if k.jsRuntime == nil {
		return nil
	}
	return k.jsRuntime.HarnessRuntime()
}
