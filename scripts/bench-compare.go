//go:build ignore

// Command bench-compare enforces the benchmark regression gate.
//
// Usage (from the repo root):
//
//	go run scripts/bench-compare.go <baseline.json> <current_bench_output.txt>
//
// The current file is the raw stdout of `go test -bench ... -benchmem`;
// the baseline is the JSON shape captured by `make bench-save` and
// curated into `test/bench/baseline.json`. Fails with exit code 1 if
// any non-skipped benchmark's ns_per_op or allocs_per_op is more than
// `tolerance_percent` slower / larger than the baseline.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type benchMetric struct {
	NsPerOp     float64 `json:"ns_per_op"`
	AllocsPerOp float64 `json:"allocs_per_op"`
	BytesPerOp  float64 `json:"bytes_per_op"`
}

type baselineDoc struct {
	Description      string                 `json:"_description"`
	Machine          string                 `json:"_machine"`
	Benchtime        string                 `json:"_benchtime"`
	TolerancePercent float64                `json:"_tolerance_percent"`
	SkipInGate       []string               `json:"_skip_in_gate"`
	Benchmarks       map[string]benchMetric `json:"benchmarks"`
}

// Bench lines come in two shapes:
//
//   1. Single line:  `BenchmarkCall-10   3    93319 ns/op   15226 B/op   209 allocs/op`
//   2. Split by interleaved stderr/stdout noise (e.g. SES init
//      banners printed mid-benchmark): the name appears on one
//      line, the metrics on a later line starting with whitespace.
//
// benchName catches the name anywhere on a line (prefix form).
// statsOnly catches lines that begin with whitespace + iteration
// count and contain "ns/op" — when we see one, we pair it with
// the most recent benchmark name we've seen.
var (
	benchName = regexp.MustCompile(`^(Benchmark[A-Za-z0-9_/]+?)(-\d+)?(\s|$)`)
	statsOnly = regexp.MustCompile(
		`^\s*\d+\s+([\d\.]+)\s+ns/op(?:\s+([\d\.]+)\s+B/op)?(?:\s+([\d\.]+)\s+allocs/op)?`,
	)
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: bench-compare <baseline.json> <current.txt>")
		os.Exit(2)
	}
	base := loadBaseline(os.Args[1])
	cur := parseBench(os.Args[2])

	tol := base.TolerancePercent
	if tol <= 0 {
		tol = 10
	}
	skip := map[string]bool{}
	for _, s := range base.SkipInGate {
		skip[s] = true
	}

	var failed []string
	for name, b := range base.Benchmarks {
		if skip[name] {
			fmt.Printf("  %s: skipped (in _skip_in_gate)\n", name)
			continue
		}
		c, ok := cur[name]
		if !ok {
			failed = append(failed,
				fmt.Sprintf("%s: missing in current run", name))
			continue
		}
		if delta := percentDelta(b.NsPerOp, c.NsPerOp); delta > tol {
			failed = append(failed,
				fmt.Sprintf("%s: ns/op regressed %.1f%% (%.0f → %.0f, tol %.0f%%)",
					name, delta, b.NsPerOp, c.NsPerOp, tol))
		} else {
			fmt.Printf("  %s: ns/op %.0f → %.0f (%+.1f%%)\n",
				name, b.NsPerOp, c.NsPerOp, delta)
		}
		if b.AllocsPerOp > 0 {
			if delta := percentDelta(b.AllocsPerOp, c.AllocsPerOp); delta > tol {
				failed = append(failed,
					fmt.Sprintf("%s: allocs regressed %.1f%% (%.0f → %.0f, tol %.0f%%)",
						name, delta, b.AllocsPerOp, c.AllocsPerOp, tol))
			}
		}
	}

	if len(failed) > 0 {
		fmt.Fprintln(os.Stderr, "\nBENCH REGRESSION GATE FAILED:")
		for _, f := range failed {
			fmt.Fprintln(os.Stderr, "  - "+f)
		}
		os.Exit(1)
	}
	fmt.Println("\nbench-compare: all metrics within tolerance.")
}

func percentDelta(base, cur float64) float64 {
	if base <= 0 {
		return 0
	}
	return (cur - base) / base * 100
}

func loadBaseline(path string) baselineDoc {
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read baseline:", err)
		os.Exit(2)
	}
	var doc baselineDoc
	if err := json.Unmarshal(b, &doc); err != nil {
		fmt.Fprintln(os.Stderr, "parse baseline:", err)
		os.Exit(2)
	}
	return doc
}

func parseBench(path string) map[string]benchMetric {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open current:", err)
		os.Exit(2)
	}
	defer f.Close()

	out := map[string]benchMetric{}
	var pending string
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 64*1024), 1<<20)
	for s.Scan() {
		line := s.Text()
		// First, see if this line contains a benchmark name.
		if m := benchName.FindStringSubmatch(line); m != nil {
			// It might also carry the stats on the same line.
			if stats := statsOnly.FindStringSubmatch(trimAfterName(line, m[0])); stats != nil {
				out[m[1]] = benchMetric{
					NsPerOp:     atof(stats[1]),
					BytesPerOp:  atof(stats[2]),
					AllocsPerOp: atof(stats[3]),
				}
				pending = ""
				continue
			}
			pending = m[1]
			continue
		}
		// Stats-only continuation from a previous bench name.
		if pending != "" {
			if stats := statsOnly.FindStringSubmatch(line); stats != nil {
				out[pending] = benchMetric{
					NsPerOp:     atof(stats[1]),
					BytesPerOp:  atof(stats[2]),
					AllocsPerOp: atof(stats[3]),
				}
				pending = ""
			}
		}
	}
	if err := s.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "scan current:", err)
		os.Exit(2)
	}
	return out
}

// trimAfterName returns the tail of line following the benchmark
// name match, so we can apply the stats regex to only what could
// be the metrics half of a single-line bench row.
func trimAfterName(line, namePart string) string {
	_, tail, ok := strings.Cut(line, namePart)
	if !ok {
		return line
	}
	return tail
}

func atof(s string) float64 {
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
