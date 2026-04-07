package packages

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveEntry determines the entry point for a package directory.
// Priority: manifest "entry" field → index.ts → only .ts file in root.
func ResolveEntry(dir string, manifest PackageManifest) (string, error) {
	// 1. Explicit entry in manifest
	if manifest.Entry != "" {
		path := filepath.Join(dir, manifest.Entry)
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("manifest entry %q not found: %w", manifest.Entry, err)
		}
		return path, nil
	}

	// 2. Convention: index.ts
	indexPath := filepath.Join(dir, "index.ts")
	if _, err := os.Stat(indexPath); err == nil {
		return indexPath, nil
	}

	// 3. Fallback: the only .ts file in root
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("read package directory: %w", err)
	}
	var tsFiles []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".ts" {
			tsFiles = append(tsFiles, e.Name())
		}
	}
	if len(tsFiles) == 1 {
		return filepath.Join(dir, tsFiles[0]), nil
	}
	if len(tsFiles) == 0 {
		return "", fmt.Errorf("no .ts files found in %s", dir)
	}
	return "", fmt.Errorf("multiple .ts files in %s — add \"entry\" to manifest.json or create index.ts", dir)
}
