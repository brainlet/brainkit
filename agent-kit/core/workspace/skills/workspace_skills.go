// Ported from: packages/core/src/workspace/skills/workspace-skills.ts
package skills

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// Internal Types
// =============================================================================

// SkillSearchEngine is a minimal search engine interface for skill search.
type SkillSearchEngine interface {
	Index(doc SkillIndexDocument) error
	Remove(id string) error
	Search(query string, options *SkillEngineSearchOptions) ([]SkillEngineSearchResult, error)
	Clear()
}

// SkillIndexDocument is a document to index for search.
type SkillIndexDocument struct {
	ID       string
	Content  string
	Metadata map[string]interface{}
}

// SkillEngineSearchOptions are search options for the engine.
type SkillEngineSearchOptions struct {
	TopK     int
	MinScore float64
	Mode     string
}

// SkillEngineSearchResult is a search result from the engine.
type SkillEngineSearchResult struct {
	ID           string
	Content      string
	Score        float64
	Metadata     map[string]interface{}
	ScoreDetails *ScoreDetails
}

// internalSkill extends Skill with indexable content for BM25 indexing.
type internalSkill struct {
	Skill
	indexableContent string
}

// =============================================================================
// WorkspaceSkillsImpl
// =============================================================================

// WorkspaceSkillsImplConfig holds configuration for WorkspaceSkillsImpl.
type WorkspaceSkillsImplConfig struct {
	// Source for loading skills.
	Source SkillSource
	// Skills are the paths to scan for skills.
	Skills SkillsResolver
	// SearchEngine for skill search (optional).
	SearchEngine SkillSearchEngine
	// ValidateOnLoad validates skills on load (default: true).
	ValidateOnLoad *bool
}

// WorkspaceSkillsImpl implements the WorkspaceSkills interface.
type WorkspaceSkillsImpl struct {
	mu sync.RWMutex

	source         SkillSource
	skillsResolver SkillsResolver
	searchEngine   SkillSearchEngine
	validateOnLoad bool

	skills            map[string]*internalSkill
	initialized       bool
	initOnce          sync.Once
	initErr           error
	lastDiscoveryTime int64
	resolvedPaths     []string
}

const (
	// stalenessCheckCooldown skips staleness check for 2s after discovery.
	stalenessCheckCooldown = 2 * time.Second
)

// NewWorkspaceSkillsImpl creates a new WorkspaceSkillsImpl.
func NewWorkspaceSkillsImpl(config WorkspaceSkillsImplConfig) *WorkspaceSkillsImpl {
	validateOnLoad := true
	if config.ValidateOnLoad != nil {
		validateOnLoad = *config.ValidateOnLoad
	}
	return &WorkspaceSkillsImpl{
		source:         config.Source,
		skillsResolver: config.Skills,
		searchEngine:   config.SearchEngine,
		validateOnLoad: validateOnLoad,
		skills:         make(map[string]*internalSkill),
	}
}

// =============================================================================
// Discovery
// =============================================================================

// List returns all discovered skills (metadata only).
func (ws *WorkspaceSkillsImpl) List() ([]SkillMetadata, error) {
	if err := ws.ensureInitialized(); err != nil {
		return nil, err
	}

	ws.mu.RLock()
	defer ws.mu.RUnlock()

	result := make([]SkillMetadata, 0, len(ws.skills))
	for _, skill := range ws.skills {
		result = append(result, skill.SkillMetadata)
	}
	return result, nil
}

// Get returns a specific skill by name (full content).
func (ws *WorkspaceSkillsImpl) Get(name string) (*Skill, error) {
	if err := ws.ensureInitialized(); err != nil {
		return nil, err
	}

	ws.mu.RLock()
	defer ws.mu.RUnlock()

	skill, ok := ws.skills[name]
	if !ok {
		return nil, nil
	}

	// Return without indexableContent field
	s := skill.Skill
	return &s, nil
}

// Has checks if a skill exists.
func (ws *WorkspaceSkillsImpl) Has(name string) (bool, error) {
	if err := ws.ensureInitialized(); err != nil {
		return false, err
	}

	ws.mu.RLock()
	defer ws.mu.RUnlock()

	_, ok := ws.skills[name]
	return ok, nil
}

