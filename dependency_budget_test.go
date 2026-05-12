package brainkit

import (
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestDependencyProfileBudgetsStayBounded(t *testing.T) {
	budgets := []struct {
		pkg string
		max int
	}{
		{".", 34},
		{"./server", 35},
		{"./server/configfile", 36},
		{"./server/standard", 254},
		{"./server/standard/runtime", 112},
		{"./server/standard/artifactruntime", 70},
		{"./server/standard/packages", 150},
		{"./presets/standard/core", 36},
		{"./presets/standard/runtime", 112},
		{"./presets/standard/artifactruntime", 70},
		{"./presets/standard/packages", 150},
		{"./modules/jsruntime", 109},
		{"./modules/jsruntime/artifact", 67},
		{"./modules/packages", 27},
		{"./modules/packages/bundlers/esbuild", 55},
		{"./modules/packages/source", 1},
		{"./modules/packages/client", 8},
		{"./modules/packages/scaffold", 2},
	}

	var drift []string
	for _, budget := range budgets {
		got := countNonStdlibDeps(t, budget.pkg)
		if got > budget.max {
			drift = append(drift, budget.pkg+": "+strconv.Itoa(got)+" > "+strconv.Itoa(budget.max))
		}
	}
	if len(drift) > 0 {
		sort.Strings(drift)
		t.Fatalf("dependency profile budgets grew; update the boundary intentionally or lower the graph:\n%s", strings.Join(drift, "\n"))
	}
}

func countNonStdlibDeps(t *testing.T, pkg string) int {
	t.Helper()
	out, err := exec.Command("go", "list", "-deps", "-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", pkg).CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps %s: %v\n%s", pkg, err, out)
	}
	seen := map[string]struct{}{}
	for _, line := range strings.Split(string(out), "\n") {
		dep := strings.TrimSpace(line)
		if dep == "" {
			continue
		}
		seen[dep] = struct{}{}
	}
	return len(seen)
}
