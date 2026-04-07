package tools

import (
	"strconv"
	"strings"
)

// ParseToolName breaks a tool name into its components.
// Handles all 5 formats:
//
//	1. "brainlet/cron@1.0.0/create"   -> owner="brainlet", pkg="cron", version="1.0.0", tool="create"
//	2. "cron@1.0.0/create"            -> owner="", pkg="cron", version="1.0.0", tool="create"
//	3. "brainlet/cron/create"          -> owner="brainlet", pkg="cron", version="", tool="create"
//	4. "cron/create"                   -> owner="", pkg="cron", version="", tool="create"
//	5. "create"                        -> owner="", pkg="", version="", tool="create"
//
// Names without "/" are bare short names — returned as-is in tool field.
func ParseToolName(name string) (owner, pkg, version, tool string) {
	if !strings.Contains(name, "/") {
		return "", "", "", name
	}

	parts := strings.Split(name, "/")

	switch len(parts) {
	case 1:
		return "", "", "", parts[0]
	case 2:
		// "cron@1.0.0/create" or "cron/create"
		pkg, version = splitVersion(parts[0])
		tool = parts[1]
		return "", pkg, version, tool
	case 3:
		// "brainlet/cron@1.0.0/create" or "brainlet/cron/create"
		owner = parts[0]
		pkg, version = splitVersion(parts[1])
		tool = parts[2]
		return owner, pkg, version, tool
	default:
		return "", "", "", name
	}
}

// splitVersion splits "cron@1.0.0" into ("cron", "1.0.0").
// If no "@", returns (s, "").
func splitVersion(s string) (string, string) {
	if atIdx := strings.Index(s, "@"); atIdx != -1 {
		return s[:atIdx], s[atIdx+1:]
	}
	return s, ""
}

// ComposeName builds a fully-qualified tool name: "owner/pkg@version/tool".
func ComposeName(owner, pkg, version, tool string) string {
	return owner + "/" + pkg + "@" + version + "/" + tool
}

// IsNewFormat returns true if the name uses "/" separators (new plugin naming).
func IsNewFormat(name string) bool {
	return strings.Contains(name, "/")
}

// CompareSemver compares two semver strings.
// Returns -1 if a < b, 0 if a == b, +1 if a > b.
// Pre-release versions sort lower than release versions.
func CompareSemver(a, b string) int {
	aMajor, aMinor, aPatch, aPre := parseSemver(a)
	bMajor, bMinor, bPatch, bPre := parseSemver(b)

	if aMajor != bMajor {
		return intCmp(aMajor, bMajor)
	}
	if aMinor != bMinor {
		return intCmp(aMinor, bMinor)
	}
	if aPatch != bPatch {
		return intCmp(aPatch, bPatch)
	}

	if aPre == "" && bPre != "" {
		return 1
	}
	if aPre != "" && bPre == "" {
		return -1
	}
	if aPre < bPre {
		return -1
	}
	if aPre > bPre {
		return 1
	}
	return 0
}

// IsPrerelease returns true if the version contains a pre-release suffix.
func IsPrerelease(version string) bool {
	_, _, _, pre := parseSemver(version)
	return pre != ""
}

func parseSemver(v string) (major, minor, patch int, pre string) {
	if dashIdx := strings.Index(v, "-"); dashIdx != -1 {
		pre = v[dashIdx+1:]
		v = v[:dashIdx]
	}

	parts := strings.Split(v, ".")
	if len(parts) >= 1 {
		major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) >= 2 {
		minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) >= 3 {
		patch, _ = strconv.Atoi(parts[2])
	}
	return
}

func intCmp(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
