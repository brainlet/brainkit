// Ported from: packages/core/src/storage/git-history.ts
package fsutil

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// GitCommit
// ---------------------------------------------------------------------------

// GitCommit represents a single Git commit entry parsed from `git log` output.
type GitCommit struct {
	// Hash is the full commit SHA.
	Hash string `json:"hash"`
	// Date is the commit author date.
	Date time.Time `json:"date"`
	// Author is the author name.
	Author string `json:"author"`
	// Message is the commit subject line.
	Message string `json:"message"`
}

// ---------------------------------------------------------------------------
// GitHistory
// ---------------------------------------------------------------------------

// GitHistory is a read-only utility for reading Git history of
// filesystem-stored JSON files.
//
// All operations are performed by shelling out to the `git` CLI.
// This class never writes to Git -- the user manages their own commits.
//
// Designed as a singleton shared across all domain helpers.
type GitHistory struct {
	mu sync.RWMutex

	// repoRootCache maps dir -> repo root (string) or "" if not a repo.
	repoRootCache map[string]string
	// repoNotRepo tracks directories confirmed not to be repos.
	repoNotRepo map[string]bool

	// commitCache maps "dir:filename:limit" -> ordered commits (newest first).
	commitCache map[string][]GitCommit

	// snapshotCache maps "dir:commitHash:filename" -> parsed JSON.
	snapshotCache map[string]map[string]any
}

// NewGitHistory creates a new GitHistory instance.
func NewGitHistory() *GitHistory {
	return &GitHistory{
		repoRootCache: make(map[string]string),
		repoNotRepo:   make(map[string]bool),
		commitCache:   make(map[string][]GitCommit),
		snapshotCache: make(map[string]map[string]any),
	}
}

// IsGitRepo returns true if dir is inside a Git repository.
// Result is cached after the first call per directory.
func (g *GitHistory) IsGitRepo(dir string) (bool, error) {
	g.mu.RLock()
	if _, notRepo := g.repoNotRepo[dir]; notRepo {
		g.mu.RUnlock()
		return false, nil
	}
	if root, ok := g.repoRootCache[dir]; ok && root != "" {
		g.mu.RUnlock()
		return true, nil
	}
	g.mu.RUnlock()

	g.mu.Lock()
	defer g.mu.Unlock()

	// Double-check after acquiring write lock
	if _, notRepo := g.repoNotRepo[dir]; notRepo {
		return false, nil
	}
	if root, ok := g.repoRootCache[dir]; ok && root != "" {
		return root != "", nil
	}

	output, err := g.exec(dir, []string{"rev-parse", "--show-toplevel"})
	if err != nil {
		g.repoNotRepo[dir] = true
		return false, nil
	}

	root := strings.TrimSpace(output)
	g.repoRootCache[dir] = root
	return true, nil
}

// GetFileHistory returns the list of commits that touched a specific file,
// newest first. Returns an empty slice if Git is unavailable or the file
// has no history.
func (g *GitHistory) GetFileHistory(dir, filename string, limit int) ([]GitCommit, error) {
	if limit <= 0 {
		limit = 50
	}

	cacheKey := fmt.Sprintf("%s:%s:%d", dir, filename, limit)

	g.mu.RLock()
	if cached, ok := g.commitCache[cacheKey]; ok {
		g.mu.RUnlock()
		return cached, nil
	}
	g.mu.RUnlock()

	isRepo, err := g.IsGitRepo(dir)
	if err != nil || !isRepo {
		g.mu.Lock()
		g.commitCache[cacheKey] = []GitCommit{}
		g.mu.Unlock()
		return []GitCommit{}, nil
	}

	raw, err := g.exec(dir, []string{
		"log",
		fmt.Sprintf("--max-count=%d", limit),
		"--format=%H|%aI|%aN|%s",
		"--follow",
		"--",
		filename,
	})
	if err != nil {
		g.mu.Lock()
		g.commitCache[cacheKey] = []GitCommit{}
		g.mu.Unlock()
		return []GitCommit{}, nil
	}

	var commits []GitCommit
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		pipeIdx1 := strings.Index(trimmed, "|")
		if pipeIdx1 == -1 {
			continue
		}
		pipeIdx2 := strings.Index(trimmed[pipeIdx1+1:], "|")
		if pipeIdx2 == -1 {
			continue
		}
		pipeIdx2 += pipeIdx1 + 1
		pipeIdx3 := strings.Index(trimmed[pipeIdx2+1:], "|")
		if pipeIdx3 == -1 {
			continue
		}
		pipeIdx3 += pipeIdx2 + 1

		date, _ := time.Parse(time.RFC3339, trimmed[pipeIdx1+1:pipeIdx2])

		commits = append(commits, GitCommit{
			Hash:    trimmed[:pipeIdx1],
			Date:    date,
			Author:  trimmed[pipeIdx2+1 : pipeIdx3],
			Message: trimmed[pipeIdx3+1:],
		})
	}

	g.mu.Lock()
	g.commitCache[cacheKey] = commits
	g.mu.Unlock()

	return commits, nil
}

// GetFileAtCommit reads and parses a JSON file at a specific Git commit.
// Returns the parsed entity map, or nil if the file didn't exist at that commit.
func (g *GitHistory) GetFileAtCommit(dir, commitHash, filename string) (map[string]any, error) {
	cacheKey := fmt.Sprintf("%s:%s:%s", dir, commitHash, filename)

	g.mu.RLock()
	if cached, ok := g.snapshotCache[cacheKey]; ok {
		g.mu.RUnlock()
		return cached, nil
	}
	g.mu.RUnlock()

	isRepo, err := g.IsGitRepo(dir)
	if err != nil || !isRepo {
		return nil, nil
	}

	relPath, err := g.relativeToRepo(dir, filename)
	if err != nil {
		return nil, nil
	}

	raw, err := g.exec(dir, []string{"show", fmt.Sprintf("%s:%s", commitHash, relPath)})
	if err != nil {
		return nil, nil
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, nil
	}

	g.mu.Lock()
	g.snapshotCache[cacheKey] = parsed
	g.mu.Unlock()

	return parsed, nil
}

// InvalidateCache clears all caches. Call after external operations that
// change Git state (e.g., the user commits or pulls).
func (g *GitHistory) InvalidateCache() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.repoRootCache = make(map[string]string)
	g.repoNotRepo = make(map[string]bool)
	g.commitCache = make(map[string][]GitCommit)
	g.snapshotCache = make(map[string]map[string]any)
}

// ---------------------------------------------------------------------------
// Internals
// ---------------------------------------------------------------------------

// relativeToRepo returns the relative path from the Git repo root to a file
// in the storage directory.
func (g *GitHistory) relativeToRepo(dir, filename string) (string, error) {
	g.mu.RLock()
	root := g.repoRootCache[dir]
	g.mu.RUnlock()

	if root == "" {
		return "", fmt.Errorf("not a git repository: %s", dir)
	}

	// Resolve symlinks for macOS /var -> /private/var differences
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		realRoot = root
	}
	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		realDir = dir
	}

	relDir, err := filepath.Rel(realRoot, realDir)
	if err != nil {
		return filename, nil
	}

	if relDir == "." || relDir == "" {
		return filename, nil
	}

	return relDir + "/" + filename, nil
}

// exec executes a git command and returns stdout.
func (g *GitHistory) exec(cwd string, args []string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
