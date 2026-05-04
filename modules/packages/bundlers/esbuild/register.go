// Package esbuild registers the esbuild-backed package builder.
package esbuild

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/brainlet/brainkit/modules/packages"
	coredeploy "github.com/brainlet/brainkit/modules/packages/internal/deploy"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// Builder prepares package.deploy payloads with the Go esbuild API.
type Builder struct{}

func init() {
	packages.RegisterPackageBuilder(Builder{})
}

// BuildPackage turns a package.deploy path or in-memory file graph into one
// normalized JavaScript artifact.
func (Builder) BuildPackage(_ context.Context, req packages.BuildRequest, plugins packages.PluginChecker, secrets packages.SecretChecker) (packages.BuiltPackage, error) {
	if req.Path != "" {
		return buildPath(req.Path, plugins, secrets)
	}
	return buildFiles(req.Manifest, req.Files, plugins, secrets)
}

func buildPath(path string, plugins packages.PluginChecker, secrets packages.SecretChecker) (packages.BuiltPackage, error) {
	info, err := os.Stat(path)
	if err != nil {
		return packages.BuiltPackage{}, fmt.Errorf("package.deploy: stat path: %w", err)
	}
	if !info.IsDir() {
		return buildFile(path)
	}

	manifestPath := filepath.Join(path, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return packages.BuiltPackage{}, fmt.Errorf("package.deploy: read manifest: %w", err)
	}

	var manifest coredeploy.PackageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return packages.BuiltPackage{}, fmt.Errorf("package.deploy: parse manifest: %w", err)
	}
	if manifest.Name == "" {
		return packages.BuiltPackage{}, fmt.Errorf("package.deploy: manifest missing 'name'")
	}

	entryPath, err := coredeploy.ResolveEntry(path, manifest)
	if err != nil {
		return packages.BuiltPackage{}, fmt.Errorf("package.deploy: %w", err)
	}
	if err := coredeploy.ValidateDeps(manifest, plugins, secrets); err != nil {
		return packages.BuiltPackage{}, err
	}

	code, err := Bundle(entryPath)
	if err != nil {
		return packages.BuiltPackage{}, fmt.Errorf("package.deploy: bundle: %w", err)
	}
	if err := ensureNonEmptyBundle(entryPath, code); err != nil {
		return packages.BuiltPackage{}, err
	}

	return packages.BuiltPackage{
		Name:    manifest.Name,
		Version: manifest.Version,
		Source:  manifest.Name + ".ts",
		Code:    code,
	}, nil
}

func buildFile(path string) (packages.BuiltPackage, error) {
	filename := filepath.Base(path)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	code, err := Bundle(path)
	if err != nil {
		return packages.BuiltPackage{}, fmt.Errorf("package.deploy: bundle %s: %w", path, err)
	}
	if err := ensureNonEmptyBundle(path, code); err != nil {
		return packages.BuiltPackage{}, err
	}

	return packages.BuiltPackage{
		Name:    name,
		Version: "0.0.0",
		Source:  name + ".ts",
		Code:    code,
	}, nil
}

func buildFiles(manifestJSON json.RawMessage, files map[string]string, plugins packages.PluginChecker, secrets packages.SecretChecker) (packages.BuiltPackage, error) {
	var manifest struct {
		Name     string                   `json:"name"`
		Version  string                   `json:"version"`
		Entry    string                   `json:"entry"`
		Requires *coredeploy.Requirements `json:"requires,omitempty"`
	}
	if len(manifestJSON) > 0 {
		if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
			return packages.BuiltPackage{}, &sdkerrors.ValidationError{Field: "manifest", Message: err.Error()}
		}
	}
	if manifest.Name == "" {
		return packages.BuiltPackage{}, &sdkerrors.ValidationError{Field: "manifest.name", Message: "is required"}
	}
	if manifest.Entry == "" {
		return packages.BuiltPackage{}, &sdkerrors.ValidationError{Field: "manifest.entry", Message: "is required"}
	}
	if _, ok := files[manifest.Entry]; !ok {
		return packages.BuiltPackage{}, &sdkerrors.ValidationError{Field: "files", Message: fmt.Sprintf("entry %q not found", manifest.Entry)}
	}

	if manifest.Requires != nil {
		pm := coredeploy.PackageManifest{
			Name:     manifest.Name,
			Version:  manifest.Version,
			Entry:    manifest.Entry,
			Requires: manifest.Requires,
		}
		if err := coredeploy.ValidateDeps(pm, plugins, secrets); err != nil {
			return packages.BuiltPackage{}, err
		}
	}

	code, err := BundleInMemory(files, manifest.Entry)
	if err != nil {
		return packages.BuiltPackage{}, &sdkerrors.DeployError{
			Source: manifest.Entry,
			Phase:  "transpile",
			Cause:  fmt.Errorf("bundle: %w", err),
		}
	}
	if err := ensureNonEmptyBundle(manifest.Entry, code); err != nil {
		return packages.BuiltPackage{}, err
	}

	return packages.BuiltPackage{
		Name:    manifest.Name,
		Version: manifest.Version,
		Source:  manifest.Name + filepath.Ext(manifest.Entry),
		Code:    code,
	}, nil
}

func ensureNonEmptyBundle(source, code string) error {
	trimmed := strings.TrimSpace(code)
	if len(trimmed) == 0 || trimmed == ";" {
		return &sdkerrors.DeployError{
			Source: source,
			Phase:  "bundle",
			Cause:  fmt.Errorf("bundle produced empty output; check for unsupported import patterns"),
		}
	}
	return nil
}
