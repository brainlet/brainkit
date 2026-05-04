package esbuild

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// Bundle reads a .ts entry point from the filesystem, resolves all relative
// imports via esbuild, strips TypeScript, and returns a single bundled JS
// string. External modules ("kit", "ai", "agent", "compiler") are provided
// by Compartment endowments and stripped during normalization.
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
		TreeShaking: api.TreeShakingTrue,
		Target:      api.ESNext,
	})

	return bundleResult(result, entryPath)
}

// BundleInMemory runs the same esbuild pipeline as Bundle but reads sources
// from an in-memory map instead of the filesystem.
func BundleInMemory(files map[string]string, entry string) (string, error) {
	if _, ok := files[entry]; !ok {
		return "", fmt.Errorf("bundle: entry %q not in files map", entry)
	}

	const ns = "pkg"
	resolve := func(args api.OnResolveArgs) (api.OnResolveResult, error) {
		if !strings.HasPrefix(args.Path, ".") && !strings.HasPrefix(args.Path, "/") {
			return api.OnResolveResult{}, nil
		}
		base := filepath.Dir(args.Importer)
		if base == "" || base == "." {
			base = ""
		}
		target := filepath.Clean(filepath.Join(base, args.Path))
		target = strings.TrimPrefix(target, "./")
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

	return NormalizeJSArtifact(string(result.OutputFiles[0].Contents)), nil
}

var esImportRe = regexp.MustCompile(`(?m)^import\s+(type\s+)?(\{[^}]*\}|\*\s+as\s+\w+|[^\s]+)\s+from\s+"[^"]+";\s*\n?`)

// NormalizeJSArtifact converts bundled package output into the runtime artifact
// shape: plain JavaScript evaluated with Brainkit endowments, not ES module
// import declarations.
func NormalizeJSArtifact(code string) string {
	return esImportRe.ReplaceAllString(code, "")
}
