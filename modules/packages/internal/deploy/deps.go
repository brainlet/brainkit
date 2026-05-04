package deploy

import (
	"fmt"
	"strings"
)

// PluginChecker checks if a plugin is running.
type PluginChecker interface {
	IsPluginRunning(name string) bool
}

// SecretChecker checks if a secret exists.
type SecretChecker interface {
	HasSecret(name string) bool
}

// ValidateDeps checks that all required plugins are running
// and all required secrets exist.
func ValidateDeps(manifest PackageManifest, plugins PluginChecker, secrets SecretChecker) error {
	if manifest.Requires == nil {
		return nil
	}

	for _, req := range manifest.Requires.Plugins {
		name := parsePluginReq(req)
		if !plugins.IsPluginRunning(name) {
			return fmt.Errorf("package %q requires plugin %q which is not running", manifest.Name, req)
		}
	}

	for _, secretName := range manifest.Requires.Secrets {
		if !secrets.HasSecret(secretName) {
			return fmt.Errorf("package %q requires secret %q which is not set", manifest.Name, secretName)
		}
	}

	return nil
}

func parsePluginReq(req string) string {
	if idx := strings.Index(req, "@"); idx != -1 {
		req = req[:idx]
	}
	parts := strings.Split(req, "/")
	return parts[len(parts)-1]
}
