package deploy

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// Bundle reads a .ts entry point from the filesystem, resolves all relative
// imports via esbuild, strips TypeScript, and returns a single bundled JS string.
// External modules ("kit", "ai", "agent", "compiler") are left as global references
// (provided by Compartment endowments).
//
// Format is IIFE for scope isolation — each file gets its own scope.
// Name collisions between files are handled by esbuild's renaming.
func Bundle(entryPath string) (string, error) {
	result := api.Build(api.BuildOptions{
		EntryPoints: []string{entryPath},
		Bundle:      true,
		Format:      api.FormatESModule,
		Platform:    api.PlatformBrowser,
		External:    []string{"kit", "ai", "agent", "compiler"},
		Write:       false,
		Loader: map[string]api.Loader{
			".ts": api.LoaderTS,
		},
		// Tree-shaking for dead code elimination
		TreeShaking: api.TreeShakingTrue,
		// Target ESNext — QuickJS supports ES2020+ and Deploy wraps in async IIFE,
		// so top-level await is safe. ES2020 rejects top-level await at bundle time.
		Target: api.ESNext,
	})

	return bundleResult(result, entryPath)
}

// BundleInMemory runs the same esbuild pipeline as Bundle but reads
// sources from an in-memory map instead of the filesystem. `entry`
// is the key in `files` that holds the entry module's source; every
// relative import it makes is resolved against the other keys.
//
// Used on the server side when `brainkit deploy` streams a multi-file
// package through the bus as a `Files map[string]string` — no temp
// directory materialization, no disk I/O, no cleanup dance.
func BundleInMemory(files map[string]string, entry string) (string, error) {
	if _, ok := files[entry]; !ok {
		return "", fmt.Errorf("bundle: entry %q not in files map", entry)
	}

	// esbuild resolves every import against the namespace of the
	// containing file. We put every in-memory file under the "pkg"
	// namespace so resolution stays inside our map and never falls
	// back to the filesystem.
	const ns = "pkg"
	resolve := func(args api.OnResolveArgs) (api.OnResolveResult, error) {
		// Bare specifiers — "kit", "agent", etc. — match the
		// external list and esbuild leaves them alone. Relative
		// imports route here.
		if !strings.HasPrefix(args.Path, ".") && !strings.HasPrefix(args.Path, "/") {
			return api.OnResolveResult{}, nil
		}
		base := filepath.Dir(args.Importer)
		if base == "" || base == "." {
			base = ""
		}
		target := filepath.Clean(filepath.Join(base, args.Path))
		target = strings.TrimPrefix(target, "./")
		// Try exact + with .ts appended (TypeScript convention).
		for _, candidate := range []string{target, target + ".ts", target + "/index.ts"} {
			if _, ok := files[candidate]; ok {
				return api.OnResolveResult{Path: candidate, Namespace: ns}, nil
			}
		}
		return api.OnResolveResult{}, fmt.Errorf("cannot resolve %q from %q", args.Path, args.Importer)
	}

	load := func(args api.OnLoadArgs) (api.OnLoadResult, error) {
		code, ok := files[args.Path]
		if !ok {
			return api.OnLoadResult{}, fmt.Errorf("in-memory file %q not found", args.Path)
		}
		loader := api.LoaderTS
		if strings.HasSuffix(args.Path, ".js") || strings.HasSuffix(args.Path, ".mjs") {
			loader = api.LoaderJS
		}
		return api.OnLoadResult{
			Contents: &code,
			Loader:   loader,
		}, nil
	}

	plugin := api.Plugin{
		Name: "brainkit-inmemory",
		Setup: func(build api.PluginBuild) {
			// Rewrite the entry point itself into our namespace.
			build.OnResolve(api.OnResolveOptions{Filter: "^" + entry + "$"},
				func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					return api.OnResolveResult{Path: entry, Namespace: ns}, nil
				})
			build.OnResolve(api.OnResolveOptions{Filter: ".*", Namespace: ns}, resolve)
			build.OnLoad(api.OnLoadOptions{Filter: ".*", Namespace: ns}, load)
		},
	}

	result := api.Build(api.BuildOptions{
		EntryPoints: []string{entry},
		Bundle:      true,
		Format:      api.FormatESModule,
		Platform:    api.PlatformBrowser,
		External:    []string{"kit", "ai", "agent", "compiler"},
		Write:       false,
		Loader:      map[string]api.Loader{".ts": api.LoaderTS},
		TreeShaking: api.TreeShakingTrue,
		Target:      api.ESNext,
		Plugins:     []api.Plugin{plugin},
	})

	return bundleResult(result, entry)
}

func bundleResult(result api.BuildResult, entryPath string) (string, error) {
	if len(result.Errors) > 0 {
		msg := result.Errors[0]
		loc := ""
		if msg.Location != nil {
			loc = fmt.Sprintf(" at %s:%d:%d", msg.Location.File, msg.Location.Line, msg.Location.Column)
		}
		return "", fmt.Errorf("bundle %s: %s%s", entryPath, msg.Text, loc)
	}

	if len(result.OutputFiles) == 0 {
		return "", fmt.Errorf("bundle %s: no output produced", entryPath)
	}

	return string(result.OutputFiles[0].Contents), nil
}
