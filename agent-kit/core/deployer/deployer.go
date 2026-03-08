// Package deployer provides the deployer abstraction for Mastra.
// It extends the bundler with a deploy step.
//
// Ported from: packages/core/src/deployer/index.ts
package deployer

import (
	"github.com/brainlet/brainkit/agent-kit/core/bundler"
)

// IDeployer extends IBundler with a Deploy method that deploys to a target
// output directory.
//
// Ported from: packages/core/src/deployer/index.ts — IDeployer
type IDeployer interface {
	bundler.IBundler

	// Deploy deploys the bundled output to the given output directory.
	Deploy(outputDirectory string) error
}

// MastraDeployer is the base struct for all deployer implementations.
// It embeds MastraBundler and requires concrete types to implement Deploy.
//
// Ported from: packages/core/src/deployer/index.ts — MastraDeployer
type MastraDeployer struct {
	bundler.MastraBundler
}

// MastraDeployerOptions holds constructor options for MastraDeployer.
//
// Ported from: packages/core/src/deployer/index.ts — constructor({ name })
type MastraDeployerOptions struct {
	Name string
}

// NewMastraDeployer creates a new MastraDeployer with the "DEPLOYER" component type.
//
// Ported from: packages/core/src/deployer/index.ts — constructor({ name })
// In TS: super({ component: 'DEPLOYER', name });
func NewMastraDeployer(opts MastraDeployerOptions) MastraDeployer {
	return MastraDeployer{
		MastraBundler: *bundler.NewMastraBundler(bundler.MastraBundlerOptions{
			Name:      opts.Name,
			Component: "DEPLOYER",
		}),
	}
}
