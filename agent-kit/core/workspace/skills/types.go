// Ported from: packages/core/src/workspace/skills/types.ts
package skills

import (
	"strings"

	"github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	"github.com/brainlet/brainkit/agent-kit/core/workspace/search"
)

// =============================================================================
// Content Source Types
// =============================================================================

// ContentSourceType is a source type identifier for content origin.
type ContentSourceType string

const (
	ContentSourceExternal ContentSourceType = "external"
	ContentSourceLocal    ContentSourceType = "local"
	ContentSourceManaged  ContentSourceType = "managed"
)

// ContentSource indicates where a skill comes from.
//
//   - external: From node_modules packages
//   - local: From project source directory
//   - managed: From .mastra directory, typically Studio-managed
type ContentSource struct {
	Type        ContentSourceType `json:"type"`
	PackagePath string            `json:"packagePath,omitempty"`
	ProjectPath string            `json:"projectPath,omitempty"`
	MastraPath  string            `json:"mastraPath,omitempty"`
}

// GetSourceForPath determines the source type for a given path.
func GetSourceForPath(path string) ContentSource {
	if strings.Contains(path, "node_modules") {
		return ContentSource{Type: ContentSourceExternal, PackagePath: path}
	}
	if strings.Contains(path, ".mastra") {
		return ContentSource{Type: ContentSourceManaged, MastraPath: path}
	}
	return ContentSource{Type: ContentSourceLocal, ProjectPath: path}
}

// =============================================================================
// Search Types
// =============================================================================

// SearchMode represents the search mode options.
type SearchMode = search.SearchMode

// ScoreDetails holds the score breakdown for hybrid search.
type ScoreDetails struct {
	// Vector is the vector similarity score (0-1).
	Vector *float64 `json:"vector,omitempty"`
	// BM25 is the BM25 relevance score.
	BM25 *float64 `json:"bm25,omitempty"`
}

