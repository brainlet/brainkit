package agentembed

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type ecosystemCanaryManifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	Canaries      []ecosystemCanary `json:"canaries"`
}

type ecosystemCanary struct {
	ID          string   `json:"id"`
	Package     string   `json:"package"`
	Version     string   `json:"version"`
	Lane        string   `json:"lane"`
	APIFamilies []string `json:"apiFamilies"`
	ProofTier   string   `json:"proofTier"`
	Expected    string   `json:"expected"`
	Runner      string   `json:"runner"`
}

func TestEcosystemCanaryManifestIsWellFormed(t *testing.T) {
	manifest := loadEcosystemCanaryManifest(t)
	if manifest.SchemaVersion != 1 {
		t.Fatalf("schemaVersion = %d, want 1", manifest.SchemaVersion)
	}

	allowedLanes := map[string]bool{
		"agent-embed":         true,
		"user-package-deploy": true,
		"future-resolver":     true,
	}
	allowedProofTiers := map[string]bool{
		"offline-fake":     true,
		"local-service":    true,
		"import-only":      true,
		"external-service": true,
		"live-provider":    true,
	}
	required := map[string]string{
		"ecosystem/http-gaxios-local":     "gaxios",
		"ecosystem/node-fetch-local":      "node-fetch",
		"ecosystem/readable-stream":       "readable-stream",
		"ecosystem/ajv-codegen":           "ajv",
		"ecosystem/yaml-parser":           "yaml",
		"ecosystem/frontmatter-parser":    "gray-matter",
		"ecosystem/xxhash-wasm":           "xxhash-wasm",
		"ecosystem/mongodb-boundary":      "mongodb",
		"ecosystem/chromadb-boundary":     "chromadb",
		"ecosystem/uuid-conditional":      "uuid",
		"ecosystem/lru-cache-conditional": "lru-cache",
		"ecosystem/bare-npm-rejected":     "uuid",
	}
	packageVersions := loadBundleDependencyVersions(t)
	seen := map[string]ecosystemCanary{}
	for _, canary := range manifest.Canaries {
		if strings.TrimSpace(canary.ID) == "" {
			t.Fatal("canary has empty id")
		}
		if _, exists := seen[canary.ID]; exists {
			t.Fatalf("duplicate canary id %q", canary.ID)
		}
		seen[canary.ID] = canary
		if strings.TrimSpace(canary.Package) == "" || strings.TrimSpace(canary.Version) == "" {
			t.Fatalf("%s: package and version are required", canary.ID)
		}
		if !allowedLanes[canary.Lane] {
			t.Fatalf("%s: invalid lane %q", canary.ID, canary.Lane)
		}
		if !allowedProofTiers[canary.ProofTier] {
			t.Fatalf("%s: invalid proofTier %q", canary.ID, canary.ProofTier)
		}
		if len(canary.APIFamilies) == 0 {
			t.Fatalf("%s: apiFamilies are required", canary.ID)
		}
		if strings.TrimSpace(canary.Expected) == "" || strings.TrimSpace(canary.Runner) == "" {
			t.Fatalf("%s: expected and runner are required", canary.ID)
		}
		if got := strings.TrimPrefix(packageVersions[canary.Package], "^"); got != canary.Version {
			t.Fatalf("%s: package.json version for %s = %q, want %q", canary.ID, canary.Package, packageVersions[canary.Package], canary.Version)
		}
		if canary.Runner == "fixtures/ts/ecosystem/bare-npm-rejected" {
			assertFixtureExists(t, "ecosystem/bare-npm-rejected")
		}
	}
	for id, pkg := range required {
		canary, ok := seen[id]
		if !ok {
			t.Fatalf("missing required ecosystem canary %q", id)
		}
		if canary.Package != pkg {
			t.Fatalf("%s package = %q, want %q", id, canary.Package, pkg)
		}
	}
}

