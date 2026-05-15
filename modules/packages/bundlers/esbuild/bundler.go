package esbuild

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/evanw/esbuild/pkg/api"
)

const (
	resolverProfileSourceRelative = "source-relative"
	resolverProfileNPMPreview     = "npm-preview"
	suggestedNPMResolverOwner     = "future opt-in npm ecosystem resolver profile"
)

var allowedBareImports = []string{"kit", "ai", "agent", "compiler"}

type bundleOptions struct {
	sourcePackage   string
	resolverProfile string
	packageRoot     string
}

// Bundle reads a .ts entry point from the filesystem, resolves all relative
// imports via esbuild, strips TypeScript, and returns a single bundled JS
// string. External modules ("kit", "ai", "agent", "compiler") are provided
// by Compartment endowments and stripped during normalization.
func Bundle(entryPath string) (string, error) {
	return bundle(entryPath, bundleOptions{})
}

func bundle(entryPath string, opts bundleOptions) (string, error) {
	opts = opts.withDefaults()
	result := api.Build(api.BuildOptions{
		EntryPoints:   []string{entryPath},
		Bundle:        true,
		Format:        api.FormatESModule,
		Platform:      api.PlatformBrowser,
		AbsWorkingDir: opts.packageRoot,
		MainFields:    []string{"browser", "module", "main"},
		Write:         false,
		Loader: map[string]api.Loader{
			".ts": api.LoaderTS,
		},
		TreeShaking: api.TreeShakingTrue,
		Target:      api.ESNext,
		Supported: map[string]bool{
			"dynamic-import": false,
		},
		Plugins: resolverPolicyPlugins(opts),
	})

	return bundleResult(result, entryPath)
}

// BundleInMemory runs the same esbuild pipeline as Bundle but reads sources
// from an in-memory map instead of the filesystem.
func BundleInMemory(files map[string]string, entry string) (string, error) {
	return bundleInMemory(files, entry, bundleOptions{})
}

func bundleInMemory(files map[string]string, entry string, opts bundleOptions) (string, error) {
	opts = opts.withDefaults()
	if _, ok := files[entry]; !ok {
		return "", fmt.Errorf("bundle: entry %q not in files map", entry)
	}

	const ns = "pkg"
	resolve := func(args api.OnResolveArgs) (api.OnResolveResult, error) {
		if isBareImport(args.Path) {
			if isAllowedBareImport(args.Path) {
				id, _ := brainkitRuntimeModuleStub(args.Path)
				return api.OnResolveResult{Path: id, Namespace: brainkitRuntimeModuleNamespace}, nil
			}
			if opts.resolverProfile == resolverProfileNPMPreview {
				return unsupportedNPMPreviewInMemoryResult(args.Path, args.Importer, opts), nil
			}
			return unsupportedBareImportResult(args.Path, args.Importer, opts), nil
		}
		base := filepath.Dir(args.Importer)
		if base == "" || base == "." {
			base = ""
		}
		target := filepath.Clean(filepath.Join(base, args.Path))
		target = strings.TrimPrefix(target, "./")
		for _, candidate := range inMemoryCandidates(target) {
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
		} else if strings.HasSuffix(args.Path, ".json") {
			loader = api.LoaderJSON
		}
		return api.OnLoadResult{
			Contents: &code,
			Loader:   loader,
		}, nil
	}

	plugin := api.Plugin{
		Name: "brainkit-inmemory",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: "^" + regexp.QuoteMeta(entry) + "$"},
				func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					return api.OnResolveResult{Path: entry, Namespace: ns}, nil
				})
			build.OnResolve(api.OnResolveOptions{Filter: ".*", Namespace: ns}, resolve)
			build.OnLoad(api.OnLoadOptions{Filter: ".*", Namespace: ns}, load)
			build.OnLoad(api.OnLoadOptions{Filter: ".*", Namespace: brainkitRuntimeModuleNamespace},
				func(args api.OnLoadArgs) (api.OnLoadResult, error) {
					contents, ok := brainkitRuntimeModuleStubContents(args.Path)
					if !ok {
						return api.OnLoadResult{}, fmt.Errorf("Brainkit runtime module stub %q is not registered", args.Path)
					}
					return api.OnLoadResult{Contents: &contents, Loader: api.LoaderJS}, nil
				})
		},
	}

	result := api.Build(api.BuildOptions{
		EntryPoints: []string{entry},
		Bundle:      true,
		Format:      api.FormatESModule,
		Platform:    api.PlatformBrowser,
		Write:       false,
		Loader:      map[string]api.Loader{".ts": api.LoaderTS},
		TreeShaking: api.TreeShakingTrue,
		Target:      api.ESNext,
		Supported: map[string]bool{
			"dynamic-import": false,
		},
		Plugins: []api.Plugin{plugin},
	})

	return bundleResult(result, entry)
}

