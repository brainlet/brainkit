// Package features provides core feature flags for agent-kit.
// Dependent packages can check for feature availability to ensure compatibility.
//
// Ported from: packages/core/src/features/index.ts
package features

// CoreFeatures is the set of feature flags available in the current version
// of agent-kit core. Dependent packages can check for feature availability
// using the Has method.
//
// Usage:
//
//	if features.CoreFeatures.Has("workspaces-v1") {
//	    // Workspace features available
//	}
//
// Ported from: packages/core/src/features/index.ts — coreFeatures
var CoreFeatures = NewFeatureSet(
	"observationalMemory",
	"asyncBuffering",
	"workspaces-v1",
	"datasets",
)

// FeatureSet is a set of feature flag strings.
//
// Ported from: packages/core/src/features/index.ts — Set<string>
type FeatureSet struct {
	features map[string]struct{}
}

// NewFeatureSet creates a FeatureSet from the given feature names.
func NewFeatureSet(names ...string) FeatureSet {
	fs := FeatureSet{
		features: make(map[string]struct{}, len(names)),
	}
	for _, name := range names {
		fs.features[name] = struct{}{}
	}
	return fs
}

// Has returns true if the given feature flag is present in the set.
//
// Ported from: packages/core/src/features/index.ts — coreFeatures.has(...)
func (fs FeatureSet) Has(name string) bool {
	_, ok := fs.features[name]
	return ok
}

// All returns all feature flag names in the set.
func (fs FeatureSet) All() []string {
	result := make([]string, 0, len(fs.features))
	for name := range fs.features {
		result = append(result, name)
	}
	return result
}
