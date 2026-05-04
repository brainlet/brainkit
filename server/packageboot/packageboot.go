// Package packageboot wires package deployment into server start hooks.
package packageboot

import (
	"context"
	"fmt"

	"github.com/brainlet/brainkit"
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/packages"
	"github.com/brainlet/brainkit/server"
)

// Add appends the packages module when needed and installs an OnStart hook that
// deploys each package before the server supervisor blocks.
func Add(cfg *server.Config, pkgs ...packages.Package) {
	if cfg == nil || len(pkgs) == 0 {
		return
	}
	if !hasModule(cfg.Modules, "packages") {
		cfg.Modules = append(cfg.Modules, packages.New())
	}
	cfg.OnStart = append(cfg.OnStart, Deploy(pkgs...))
}

// Deploy returns a server start hook that deploys packages in order.
func Deploy(pkgs ...packages.Package) server.StartHook {
	return func(ctx context.Context, kit *brainkit.Kit) error {
		for _, pkg := range pkgs {
			if _, err := packages.Deploy(ctx, kit, pkg); err != nil {
				return fmt.Errorf("server: deploy package: %w", err)
			}
		}
		return nil
	}
}

func hasModule(mods []bkmodule.Module, id string) bool {
	for _, m := range mods {
		if m != nil && m.ID() == id {
			return true
		}
	}
	return false
}
