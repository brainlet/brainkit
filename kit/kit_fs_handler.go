package kit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/brainlet/brainkit/internal/bus"
)

func (k *Kit) handleFs(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	switch msg.Topic {
	case "fs.read":
		return k.handleFsRead(ctx, msg)
	case "fs.write":
		return k.handleFsWrite(ctx, msg)
	case "fs.list":
		return k.handleFsList(ctx, msg)
	case "fs.stat":
		return k.handleFsStat(ctx, msg)
	case "fs.delete":
		return k.handleFsDelete(ctx, msg)
	case "fs.mkdir":
		return k.handleFsMkdir(ctx, msg)
	default:
		return nil, fmt.Errorf("fs: unknown topic %q", msg.Topic)
	}
}

func (k *Kit) handleFsRead(_ context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("fs.read: invalid request: %w", err)
	}
	if req.Path == "" {
		return nil, fmt.Errorf("fs.read: path is required")
	}

	absPath, err := k.resolveWorkspacePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("fs.read: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("fs.read: %w", err)
	}

	result, _ := json.Marshal(map[string]string{"data": string(data)})
	return &bus.Message{Payload: result}, nil
}

func (k *Kit) handleFsWrite(_ context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Path string `json:"path"`
		Data string `json:"data"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("fs.write: invalid request: %w", err)
	}
	if req.Path == "" {
		return nil, fmt.Errorf("fs.write: path is required")
	}

	absPath, err := k.resolveWorkspacePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("fs.write: %w", err)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return nil, fmt.Errorf("fs.write: %w", err)
	}

	if err := os.WriteFile(absPath, []byte(req.Data), 0o644); err != nil {
		return nil, fmt.Errorf("fs.write: %w", err)
	}

	result, _ := json.Marshal(map[string]bool{"ok": true})
	return &bus.Message{Payload: result}, nil
}

func (k *Kit) handleFsList(_ context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Path    string `json:"path"`
		Pattern string `json:"pattern,omitempty"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("fs.list: invalid request: %w", err)
	}
	if req.Path == "" {
		req.Path = "."
	}

	absPath, err := k.resolveWorkspacePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("fs.list: %w", err)
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("fs.list: %w", err)
	}

	type fileInfo struct {
		Name  string `json:"name"`
		Size  int64  `json:"size"`
		IsDir bool   `json:"isDir"`
	}

	var files []fileInfo
	for _, entry := range entries {
		if req.Pattern != "" {
			matched, _ := filepath.Match(req.Pattern, entry.Name())
			if !matched {
				continue
			}
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{
			Name:  entry.Name(),
			Size:  info.Size(),
			IsDir: entry.IsDir(),
		})
	}

	if files == nil {
		files = []fileInfo{}
	}

	result, _ := json.Marshal(map[string]any{"files": files})
	return &bus.Message{Payload: result}, nil
}

func (k *Kit) handleFsStat(_ context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("fs.stat: invalid request: %w", err)
	}
	if req.Path == "" {
		return nil, fmt.Errorf("fs.stat: path is required")
	}

	absPath, err := k.resolveWorkspacePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("fs.stat: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("fs.stat: %w", err)
	}

	result, _ := json.Marshal(map[string]any{
		"size":    info.Size(),
		"isDir":   info.IsDir(),
		"modTime": info.ModTime().Format("2006-01-02T15:04:05Z07:00"),
	})
	return &bus.Message{Payload: result}, nil
}

func (k *Kit) handleFsDelete(_ context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("fs.delete: invalid request: %w", err)
	}
	if req.Path == "" {
		return nil, fmt.Errorf("fs.delete: path is required")
	}

	absPath, err := k.resolveWorkspacePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("fs.delete: %w", err)
	}

	if err := os.Remove(absPath); err != nil {
		return nil, fmt.Errorf("fs.delete: %w", err)
	}

	result, _ := json.Marshal(map[string]bool{"ok": true})
	return &bus.Message{Payload: result}, nil
}

func (k *Kit) handleFsMkdir(_ context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("fs.mkdir: invalid request: %w", err)
	}
	if req.Path == "" {
		return nil, fmt.Errorf("fs.mkdir: path is required")
	}

	absPath, err := k.resolveWorkspacePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("fs.mkdir: %w", err)
	}

	if err := os.MkdirAll(absPath, 0o755); err != nil {
		return nil, fmt.Errorf("fs.mkdir: %w", err)
	}

	result, _ := json.Marshal(map[string]bool{"ok": true})
	return &bus.Message{Payload: result}, nil
}

// resolveWorkspacePath resolves a path relative to the Kit's workspace.
// Returns error if the path escapes the workspace.
func (k *Kit) resolveWorkspacePath(path string) (string, error) {
	workspace := k.config.WorkspaceDir
	if workspace == "" {
		return "", fmt.Errorf("workspace not configured")
	}

	abs := filepath.Join(workspace, filepath.Clean("/"+path))
	cleanWorkspace := filepath.Clean(workspace)
	if abs != cleanWorkspace && !strings.HasPrefix(abs, cleanWorkspace+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes workspace", path)
	}
	return abs, nil
}
