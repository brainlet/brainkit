package brainkit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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
		{"github.com/brainlet/brainkit/modules/jsruntime", "root must not auto-register the concrete JS runtime module"},
		{"github.com/brainlet/brainkit/storagebridges", "storage bridge registration is optional runtime wiring"},
		{"github.com/brainlet/brainkit/internal/libsql", "embedded libsql server is optional storage/vector infrastructure"},
		{"github.com/brainlet/brainkit/internal/transport/backends", "concrete transport backends belong behind transports or tests"},
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
		t.Fatalf("%s must not exist; provider/storage/vector registry implementation belongs in modules/registry/providerreg", internalDir)
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
		t.Fatalf("provider registry imports must point at modules/registry/providerreg, not internal/providers:\n%s", strings.Join(violations, "\n"))
	}
}

func TestStorageHostImplementationStaysModuleOwned(t *testing.T) {
	for _, file := range []string{
		filepath.Join("internal", "engine", "storage_bridges.go"),
		filepath.Join("internal", "engine", "storage_config.go"),
	} {
		if _, err := os.Stat(file); err == nil {
			t.Fatalf("%s must not exist; storage bridge host implementation belongs in modules/registry/storagehost", file)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", file, err)
		}
	}

	parsed := parseImportsOnly(t, filepath.Join("storagebridges", "storagebridges.go"))
	for _, imp := range parsed.Imports {
		if strings.Trim(imp.Path.Value, `"`) == "github.com/brainlet/brainkit/internal/engine" {
			t.Fatalf("storagebridges must register with modules/registry/storagehost, not internal/engine")
		}
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

		parsed := parseImportsOnly(t, gen)
		hasSDKImport := false
		for _, imp := range parsed.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if path == "github.com/brainlet/brainkit" {
				t.Fatalf("%s imports root brainkit; generated wrappers must depend only on sdk", gen)
			}
			if path == "github.com/brainlet/brainkit/sdk" {
				hasSDKImport = true
			}
		}
		if !hasSDKImport {
			t.Fatalf("%s must import github.com/brainlet/brainkit/sdk for generated runtime helpers", gen)
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
