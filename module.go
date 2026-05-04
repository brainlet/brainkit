package brainkit

import (
	bkmodule "github.com/brainlet/brainkit/module"
)

// Module looks up a mounted Kit-scoped module by ID. Used for cross-module
// coordination (e.g. WithCallTo consults the topology module when
// present to resolve peer names). Returns (nil, false) when the
// module is absent.
func (k *Kit) Module(name string) (bkmodule.Module, bool) {
	k.mountMu.Lock()
	defer k.mountMu.Unlock()
	m, ok := k.modules[name]
	return m, ok
}

// Namespace returns the Kit's bus namespace (message topic scoping).
func (k *Kit) Namespace() string { return k.kernel.Namespace() }

// CallerID returns the Kit's identity stamped onto outbound bus messages.
func (k *Kit) CallerID() string { return k.kernel.CallerID() }

func (k *Kit) hasCommand(topic string) bool { return k.kernel.HasCommand(topic) }
