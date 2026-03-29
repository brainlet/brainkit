package packages

import (
	"fmt"

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
		Format:      api.FormatIIFE,
		Platform:    api.PlatformBrowser,
		External:    []string{"kit", "ai", "agent", "compiler"},
		Write:       false,
		Loader: map[string]api.Loader{
			".ts": api.LoaderTS,
		},
		// Tree-shaking for dead code elimination
		TreeShaking: api.TreeShakingTrue,
		// Target modern JS (QuickJS supports ES2020+)
		Target: api.ES2020,
	})

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
