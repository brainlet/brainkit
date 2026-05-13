package agentembed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
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
	Aliases      []string `json:"aliases"`
	Globals      []string `json:"globals"`
	Exports      []string `json:"exports"`
	Dependencies []string `json:"dependencies"`
	Resources    []string `json:"resources"`
	Tests        []string `json:"tests"`
	Reason       string   `json:"reason"`

	Package          string `json:"package"`
	VersionRange     string `json:"versionRange"`
	PatchType        string `json:"patchType"`
	Required         *bool  `json:"required"`
	Expected         string `json:"expected"`
	RemovalCondition string `json:"removalCondition"`
}

type compatReport struct {
	SchemaVersion         int               `json:"schemaVersion"`
	NodeModulesUsed       []string          `json:"nodeModulesUsed"`
	NodeModuleAliasesUsed []nodeModuleAlias `json:"nodeModuleAliasesUsed"`
	ExternalImports       []string          `json:"externalImports"`
	DynamicRequires       []string          `json:"dynamicRequires"`
	PackagePatches        []string          `json:"packagePatches"`
}

type nodeModuleAlias struct {
	Raw        string `json:"raw"`
	Normalized string `json:"normalized"`
}

type capabilityMatrix struct {
	SchemaVersion int             `json:"schemaVersion"`
	Rows          []capabilityRow `json:"rows"`
}

type capabilityRow struct {
	ID         string            `json:"id"`
	Capability string            `json:"capability"`
	Status     string            `json:"status"`
	Tiers      []string          `json:"tiers"`
	Proofs     []capabilityProof `json:"proofs"`
	Notes      string            `json:"notes"`
}

type capabilityProof struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
	Name string `json:"name"`
	Tier string `json:"tier"`
}

type esbuildMeta struct {
	Inputs  map[string]json.RawMessage `json:"inputs"`
	Outputs map[string]struct {
		Imports []struct {
			Path     string `json:"path"`
			External bool   `json:"external"`
		} `json:"imports"`
	} `json:"outputs"`
}

func TestCompatManifestIsWellFormed(t *testing.T) {
	manifest := loadCompatManifest(t)

	if manifest.SchemaVersion != 1 {
		t.Fatalf("schemaVersion = %d, want 1", manifest.SchemaVersion)
	}

	allowedKinds := map[string]bool{
		"node-module":     true,
		"node-subpath":    true,
		"web-global":      true,
		"dynamic-require": true,
		"package-patch":   true,
		"external":        true,
	}
	allowedStatuses := map[string]bool{
		"exact":       true,
		"compat":      true,
		"stub":        true,
		"unsupported": true,
	}
	allowedPatchTypes := map[string]bool{
		"esbuild-plugin":     true,
		"post-build-rewrite": true,
		"entry-exclusion":    true,
	}

	seen := map[string]compatEntry{}
	aliases := map[string]string{}
	for _, entry := range manifest.Entries {
		if strings.TrimSpace(entry.ID) == "" {
			t.Fatal("manifest entry with empty id")
		}
		if _, exists := seen[entry.ID]; exists {
			t.Fatalf("duplicate manifest id %q", entry.ID)
		}
		seen[entry.ID] = entry
		if !allowedKinds[entry.Kind] {
			t.Fatalf("%s: invalid kind %q", entry.ID, entry.Kind)
		}
		if !allowedStatuses[entry.Status] {
			t.Fatalf("%s: invalid status %q", entry.ID, entry.Status)
		}
		if strings.TrimSpace(entry.Owner) == "" {
			t.Fatalf("%s: owner is required", entry.ID)
		}
		if len(entry.Tests) == 0 {
			t.Fatalf("%s: tests are required", entry.ID)
		}
		for _, testRef := range entry.Tests {
			assertManifestTestRefExists(t, entry.ID, testRef)
		}
		if strings.TrimSpace(entry.Reason) == "" {
			t.Fatalf("%s: reason is required", entry.ID)
		}
		if entry.Kind == "package-patch" {
			if strings.TrimSpace(entry.Package) == "" {
				t.Fatalf("%s: package patch package is required", entry.ID)
			}
			if strings.TrimSpace(entry.VersionRange) == "" {
				t.Fatalf("%s: package patch versionRange is required", entry.ID)
			}
			if entry.VersionRange == "bundled dependency graph" {
				t.Fatalf("%s: package patch versionRange must name the audited package version", entry.ID)
			}
			if !allowedPatchTypes[entry.PatchType] {
				t.Fatalf("%s: invalid package patch type %q", entry.ID, entry.PatchType)
			}
			if entry.Required == nil {
				t.Fatalf("%s: package patch required flag is required", entry.ID)
			}
			if strings.TrimSpace(entry.Expected) == "" {
				t.Fatalf("%s: package patch expected pattern is required", entry.ID)
			}
			if strings.TrimSpace(entry.RemovalCondition) == "" {
				t.Fatalf("%s: package patch removalCondition is required", entry.ID)
			}
		}
		if (entry.Kind == "node-module" || entry.Kind == "node-subpath") && strings.HasPrefix(entry.ID, "node:") {
			t.Fatalf("%s: node ids must be normalized without node: prefix", entry.ID)
		}
		for _, alias := range entry.Aliases {
			if strings.TrimSpace(alias) == "" {
				t.Fatalf("%s: empty alias", entry.ID)
			}
			if prev, exists := aliases[alias]; exists {
				t.Fatalf("alias %q points to both %q and %q", alias, prev, entry.ID)
			}
			aliases[alias] = entry.ID
		}
	}

	for _, required := range []string{
		"crypto",
		"stream",
		"fs",
		"fs/promises",
		"net",
		"tls",
		"@opentelemetry/api",
		"execa-dynamic-import",
		"tiktoken-fallback",
	} {
		if _, ok := seen[required]; !ok {
			t.Fatalf("required compatibility row %q is missing", required)
		}
	}
}

