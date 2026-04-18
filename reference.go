package brainkit

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/brainlet/brainkit/sdk"
)

// referenceFS bundles the brainkit reference corpus that an LLM
// building on this Kit can condition on. Two sources:
//
//   - docs/llm/*.md   — curated prose/reference pages for Go API,
//     TypeScript runtime endowments, Mastra, and the AI SDK.
//   - internal/engine/runtime/*.d.ts (excluding AssemblyScript,
//     which is dormant) — type-precise declarations of every
//     global a deployed .ts can see.
//
// Embedding both means a Kit ships a self-describing copy of its
// surface. Agents that design other agents can ground their LLM
// on this corpus and produce code against real symbols instead of
// guessed ones.
//
//go:embed docs/llm/*.md internal/engine/runtime/agent.d.ts internal/engine/runtime/ai.d.ts internal/engine/runtime/brainkit.d.ts internal/engine/runtime/globals.d.ts internal/engine/runtime/kit.d.ts
var referenceFS embed.FS

// rawFiles catalogs every embedded file by the short name callers
// use, mapping to the embed path. The short name is the filename
// minus its directory (extensions preserved to keep .md vs .d.ts
// unambiguous).
var rawFiles = map[string]string{
	// Prose/reference markdown.
	"go-sdk.md":     "docs/llm/go-sdk.md",
	"go-config.md":  "docs/llm/go-config.md",
	"ts-runtime.md": "docs/llm/ts-runtime.md",
	"ai-sdk.md":     "docs/llm/ai-sdk.md",
	"mastra.md":     "docs/llm/mastra.md",
	// TypeScript declaration files.
	"agent.d.ts":    "internal/engine/runtime/agent.d.ts",
	"ai.d.ts":       "internal/engine/runtime/ai.d.ts",
	"brainkit.d.ts": "internal/engine/runtime/brainkit.d.ts",
	"globals.d.ts":  "internal/engine/runtime/globals.d.ts",
	"kit.d.ts":      "internal/engine/runtime/kit.d.ts",
}

// packs are curated bundles composed from rawFiles. Each pack is
// a name an LLM-orchestration layer can request by intent instead
// of assembling files manually. Order within a pack is the order
// content is concatenated (with a small separator between files).
var packs = map[string][]string{
	// Everything a coder agent needs to write a .ts deployment
	// that runs an Agent with tools + handlers + memory.
	"agent-author": {
		"globals.d.ts",
		"kit.d.ts",
		"agent.d.ts",
		"ai.d.ts",
		"ts-runtime.md",
		"ai-sdk.md",
		"mastra.md",
	},
	// What's needed to write a createTool(...) that slots into an
	// Agent or a bus.on handler — thinner than agent-author.
	"tool-author": {
		"globals.d.ts",
		"kit.d.ts",
		"ts-runtime.md",
	},
	// Everything for someone writing a createWorkflow pipeline
	// with agent-driven and code-driven steps.
	"workflow-author": {
		"globals.d.ts",
		"kit.d.ts",
		"agent.d.ts",
		"ts-runtime.md",
		"mastra.md",
	},
	// What a Go consumer embedding brainkit needs: Config, Kit
	// methods, Call wrappers, Module composition.
	"kit-consumer": {
		"go-config.md",
		"go-sdk.md",
	},
	// Everything. Use this pack when a coder agent needs maximum
	// grounding and you don't want to guess which docs it should
	// see. ~250kb; fits comfortably in a modern model's context.
	"everything": {
		"globals.d.ts",
		"kit.d.ts",
		"agent.d.ts",
		"ai.d.ts",
		"brainkit.d.ts",
		"ts-runtime.md",
		"ai-sdk.md",
		"mastra.md",
		"go-config.md",
		"go-sdk.md",
	},
}

// ReferenceKind is "pack" for curated bundles, "raw" for direct
// files under docs/llm or internal/engine/runtime.
type ReferenceKind string