// Refresh re-scans skills from filesystem.
func (ws *WorkspaceSkillsImpl) Refresh() error {
	ws.mu.Lock()
	ws.skills = make(map[string]*internalSkill)
	if ws.searchEngine != nil {
		ws.searchEngine.Clear()
	}
	ws.initialized = false
	ws.initOnce = sync.Once{}
	ws.mu.Unlock()

	if err := ws.discoverSkills(); err != nil {
		return err
	}

	ws.mu.Lock()
	ws.initialized = true
	ws.mu.Unlock()

	return nil
}

// MaybeRefresh conditionally refreshes skills if they have been modified.
func (ws *WorkspaceSkillsImpl) MaybeRefresh(ctx *SkillsContext) error {
	if err := ws.ensureInitialized(); err != nil {
		return err
	}

	// Resolve current paths
	skillsCtx := SkillsContext{}
	if ctx != nil {
		skillsCtx = *ctx
	}
	currentPaths, err := ws.skillsResolver.ResolvePaths(skillsCtx)
	if err != nil {
		return err
	}

	// Check if paths have changed
	ws.mu.RLock()
	pathsChanged := !arePathsEqual(ws.resolvedPaths, currentPaths)
	ws.mu.RUnlock()

	if pathsChanged {
		ws.mu.Lock()
		ws.resolvedPaths = currentPaths
		ws.mu.Unlock()
		return ws.Refresh()
	}

	// Check if any skills path has been modified
	isStale, err := ws.isSkillsPathStale()
	if err != nil {
		return err
	}
	if isStale {
		return ws.Refresh()
	}

	return nil
}

// AddSkill surgically adds or updates a single skill in the cache.
func (ws *WorkspaceSkillsImpl) AddSkill(skillPath string) error {
	if err := ws.ensureInitialized(); err != nil {
		return err
	}

	// Determine SKILL.md path and dirName
	var skillFilePath, dirName string
	if strings.HasSuffix(skillPath, "/SKILL.md") || skillPath == "SKILL.md" {
		skillFilePath = skillPath
		parent := getParentPath(skillPath)
		parts := strings.Split(parent, "/")
		dirName = parts[len(parts)-1]
		if dirName == "" {
			dirName = "unknown"
		}
	} else {
		skillFilePath = skillJoinPath(skillPath, "SKILL.md")
		parts := strings.Split(skillPath, "/")
		dirName = parts[len(parts)-1]
		if dirName == "" {
			dirName = "unknown"
		}
	}

	// Determine source
	source := ws.inferSource(skillPath)

	// Parse and add
	skill, err := ws.parseSkillFile(skillFilePath, dirName, source)
	if err != nil {
		return err
	}

	ws.mu.Lock()
	// Remove old index entries if skill already exists
	if existing, ok := ws.skills[skill.Name]; ok {
		ws.mu.Unlock()
		ws.removeSkillFromIndex(existing)
		ws.mu.Lock()
	}

	ws.skills[skill.Name] = skill
	ws.lastDiscoveryTime = time.Now().UnixMilli()
	ws.mu.Unlock()

	ws.indexSkill(skill)

	return nil
}

// RemoveSkill surgically removes a single skill from the cache by name.
func (ws *WorkspaceSkillsImpl) RemoveSkill(skillName string) error {
	if err := ws.ensureInitialized(); err != nil {
		return err
	}

	ws.mu.Lock()
	skill, ok := ws.skills[skillName]
	if !ok {
		ws.mu.Unlock()
		return nil
	}
	delete(ws.skills, skillName)
	ws.lastDiscoveryTime = time.Now().UnixMilli()
	ws.mu.Unlock()

	ws.removeSkillFromIndex(skill)
	return nil
}

// =============================================================================
// Search
// =============================================================================