func TestMastraCapabilityMatrixIsWellFormed(t *testing.T) {
	matrix := loadCapabilityMatrix(t)

	if matrix.SchemaVersion != 1 {
		t.Fatalf("schemaVersion = %d, want 1", matrix.SchemaVersion)
	}

	allowedStatuses := map[string]bool{
		"supported":   true,
		"partial":     true,
		"import-only": true,
	}
	allowedTiers := map[string]bool{
		"import-only":      true,
		"offline-fake":     true,
		"local-service":    true,
		"live-provider":    true,
		"external-service": true,
	}
	allowedProofKinds := map[string]bool{
		"fixture": true,
		"example": true,
		"test":    true,
	}
	requiredRows := map[string]bool{
		"agent.generate":                  true,
		"agent.structured-output":         true,
		"agent.stream":                    true,
		"agent.tools":                     true,
		"workflow":                        true,
		"agent.internal-workflow-storage": true,
		"scorers-evals":                   true,
		"memory":                          true,
		"rag":                             true,
		"vector-stores":                   true,
		"storage-adapters":                true,
		"observability":                   true,
		"voice":                           true,
		"openai-provider":                 true,
		"provider-imports":                true,
	}

	seen := map[string]capabilityRow{}
	for _, row := range matrix.Rows {
		if strings.TrimSpace(row.ID) == "" {
			t.Fatal("capability row with empty id")
		}
		if _, exists := seen[row.ID]; exists {
			t.Fatalf("duplicate capability row %q", row.ID)
		}
		seen[row.ID] = row
		if strings.TrimSpace(row.Capability) == "" {
			t.Fatalf("%s: capability is required", row.ID)
		}
		if !allowedStatuses[row.Status] {
			t.Fatalf("%s: invalid status %q", row.ID, row.Status)
		}
		if strings.TrimSpace(row.Notes) == "" {
			t.Fatalf("%s: notes are required", row.ID)
		}
		if len(row.Tiers) == 0 {
			t.Fatalf("%s: tiers are required", row.ID)
		}
		tiers := map[string]bool{}
		for _, tier := range row.Tiers {
			if !allowedTiers[tier] {
				t.Fatalf("%s: invalid tier %q", row.ID, tier)
			}
			tiers[tier] = true
		}
		if len(row.Proofs) == 0 {
			t.Fatalf("%s: proofs are required", row.ID)
		}
		for _, proof := range row.Proofs {
			if !allowedProofKinds[proof.Kind] {
				t.Fatalf("%s: invalid proof kind %q", row.ID, proof.Kind)
			}
			if !allowedTiers[proof.Tier] {
				t.Fatalf("%s: invalid proof tier %q", row.ID, proof.Tier)
			}
			if !tiers[proof.Tier] {
				t.Fatalf("%s: proof tier %q not listed in row tiers", row.ID, proof.Tier)
			}
			if strings.TrimSpace(proof.Path) == "" {
				t.Fatalf("%s: proof path is required", row.ID)
			}
			assertCapabilityProofExists(t, proof)
		}
	}

	for id := range requiredRows {
		if _, ok := seen[id]; !ok {
			t.Fatalf("required capability row %q is missing", id)
		}
	}
	assertCapabilityHasProof(t, seen["scorers-evals"], "example", "custom-scorer")
	assertCapabilityHasProof(t, seen["agent.internal-workflow-storage"], "example", "working-memory")
	assertCapabilityHasProof(t, seen["agent.internal-workflow-storage"], "test", "internal/engine/deployment_typescript_test.go")
}