func bundleResult(result api.BuildResult, entryPath string) (string, error) {
	if len(result.Errors) > 0 {
		msg := result.Errors[0]
		if resolverErr, ok := msg.Detail.(*sdkerrors.PackageResolverError); ok {
			return "", resolverErr
		}
		loc := ""
		if msg.Location != nil {
			loc = fmt.Sprintf(" at %s:%d:%d", msg.Location.File, msg.Location.Line, msg.Location.Column)
		}
		return "", fmt.Errorf("bundle %s: %s%s", entryPath, msg.Text, loc)
	}

	if len(result.OutputFiles) == 0 {
		return "", fmt.Errorf("bundle %s: no output produced", entryPath)
	}

	code := NormalizeJSArtifact(string(result.OutputFiles[0].Contents))
	return sanitizeSESRejectedImportText(code), nil
}

func (opts bundleOptions) withDefaults() bundleOptions {
	if opts.resolverProfile == "" {
		opts.resolverProfile = resolverProfileSourceRelative
	}
	return opts
}

func resolverPolicyPlugins(opts bundleOptions) []api.Plugin {
	return []api.Plugin{resolverPolicyPlugin(opts)}
}

func resolverPolicyPlugin(opts bundleOptions) api.Plugin {
	opts = opts.withDefaults()
	name := "brainkit-source-relative-resolver"
	if opts.resolverProfile == resolverProfileNPMPreview {
		name = "brainkit-npm-preview-resolver"
	}
	return api.Plugin{
		Name: name,
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: ".*"},
				func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					if args.Kind == api.ResolveEntryPoint || !isBareImport(args.Path) {
						return api.OnResolveResult{}, nil
					}
					if isAllowedBareImport(args.Path) {
						id, _ := brainkitRuntimeModuleStub(args.Path)
						return api.OnResolveResult{Path: id, Namespace: brainkitRuntimeModuleNamespace}, nil
					}
					if opts.resolverProfile == resolverProfileNPMPreview {
						if args.Path == "zod" {
							resolved := build.Resolve("zod/v4", api.ResolveOptions{
								Importer:   args.Importer,
								Namespace:  args.Namespace,
								ResolveDir: args.ResolveDir,
								Kind:       args.Kind,
								PluginData: args.PluginData,
								With:       args.With,
							})
							return api.OnResolveResult{
								Errors:     resolved.Errors,
								Warnings:   resolved.Warnings,
								Path:       resolved.Path,
								External:   resolved.External,
								Namespace:  resolved.Namespace,
								Suffix:     resolved.Suffix,
								PluginData: resolved.PluginData,
							}, nil
						}
						if id, _, ok := npmPreviewPackageStub(args.Path); ok {
							return api.OnResolveResult{Path: id, Namespace: npmPreviewPackageStubNamespace}, nil
						}
						if isNodeBuiltinImport(args.Path) {
							if id, _, ok := npmPreviewNodeStub(args.Path); ok {
								return api.OnResolveResult{Path: id, Namespace: npmPreviewNodeStubNamespace}, nil
							}
							return unsupportedNodeBuiltinResult(args.Path, args.Importer, opts), nil
						}
						return api.OnResolveResult{}, nil
					}
					return unsupportedBareImportResult(args.Path, args.Importer, opts), nil
				})
			build.OnLoad(api.OnLoadOptions{Filter: ".*", Namespace: npmPreviewNodeStubNamespace},
				func(args api.OnLoadArgs) (api.OnLoadResult, error) {
					contents, ok := npmPreviewNodeStubs[args.Path]
					if !ok {
						return api.OnLoadResult{}, fmt.Errorf("npm-preview Node stub %q is not registered", args.Path)
					}
					return api.OnLoadResult{Contents: &contents, Loader: api.LoaderJS}, nil
				})
			build.OnLoad(api.OnLoadOptions{Filter: ".*", Namespace: npmPreviewPackageStubNamespace},
				func(args api.OnLoadArgs) (api.OnLoadResult, error) {
					contents, ok := npmPreviewPackageStubs[args.Path]
					if !ok {
						return api.OnLoadResult{}, fmt.Errorf("npm-preview package stub %q is not registered", args.Path)
					}
					return api.OnLoadResult{Contents: &contents, Loader: api.LoaderJS}, nil
				})
			build.OnLoad(api.OnLoadOptions{Filter: ".*", Namespace: brainkitRuntimeModuleNamespace},
				func(args api.OnLoadArgs) (api.OnLoadResult, error) {
					contents, ok := brainkitRuntimeModuleStubContents(args.Path)
					if !ok {
						return api.OnLoadResult{}, fmt.Errorf("Brainkit runtime module stub %q is not registered", args.Path)
					}
					return api.OnLoadResult{Contents: &contents, Loader: api.LoaderJS}, nil
				})
		},
	}
}

