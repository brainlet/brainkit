package brainkit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ScaffoldOptions configures ScaffoldPackage. The zero value is
// fine for the common case — pass a custom ScaffoldOptions only
// to override the version, description, tsconfig, or to pass
// additional files.
type ScaffoldOptions struct {
	// Version is written into manifest.json. Default: "0.1.0".
	Version string
	// Description is written into manifest.json. Default: "".
	Description string
	// Extra files written alongside the entry file. Keys are
	// relative to the package dir, values are raw file contents.
	// Do NOT include manifest.json / tsconfig.json / entry file /
	// types/*.d.ts here — those are owned by Scaffold.
	Extra map[string]string
	// Overwrite allows Scaffold to replace an existing directory.
	// Default false: an existing non-empty directory errors out
	// so a caller can't silently stomp a user's work.
	Overwrite bool
}

// ScaffoldPackage writes a brainkit package on disk at dir using
// the same layout the CLI's `brainkit new package` produces:
//
//	<dir>/
//	  manifest.json
//	  <entry>              (the TypeScript source — usually "index.ts")
//	  tsconfig.json        (paths-mapped so the IDE finds types/*)
//	  types/
//	    kit.d.ts
//	    ai.d.ts
//	    agent.d.ts
//	    brainkit.d.ts
//	    globals.d.ts
//
// The dir is ready for `PackageFromDir(dir)` and for a developer
// to open in any TypeScript-aware IDE.
//
//   - name is the package identifier written into manifest.json
//     (it's the handle `kit.Deploy` / teardown look up).
//   - entry is the entry filename (e.g. "index.ts"). It's recorded
//     in manifest.json and becomes the source of the deployment.
//   - source is the TypeScript source written to <dir>/<entry>.
//
// Subsequent edits inside <dir> survive — ScaffoldPackage is a
// one-shot layout generator, not a watcher. Pass
// ScaffoldOptions.Overwrite to wipe and regenerate.
func ScaffoldPackage(dir, name, entry, source string, opts ...ScaffoldOptions) error {
	if dir == "" {
		return fmt.Errorf("brainkit.ScaffoldPackage: dir is required")
	}
	if name == "" {
		return fmt.Errorf("brainkit.ScaffoldPackage: name is required")
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

	// Check for existing content unless Overwrite is set.
	if _, err := os.Stat(dir); err == nil && !opt.Overwrite {
		entries, readErr := os.ReadDir(dir)
		if readErr == nil && len(entries) > 0 {
			return fmt.Errorf("brainkit.ScaffoldPackage: %s already exists and is not empty — pass ScaffoldOptions{Overwrite: true} to replace", dir)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "types"), 0o755); err != nil {
		return fmt.Errorf("brainkit.ScaffoldPackage: mkdir: %w", err)
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
		return fmt.Errorf("brainkit.ScaffoldPackage: marshal manifest: %w", err)
	}

	files := map[string]string{
		"manifest.json":          string(manifestJSON) + "\n",
		entry:                    source,
		"tsconfig.json":          tsconfigTemplate,
		"types/kit.d.ts":         KitDTS,
		"types/ai.d.ts":          AiDTS,
		"types/agent.d.ts":       AgentDTS,
		"types/brainkit.d.ts":    BrainkitDTS,
		"types/globals.d.ts":     GlobalsDTS,
	}
	for path, content := range opt.Extra {
		// Protect scaffold-owned files so callers can't shoot
		// themselves in the foot by smuggling a manifest / tsconfig
		// override through Extra.
		if _, owned := files[path]; owned {
			return fmt.Errorf("brainkit.ScaffoldPackage: Extra tried to overwrite scaffold-owned file %q — use ScaffoldOptions top-level fields instead", path)
		}
		files[path] = content
	}
	for path, content := range files {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return fmt.Errorf("brainkit.ScaffoldPackage: mkdir %s: %w", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			return fmt.Errorf("brainkit.ScaffoldPackage: write %s: %w", path, err)
		}
	}
	return nil
}

// tsconfigTemplate is the standard tsconfig.json every scaffolded
// package ships. paths map points at the sibling `types/` dir so
// an IDE (VS Code, WebStorm) resolves `import … from "kit"` etc.
// to the bundled .d.ts declarations without an npm install.
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
