package brainkit

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestRootExportManifest(t *testing.T) {
	expectedBytes, err := os.ReadFile(filepath.Join("api", "brainkit-root.exports"))
	if err != nil {
		t.Fatalf("read root export manifest: %v", err)
	}
	expected := nonEmptyLines(string(expectedBytes))

	fset := token.NewFileSet()
	var actual []string
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob root go files: %v", err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, decl := range parsed.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && d.Name.IsExported() {
					actual = append(actual, "func "+d.Name.Name)
				}
			case *ast.GenDecl:
				kind := d.Tok.String()
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if s.Name.IsExported() {
							actual = append(actual, kind+" "+s.Name.Name)
						}
					case *ast.ValueSpec:
						for _, name := range s.Names {
							if name.IsExported() {
								actual = append(actual, kind+" "+name.Name)
							}
						}
					}
				}
			}
		}
	}
	sort.Strings(actual)
	if strings.Join(actual, "\n") != strings.Join(expected, "\n") {
		t.Fatalf("root export manifest drifted\nexpected:\n%s\n\nactual:\n%s", strings.Join(expected, "\n"), strings.Join(actual, "\n"))
	}
}

func TestRootDependencyManifest(t *testing.T) {
	expectedBytes, err := os.ReadFile(filepath.Join("api", "brainkit-root.deps"))
	if err != nil {
		t.Fatalf("read root dependency manifest: %v", err)
	}
	expected := nonEmptyLines(string(expectedBytes))

	out, err := exec.Command("go", "list", "-deps", "-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go list root deps: %v\n%s", err, out)
	}
	actual := nonEmptyLines(string(out))
	if strings.Join(actual, "\n") != strings.Join(expected, "\n") {
		t.Fatalf("root dependency manifest drifted; run `make deps-root-save` if intentional\nexpected:\n%s\n\nactual:\n%s", strings.Join(expected, "\n"), strings.Join(actual, "\n"))
	}
}

func TestRootDoesNotExposeModuleAuthoringAliases(t *testing.T) {
	forbidden := map[string]bool{
		"Module":                     true,
		"ModuleStatus":               true,
		"ModuleStatusStable":         true,
		"ModuleStatusBeta":           true,
		"ModuleStatusWIP":            true,
		"StatusReporter":             true,
		"ModuleContext":              true,
		"ModuleFactory":              true,
		"ModuleFactoryFunc":          true,
		"ModuleDescriptor":           true,
		"ModuleMessageDescriptor":    true,
		"ModuleCapabilityDescriptor": true,
		"ModuleDescriber":            true,
		"RegisterModule":             true,
		"LookupModuleFactory":        true,
		"RegisteredModuleNames":      true,
		"RegisteredModules":          true,
	}

	fset := token.NewFileSet()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob root go files: %v", err)
	}
	var violations []string
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, decl := range parsed.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && forbidden[d.Name.Name] {
					violations = append(violations, file+": func "+d.Name.Name)
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if forbidden[s.Name.Name] {
							violations = append(violations, file+": type "+s.Name.Name)
						}
					case *ast.ValueSpec:
						for _, name := range s.Names {
							if forbidden[name.Name] {
								violations = append(violations, file+": "+d.Tok.String()+" "+name.Name)
							}
						}
					}
				}
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("root package must not re-export module authoring aliases; import github.com/brainlet/brainkit/module instead:\n%s", strings.Join(violations, "\n"))
	}
}

func nonEmptyLines(text string) []string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	sort.Strings(out)
	return out
}

func TestRootImportBoundary(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps .: %v\n%s", err, out)
	}

	forbidden := []struct {
		path   string
		reason string
	}{
		{"github.com/brainlet/brainkit/internal/jsbridge", "JS bridge belongs behind modules/jsruntime"},
		{"github.com/brainlet/brainkit/internal/jsruntime", "concrete JS runtime must stay outside the root API graph"},
		{"github.com/brainlet/brainkit/internal/embed/agent", "agent execution belongs behind the JS runtime module"},
		{"github.com/brainlet/brainkit/modulecap/harness", "harness capability belongs behind modules/jsruntime/modules/harness"},
		{"github.com/brainlet/brainkit/internal/braintest", "test runner implementation belongs behind modules/testing"},
		{"github.com/brainlet/brainkit/modules", "root must not depend on optional module packages"},
		{"github.com/brainlet/brainkit/modules/jsruntime", "root must not auto-register the concrete JS runtime module"},
		{"github.com/brainlet/brainkit/modules/packages", "package deployment helpers and commands belong in modules/packages"},
		{"github.com/brainlet/brainkit/modules/plugins", "plugin config and subprocess supervision belong in modules/plugins"},
		{"github.com/brainlet/brainkit/modules/schedules", "schedule config and scheduler runtime belong in modules/schedules"},
		{"github.com/brainlet/brainkit/modules/mcp", "MCP server config and clients belong in modules/mcp"},
		{"github.com/brainlet/brainkit/storagebridges", "storage bridge registration is optional runtime wiring"},
		{"github.com/brainlet/brainkit/internal/libsql", "embedded libsql server is optional storage/vector infrastructure"},
		{"github.com/brainlet/brainkit/internal/transport/backends", "concrete transport backends belong behind transports or tests"},
		{"github.com/ThreeDotsLabs/watermill", "Watermill adapters must stay behind optional transports; root uses Brainkit-owned in-process transport"},
		{"github.com/evanw/esbuild", "TypeScript/package bundling belongs behind modules/packages"},
		{"github.com/brainlet/brainkit/vendor_typescript", "TypeScript transpilation belongs behind modules/jsruntime or package tooling"},
		{"github.com/buke/quickjs-go", "QuickJS must not be in the light root API graph"},
		{"github.com/tetratelabs/wazero", "WASM runtime must not be in the light root API graph"},
		{"modernc.org/sqlite", "SQLite driver must stay behind optional storage bridges"},
		{"github.com/nats-io/nats.go", "NATS client must stay behind optional transports"},
		{"github.com/rabbitmq/amqp091-go", "AMQP client must stay behind optional transports"},
		{"github.com/redis/go-redis/v9", "Redis client must stay behind optional transports"},
		{"github.com/testcontainers/testcontainers-go", "testcontainers must not leak into the root API graph"},
		{"github.com/docker/docker", "Docker client must not leak into the root API graph"},
	}

	deps := strings.Split(strings.TrimSpace(string(out)), "\n")
	var violations []string
	for _, dep := range deps {
		dep = strings.TrimSpace(dep)
		if dep == "" {
			continue
		}
		for _, rule := range forbidden {
			if dep == rule.path || strings.HasPrefix(dep, rule.path+"/") {
				violations = append(violations, dep+" ("+rule.reason+")")
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("root package imported forbidden optional dependencies:\n%s", strings.Join(violations, "\n"))
	}
}

func TestTopLevelSDKHasNoFixedTopicMessages(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("sdk", "*.go"))
	if err != nil {
		t.Fatalf("glob sdk/*.go: %v", err)
	}

	allowedDynamic := map[string]bool{
		"CustomMsg":   true,
		"CustomEvent": true,
	}

	fset := token.NewFileSet()
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "BusTopic" || fn.Recv == nil || len(fn.Recv.List) == 0 {
				continue
			}
			recv := receiverName(fn.Recv.List[0].Type)
			if !allowedDynamic[recv] {
				t.Fatalf("%s defines %s.BusTopic in top-level sdk; fixed-topic schemas belong in module packages or sdk/systemmsg", file, recv)
			}
		}
	}
}

func TestPackageBundlingImplementationStaysPackageModuleOwned(t *testing.T) {
	if _, err := os.Stat(filepath.Join("modules", "packages", "bundlers", "esbuild", "register.go")); err != nil {
		t.Fatalf("package bundling implementation must live under modules/packages/bundlers/esbuild: %v", err)
	}
	if _, err := os.Stat(filepath.Join("internal", "deploy")); err == nil {
		t.Fatalf("internal/deploy must not exist; package bundling belongs to modules/packages/bundlers/esbuild")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat internal/deploy: %v", err)
	}

	var violations []string
	for _, file := range goFilesUnder(t, ".") {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		if strings.Contains(string(body), "github.com/brainlet/brainkit/internal/deploy") {
			violations = append(violations, file)
		}
	}
	if len(violations) > 0 {
		t.Fatalf("package bundling must not be imported from root internal/deploy:\n%s", strings.Join(violations, "\n"))
	}
}

func TestPackageBuilderStaysOptional(t *testing.T) {
	checkDeps := func(pkg string, forbidden ...string) {
		t.Helper()
		out, err := exec.Command("go", "list", "-deps", pkg).CombinedOutput()
		if err != nil {
			t.Fatalf("go list -deps %s: %v\n%s", pkg, err, out)
		}
		deps := strings.Split(strings.TrimSpace(string(out)), "\n")
		var violations []string
		for _, dep := range deps {
			dep = strings.TrimSpace(dep)
			if dep == "" {
				continue
			}
			for _, forbiddenPrefix := range forbidden {
				if dep == forbiddenPrefix || strings.HasPrefix(dep, forbiddenPrefix+"/") {
					violations = append(violations, dep)
				}
			}
		}
		if len(violations) > 0 {
			t.Fatalf("%s must keep package builders optional; found:\n%s", pkg, strings.Join(violations, "\n"))
		}
	}
	checkExactDeps := func(pkg string, forbidden ...string) {
		t.Helper()
		out, err := exec.Command("go", "list", "-deps", pkg).CombinedOutput()
		if err != nil {
			t.Fatalf("go list -deps %s: %v\n%s", pkg, err, out)
		}
		deps := strings.Split(strings.TrimSpace(string(out)), "\n")
		depSet := make(map[string]bool, len(deps))
		for _, dep := range deps {
			depSet[strings.TrimSpace(dep)] = true
		}
		var violations []string
		for _, forbiddenDep := range forbidden {
			if depSet[forbiddenDep] {
				violations = append(violations, forbiddenDep)
			}
		}
		if len(violations) > 0 {
			t.Fatalf("%s must keep package helper surfaces separate; found:\n%s", pkg, strings.Join(violations, "\n"))
		}
	}

	checkDeps("./modules/packages",
		"github.com/brainlet/brainkit/modules/packages/client",
		"github.com/brainlet/brainkit/modules/packages/scaffold",
		"github.com/brainlet/brainkit/modules/packages/source",
		"github.com/brainlet/brainkit/modules/packages/bundlers",
		"github.com/brainlet/brainkit/modules/packages/internal/deploy",
		"github.com/evanw/esbuild",
		"github.com/brainlet/brainkit/vendor_typescript",
		"github.com/buke/quickjs-go",
		"github.com/tetratelabs/wazero",
	)
	checkDeps("./modules/packages/source",
		"github.com/brainlet/brainkit/module",
		"github.com/brainlet/brainkit/modules/packages/client",
		"github.com/brainlet/brainkit/modules/packages/scaffold",
		"github.com/brainlet/brainkit/modules/packages/packagemsg",
		"github.com/brainlet/brainkit/sdk",
		"github.com/evanw/esbuild",
		"github.com/buke/quickjs-go",
		"github.com/tetratelabs/wazero",
	)
	checkDeps("./modules/packages/client",
		"github.com/brainlet/brainkit/modules/packages/bundlers",
		"github.com/brainlet/brainkit/modules/packages/internal/deploy",
		"github.com/brainlet/brainkit/modules/jsruntime",
		"github.com/evanw/esbuild",
		"github.com/brainlet/brainkit/vendor_typescript",
		"github.com/buke/quickjs-go",
		"github.com/tetratelabs/wazero",
	)
	checkDeps("./modules/packages/scaffold",
		"github.com/brainlet/brainkit/module",
		"github.com/brainlet/brainkit/modules/packages/client",
		"github.com/brainlet/brainkit/modules/packages/source",
		"github.com/brainlet/brainkit/modules/packages/packagemsg",
		"github.com/brainlet/brainkit/sdk",
		"github.com/evanw/esbuild",
		"github.com/buke/quickjs-go",
		"github.com/tetratelabs/wazero",
	)
	checkExactDeps("./modules/packages/client",
		"github.com/brainlet/brainkit/modules/packages",
	)
	checkExactDeps("./modules/packages/scaffold",
		"github.com/brainlet/brainkit/modules/packages",
	)
	checkDeps("./modules/packages/bundlers/esbuild",
		"github.com/brainlet/brainkit/modules/jsruntime",
		"github.com/brainlet/brainkit/vendor_typescript",
		"github.com/buke/quickjs-go",
		"github.com/tetratelabs/wazero",
	)
	checkDeps("./modules/packages/internal/deploy",
		"github.com/evanw/esbuild",
		"github.com/brainlet/brainkit/vendor_typescript",
		"github.com/buke/quickjs-go",
		"github.com/tetratelabs/wazero",
	)
}

