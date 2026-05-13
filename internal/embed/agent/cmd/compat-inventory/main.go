package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
)

type compatManifest struct {
	SchemaVersion int           `json:"schemaVersion"`
	Entries       []compatEntry `json:"entries"`
}

type compatEntry struct {
	ID           string   `json:"id"`
	Kind         string   `json:"kind"`
	Status       string   `json:"status"`
	Owner        string   `json:"owner"`
	Aliases      []string `json:"aliases,omitempty"`
	Globals      []string `json:"globals,omitempty"`
	Exports      []string `json:"exports,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	Resources    []string `json:"resources,omitempty"`
	Tests        []string `json:"tests,omitempty"`
	Reason       string   `json:"reason,omitempty"`

	Package          string `json:"package,omitempty"`
	VersionRange     string `json:"versionRange,omitempty"`
	PatchType        string `json:"patchType,omitempty"`
	Required         *bool  `json:"required,omitempty"`
	Expected         string `json:"expected,omitempty"`
	RemovalCondition string `json:"removalCondition,omitempty"`
}

type esbuildMeta struct {
	Inputs  map[string]metaInput `json:"inputs"`
	Outputs map[string]struct {
		Imports []metaImport `json:"imports"`
	} `json:"outputs"`
}

type metaInput struct {
	Bytes   int64        `json:"bytes"`
	Imports []metaImport `json:"imports"`
}

type metaImport struct {
	Path     string `json:"path"`
	Kind     string `json:"kind"`
	External bool   `json:"external"`
}

type bundlePackageJSON struct {
	PackageManager  string            `json:"packageManager"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

type compatInventory struct {
	SchemaVersion int                `json:"schemaVersion"`
	Sources       inventorySources   `json:"sources"`
	PackageCount  int                `json:"packageCount"`
	SurfaceCount  int                `json:"surfaceCount"`
	Packages      []inventoryPackage `json:"packages"`
	Surfaces      []inventorySurface `json:"surfaces"`
}

type inventorySources struct {
	Manifest       string `json:"manifest"`
	Meta           string `json:"meta"`
	PackageJSON    string `json:"packageJSON"`
	PackageManager string `json:"packageManager"`
}

type inventoryPackage struct {
	Name               string   `json:"name"`
	Version            string   `json:"version,omitempty"`
	DependencyClass    string   `json:"dependencyClass"`
	ExternalService    bool     `json:"externalService,omitempty"`
	Inputs             int      `json:"inputs,omitempty"`
	DisabledInputs     int      `json:"disabledInputs,omitempty"`
	Bytes              int64    `json:"bytes,omitempty"`
	NodeAPIs           []string `json:"nodeAPIs,omitempty"`
	DynamicRequires    []string `json:"dynamicRequires,omitempty"`
	ExternalImports    []string `json:"externalImports,omitempty"`
	PackagePatches     []string `json:"packagePatches,omitempty"`
	CompatibilityTests []string `json:"compatibilityTests,omitempty"`
}

type inventorySurface struct {
	ID                 string   `json:"id"`
	Kind               string   `json:"kind"`
	Classification     string   `json:"classification"`
	Status             string   `json:"status"`
	Owner              string   `json:"owner"`
	Used               bool     `json:"used"`
	UsedByPackages     []string `json:"usedByPackages,omitempty"`
	Aliases            []string `json:"aliases,omitempty"`
	Globals            []string `json:"globals,omitempty"`
	Exports            []string `json:"exports,omitempty"`
	Dependencies       []string `json:"dependencies,omitempty"`
	Resources          []string `json:"resources,omitempty"`
	Package            string   `json:"package,omitempty"`
	VersionRange       string   `json:"versionRange,omitempty"`
	PatchType          string   `json:"patchType,omitempty"`
	Required           *bool    `json:"required,omitempty"`
	Expected           string   `json:"expected,omitempty"`
	RemovalCondition   string   `json:"removalCondition,omitempty"`
	CompatibilityTests []string `json:"compatibilityTests,omitempty"`
	Reason             string   `json:"reason,omitempty"`
}

type packageAccumulator struct {
	name               string
	version            string
	dependencyClass    string
	externalService    bool
	inputs             int
	disabledInputs     int
	bytes              int64
	nodeAPIs           map[string]bool
	dynamicRequires    map[string]bool
	externalImports    map[string]bool
	packagePatches     map[string]bool
	compatibilityTests map[string]bool
}

func main() {
	manifestPath := flag.String("manifest", "internal/embed/agent/bundle/compat/manifest.json", "compat manifest path")
	metaPath := flag.String("meta", "internal/embed/agent/bundle/meta.json", "esbuild metafile path")
	packagePath := flag.String("package", "internal/embed/agent/bundle/package.json", "agent bundle package.json path")
	outPath := flag.String("out", "", "write inventory to this path instead of stdout")
	checkPath := flag.String("check", "", "compare generated inventory with a checked-in inventory path")
	flag.Parse()

	inventory, err := buildInventory(*manifestPath, *metaPath, *packagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "compat-inventory: %v\n", err)
		os.Exit(1)
	}

	if *checkPath != "" {
		if err := checkInventory(inventory, *checkPath); err != nil {
			fmt.Fprintf(os.Stderr, "compat-inventory: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("compat inventory matches %s\n", *checkPath)
		return
	}

	data, err := json.MarshalIndent(inventory, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "compat-inventory: marshal: %v\n", err)
		os.Exit(1)
	}
	data = append(data, '\n')

	if *outPath != "" {
		if err := os.WriteFile(*outPath, data, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "compat-inventory: write %s: %v\n", *outPath, err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s\n", *outPath)
		return
	}
	_, _ = os.Stdout.Write(data)
}

func buildInventory(manifestPath, metaPath, packagePath string) (compatInventory, error) {
	var manifest compatManifest
	if err := readJSON(manifestPath, &manifest); err != nil {
		return compatInventory{}, err
	}
	var meta esbuildMeta
	if err := readJSON(metaPath, &meta); err != nil {
		return compatInventory{}, err
	}
	var pkgJSON bundlePackageJSON
	if err := readJSON(packagePath, &pkgJSON); err != nil {
		return compatInventory{}, err
	}

	manifestByID := manifestEntriesByID(manifest)
	aliasToID := manifestAliases(manifest)
	rootClasses := rootDependencyClasses(pkgJSON)
	packages := map[string]*packageAccumulator{}
	surfaceUsers := map[string]map[string]bool{}
	usedSurfaces := map[string]bool{}

	for inputPath, input := range meta.Inputs {
		ref, ok := packageFromPath(inputPath)
		if !ok || ref.name == "" {
			continue
		}
		acc := ensurePackage(packages, ref.name, ref.version, rootClasses)
		acc.inputs++
		acc.bytes += input.Bytes
		if ref.disabled {
			acc.disabledInputs++
		}

		for _, imp := range input.Imports {
			if strings.HasPrefix(imp.Path, "node-stub:") {
				raw := strings.TrimPrefix(imp.Path, "node-stub:")
				id := normalizeNodeModuleID(raw, aliasToID, manifestByID)
				acc.nodeAPIs[id] = true
				markSurfaceUser(surfaceUsers, id, ref.name)
				usedSurfaces[id] = true
				if entry, ok := manifestByID[id]; ok {
					addTests(acc.compatibilityTests, entry.Tests)
				}
				continue
			}
			if imp.External && isInventoryExternal(imp.Path, manifestByID) {
				acc.externalImports[imp.Path] = true
				markSurfaceUser(surfaceUsers, imp.Path, ref.name)
				usedSurfaces[imp.Path] = true
				if entry, ok := manifestByID[imp.Path]; ok {
					addTests(acc.compatibilityTests, entry.Tests)
				}
			}
		}
	}

	for _, entry := range manifest.Entries {
		switch entry.Kind {
		case "dynamic-require":
			usedSurfaces[entry.ID] = true
		case "package-patch":
			usedSurfaces[entry.ID] = true
			if entry.Package != "" {
				targets := matchingPackages(packages, entry.Package)
				if len(targets) == 0 {
					targets = []*packageAccumulator{ensurePackage(packages, entry.Package, "", rootClasses)}
				}
				for _, acc := range targets {
					acc.packagePatches[entry.ID] = true
					addTests(acc.compatibilityTests, entry.Tests)
				}
				markSurfaceUser(surfaceUsers, entry.ID, entry.Package)
			}
		}
	}

	inventoryPackages := make([]inventoryPackage, 0, len(packages))
	for _, acc := range packages {
		inventoryPackages = append(inventoryPackages, inventoryPackage{
			Name:               acc.name,
			Version:            acc.version,
			DependencyClass:    acc.dependencyClass,
			ExternalService:    acc.externalService,
			Inputs:             acc.inputs,
			DisabledInputs:     acc.disabledInputs,
			Bytes:              acc.bytes,
			NodeAPIs:           sortedKeys(acc.nodeAPIs),
			DynamicRequires:    sortedKeys(acc.dynamicRequires),
			ExternalImports:    sortedKeys(acc.externalImports),
			PackagePatches:     sortedKeys(acc.packagePatches),
			CompatibilityTests: sortedKeys(acc.compatibilityTests),
		})
	}
	sort.Slice(inventoryPackages, func(i, j int) bool {
		if inventoryPackages[i].Name == inventoryPackages[j].Name {
			return inventoryPackages[i].Version < inventoryPackages[j].Version
		}
		return inventoryPackages[i].Name < inventoryPackages[j].Name
	})

	surfaces := make([]inventorySurface, 0, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		users := sortedKeys(surfaceUsers[entry.ID])
		surfaces = append(surfaces, inventorySurface{
			ID:                 entry.ID,
			Kind:               entry.Kind,
			Classification:     classifySurface(entry),
			Status:             entry.Status,
			Owner:              entry.Owner,
			Used:               usedSurfaces[entry.ID],
			UsedByPackages:     users,
			Aliases:            sortedStrings(entry.Aliases),
			Globals:            sortedStrings(entry.Globals),
			Exports:            sortedStrings(entry.Exports),
			Dependencies:       sortedStrings(entry.Dependencies),
			Resources:          sortedStrings(entry.Resources),
			Package:            entry.Package,
			VersionRange:       entry.VersionRange,
			PatchType:          entry.PatchType,
			Required:           entry.Required,
			Expected:           entry.Expected,
			RemovalCondition:   entry.RemovalCondition,
			CompatibilityTests: sortedStrings(entry.Tests),
			Reason:             entry.Reason,
		})
	}
	sort.Slice(surfaces, func(i, j int) bool {
		if surfaces[i].Kind == surfaces[j].Kind {
			return surfaces[i].ID < surfaces[j].ID
		}
		return surfaces[i].Kind < surfaces[j].Kind
	})

	return compatInventory{
		SchemaVersion: 1,
		Sources: inventorySources{
			Manifest:       manifestPath,
			Meta:           metaPath,
			PackageJSON:    packagePath,
			PackageManager: pkgJSON.PackageManager,
		},
		PackageCount: len(inventoryPackages),
		SurfaceCount: len(surfaces),
		Packages:     inventoryPackages,
		Surfaces:     surfaces,
	}, nil
}

func checkInventory(generated compatInventory, path string) error {
	var checkedIn compatInventory
	if err := readJSON(path, &checkedIn); err != nil {
		return err
	}
	checkedJSON, _ := json.MarshalIndent(checkedIn, "", "  ")
	generatedJSON, _ := json.MarshalIndent(generated, "", "  ")
	if bytes.Equal(checkedJSON, generatedJSON) || reflect.DeepEqual(checkedIn, generated) {
		return nil
	}
	var buf bytes.Buffer
	buf.WriteString("checked-in inventory is stale\nchecked-in:\n")
	buf.Write(checkedJSON)
	buf.WriteString("\n\ngenerated:\n")
	buf.Write(generatedJSON)
	buf.WriteByte('\n')
	return fmt.Errorf("%s", buf.String())
}

func readJSON(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func manifestEntriesByID(manifest compatManifest) map[string]compatEntry {
	entries := make(map[string]compatEntry, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		entries[entry.ID] = entry
	}
	return entries
}

func manifestAliases(manifest compatManifest) map[string]string {
	aliases := map[string]string{}
	for _, entry := range manifest.Entries {
		for _, alias := range entry.Aliases {
			aliases[alias] = entry.ID
		}
	}
	return aliases
}

func normalizeNodeModuleID(raw string, aliases map[string]string, manifest map[string]compatEntry) string {
	id := strings.TrimPrefix(raw, "node:")
	if target, ok := aliases[id]; ok {
		return target
	}
	if strings.HasSuffix(id, "/") {
		trimmed := strings.TrimSuffix(id, "/")
		if _, ok := manifest[trimmed]; ok {
			return trimmed
		}
	}
	return id
}

type packageRef struct {
	name     string
	version  string
	disabled bool
}

func packageFromPath(path string) (packageRef, bool) {
	disabled := false
	if strings.HasPrefix(path, "(disabled):") {
		disabled = true
		path = strings.TrimPrefix(path, "(disabled):")
	}
	if strings.HasPrefix(path, "node-stub:") || strings.Contains(path, ":") && !strings.HasPrefix(path, "node_modules/") {
		return packageRef{}, false
	}

	const pnpmPrefix = "node_modules/.pnpm/"
	if idx := strings.Index(path, pnpmPrefix); idx >= 0 {
		rest := path[idx+len(pnpmPrefix):]
		parts := strings.SplitN(rest, "/node_modules/", 2)
		if len(parts) != 2 {
			return packageRef{}, false
		}
		name := packageNameFromNodeModules(parts[1])
		if name == "" {
			return packageRef{}, false
		}
		return packageRef{name: name, version: versionFromPNPMID(parts[0]), disabled: disabled}, true
	}

	const nmPrefix = "node_modules/"
	if idx := strings.Index(path, nmPrefix); idx >= 0 {
		name := packageNameFromNodeModules(path[idx+len(nmPrefix):])
		if name == "" || name == ".pnpm" {
			return packageRef{}, false
		}
		return packageRef{name: name, disabled: disabled}, true
	}
	return packageRef{}, false
}

func packageNameFromNodeModules(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		return ""
	}
	if strings.HasPrefix(parts[0], "@") {
		if len(parts) < 2 || parts[1] == "" {
			return ""
		}
		return parts[0] + "/" + parts[1]
	}
	return parts[0]
}

func versionFromPNPMID(id string) string {
	base := strings.SplitN(id, "_", 2)[0]
	at := strings.LastIndex(base, "@")
	if at < 0 || at == len(base)-1 {
		return ""
	}
	return base[at+1:]
}

func rootDependencyClasses(pkg bundlePackageJSON) map[string]string {
	out := map[string]string{}
	for name := range pkg.Dependencies {
		out[name] = "direct"
	}
	for name := range pkg.DevDependencies {
		out[name] = "dev"
	}
	return out
}

func ensurePackage(packages map[string]*packageAccumulator, name, version string, rootClasses map[string]string) *packageAccumulator {
	key := name + "@" + version
	if version == "" {
		key = name
	}
	if acc, ok := packages[key]; ok {
		if acc.version == "" && version != "" {
			acc.version = version
		}
		return acc
	}
	class := classifyPackage(name, rootClasses[name])
	acc := &packageAccumulator{
		name:               name,
		version:            version,
		dependencyClass:    class,
		externalService:    isExternalServicePackage(name),
		nodeAPIs:           map[string]bool{},
		dynamicRequires:    map[string]bool{},
		externalImports:    map[string]bool{},
		packagePatches:     map[string]bool{},
		compatibilityTests: map[string]bool{},
	}
	packages[key] = acc
	return acc
}

func matchingPackages(packages map[string]*packageAccumulator, name string) []*packageAccumulator {
	var out []*packageAccumulator
	for _, acc := range packages {
		if acc.name == name {
			out = append(out, acc)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].version < out[j].version
	})
	return out
}

func classifyPackage(name, rootClass string) string {
	prefix := "transitive"
	if rootClass != "" {
		prefix = rootClass
	}
	switch {
	case strings.HasPrefix(name, "@ai-sdk/"):
		return prefix + "-provider"
	case name == "ai" || strings.HasPrefix(name, "@ai-sdk/provider"):
		return prefix + "-ai-sdk"
	case name == "@mastra/core":
		return prefix + "-mastra-core"
	case name == "@mastra/evals":
		return prefix + "-mastra-evals"
	case name == "@mastra/rag":
		return prefix + "-mastra-rag"
	case name == "@mastra/memory":
		return prefix + "-mastra-memory"
	case strings.HasPrefix(name, "@mastra/voice"):
		return prefix + "-mastra-voice"
	case name == "@mastra/pg" || name == "@mastra/mongodb" || name == "@mastra/libsql" || name == "@mastra/upstash" || name == "@mastra/redis":
		return prefix + "-mastra-storage"
	case name == "@mastra/chroma" || name == "@mastra/pinecone" || name == "@mastra/qdrant":
		return prefix + "-mastra-vector"
	case name == "pg" || name == "mongodb" || strings.HasPrefix(name, "@libsql/") || strings.HasPrefix(name, "@redis/"):
		return prefix + "-storage-client"
	default:
		return prefix
	}
}

func isExternalServicePackage(name string) bool {
	switch {
	case strings.HasPrefix(name, "@ai-sdk/") && name != "@ai-sdk/provider" && name != "@ai-sdk/provider-utils":
		return true
	case strings.HasPrefix(name, "@mastra/voice"):
		return true
	case name == "@mastra/chroma" || name == "@mastra/pinecone" || name == "@mastra/qdrant" || name == "@mastra/upstash":
		return true
	case name == "pg" || name == "mongodb" || strings.HasPrefix(name, "@redis/"):
		return true
	default:
		return false
	}
}

func isInventoryExternal(path string, manifest map[string]compatEntry) bool {
	if path == "" || path == "<runtime>" || strings.Contains(path, "*") {
		return false
	}
	entry, ok := manifest[path]
	return ok && entry.Kind == "external"
}

func classifySurface(entry compatEntry) string {
	if entry.Kind == "package-patch" {
		return "package-patched"
	}
	switch entry.Status {
	case "exact":
		return "exact"
	case "compat":
		return "compat"
	case "stub":
		return "shape-only"
	case "unsupported":
		return "unsupported-guarded"
	default:
		return entry.Status
	}
}

func markSurfaceUser(users map[string]map[string]bool, surface, pkg string) {
	if surface == "" || pkg == "" {
		return
	}
	if users[surface] == nil {
		users[surface] = map[string]bool{}
	}
	users[surface][pkg] = true
}

func addTests(dst map[string]bool, tests []string) {
	for _, test := range tests {
		if strings.TrimSpace(test) != "" {
			dst[test] = true
		}
	}
}

func sortedKeys(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func sortedStrings(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}
