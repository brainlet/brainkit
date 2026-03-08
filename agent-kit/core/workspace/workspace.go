// Ported from: packages/core/src/workspace/workspace.ts
package workspace

import (
	"fmt"
	"math/rand"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/logger"
	"github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	"github.com/brainlet/brainkit/agent-kit/core/workspace/lsp"
	"github.com/brainlet/brainkit/agent-kit/core/workspace/search"
	"github.com/brainlet/brainkit/agent-kit/core/workspace/skills"
)

// =============================================================================
// Stub types for sub-packages that import the parent workspace package.
// These MUST remain as stubs to avoid circular imports:
//   workspace -> workspace/filesystem -> workspace (cycle)
//   workspace -> workspace/sandbox   -> (no cycle, but kept for interface
//     signature consistency with the workspace-level API)
// =============================================================================

// WorkspaceFilesystem is the abstract filesystem interface for workspace storage.
// Kept as a stub because workspace/filesystem imports the parent workspace package,
// which would create a circular import if we imported it here.
type WorkspaceFilesystem interface {
	// Identity
	ID() string
	Name() string
	Provider() string
	ReadOnly() bool
	BasePath() string
	Icon() *FilesystemIcon
	DisplayName() string
	Description() string

	// Instructions
	GetInstructions(opts *InstructionsOpts) string

	// Mount config
	GetMountConfig() *FilesystemMountConfig

	// File operations
	ReadFile(path string, options *ReadOptions) (interface{}, error)
	WriteFile(path string, content interface{}, options *WriteOptions) error
	AppendFile(path string, content interface{}) error
	DeleteFile(path string, options *RemoveOptions) error
	CopyFile(src, dest string, options *CopyOptions) error
	MoveFile(src, dest string, options *CopyOptions) error

	// Directory operations
	Mkdir(path string, options *MkdirOptions) error
	Rmdir(path string, options *RemoveOptions) error
	Readdir(path string, options *ListOptions) ([]FileEntry, error)

	// Path operations
	ResolveAbsolutePath(path string) string
	Exists(path string) (bool, error)
	Stat(path string) (*FileStat, error)

	// Lifecycle
	Init() error
	Destroy() error
	GetInfo() (*FilesystemInfo, error)

	// Status
	Status() ProviderStatus
}

// InstructionsOpts holds options for GetInstructions calls.
type InstructionsOpts struct {
	RequestContext *requestcontext.RequestContext
}

// FilesystemIcon represents an icon identifier for UI display.
// Kept as a stub to avoid circular import with workspace/filesystem.
type FilesystemIcon string

// FilesystemMountConfig holds mount configuration for a filesystem.
// Kept as a stub to avoid circular import with workspace/filesystem.
type FilesystemMountConfig struct{}

// ReadOptions configures file read behavior.
// Kept as a stub to avoid circular import with workspace/filesystem.
type ReadOptions struct {
	Encoding string
}

// WriteOptions configures file write behavior.
// Kept as a stub to avoid circular import with workspace/filesystem.
type WriteOptions struct {
	Recursive bool
	Overwrite *bool
	MimeType  string
}

// ListOptions configures directory listing behavior.
// Kept as a stub to avoid circular import with workspace/filesystem.
type ListOptions struct {
	Recursive bool
	Extension []string
	MaxDepth  int
}

// RemoveOptions configures file/directory removal behavior.
// Kept as a stub to avoid circular import with workspace/filesystem.
type RemoveOptions struct {
	Recursive bool
	Force     bool
}

// CopyOptions configures file copy behavior.
// Kept as a stub to avoid circular import with workspace/filesystem.
type CopyOptions struct {
	Overwrite bool
	Recursive bool
}

// MkdirOptions configures directory creation behavior.
// Kept as a stub to avoid circular import with workspace/filesystem.
type MkdirOptions struct {
	Recursive bool
}

// FileStat represents file/directory metadata.
// Kept as a stub to avoid circular import with workspace/filesystem.
type FileStat struct {
	Name       string
	Path       string
	Type       string // "file" or "directory"
	Size       int64
	CreatedAt  time.Time
	ModifiedAt time.Time
	MimeType   string
}

// FileEntry represents a directory entry.
// Kept as a stub to avoid circular import with workspace/filesystem.
type FileEntry struct {
	Name          string
	Type          string // "file" or "directory"
	Size          int64
	IsSymlink     bool
	SymlinkTarget string
	Mount         *FileEntryMount
}