// Search searches across all skills content.
func (ws *WorkspaceSkillsImpl) Search(query string, options *SkillSearchOptions) ([]SkillSearchResult, error) {
	if err := ws.ensureInitialized(); err != nil {
		return nil, err
	}

	if options == nil {
		options = &SkillSearchOptions{}
	}

	if ws.searchEngine == nil {
		return ws.simpleSearch(query, options)
	}

	topK := options.TopK
	if topK == 0 {
		topK = 5
	}

	expandedTopK := topK
	if len(options.SkillNames) > 0 {
		expandedTopK = topK * 3
	}

	searchResults, err := ws.searchEngine.Search(query, &SkillEngineSearchOptions{
		TopK:     expandedTopK,
		MinScore: options.MinScore,
		Mode:     string(options.Mode),
	})
	if err != nil {
		return nil, err
	}

	var results []SkillSearchResult
	includeRefs := options.ShouldIncludeReferences()

	for _, result := range searchResults {
		skillName, _ := result.Metadata["skillName"].(string)
		sourceFile, _ := result.Metadata["source"].(string)

		if skillName == "" || sourceFile == "" {
			continue
		}

		if len(options.SkillNames) > 0 && !containsString(options.SkillNames, skillName) {
			continue
		}

		if !includeRefs && sourceFile != "SKILL.md" {
			continue
		}

		results = append(results, SkillSearchResult{
			BaseSearchResult: BaseSearchResult{
				Content:      result.Content,
				Score:        result.Score,
				ScoreDetails: result.ScoreDetails,
			},
			SkillName:  skillName,
			SourceFile: sourceFile,
		})

		if len(results) >= topK {
			break
		}
	}

	return results, nil
}

// =============================================================================
// Single-item Accessors
// =============================================================================

// GetReference gets reference file content from a skill.
func (ws *WorkspaceSkillsImpl) GetReference(skillName, referencePath string) (string, error) {
	if err := ws.ensureInitialized(); err != nil {
		return "", err
	}

	ws.mu.RLock()
	skill, ok := ws.skills[skillName]
	ws.mu.RUnlock()
	if !ok {
		return "", nil
	}

	safeRefPath, err := assertRelativePath(referencePath, "reference")
	if err != nil {
		return "", err
	}
	refFilePath := skillJoinPath(skill.Path, safeRefPath)

	exists, err := ws.source.Exists(refFilePath)
	if err != nil || !exists {
		return "", nil
	}

	content, err := ws.source.ReadFile(refFilePath)
	if err != nil {
		return "", nil
	}
	return string(content), nil
}

// GetScript gets script file content from a skill.
func (ws *WorkspaceSkillsImpl) GetScript(skillName, scriptPath string) (string, error) {
	if err := ws.ensureInitialized(); err != nil {
		return "", err
	}

	ws.mu.RLock()
	skill, ok := ws.skills[skillName]
	ws.mu.RUnlock()
	if !ok {
		return "", nil
	}

	safePath, err := assertRelativePath(scriptPath, "script")
	if err != nil {
		return "", err
	}
	filePath := skillJoinPath(skill.Path, safePath)

	exists, err := ws.source.Exists(filePath)
	if err != nil || !exists {
		return "", nil
	}

	content, err := ws.source.ReadFile(filePath)
	if err != nil {
		return "", nil
	}
	return string(content), nil
}

// GetAsset gets asset file content from a skill.
func (ws *WorkspaceSkillsImpl) GetAsset(skillName, assetPath string) ([]byte, error) {
	if err := ws.ensureInitialized(); err != nil {
		return nil, err
	}

	ws.mu.RLock()
	skill, ok := ws.skills[skillName]
	ws.mu.RUnlock()
	if !ok {
		return nil, nil
	}

	safePath, err := assertRelativePath(assetPath, "asset")
	if err != nil {
		return nil, err
	}
	filePath := skillJoinPath(skill.Path, safePath)

	exists, err := ws.source.Exists(filePath)
	if err != nil || !exists {
		return nil, nil
	}

	return ws.source.ReadFile(filePath)
}

// =============================================================================
// Listing Accessors
// =============================================================================

// ListReferences gets all reference file paths for a skill.
func (ws *WorkspaceSkillsImpl) ListReferences(skillName string) ([]string, error) {
	if err := ws.ensureInitialized(); err != nil {
		return nil, err
	}

	ws.mu.RLock()
	defer ws.mu.RUnlock()

	skill, ok := ws.skills[skillName]
	if !ok {
		return nil, nil
	}
	return skill.References, nil
}

