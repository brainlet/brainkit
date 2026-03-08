// Ported from: packages/core/src/bundler/types.ts
package bundler

// BundlerConfig holds configuration for the bundler.
type BundlerConfig struct {
	// Externals controls which dependencies are excluded from the bundle and
	// installed separately.
	// When ExternalsAll is true, all non-workspace packages are excluded.
	// When ExternalsList is non-nil, those specific packages are excluded
	// (merged with global externals like 'pino', 'pg', '@libsql/client').
	ExternalsAll  bool     `json:"externalsAll,omitempty"`
	ExternalsList []string `json:"externalsList,omitempty"`

	// Sourcemap enables source map generation for debugging bundled code.
	// Generates '.mjs.map' files alongside bundled output.
	Sourcemap bool `json:"sourcemap,omitempty"`

	// TranspilePackages lists packages requiring TypeScript/modern JS
	// transpilation during bundling. Automatically includes workspace packages.
	TranspilePackages []string `json:"transpilePackages,omitempty"`
}
