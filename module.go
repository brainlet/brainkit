package brainkit

import (
	bkmodule "github.com/brainlet/brainkit/module"
)

// Module is Brainkit's hot-mountable module contract.
type Module = bkmodule.Module

// ModuleStatus reports a module's maturity. Modules can optionally report
// their status for CLI listing / docs.
type ModuleStatus = bkmodule.Status

const (
	ModuleStatusStable ModuleStatus = bkmodule.StatusStable
	ModuleStatusBeta   ModuleStatus = bkmodule.StatusBeta
	ModuleStatusWIP    ModuleStatus = bkmodule.StatusWIP
)

// StatusReporter is implemented by modules that expose a maturity tag.
type StatusReporter interface {
	Status() ModuleStatus
}

// Module looks up a mounted Kit-scoped module by ID. Used for cross-module
// coordination (e.g. WithCallTo consults the topology module when
// present to resolve peer names). Returns (nil, false) when the
// module is absent.
func (k *Kit) Module(name string) (Module, bool) {
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
