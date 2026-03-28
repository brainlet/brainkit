package kit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
)

// FSDomain handles sandboxed filesystem operations.
type FSDomain struct {
	kit *Kernel
}

func newFSDomain(k *Kernel) *FSDomain {
	return &FSDomain{kit: k}
}

func (d *FSDomain) Read(_ context.Context, req messages.FsReadMsg) (*messages.FsReadResp, error) {
	absPath, err := d.resolveWorkspacePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("fs.read: %w", err)
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("fs.read: %w", err)
	}
	return &messages.FsReadResp{Data: string(data)}, nil
}

func (d *FSDomain) Write(_ context.Context, req messages.FsWriteMsg) (*messages.FsWriteResp, error) {
	absPath, err := d.resolveWorkspacePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("fs.write: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return nil, fmt.Errorf("fs.write: %w", err)
	}
	if err := os.WriteFile(absPath, []byte(req.Data), 0o644); err != nil {
		return nil, fmt.Errorf("fs.write: %w", err)
	}
	return &messages.FsWriteResp{OK: true}, nil
}

func (d *FSDomain) List(_ context.Context, req messages.FsListMsg) (*messages.FsListResp, error) {
	path := req.Path
	if path == "" {
		path = "."
	}
	absPath, err := d.resolveWorkspacePath(path)
	if err != nil {
		return nil, fmt.Errorf("fs.list: %w", err)
	}
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("fs.list: %w", err)
	}

	var files []messages.FsFileInfo
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
		files = append(files, messages.FsFileInfo{Name: entry.Name(), Size: info.Size(), IsDir: entry.IsDir()})
	}
	if files == nil {
		files = []messages.FsFileInfo{}
	}
	return &messages.FsListResp{Files: files}, nil
}

func (d *FSDomain) Stat(_ context.Context, req messages.FsStatMsg) (*messages.FsStatResp, error) {
	absPath, err := d.resolveWorkspacePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("fs.stat: %w", err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("fs.stat: %w", err)
	}
	return &messages.FsStatResp{
		Size:    info.Size(),
		IsDir:   info.IsDir(),
		ModTime: info.ModTime().Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (d *FSDomain) Delete(_ context.Context, req messages.FsDeleteMsg) (*messages.FsDeleteResp, error) {
	absPath, err := d.resolveWorkspacePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("fs.delete: %w", err)
	}
	if err := os.Remove(absPath); err != nil {
		return nil, fmt.Errorf("fs.delete: %w", err)
	}
	return &messages.FsDeleteResp{OK: true}, nil
}

func (d *FSDomain) Mkdir(_ context.Context, req messages.FsMkdirMsg) (*messages.FsMkdirResp, error) {
	absPath, err := d.resolveWorkspacePath(req.Path)
	if err != nil {
		return nil, fmt.Errorf("fs.mkdir: %w", err)
	}
	if err := os.MkdirAll(absPath, 0o755); err != nil {
		return nil, fmt.Errorf("fs.mkdir: %w", err)
	}
	return &messages.FsMkdirResp{OK: true}, nil
}

func (d *FSDomain) resolveWorkspacePath(path string) (string, error) {
	workspace := d.kit.config.FSRoot
	if workspace == "" {
		return "", ErrNoWorkspace
	}
	abs := filepath.Join(workspace, filepath.Clean("/"+path))
	cleanWorkspace := filepath.Clean(workspace)
	if abs != cleanWorkspace && !strings.HasPrefix(abs, cleanWorkspace+string(filepath.Separator)) {
		return "", &sdk.WorkspaceEscapeError{Path: path}
	}
	return abs, nil
}