// FileEntryMount holds mount point metadata for a directory entry.
type FileEntryMount struct {
	Provider    string
	Icon        *FilesystemIcon
	DisplayName string
	Description string
	Status      *ProviderStatus
	Error       string
}

// FilesystemInfo holds information about a filesystem provider's state.
// Kept as a stub to avoid circular import with workspace/filesystem.
type FilesystemInfo struct {
	ID       string
	Name     string
	Provider string
	Status   *ProviderStatus
	Error    string
	ReadOnly bool
	Icon     *FilesystemIcon
	Metadata map[string]interface{}
}

// WorkspaceSandbox is the abstract sandbox interface for code execution.
// Kept as a stub to maintain workspace-level API consistency.
// The sandbox sub-package defines its own WorkspaceSandbox with slightly
// different method signatures (e.g., GetInstructions() vs GetInstructions(opts)).
type WorkspaceSandbox interface {
	// Identity
	ID() string
	Name() string
	Provider() string
	Status() ProviderStatus

	// Instructions
	GetInstructions(opts *InstructionsOpts) string

	// Lifecycle
	Start() error
	Stop() error
	Destroy() error
	GetInfo() (*SandboxInfo, error)

	// Command execution
	ExecuteCommand(command string, args []string, options *ExecuteCommandOptions) (*CommandResult, error)

	// Process management (optional)
	Processes() SandboxProcessManager

	// Mount management (optional)
	Mounts() MountManager
	Mount(filesystem WorkspaceFilesystem, mountPath string) (*MountResult, error)
	Unmount(mountPath string) error
}

// ExecuteCommandOptions configures command execution.
// Kept as a stub to avoid potential circular import with workspace/sandbox.
type ExecuteCommandOptions struct {
	Cwd     string
	Timeout time.Duration
}

// CommandResult holds the result of a command execution.
// Kept as a stub to avoid potential circular import with workspace/sandbox.
type CommandResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// SandboxInfo holds information about a sandbox provider's state.
// Kept as a stub to avoid potential circular import with workspace/sandbox.
type SandboxInfo struct {
	Status    string
	Resources *SandboxResources
}

// SandboxResources holds sandbox resource information.
type SandboxResources struct {
	MemoryMB     *float64
	MemoryUsedMB *float64
	CPUCores     *float64
	CPUPercent   *float64
	DiskMB       *float64
	DiskUsedMB   *float64
}

// MountResult holds the result of a mount operation.
// Kept as a stub to avoid potential circular import with workspace/sandbox.
type MountResult struct {
	Success   bool
	MountPath string
	Error     string
}

// OnMountHook is a hook called before mounting each filesystem into the sandbox.
// Kept as a stub to avoid potential circular import with workspace/sandbox.
type OnMountHook func(args OnMountArgs) *OnMountResult

// OnMountArgs holds arguments for the OnMount hook.
type OnMountArgs struct {
	Filesystem WorkspaceFilesystem
	MountPath  string
	Sandbox    WorkspaceSandbox
}

// OnMountResult holds the result of an OnMount hook.
type OnMountResult struct {
	Success bool
	Error   string
}

// SandboxProcessManager is the process management interface for sandboxes.
// Cannot use sandbox.ProcessManager directly because it references sandbox-local types
// (SpawnProcessOptions, ProcessHandle, ProcessInfo) in its method signatures.
// These methods mirror the real sandbox.ProcessManager with simplified types
// so the workspace package can define the WorkspaceSandbox.Processes() return type
// without importing sandbox-specific value types.
type SandboxProcessManager interface {
	// List returns info about all tracked processes.
	List() ([]any, error)
	// Kill kills a process by PID. Returns true if the process was found and killed.
	Kill(pid int) (bool, error)
}

// MountManager manages filesystem mounts in a sandbox.
// Kept as a stub to avoid potential circular import with workspace/sandbox.
type MountManager interface {
	SetContext(ctx MountManagerContext)
	Add(mounts map[string]WorkspaceFilesystem)
	SetOnMount(hook OnMountHook)
	Entries() map[string]*MountEntry
}

// MountManagerContext holds the context for a mount manager.
type MountManagerContext struct {
	Sandbox   WorkspaceSandbox
	Workspace *Workspace
}

