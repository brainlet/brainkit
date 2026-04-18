package sdk

// KitReferenceMsg fetches one of the bundled reference documents
// that describe brainkit's Go + TypeScript surface. Used by
// deployments that build architect-style agents — drop the
// reference body into a system prompt so the LLM writes code
// against the real surface instead of guessing.
type KitReferenceMsg struct {
	// Name of a pack ("agent-author", "tool-author",
	// "workflow-author", "kit-consumer") or a raw file
	// ("go-sdk.md", "ts-runtime.md", "agent.d.ts", "kit.d.ts",
	// "globals.d.ts", "ai.d.ts", "brainkit.d.ts",
	// "go-config.md", "ai-sdk.md", "mastra.md"). Bare names
	// without the extension resolve when unambiguous
	// ("ts-runtime" → "ts-runtime.md").
	Name string `json:"name"`
}

func (KitReferenceMsg) BusTopic() string { return "kit.reference" }

// KitReferenceResp carries the reference document's content.
// Content is UTF-8 (markdown for .md packs, TypeScript declaration
// syntax for .d.ts entries) and is ready to feed straight into a
// model as part of a system prompt.
type KitReferenceResp struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// KitReferenceListMsg asks for the catalog of available reference
// names.
type KitReferenceListMsg struct{}

func (KitReferenceListMsg) BusTopic() string { return "kit.reference.list" }

// KitReferenceListEntry is one row in the catalog.
type KitReferenceListEntry struct {
	Name        string   `json:"name"`
	Kind        string   `json:"kind"` // "pack" or "raw"
	Description string   `json:"description"`
	Size        int      `json:"size"`
	Parts       []string `json:"parts,omitempty"`
}

// KitReferenceListResp enumerates every reference with size + kind
// so a caller can budget token windows and know what each pack
// bundles without fetching it.
type KitReferenceListResp struct {
	References []KitReferenceListEntry `json:"references"`
}