// ListScripts gets all script file paths for a skill.
func (ws *WorkspaceSkillsImpl) ListScripts(skillName string) ([]string, error) {
	if err := ws.ensureInitialized(); err != nil {
		return nil, err
	}

	ws.mu.RLock()
	defer ws.mu.RUnlock()

	skill, ok := ws.skills[skillName]
	if !ok {
		return nil, nil
	}
	return skill.Scripts, nil
}

// ListAssets gets all asset file paths for a skill.
func (ws *WorkspaceSkillsImpl) ListAssets(skillName string) ([]string, error) {
	if err := ws.ensureInitialized(); err != nil {
		return nil, err
	}

	ws.mu.RLock()
	defer ws.mu.RUnlock()

	skill, ok := ws.skills[skillName]
	if !ok {
		return nil, nil
	}
	return skill.Assets, nil
}

// =============================================================================
// Private Methods
// =============================================================================

// ensureInitialized ensures skills have been discovered.
func (ws *WorkspaceSkillsImpl) ensureInitialized() error {
	ws.mu.RLock()
	if ws.initialized {
		ws.mu.RUnlock()
		return nil
	}
	ws.mu.RUnlock()

	ws.initOnce.Do(func() {
		// Resolve paths on first initialization
		ws.mu.Lock()
		if len(ws.resolvedPaths) == 0 {
			ws.resolvedPaths, ws.initErr = ws.skillsResolver.ResolvePaths(SkillsContext{})
			if ws.initErr != nil {
				ws.mu.Unlock()
				return
			}
		}
		ws.mu.Unlock()

		ws.initErr = ws.discoverSkills()
		if ws.initErr == nil {
			ws.mu.Lock()
			ws.initialized = true
			ws.mu.Unlock()
		}
	})

	return ws.initErr
}

// discoverSkills discovers skills from all skills paths.
func (ws *WorkspaceSkillsImpl) discoverSkills() error {
	ws.mu.RLock()
	paths := ws.resolvedPaths
	ws.mu.RUnlock()

	for _, rawSkillsPath := range paths {
		// Strip trailing slash
		skillsPath := rawSkillsPath
		if len(skillsPath) > 1 && strings.HasSuffix(skillsPath, "/") {
			skillsPath = skillsPath[:len(skillsPath)-1]
		}
		source := ws.determineSource(skillsPath)

		// Try as direct skill first
		isDirect, err := ws.discoverDirectSkill(skillsPath, source)
		if err != nil {
			continue
		}
		if !isDirect {
			// Plain path: scan subdirectories for skills
			ws.discoverSkillsInPath(skillsPath, source)
		}
	}

	ws.mu.Lock()
	ws.lastDiscoveryTime = time.Now().UnixMilli()
	ws.mu.Unlock()

	return nil
}

// discoverDirectSkill attempts to discover a skill from a direct path reference.
func (ws *WorkspaceSkillsImpl) discoverDirectSkill(skillsPath string, source ContentSource) (bool, error) {
	// Case 1: Path points directly to a SKILL.md file
	if strings.HasSuffix(skillsPath, "/SKILL.md") || skillsPath == "SKILL.md" {
		exists, err := ws.source.Exists(skillsPath)
		if err != nil || !exists {
			return true, nil
		}

		skillDir := getParentPath(skillsPath)
		parts := strings.Split(skillDir, "/")
		dirName := parts[len(parts)-1]
		if dirName == "" {
			dirName = skillDir
		}

		skill, err := ws.parseSkillFile(skillsPath, dirName, source)
		if err != nil {
			return true, nil
		}

		ws.mu.Lock()
		ws.skills[skill.Name] = skill
		ws.mu.Unlock()
		ws.indexSkill(skill)
		return true, nil
	}

	// Case 2: Path is a directory that directly contains SKILL.md
	exists, err := ws.source.Exists(skillsPath)
	if err != nil || !exists {
		return false, nil
	}

	skillFilePath := skillJoinPath(skillsPath, "SKILL.md")
	exists, err = ws.source.Exists(skillFilePath)
	if err != nil || !exists {
		return false, nil
	}

	parts := strings.Split(skillsPath, "/")
	dirName := parts[len(parts)-1]
	if dirName == "" {
		dirName = skillsPath
	}

	skill, err := ws.parseSkillFile(skillFilePath, dirName, source)
	if err != nil {
		return true, nil
	}

	ws.mu.Lock()
	ws.skills[skill.Name] = skill
	ws.mu.Unlock()
	ws.indexSkill(skill)
	return true, nil
}