func unsupportedBareImportResult(specifier, importer string, opts bundleOptions) api.OnResolveResult {
	err := &sdkerrors.PackageResolverError{
		Specifier:          specifier,
		Importer:           importer,
		Source:             opts.sourcePackage,
		Profile:            opts.withDefaults().resolverProfile,
		AllowedBareImports: append([]string(nil), allowedBareImports...),
		SuggestedOwner:     suggestedNPMResolverOwner,
	}
	return api.OnResolveResult{
		Errors: []api.Message{{
			Text:   err.Error(),
			Detail: err,
		}},
	}
}

func unsupportedNodeBuiltinResult(specifier, importer string, opts bundleOptions) api.OnResolveResult {
	err := &sdkerrors.PackageResolverError{
		Specifier:          specifier,
		Importer:           importer,
		Source:             opts.sourcePackage,
		Profile:            opts.withDefaults().resolverProfile,
		AllowedBareImports: append([]string(nil), allowedBareImports...),
		SuggestedOwner:     "jsbridge, browser module, or package-specific adapter",
		Reason:             "npm-preview does not currently expose this Node builtin/subpath for package deployments",
		BoundaryClass:      "unsupported-node-api",
	}
	return api.OnResolveResult{
		Errors: []api.Message{{
			Text:   err.Error(),
			Detail: err,
		}},
	}
}

func unsupportedNPMPreviewInMemoryResult(specifier, importer string, opts bundleOptions) api.OnResolveResult {
	err := &sdkerrors.PackageResolverError{
		Specifier:          specifier,
		Importer:           importer,
		Source:             opts.sourcePackage,
		Profile:            opts.withDefaults().resolverProfile,
		AllowedBareImports: append([]string(nil), allowedBareImports...),
		SuggestedOwner:     "filesystem package deploy with npm-preview resolver",
		Reason:             "npm-preview requires a filesystem package root with package.json and pnpm-lock.yaml",
		BoundaryClass:      "resolver-profile",
	}
	return api.OnResolveResult{
		Errors: []api.Message{{
			Text:   err.Error(),
			Detail: err,
		}},
	}
}

