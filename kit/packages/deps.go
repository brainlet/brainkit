package packages

import (
	"fmt"
	"strings"
)

// PackageManifestV2 is the package-level manifest (distinct from the plugin registry manifest).
// Lives in manifest.json alongside .ts source files.
type PackageManifestV2 struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description,omitempty"`
	Services    map[string]Service `json:"services"`
	Requires    *Requirements     `json:"requires,omitempty"`
	Tests       string            `json:"tests,omitempty"` // directory of *.test.ts files
}

// Service is a .ts entry point deployed as a separate Compartment.
type Service struct {
	Entry string `json:"entry"`
}

// Requirements declares plugin and secret dependencies.
type Requirements struct {
	Plugins []string `json:"plugins,omitempty"` // "brainlet/telegram-gateway@>=1.0.0"
	Secrets []string `json:"secrets,omitempty"` // "TELEGRAM_BOT_TOKEN"
}

// PluginChecker checks if a plugin is installed/running.
type PluginChecker interface {
	IsPluginInstalled(name string) bool
	IsPluginRunning(name string) bool
}

// SecretChecker checks if a secret exists.
type SecretChecker interface {
	HasSecret(name string) bool
}

// ValidateDeps checks that all required plugins are installed+running
// and all required secrets exist.
func ValidateDeps(manifest PackageManifestV2, plugins PluginChecker, secrets SecretChecker) error {
	if manifest.Requires == nil {
		return nil
	}

	for _, req := range manifest.Requires.Plugins {
		name := parsePluginReq(req)
		if !plugins.IsPluginInstalled(name) {
			return fmt.Errorf("package %q requires plugin %q which is not installed", manifest.Name, req)
		}
		if !plugins.IsPluginRunning(name) {
			return fmt.Errorf("package %q requires plugin %q which is installed but not running", manifest.Name, req)
		}
	}

	for _, secretName := range manifest.Requires.Secrets {
		if !secrets.HasSecret(secretName) {
			return fmt.Errorf("package %q requires secret %q which is not set", manifest.Name, secretName)
		}
	}

	return nil
}

// parsePluginReq extracts the plugin name from a requirement string.
// "brainlet/telegram-gateway@>=1.0.0" → "telegram-gateway"
// "brainlet/postgres-driver" → "postgres-driver"
func parsePluginReq(req string) string {
	// Strip version constraint
	if idx := strings.Index(req, "@"); idx != -1 {
		req = req[:idx]
	}
	// Get the plugin name (after last /)
	parts := strings.Split(req, "/")
	return parts[len(parts)-1]
}