func TestPackagesConsumeArtifactDeployerNotRawSourceDeployer(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("modules", "packages", "module.go"))
	if err != nil {
		t.Fatalf("read packages module: %v", err)
	}
	text := string(body)
	required := []string{
		"CapabilityArtifactDeployer",
		"runtimecap.ArtifactDeployer",
		"DeployArtifact(",
	}
	for _, want := range required {
		if !strings.Contains(text, want) {
			t.Fatalf("packages module must consume artifact-only deploy capability; missing %q", want)
		}
	}
	forbidden := []string{
		"bkmodule.CapabilityDeployer",
		"runtimecap.Deployer",
		".Deploy(ctx, source, code",
	}
	for _, phrase := range forbidden {
		if strings.Contains(text, phrase) {
			t.Fatalf("packages module must not consume broad raw-source deploy capability; found %q", phrase)
		}
	}
}

func TestEvalModuleConsumesNarrowEvalRuntime(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("modules", "eval", "module.go"))
	if err != nil {
		t.Fatalf("read eval module: %v", err)
	}
	text := string(body)
	required := []string{
		"CapabilityEvalRuntime",
		"runtimecap.EvalRuntime",
		"EvalScript(",
	}
	for _, want := range required {
		if !strings.Contains(text, want) {
			t.Fatalf("eval module must consume the narrow eval runtime; missing %q", want)
		}
	}
	forbidden := []string{
		".Deploy(ctx,",
		".Teardown(ctx,",
		"runtimecap.Deployer",
		"CapabilityDeployer",
	}
	for _, phrase := range forbidden {
		if strings.Contains(text, phrase) {
			t.Fatalf("eval module must not consume raw deploy/teardown; found %q", phrase)
		}
	}
}

func TestEvalTSDoesNotOwnPackageBundling(t *testing.T) {
	bannedDocs := []string{
		"KitEvalMsg.Mode` is whitelisted to `script`, `ts`, `module`. `ts` transpiles via esbuild",
		"EvalTS bundles",
		"EvalTS runs esbuild",
		"kit.eval` TypeScript uses esbuild",
	}
	var violations []string
	for _, root := range []string{"docs", "README.md"} {
		info, err := os.Stat(root)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatalf("stat %s: %v", root, err)
		}
		check := func(path string) {
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			text := string(body)
			for _, phrase := range bannedDocs {
				if strings.Contains(text, phrase) {
					violations = append(violations, path+": "+phrase)
				}
			}
		}
		if !info.IsDir() {
			check(root)
			continue
		}
		if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				switch entry.Name() {
				case "node_modules", "vendor", "vendor_typescript", "vendor_quickjs":
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) == ".md" {
				check(path)
			}
			return nil
		}); err != nil {
			t.Fatalf("scan %s: %v", root, err)
		}
	}
	if len(violations) > 0 {
		t.Fatalf("EvalTS/kit.eval docs must not claim package/file bundling ownership; package bundling belongs to modules/packages:\n%s", strings.Join(violations, "\n"))
	}
}

func TestJSTestRuntimeIsNarrowModuleCapability(t *testing.T) {
	for _, file := range []string{
		filepath.Join("modules", "jsruntime", "module.go"),
		filepath.Join("modules", "jsruntime", "README.md"),
		filepath.Join("modules", "testing", "module.go"),
		filepath.Join("modules", "testing", "README.md"),
	} {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		text := string(body)
		if !strings.Contains(text, "CapabilityTestRuntime") && strings.HasSuffix(file, ".go") {
			t.Fatalf("%s must use the explicit test runtime capability", file)
		}
		if !strings.Contains(text, "brainkit.core.test_runtime") && strings.HasSuffix(file, ".md") {
			t.Fatalf("%s must document the explicit test runtime capability", file)
		}
		for _, forbidden := range []string{
			"CapabilitySourceDeployer",
			"CapabilityTSRunner",
			"brainkit.core.source_deployer",
			"brainkit.core.ts_runner",
		} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s must not expose broad source/eval runtime capability %q", file, forbidden)
			}
		}
	}
}

func TestTestingRunnerImplementationStaysTestingModuleOwned(t *testing.T) {
	if _, err := os.Stat(filepath.Join("modules", "testing", "internal", "braintest", "runner.go")); err != nil {
		t.Fatalf("test runner implementation must live under modules/testing/internal/braintest: %v", err)
	}
	if _, err := os.Stat(filepath.Join("internal", "braintest")); err == nil {
		t.Fatalf("internal/braintest must not exist; test runner implementation belongs to modules/testing/internal/braintest")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat internal/braintest: %v", err)
	}

	var violations []string
	for _, file := range goFilesUnder(t, ".") {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		if strings.Contains(string(body), "github.com/brainlet/brainkit/internal/braintest") {
			violations = append(violations, file)
		}
	}
	if len(violations) > 0 {
		t.Fatalf("test runner implementation must not be imported from root internal/braintest:\n%s", strings.Join(violations, "\n"))
	}
}

func TestRuntimeAssetsDoNotLiveUnderEngine(t *testing.T) {
	if _, err := os.Stat(filepath.Join("internal", "engine", "runtime")); err == nil {
		t.Fatalf("internal/engine/runtime must not exist; JS runtime assets belong under internal/jsruntime/runtime and declaration files under internal/dts/runtime")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat internal/engine/runtime: %v", err)
	}

	required := []string{
		filepath.Join("internal", "jsruntime", "runtime", "kit_runtime.js"),
		filepath.Join("internal", "jsruntime", "runtime", "bus.js"),
		filepath.Join("internal", "dts", "runtime", "kit.d.ts"),
		filepath.Join("internal", "dts", "runtime", "agent.d.ts"),
	}
	for _, file := range required {
		if _, err := os.Stat(file); err != nil {
			t.Fatalf("runtime asset owner missing %s: %v", file, err)
		}
	}

	roots := []string{
		"reference.go",
		filepath.Join("fixtures", "tsconfig.base.json"),
		"Makefile",
		filepath.Join("docs", "llm"),
		filepath.Join("examples", "voice-agent"),
		filepath.Join("examples", "hitl-tool-approval"),
		filepath.Join("examples", "hitl-workflow"),
		filepath.Join("examples", "evals"),
	}
	var violations []string
	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil {
			t.Fatalf("stat %s: %v", root, err)
		}
		check := func(path string) {
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			if strings.Contains(string(body), "internal/engine/runtime") {
				violations = append(violations, path)
			}
		}
		if !info.IsDir() {
			check(root)
			continue
		}
		if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			switch filepath.Ext(path) {
			case ".go", ".json", ".md":
				check(path)
			}
			return nil
		}); err != nil {
			t.Fatalf("scan %s: %v", root, err)
		}
	}
	if len(violations) > 0 {
		t.Fatalf("active runtime references must point at internal/jsruntime/runtime or internal/dts/runtime, not internal/engine/runtime:\n%s", strings.Join(violations, "\n"))
	}
}

func TestTopLevelSDKGeneratesNoTypedWrappers(t *testing.T) {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, filepath.Join("sdk", "typed_gen.go"), nil, 0)
	if err != nil {
		t.Fatalf("parse sdk/typed_gen.go: %v", err)
	}
	for _, decl := range parsed.Decls {
		if _, ok := decl.(*ast.FuncDecl); ok {
			t.Fatalf("top-level sdk/typed_gen.go must stay empty; generated wrappers belong in module packages or sdk/systemmsg")
		}
	}
}

func TestPublicMessagingDocsDoNotTeachRawReplyTopics(t *testing.T) {
	banned := []string{
		"Publish to bus with replyTo",
		"Callback function receives the reply",
		"Promise-based sendTo",
		"subscribes to replyTo",
		"SendTo publishes with replyTo",
		"Publish with replyTo",
		"bus publish+reply",
		"bus.publish+reply",
		"each goes through [sdk.Publish]",
		"or `sdk.SendToService(kit, ctx",
		"`sdk.SendToService` / event send",
		"publishes, waits for the reply on a private `replyTo` topic",
		"external callers reach them with `bus.publish(\"ts.<source>.<topic>\", …)`",
	}
	roots := []string{
		"doc.go",
		"README.md",
		"docs",
		"fixtures",
		filepath.Join("test", "fixtures"),
		filepath.Join("internal", "dts", "runtime"),
		filepath.Join("internal", "jsruntime", "runtime"),
		filepath.Join("internal", "engine", "runtime"),
	}
	allowedExt := map[string]bool{
		".d.ts": true,
		".js":   true,
		".md":   true,
		".ts":   true,
	}
	var violations []string
	for _, root := range roots {
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		}
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				switch entry.Name() {
				case "node_modules", "vendor", "vendor_typescript", "vendor_quickjs":
					return filepath.SkipDir
				}
				return nil
			}
			if !allowedExt[filepath.Ext(path)] && !strings.HasSuffix(path, ".d.ts") {
				return nil
			}
			body, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(body)
			for _, phrase := range banned {
				if strings.Contains(text, phrase) {
					violations = append(violations, path+": "+phrase)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", root, err)
		}
	}
	if len(violations) > 0 {
		t.Fatalf("public messaging docs/runtime helpers must point users at Call/CallStream, not raw reply topics:\n%s", strings.Join(violations, "\n"))
	}
}

