// Ported from: node-ignore/test/git-check-ignore.test.js
package ignore

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestGitCheckIgnoreParity(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	if runtimeGOOS := strings.ToLower(os.Getenv("GOOS")); runtimeGOOS == "windows" {
		t.Skip("git parity tests are skipped on windows")
	}

	for _, tc := range loadUpstreamCases(t) {
		tc := tc

		if tc.SkipTestFixture {
			continue
		}
		if !hasAnyNonEmptyPath(tc.Paths) {
			continue
		}
		if !allPathsNotGitBuiltin(tc.Expected) {
			continue
		}

		t.Run(tc.Description, func(t *testing.T) {
			got := getNativeGitIgnoreResults(t, tc.Patterns, tc.Paths)
			assertSameStrings(t, got, tc.Expected)
		})
	}
}

func hasAnyNonEmptyPath(paths []string) bool {
	for _, path := range paths {
		if path != "" {
			return true
		}
	}
	return false
}

func allPathsNotGitBuiltin(paths []string) bool {
	for _, path := range paths {
		if strings.HasPrefix(path, ".git/") {
			return false
		}
	}
	return true
}

func getNativeGitIgnoreResults(t *testing.T, rules fixturePatterns, paths []string) []string {
	t.Helper()

	root := t.TempDir()
	var builder strings.Builder
	for i, rule := range rules.Values {
		if i > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(rule.stringValue())
	}

	touchFile(t, root, ".gitignore", builder.String())
	for i, path := range paths {
		if path == ".gitignore" || containsInOthers(path, i, paths) {
			continue
		}
		touchFile(t, root, path, "")
	}

	runCommand(t, root, "git", "init")
	runCommand(t, root, "git", "add", "-A")

	var result []string
	for _, path := range paths {
		out := runCommandAllowFailure(t, root, "git", "check-ignore", "--no-index", path)
		normalized := strings.TrimSpace(strings.ReplaceAll(out, `\\`, `\`))
		normalized = strings.Trim(normalized, `"`)
		if normalized != path {
			result = append(result, path)
		}
	}

	sort.Strings(result)
	return result
}

func touchFile(t *testing.T, root, file, content string) {
	t.Helper()

	dirs := strings.Split(file, "/")
	basename := dirs[len(dirs)-1]
	dir := strings.Join(dirs[:len(dirs)-1], "/")

	if dir != "" {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %q: %v", dir, err)
		}
	}

	if basename == "" {
		return
	}

	if err := os.WriteFile(filepath.Join(root, file), []byte(content), 0o644); err != nil {
		t.Fatalf("write %q: %v", file, err)
	}
}

func containsInOthers(path string, index int, paths []string) bool {
	path = strings.TrimSuffix(path, "/")
	for i, other := range paths {
		if i == index {
			continue
		}
		if other == path {
			return true
		}
		if strings.HasPrefix(other, path) && len(other) > len(path) && other[len(path)] == '/' {
			return true
		}
	}
	return false
}

func runCommand(t *testing.T, dir, name string, args ...string) string {
	t.Helper()

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, string(output))
	}
	return string(output)
}

func runCommandAllowFailure(t *testing.T, dir, name string, args ...string) string {
	t.Helper()

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, _ := cmd.CombinedOutput()
	return string(output)
}
