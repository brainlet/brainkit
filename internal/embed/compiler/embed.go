package asembed

//go:generate go run ./cmd/compile-bundle

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
	"sync"

	"github.com/brainlet/brainkit/internal/jsbridge"
)

//go:embed as_compiler_bundle.js
var bundleSource string

//go:embed as_compiler_bundle.bc
var bundleBytecode []byte

//go:embed binaryen_shim.js
var shimSource string

//go:embed std
var stdFS embed.FS

var (
	cachedStdSources map[string]string
	stdSourcesOnce   sync.Once
)

// stdSources returns a map from ~lib/ internal paths (without .ts extension)
// to source text for all AS standard library files. Cached after first call.
func stdSources() map[string]string {
	stdSourcesOnce.Do(func() {
		m := make(map[string]string)
		fs.WalkDir(stdFS, "std", func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			if !strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".d.ts") {
				return nil
			}
			data, err := stdFS.ReadFile(path)
			if err != nil {
				return err
			}
			rel := strings.TrimPrefix(path, "std/")
			key := "~lib/" + strings.TrimSuffix(rel, ".ts")
			m[key] = string(data)
			return nil
		})
		cachedStdSources = m
	})
	return cachedStdSources
}

// LoadShim evaluates the binaryen shim in the given bridge context.
// The shim sets up self/global aliases, a Proxy-based binaryen module,
// and a require() function — all of which must be in place before the
// AS compiler bundle is loaded.
//
// Call order: RegisterMemoryBridge → RegisterBinaryenBridge → LoadShim → LoadBundle
func LoadShim(b *jsbridge.Bridge) error {
	val, err := b.Eval("binaryen-shim.js", shimSource)
	if err != nil {
		return fmt.Errorf("as-embed: load shim: %w", err)
	}
	val.Free()
	return nil
}

// LoadBundle evaluates the AS compiler bundle in the given bridge context.
// After loading, globalThis.__as_compiler is available with all compiler API functions.
//
// If precompiled bytecode is available, it is loaded directly (skipping the
// shim check, since bytecode already captured the module-level execution).
// Otherwise, the shim/prelude fallback is used before evaluating JS source.
//
// Before calling LoadBundle, you must call LoadShim (or the legacy inline prelude below
// will be used as a fallback when loading from source).
func LoadBundle(b *jsbridge.Bridge) error {
	// Try bytecode first — it's faster and doesn't need the shim detection.
	if len(bundleBytecode) > 0 {
		// Even with bytecode, ensure the shim/prelude is set up so that
		// require("binaryen") and global aliases are available.
		if err := ensurePrelude(b); err != nil {
			return err
		}
		val, err := b.EvalBytecode(bundleBytecode)
		if err != nil {
			return fmt.Errorf("as-embed: load bytecode: %w", err)
		}
		val.Free()
		return nil
	}

	// Fallback: evaluate from JS source.
	if err := ensurePrelude(b); err != nil {
		return err
	}

	val, err := b.Eval("as-compiler-bundle.js", bundleSource)
	if err != nil {
		return fmt.Errorf("as-embed: load bundle: %w", err)
	}
	val.Free()
	return nil
}

// ensurePrelude makes sure that require() and global aliases are set up.
// If LoadShim was already called, this is a no-op. Otherwise it installs a
// minimal inline prelude for backward compatibility.
func ensurePrelude(b *jsbridge.Bridge) error {
	check, err := b.Eval("shim-check.js", `typeof globalThis.require !== "undefined"`)
	if err != nil {
		return fmt.Errorf("as-embed: shim check: %w", err)
	}
	shimLoaded := check.String() == "true"
	check.Free()

	if shimLoaded {
		return nil
	}

	stub, err := b.Eval("as-compiler-prelude.js", `
		if (typeof self === "undefined") globalThis.self = globalThis;
		if (typeof global === "undefined") globalThis.global = globalThis;
		globalThis.require = function(m) {
			if (m === "binaryen") {
				return new Proxy({}, {
					get(target, prop) {
						if (prop in target) return target[prop];
						return function() { return 0; };
					}
				});
			}
			throw new Error("require: " + m);
		};
	`)
	if err != nil {
		return fmt.Errorf("as-embed: load require stub: %w", err)
	}
	stub.Free()
	return nil
}
