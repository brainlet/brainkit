// Ported from: packages/core/src/workspace/lsp/servers.ts
package lsp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// =============================================================================
// Binary Resolution Helpers
// =============================================================================

// whichSync checks if a binary exists on PATH.
func whichSync(binary string) bool {
	_, err := exec.LookPath(binary)
	return err == nil
}

// resolveNodeBin finds a binary in node_modules/.bin, searching root, cwd,
// then any searchPaths. Returns empty string if not found.
func resolveNodeBin(root, binary string, searchPaths []string) string {
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".cmd"
	}

	local := filepath.Join(root, "node_modules", ".bin", binary+suffix)
	if fileExists(local) {
		return local
	}

	cwd, _ := os.Getwd()
	if cwd != "" {
		cwdBin := filepath.Join(cwd, "node_modules", ".bin", binary+suffix)
		if fileExists(cwdBin) {
			return cwdBin
		}
	}

	for _, dir := range searchPaths {
		p := filepath.Join(dir, "node_modules", ".bin", binary+suffix)
		if fileExists(p) {
			return p
		}
	}

	return ""
}

// resolveNodeModule checks if a Node.js module can be found from root, cwd,
// or searchPaths. Returns the resolved path or empty string.
func resolveNodeModule(root, moduleID string, searchPaths []string) string {
	// Check from root
	p := filepath.Join(root, "node_modules", moduleID)
	if fileExists(p) {
		return p
	}

	// Check from cwd
	cwd, _ := os.Getwd()
	if cwd != "" {
		p = filepath.Join(cwd, "node_modules", moduleID)
		if fileExists(p) {
			return p
		}
	}

	// Check searchPaths
	for _, dir := range searchPaths {
		p = filepath.Join(dir, "node_modules", moduleID)
		if fileExists(p) {
			return p
		}
	}

	return ""
}

// fileExists checks if a file or directory exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// =============================================================================
// Walk Up
// =============================================================================