// discoverSkillsInPath discovers skills in a single path.
func (ws *WorkspaceSkillsImpl) discoverSkillsInPath(skillsPath string, source ContentSource) {
	exists, err := ws.source.Exists(skillsPath)
	if err != nil || !exists {
		return
	}

	entries, err := ws.source.Readdir(skillsPath)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.Type != "directory" {
			continue
		}

		entryPath := skillJoinPath(skillsPath, entry.Name)
		skillFilePath := skillJoinPath(entryPath, "SKILL.md")

		exists, err := ws.source.Exists(skillFilePath)
		if err != nil || !exists {
			continue
		}

		skill, err := ws.parseSkillFile(skillFilePath, entry.Name, source)
		if err != nil {
			continue
		}

		ws.mu.Lock()
		ws.skills[skill.Name] = skill
		ws.mu.Unlock()
		ws.indexSkill(skill)
	}
}

// isSkillsPathStale checks if any skills path directory has been modified since last discovery.
func (ws *WorkspaceSkillsImpl) isSkillsPathStale() (bool, error) {
	ws.mu.RLock()
	lastDiscovery := ws.lastDiscoveryTime
	paths := ws.resolvedPaths
	ws.mu.RUnlock()

	if lastDiscovery == 0 {
		return true, nil
	}

	// Skip if discovery happened very recently
	if time.Now().UnixMilli()-lastDiscovery < stalenessCheckCooldown.Milliseconds() {
		return false, nil
	}

	for _, skillsPath := range paths {
		stat, err := ws.source.Stat(skillsPath)
		if err != nil {
			continue
		}

		if stat.ModifiedAt.UnixMilli() > lastDiscovery {
			return true, nil
		}

		if stat.Type != "directory" {
			continue
		}

		// Also check subdirectories
		entries, err := ws.source.Readdir(skillsPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.Type != "directory" {
				continue
			}

			entryPath := skillJoinPath(skillsPath, entry.Name)
			entryStat, err := ws.source.Stat(entryPath)
			if err != nil {
				continue
			}

			if entryStat.ModifiedAt.UnixMilli() > lastDiscovery {
				return true, nil
			}
		}
	}

	return false, nil
}

// parseSkillFile parses a SKILL.md file.
func (ws *WorkspaceSkillsImpl) parseSkillFile(filePath, dirName string, source ContentSource) (*internalSkill, error) {
	rawContent, err := ws.source.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	content := string(rawContent)
	frontmatter, body := parseSimpleFrontmatter(content)
	body = strings.TrimSpace(body)

	metadata := SkillMetadata{
		Name:        getString(frontmatter, "name"),
		Description: getString(frontmatter, "description"),
		License:     getString(frontmatter, "license"),
	}
	if v, ok := frontmatter["compatibility"]; ok {
		metadata.Compatibility = v
	}
	if v, ok := frontmatter["metadata"].(map[string]interface{}); ok {
		metadata.Metadata = v
	}

	// Validate if enabled
	if ws.validateOnLoad {
		result := ValidateSkillMetadata(metadata, dirName, body)
		if !result.Valid {
			return nil, fmt.Errorf("invalid skill metadata in %s:\n%s", filePath, strings.Join(result.Errors, "\n"))
		}
	}

	// Get skill directory path
	skillPath := getParentPath(filePath)

	// Discover reference, script, and asset files
	references := ws.discoverFilesInSubdir(skillPath, "references")
	scripts := ws.discoverFilesInSubdir(skillPath, "scripts")
	assets := ws.discoverFilesInSubdir(skillPath, "assets")

	// Build indexable content
	indexableContent := ws.buildIndexableContent(body, skillPath, references)

	return &internalSkill{
		Skill: Skill{
			SkillMetadata: metadata,
			Path:          skillPath,
			Instructions:  body,
			Source:        source,
			References:    references,
			Scripts:       scripts,
			Assets:        assets,
		},
		indexableContent: indexableContent,
	}, nil
}

