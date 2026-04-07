package packages

import (
	"fmt"
	"strings"
)

// PackageManifest describes a package (the deployable unit).
// Lives in manifest.json alongside .ts source files.
type PackageManifest struct {
	Name        string        `json:"name"`
	Version     string        `json:"version"`
	Description string        `json:"description,omitempty"`
	Entry       string        `json:"entry,omitempty"` // default: resolved by ResolveEntry
	Requires    *Requirements `json:"requires,omitempty"`
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
	InstalledVersion(name string) string // returns "" if not installed
}

// SecretChecker checks if a secret exists.
type SecretChecker interface {
	HasSecret(name string) bool
}

// ValidateDeps checks that all required plugins are installed+running
// and all required secrets exist. Supports version constraints (>=, >, =).
func ValidateDeps(manifest PackageManifest, plugins PluginChecker, secrets SecretChecker) error {
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
		// Check version constraint if specified
		constraint := parseVersionConstraint(req)
		if constraint != "" {
			installed := plugins.InstalledVersion(name)
			if installed != "" && !checkVersionConstraint(installed, constraint) {
				return fmt.Errorf("package %q requires plugin %q but installed version %q does not satisfy %q",
					manifest.Name, name, installed, constraint)
			}
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

// parseVersionConstraint extracts the version constraint from a requirement.
// "brainlet/telegram-gateway@>=1.0.0" → ">=1.0.0"
// "brainlet/postgres-driver" → ""
func parseVersionConstraint(req string) string {
	idx := strings.Index(req, "@")
	if idx == -1 {
		return ""
	}
	return req[idx+1:]
}

// checkVersionConstraint checks if installedVersion satisfies the constraint.
// Supports: ">=X.Y.Z", ">X.Y.Z", "<=X.Y.Z", "<X.Y.Z", "=X.Y.Z", "X.Y.Z" (exact).
func checkVersionConstraint(installed, constraint string) bool {
	if constraint == "" || installed == "" {
		return true
	}

	var op string
	var target string
	if strings.HasPrefix(constraint, ">=") {
		op, target = ">=", constraint[2:]
	} else if strings.HasPrefix(constraint, ">") {
		op, target = ">", constraint[1:]
	} else if strings.HasPrefix(constraint, "<=") {
		op, target = "<=", constraint[2:]
	} else if strings.HasPrefix(constraint, "<") {
		op, target = "<", constraint[1:]
	} else if strings.HasPrefix(constraint, "=") {
		op, target = "=", constraint[1:]
	} else {
		op, target = "=", constraint
	}

	cmp := compareSemver(installed, target)
	switch op {
	case ">=":
		return cmp >= 0
	case ">":
		return cmp > 0
	case "<=":
		return cmp <= 0
	case "<":
		return cmp < 0
	case "=":
		return cmp == 0
	}
	return true
}

func compareSemver(a, b string) int {
	ap := parseSemverParts(a)
	bp := parseSemverParts(b)
	for i := 0; i < 3; i++ {
		if ap[i] < bp[i] {
			return -1
		}
		if ap[i] > bp[i] {
			return 1
		}
	}
	return 0
}

func parseSemverParts(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	if idx := strings.Index(v, "-"); idx != -1 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	var result [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		n := 0
		for _, c := range parts[i] {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			}
		}
		result[i] = n
	}
	return result
}