func TestBundleStubsMatchCompatManifest(t *testing.T) {
	manifest := loadCompatManifest(t)
	manifestNodeIDs := manifestNodeEntries(manifest)
	buildSource := readAgentFile(t, "bundle", "build.mjs")

	if strings.Contains(buildSource, `fallbackStub = "export default {};"`) ||
		strings.Contains(buildSource, "|| fallbackStub") {
		t.Fatal("build.mjs must not silently fall back to export default {} for undeclared Node builtins")
	}

	stubIDs := extractModuleStubIDs(t, buildSource)
	if !reflect.DeepEqual(stubIDs, manifestNodeIDs) {
		t.Fatalf("module stubs do not match node manifest entries\nstubs:    %v\nmanifest: %v", stubIDs, manifestNodeIDs)
	}
}

func TestJSBridgeOwnedModuleStubsAreThinExports(t *testing.T) {
	manifest := loadCompatManifest(t)
	buildSource := readAgentFile(t, "bundle", "build.mjs")
	stubs := extractModuleStubs(t, buildSource)

	for _, entry := range manifest.Entries {
		if !isJSBridgeOwnedNodeEntry(entry) {
			continue
		}
		source, ok := stubs[entry.ID]
		if !ok {
			t.Fatalf("%s: missing module stub", entry.ID)
		}
		if strings.Contains(source, "||") {
			t.Fatalf("%s: jsbridge-owned stubs must directly export jsbridge globals, not keep build-time fallback behavior:\n%s", entry.ID, source)
		}
		for _, global := range entry.Globals {
			if global == "GoSocket" || strings.HasPrefix(global, "__go_") {
				continue
			}
			if !strings.Contains(source, "globalThis."+global) {
				t.Fatalf("%s: jsbridge-owned stub does not reference declared global %q:\n%s", entry.ID, global, source)
			}
		}
	}
}

func TestBundleMetaMatchesCompatReport(t *testing.T) {
	manifest := loadCompatManifest(t)
	report := loadCompatReport(t)
	generated := buildCompatReport(t, manifest)

	if !reflect.DeepEqual(report, generated) {
		reportJSON, _ := json.MarshalIndent(report, "", "  ")
		generatedJSON, _ := json.MarshalIndent(generated, "", "  ")
		t.Fatalf("compat report is stale\nchecked-in:\n%s\n\ngenerated:\n%s", reportJSON, generatedJSON)
	}

	manifestByID := manifestEntriesByID(manifest)
	for _, id := range generated.NodeModulesUsed {
		entry, ok := manifestByID[id]
		if !ok {
			t.Fatalf("node module %q appears in bundle meta but is missing from manifest", id)
		}
		if entry.Kind != "node-module" && entry.Kind != "node-subpath" {
			t.Fatalf("node module %q manifest kind = %q", id, entry.Kind)
		}
	}
	for _, id := range generated.ExternalImports {
		entry, ok := manifestByID[id]
		if !ok {
			t.Fatalf("external import %q appears in bundle meta but is missing from manifest", id)
		}
		if entry.Kind != "external" {
			t.Fatalf("external import %q manifest kind = %q", id, entry.Kind)
		}
	}
}

func TestDynamicRequireShimMatchesCompatManifest(t *testing.T) {
	manifest := loadCompatManifest(t)
	embedSource := readAgentFile(t, "embed.go")

	matches := regexp.MustCompile(`mod === "([^"]+)"`).FindAllStringSubmatch(embedSource, -1)
	seen := map[string]bool{}
	for _, match := range matches {
		seen[match[1]] = true
	}

	actual := sortedKeys(seen)
	expected := sortedManifestIDs(manifest, "dynamic-require")
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("runtimeGlobalsJS dynamic require shim does not match manifest\nactual:   %v\nexpected: %v", actual, expected)
	}
}