// discoverFilesInSubdir discovers files in a subdirectory (references/, scripts/, assets/).
func (ws *WorkspaceSkillsImpl) discoverFilesInSubdir(skillPath, subdir string) []string {
	subdirPath := skillJoinPath(skillPath, subdir)

	exists, err := ws.source.Exists(subdirPath)
	if err != nil || !exists {
		return nil
	}

	var files []string
	ws.walkDirectory(subdirPath, subdirPath, func(relativePath string) {
		files = append(files, relativePath)
	}, 0, 20)

	return files
}

// walkDirectory walks a directory recursively.
func (ws *WorkspaceSkillsImpl) walkDirectory(basePath, dirPath string, callback func(string), depth, maxDepth int) {
	if depth >= maxDepth {
		return
	}

	entries, err := ws.source.Readdir(dirPath)
	if err != nil {
		return
	}

	for _, entry := range entries {
		entryPath := skillJoinPath(dirPath, entry.Name)

		if entry.Type == "directory" && !entry.IsSymlink {
			ws.walkDirectory(basePath, entryPath, callback, depth+1, maxDepth)
		} else {
			relativePath := entryPath
			if strings.HasPrefix(entryPath, basePath+"/") {
				relativePath = entryPath[len(basePath)+1:]
			}
			callback(relativePath)
		}
	}
}

// buildIndexableContent builds indexable content from instructions and references.
func (ws *WorkspaceSkillsImpl) buildIndexableContent(instructions, skillPath string, references []string) string {
	parts := []string{instructions}

	for _, refPath := range references {
		fullPath := skillJoinPath(skillPath, "references", refPath)
		content, err := ws.source.ReadFile(fullPath)
		if err == nil {
			parts = append(parts, string(content))
		}
	}

	return strings.Join(parts, "\n\n")
}

// indexSkill indexes a skill for search.
func (ws *WorkspaceSkillsImpl) indexSkill(skill *internalSkill) {
	if ws.searchEngine == nil {
		return
	}

	_ = ws.searchEngine.Index(SkillIndexDocument{
		ID:      fmt.Sprintf("skill:%s:SKILL.md", skill.Name),
		Content: skill.Instructions,
		Metadata: map[string]interface{}{
			"skillName": skill.Name,
			"source":    "SKILL.md",
		},
	})

	for _, refPath := range skill.References {
		fullPath := skillJoinPath(skill.Path, "references", refPath)
		content, err := ws.source.ReadFile(fullPath)
		if err != nil {
			continue
		}
		_ = ws.searchEngine.Index(SkillIndexDocument{
			ID:      fmt.Sprintf("skill:%s:%s", skill.Name, refPath),
			Content: string(content),
			Metadata: map[string]interface{}{
				"skillName": skill.Name,
				"source":    "references/" + refPath,
			},
		})
	}
}

// removeSkillFromIndex removes a skill's entries from the search index.
func (ws *WorkspaceSkillsImpl) removeSkillFromIndex(skill *internalSkill) {
	if ws.searchEngine == nil {
		return
	}

	ids := []string{fmt.Sprintf("skill:%s:SKILL.md", skill.Name)}
	for _, r := range skill.References {
		ids = append(ids, fmt.Sprintf("skill:%s:%s", skill.Name, r))
	}
	for _, id := range ids {
		_ = ws.searchEngine.Remove(id)
	}
}

// inferSource infers the ContentSource for a skill path.
func (ws *WorkspaceSkillsImpl) inferSource(skillPath string) ContentSource {
	ws.mu.RLock()
	paths := ws.resolvedPaths
	ws.mu.RUnlock()

	for _, rp := range paths {
		if skillPath == rp || strings.HasPrefix(skillPath, rp+"/") {
			return ws.determineSource(rp)
		}
	}
	return ws.determineSource(skillPath)
}

