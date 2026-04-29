package persistence

import (
	"github.com/brainlet/brainkit"
	packagesmod "github.com/brainlet/brainkit/modules/packages"
)

func packageModules(extra ...brainkit.Module) []brainkit.Module {
	modules := make([]brainkit.Module, 0, 1+len(extra))
	modules = append(modules, packagesmod.New())
	modules = append(modules, extra...)
	return modules
}