// MountEntry represents a mount entry tracked by the mount manager.
type MountEntry struct {
	Filesystem WorkspaceFilesystem
	State      string // "pending", "mounting", "mounted", "error", "unsupported", "unavailable"
}

// CompositeFilesystem is a filesystem that routes operations based on path.
// Kept as a stub to avoid circular import with workspace/filesystem.
type CompositeFilesystem struct{}

// LocalFilesystem is a filesystem backed by a local folder.
// Kept as a stub to avoid circular import with workspace/filesystem.
type LocalFilesystem struct {
	Contained bool
}

// WorkingDirectoryProvider is implemented by sandboxes that expose a working directory.
// Used to detect LocalSandbox-like providers without requiring full interface implementation.
type WorkingDirectoryProvider interface {
	GetWorkingDirectory() string
}

// LocalSandbox is a development-only local sandbox.
// Kept as a stub to avoid potential circular import with workspace/sandbox.
type LocalSandbox struct {
	WorkingDirectory string
}

// GetWorkingDirectory implements WorkingDirectoryProvider.
func (ls *LocalSandbox) GetWorkingDirectory() string {
	return ls.WorkingDirectory
}

// MastraFilesystem is the base class for filesystem providers with logger integration.
// Kept as a stub to avoid circular import with workspace/filesystem.
type MastraFilesystem struct{}

// SetLogger sets the logger on a MastraFilesystem.
func (mf *MastraFilesystem) SetLogger(_ logger.IMastraLogger) {}

// MastraSandbox is the base class for sandbox providers with logger integration.
// Kept as a stub to avoid potential circular import with workspace/sandbox.
type MastraSandbox struct{}

// SetLogger sets the logger on a MastraSandbox.
func (ms *MastraSandbox) SetLogger(_ logger.IMastraLogger) {}

// =============================================================================
// Per-tool Configuration
// =============================================================================

// WorkspaceToolsConfig holds per-tool configuration.
type WorkspaceToolsConfig map[string]WorkspaceToolConfig

// WorkspaceToolConfig configures a single workspace tool.
type WorkspaceToolConfig struct {
	Enabled               *bool
	RequireApproval       *bool
	RequireReadBeforeWrite *bool
}

// =============================================================================
// Workspace Configuration
// =============================================================================

// WorkspaceConfig holds configuration for creating a Workspace.
type WorkspaceConfig struct {
	// ID is a unique identifier (auto-generated if not provided).
	ID string

	// Name is a human-readable name.
	Name string

	// Filesystem is the filesystem provider instance.
	Filesystem WorkspaceFilesystem

	// Sandbox is the sandbox provider instance.
	Sandbox WorkspaceSandbox

	// Mounts maps paths to filesystem providers for composite filesystem.
	// Cannot be used together with Filesystem.
	Mounts map[string]WorkspaceFilesystem

	// OnMount is a hook called before mounting each filesystem into the sandbox.
	OnMount OnMountHook

	// VectorStore for semantic search.
	VectorStore search.MastraVector

	// Embedder function for generating vectors.
	Embedder search.Embedder

	// BM25 enables BM25 keyword search.
	// nil = disabled, non-nil = enabled.
	BM25 *search.BM25Config

	// BM25Enabled enables BM25 with default config when true (and BM25 is nil).
	BM25Enabled bool

	// SearchIndexName is a custom index name for the vector store.
	SearchIndexName string

	// AutoIndexPaths are paths to auto-index on Init().
	AutoIndexPaths []string

	// Skills are paths where skills (SKILL.md files) are located.
	Skills skills.SkillsResolver

	// SkillSource is a custom SkillSource for skill discovery.
	SkillSource skills.SkillSource

	// LSP enables LSP diagnostics for edit tools.
	// nil = disabled. Non-nil = enabled.
	LSP *lsp.LSPConfig

	// LSPEnabled enables LSP with default config when true (and LSP is nil).
	LSPEnabled bool

	// Tools holds per-tool configuration.
	Tools WorkspaceToolsConfig

	// AutoSync enables auto-sync between fs and sandbox (default: false).
	AutoSync bool

	// OperationTimeout is the timeout for individual operations in milliseconds.
	OperationTimeout int
}