func TestAgentEmbedEcosystemCanaryRuntime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"method": r.Method,
			"path":   r.URL.Path,
			"query":  r.URL.RawQuery,
			"header": r.Header.Get("X-Canary"),
			"body":   string(body),
		})
	}))
	defer server.Close()

	client := NewClient(ClientConfig{})
	sandbox, err := client.CreateSandbox(SandboxConfig{HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}
	defer sandbox.Close()

	result, err := sandbox.Eval(context.Background(), "ecosystem-canaries.js", fmt.Sprintf(`
		(async function() {
			var results = await globalThis.__agent_embed.ecosystemCanaries.runAll({ baseURL: %q });
			return JSON.stringify(results);
		})()
	`, server.URL))
	if err != nil {
		t.Fatalf("eval ecosystem canaries: %v", err)
	}

	var results []struct {
		ID     string          `json:"id"`
		OK     bool            `json:"ok"`
		Detail json.RawMessage `json:"detail"`
	}
	if err := json.Unmarshal([]byte(result), &results); err != nil {
		t.Fatalf("parse result %q: %v", result, err)
	}
	if len(results) != 11 {
		t.Fatalf("canary result count = %d, want 11: %s", len(results), result)
	}
	byID := map[string]json.RawMessage{}
	for _, result := range results {
		if !result.OK {
			t.Fatalf("%s ok=false", result.ID)
		}
		byID[result.ID] = result.Detail
	}
	assertJSONField(t, byID, "ecosystem/http-gaxios-local", "post.body", `{"value":"gaxios-post"}`)
	assertJSONField(t, byID, "ecosystem/node-fetch-local", "post.body", `{"value":"node-fetch-post"}`)
	assertJSONField(t, byID, "ecosystem/node-fetch-local", "exports.Headers", "function")
	assertJSONField(t, byID, "ecosystem/node-fetch-local", "exports.Blob", "function")
	assertJSONField(t, byID, "ecosystem/node-fetch-local", "exports.FormData", "function")
	assertJSONField(t, byID, "ecosystem/readable-stream", "output", "BRAINKIT")
	assertJSONField(t, byID, "ecosystem/xxhash-wasm", "h32", "debd63d6")
	assertJSONField(t, byID, "ecosystem/xxhash-wasm", "h64", "7f7a140612a5edb0")
	assertJSONField(t, byID, "ecosystem/yaml-parser", "name", "brainkit")
	assertJSONField(t, byID, "ecosystem/frontmatter-parser", "title", "Brainkit")
	assertBoundaryDetails(t, byID, "ecosystem/mongodb-boundary", "optional-native", "kerberos", "@mongodb-js/zstd", "snappy", "mongodb-client-encryption")
	assertBoundaryDetails(t, byID, "ecosystem/chromadb-boundary", "optional-native", "@chroma-core/default-embed")
	assertBoundaryDetails(t, byID, "ecosystem/chromadb-boundary", "native-addon", "fastembed")
}

func loadEcosystemCanaryManifest(t *testing.T) ecosystemCanaryManifest {
	t.Helper()
	var manifest ecosystemCanaryManifest
	data, err := os.ReadFile(filepath.Join("bundle", "compat", "ecosystem-canaries.json"))
	if err != nil {
		t.Fatalf("read ecosystem canary manifest: %v", err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse ecosystem canary manifest: %v", err)
	}
	return manifest
}

func loadBundleDependencyVersions(t *testing.T) map[string]string {
	t.Helper()
	var pkg struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	data, err := os.ReadFile(filepath.Join("bundle", "package.json"))
	if err != nil {
		t.Fatalf("read bundle package.json: %v", err)
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		t.Fatalf("parse bundle package.json: %v", err)
	}
	return pkg.Dependencies
}

func assertFixtureExists(t *testing.T, relPath string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		path := filepath.Join(dir, "fixtures", "ts", relPath, "index.ts")
		if _, err := os.Stat(path); err == nil {
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("fixture %s not found walking up from %s", relPath, wd)
		}
	}
}

func assertJSONField(t *testing.T, values map[string]json.RawMessage, id, field, want string) {
	t.Helper()
	raw, ok := values[id]
	if !ok {
		t.Fatalf("missing canary result %s", id)
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("parse detail for %s: %v", id, err)
	}
	for _, part := range strings.Split(field, ".") {
		obj, ok := value.(map[string]any)
		if !ok {
			t.Fatalf("%s detail field %s is not an object: %#v", id, field, value)
		}
		value = obj[part]
	}
	if got, _ := value.(string); got != want {
		t.Fatalf("%s detail field %s = %#v, want %q", id, field, value, want)
	}
}

func assertBoundaryDetails(t *testing.T, values map[string]json.RawMessage, id, boundaryClass string, specifiers ...string) {
	t.Helper()
	raw, ok := values[id]
	if !ok {
		t.Fatalf("missing canary result %s", id)
	}
	var detail struct {
		Boundaries []struct {
			Specifier     string `json:"specifier"`
			Code          string `json:"code"`
			BoundaryCode  string `json:"boundaryCode"`
			BoundaryClass string `json:"boundaryClass"`
			PackageName   string `json:"packageName"`
		} `json:"boundaries"`
	}
	if err := json.Unmarshal(raw, &detail); err != nil {
		t.Fatalf("parse boundary detail for %s: %v", id, err)
	}
	seen := map[string]bool{}
	for _, boundary := range detail.Boundaries {
		if boundary.BoundaryClass != boundaryClass {
			continue
		}
		if boundary.Code != "BRAINKIT_UNSUPPORTED_DYNAMIC_REQUIRE" ||
			boundary.BoundaryCode != "BRAINKIT_UNSUPPORTED_BOUNDARY" ||
			boundary.PackageName == "" {
			t.Fatalf("%s boundary for %s is not typed: %+v", id, boundary.Specifier, boundary)
		}
		seen[boundary.Specifier] = true
	}
	for _, specifier := range specifiers {
		if !seen[specifier] {
			t.Fatalf("%s missing %s boundary for %s; detail=%s", id, boundaryClass, specifier, string(raw))
		}
	}
}
