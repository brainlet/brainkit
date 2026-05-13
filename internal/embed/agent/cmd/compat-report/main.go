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
	ID      string   `json:"id"`
	Kind    string   `json:"kind"`
	Aliases []string `json:"aliases"`
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

type esbuildMeta struct {
	Inputs  map[string]json.RawMessage `json:"inputs"`
	Outputs map[string]struct {
		Imports []struct {
			Path     string `json:"path"`
			External bool   `json:"external"`
		} `json:"imports"`
	} `json:"outputs"`
}

func main() {
	manifestPath := flag.String("manifest", "internal/embed/agent/bundle/compat/manifest.json", "compat manifest path")
	metaPath := flag.String("meta", "internal/embed/agent/bundle/meta.json", "esbuild metafile path")
	outPath := flag.String("out", "", "write report to this path instead of stdout")
	checkPath := flag.String("check", "", "compare generated report with a checked-in report path")
	flag.Parse()

	report, err := buildReport(*manifestPath, *metaPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "compat-report: %v\n", err)
		os.Exit(1)
	}

	if *checkPath != "" {
		if err := checkReport(report, *checkPath); err != nil {
			fmt.Fprintf(os.Stderr, "compat-report: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("compat report matches %s\n", *checkPath)
		return
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "compat-report: marshal: %v\n", err)
		os.Exit(1)
	}
	data = append(data, '\n')

	if *outPath != "" {
		if err := os.WriteFile(*outPath, data, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "compat-report: write %s: %v\n", *outPath, err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s\n", *outPath)
		return
	}
	_, _ = os.Stdout.Write(data)
}

func buildReport(manifestPath, metaPath string) (compatReport, error) {
	var manifest compatManifest
	if err := readJSON(manifestPath, &manifest); err != nil {
		return compatReport{}, err
	}
	var meta esbuildMeta
	if err := readJSON(metaPath, &meta); err != nil {
		return compatReport{}, err
	}

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
	}, nil
}

func checkReport(generated compatReport, path string) error {
	var checkedIn compatReport
	if err := readJSON(path, &checkedIn); err != nil {
		return err
	}
	if reflect.DeepEqual(checkedIn, generated) {
		return nil
	}
	checkedJSON, _ := json.MarshalIndent(checkedIn, "", "  ")
	generatedJSON, _ := json.MarshalIndent(generated, "", "  ")
	var buf bytes.Buffer
	buf.WriteString("checked-in report is stale\nchecked-in:\n")
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
