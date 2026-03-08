// Ported from: packages/core/src/bundler/index.ts
package bundler

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// MastraBase is a stub for the root agentkit.MastraBase.
// TODO: import from agentkit package once circular-import issues are resolved.
type MastraBase struct {
	Component string
	Name      string
}

// ---------------------------------------------------------------------------
// IBundler interface
// ---------------------------------------------------------------------------

// IBundler defines the contract for bundler implementations.
type IBundler interface {
	// LoadEnvVars reads environment variables from .env files and returns them
	// as a map of key-value pairs.
	LoadEnvVars() (map[string]string, error)

	// GetEnvFiles returns the list of .env files to load.
	GetEnvFiles() ([]string, error)

	// GetAllToolPaths returns all tool paths for the given mastra directory
	// and tool path configurations.
	GetAllToolPaths(mastraDir string, toolsPaths []ToolPath) []ToolPath

	// Bundle bundles the entry file into the output directory.
	Bundle(entryFile string, outputDirectory string, options BundleOptions) error

	// Prepare prepares the output directory for bundling.
	Prepare(outputDirectory string) error

	// WritePackageJSON writes a package.json to the output directory with
	// the specified dependencies.
	WritePackageJSON(outputDirectory string, dependencies map[string]string) error

	// Lint runs linting on the bundled output.
	Lint(entryFile string, outputDirectory string, toolsPaths []ToolPath) error
}

// ToolPath represents a tool path which can be a single string or a list of
// strings (matching the TypeScript (string | string[])[] type).
type ToolPath struct {
	// Single is set when the tool path is a single string.
	Single string
	// Multiple is set when the tool path is a list of strings.
	Multiple []string
}

// BundleOptions holds options for the Bundle method.
type BundleOptions struct {
	ToolsPaths  []ToolPath
	ProjectRoot string
}

// ---------------------------------------------------------------------------
// MastraBundler — abstract base
// ---------------------------------------------------------------------------

// MastraBundler is the abstract base for bundler implementations.
// In TypeScript this extends MastraBase and implements IBundler.
// In Go, concrete implementations embed this struct and implement IBundler.
type MastraBundler struct {
	MastraBase
}

// MastraBundlerOptions holds constructor options for MastraBundler.
type MastraBundlerOptions struct {
	Name      string
	Component string // defaults to "BUNDLER" if empty
}

// NewMastraBundler creates a new MastraBundler base.
func NewMastraBundler(opts MastraBundlerOptions) *MastraBundler {
	component := opts.Component
	if component == "" {
		component = "BUNDLER"
	}
	return &MastraBundler{
		MastraBase: MastraBase{
			Component: component,
			Name:      opts.Name,
		},
	}
}

// LoadEnvVars reads environment variables from the files returned by
// GetEnvFiles. Concrete implementations must provide GetEnvFiles.
// This mirrors the TypeScript base class's non-abstract loadEnvVars().
func (b *MastraBundler) LoadEnvVars(getEnvFiles func() ([]string, error)) (map[string]string, error) {
	files, err := getEnvFiles()
	if err != nil {
		return nil, fmt.Errorf("bundler: failed to get env files: %w", err)
	}

	envVars := make(map[string]string)

	for _, file := range files {
		parsed, err := parseDotenv(file)
		if err != nil {
			return nil, fmt.Errorf("bundler: failed to parse env file %s: %w", file, err)
		}
		for k, v := range parsed {
			envVars[k] = v
		}
	}

	return envVars, nil
}

// parseDotenv is a minimal .env parser mirroring the Node.js dotenv.parse().
// It reads key=value pairs (one per line), ignoring comments and blank lines.
func parseDotenv(filePath string) (map[string]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Strip surrounding quotes
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		result[key] = value
	}

	return result, scanner.Err()
}