// AnyWorkspace is a type alias for a Workspace with any configuration.
type AnyWorkspace = *Workspace

// RegisteredWorkspace is a workspace entry in the Mastra registry.
type RegisteredWorkspace struct {
	Workspace *Workspace
	Source    string // "mastra" or "agent"
	AgentID   string
	AgentName string
}

// =============================================================================
// Path Context Types
// =============================================================================

// PathContext contains information about how filesystem and sandbox paths relate.
type PathContext struct {
	// Filesystem details (if available).
	Filesystem *PathContextFilesystem

	// Sandbox details (if available).
	Sandbox *PathContextSandbox

	// Instructions is a human-readable string describing how to access
	// filesystem files from sandbox code.
	Instructions string
}

// PathContextFilesystem holds filesystem details for PathContext.
type PathContextFilesystem struct {
	Provider string
	BasePath string
}

// PathContextSandbox holds sandbox details for PathContext.
type PathContextSandbox struct {
	Provider         string
	WorkingDirectory string
}

// WorkspaceInfoResult holds workspace information.
type WorkspaceInfoResult struct {
	ID             string
	Name           string
	Status         WorkspaceStatus
	CreatedAt      time.Time
	LastAccessedAt time.Time
	Filesystem     *WorkspaceInfoFilesystem
	Sandbox        *WorkspaceInfoSandbox
}

// WorkspaceInfoFilesystem holds filesystem info for WorkspaceInfoResult.
type WorkspaceInfoFilesystem struct {
	FilesystemInfo
	TotalFiles *int
	TotalSize  *int64
}

// WorkspaceInfoSandbox holds sandbox info for WorkspaceInfoResult.
type WorkspaceInfoSandbox struct {
	Provider  string
	Status    string
	Resources *SandboxResources
}

// =============================================================================
// Workspace Class
// =============================================================================

// Workspace provides agents with filesystem and execution capabilities.
//
// At minimum, a workspace has either a filesystem or a sandbox (or both).
// Users pass instantiated provider objects to the constructor.
type Workspace struct {
	ID             string
	WorkspaceName  string
	CreatedAt      time.Time
	LastAccessedAt time.Time

	status       WorkspaceStatus
	fs           WorkspaceFilesystem
	sandbox      WorkspaceSandbox
	config       *WorkspaceConfig
	searchEngine *search.SearchEngine
	skills       skills.WorkspaceSkills
	lsp          *lsp.LSPManager
}

