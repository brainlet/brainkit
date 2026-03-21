package brainkit

import harnesspkg "github.com/brainlet/brainkit/harness"

// InitHarness creates and initializes a Harness backed by this Kit.
func (k *Kit) InitHarness(cfg harnesspkg.HarnessConfig) (*harnesspkg.Harness, error) {
	return harnesspkg.New(k.bridge, k.EvalTS, cfg)
}
