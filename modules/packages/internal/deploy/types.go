package deploy

import "context"

// PackageManifest describes a package (the deployable unit).
type PackageManifest struct {
	Name        string        `json:"name"`
	Version     string        `json:"version"`
	Description string        `json:"description,omitempty"`
	Entry       string        `json:"entry,omitempty"`
	Requires    *Requirements `json:"requires,omitempty"`
}

// Requirements declares plugin and secret dependencies.
type Requirements struct {
	Plugins []string `json:"plugins,omitempty"`
	Secrets []string `json:"secrets,omitempty"`
}

// Package describes a deployed package.
type Package struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Dir     string `json:"dir"`
	Source  string `json:"source"`
}

// Deployer deploys bundled code into the runtime.
type Deployer interface {
	Deploy(ctx context.Context, source, code string) error
	Teardown(ctx context.Context, source string) error
}
