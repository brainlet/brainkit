// Package packagescaffold writes deployable TypeScript package layouts.
package packagescaffold

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/brainlet/brainkit/internal/dts"
)

// ScaffoldOptions configures ScaffoldPackage.
type ScaffoldOptions struct {
	// Version is written into manifest.json. Default: "0.1.0".
	Version string
	// Description is written into manifest.json. Default: "".
	Description string
	// Extra files written alongside the entry file.
	Extra map[string]string
	// Overwrite allows ScaffoldPackage to replace an existing directory.
	Overwrite bool
}

// ScaffoldPackage writes a deployable package layout to dir.
func ScaffoldPackage(dir, name, entry, source string, opts ...ScaffoldOptions) error {
	if dir == "" {
		return fmt.Errorf("packagescaffold.ScaffoldPackage: dir is required")
	}
	if name == "" {
		return fmt.Errorf("packagescaffold.ScaffoldPackage: name is required")
	}
	if entry == "" {
		entry = "index.ts"
	}
	var opt ScaffoldOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.Version == "" {
		opt.Version = "0.1.0"
	}

	if _, err := os.Stat(dir); err == nil && !opt.Overwrite {
		entries, readErr := os.ReadDir(dir)
		if readErr == nil && len(entries) > 0 {
			return fmt.Errorf("packagescaffold.ScaffoldPackage: %s already exists and is not empty; pass ScaffoldOptions{Overwrite: true} to replace", dir)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "types"), 0o755); err != nil {
		return fmt.Errorf("packagescaffold.ScaffoldPackage: mkdir: %w", err)
	}

	manifest := map[string]string{
		"name":    name,
		"version": opt.Version,
		"entry":   entry,
	}
	if opt.Description != "" {
		manifest["description"] = opt.Description
	}
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("packagescaffold.ScaffoldPackage: marshal manifest: %w", err)
	}

	files := map[string]string{
		"manifest.json":       string(manifestJSON) + "\n",
		entry:                 source,
		"tsconfig.json":       tsconfigTemplate,
		"types/kit.d.ts":      dts.Kit,
		"types/ai.d.ts":       dts.AI,
		"types/agent.d.ts":    dts.Agent,
		"types/brainkit.d.ts": dts.Brainkit,
		"types/globals.d.ts":  dts.Globals,
	}
	for path, content := range opt.Extra {
		if _, owned := files[path]; owned {
			return fmt.Errorf("packagescaffold.ScaffoldPackage: Extra tried to overwrite scaffold-owned file %q; use ScaffoldOptions top-level fields instead", path)
		}
		files[path] = content
	}
	for path, content := range files {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return fmt.Errorf("packagescaffold.ScaffoldPackage: mkdir %s: %w", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			return fmt.Errorf("packagescaffold.ScaffoldPackage: write %s: %w", path, err)
		}
	}
	return nil
}

const tsconfigTemplate = `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ES2022",
    "moduleResolution": "bundler",
    "strict": false,
    "noImplicitAny": false,
    "noEmit": true,
    "skipLibCheck": true,
    "esModuleInterop": true,
    "paths": {
      "kit": ["./types/kit.d.ts"],
      "ai": ["./types/ai.d.ts"],
      "agent": ["./types/agent.d.ts"]
    }
  },
  "include": ["*.ts", "./types/globals.d.ts"]
}
`