// NewWorkspace creates a new Workspace from the given configuration.
func NewWorkspace(config WorkspaceConfig) (*Workspace, error) {
	w := &Workspace{
		ID:             config.ID,
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
		status:         WorkspaceStatusPending,
		config:         &config,
		sandbox:        config.Sandbox,
	}

	if w.ID == "" {
		w.ID = w.generateID()
	}

	w.WorkspaceName = config.Name
	if w.WorkspaceName == "" {
		w.WorkspaceName = fmt.Sprintf("workspace-%s", w.ID[:min(8, len(w.ID))])
	}

	// Setup mounts - creates CompositeFilesystem and informs sandbox
	if len(config.Mounts) > 0 {
		// Validate: can't use both filesystem and mounts
		if config.Filesystem != nil {
			return nil, NewWorkspaceError("Cannot use both \"filesystem\" and \"mounts\"", "INVALID_CONFIG")
		}

		// NOTE: In the TypeScript version, this creates a CompositeFilesystem
		// and sets up mount tracking on the sandbox. Since CompositeFilesystem
		// is in workspace/filesystem (which imports workspace, creating a cycle),
		// we store the first mount's filesystem as a placeholder.
		for _, fs := range config.Mounts {
			w.fs = fs
			break
		}

		if w.sandbox != nil {
			mounts := w.sandbox.Mounts()
			if mounts != nil {
				mounts.SetContext(MountManagerContext{
					Sandbox:   w.sandbox,
					Workspace: w,
				})
				mounts.Add(config.Mounts)
				if config.OnMount != nil {
					mounts.SetOnMount(config.OnMount)
				}
			}
		}
	} else {
		w.fs = config.Filesystem
	}

	// Validate vector search config - embedder is required with vectorStore
	if config.VectorStore != nil && config.Embedder == nil {
		return nil, NewWorkspaceError("vectorStore requires an embedder", "INVALID_SEARCH_CONFIG")
	}

	// Create search engine if search is configured
	bm25Enabled := config.BM25Enabled || config.BM25 != nil
	vectorEnabled := config.VectorStore != nil && config.Embedder != nil

	if bm25Enabled || vectorEnabled {
		buildIndexName := func() (string, error) {
			// Sanitize default name: replace all non-alphanumeric chars with underscores
			re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
			defaultName := re.ReplaceAllString(w.ID+"_search", "_")
			indexName := config.SearchIndexName
			if indexName == "" {
				indexName = defaultName
			}

			// Validate SQL identifier format
			validID := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
			if !validID.MatchString(indexName) {
				return "", NewWorkspaceError(
					fmt.Sprintf("Invalid searchIndexName: %q. Must start with a letter or underscore, and contain only letters, numbers, or underscores.", indexName),
					"INVALID_SEARCH_CONFIG",
					w.ID,
				)
			}
			if len(indexName) > 63 {
				return "", NewWorkspaceError(
					fmt.Sprintf("searchIndexName exceeds 63 characters (got %d)", len(indexName)),
					"INVALID_SEARCH_CONFIG",
					w.ID,
				)
			}
			return indexName, nil
		}

		seConfig := &search.SearchEngineConfig{}
		if bm25Enabled {
			seConfig.BM25 = &search.BM25SearchConfig{
				BM25: config.BM25,
			}
		}
		if vectorEnabled {
			indexName, err := buildIndexName()
			if err != nil {
				return nil, err
			}
			seConfig.Vector = &search.VectorConfig{
				VectorStore: config.VectorStore,
				Embedder:    config.Embedder,
				IndexName:   indexName,
			}
		}
		w.searchEngine = search.NewSearchEngine(seConfig)
	}

	// NOTE: LSP initialization is skipped in this port because it requires
	// a sandbox process manager (ProcessSpawner interface) and optional peer
	// dependencies. The TypeScript version checks for sandbox process manager
	// availability.
	// TODO: Implement LSP initialization by wiring lsp.NewLSPManager with the
	// sandbox's process manager once the full integration is validated.

	// Validate at least one provider is given
	if w.fs == nil && w.sandbox == nil && !w.hasSkillsConfig() {
		return nil, NewWorkspaceError("Workspace requires at least a filesystem, sandbox, or skills", "NO_PROVIDERS")
	}

	return w, nil
}

func (w *Workspace) generateID() string {
	return fmt.Sprintf("ws-%s-%s",
		fmt.Sprintf("%x", time.Now().UnixMilli()),
		randomAlphanumeric(6),
	)
}

