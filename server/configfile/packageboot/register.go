// Package packageboot registers top-level `packages:` YAML support for
// server/configfile.
package packageboot

import (
	"fmt"

	"github.com/brainlet/brainkit/modules/packages/client"
	_ "github.com/brainlet/brainkit/modules/packages/bundlers/esbuild"
	"github.com/brainlet/brainkit/server"
	"github.com/brainlet/brainkit/server/configfile"
	serverpackageboot "github.com/brainlet/brainkit/server/packageboot"
)

func init() {
	configfile.RegisterPackageLoader(load)
}

func load(cfg *server.Config, specs []configfile.PackageYAML) error {
	pkgs := make([]packageclient.Package, 0, len(specs))
	for _, spec := range specs {
		pkg, err := packageclient.FromDir(spec.Path)
		if err != nil {
			return fmt.Errorf("server: load package %q: %w", spec.Path, err)
		}
		pkgs = append(pkgs, pkg)
	}
	serverpackageboot.Add(cfg, pkgs...)
	return nil
}