// WalkUp walks up from a starting directory looking for any of the given markers.
// Returns the first directory that contains a marker, or empty string if not found.
func WalkUp(startDir string, markers []string) string {
	current := startDir

	for {
		for _, marker := range markers {
			p := filepath.Join(current, marker)
			if fileExists(p) {
				return current
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return ""
}

// ExistsFunc is a function that checks if a path exists.
// Used by WalkUpAsync for filesystem abstraction.
type ExistsFunc func(path string) (bool, error)

// WalkUpAsync walks up from a starting directory looking for any of the given markers.
// Uses the provided exists function for filesystem abstraction (supports remote filesystems).
// Returns empty string if not found.
func WalkUpAsync(startDir string, markers []string, exists ExistsFunc) (string, error) {
	current := startDir

	for {
		for _, marker := range markers {
			p := filepath.Join(current, marker)
			found, err := exists(p)
			if err != nil {
				return "", err
			}
			if found {
				return current, nil
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", nil
}

// =============================================================================
// Default Markers
// =============================================================================

// DefaultMarkers are the default markers used to find a project root.
var DefaultMarkers = []string{
	"tsconfig.json",
	"package.json",
	"pyproject.toml",
	"go.mod",
	"Cargo.toml",
	"composer.json",
	".git",
}

// FindProjectRoot finds a project root by walking up from a starting directory.
// Uses default markers (tsconfig.json, package.json, go.mod, etc.).
func FindProjectRoot(startDir string) string {
	return WalkUp(startDir, DefaultMarkers)
}

// FindProjectRootAsync finds a project root using a filesystem's exists function.
func FindProjectRootAsync(startDir string, exists ExistsFunc) (string, error) {
	return WalkUpAsync(startDir, DefaultMarkers, exists)
}

// =============================================================================
// Server Definitions
// =============================================================================

// BuildServerDefs builds a set of server definitions that incorporate LSP config overrides.
//
// Resolution order per server:
//  1. config.BinaryOverrides[id] - explicit binary command override
//  2. Project node_modules/.bin/ binary
//  3. process.cwd() node_modules/.bin/ binary
//  4. config.SearchPaths node_modules/.bin/ binary lookup
//  5. Global PATH lookup (system-installed binaries)
//  6. config.PackageRunner - package runner fallback (off by default)
func BuildServerDefs(config *LSPConfig) map[string]*LSPServerDef {
	var binaryOverrides map[string]string
	var searchPaths []string
	var packageRunner string

	if config != nil {
		binaryOverrides = config.BinaryOverrides
		searchPaths = config.SearchPaths
		packageRunner = config.PackageRunner
	}

	return map[string]*LSPServerDef{
		"typescript": {
			ID:          "typescript",
			Name:        "TypeScript Language Server",
			LanguageIDs: []string{"typescript", "typescriptreact", "javascript", "javascriptreact"},
			Markers:     []string{"tsconfig.json", "package.json"},
			Command: func(root string) string {
				if override, ok := binaryOverrides["typescript"]; ok && override != "" {
					return override
				}
				if resolveNodeModule(root, "typescript/lib/tsserver.js", searchPaths) == "" {
					return ""
				}
				bin := resolveNodeBin(root, "typescript-language-server", searchPaths)
				if bin != "" {
					return bin + " --stdio"
				}
				if whichSync("typescript-language-server") {
					return "typescript-language-server --stdio"
				}
				if packageRunner != "" {
					return packageRunner + " typescript-language-server --stdio"
				}
				return ""
			},
			Initialization: func(root string) map[string]interface{} {
				tsPath := resolveNodeModule(root, "typescript/lib/tsserver.js", searchPaths)
				if tsPath == "" {
					return nil
				}
				return map[string]interface{}{
					"tsserver": map[string]interface{}{
						"path":       tsPath,
						"logVerbosity": "off",
					},
				}
			},
		},

		"eslint": {
			ID:          "eslint",
			Name:        "ESLint Language Server",
			LanguageIDs: []string{"typescript", "typescriptreact", "javascript", "javascriptreact"},
			Markers: []string{
				"package.json",
				".eslintrc.js",
				".eslintrc.json",
				".eslintrc.yml",
				".eslintrc.yaml",
				"eslint.config.js",
				"eslint.config.mjs",
				"eslint.config.ts",
			},
			Command: func(root string) string {
				if override, ok := binaryOverrides["eslint"]; ok && override != "" {
					return override
				}
				bin := resolveNodeBin(root, "vscode-eslint-language-server", searchPaths)
				if bin != "" {
					return bin + " --stdio"
				}
				if whichSync("vscode-eslint-language-server") {
					return "vscode-eslint-language-server --stdio"
				}
				if packageRunner != "" {
					return fmt.Sprintf("%s vscode-eslint-language-server --stdio", packageRunner)
				}
				return ""
			},
		},

		"python": {
			ID:          "python",
			Name:        "Python Language Server (Pyright)",
			LanguageIDs: []string{"python"},
			Markers:     []string{"pyproject.toml", "setup.py", "requirements.txt", "setup.cfg"},
			Command: func(root string) string {
				if override, ok := binaryOverrides["python"]; ok && override != "" {
					return override
				}
				bin := resolveNodeBin(root, "pyright-langserver", searchPaths)
				if bin != "" {
					return bin + " --stdio"
				}
				if whichSync("pyright-langserver") {
					return "pyright-langserver --stdio"
				}
				if packageRunner != "" {
					return fmt.Sprintf("%s pyright-langserver --stdio", packageRunner)
				}
				return ""
			},
		},

		"go": {
			ID:          "go",
			Name:        "Go Language Server (gopls)",
			LanguageIDs: []string{"go"},
			Markers:     []string{"go.mod"},
			Command: func(root string) string {
				if override, ok := binaryOverrides["go"]; ok && override != "" {
					return override
				}
				if whichSync("gopls") {
					return "gopls serve"
				}
				return ""
			},
		},

		"rust": {
			ID:          "rust",
			Name:        "Rust Language Server (rust-analyzer)",
			LanguageIDs: []string{"rust"},
			Markers:     []string{"Cargo.toml"},
			Command: func(root string) string {
				if override, ok := binaryOverrides["rust"]; ok && override != "" {
					return override
				}
				if whichSync("rust-analyzer") {
					return "rust-analyzer --stdio"
				}
				return ""
			},
		},
	}
}

// BuiltinServers returns built-in LSP server definitions with no config overrides.
func BuiltinServers() map[string]*LSPServerDef {
	return BuildServerDefs(nil)
}

// GetServersForFile returns all server definitions that can handle the given file.
// Filters by language ID match only -- the manager resolves the root and checks
// command availability. Pass defs to use config-aware server definitions from
// BuildServerDefs().
func GetServersForFile(filePath string, disabledServers []string, defs map[string]*LSPServerDef) []*LSPServerDef {
	languageID := GetLanguageId(filePath)
	if languageID == "" {
		return nil
	}

	disabled := make(map[string]bool)
	for _, s := range disabledServers {
		disabled[s] = true
	}

	servers := defs
	if servers == nil {
		servers = BuiltinServers()
	}

	var result []*LSPServerDef
	for _, server := range servers {
		if disabled[server.ID] {
			continue
		}
		for _, langID := range server.LanguageIDs {
			if langID == languageID {
				result = append(result, server)
				break
			}
		}
	}

	return result
}