func TestRawReplyTopicAPIsStayLowLevel(t *testing.T) {
	rawPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\bsdk\.PublishProtocol(?:To)?\s*\(`),
		regexp.MustCompile(`\bsdk\.Publish(?:To)?\s*\(`),
		regexp.MustCompile(`\bsdk\.SendToService(?:Protocol)?\s*\(`),
		regexp.MustCompile(`\bprotocol\.Publish(?:To)?\s*\(`),
		regexp.MustCompile(`\bprotocol\.SendToService\s*\(`),
		regexp.MustCompile(`\bprotocol\.WithReplyTo\s*\(`),
		regexp.MustCompile(`\bWith(?:Protocol)?ReplyTo\s*\(`),
		regexp.MustCompile(`\bSubscribe[A-Za-z0-9]+Resp\s*\(`),
	}
	allowed := func(file string) bool {
		if strings.HasSuffix(file, "_test.go") {
			return true
		}
		if strings.HasPrefix(file, "test"+string(filepath.Separator)) ||
			strings.HasPrefix(file, "fixtures"+string(filepath.Separator)) {
			return true
		}
		switch file {
		case filepath.Join("sdk", "protocol", "publish.go"),
			filepath.Join("cmd", "sdkgen", "main.go"):
			return true
		}
		return filepath.Base(file) == "typed_gen.go"
	}

	var violations []string
	for _, file := range goFilesUnder(t, ".") {
		if allowed(file) {
			continue
		}
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		text := string(body)
		for _, pattern := range rawPatterns {
			if pattern.MatchString(text) {
				violations = append(violations, file+": "+pattern.String())
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("raw reply-topic helpers are low-level protocol/test APIs; normal request/reply code must use Call/CallXxx:\n%s", strings.Join(violations, "\n"))
	}
}

func TestPluginAppCodeDoesNotImportSDKProtocol(t *testing.T) {
	allowed := map[string]string{
		filepath.Join("test", "suite", "plugins", "tool_call_bus_test.go"): "low-level plugin pass-through protocol regression test",
	}
	roots := []string{
		filepath.Join("test", "suite", "plugins"),
	}
	if _, err := os.Stat(filepath.Join("..", "plugins")); err == nil {
		roots = append(roots, filepath.Join("..", "plugins"))
	}

	var violations []string
	for _, root := range roots {
		for _, file := range goFilesUnder(t, root) {
			file = filepath.Clean(file)
			if _, ok := allowed[file]; ok {
				continue
			}
			parsed := parseImportsOnly(t, file)
			for _, imp := range parsed.Imports {
				path, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					t.Fatalf("unquote import %s in %s: %v", imp.Path.Value, file, err)
				}
				if path == "github.com/brainlet/brainkit/sdk/protocol" {
					violations = append(violations, file+": use sdk.Call/CallStream or generated CallXxx helpers; sdk/protocol is only for bridge, diagnostics, and protocol tests")
				}
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("plugin app and normal plugin-suite code must not import sdk/protocol:\n%s", strings.Join(violations, "\n"))
	}
}

func TestRootSuiteSDKProtocolImportsAreLowLevelAllowlisted(t *testing.T) {
	allowed := map[string]string{
		filepath.Join("test", "suite", "bus", "async.go"):                  "bus async fire-and-forget/protocol diagnostics",
		filepath.Join("test", "suite", "bus", "async_diag.go"):             "bus async diagnostic service-topic assertion",
		filepath.Join("test", "suite", "bus", "audit.go"):                  "bus audit raw envelope diagnostic",
		filepath.Join("test", "suite", "bus", "backend_advanced.go"):       "raw backend envelope/transport behavior",
		filepath.Join("test", "suite", "bus", "call_cancel_failfast.go"):   "caller cancellation and reply-topic failure behavior",
		filepath.Join("test", "suite", "bus", "cross_feature.go"):          "cross-feature raw envelope diagnostics",
		filepath.Join("test", "suite", "bus", "e2e.go"):                    "bus e2e raw envelope regression coverage",
		filepath.Join("test", "suite", "bus", "errors.go"):                 "reply-without-replyTo and raw error paths",
		filepath.Join("test", "suite", "bus", "failure.go"):                "caller failure/retry/dead-letter protocol coverage",
		filepath.Join("test", "suite", "bus", "failure_cascade.go"):        "failure cascade raw envelope coverage",
		filepath.Join("test", "suite", "bus", "integration.go"):            "bus integration raw response metadata assertions",
		filepath.Join("test", "suite", "bus", "messaging_no_module.go"):    "no-module envelope assertion for messaging bridge",
		filepath.Join("test", "suite", "bus", "publish.go"):                "explicit JS raw publish/reply-topic bridge coverage",
		filepath.Join("test", "suite", "bus", "pump.go"):                   "pump latency diagnostic service-topic coverage",
		filepath.Join("test", "suite", "bus", "reference.go"):              "reference no-module/raw envelope assertion",
		filepath.Join("test", "suite", "bus", "sdk_reply.go"):              "SDK reply-topic regression coverage",
		filepath.Join("test", "suite", "bus", "transport_matrix.go"):       "transport matrix protocol behavior",
		filepath.Join("test", "suite", "bus", "ts_call.go"):                "TS call diagnostic service-topic coverage",
		filepath.Join("test", "suite", "cross", "plugins.go"):              "cross-kit plugin raw envelope assertion",
		filepath.Join("test", "suite", "envelope", "shape.go"):             "wire envelope shape assertions",
		filepath.Join("test", "suite", "envelope", "typed_errors.go"):      "typed error envelope assertions",
		filepath.Join("test", "suite", "plugins", "tool_call_bus_test.go"): "plugin pass-through reply-topic regression test",
		filepath.Join("test", "suite", "scheduling", "no_module.go"):       "no-module eval envelope assertion",
		filepath.Join("test", "suite", "security", "bus_forgery.go"):       "reply-topic forgery/collision security tests",
		filepath.Join("test", "suite", "security", "cross_deploy.go"):      "cross-deploy isolation raw protocol tests",
		filepath.Join("test", "suite", "security", "data_leakage.go"):      "data leakage raw protocol security test",
		filepath.Join("test", "suite", "security", "internal_exploit.go"):  "internal exploit raw protocol tests",
		filepath.Join("test", "suite", "security", "run.go"):               "security suite raw message helper",
		filepath.Join("test", "suite", "security", "secrets.go"):           "secret leakage raw protocol security test",
		filepath.Join("test", "suite", "security", "timing.go"):            "timing/side-channel raw protocol tests",
	}

	seen := make(map[string]bool, len(allowed))
	var violations []string
	for _, root := range []string{filepath.Join("test", "suite"), filepath.Join("test", "bench")} {
		for _, file := range goFilesUnder(t, root) {
			parsed := parseImportsOnly(t, file)
			for _, imp := range parsed.Imports {
				path, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					t.Fatalf("unquote import %s in %s: %v", imp.Path.Value, file, err)
				}
				if path != "github.com/brainlet/brainkit/sdk/protocol" {
					continue
				}
				if _, ok := allowed[file]; !ok {
					violations = append(violations, file+": sdk/protocol imports in root suite/bench must be explicit low-level protocol, envelope, cross-namespace, or security coverage")
					continue
				}
				seen[file] = true
			}
		}
	}
	for file, reason := range allowed {
		if !seen[file] {
			violations = append(violations, file+": stale sdk/protocol allowlist entry ("+reason+")")
		}
	}
	if len(violations) > 0 {
		sort.Strings(violations)
		t.Fatalf("root suite/bench sdk/protocol imports must stay narrowly allowlisted:\n%s", strings.Join(violations, "\n"))
	}
}

func TestSDKProtocolPublishingLivesInProtocolPackage(t *testing.T) {
	forbiddenRootNames := map[string]struct{}{
		"PublishProtocol":       {},
		"PublishProtocolTo":     {},
		"SendToServiceProtocol": {},
		"WithProtocolReplyTo":   {},
		"ProtocolPublishResult": {},
		"ProtocolPublishOption": {},
		"ResolveServiceTopic":   {},
		"Publish":               {},
		"PublishTo":             {},
		"SendToService":         {},
		"WithReplyTo":           {},
		"PublishResult":         {},
		"PublishOption":         {},
	}
	requiredProtocolNames := map[string]struct{}{
		"Publish":             {},
		"PublishTo":           {},
		"SendToService":       {},
		"WithReplyTo":         {},
		"PublishResult":       {},
		"PublishOption":       {},
		"ResolveServiceTopic": {},
	}

	files, err := filepath.Glob(filepath.Join("sdk", "*.go"))
	if err != nil {
		t.Fatalf("glob sdk files: %v", err)
	}
	fset := token.NewFileSet()
	var violations []string
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		if parsed.Name.Name != "sdk" {
			continue
		}
		for _, decl := range parsed.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if _, forbidden := forbiddenRootNames[d.Name.Name]; forbidden {
					violations = append(violations, fmt.Sprintf("%s: root sdk must not expose low-level protocol func %s", file, d.Name.Name))
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if _, forbidden := forbiddenRootNames[s.Name.Name]; forbidden {
							violations = append(violations, fmt.Sprintf("%s: root sdk must not expose low-level protocol type %s", file, s.Name.Name))
						}
					case *ast.ValueSpec:
						for _, name := range s.Names {
							if _, forbidden := forbiddenRootNames[name.Name]; forbidden {
								violations = append(violations, fmt.Sprintf("%s: root sdk must not expose low-level protocol %s %s", file, d.Tok.String(), name.Name))
							}
						}
					}
				}
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("low-level protocol publishing must live under sdk/protocol, not root sdk:\n%s", strings.Join(violations, "\n"))
	}

	protocolFile := filepath.Join("sdk", "protocol", "publish.go")
	parsed := parseImportsOnly(t, protocolFile)
	if parsed.Name.Name != "protocol" {
		t.Fatalf("%s package = %q, want protocol", protocolFile, parsed.Name.Name)
	}
	full, err := parser.ParseFile(fset, protocolFile, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", protocolFile, err)
	}
	defined := map[string]struct{}{}
	for _, decl := range full.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			defined[d.Name.Name] = struct{}{}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				if s, ok := spec.(*ast.TypeSpec); ok {
					defined[s.Name.Name] = struct{}{}
				}
			}
		}
	}
	for name := range requiredProtocolNames {
		if _, ok := defined[name]; !ok {
			t.Fatalf("sdk/protocol must expose %s", name)
		}
	}
}

func TestStatefulModuleStoreBackendsStayOptional(t *testing.T) {
	checkDeps := func(pkg string, forbidden ...string) {
		t.Helper()
		out, err := exec.Command("go", "list", "-deps", pkg).CombinedOutput()
		if err != nil {
			t.Fatalf("go list -deps %s: %v\n%s", pkg, err, out)
		}
		text := string(out)
		var violations []string
		for _, dep := range forbidden {
			if strings.Contains(text, dep) {
				violations = append(violations, dep)
			}
		}
		if len(violations) > 0 {
			t.Fatalf("%s must keep store backends behind standard/store packages; found:\n%s", pkg, strings.Join(violations, "\n"))
		}
	}

	checkDeps("./modules/audit",
		"github.com/brainlet/brainkit/modules/audit/stores",
		"github.com/lib/pq",
		"modernc.org/sqlite",
	)
	checkDeps("./modules/audit/standard",
		"github.com/brainlet/brainkit/modules/audit/stores/postgres",
		"github.com/lib/pq",
	)
	checkDeps("./modules/audit/stores/sqlite",
		"github.com/brainlet/brainkit/modules/audit/stores/postgres",
		"github.com/lib/pq",
	)
	checkDeps("./modules/audit/stores/postgres",
		"github.com/brainlet/brainkit/modules/audit/stores/sqlite",
		"modernc.org/sqlite",
	)
	checkDeps("./modules/tracing",
		"modernc.org/sqlite",
	)
	checkDeps("./modules/schedules",
		"github.com/brainlet/brainkit/internal/store",
		"github.com/brainlet/brainkit/stores",
		"github.com/lib/pq",
		"modernc.org/sqlite",
	)
	checkDeps("./modules/schedules/standard",
		"github.com/brainlet/brainkit/stores/postgres",
		"github.com/brainlet/brainkit/internal/store/postgres",
		"github.com/lib/pq",
	)
	checkDeps("./stores/sqlite",
		"github.com/brainlet/brainkit/stores/postgres",
		"github.com/brainlet/brainkit/internal/store/postgres",
		"github.com/lib/pq",
	)
	checkDeps("./stores/postgres",
		"github.com/brainlet/brainkit/stores/sqlite",
		"github.com/brainlet/brainkit/internal/store/sqlite",
		"modernc.org/sqlite",
	)
	checkDeps("./server/configfile",
		"github.com/brainlet/brainkit/stores/postgres",
		"github.com/brainlet/brainkit/internal/store/postgres",
		"github.com/lib/pq",
	)
	checkDeps("./server/standard",
		"github.com/brainlet/brainkit/modules/audit/stores/postgres",
		"github.com/lib/pq",
	)

	out, err := exec.Command("go", "list", "-deps", "./server/standard").CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps ./server/standard: %v\n%s", err, out)
	}
	for _, want := range []string{
		"github.com/brainlet/brainkit/modules/audit/standard",
		"github.com/brainlet/brainkit/modules/schedules/standard",
		"github.com/brainlet/brainkit/modules/tracing/standard",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("server/standard must import %s for YAML module registration", want)
		}
	}
}

func TestTransportBackendsStayOptional(t *testing.T) {
	checkDeps := func(pkg string, forbidden ...string) {
		t.Helper()
		out, err := exec.Command("go", "list", "-deps", pkg).CombinedOutput()
		if err != nil {
			t.Fatalf("go list -deps %s: %v\n%s", pkg, err, out)
		}
		deps := strings.Split(strings.TrimSpace(string(out)), "\n")
		var violations []string
		for _, dep := range deps {
			dep = strings.TrimSpace(dep)
			if dep == "" {
				continue
			}
			for _, forbiddenPrefix := range forbidden {
				if dep == forbiddenPrefix || strings.HasPrefix(dep, forbiddenPrefix+"/") {
					violations = append(violations, dep)
				}
			}
		}
		if len(violations) > 0 {
			t.Fatalf("%s must keep transport backends optional; found:\n%s", pkg, strings.Join(violations, "\n"))
		}
	}

	checkDeps("./transports/nats",
		"github.com/ThreeDotsLabs/watermill-amqp",
		"github.com/ThreeDotsLabs/watermill-redisstream",
		"github.com/brainlet/brainkit/transports/amqp",
		"github.com/brainlet/brainkit/transports/embeddednats",
		"github.com/brainlet/brainkit/transports/redis",
		"github.com/nats-io/nats-server",
		"github.com/rabbitmq/amqp091-go",
		"github.com/redis/go-redis",
	)
	checkDeps("./transports/embeddednats",
		"github.com/ThreeDotsLabs/watermill-amqp",
		"github.com/ThreeDotsLabs/watermill-redisstream",
		"github.com/brainlet/brainkit/transports/amqp",
		"github.com/brainlet/brainkit/transports/redis",
		"github.com/rabbitmq/amqp091-go",
		"github.com/redis/go-redis",
	)
	checkDeps("./transports/amqp",
		"github.com/ThreeDotsLabs/watermill-nats",
		"github.com/ThreeDotsLabs/watermill-redisstream",
		"github.com/brainlet/brainkit/transports/embeddednats",
		"github.com/brainlet/brainkit/transports/nats",
		"github.com/brainlet/brainkit/transports/redis",
		"github.com/nats-io/nats-server",
		"github.com/nats-io/nats.go",
		"github.com/redis/go-redis",
	)
	checkDeps("./transports/redis",
		"github.com/ThreeDotsLabs/watermill-amqp",
		"github.com/ThreeDotsLabs/watermill-nats",
		"github.com/brainlet/brainkit/transports/amqp",
		"github.com/brainlet/brainkit/transports/embeddednats",
		"github.com/brainlet/brainkit/transports/nats",
		"github.com/nats-io/nats-server",
		"github.com/nats-io/nats.go",
		"github.com/rabbitmq/amqp091-go",
	)
	checkDeps("./server/configfile",
		"github.com/ThreeDotsLabs/watermill",
		"github.com/brainlet/brainkit/internal/transport/backends",
		"github.com/brainlet/brainkit/transports",
		"github.com/nats-io/nats-server",
		"github.com/nats-io/nats.go",
		"github.com/rabbitmq/amqp091-go",
		"github.com/redis/go-redis",
	)
	checkDeps("./server/standard",
		"github.com/ThreeDotsLabs/watermill",
		"github.com/brainlet/brainkit/internal/transport/backends",
		"github.com/brainlet/brainkit/server/configfile/transportbackends",
		"github.com/brainlet/brainkit/transports",
		"github.com/nats-io/nats-server",
		"github.com/nats-io/nats.go",
		"github.com/rabbitmq/amqp091-go",
		"github.com/redis/go-redis",
	)
	checkDeps("./server/quickstart",
		"github.com/ThreeDotsLabs/watermill-amqp",
		"github.com/ThreeDotsLabs/watermill-redisstream",
		"github.com/brainlet/brainkit/transports/amqp",
		"github.com/brainlet/brainkit/transports/redis",
		"github.com/rabbitmq/amqp091-go",
		"github.com/redis/go-redis",
	)
}

func TestStorageBridgeBackendsStayOptional(t *testing.T) {
	checkDeps := func(pkg string, forbidden ...string) {
		t.Helper()
		out, err := exec.Command("go", "list", "-deps", pkg).CombinedOutput()
		if err != nil {
			t.Fatalf("go list -deps %s: %v\n%s", pkg, err, out)
		}
		deps := strings.Split(strings.TrimSpace(string(out)), "\n")
		var violations []string
		for _, dep := range deps {
			dep = strings.TrimSpace(dep)
			if dep == "" {
				continue
			}
			for _, forbiddenPrefix := range forbidden {
				if dep == forbiddenPrefix || strings.HasPrefix(dep, forbiddenPrefix+"/") {
					violations = append(violations, dep)
				}
			}
		}
		if len(violations) > 0 {
			t.Fatalf("%s must keep storage bridge backends optional; found:\n%s", pkg, strings.Join(violations, "\n"))
		}
	}

	checkDeps("./modulehost/storagehost",
		"github.com/brainlet/brainkit/internal/libsql",
		"github.com/brainlet/brainkit/storagebridges",
		"modernc.org/sqlite",
	)
	checkDeps("./server/configfile",
		"github.com/brainlet/brainkit/internal/libsql",
		"github.com/brainlet/brainkit/storagebridges",
	)
	checkDeps("./server/standard",
		"github.com/brainlet/brainkit/internal/libsql",
		"github.com/brainlet/brainkit/storagebridges",
	)
}

func TestConfigFileStoreBackendsStayOptional(t *testing.T) {
	checkDeps := func(pkg string, forbidden ...string) {
		t.Helper()
		out, err := exec.Command("go", "list", "-deps", pkg).CombinedOutput()
		if err != nil {
			t.Fatalf("go list -deps %s: %v\n%s", pkg, err, out)
		}
		deps := strings.Split(strings.TrimSpace(string(out)), "\n")
		var violations []string
		for _, dep := range deps {
			dep = strings.TrimSpace(dep)
			if dep == "" {
				continue
			}
			for _, forbiddenPrefix := range forbidden {
				if dep == forbiddenPrefix || strings.HasPrefix(dep, forbiddenPrefix+"/") {
					violations = append(violations, dep)
				}
			}
		}
		if len(violations) > 0 {
			t.Fatalf("%s must keep configfile store backends optional; found:\n%s", pkg, strings.Join(violations, "\n"))
		}
	}

	checkDeps("./server/configfile",
		"github.com/brainlet/brainkit/internal/store",
		"github.com/brainlet/brainkit/server/configfile/storebackends",
		"github.com/brainlet/brainkit/stores",
		"github.com/lib/pq",
		"modernc.org/sqlite",
	)
	checkDeps("./server/configfile/storebackends/sqlite",
		"github.com/brainlet/brainkit/internal/store/postgres",
		"github.com/brainlet/brainkit/stores/postgres",
		"github.com/lib/pq",
	)
	checkDeps("./server/configfile/storebackends",
		"github.com/brainlet/brainkit/internal/store/postgres",
		"github.com/brainlet/brainkit/stores/postgres",
		"github.com/lib/pq",
	)
}

func TestConfigFilePackageBootStaysOptional(t *testing.T) {
	checkDeps := func(pkg string, forbidden ...string) {
		t.Helper()
		out, err := exec.Command("go", "list", "-deps", pkg).CombinedOutput()
		if err != nil {
			t.Fatalf("go list -deps %s: %v\n%s", pkg, err, out)
		}
		deps := strings.Split(strings.TrimSpace(string(out)), "\n")
		var violations []string
		for _, dep := range deps {
			dep = strings.TrimSpace(dep)
			if dep == "" {
				continue
			}
			for _, forbiddenPrefix := range forbidden {
				if dep == forbiddenPrefix || strings.HasPrefix(dep, forbiddenPrefix+"/") {
					violations = append(violations, dep)
				}
			}
		}
		if len(violations) > 0 {
			t.Fatalf("%s must keep top-level package auto-deploy optional; found:\n%s", pkg, strings.Join(violations, "\n"))
		}
	}

	checkDeps("./server/configfile",
		"github.com/brainlet/brainkit/modules/packages",
		"github.com/brainlet/brainkit/server/packageboot",
		"github.com/brainlet/brainkit/vendor_typescript",
		"github.com/evanw/esbuild",
	)
	checkDeps("./server/configfile/packageboot",
		"github.com/brainlet/brainkit/server/configfile/storebackends",
		"github.com/brainlet/brainkit/server/configfile/transportbackends",
		"github.com/brainlet/brainkit/storagebridges",
		"github.com/brainlet/brainkit/transports",
		"github.com/ThreeDotsLabs/watermill",
		"modernc.org/sqlite",
	)
}

func TestStandardProfilesStayScoped(t *testing.T) {
	checkDeps := func(pkg string, forbidden ...string) {
		t.Helper()
		out, err := exec.Command("go", "list", "-deps", pkg).CombinedOutput()
		if err != nil {
			t.Fatalf("go list -deps %s: %v\n%s", pkg, err, out)
		}
		deps := strings.Split(strings.TrimSpace(string(out)), "\n")
		var violations []string
		for _, dep := range deps {
			dep = strings.TrimSpace(dep)
			if dep == "" {
				continue
			}
			for _, forbiddenPrefix := range forbidden {
				if dep == forbiddenPrefix || strings.HasPrefix(dep, forbiddenPrefix+"/") {
					violations = append(violations, dep)
				}
			}
		}
		if len(violations) > 0 {
			t.Fatalf("%s standard profile pulled out-of-scope deps:\n%s", pkg, strings.Join(violations, "\n"))
		}
	}

	coreForbidden := []string{
		"github.com/brainlet/brainkit/modules/audit",
		"github.com/brainlet/brainkit/modules/eval",
		"github.com/brainlet/brainkit/modules/gateway",
		"github.com/brainlet/brainkit/modules/harness",
		"github.com/brainlet/brainkit/modules/jsruntime",
		"github.com/brainlet/brainkit/modules/mcp",
		"github.com/brainlet/brainkit/modules/packages",
		"github.com/brainlet/brainkit/modules/plugins",
		"github.com/brainlet/brainkit/modules/probes",
		"github.com/brainlet/brainkit/modules/schedules",
		"github.com/brainlet/brainkit/modules/testing",
		"github.com/brainlet/brainkit/modules/tracing",
		"github.com/brainlet/brainkit/modules/workflow",
		"github.com/buke/quickjs-go",
		"github.com/evanw/esbuild",
		"github.com/mark3labs/mcp-go",
		"modernc.org/sqlite",
	}
	checkDeps("./presets/standard/core", coreForbidden...)
	checkDeps("./server/standard/core", coreForbidden...)

	checkDeps("./presets/standard/runtime",
		"github.com/brainlet/brainkit/modules/audit",
		"github.com/brainlet/brainkit/modules/gateway",
		"github.com/brainlet/brainkit/modules/harness",
		"github.com/brainlet/brainkit/modules/mcp",
		"github.com/brainlet/brainkit/modules/packages",
		"github.com/brainlet/brainkit/modules/packages/bundlers",
		"github.com/brainlet/brainkit/modules/plugins",
		"github.com/brainlet/brainkit/modules/probes",
		"github.com/brainlet/brainkit/modules/schedules",
		"github.com/brainlet/brainkit/modules/testing",
		"github.com/brainlet/brainkit/modules/tracing",
		"github.com/brainlet/brainkit/modules/workflow",
		"github.com/evanw/esbuild",
		"github.com/mark3labs/mcp-go",
		"modernc.org/sqlite",
	)
	checkDeps("./server/standard/runtime",
		"github.com/brainlet/brainkit/modules/audit",
		"github.com/brainlet/brainkit/modules/gateway",
		"github.com/brainlet/brainkit/modules/harness",
		"github.com/brainlet/brainkit/modules/mcp",
		"github.com/brainlet/brainkit/modules/packages",
		"github.com/brainlet/brainkit/modules/packages/bundlers",
		"github.com/brainlet/brainkit/modules/plugins",
		"github.com/brainlet/brainkit/modules/probes",
		"github.com/brainlet/brainkit/modules/schedules",
		"github.com/brainlet/brainkit/modules/testing",
		"github.com/brainlet/brainkit/modules/tracing",
		"github.com/brainlet/brainkit/modules/workflow",
		"github.com/evanw/esbuild",
		"github.com/mark3labs/mcp-go",
		"modernc.org/sqlite",
	)
	checkDeps("./presets/standard/packages",
		"github.com/brainlet/brainkit/modules/audit",
		"github.com/brainlet/brainkit/modules/gateway",
		"github.com/brainlet/brainkit/modules/harness",
		"github.com/brainlet/brainkit/modules/mcp",
		"github.com/brainlet/brainkit/modules/plugins",
		"github.com/brainlet/brainkit/modules/probes",
		"github.com/brainlet/brainkit/modules/schedules",
		"github.com/brainlet/brainkit/modules/testing",
		"github.com/brainlet/brainkit/modules/tracing",
		"github.com/brainlet/brainkit/modules/workflow",
		"github.com/mark3labs/mcp-go",
		"modernc.org/sqlite",
	)
	checkDeps("./server/standard/packages",
		"github.com/brainlet/brainkit/modules/audit",
		"github.com/brainlet/brainkit/modules/gateway",
		"github.com/brainlet/brainkit/modules/harness",
		"github.com/brainlet/brainkit/modules/mcp",
		"github.com/brainlet/brainkit/modules/plugins",
		"github.com/brainlet/brainkit/modules/probes",
		"github.com/brainlet/brainkit/modules/schedules",
		"github.com/brainlet/brainkit/modules/testing",
		"github.com/brainlet/brainkit/modules/tracing",
		"github.com/brainlet/brainkit/modules/workflow",
		"github.com/mark3labs/mcp-go",
		"modernc.org/sqlite",
	)

	checkDeps("./server/standard/commands",
		"github.com/brainlet/brainkit/modules/audit",
		"github.com/brainlet/brainkit/modules/gateway",
		"github.com/brainlet/brainkit/modules/harness",
		"github.com/brainlet/brainkit/modules/mcp",
		"github.com/brainlet/brainkit/modules/plugins",
		"github.com/brainlet/brainkit/modules/probes",
		"github.com/brainlet/brainkit/modules/schedules",
		"github.com/brainlet/brainkit/modules/testing",
		"github.com/brainlet/brainkit/modules/topology",
		"github.com/brainlet/brainkit/modules/tracing",
		"github.com/brainlet/brainkit/modules/workflow",
		"github.com/mark3labs/mcp-go",
		"modernc.org/sqlite",
	)

	checkDeps("./server/standard/server",
		"github.com/brainlet/brainkit/modules/audit",
		"github.com/brainlet/brainkit/modules/harness",
		"github.com/brainlet/brainkit/modules/jsruntime",
		"github.com/brainlet/brainkit/modules/mcp",
		"github.com/brainlet/brainkit/modules/packages",
		"github.com/brainlet/brainkit/modules/plugins",
		"github.com/brainlet/brainkit/modules/schedules",
		"github.com/brainlet/brainkit/modules/testing",
		"github.com/brainlet/brainkit/modules/tracing",
		"github.com/brainlet/brainkit/modules/workflow",
		"github.com/buke/quickjs-go",
		"github.com/evanw/esbuild",
		"github.com/mark3labs/mcp-go",
		"modernc.org/sqlite",
	)

	checkDeps("./server/standard/observability",
		"github.com/brainlet/brainkit/modules/gateway",
		"github.com/brainlet/brainkit/modules/harness",
		"github.com/brainlet/brainkit/modules/jsruntime",
		"github.com/brainlet/brainkit/modules/mcp",
		"github.com/brainlet/brainkit/modules/packages",
		"github.com/brainlet/brainkit/modules/plugins",
		"github.com/brainlet/brainkit/modules/testing",
		"github.com/brainlet/brainkit/modules/workflow",
		"github.com/buke/quickjs-go",
		"github.com/evanw/esbuild",
		"github.com/mark3labs/mcp-go",
	)

	checkDeps("./server/standard/full",
		"github.com/brainlet/brainkit/internal/libsql",
		"github.com/brainlet/brainkit/server/configfile/storebackends",
		"github.com/brainlet/brainkit/server/configfile/transportbackends",
		"github.com/brainlet/brainkit/storagebridges",
		"github.com/brainlet/brainkit/transports",
	)
}

func TestPublicDocsDoNotTeachRemovedRootAPIs(t *testing.T) {
	banned := []string{
		"brainkit.MustJSON",
		"brainkit.RuntimeID",
		"brainkit.NewClient",
		"brainkit.BusClient",
		"brainkit.StreamEvent",
		"brainkit.Result",
		"brainkit.ResourceInfo",
		"brainkit.KernelMetrics",
		"brainkit.ErrorContext",
		"brainkit.Span",
		"brainkit.NewMemoryTraceStore",
		"brainkit.SecretMeta",
		"brainkit.Module",
		"brainkit.ModuleStatus",
		"brainkit.StatusReporter",
		"brainkit.RegisterModule",
		"brainkit.LookupModuleFactory",
		"brainkit.RegisteredModuleNames",
		"brainkit.RegisteredModules",
		"brainkit.ModuleContext",
		"brainkit.ModuleFactory",
		"brainkit.ModuleDescriptor",
	}
	roots := []string{
		"README.md",
		"docs",
		"examples",
	}
	allowedExt := map[string]bool{
		".go": true,
		".md": true,
		".ts": true,
	}
	var violations []string
	for _, root := range roots {
		info, err := os.Stat(root)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatalf("stat %s: %v", root, err)
		}
		if !info.IsDir() {
			body, err := os.ReadFile(root)
			if err != nil {
				t.Fatalf("read %s: %v", root, err)
			}
			for _, phrase := range banned {
				if strings.Contains(string(body), phrase) {
					violations = append(violations, root+": "+phrase)
				}
			}
			continue
		}
		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				switch entry.Name() {
				case "node_modules", "vendor", "vendor_typescript", "vendor_quickjs":
					return filepath.SkipDir
				}
				return nil
			}
			if !allowedExt[filepath.Ext(path)] {
				return nil
			}
			body, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(body)
			for _, phrase := range banned {
				if strings.Contains(text, phrase) {
					violations = append(violations, path+": "+phrase)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", root, err)
		}
	}
	if len(violations) > 0 {
		t.Fatalf("public docs/examples mention removed root APIs:\n%s", strings.Join(violations, "\n"))
	}
}

func TestPublicTransportSurfaceDoesNotExposeWatermill(t *testing.T) {
	roots := []string{
		"README.md",
		"docs",
		"sdk",
		"module",
		"modulecap",
		filepath.Join("internal", "engine"),
		filepath.Join("modulehost", "transporthost"),
	}
	allowedExt := map[string]bool{
		".go": true,
		".md": true,
	}
	var violations []string
	for _, root := range roots {
		info, err := os.Stat(root)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatalf("stat %s: %v", root, err)
		}
		checkFile := func(path string) {
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			text := string(body)
			if strings.Contains(text, "Watermill") || strings.Contains(text, "watermill") {
				violations = append(violations, path)
			}
		}
		if !info.IsDir() {
			checkFile(root)
			continue
		}
		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				switch entry.Name() {
				case "node_modules", "vendor", "vendor_typescript", "vendor_quickjs":
					return filepath.SkipDir
				}
				return nil
			}
			if allowedExt[filepath.Ext(path)] {
				checkFile(path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", root, err)
		}
	}
	if len(violations) > 0 {
		t.Fatalf("public transport surface must use Brainkit-owned messaging language; Watermill is an internal adapter detail:\n%s", strings.Join(violations, "\n"))
	}
}

func TestModulesDoNotImportRootBrainkit(t *testing.T) {
	files := goFilesUnder(t, "modules")
	var violations []string
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed := parseImportsOnly(t, file)
		for _, imp := range parsed.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if path == "github.com/brainlet/brainkit" {
				violations = append(violations, file)
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("module packages must not import root brainkit; use sdk/module/capabilities instead:\n%s", strings.Join(violations, "\n"))
	}
}

func TestModulesDoNotImportInternalEngine(t *testing.T) {
	files := goFilesUnder(t, "modules")
	var violations []string
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed := parseImportsOnly(t, file)
		for _, imp := range parsed.Imports {
			if strings.Trim(imp.Path.Value, `"`) == "github.com/brainlet/brainkit/internal/engine" {
				violations = append(violations, file)
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("modules must use module host capabilities, not internal/engine:\n%s", strings.Join(violations, "\n"))
	}
}

func TestJSDependentModulesDoNotAutoImportJSRuntime(t *testing.T) {
	jsDependentModules := []string{
		filepath.Join("modules", "eval"),
		filepath.Join("modules", "harness"),
		filepath.Join("modules", "packages"),
		filepath.Join("modules", "testing"),
		filepath.Join("modules", "workflow"),
	}
	var violations []string
	for _, root := range jsDependentModules {
		for _, file := range goFilesUnder(t, root) {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}
			parsed := parseImportsOnly(t, file)
			for _, imp := range parsed.Imports {
				if strings.Trim(imp.Path.Value, `"`) == "github.com/brainlet/brainkit/modules/jsruntime" {
					violations = append(violations, file)
				}
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("JS-dependent modules must require jsruntime by descriptor/capability but must not blank-import the concrete module; callers import modules/jsruntime or presets/standard explicitly:\n%s", strings.Join(violations, "\n"))
	}
}

func TestTopLevelModulesAreMountableModules(t *testing.T) {
	entries, err := os.ReadDir("modules")
	if err != nil {
		t.Fatalf("read modules: %v", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		moduleFile := filepath.Join("modules", entry.Name(), "module.go")
		if _, err := os.Stat(moduleFile); err != nil {
			t.Fatalf("top-level modules/%s must be a mountable module with module.go; non-mountable hosts belong in modulehost and contracts in modulecap: %v", entry.Name(), err)
		}
	}
}

func TestTopLevelModulesHaveContractReadmes(t *testing.T) {
	surfaceMarkers := []string{
		"## Bus commands",
		"## Surface",
		"## Runtime surface",
		"## Provider types",
		"## Route types",
		"## What this gives you",
		"## Transport requirement",
		"## Stores",
		"## Provides",
		"## Go tools",
		"## Usage",
	}
	entries, err := os.ReadDir("modules")
	if err != nil {
		t.Fatalf("read modules: %v", err)
	}
	var violations []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		readme := filepath.Join("modules", entry.Name(), "README.md")
		body, err := os.ReadFile(readme)
		if err != nil {
			violations = append(violations, fmt.Sprintf("modules/%s: missing README.md (%v)", entry.Name(), err))
			continue
		}
		text := string(body)
		if !strings.HasPrefix(text, "# modules/"+entry.Name()+" ") {
			violations = append(violations, readme+": first heading must start with '# modules/"+entry.Name()+" '")
		}
		hasSurface := false
		for _, marker := range surfaceMarkers {
			if strings.Contains(text, marker) {
				hasSurface = true
				break
			}
		}
		if !hasSurface {
			violations = append(violations, readme+": document bus commands, runtime surface, provided resources, or usage")
		}
		for _, marker := range []string{"## Capabilities", "## Runtime resources", "## Hot unmount"} {
			if !strings.Contains(text, marker) {
				violations = append(violations, readme+": missing "+marker)
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("top-level module READMEs must document the module contract:\n%s", strings.Join(violations, "\n"))
	}
}

func TestModuleReadmesMentionReferencedCapabilitiesAndResources(t *testing.T) {
	capabilities := capabilityConstValues(t)
	resourceNameRe := regexp.MustCompile(`bkmodule\.Resource(?:WithMetadata)?\([^,]+,\s*"([^"]+)"`)

	entries, err := os.ReadDir("modules")
	if err != nil {
		t.Fatalf("read modules: %v", err)
	}
	var violations []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join("modules", entry.Name())
		readme := filepath.Join(dir, "README.md")
		body, err := os.ReadFile(readme)
		if err != nil {
			t.Fatalf("read %s: %v", readme, err)
		}
		text := string(body)

		var source strings.Builder
		for _, file := range goFilesUnder(t, dir) {
			if strings.HasSuffix(file, "_test.go") || strings.HasSuffix(file, "typed_gen.go") {
				continue
			}
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read %s: %v", file, err)
			}
			source.Write(data)
			source.WriteByte('\n')
		}
		src := source.String()

		var required []string
		for constName, value := range capabilities {
			if strings.Contains(src, "bkmodule."+constName) {
				required = append(required, value)
			}
		}
		if strings.Contains(src, `"discovery.provider"`) {
			required = append(required, "discovery.provider")
		}
		for _, capability := range uniqueSortedStrings(required) {
			if !strings.Contains(text, capability) {
				violations = append(violations, readme+": missing capability "+capability)
			}
		}

		var resources []string
		for _, match := range resourceNameRe.FindAllStringSubmatch(src, -1) {
			resources = append(resources, match[1])
		}
		for _, resource := range uniqueSortedStrings(resources) {
			if !strings.Contains(text, resource) {
				violations = append(violations, readme+": missing resource "+resource)
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("module READMEs must mention capability/resource names used by module descriptors or scopes:\n%s", strings.Join(violations, "\n"))
	}
}

func TestModuleDescriptorsDeclareCapabilityUsage(t *testing.T) {
	capabilities := capabilityConstValues(t)
	requiredUseRe := regexp.MustCompile(`bkmodule\.RequireCapability\s*\[[^\n]*\]\s*\([^,]+,\s*bkmodule\.(Capability[A-Za-z0-9_]+)`)
	optionalUseRe := regexp.MustCompile(`bkmodule\.Capability\s*\[[^\n]*\]\s*\([^,]+,\s*bkmodule\.(Capability[A-Za-z0-9_]+)`)
	providedUseRe := regexp.MustCompile(`Capabilities\(\)\.Provide\s*\([^,]+,\s*bkmodule\.(Capability[A-Za-z0-9_]+)`)
	declRe := regexp.MustCompile(`bkmodule\.(Required|Optional|Provided)Capability(?:Of)?(?:\[[^\n]*\])?\s*\(\s*bkmodule\.(Capability[A-Za-z0-9_]+)`)

	entries, err := os.ReadDir("modules")
	if err != nil {
		t.Fatalf("read modules: %v", err)
	}
	var violations []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join("modules", entry.Name())
		src := moduleSourceForGuard(t, dir)
		declared := map[string]map[string]bool{}
		for _, match := range declRe.FindAllStringSubmatch(src, -1) {
			direction := strings.ToLower(match[1])
			constName := match[2]
			if declared[constName] == nil {
				declared[constName] = map[string]bool{}
			}
			declared[constName][direction] = true
		}
		check := func(matches [][]string, direction string) {
			for _, match := range matches {
				constName := match[1]
				if declared[constName][direction] {
					continue
				}
				name := capabilities[constName]
				if name == "" {
					name = constName
				}
				violations = append(violations, fmt.Sprintf("%s: %s uses %s but descriptor lacks %s capability", dir, direction, name, direction))
			}
		}
		check(requiredUseRe.FindAllStringSubmatch(src, -1), "required")
		check(optionalUseRe.FindAllStringSubmatch(src, -1), "optional")
		check(providedUseRe.FindAllStringSubmatch(src, -1), "provided")
	}
	if len(violations) > 0 {
		t.Fatalf("module descriptors must declare required, optional, and provided capabilities used during mount:\n%s", strings.Join(violations, "\n"))
	}
}

func TestRuntimeHookCapabilitiesUseScopedLeases(t *testing.T) {
	forbidden := []string{
		"CapabilitySetScheduleHandler",
		"CapabilitySetAuditStore",
		"CapabilitySetAuditVerbosity",
		"CapabilitySetTraceStore",
		"CapabilitySetPluginChecker",
		"CapabilitySetPluginRestarter",
		"CapabilitySetToolEvaluator",
		"brainkit.core.set_schedule_handler",
		"brainkit.core.set_audit_store",
		"brainkit.core.set_audit_verbosity",
		"brainkit.core.set_trace_store",
		"brainkit.core.set_plugin_checker",
		"brainkit.core.set_plugin_restarter",
		"brainkit.core.set_tool_evaluator",
		"SetScheduleHandler",
		"SetAuditStore",
		"SetAuditVerbosity",
		"SetTraceStore",
		"SetPluginChecker",
		"SetPluginRestarter",
		"SetToolEvaluator",
	}

	var violations []string
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "vendor", "vendor_typescript":
				return fs.SkipDir
			}
			if strings.HasPrefix(d.Name(), "vendor_") {
				return fs.SkipDir
			}
			return nil
		}
		if path == "CHANGELOG.md" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if !strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, ".md") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(body)
		for _, marker := range forbidden {
			if strings.Contains(text, marker) {
				violations = append(violations, path+": contains "+marker)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan runtime hook capabilities: %v", err)
	}
	if len(violations) > 0 {
		t.Fatalf("runtime hook capabilities must use scoped lease capabilities, not setter-style hooks:\n%s", strings.Join(violations, "\n"))
	}
}

func TestModulesUseDescriptorRequiresForDependencies(t *testing.T) {
	var violations []string
	for _, file := range goFilesUnder(t, "modules") {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		if strings.Contains(string(body), "Dependencies()") {
			violations = append(violations, file)
		}
	}
	if len(violations) > 0 {
		t.Fatalf("module dependencies must be declared through module.Descriptor.Requires, not Dependencies methods:\n%s", strings.Join(violations, "\n"))
	}
}

func TestModuleHostDoesNotExposeRawRuntimeCallerOrStore(t *testing.T) {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, filepath.Join("module", "module.go"), nil, 0)
	if err != nil {
		t.Fatalf("parse module/module.go: %v", err)
	}
	var hostFound bool
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != "Host" {
				continue
			}
			hostFound = true
			iface, ok := ts.Type.(*ast.InterfaceType)
			if !ok {
				t.Fatalf("module.Host must be an interface")
			}
			for _, field := range iface.Methods.List {
				for _, name := range field.Names {
					switch name.Name {
					case "Runtime", "Caller", "Store":
						t.Fatalf("module.Host must not expose %s; modules should use Messages, Commands, or named capabilities", name.Name)
					}
				}
			}
		}
	}
	if !hostFound {
		t.Fatalf("module.Host interface not found")
	}
}

func TestEngineCommandDomainsStayInModules(t *testing.T) {
	matches, err := filepath.Glob(filepath.Join("internal", "engine", "handlers_*.go"))
	if err != nil {
		t.Fatalf("glob internal/engine/handlers_*.go: %v", err)
	}
	if len(matches) > 0 {
		t.Fatalf("engine command-domain files must stay module-owned, not internal/engine:\n%s", strings.Join(matches, "\n"))
	}
}

func TestProviderRegistryImplementationStaysModuleOwned(t *testing.T) {
	internalDir := filepath.Join("internal", "providers")
	if _, err := os.Stat(internalDir); err == nil {
		t.Fatalf("%s must not exist; provider/storage/vector registry implementation belongs in modulehost/providerhost/providerreg", internalDir)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", internalDir, err)
	}

	var violations []string
	for _, file := range goFilesUnder(t, ".") {
		parsed := parseImportsOnly(t, file)
		for _, imp := range parsed.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if path == "github.com/brainlet/brainkit/internal/providers" || strings.HasPrefix(path, "github.com/brainlet/brainkit/internal/providers/") {
				violations = append(violations, file)
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("provider registry imports must point at modulehost/providerhost/providerreg, not internal/providers:\n%s", strings.Join(violations, "\n"))
	}
}

func TestProviderHostImplementationStaysModuleOwned(t *testing.T) {
	if _, err := os.Stat(filepath.Join("modulehost", "providerhost", "host.go")); err != nil {
		t.Fatalf("provider host implementation must live in modulehost/providerhost: %v", err)
	}

	for _, file := range []string{
		filepath.Join("internal", "engine", "kernel_providers.go"),
		filepath.Join("internal", "engine", "kernel_probing.go"),
		filepath.Join("internal", "engine", "provider_refresh.go"),
	} {
		if _, err := os.Stat(file); err == nil {
			t.Fatalf("%s must not exist; provider bootstrap/probe/refresh implementation belongs in modulehost/providerhost", file)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", file, err)
		}
	}

	fields := structFieldNames(t, filepath.Join("internal", "engine", "kernel.go"), "Kernel")
	if fields["providers"] {
		t.Fatalf("Kernel must not own direct provider registry field; use providerHost")
	}
	if !fields["providerHost"] {
		t.Fatalf("Kernel must keep provider lifecycle behind providerHost")
	}

	for _, file := range goFilesUnder(t, filepath.Join("internal", "engine")) {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		for _, forbidden := range []string{
			"UpdateProbeResult",
			"__probe_vectorstore.ts",
			"__probe_storage.ts",
			"__brainkit.secrets.refreshProvider",
			"__brainkit.registry.clearCache",
			"OPENAI_API_KEY",
		} {
			if strings.Contains(string(src), forbidden) {
				t.Fatalf("%s contains provider host implementation marker %q; keep it in modulehost/providerhost", file, forbidden)
			}
		}
	}

	moduleMount, err := os.ReadFile("module_mount.go")
	if err != nil {
		t.Fatalf("read module_mount.go: %v", err)
	}
	for _, forbidden := range []string{
		"ProviderSecretRefresherFunc",
		"RegistryRuntimeCacheInvalidatorFunc",
	} {
		if strings.Contains(string(moduleMount), forbidden) {
			t.Fatalf("module_mount.go must expose provider-host owned capability objects, not %q adapters", forbidden)
		}
	}
}

func TestStorageHostImplementationStaysModuleOwned(t *testing.T) {
	if _, err := os.Stat(filepath.Join("modulehost", "storagehost", "manager.go")); err != nil {
		t.Fatalf("storage bridge host implementation must live in modulehost/storagehost: %v", err)
	}

	for _, file := range []string{
		filepath.Join("internal", "engine", "storage_bridges.go"),
		filepath.Join("internal", "engine", "storage_config.go"),
	} {
		if _, err := os.Stat(file); err == nil {
			t.Fatalf("%s must not exist; storage bridge host implementation belongs in modulehost/storagehost", file)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", file, err)
		}
	}

	for _, file := range []string{
		filepath.Join("storagebridges", "storagebridges.go"),
		filepath.Join("storagebridges", "sqlite", "sqlite.go"),
	} {
		parsed := parseImportsOnly(t, file)
		for _, imp := range parsed.Imports {
			if strings.Trim(imp.Path.Value, `"`) == "github.com/brainlet/brainkit/internal/engine" {
				t.Fatalf("%s must register with modulehost/storagehost, not internal/engine", file)
			}
		}
	}
}

func TestTransportHostImplementationStaysModuleOwned(t *testing.T) {
	if _, err := os.Stat(filepath.Join("modulehost", "transporthost", "host.go")); err != nil {
		t.Fatalf("transport host implementation must live in modulehost/transporthost: %v", err)
	}

	fields := structFieldNames(t, filepath.Join("internal", "engine", "kernel.go"), "Kernel")
	for _, forbidden := range []string{
		"transport",
		"router",
		"remote",
		"host",
		"ownsTransport",
		"caller",
		"busMetrics",
	} {
		if fields[forbidden] {
			t.Fatalf("Kernel must not own direct transport field %q; use transportHost", forbidden)
		}
	}
	if !fields["transportHost"] {
		t.Fatalf("Kernel must keep transport lifecycle behind transportHost")
	}

	parsed := parseImportsOnly(t, filepath.Join("internal", "engine", "kernel_init.go"))
	for _, imp := range parsed.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if path == "github.com/ThreeDotsLabs/watermill" || path == "github.com/ThreeDotsLabs/watermill/message" {
			t.Fatalf("kernel_init.go must not construct Watermill router/caller directly; use modulehost/transporthost")
		}
	}
}

func TestJSRuntimeResourceHostImplementationStaysModuleOwned(t *testing.T) {
	if _, err := os.Stat(filepath.Join("modulehost", "resourcehost", "registry.go")); err != nil {
		t.Fatalf("resource host implementation must live in modulehost/resourcehost: %v", err)
	}

	for _, file := range []string{
		filepath.Join("internal", "engine", "resource_registry.go"),
		filepath.Join("internal", "engine", "resource_registry_test.go"),
	} {
		if _, err := os.Stat(file); err == nil {
			t.Fatalf("%s must not exist; JS-runtime resource tracking belongs in modulehost/resourcehost", file)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", file, err)
		}
	}

	for _, file := range goFilesUnder(t, filepath.Join("internal", "jsruntime")) {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		for _, forbidden := range []string{
			"engine.ResourceEntry",
			"engine.ResourceRegistry",
			"engine.NewResourceRegistry",
		} {
			if strings.Contains(string(src), forbidden) {
				t.Fatalf("%s contains engine-owned resource registry marker %q; use modulehost/resourcehost", file, forbidden)
			}
		}
	}
}

func TestRuntimePersistenceHostImplementationStaysModuleOwned(t *testing.T) {
	if _, err := os.Stat(filepath.Join("modulehost", "runtimehost", "manager.go")); err != nil {
		t.Fatalf("runtime persistence host implementation must live in modulehost/runtimehost: %v", err)
	}

	if _, err := os.Stat(filepath.Join("internal", "engine", "kernel_propagation.go")); err == nil {
		t.Fatalf("internal/engine/kernel_propagation.go must not exist; runtime persistence and propagation belong in modulehost/runtimehost")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat internal/engine/kernel_propagation.go: %v", err)
	}

	for _, file := range goFilesUnder(t, filepath.Join("internal", "engine")) {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		for _, forbidden := range []string{
			"LoadDeployments",
			"TopicKitDeployed",
			"TopicKitTeardowned",
			"RestartActiveWorkflows",
			"restartWorkflows",
		} {
			if strings.Contains(string(src), forbidden) {
				t.Fatalf("%s contains runtime persistence or workflow-recovery marker %q; persistence belongs to modulehost/runtimehost and workflow recovery belongs to modules/workflow", file, forbidden)
			}
		}
	}
}

func TestRuntimePersistenceHostDoesNotOwnWorkflowRecovery(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("modulehost", "runtimehost", "manager.go"))
	if err != nil {
		t.Fatalf("read runtimehost manager: %v", err)
	}
	for _, forbidden := range []string{
		"RestartActiveWorkflows",
		"restartWorkflows",
		"restartActive",
		"workflow",
	} {
		if strings.Contains(string(src), forbidden) {
			t.Fatalf("modulehost/runtimehost/manager.go contains workflow recovery marker %q; workflow recovery belongs to modules/workflow", forbidden)
		}
	}
}

func TestJSRuntimeRegistryMutationsUseStorageHostForLifecycleResources(t *testing.T) {
	var violations []string
	for _, file := range goFilesUnder(t, filepath.Join("internal", "jsruntime")) {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		for _, forbidden := range []string{
			"RegisterVectorStore(",
			"UnregisterVectorStore(",
			"RegisterStorage(",
			"UnregisterStorage(",
		} {
			if strings.Contains(string(src), forbidden) {
				violations = append(violations, file+": "+forbidden)
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("JS runtime storage/vector registry mutations must go through runtimecap.StorageHost so bridge resources have one lifecycle owner:\n%s", strings.Join(violations, "\n"))
	}
}

func TestRegistryModuleMutationsUseCoreMutationCapability(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("modules", "registry", "module.go"))
	if err != nil {
		t.Fatalf("read registry module: %v", err)
	}
	for _, forbidden := range []string{
		"RegisterAIProvider(",
		"UnregisterAIProvider(",
		"RegisterStorage(",
		"UnregisterStorage(",
		"RegisterVectorStore(",
		"UnregisterVectorStore(",
		"CapabilityStorageManager",
		"CapabilityRegistryRuntimeCache",
		"InvalidateRegistryRuntimeCache",
	} {
		if strings.Contains(string(src), forbidden) {
			t.Fatalf("modules/registry should route live mutations through brainkit.core.registry_mutation, found %q", forbidden)
		}
	}
}

func TestModulesDoNotConsumeLowLevelRegistryMutationCapabilities(t *testing.T) {
	var violations []string
	for _, file := range goFilesUnder(t, "modules") {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		for _, forbidden := range []string{
			"brainkit.core.storage_manager",
			"brainkit.core.registry_runtime_cache",
			"CapabilityStorageManager",
			"CapabilityRegistryRuntimeCache",
			"RegistryRuntimeCacheInvalidator",
			"InvalidateRegistryRuntimeCache",
		} {
			if strings.Contains(string(src), forbidden) {
				violations = append(violations, file+": "+forbidden)
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("normal modules must use provider_registry for reads and registry_mutation for live mutations, not low-level mutation helpers:\n%s", strings.Join(violations, "\n"))
	}
}

func TestModuleCapabilitySurfaceDoesNotExposeLowLevelRegistryMutationHelpers(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("module", "capabilities.go"))
	if err != nil {
		t.Fatalf("read module capabilities: %v", err)
	}
	for _, forbidden := range []string{
		"brainkit.core.storage_manager",
		"brainkit.core.registry_runtime_cache",
		"CapabilityStorageManager",
		"CapabilityRegistryRuntimeCache",
		"RegistryRuntimeCacheInvalidator",
	} {
		if strings.Contains(string(src), forbidden) {
			t.Fatalf("module capability surface should not expose low-level registry mutation helper %q", forbidden)
		}
	}
}

func TestJSRuntimeCapabilityContractsStayModuleOwned(t *testing.T) {
	if _, err := os.Stat(filepath.Join("modulecap", "runtime", "runtime.go")); err != nil {
		t.Fatalf("JS runtime capability contracts must live in modulecap/runtime: %v", err)
	}
	if _, err := os.Stat(filepath.Join("modulecap", "harness", "runtime.go")); err != nil {
		t.Fatalf("harness runtime capability contract must live in modulecap/harness: %v", err)
	}

	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, filepath.Join("modulecap", "runtime", "runtime.go"), nil, 0)
	if err != nil {
		t.Fatalf("parse modulecap/runtime/runtime.go: %v", err)
	}
	var hostFound bool
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != "Host" {
				continue
			}
			hostFound = true
			iface, ok := ts.Type.(*ast.InterfaceType)
			if !ok {
				t.Fatalf("runtimecap.Host must be an interface")
			}
			for _, field := range iface.Methods.List {
				if len(field.Names) > 0 {
					t.Fatalf("runtimecap.Host must compose smaller interfaces, not declare method %s directly", field.Names[0].Name)
				}
			}
		}
	}
	if !hostFound {
		t.Fatalf("runtimecap.Host interface not found")
	}

	enableParsed, err := parser.ParseFile(fset, filepath.Join("internal", "jsruntime", "enable.go"), nil, 0)
	if err != nil {
		t.Fatalf("parse internal/jsruntime/enable.go: %v", err)
	}
	enableBody, err := os.ReadFile(filepath.Join("internal", "jsruntime", "enable.go"))
	if err != nil {
		t.Fatalf("read internal/jsruntime/enable.go: %v", err)
	}
	enableText := string(enableBody)
	if !strings.Contains(enableText, "func Enable(ctx context.Context, host runtimecap.EnableHost) error") {
		t.Fatalf("internal/jsruntime.Enable must accept runtimecap.EnableHost, not the broader runtimecap.Host")
	}
	if strings.Contains(enableText, "func Enable(ctx context.Context, host runtimecap.Host) error") {
		t.Fatalf("internal/jsruntime.Enable must not require runtimecap.Host access before runtime activation")
	}
	for _, decl := range enableParsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != "Runtime" {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			for _, field := range st.Fields.List {
				for _, name := range field.Names {
					if name.Name == "host" {
						t.Fatalf("internal/jsruntime.Runtime must not store the composed runtimecap.Host; store grouped subinterfaces instead")
					}
				}
			}
		}
	}

	var violations []string
	for _, dir := range []string{
		filepath.Join("modules", "packages"),
		filepath.Join("modules", "testing"),
	} {
		for _, file := range goFilesUnder(t, dir) {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}
			parsed := parseImportsOnly(t, file)
			for _, imp := range parsed.Imports {
				if strings.Trim(imp.Path.Value, `"`) == "github.com/brainlet/brainkit/internal/engine" {
					violations = append(violations, file)
				}
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("JS-dependent feature modules must use modulecap/runtime, not internal/engine capability contracts:\n%s", strings.Join(violations, "\n"))
	}
}

func TestHarnessRuntimeCapabilityIsTypedAtModuleBoundary(t *testing.T) {
	requiredImports := map[string]string{
		filepath.Join("modules", "harness", "module.go"):   "github.com/brainlet/brainkit/modulecap/harness",
		filepath.Join("modules", "jsruntime", "module.go"): "github.com/brainlet/brainkit/modulecap/harness",
	}
	for file, want := range requiredImports {
		parsed := parseImportsOnly(t, file)
		var found bool
		for _, imp := range parsed.Imports {
			if strings.Trim(imp.Path.Value, `"`) == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%s must use typed harness capability %s", file, want)
		}
	}

	forbidden := []*regexp.Regexp{
		regexp.MustCompile(`RequireCapability\s*\[\s*func\(\)\s+any\s*\]\s*\([^)]*CapabilityHarnessRuntime`),
		regexp.MustCompile(`Capability\s*\[\s*func\(\)\s+any\s*\]\s*\([^)]*CapabilityHarnessRuntime`),
		regexp.MustCompile(`RequiredCapabilityOf\s*\[\s*func\(\)\s+any\s*\]\s*\([^)]*CapabilityHarnessRuntime`),
		regexp.MustCompile(`ProvidedCapabilityOf\s*\[\s*func\(\)\s+any\s*\]\s*\([^)]*CapabilityHarnessRuntime`),
		regexp.MustCompile(`CapabilityHarnessRuntime[^,\n]*,\s*func\(\)\s+any`),
	}

	var violations []string
	for _, file := range goFilesUnder(t, ".") {
		if strings.HasPrefix(file, "vendor_typescript"+string(filepath.Separator)) {
			continue
		}
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		text := string(body)
		for _, pattern := range forbidden {
			if pattern.MatchString(text) {
				violations = append(violations, file+": "+pattern.String())
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("brainkit.core.harness_runtime must be typed at module boundaries; keep any only inside the opaque engine/runtime attachment bridge:\n%s", strings.Join(violations, "\n"))
	}

	for _, file := range append(goFilesUnder(t, filepath.Join("modules", "harness")), filepath.Join("modulecap", "harness", "runtime.go")) {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed := parseImportsOnly(t, file)
		for _, imp := range parsed.Imports {
			if strings.Trim(imp.Path.Value, `"`) == "github.com/buke/quickjs-go" {
				t.Fatalf("%s must not import quickjs-go; QuickJS details belong inside internal/jsruntime", file)
			}
		}
	}
}

func TestPluginCapabilityContractsStayModuleOwned(t *testing.T) {
	if _, err := os.Stat(filepath.Join("modulecap", "plugin", "plugin.go")); err != nil {
		t.Fatalf("plugin capability contracts must live in modulecap/plugin: %v", err)
	}

	var violations []string
	for _, file := range goFilesUnder(t, filepath.Join("modules", "plugins")) {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed := parseImportsOnly(t, file)
		for _, imp := range parsed.Imports {
			if strings.Trim(imp.Path.Value, `"`) == "github.com/brainlet/brainkit/internal/engine" {
				violations = append(violations, file)
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("plugins module must use modulecap/plugin, not internal/engine capability contracts:\n%s", strings.Join(violations, "\n"))
	}

	forbidden := []*regexp.Regexp{
		regexp.MustCompile(`CapabilityPluginRestarter[^,\n]*,\s*func\(\)\s+any`),
		regexp.MustCompile(`Capability\s*\[\s*func\(\)\s+any\s*\]\s*\([^)]*CapabilityPluginRestarter`),
	}
	violations = nil
	for _, file := range []string{
		"module_mount.go",
		filepath.Join("modules", "secrets", "module.go"),
	} {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		text := string(body)
		for _, pattern := range forbidden {
			if pattern.MatchString(text) {
				violations = append(violations, file+": "+pattern.String())
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("plugin restarter capability must use modulecap/plugin.Restarter, not func() any:\n%s", strings.Join(violations, "\n"))
	}
}

func TestMetricsSnapshotCapabilityIsTyped(t *testing.T) {
	forbidden := []*regexp.Regexp{
		regexp.MustCompile(`CapabilityMetricsSnapshot[^,\n]*,\s*func\(\)\s+any`),
		regexp.MustCompile(`RequireCapability\s*\[\s*func\(\)\s+any\s*\]\s*\([^)]*CapabilityMetricsSnapshot`),
		regexp.MustCompile(`RequiredCapabilityOf\s*\[\s*func\(\)\s+any\s*\]\s*\([^)]*CapabilityMetricsSnapshot`),
	}
	var violations []string
	for _, file := range []string{
		"module_mount.go",
		filepath.Join("modules", "metrics", "module.go"),
	} {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		text := string(body)
		for _, pattern := range forbidden {
			if pattern.MatchString(text) {
				violations = append(violations, file+": "+pattern.String())
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("metrics snapshot capability must use the typed KernelMetrics callback, not func() any:\n%s", strings.Join(violations, "\n"))
	}
}

func TestProbeAllCapabilityIsContextAware(t *testing.T) {
	forbidden := []*regexp.Regexp{
		regexp.MustCompile(`Capability\s*\[\s*func\(\)\s*\]\s*\([^)]*CapabilityProbeAll`),
		regexp.MustCompile(`RequireCapability\s*\[\s*func\(\)\s*\]\s*\([^)]*CapabilityProbeAll`),
		regexp.MustCompile(`RequiredCapabilityOf\s*\[\s*func\(\)\s*\]\s*\([^)]*CapabilityProbeAll`),
		regexp.MustCompile(`OptionalCapabilityOf\s*\[\s*func\(\)\s*\]\s*\([^)]*CapabilityProbeAll`),
		regexp.MustCompile(`CapabilityProbeAll[^,\n]*,\s*func\(\)`),
	}
	var violations []string
	for _, file := range []string{
		"module_mount.go",
		filepath.Join("module", "capabilities.go"),
		filepath.Join("modules", "probes", "module.go"),
	} {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		text := string(body)
		for _, pattern := range forbidden {
			if pattern.MatchString(text) {
				violations = append(violations, file+": "+pattern.String())
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("probe_all capability must use the typed context-aware ProbeRunner contract, not func():\n%s", strings.Join(violations, "\n"))
	}
}

func TestModuleMessagePackagesOwnGeneratedWrappers(t *testing.T) {
	messageFiles := messageFilesUnder(t, "modules")
	for _, messageFile := range messageFiles {
		dir := filepath.Dir(messageFile)
		gen := filepath.Join(dir, "typed_gen.go")
		if _, err := os.Stat(gen); err != nil {
			t.Fatalf("%s declares BusTopic messages but has no generated wrapper file %s", messageFile, gen)
		}

		msgPkg := packageName(t, messageFile)
		genPkg := packageName(t, gen)
		if genPkg != msgPkg {
			t.Fatalf("%s package = %q, want message package %q", gen, genPkg, msgPkg)
		}

		src, err := os.ReadFile(gen)
		if err != nil {
			t.Fatalf("read %s: %v", gen, err)
		}
		text := string(src)
		if strings.Contains(text, "WithCaller(caller sdk.RequestCaller") {
			t.Fatalf("%s must use module.RequestCaller for generated module-owned WithCaller helpers", gen)
		}
		if regexp.MustCompile(`func\s+Publish[A-Z][A-Za-z0-9_]*\s*\(`).MatchString(text) {
			t.Fatalf("%s must not generate low-level PublishXxx command helpers; use protocol.Publish directly for protocol tests", gen)
		}
		if regexp.MustCompile(`func\s+Subscribe[A-Z][A-Za-z0-9_]*Resp\s*\(`).MatchString(text) {
			t.Fatalf("%s must not generate low-level SubscribeXxxResp helpers; normal request/reply uses CallXxx", gen)
		}

		parsed := parseImportsOnly(t, gen)
		hasSDKImport := false
		hasModuleImport := false
		for _, imp := range parsed.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if path == "github.com/brainlet/brainkit" {
				t.Fatalf("%s imports root brainkit; generated wrappers must depend only on sdk/module", gen)
			}
			if path == "github.com/brainlet/brainkit/sdk" {
				hasSDKImport = true
			}
			if path == "github.com/brainlet/brainkit/module" {
				hasModuleImport = true
			}
		}
		if !hasSDKImport {
			t.Fatalf("%s must import github.com/brainlet/brainkit/sdk for generated runtime helpers", gen)
		}
		if strings.Contains(text, "WithCaller(caller ") && !hasModuleImport {
			t.Fatalf("%s must import github.com/brainlet/brainkit/module for generated module-owned WithCaller helpers", gen)
		}
	}
}

func TestSystemMessagesOwnSystemEventWrappers(t *testing.T) {
	gen := filepath.Join("sdk", "systemmsg", "typed_gen.go")
	parsed := parseImportsOnly(t, gen)
	if parsed.Name.Name != "systemmsg" {
		t.Fatalf("%s package = %q, want systemmsg", gen, parsed.Name.Name)
	}

	hasSDKImport := false
	for _, imp := range parsed.Imports {
		if strings.Trim(imp.Path.Value, `"`) == "github.com/brainlet/brainkit/sdk" {
			hasSDKImport = true
		}
	}
	if !hasSDKImport {
		t.Fatalf("%s must import github.com/brainlet/brainkit/sdk for generated runtime helpers", gen)
	}

	fset := token.NewFileSet()
	full, err := parser.ParseFile(fset, gen, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", gen, err)
	}
	var wrappers []string
	for _, decl := range full.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			wrappers = append(wrappers, fn.Name.Name)
		}
	}
	for _, want := range []string{
		"EmitKitDeployed",
		"SubscribeKitDeployed",
		"EmitKitTeardowned",
		"SubscribeKitTeardowned",
		"EmitHandlerFailed",
		"SubscribeHandlerFailed",
		"EmitHandlerExhausted",
		"SubscribeHandlerExhausted",
	} {
		if !contains(wrappers, want) {
			t.Fatalf("%s missing generated wrapper %s; got %s", gen, want, strings.Join(wrappers, ", "))
		}
	}
}

func receiverName(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.StarExpr:
		if ident, ok := v.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

func goFilesUnder(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "vendor":
				return fs.SkipDir
			}
			if strings.HasPrefix(d.Name(), "vendor_") {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return files
}

func messageFilesUnder(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	for _, file := range goFilesUnder(t, root) {
		if strings.HasSuffix(file, "_messages.go") {
			files = append(files, file)
		}
	}
	return files
}

func moduleSourceForGuard(t *testing.T, dir string) string {
	t.Helper()
	var source strings.Builder
	for _, file := range goFilesUnder(t, dir) {
		if strings.HasSuffix(file, "_test.go") || strings.HasSuffix(file, "typed_gen.go") {
			continue
		}
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		source.Write(body)
		source.WriteByte('\n')
	}
	return source.String()
}

func uniqueSortedStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, value := range in {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func capabilityConstValues(t *testing.T) map[string]string {
	t.Helper()
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, filepath.Join("module", "capabilities.go"), nil, 0)
	if err != nil {
		t.Fatalf("parse module/capabilities.go: %v", err)
	}
	values := map[string]string{}
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range valueSpec.Names {
				if !strings.HasPrefix(name.Name, "Capability") || i >= len(valueSpec.Values) {
					continue
				}
				lit, ok := valueSpec.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				value, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatalf("unquote capability %s: %v", name.Name, err)
				}
				values[name.Name] = value
			}
		}
	}
	return values
}

func parseImportsOnly(t *testing.T, file string) *ast.File {
	t.Helper()
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse imports %s: %v", file, err)
	}
	return parsed
}

func packageName(t *testing.T, file string) string {
	t.Helper()
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file, nil, parser.PackageClauseOnly)
	if err != nil {
		t.Fatalf("parse package %s: %v", file, err)
	}
	return parsed.Name.Name
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func structFieldNames(t *testing.T, file string, typeName string) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != typeName {
				continue
			}
			st, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				t.Fatalf("%s is not a struct", typeName)
			}
			fields := map[string]bool{}
			for _, field := range st.Fields.List {
				for _, name := range field.Names {
					fields[name.Name] = true
				}
			}
			return fields
		}
	}
	t.Fatalf("type %s not found in %s", typeName, file)
	return nil
}