const (
	ReferenceKindPack ReferenceKind = "pack"
	ReferenceKindRaw  ReferenceKind = "raw"
)

// ReferenceInfo describes a single addressable reference.
type ReferenceInfo struct {
	Name        string        `json:"name"`
	Kind        ReferenceKind `json:"kind"`
	Description string        `json:"description"`
	Size        int           `json:"size"`
	// Parts lists the component names a pack composes from. Empty
	// for raw entries. Exposed so a caller can see what a pack
	// bundles without reading the content.
	Parts []string `json:"parts,omitempty"`
}

// packDescriptions keeps the short human-readable blurb shown in
// ReferenceList alongside each pack name.
var packDescriptions = map[string]string{
	"agent-author":    "Complete set for writing a .ts package that runs an Agent with tools, handlers, and memory.",
	"tool-author":     "Everything needed to write createTool(...) + bus handlers. Lighter than agent-author.",
	"workflow-author": "Compose Mastra workflows (createWorkflow + createStep + agent steps) inside a .ts deployment.",
	"kit-consumer":    "Go consumer reference: Config, Kit methods, Call wrappers, Module composition.",
	"everything":      "The whole corpus — every .d.ts + every .md, concatenated. Use when you want maximum grounding and don't want to pick.",
}

// Reference returns the content of the named reference. Names are
// either a pack (composed bundle) or a raw entry like
// "go-sdk.md" / "kit.d.ts". Pack names always resolve first; raw
// files also accept bare names without their extension
// ("ts-runtime" → "ts-runtime.md", "agent" → "agent.d.ts" when
// unambiguous).
//
// The returned string is UTF-8 and ready to drop into a model's
// system prompt as-is. No processing, no trimming — callers
// assemble their own prompts around it.
func Reference(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("brainkit: Reference: name is required")
	}
	if parts, ok := packs[name]; ok {
		return composePack(name, parts)
	}
	p, err := resolveRaw(name)
	if err != nil {
		return "", err
	}
	b, err := fs.ReadFile(referenceFS, p)
	if err != nil {
		return "", fmt.Errorf("brainkit: Reference %q: %w", name, err)
	}
	return string(b), nil
}

// ReferenceList returns every addressable reference — packs
// first, then raw files — sorted alphabetically within each
// group. Size is in bytes of the composed or raw content; a
// caller can use it to budget token windows.
func ReferenceList() []ReferenceInfo {
	out := make([]ReferenceInfo, 0, len(packs)+len(rawFiles))

	packNames := make([]string, 0, len(packs))
	for name := range packs {
		packNames = append(packNames, name)
	}
	sort.Strings(packNames)
	for _, name := range packNames {
		body, _ := composePack(name, packs[name])
		out = append(out, ReferenceInfo{
			Name:        name,
			Kind:        ReferenceKindPack,
			Description: packDescriptions[name],
			Size:        len(body),
			Parts:       append([]string(nil), packs[name]...),
		})
	}

	rawNames := make([]string, 0, len(rawFiles))
	for name := range rawFiles {
		rawNames = append(rawNames, name)
	}
	sort.Strings(rawNames)
	for _, name := range rawNames {
		b, _ := fs.ReadFile(referenceFS, rawFiles[name])
		out = append(out, ReferenceInfo{
			Name:        name,
			Kind:        ReferenceKindRaw,
			Description: rawDescription(name),
			Size:        len(b),
		})
	}
	return out
}

// ReferenceNames returns just the name strings from ReferenceList
// in the same order — handy for the bus list command.
func ReferenceNames() []string {
	infos := ReferenceList()
	out := make([]string, len(infos))
	for i, info := range infos {
		out[i] = info.Name
	}
	return out
}

