package fixtures

import (
	"strings"
)

// FixtureNeeds describes what infrastructure a fixture requires.
type FixtureNeeds struct {
	Container    string // "postgres", "mongodb", "" for none
	Credential   string // "UPSTASH_REDIS_REST_URL", "" for none
	LibSQLServer bool   // needs libsql-server container (vector extensions)
	AI           bool   // needs OPENAI_API_KEY
	MCP          bool   // needs in-process MCP server
}

// containerBackends maps path segments to container names.
// A fixture needs a container if any segment in its relative path matches.
var containerBackends = map[string]string{
	"postgres":       "postgres",
	"postgres-scram": "postgres",
	"pgvector":       "postgres",
	"mongodb":        "mongodb",
	"mongodb-scram":  "mongodb",
}

// credentialBackends need cloud credentials, not containers.
var credentialBackends = map[string]string{
	"upstash": "UPSTASH_REDIS_REST_URL",
}

// libsqlServerSegments flag fixtures that need a libsql-server
// container. Any fixture whose source reads `process.env.LIBSQL_URL`
// must be covered here — the test runner starts the container and
// injects LIBSQL_URL only when needs.LibSQLServer is true.
var libsqlServerSegments = map[string]bool{
	"libsql":             true,
	"libsql-local":       true,
	"libsql-local-debug": true,
	// memory/semantic-recall/basic + memory/working-memory/basic
	// construct LibSQLStore+LibSQLVector directly; their variant
	// leaves are listed below in classifyLibSQLLeaves.
}

// libsqlLeafPaths are individual fixtures that need libsql-server
// but whose leaf path doesn't include a libsql segment (so the
// segment map above can't catch them).
var libsqlLeafPaths = map[string]bool{
	"memory/semantic-recall/basic": true,
	"memory/working-memory/basic":  true,
	"rag/vector-query-tool":        true,
}

// aiCategories are top-level categories where every fixture needs AI.
var aiCategories = map[string]bool{
	"agent":         true,
	"ai":            true,
	"observability": true,
	"composition":   true,
	// Voice fixtures need at least OPENAI_API_KEY (the default
	// provider) for most tests; construct-only fixtures for
	// non-OpenAI providers skip gracefully when the provider's
	// own key isn't set.
	"voice": true,
	// Processor fixtures: most useful ones are LLM-gated
	// (moderation / PII / injection / structured-output /
	// language / system-prompt-scrubber / tool-search /
	// skill-search). The model-free variants pay no cost when
	// AI is configured.
	"processors": true,
}

// aiSegments trigger AI need when found anywhere in the path.
var aiSegments = map[string]bool{
	"with-agent-step":   true,
	"vector-query-tool": true,
	"with-llm-judge":    true,
	"semantic-recall":   true,
	"generate-title":    true,
	"working-memory":    true,
	// RAG rerank / graph-rag use embedding or judge models.
	"rerank":    true,
	"graph-rag": true,
	// Prebuilt scorers under evals/prebuilt/ — every LLM-judge
	// scorer needs AI; a few code scorers don't, but they still
	// resolve a judge model when one is set.
	"prebuilt": true,
}

// ClassifyFixture determines what infrastructure a fixture needs based on its
// relative path from the fixtures root (e.g. "agent/generate/basic",
// "memory/storage/postgres", "vector/create-upsert-query/pgvector").
//
// Classification scans ALL path segments, so nesting depth doesn't matter.
func ClassifyFixture(relPath string) FixtureNeeds {
	var needs FixtureNeeds

	segments := strings.Split(relPath, "/")
	if len(segments) < 2 {
		return needs
	}

	category := segments[0]

	// MCP detection
	if category == "mcp" {
		needs.MCP = true
	}

	// AI detection: category-level
	if aiCategories[category] {
		needs.AI = true
	}

	// Scan all segments for infrastructure needs
	for _, seg := range segments {
		// Container detection
		if backend, ok := containerBackends[seg]; ok && needs.Container == "" {
			needs.Container = backend
		}

		// Credential detection
		if envVar, ok := credentialBackends[seg]; ok && needs.Credential == "" {
			needs.Credential = envVar
		}

		// LibSQL server detection: memory + agent + vector fixtures
		// that construct LibSQLStore / LibSQLVector against an
		// injected LIBSQL_URL all need the container.
		if libsqlServerSegments[seg] {
			needs.LibSQLServer = true
		}

		// AI segment detection
		if aiSegments[seg] {
			needs.AI = true
		}
	}

	// Leaf-path overrides for fixtures whose libsql usage can't be
	// inferred from any single segment (semantic-recall/basic,
	// working-memory/basic, rag/vector-query-tool).
	if libsqlLeafPaths[relPath] {
		needs.LibSQLServer = true
	}

	// memory/storage/* all need AI (they use Agent conversation)
	if category == "memory" && hasSegment(segments, "storage") {
		needs.AI = true
	}

	return needs
}

// hasSegment checks if a specific segment exists in the path.
func hasSegment(segments []string, target string) bool {
	for _, seg := range segments {
		if seg == target {
			return true
		}
	}
	return false
}