func TestPackagePatchMarkersMatchCompatManifest(t *testing.T) {
	manifest := loadCompatManifest(t)
	buildSource := readAgentFile(t, "bundle", "build.mjs")

	matches := regexp.MustCompile(`requirePackagePatch\("([^"]+)"\)`).FindAllStringSubmatch(buildSource, -1)
	seen := map[string]bool{}
	for _, match := range matches {
		seen[match[1]] = true
	}
	if len(seen) == 0 {
		t.Fatal("build.mjs has no requirePackagePatch markers")
	}

	manifestByID := manifestEntriesByID(manifest)
	for _, id := range sortedKeys(seen) {
		entry, ok := manifestByID[id]
		if !ok {
			t.Fatalf("package patch marker %q is missing from manifest", id)
		}
		if entry.Kind != "package-patch" {
			t.Fatalf("package patch marker %q manifest kind = %q", id, entry.Kind)
		}
		if strings.TrimSpace(entry.Reason) == "" || len(entry.Tests) == 0 {
			t.Fatalf("package patch marker %q lacks reason/tests", id)
		}
	}

	for _, entry := range manifest.Entries {
		if entry.Kind != "package-patch" || entry.PatchType == "entry-exclusion" {
			continue
		}
		if !seen[entry.ID] {
			t.Fatalf("non-entry package patch %q has no requirePackagePatch marker in build.mjs", entry.ID)
		}
	}
}

func TestPostBuildPackagePatchesHaveMissPolicy(t *testing.T) {
	manifest := loadCompatManifest(t)
	buildSource := readAgentFile(t, "bundle", "build.mjs")

	if strings.Contains(buildSource, `console.warn("WARNING:`) {
		t.Fatal("post-build package patches must use packagePatchMissed instead of ambiguous WARNING logs")
	}

	for _, entry := range manifest.Entries {
		if entry.Kind != "package-patch" || entry.PatchType != "post-build-rewrite" {
			continue
		}
		if !strings.Contains(buildSource, `requirePackagePatch("`+entry.ID+`")`) {
			t.Fatalf("%s: post-build package patch missing requirePackagePatch marker", entry.ID)
		}
		if !strings.Contains(buildSource, `packagePatchMissed("`+entry.ID+`"`) {
			t.Fatalf("%s: post-build package patch missing packagePatchMissed policy", entry.ID)
		}
	}
}

func TestPackagePatchMissDiagnosticsIncludeContext(t *testing.T) {
	buildSource := readAgentFile(t, "bundle", "build.mjs")

	for _, want := range []string{
		"packagePatchInstalledVersion",
		"Package:",
		"Version range:",
		"Patch type:",
		"Expected:",
		"Removal condition:",
	} {
		if !strings.Contains(buildSource, want) {
			t.Fatalf("package patch miss diagnostics missing %q", want)
		}
	}
}

func loadCompatManifest(t *testing.T) compatManifest {
	t.Helper()
	var manifest compatManifest
	readJSON(t, &manifest, "bundle", "compat", "manifest.json")
	return manifest
}

func loadCompatReport(t *testing.T) compatReport {
	t.Helper()
	var report compatReport
	readJSON(t, &report, "bundle", "compat", "report.json")
	return report
}

func loadCapabilityMatrix(t *testing.T) capabilityMatrix {
	t.Helper()
	var matrix capabilityMatrix
	readJSON(t, &matrix, "bundle", "compat", "capability-matrix.json")
	return matrix
}

func buildCompatReport(t *testing.T, manifest compatManifest) compatReport {
	t.Helper()
	var meta esbuildMeta
	readJSON(t, &meta, "bundle", "meta.json")

	aliasToID := manifestAliases(manifest)
	manifestByID := manifestEntriesByID(manifest)

	nodeUsed := map[string]bool{}
	aliasesUsed := map[nodeModuleAlias]bool{}
	for input := range meta.Inputs {
		if !strings.HasPrefix(input, "node-stub:") {
			continue
		}
		raw := strings.TrimPrefix(input, "node-stub:")
		normalized := normalizeNodeModuleID(raw, aliasToID, manifestByID)
		nodeUsed[normalized] = true
		if raw != normalized {
			aliasesUsed[nodeModuleAlias{Raw: raw, Normalized: normalized}] = true
		}
	}

	externalImports := map[string]bool{}
	for _, output := range meta.Outputs {
		for _, imp := range output.Imports {
			if imp.External {
				externalImports[imp.Path] = true
			}
		}
	}

	return compatReport{
		SchemaVersion:         1,
		NodeModulesUsed:       sortedKeys(nodeUsed),
		NodeModuleAliasesUsed: sortedAliases(aliasesUsed),
		ExternalImports:       sortedKeys(externalImports),
		DynamicRequires:       sortedManifestIDs(manifest, "dynamic-require"),
		PackagePatches:        sortedManifestIDs(manifest, "package-patch"),
	}
}

