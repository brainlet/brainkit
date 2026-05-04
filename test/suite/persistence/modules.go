package persistence

import (
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/packages"
	_ "github.com/brainlet/brainkit/modules/packages/bundlers/esbuild"
)

func packageModules(extra ...bkmodule.Module) []bkmodule.Module {
	modules := make([]bkmodule.Module, 0, 1+len(extra))
	modules = append(modules, packages.New())
	modules = append(modules, extra...)
	return modules
}