// BaseSearchResult is a base search result with common fields.
type BaseSearchResult struct {
	// Content that was matched.
	Content string `json:"content"`
	// Score is the relevance score (higher is more relevant).
	Score float64 `json:"score"`
	// LineRange is where query terms were found (if available).
	LineRange *search.LineRange `json:"lineRange,omitempty"`
	// ScoreDetails is the score breakdown for hybrid search.
	ScoreDetails *ScoreDetails `json:"scoreDetails,omitempty"`
	// Metadata holds additional metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// BaseSearchOptions are base search options with common fields.
type BaseSearchOptions struct {
	// TopK is the maximum number of results to return (default: 5).
	TopK int `json:"topK,omitempty"`
	// MinScore is the minimum score threshold.
	MinScore float64 `json:"minScore,omitempty"`
	// Mode is the search mode.
	Mode SearchMode `json:"mode,omitempty"`
	// Hybrid holds hybrid search configuration.
	Hybrid *HybridConfig `json:"hybrid,omitempty"`
}

// HybridConfig holds hybrid search configuration.
type HybridConfig struct {
	// VectorWeight is the weight for vector similarity score (0-1).
	VectorWeight *float64 `json:"vectorWeight,omitempty"`
}

// =============================================================================
// Skills Types
// =============================================================================

// SkillsContext is the context passed to skills resolver function.
type SkillsContext struct {
	// RequestContext with user/thread information.
	RequestContext *requestcontext.RequestContext
}

// SkillsResolver resolves skills paths. Can be a static array of paths or a dynamic function.
type SkillsResolver interface {
	ResolvePaths(ctx SkillsContext) ([]string, error)
}

// StaticSkillsResolver is a fixed array of paths to scan for skills.
type StaticSkillsResolver struct {
	Paths []string
}

// ResolvePaths returns the static paths.
func (r *StaticSkillsResolver) ResolvePaths(_ SkillsContext) ([]string, error) {
	return r.Paths, nil
}

// DynamicSkillsResolver is a function that returns paths based on context.
type DynamicSkillsResolver struct {
	Resolve func(ctx SkillsContext) ([]string, error)
}

// ResolvePaths calls the resolver function.
func (r *DynamicSkillsResolver) ResolvePaths(ctx SkillsContext) ([]string, error) {
	return r.Resolve(ctx)
}

// SkillFormat represents supported skill format types for system message injection.
type SkillFormat string

const (
	SkillFormatXML      SkillFormat = "xml"
	SkillFormatJSON     SkillFormat = "json"
	SkillFormatMarkdown SkillFormat = "markdown"
)

// SkillMetadata holds skill metadata from YAML frontmatter (following Agent Skills spec).
type SkillMetadata struct {
	// Name is the skill name (1-64 chars, lowercase, hyphens only).
	Name string `json:"name"`
	// Description of what the skill does and when to use it (1-1024 chars).
	Description string `json:"description"`
	// License is the optional license.
	License string `json:"license,omitempty"`
	// Compatibility holds optional compatibility requirements.
	Compatibility interface{} `json:"compatibility,omitempty"`
	// Metadata holds optional arbitrary metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Skill is a full skill with parsed instructions and path info.
type Skill struct {
	SkillMetadata
	// Path is the path to skill directory (relative to workspace root).
	Path string `json:"path"`
	// Instructions is the Markdown body from SKILL.md.
	Instructions string `json:"instructions"`
	// Source of the skill (external package, local project, or managed).
	Source ContentSource `json:"source"`
	// References is a list of reference file paths (relative to references/ directory).
	References []string `json:"references"`
	// Scripts is a list of script file paths (relative to scripts/ directory).
	Scripts []string `json:"scripts"`
	// Assets is a list of asset file paths (relative to assets/ directory).
	Assets []string `json:"assets"`
}

// SkillSearchResult is a search result when searching across skills.
type SkillSearchResult struct {
	BaseSearchResult
	// SkillName is the skill name.
	SkillName string `json:"skillName"`
	// SourceFile is the source file (SKILL.md or reference path).
	SourceFile string `json:"source"`
}

// SkillSearchOptions are options for searching skills.
type SkillSearchOptions struct {
	BaseSearchOptions
	// SkillNames limits search to specific skill names.
	SkillNames []string `json:"skillNames,omitempty"`
	// IncludeReferences includes reference files in search (default: true).
	IncludeReferences *bool `json:"includeReferences,omitempty"`
}

// ShouldIncludeReferences returns true if reference files should be included (default: true).
func (o *SkillSearchOptions) ShouldIncludeReferences() bool {
	if o.IncludeReferences == nil {
		return true
	}
	return *o.IncludeReferences
}

// =============================================================================
// WorkspaceSkills Interface
// =============================================================================

// WorkspaceSkills is the interface for skills accessed via workspace.skills.
// Provides discovery and search operations for skills in the workspace.
type WorkspaceSkills interface {
	// List returns all discovered skills (metadata only).
	List() ([]SkillMetadata, error)
	// Get returns a specific skill by name (full content).
	Get(name string) (*Skill, error)
	// Has checks if a skill exists.
	Has(name string) (bool, error)
	// Refresh re-scans skills from filesystem.
	Refresh() error
	// MaybeRefresh conditionally refreshes skills if they have been modified.
	MaybeRefresh(ctx *SkillsContext) error
	// Search searches across all skills content.
	Search(query string, options *SkillSearchOptions) ([]SkillSearchResult, error)
	// GetReference gets reference file content from a skill.
	GetReference(skillName, referencePath string) (string, error)
	// GetScript gets script file content from a skill.
	GetScript(skillName, scriptPath string) (string, error)
	// GetAsset gets asset file content from a skill (returns []byte for binary files).
	GetAsset(skillName, assetPath string) ([]byte, error)
	// ListReferences gets all reference file paths for a skill.
	ListReferences(skillName string) ([]string, error)
	// ListScripts gets all script file paths for a skill.
	ListScripts(skillName string) ([]string, error)
	// ListAssets gets all asset file paths for a skill.
	ListAssets(skillName string) ([]string, error)
	// AddSkill surgically adds or updates a single skill in the cache.
	AddSkill(skillPath string) error
	// RemoveSkill surgically removes a single skill from the cache by name.
	RemoveSkill(skillName string) error
}