func assertCapabilityProofExists(t *testing.T, proof capabilityProof) {
	t.Helper()
	root := repoRoot(t)
	var path string
	switch proof.Kind {
	case "fixture":
		path = filepath.Join(root, "fixtures", "ts", filepath.FromSlash(proof.Path), "index.ts")
	case "example":
		path = filepath.Join(root, "examples", filepath.FromSlash(proof.Path), "main.go")
	case "test":
		path = filepath.Join(root, filepath.FromSlash(proof.Path))
	default:
		t.Fatalf("unknown proof kind %q", proof.Kind)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s proof %q is missing: %v", proof.Kind, proof.Path, err)
	}
	if proof.Kind == "test" && proof.Name != "" && !strings.Contains(string(data), proof.Name) {
		t.Fatalf("test proof %q does not contain %q", proof.Path, proof.Name)
	}
}

func assertCapabilityHasProof(t *testing.T, row capabilityRow, kind, path string) {
	t.Helper()
	for _, proof := range row.Proofs {
		if proof.Kind == kind && proof.Path == path {
			return
		}
	}
	t.Fatalf("%s: missing %s proof %q", row.ID, kind, path)
}

func assertManifestTestRefExists(t *testing.T, entryID, ref string) {
	t.Helper()
	root := repoRoot(t)
	path := filepath.Join(root, filepath.FromSlash(ref))
	if _, err := os.Stat(path); err == nil {
		return
	}
	if strings.HasPrefix(ref, "examples/") {
		if _, err := os.Stat(filepath.Join(path, "main.go")); err == nil {
			return
		}
	}
	t.Fatalf("%s: test/proof reference %q does not exist", entryID, ref)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func readJSON[T any](t *testing.T, out *T, parts ...string) {
	t.Helper()
	data := readAgentFile(t, parts...)
	if err := json.Unmarshal([]byte(data), out); err != nil {
		t.Fatalf("parse %s: %v", filepath.Join(parts...), err)
	}
}

func readAgentFile(t *testing.T, parts ...string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(append([]string{filepath.Dir(file)}, parts...)...)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func extractModuleStubIDs(t *testing.T, buildSource string) []string {
	t.Helper()
	stubs := extractModuleStubs(t, buildSource)
	ids := make([]string, 0, len(stubs))
	for id := range stubs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func extractModuleStubs(t *testing.T, buildSource string) map[string]string {
	t.Helper()
	start := strings.Index(buildSource, "const moduleStubs = {")
	if start < 0 {
		t.Fatal("moduleStubs block not found")
	}
	end := strings.Index(buildSource[start:], "\n};")
	if end < 0 {
		t.Fatal("moduleStubs block end not found")
	}
	block := buildSource[start : start+end]
	matches := regexp.MustCompile("(?ms)^\\s*\"([^\"]+)\":\\s*`(.*?)`,").FindAllStringSubmatch(block, -1)
	stubs := make(map[string]string, len(matches))
	for _, match := range matches {
		stubs[match[1]] = match[2]
	}
	return stubs
}

func isJSBridgeOwnedNodeEntry(entry compatEntry) bool {
	if entry.Kind != "node-module" && entry.Kind != "node-subpath" {
		return false
	}
	owner := strings.ToLower(entry.Owner)
	return strings.Contains(owner, "internal/jsbridge.") && !strings.Contains(owner, "legacy build.mjs")
}

func manifestNodeEntries(manifest compatManifest) []string {
	var ids []string
	for _, entry := range manifest.Entries {
		if entry.Kind == "node-module" || entry.Kind == "node-subpath" {
			ids = append(ids, entry.ID)
		}
	}
	sort.Strings(ids)
	return ids
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

func sortedManifestIDs(manifest compatManifest, kind string) []string {
	ids := map[string]bool{}
	for _, entry := range manifest.Entries {
		if entry.Kind == kind {
			ids[entry.ID] = true
		}
	}
	return sortedKeys(ids)
}

func sortedKeys(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func sortedAliases(values map[nodeModuleAlias]bool) []nodeModuleAlias {
	out := make([]nodeModuleAlias, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Raw == out[j].Raw {
			return out[i].Normalized < out[j].Normalized
		}
		return out[i].Raw < out[j].Raw
	})
	return out
}