func composePack(name string, parts []string) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "# Reference pack: %s\n\n", name)
	if desc := packDescriptions[name]; desc != "" {
		fmt.Fprintf(&b, "%s\n\n", desc)
	}
	fmt.Fprintln(&b, "Composed from:")
	for _, p := range parts {
		fmt.Fprintf(&b, "  - %s\n", p)
	}
	b.WriteString("\n---\n\n")
	for _, p := range parts {
		body, err := Reference(p)
		if err != nil {
			return "", fmt.Errorf("brainkit: pack %q: part %q: %w", name, p, err)
		}
		fmt.Fprintf(&b, "## %s\n\n", p)
		b.WriteString(body)
		if !strings.HasSuffix(body, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("\n---\n\n")
	}
	return b.String(), nil
}

// resolveRaw maps a short name (with or without extension) to its
// embed path. Unextended names win only if unambiguous — a single
// file matches the stem. That lets callers say "ts-runtime" when
// they mean the markdown and "agent.d.ts" when they mean the
// type declarations without guessing wrong.
func resolveRaw(name string) (string, error) {
	if p, ok := rawFiles[name]; ok {
		return p, nil
	}
	stem := name
	if path.Ext(stem) == "" {
		var matches []string
		for full, p := range rawFiles {
			if strings.HasPrefix(full, name+".") {
				matches = append(matches, p)
			}
		}
		switch len(matches) {
		case 1:
			return matches[0], nil
		case 0:
			// fall through to the error below
		default:
			sort.Strings(matches)
			return "", fmt.Errorf("brainkit: Reference %q is ambiguous: matches %v — request the full name with extension", name, matches)
		}
	}
	return "", fmt.Errorf("brainkit: Reference %q: not found (try one of %v)", name, ReferenceNames())
}

// registerReferenceCommands wires the kit.reference and
// kit.reference.list bus handlers on a freshly-constructed Kit.
// Called from brainkit.New before module init so Module.Init
// implementations can depend on the commands being present.
func registerReferenceCommands(k *Kit) {
	k.RegisterCommand(Command(func(_ context.Context, req sdk.KitReferenceMsg) (*sdk.KitReferenceResp, error) {
		content, err := Reference(req.Name)
		if err != nil {
			return nil, err
		}
		return &sdk.KitReferenceResp{Name: req.Name, Content: content}, nil
	}))
	k.RegisterCommand(Command(func(_ context.Context, _ sdk.KitReferenceListMsg) (*sdk.KitReferenceListResp, error) {
		infos := ReferenceList()
		entries := make([]sdk.KitReferenceListEntry, len(infos))
		for i, info := range infos {
			entries[i] = sdk.KitReferenceListEntry{
				Name:        info.Name,
				Kind:        string(info.Kind),
				Description: info.Description,
				Size:        info.Size,
				Parts:       info.Parts,
			}
		}
		return &sdk.KitReferenceListResp{References: entries}, nil
	}))
}

func rawDescription(name string) string {
	switch name {
	case "go-sdk.md":
		return "Go SDK reference: Kit methods, accessors, Call wrappers, SDK helpers, errors, envelope."
	case "go-config.md":
		return "Config struct fields: transport, providers, storages, vectors, secrets, modules, tracing."
	case "ts-runtime.md":
		return "The .ts compartment surface: globals, bus.*, kit.register, msg.reply/send, cancellation."
	case "ai-sdk.md":
		return "AI SDK surface inside .ts: generateText / streamText / generateObject / embed, model() resolver."
	case "mastra.md":
		return "Mastra surface inside .ts: Agent, createTool, createWorkflow, Memory, vector stores, scorers."
	case "agent.d.ts":
		return "TypeScript declarations for the Mastra Agent surface inside a deployed .ts package."
	case "ai.d.ts":
		return "TypeScript declarations for the bundled AI SDK (generateText, streamText, tool, model)."
	case "brainkit.d.ts":
		return "TypeScript declarations for the top-level brainkit globals (bus, kit, output)."
	case "globals.d.ts":
		return "TypeScript declarations for JS built-ins and Node-compat polyfills available in .ts deployments."
	case "kit.d.ts":
		return "TypeScript declarations for the `kit` object: register, secrets, fs, mcp."
	}
	return ""
}