func prepareNPMPreviewPackage(ctx context.Context, dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err != nil {
		if os.IsNotExist(err) {
			return &sdkerrors.ValidationError{
				Field:   "manifest.resolver",
				Message: "npm-preview requires package.json in the filesystem package root",
			}
		}
		return fmt.Errorf("npm-preview resolver stat package.json: %w", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "pnpm-lock.yaml")); err != nil {
		if os.IsNotExist(err) {
			return &sdkerrors.ValidationError{
				Field:   "manifest.resolver",
				Message: "npm-preview requires pnpm-lock.yaml and runs pnpm install --frozen-lockfile --ignore-scripts",
			}
		}
		return fmt.Errorf("npm-preview resolver stat pnpm-lock.yaml: %w", err)
	}
	cmd := exec.CommandContext(ctx, "pnpm", "install", "--frozen-lockfile", "--ignore-scripts")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CI=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm-preview resolver install failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func isBareImport(specifier string) bool {
	return specifier != "" &&
		!strings.HasPrefix(specifier, ".") &&
		!strings.HasPrefix(specifier, "/") &&
		!filepath.IsAbs(specifier)
}

func isAllowedBareImport(specifier string) bool {
	for _, allowed := range allowedBareImports {
		if specifier == allowed {
			return true
		}
	}
	return false
}

func inMemoryCandidates(target string) []string {
	return []string{
		target,
		target + ".ts",
		target + ".js",
		target + ".mjs",
		target + ".json",
		target + "/index.ts",
		target + "/index.js",
		target + "/index.mjs",
		target + "/index.json",
	}
}

var esImportRe = regexp.MustCompile(`(?m)^import\s+(type\s+)?(\{[^}]*\}|\*\s+as\s+\w+|[^\s]+)\s+from\s+"[^"]+";\s*\n?`)

// NormalizeJSArtifact converts bundled package output into the runtime artifact
// shape: plain JavaScript evaluated with Brainkit endowments, not ES module
// import declarations.
func NormalizeJSArtifact(code string) string {
	return esImportRe.ReplaceAllString(code, "")
}

func sanitizeSESRejectedImportText(code string) string {
	var out strings.Builder
	out.Grow(len(code))

	const (
		stateNormal = iota
		stateSingle
		stateDouble
		stateTemplate
		stateLineComment
		stateBlockComment
	)

	state := stateNormal
	escaped := false
	for i := 0; i < len(code); {
		switch state {
		case stateNormal:
			if code[i] == '\'' {
				state = stateSingle
			} else if code[i] == '"' {
				state = stateDouble
			} else if code[i] == '`' {
				state = stateTemplate
			} else if code[i] == '/' && i+1 < len(code) && code[i+1] == '/' {
				state = stateLineComment
				out.WriteByte(code[i])
				out.WriteByte(code[i+1])
				i += 2
				continue
			} else if code[i] == '/' && i+1 < len(code) && code[i+1] == '*' {
				state = stateBlockComment
				out.WriteByte(code[i])
				out.WriteByte(code[i+1])
				i += 2
				continue
			}
			out.WriteByte(code[i])
			i++

		case stateSingle:
			if strings.HasPrefix(code[i:], "import(") {
				out.WriteString("import\\u0028")
				i += len("import(")
				continue
			}
			out.WriteByte(code[i])
			if escaped {
				escaped = false
			} else if code[i] == '\\' {
				escaped = true
			} else if code[i] == '\'' {
				state = stateNormal
			}
			i++

		case stateDouble:
			if strings.HasPrefix(code[i:], "import(") {
				out.WriteString("import\\u0028")
				i += len("import(")
				continue
			}
			out.WriteByte(code[i])
			if escaped {
				escaped = false
			} else if code[i] == '\\' {
				escaped = true
			} else if code[i] == '"' {
				state = stateNormal
			}
			i++

		case stateTemplate:
			if strings.HasPrefix(code[i:], "import(") {
				out.WriteString("import\\u0028")
				i += len("import(")
				continue
			}
			out.WriteByte(code[i])
			if escaped {
				escaped = false
			} else if code[i] == '\\' {
				escaped = true
			} else if code[i] == '`' {
				state = stateNormal
			}
			i++

		case stateLineComment:
			if strings.HasPrefix(code[i:], "import(") {
				out.WriteString("import\\u0028")
				i += len("import(")
				continue
			}
			out.WriteByte(code[i])
			if code[i] == '\n' {
				state = stateNormal
			}
			i++

		case stateBlockComment:
			if strings.HasPrefix(code[i:], "import(") {
				out.WriteString("import\\u0028")
				i += len("import(")
				continue
			}
			if code[i] == '*' && i+1 < len(code) && code[i+1] == '/' {
				out.WriteByte(code[i])
				out.WriteByte(code[i+1])
				i += 2
				state = stateNormal
				continue
			}
			out.WriteByte(code[i])
			i++
		}
	}
	return out.String()
}