func randomAlphanumeric(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func (w *Workspace) hasSkillsConfig() bool {
	return w.config.Skills != nil
}

// Status returns the current workspace status.
func (w *Workspace) Status() WorkspaceStatus {
	return w.status
}

// Filesystem returns the filesystem provider (if configured).
func (w *Workspace) Filesystem() WorkspaceFilesystem {
	return w.fs
}

// Sandbox returns the sandbox provider (if configured).
func (w *Workspace) Sandbox() WorkspaceSandbox {
	return w.sandbox
}

// GetToolsConfig returns the per-tool configuration for this workspace.
func (w *Workspace) GetToolsConfig() WorkspaceToolsConfig {
	return w.config.Tools
}

// LSP returns the LSP manager (if configured and initialized).
func (w *Workspace) LSP() *lsp.LSPManager {
	return w.lsp
}

// SetToolsConfig updates the per-tool configuration for this workspace.
func (w *Workspace) SetToolsConfig(config WorkspaceToolsConfig) {
	w.config.Tools = config
}

// Skills returns the skills interface (if configured).
func (w *Workspace) Skills() skills.WorkspaceSkills {
	if !w.hasSkillsConfig() {
		return nil
	}
	// NOTE: Lazy initialization of skills is not implemented because
	// WorkspaceSkillsImpl requires a SkillSource and SearchEngine that
	// need to be configured from the skill sub-package.
	// TODO: Implement lazy initialization by calling skills.NewWorkspaceSkillsImpl
	// with the appropriate SkillSource and search configuration.
	return w.skills
}

// CanBM25 checks if BM25 keyword search is available.
func (w *Workspace) CanBM25() bool {
	if w.searchEngine == nil {
		return false
	}
	return w.searchEngine.CanBM25()
}

// CanVector checks if vector semantic search is available.
func (w *Workspace) CanVector() bool {
	if w.searchEngine == nil {
		return false
	}
	return w.searchEngine.CanVector()
}

// CanHybrid checks if hybrid search is available.
func (w *Workspace) CanHybrid() bool {
	if w.searchEngine == nil {
		return false
	}
	return w.searchEngine.CanHybrid()
}

// IndexOptions configures content indexing.
type IndexOptions struct {
	Type            string // "text", "image", "file"
	MimeType        string
	Metadata        map[string]interface{}
	StartLineOffset int
}

// Index indexes content for search.
// The filePath becomes the document ID in search results.
func (w *Workspace) Index(filePath, content string, options *IndexOptions) error {
	if w.searchEngine == nil {
		return NewSearchNotAvailableError()
	}
	w.LastAccessedAt = time.Now()

	doc := search.IndexDocument{
		ID:      filePath,
		Content: content,
	}

	if options != nil {
		metadata := make(map[string]interface{})
		if options.Type != "" {
			metadata["type"] = options.Type
		}
		if options.MimeType != "" {
			metadata["mimeType"] = options.MimeType
		}
		for k, v := range options.Metadata {
			metadata[k] = v
		}
		doc.Metadata = metadata
		doc.StartLineOffset = options.StartLineOffset
	}

	return w.searchEngine.Index(doc)
}

// Search searches indexed content.
func (w *Workspace) Search(query string, options *search.SearchOptions) ([]search.SearchResult, error) {
	if w.searchEngine == nil {
		return nil, NewSearchNotAvailableError()
	}
	w.LastAccessedAt = time.Now()
	return w.searchEngine.Search(query, options)
}

// rebuildSearchIndex rebuilds the search index from filesystem paths.
func (w *Workspace) rebuildSearchIndex(paths []string) error {
	if w.searchEngine == nil || w.fs == nil || len(paths) == 0 {
		return nil
	}

	w.searchEngine.Clear()

	// Adapt filesystem readdir to the ReaddirEntry interface
	readdir := func(dir string) ([]ReaddirEntry, error) {
		entries, err := w.fs.Readdir(dir, nil)
		if err != nil {
			return nil, err
		}
		result := make([]ReaddirEntry, len(entries))
		for i, e := range entries {
			result[i] = ReaddirEntry{
				Name:      e.Name,
				Type:      e.Type,
				IsSymlink: e.IsSymlink,
			}
		}
		return result, nil
	}

	indexedPaths := make(map[string]bool)
	for _, pathOrGlob := range paths {
		resolved, err := ResolvePathPattern(pathOrGlob, readdir, nil)
		if err != nil {
			continue
		}

		var filesToIndex []string
		var directoryRoots []string

		for _, entry := range resolved {
			if entry.Type == "file" {
				filesToIndex = append(filesToIndex, entry.Path)
				continue
			}
			// Skip directories already covered by a parent directory
			alreadyCovered := false
			for _, root := range directoryRoots {
				if entry.Path == root {
					alreadyCovered = true
					break
				}
				prefix := root + "/"
				if root == "/" {
					prefix = "/"
				}
				if strings.HasPrefix(entry.Path, prefix) {
					alreadyCovered = true
					break
				}
			}
			if !alreadyCovered {
				directoryRoots = append(directoryRoots, entry.Path)
			}
		}

		// Index direct file matches first
		for _, filePath := range filesToIndex {
			if indexedPaths[filePath] {
				continue
			}
			_ = w.indexFileForSearch(filePath)
			indexedPaths[filePath] = true
		}

		for _, dir := range directoryRoots {
			files := w.getAllFiles(dir, 0, 10)
			for _, filePath := range files {
				if !indexedPaths[filePath] {
					_ = w.indexFileForSearch(filePath)
					indexedPaths[filePath] = true
				}
			}
		}
	}

	return nil
}

// indexFileForSearch indexes a single file for search.
func (w *Workspace) indexFileForSearch(filePath string) error {
	raw, err := w.fs.ReadFile(filePath, &ReadOptions{Encoding: "utf-8"})
	if err != nil {
		return err
	}
	content, ok := raw.(string)
	if !ok {
		return nil
	}
	return w.searchEngine.Index(search.IndexDocument{
		ID:      filePath,
		Content: content,
	})
}

// getAllFiles returns all files in a directory recursively.
func (w *Workspace) getAllFiles(dir string, depth, maxDepth int) []string {
	if w.fs == nil || depth >= maxDepth {
		return nil
	}

	entries, err := w.fs.Readdir(dir, nil)
	if err != nil {
		return nil
	}

	var files []string
	for _, entry := range entries {
		var fullPath string
		if dir == "/" {
			fullPath = "/" + entry.Name
		} else {
			fullPath = dir + "/" + entry.Name
		}
		if entry.Type == "file" {
			files = append(files, fullPath)
		} else if entry.Type == "directory" && !entry.IsSymlink {
			files = append(files, w.getAllFiles(fullPath, depth+1, maxDepth)...)
		}
	}

	return files
}

// =============================================================================
// Lifecycle
// =============================================================================

// Init initializes the workspace.
// Starts the sandbox, initializes the filesystem, and auto-mounts filesystems.
func (w *Workspace) Init() error {
	w.status = WorkspaceStatusInitializing

	if w.fs != nil {
		if err := CallLifecycle(w.fs, "init"); err != nil {
			w.status = WorkspaceStatusError
			return err
		}
	}

	if w.sandbox != nil {
		if err := CallLifecycle(w.sandbox, "start"); err != nil {
			w.status = WorkspaceStatusError
			return err
		}
	}

	// Auto-index files if autoIndexPaths is configured
	if w.searchEngine != nil && len(w.config.AutoIndexPaths) > 0 {
		if err := w.rebuildSearchIndex(w.config.AutoIndexPaths); err != nil {
			w.status = WorkspaceStatusError
			return err
		}
	}

	w.status = WorkspaceStatusReady
	return nil
}

// Destroy destroys the workspace and cleans up all resources.
func (w *Workspace) Destroy() error {
	w.status = WorkspaceStatusDestroying

	// Shutdown LSP before sandbox
	if w.lsp != nil {
		w.lsp.ShutdownAll() // LSP shutdown errors are non-blocking
		w.lsp = nil
	}

	if w.sandbox != nil {
		if err := CallLifecycle(w.sandbox, "destroy"); err != nil {
			w.status = WorkspaceStatusError
			return err
		}
	}

	if w.fs != nil {
		if err := CallLifecycle(w.fs, "destroy"); err != nil {
			w.status = WorkspaceStatusError
			return err
		}
	}

	w.status = WorkspaceStatusDestroyed
	return nil
}

// GetInfoOptions configures GetInfo behavior.
type GetInfoOptions struct {
	IncludeFileCount bool
}

// GetInfo returns workspace information.
func (w *Workspace) GetInfo(options *GetInfoOptions) (*WorkspaceInfoResult, error) {
	info := &WorkspaceInfoResult{
		ID:             w.ID,
		Name:           w.WorkspaceName,
		Status:         w.status,
		CreatedAt:      w.CreatedAt,
		LastAccessedAt: w.LastAccessedAt,
	}

	if w.fs != nil {
		fsInfo, _ := w.fs.GetInfo()
		fsResult := &WorkspaceInfoFilesystem{}
		if fsInfo != nil {
			fsResult.FilesystemInfo = *fsInfo
		} else {
			fsResult.ID = w.fs.ID()
			fsResult.Name = w.fs.Name()
			fsResult.Provider = w.fs.Provider()
			fsResult.ReadOnly = w.fs.ReadOnly()
		}

		if options != nil && options.IncludeFileCount {
			files := w.getAllFiles("/", 0, 10)
			count := len(files)
			fsResult.TotalFiles = &count
		}
		info.Filesystem = fsResult
	}

	if w.sandbox != nil {
		sandboxInfo, _ := w.sandbox.GetInfo()
		sbResult := &WorkspaceInfoSandbox{
			Provider: w.sandbox.Provider(),
			Status:   string(w.sandbox.Status()),
		}
		if sandboxInfo != nil {
			sbResult.Status = sandboxInfo.Status
			sbResult.Resources = sandboxInfo.Resources
		}
		info.Sandbox = sbResult
	}

	return info, nil
}

// GetInstructions returns human-readable instructions describing the workspace environment.
func (w *Workspace) GetInstructions(opts *InstructionsOpts) string {
	var parts []string

	// Sandbox-level instructions
	if w.sandbox != nil {
		sandboxInstructions := w.sandbox.GetInstructions(opts)
		if sandboxInstructions != "" {
			parts = append(parts, sandboxInstructions)
		}
	}

	// Mount state overlay: check actual MountManager state
	if w.sandbox != nil {
		mounts := w.sandbox.Mounts()
		if mounts != nil {
			mountEntries := mounts.Entries()
			if len(mountEntries) > 0 {
				var sandboxAccessible []string
				var workspaceOnly []string

				var workingDir string
				if wdp, ok := w.sandbox.(WorkingDirectoryProvider); ok {
					workingDir = wdp.GetWorkingDirectory()
				}

				for mountPath, entry := range mountEntries {
					fsName := entry.Filesystem.DisplayName()
					if fsName == "" {
						fsName = entry.Filesystem.Provider()
					}
					access := "read-write"
					if entry.Filesystem.ReadOnly() {
						access = "read-only"
					}

					displayPath := mountPath
					if workingDir != "" {
						displayPath = path.Join(workingDir, strings.TrimLeft(mountPath, "/"))
					}

					switch entry.State {
					case "mounted", "pending", "mounting":
						sandboxAccessible = append(sandboxAccessible, fmt.Sprintf("  - %s: %s (%s)", displayPath, fsName, access))
					default:
						workspaceOnly = append(workspaceOnly, fmt.Sprintf("  - %s: %s (%s)", mountPath, fsName, access))
					}
				}

				if len(sandboxAccessible) > 0 {
					parts = append(parts, "Sandbox-mounted filesystems (accessible in shell commands):\n"+strings.Join(sandboxAccessible, "\n"))
				}
				if len(workspaceOnly) > 0 {
					parts = append(parts, "Workspace-only filesystems (use file tools, NOT available in shell commands):\n"+strings.Join(workspaceOnly, "\n"))
				}

				return strings.Join(parts, "\n\n")
			}
		}
	}

	// No mounts or no sandbox — fall back to filesystem-level instructions
	if w.fs != nil {
		fsInstructions := w.fs.GetInstructions(opts)
		if fsInstructions != "" {
			parts = append(parts, fsInstructions)
		}
	}

	return strings.Join(parts, "\n\n")
}

// GetPathContext returns information about how filesystem and sandbox paths relate.
//
// Deprecated: Use GetInstructions instead.
func (w *Workspace) GetPathContext() PathContext {
	fsInstructions := ""
	if w.fs != nil {
		fsInstructions = w.fs.GetInstructions(nil)
	}
	sandboxInstructions := ""
	if w.sandbox != nil {
		sandboxInstructions = w.sandbox.GetInstructions(nil)
	}

	var instructionParts []string
	if fsInstructions != "" {
		instructionParts = append(instructionParts, fsInstructions)
	}
	if sandboxInstructions != "" {
		instructionParts = append(instructionParts, sandboxInstructions)
	}

	ctx := PathContext{
		Instructions: strings.Join(instructionParts, " "),
	}

	if w.fs != nil {
		ctx.Filesystem = &PathContextFilesystem{
			Provider: w.fs.Provider(),
			BasePath: w.fs.BasePath(),
		}
	}

	if w.sandbox != nil {
		sbCtx := &PathContextSandbox{
			Provider: w.sandbox.Provider(),
		}
		if wdp, ok := w.sandbox.(WorkingDirectoryProvider); ok {
			sbCtx.WorkingDirectory = wdp.GetWorkingDirectory()
		}
		ctx.Sandbox = sbCtx
	}

	return ctx
}

// SetLogger sets the logger for this workspace and propagates to providers.
// Called by Mastra when the logger is set.
func (w *Workspace) SetLogger(l logger.IMastraLogger) {
	// Propagate logger to filesystem provider if it supports SetLogger.
	if w.fs != nil {
		if setter, ok := w.fs.(interface{ SetLogger(logger.IMastraLogger) }); ok {
			setter.SetLogger(l)
		}
	}
	// Propagate logger to sandbox provider if it supports SetLogger.
	if w.sandbox != nil {
		if setter, ok := w.sandbox.(interface{ SetLogger(logger.IMastraLogger) }); ok {
			setter.SetLogger(l)
		}
	}
}

// min returns the smaller of two ints.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
