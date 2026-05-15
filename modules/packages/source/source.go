// Package packagesource describes package.deploy source payloads without
// importing the packages module or runtime deploy machinery.
package packagesource

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Package describes a deployment unit.
type Package struct {
	Name     string            `json:"name"`
	Version  string            `json:"version,omitempty"`
	Entry    string            `json:"entry,omitempty"`
	Resolver string            `json:"resolver,omitempty"`
	Files    map[string]string `json:"files,omitempty"`

	path string `json:"-"`
}

const (
	ResolverSourceRelative = "source-relative"
	ResolverNPMPreview     = "npm-preview"
)

// DeployPayload is the source portion of a package.deploy request.
type DeployPayload struct {
	Path     string
	Manifest json.RawMessage
	Files    map[string]string
}

// Inline builds a Package from an inline source string.
func Inline(name, entry, source string) Package {
	return Package{
		Name:  name,
		Entry: entry,
		Files: map[string]string{entry: source},
	}
}

// FromDir loads a package from a directory containing manifest.json and source files.
func FromDir(dir string) (Package, error) {
	manifestPath := filepath.Join(dir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return Package{}, fmt.Errorf("packagesource.FromDir: read manifest: %w", err)
	}
	var m struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Entry   string `json:"entry"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return Package{}, fmt.Errorf("packagesource.FromDir: parse manifest: %w", err)
	}
	return Package{
		Name:    m.Name,
		Version: m.Version,
		Entry:   m.Entry,
		path:    dir,
	}, nil
}

// FromFile loads a single .ts file as a virtual package.
func FromFile(path string) (Package, error) {
	if _, err := os.Stat(path); err != nil {
		return Package{}, fmt.Errorf("packagesource.FromFile: %w", err)
	}
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return Package{
		Name:  name,
		Entry: filepath.Base(path),
		path:  path,
	}, nil
}

// WithResolver returns a copy configured for an explicit package resolver
// profile. The default resolver is source-relative.
func (p Package) WithResolver(resolver string) Package {
	p.Resolver = resolver
	return p
}

// DeployPayload builds the source payload carried by package.deploy.
func (p Package) DeployPayload() (DeployPayload, error) {
	if p.path != "" {
		if p.Resolver == "" {
			return DeployPayload{Path: p.path}, nil
		}
		raw, err := json.Marshal(map[string]string{"resolver": p.Resolver})
		if err != nil {
			return DeployPayload{}, err
		}
		return DeployPayload{Path: p.path, Manifest: raw}, nil
	}
	if p.Name == "" {
		return DeployPayload{}, fmt.Errorf("packagesource: Package.Name is required for inline deploy")
	}
	if p.Entry == "" {
		return DeployPayload{}, fmt.Errorf("packagesource: Package.Entry is required for inline deploy")
	}
	if len(p.Files) == 0 {
		return DeployPayload{}, fmt.Errorf("packagesource: Package.Files is required for inline deploy")
	}
	manifest := map[string]string{"name": p.Name, "entry": p.Entry}
	if p.Version != "" {
		manifest["version"] = p.Version
	}
	if p.Resolver != "" {
		manifest["resolver"] = p.Resolver
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		return DeployPayload{}, err
	}
	return DeployPayload{Manifest: raw, Files: p.Files}, nil
}