// determineSource determines the source type based on the path.
func (ws *WorkspaceSkillsImpl) determineSource(skillsPath string) ContentSource {
	segments := strings.Split(skillsPath, "/")
	for _, seg := range segments {
		if seg == "node_modules" {
			return ContentSource{Type: ContentSourceExternal, PackagePath: skillsPath}
		}
	}
	if strings.Contains(skillsPath, "/.mastra/skills") || strings.HasPrefix(skillsPath, ".mastra/skills") {
		return ContentSource{Type: ContentSourceManaged, MastraPath: skillsPath}
	}
	return ContentSource{Type: ContentSourceLocal, ProjectPath: skillsPath}
}

// simpleSearch is a fallback search when no search engine is configured.
func (ws *WorkspaceSkillsImpl) simpleSearch(query string, options *SkillSearchOptions) ([]SkillSearchResult, error) {
	topK := options.TopK
	if topK == 0 {
		topK = 5
	}
	queryLower := strings.ToLower(query)
	includeRefs := options.ShouldIncludeReferences()

	ws.mu.RLock()
	defer ws.mu.RUnlock()

	var results []SkillSearchResult

	for _, skill := range ws.skills {
		if len(options.SkillNames) > 0 && !containsString(options.SkillNames, skill.Name) {
			continue
		}

		if strings.Contains(strings.ToLower(skill.Instructions), queryLower) {
			truncated := skill.Instructions
			if len(truncated) > 200 {
				truncated = truncated[:200]
			}
			results = append(results, SkillSearchResult{
				BaseSearchResult: BaseSearchResult{
					Content: truncated,
					Score:   1,
				},
				SkillName:  skill.Name,
				SourceFile: "SKILL.md",
			})
		}

		if includeRefs {
			for _, refPath := range skill.References {
				if len(results) >= topK {
					break
				}
				fullPath := skillJoinPath(skill.Path, "references", refPath)
				content, err := ws.source.ReadFile(fullPath)
				if err != nil {
					continue
				}
				contentStr := string(content)
				if strings.Contains(strings.ToLower(contentStr), queryLower) {
					truncated := contentStr
					if len(truncated) > 200 {
						truncated = truncated[:200]
					}
					results = append(results, SkillSearchResult{
						BaseSearchResult: BaseSearchResult{
							Content: truncated,
							Score:   0.8,
						},
						SkillName:  skill.Name,
						SourceFile: "references/" + refPath,
					})
				}
			}
		}

		if len(results) >= topK {
			break
		}
	}

	if len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// skillJoinPath joins path segments using forward slashes.
func skillJoinPath(segments ...string) string {
	var result []string
	for i, seg := range segments {
		if i == 0 {
			result = append(result, strings.TrimRight(seg, "/"))
		} else {
			trimmed := strings.Trim(seg, "/")
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
	}
	return strings.Join(result, "/")
}

// getParentPath gets the parent path.
func getParentPath(path string) string {
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash > 0 {
		return path[:lastSlash]
	}
	return "/"
}

// assertRelativePath validates and normalizes a relative path to prevent directory traversal.
func assertRelativePath(input, label string) (string, error) {
	normalized := strings.ReplaceAll(input, "\\", "/")
	segments := strings.Split(normalized, "/")
	var filtered []string
	for _, seg := range segments {
		if seg == "" || seg == "." {
			continue
		}
		if seg == ".." {
			return "", fmt.Errorf("invalid %s path: %s", label, input)
		}
		filtered = append(filtered, seg)
	}
	if strings.HasPrefix(normalized, "/") {
		return "", fmt.Errorf("invalid %s path: %s", label, input)
	}
	return strings.Join(filtered, "/"), nil
}

// arePathsEqual compares two path arrays for equality (order-independent).
func arePathsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sortedA := make([]string, len(a))
	copy(sortedA, a)
	sort.Strings(sortedA)

	sortedB := make([]string, len(b))
	copy(sortedB, b)
	sort.Strings(sortedB)

	for i := range sortedA {
		if sortedA[i] != sortedB[i] {
			return false
		}
	}
	return true
}

// containsString checks if a slice contains a string.
func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
